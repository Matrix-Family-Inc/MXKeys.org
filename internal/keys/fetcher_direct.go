/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

package keys

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"mxkeys/internal/zero/log"
)

// fetchDirect fetches keys directly from the target server after resolver
// discovery. Rejects responses with mismatched server_name. Verifies the
// self-signature over canonical JSON before returning.
func (f *Fetcher) fetchDirect(ctx context.Context, serverName string) (*ServerKeysResponse, error) {
	resolved, err := f.resolver.ResolveServerName(ctx, serverName)
	if err != nil {
		return nil, NewResolveError(serverName, err)
	}
	if err := f.rejectPrivateAddress(ctx, serverName, resolved); err != nil {
		return nil, NewResolveError(serverName, err)
	}

	url := fmt.Sprintf("%s/_matrix/key/v2/server", resolved.URL())

	log.Debug("Fetching server keys directly",
		"server", serverName,
		"resolved_host", resolved.Host,
		"resolved_port", resolved.Port,
		"url", url,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Host = resolved.ServerName

	resp, err := f.clientForResolved(resolved).Do(req)
	if err != nil {
		classifyUpstreamFailure(err)
		return nil, fmt.Errorf("HTTP request to %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		recordUpstreamFailure(UpstreamFailureHTTP)
		body, _ := readLimitedBody(resp.Body, maxFederationBody)
		return nil, fmt.Errorf("key fetch from %s returned status %d: %s", url, resp.StatusCode, string(body))
	}

	body, err := readLimitedBody(resp.Body, maxFederationBody)
	if err != nil {
		return nil, err
	}

	var keysResp ServerKeysResponse
	if err := json.Unmarshal(body, &keysResp); err != nil {
		return nil, fmt.Errorf("failed to parse response from %s: %w", url, err)
	}

	if keysResp.ServerName != serverName {
		recordUpstreamFailure(UpstreamFailureServerMismatch)
		return nil, &KeyError{
			Op:         "validate",
			ServerName: serverName,
			Err:        fmt.Errorf("%w: expected %s, got %s", ErrServerNameMismatch, serverName, keysResp.ServerName),
		}
	}

	if err := f.verifySelfSignature(&keysResp, body); err != nil {
		recordUpstreamFailure(UpstreamFailureInvalidSignature)
		return nil, NewSignatureError(serverName, err)
	}

	// Preserve the exact origin-delivered bytes. This is what later
	// lets notary_query attach a perspective signature without
	// reshaping origin fields and keeps origin self-signature
	// verification valid end-to-end.
	keysResp.Raw = append([]byte(nil), body...)

	log.Info("Successfully fetched server keys",
		"server", serverName,
		"keys_count", len(keysResp.VerifyKeys),
		"valid_until", time.UnixMilli(keysResp.ValidUntilTS).Format(time.RFC3339),
	)

	return &keysResp, nil
}

// classifyUpstreamFailure records the best-matching upstream failure reason
// based on the transport error message.
func classifyUpstreamFailure(err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "tls"), strings.Contains(msg, "certificate"):
		recordUpstreamFailure(UpstreamFailureTLS)
	case strings.Contains(msg, "timeout"):
		recordUpstreamFailure(UpstreamFailureTimeout)
	default:
		recordUpstreamFailure(UpstreamFailureHTTP)
	}
}
