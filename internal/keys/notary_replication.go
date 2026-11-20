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
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"mxkeys/internal/zero/log"
)

// ClusterReplicatedResponseKeyID is the sentinel key_id used to mark the
// replicated whole-server-response payloads on the cluster wire.
const ClusterReplicatedResponseKeyID = "_server_response"

// ApplyReplicatedServerResponse validates, persists, and caches a server key
// response delivered via cluster replication.
func (n *Notary) ApplyReplicatedServerResponse(serverName string, rawResponse string, validUntilTS int64) error {
	response, err := n.validateReplicatedServerResponse(serverName, rawResponse, validUntilTS)
	if err != nil {
		return err
	}
	validUntil := time.UnixMilli(response.ValidUntilTS)
	if err := n.storage.StoreServerResponse(serverName, response, validUntil); err != nil {
		return fmt.Errorf("failed to store replicated server response: %w", err)
	}

	for keyID, verifyKey := range response.VerifyKeys {
		pubKeyBytes, err := decodeBase64(verifyKey.Key)
		if err != nil {
			log.Debug("Skipping replicated key with invalid base64", "server", serverName, "key_id", keyID, "error", err)
			continue
		}
		if err := n.storage.StoreKey(serverName, keyID, pubKeyBytes, validUntil); err != nil {
			log.Warn("Failed to store replicated key", "server", serverName, "key_id", keyID, "error", err)
		}
	}

	n.storeInMemoryCache(serverName, cloneServerKeysResponse(response))

	return nil
}

// validateReplicatedServerResponse parses and verifies the replicated payload.
// The payload must carry a self-signature for the remote server. The notary
// must not accept replicated responses without cryptographic verification.
func (n *Notary) validateReplicatedServerResponse(serverName string, rawResponse string, validUntilTS int64) (*ServerKeysResponse, error) {
	if strings.TrimSpace(rawResponse) == "" {
		return nil, fmt.Errorf("replicated response payload is empty")
	}

	response, obj, err := parseServerKeys([]byte(rawResponse))
	if err != nil {
		return nil, fmt.Errorf("failed to decode replicated response: %w", err)
	}
	if response.ServerName != serverName {
		return nil, fmt.Errorf("replicated response server mismatch: expected %s got %s", serverName, response.ServerName)
	}
	if response.ValidUntilTS != validUntilTS {
		return nil, fmt.Errorf("replicated response valid_until_ts mismatch: expected %d got %d", validUntilTS, response.ValidUntilTS)
	}
	if n.fetcher == nil {
		return nil, fmt.Errorf("replicated response verification is unavailable")
	}
	if err := n.fetcher.checkSelfSignature(response, obj); err != nil {
		return nil, fmt.Errorf("replicated response failed verification: %w", err)
	}
	return response, nil
}

// decodeBase64 accepts both raw-URL and standard base64 inputs.
func decodeBase64(v string) ([]byte, error) {
	b, err := base64.RawStdEncoding.DecodeString(v)
	if err == nil {
		return b, nil
	}
	return base64.StdEncoding.DecodeString(v)
}
