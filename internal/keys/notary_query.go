/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 20 Nov 2026 22:39:55 UTC
 * Status: Updated
 */

package keys

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"mxkeys/internal/zero/log"
)

// queryConcurrency bounds how many servers of one query are looked up at
// the same time.
const queryConcurrency = 16

// queryResult is the outcome for one server of a query.
type queryResult struct {
	keys    *ServerKeysResponse
	failure map[string]interface{}
}

// QueryKeys answers POST /_matrix/key/v2/query. Servers are looked up in
// parallel under one deadline; servers not answered in time are reported
// under failures. Output order follows the sorted server names.
func (n *Notary) QueryKeys(ctx context.Context, request *KeyQueryRequest) *KeyQueryResponse {
	response := &KeyQueryResponse{
		ServerKeys: make([]ServerKeysResponse, 0),
		Failures:   make(map[string]interface{}),
	}

	for serverName, failure := range n.applyTrustPolicyToRequest(request) {
		response.Failures[serverName] = failure
	}

	ctx, cancel := context.WithTimeout(ctx, n.queryTimeout())
	defer cancel()

	names := sortedServerNames(request.ServerKeys)
	results := make([]queryResult, len(names))
	slots := make(chan struct{}, queryConcurrency)
	var wg sync.WaitGroup
	for i, serverName := range names {
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
			results[i] = queryResult{failure: matrixFailure("M_UNKNOWN", "Lookup did not finish in time")}
			continue
		}
		wg.Add(1)
		go func(i int, serverName string) {
			defer wg.Done()
			defer func() { <-slots }()
			results[i] = n.queryServer(ctx, serverName, minimumValidUntil(request.ServerKeys[serverName]))
		}(i, serverName)
	}
	wg.Wait()

	for i, serverName := range names {
		if results[i].keys != nil {
			response.ServerKeys = append(response.ServerKeys, *results[i].keys)
		} else if results[i].failure != nil {
			response.Failures[serverName] = results[i].failure
		}
	}
	return response
}

// queryServer looks up, countersigns and policy-checks one server.
func (n *Notary) queryServer(ctx context.Context, serverName string, minValidUntil int64) queryResult {
	keys, err := n.GetServerKeysWithCriteria(ctx, serverName, minValidUntil)
	if err != nil {
		log.Warn("Failed to get keys for server", "server", serverName, "error", err)
		if analytics := n.getAnalytics(); analytics != nil {
			analytics.RecordFetchFailure(serverName)
		}
		if transparency := n.getTransparency(); transparency != nil {
			_ = transparency.LogFailure(ctx, serverName, err.Error())
		}
		return queryResult{failure: sanitizeQueryFailure(err)}
	}

	countersigned, err := n.countersign(keys)
	if err != nil {
		log.Error("Failed to add notary signature", "server", serverName, "error", err)
		return queryResult{failure: matrixFailure("M_UNKNOWN", "Internal signing error")}
	}

	if violation := n.checkResponsePolicy(serverName, countersigned); violation != nil {
		if transparency := n.getTransparency(); transparency != nil {
			_ = transparency.LogPolicyViolation(ctx, violation)
		}
		return queryResult{failure: matrixFailure("M_FORBIDDEN",
			fmt.Sprintf("Trust policy violation (%s): %s", violation.Rule, violation.Details))}
	}

	if analytics := n.getAnalytics(); analytics != nil {
		analytics.RecordKeyObservation(serverName, countersigned)
	}
	if transparency := n.getTransparency(); transparency != nil {
		_ = transparency.LogKey(ctx, serverName, countersigned)
	}
	return queryResult{keys: countersigned}
}

// countersign returns a copy of a verified key document with this notary's
// perspective signature added to the document bytes. The struct fields are
// re-read from the signed bytes so policy checks see what is served.
func (n *Notary) countersign(keys *ServerKeysResponse) (*ServerKeysResponse, error) {
	if keys == nil || len(keys.Raw) == 0 {
		return nil, fmt.Errorf("verified key document bytes are unavailable")
	}
	signed, err := AttachNotarySignature(keys.Raw, n.serverName, n.serverKeyID, n.serverKeyPair)
	if err != nil {
		return nil, err
	}
	countersigned, _, err := parseServerKeys(signed)
	if err != nil {
		return nil, err
	}
	return countersigned, nil
}

func minimumValidUntil(criteria map[string]KeyCriteria) int64 {
	var minValidUntil int64
	for _, c := range criteria {
		if c.MinimumValidUntilTS > minValidUntil {
			minValidUntil = c.MinimumValidUntilTS
		}
	}
	return minValidUntil
}

func sanitizeQueryFailure(err error) map[string]interface{} {
	switch {
	case errors.Is(err, ErrContextCanceled), errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return matrixFailure("M_UNKNOWN", "Lookup did not finish in time")
	case errors.Is(err, ErrConcurrencyLimit), errors.Is(err, ErrCircuitOpen):
		return matrixFailure("M_LIMIT_EXCEEDED", "Upstream temporarily unavailable")
	case errors.Is(err, ErrResolveFailed):
		return matrixFailure("M_NOT_FOUND", "Unable to resolve remote server")
	case errors.Is(err, ErrFetchFailed):
		return matrixFailure("M_UNKNOWN", "Unable to fetch remote server keys")
	case errors.Is(err, ErrSignatureInvalid), errors.Is(err, ErrServerNameMismatch), errors.Is(err, ErrInvalidResponse), errors.Is(err, ErrNotaryKeyMismatch):
		return matrixFailure("M_INVALID_PARAM", "Remote server keys failed verification")
	default:
		return matrixFailure("M_UNKNOWN", "Unable to obtain verified keys")
	}
}

func matrixFailure(errCode, message string) map[string]interface{} {
	return map[string]interface{}{
		"errcode": errCode,
		"error":   message,
	}
}

func sortedServerNames(serverKeys map[string]map[string]KeyCriteria) []string {
	names := make([]string, 0, len(serverKeys))
	for serverName := range serverKeys {
		names = append(names, serverName)
	}
	sort.Strings(names)
	return names
}

func (n *Notary) applyTrustPolicyToRequest(request *KeyQueryRequest) map[string]interface{} {
	failures := make(map[string]interface{})
	trustPolicy := n.getTrustPolicy()
	if request == nil || trustPolicy == nil {
		return failures
	}

	for serverName := range request.ServerKeys {
		if violation := trustPolicy.CheckServer(serverName); violation != nil {
			failures[serverName] = matrixFailure("M_FORBIDDEN",
				fmt.Sprintf("Trust policy violation (%s): %s", violation.Rule, violation.Details))
			delete(request.ServerKeys, serverName)
		}
	}

	return failures
}

func (n *Notary) checkResponsePolicy(serverName string, resp *ServerKeysResponse) *PolicyViolation {
	trustPolicy := n.getTrustPolicy()
	if trustPolicy == nil || resp == nil {
		return nil
	}
	return trustPolicy.CheckResponse(serverName, resp)
}
