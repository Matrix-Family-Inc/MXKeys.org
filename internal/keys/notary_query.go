/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Mon 22 Jun 2026 00:50:40 UTC
 * Status: Updated
 */

package keys

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"mxkeys/internal/zero/log"
)

// QueryKeys queries keys for multiple servers (notary functionality).
func (n *Notary) QueryKeys(ctx context.Context, request *KeyQueryRequest) *KeyQueryResponse {
	response := &KeyQueryResponse{
		ServerKeys: make([]ServerKeysResponse, 0),
		Failures:   make(map[string]interface{}),
	}

	for serverName, failure := range n.applyTrustPolicyToRequest(request) {
		response.Failures[serverName] = failure
	}

	for _, serverName := range sortedServerNames(request.ServerKeys) {
		keyCriteria := request.ServerKeys[serverName]
		var minValidUntil int64
		for _, criteria := range keyCriteria {
			if criteria.MinimumValidUntilTS > minValidUntil {
				minValidUntil = criteria.MinimumValidUntilTS
			}
		}

		keys, err := n.GetServerKeysWithCriteria(ctx, serverName, minValidUntil)
		if err != nil {
			log.Warn("Failed to get keys for server", "server", serverName, "error", err)
			if analytics := n.getAnalytics(); analytics != nil {
				analytics.RecordFetchFailure(serverName)
			}
			if transparency := n.getTransparency(); transparency != nil {
				_ = transparency.LogFailure(ctx, serverName, err.Error())
			}
			response.Failures[serverName] = sanitizeQueryFailure(err)
			continue
		}

		keysForResponse, err := n.countersign(keys)
		if err != nil {
			log.Error("Failed to add notary signature", "server", serverName, "error", err)
			response.Failures[serverName] = matrixFailure("M_UNKNOWN", "Internal signing error")
			continue
		}

		if violation := n.checkResponsePolicy(serverName, keysForResponse); violation != nil {
			if transparency := n.getTransparency(); transparency != nil {
				_ = transparency.LogPolicyViolation(ctx, violation)
			}
			response.Failures[serverName] = map[string]interface{}{
				"errcode": "M_FORBIDDEN",
				"error":   fmt.Sprintf("Trust policy violation (%s): %s", violation.Rule, violation.Details),
			}
			continue
		}

		if analytics := n.getAnalytics(); analytics != nil {
			analytics.RecordKeyObservation(serverName, keysForResponse)
		}
		if transparency := n.getTransparency(); transparency != nil {
			_ = transparency.LogKey(ctx, serverName, keysForResponse)
		}

		response.ServerKeys = append(response.ServerKeys, *keysForResponse)
	}

	return response
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

func sanitizeQueryFailure(err error) map[string]interface{} {
	switch {
	case errors.Is(err, ErrContextCanceled):
		return matrixFailure("M_UNKNOWN", "Request canceled")
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
			failures[serverName] = map[string]interface{}{
				"errcode": "M_FORBIDDEN",
				"error":   fmt.Sprintf("Trust policy violation (%s): %s", violation.Rule, violation.Details),
			}
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

// GetServerKeysWithCriteria gets keys for a server respecting minimum_valid_until_ts.
func (n *Notary) GetServerKeysWithCriteria(ctx context.Context, serverName string, minValidUntil int64) (*ServerKeysResponse, error) {
	keys, err := n.GetServerKeys(ctx, serverName)
	if err != nil {
		return nil, err
	}

	if minValidUntil > 0 && keys.ValidUntilTS < minValidUntil {
		log.Debug("Cached keys don't meet minimum_valid_until_ts, refetching",
			"server", serverName,
			"cached_valid", keys.ValidUntilTS,
			"required_valid", minValidUntil,
		)
		recordRefetch(RefetchReasonMinValidUntil)
		keys, err = n.fetchAndStoreWithSource(ctx, serverName, FetchSourceRefetch)
		if err != nil {
			return nil, err
		}
	}

	return keys, nil
}

// GetServerKeys gets keys for a server (from cache, storage, or fetch).
func (n *Notary) GetServerKeys(ctx context.Context, serverName string) (*ServerKeysResponse, error) {
	if ctx.Err() != nil {
		return nil, &KeyError{Op: "get_keys", ServerName: serverName, Err: ErrContextCanceled}
	}

	now := time.Now()
	n.cacheMu.RLock()
	cached, ok := n.cache[serverName]
	n.cacheMu.RUnlock()

	if ok && n.cacheEntryValid(cached, now) {
		log.Debug("Returning keys from memory cache", "server", serverName)
		recordMemoryCacheHit()
		return cloneServerKeysResponse(cached.response), nil
	}
	if ok {
		n.cacheMu.Lock()
		current, exists := n.cache[serverName]
		if exists && current == cached && !n.cacheEntryValid(current, now) {
			delete(n.cache, serverName)
			updateMemoryCacheSize(len(n.cache))
		}
		n.cacheMu.Unlock()
	}
	recordMemoryCacheMiss()

	stored, dbErr := n.storage.GetServerResponse(serverName)
	if dbErr != nil {
		log.Warn("Database cache unavailable, continuing with remote fetch", "error", dbErr)
	}
	if stored != nil && len(stored.Raw) > 0 && stored.ValidUntilTS > now.UnixMilli() {
		recordDBCacheHit()
		n.storeInMemoryCache(serverName, stored)

		log.Debug("Returning keys from database cache", "server", serverName)
		return stored, nil
	}
	recordDBCacheMiss()

	result, err, _ := n.fetchGroup.Do(serverName, func() (interface{}, error) {
		return n.fetchAndStore(ctx, serverName)
	})
	if err != nil {
		n.cacheMu.RLock()
		expired, hasExpired := n.cache[serverName]
		n.cacheMu.RUnlock()

		if hasExpired && expired.response != nil {
			validUntil := time.UnixMilli(expired.response.ValidUntilTS)
			if validUntil.After(time.Now()) {
				log.Warn("Using stale memory cache entry after fetch failure",
					"server", serverName,
					"error", err,
					"response_valid_until", validUntil,
				)
				return expired.response, nil
			}
		}

		return nil, err
	}

	return result.(*ServerKeysResponse), nil
}
