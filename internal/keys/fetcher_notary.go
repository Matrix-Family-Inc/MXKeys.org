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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// fetchFromNotary fetches keys for serverName by querying a perspective
// notary. The returned key document must pass the same checks as a direct
// fetch: exact server_name, current validity and a self-signature by one of
// its verify_keys. When a key is pinned for the notary, the notary's own
// signature is verified too. The document bytes are kept as delivered so
// every signature on them stays verifiable downstream.
func (f *Fetcher) fetchFromNotary(ctx context.Context, notary, serverName string) (*ServerKeysResponse, error) {
	resolved, err := f.resolver.ResolveServerName(ctx, notary)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve notary %s: %w", notary, err)
	}
	if err := f.rejectPrivateAddress(ctx, notary, resolved); err != nil {
		return nil, fmt.Errorf("failed private-address check for notary %s: %w", notary, err)
	}

	url := fmt.Sprintf("%s/_matrix/key/v2/query", resolved.URL())

	reqJSON, err := json.Marshal(KeyQueryRequest{
		ServerKeys: map[string]map[string]KeyCriteria{serverName: {}},
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Host = resolved.HostHeader()

	resp, err := f.clientForResolved(resolved).Do(req)
	if err != nil {
		return nil, fmt.Errorf("notary query to %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := readLimitedBody(resp.Body, maxFederationBody)
		return nil, fmt.Errorf("notary %s returned status %d: %s", notary, resp.StatusCode, string(body))
	}

	body, err := readLimitedBody(resp.Body, maxFederationBody)
	if err != nil {
		return nil, err
	}

	var notaryResp struct {
		ServerKeys []json.RawMessage `json:"server_keys"`
	}
	if err := json.Unmarshal(body, &notaryResp); err != nil {
		return nil, fmt.Errorf("failed to parse notary response from %s: %w", notary, err)
	}

	for _, rawDoc := range notaryResp.ServerKeys {
		keys, obj, err := parseServerKeys(rawDoc)
		if err != nil {
			return nil, NewValidationError(serverName, fmt.Errorf("notary %s: %w", notary, err))
		}
		if keys.ServerName != serverName {
			continue
		}
		if err := f.checkSelfSignature(keys, obj); err != nil {
			return nil, NewSignatureError(serverName, fmt.Errorf("via notary %s: %w", notary, err))
		}
		if err := f.verifyNotarySignature(notary, rawDoc); err != nil {
			return nil, err
		}
		return keys, nil
	}

	return nil, fmt.Errorf("server %s not found in notary %s response", serverName, notary)
}
