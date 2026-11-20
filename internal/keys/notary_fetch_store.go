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

// Fetch/store side-effects for notary queries. Kept separate from
// notary_query.go so the query orchestration and the
// fetch-and-persist plumbing stay individually focused and the
// per-file line budget remains under the house rule.

package keys

import (
	"context"
	"encoding/base64"
	"time"

	"mxkeys/internal/zero/log"
)

// fetchAndStore fetches keys from remote and stores them.
func (n *Notary) fetchAndStore(ctx context.Context, serverName string) (*ServerKeysResponse, error) {
	return n.fetchAndStoreWithSource(ctx, serverName, FetchSourceDirect)
}

// fetchAndStoreWithSource fetches keys with source tracking for metrics,
// persists them to storage, updates the in-memory cache, and
// broadcasts the raw origin payload to cluster peers when a broadcast
// hook is installed.
func (n *Notary) fetchAndStoreWithSource(ctx context.Context, serverName, source string) (*ServerKeysResponse, error) {
	start := time.Now()
	keys, err := n.fetcher.FetchServerKeys(ctx, serverName)
	duration := time.Since(start).Seconds()
	if err != nil {
		recordFetchFailure(source, duration)
		return nil, err
	}
	recordFetchSuccess(source, duration)

	validUntil := time.UnixMilli(keys.ValidUntilTS)
	if err := n.storage.StoreServerResponse(serverName, keys, validUntil); err != nil {
		log.Warn("Failed to store server response in database", "error", err)
	}

	for keyID, verifyKey := range keys.VerifyKeys {
		pubKeyBytes, err := base64.RawStdEncoding.DecodeString(verifyKey.Key)
		if err != nil {
			continue
		}
		if err := n.storage.StoreKey(serverName, keyID, pubKeyBytes, validUntil); err != nil {
			log.Warn("Failed to store key", "error", err)
		}
	}

	n.storeInMemoryCache(serverName, keys)

	// Peers re-verify the origin self-signature on these exact bytes.
	if broadcastHook := n.getKeyBroadcastHook(); broadcastHook != nil && len(keys.Raw) > 0 {
		broadcastHook(serverName, ClusterReplicatedResponseKeyID, string(keys.Raw), keys.ValidUntilTS)
	}

	return keys, nil
}
