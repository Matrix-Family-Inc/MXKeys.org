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
)

// fetchFromNotary fetches keys for serverName by querying a perspective notary.
// When a pinned notary key is configured, the response is additionally verified
// against that key via verifyNotarySignature before being returned.
func (f *Fetcher) fetchFromNotary(ctx context.Context, notary, serverName string) (*ServerKeysResponse, error) {
	resolved, err := f.resolver.ResolveServerName(ctx, notary)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve notary %s: %w", notary, err)
	}
	if err := f.rejectPrivateAddress(ctx, notary, resolved); err != nil {
		return nil, fmt.Errorf("failed private-address check for notary %s: %w", notary, err)
	}

	url := fmt.Sprintf("%s/_matrix/key/v2/query", resolved.URL())

	reqBody := KeyQueryRequest{
		ServerKeys: map[string]map[string]KeyCriteria{
			serverName: {},
		},
	}
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(reqJSON)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Host = resolved.ServerName

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

	var notaryResp KeyQueryResponse
	if err := json.Unmarshal(body, &notaryResp); err != nil {
		return nil, fmt.Errorf("failed to parse notary response from %s: %w", notary, err)
	}

	for _, keys := range notaryResp.ServerKeys {
		if keys.ServerName == serverName {
			if err := f.verifyNotarySignature(notary, &keys); err != nil {
				return nil, err
			}
			return &keys, nil
		}
	}

	return nil, fmt.Errorf("server %s not found in notary %s response", serverName, notary)
}
