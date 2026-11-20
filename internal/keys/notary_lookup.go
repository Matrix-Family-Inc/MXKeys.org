/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 20 Nov 2026 22:39:55 UTC
 * Status: Created
 */

package keys

import (
	"context"
	"fmt"
	"time"

	"mxkeys/internal/zero/log"
)

const (
	// minRefetchInterval is how long a fetched document counts as the
	// newest one: a minimum_valid_until_ts it does not meet triggers no
	// new fetch of the same server within this window.
	minRefetchInterval = 5 * time.Minute

	// storageReadTimeout bounds one key document read from PostgreSQL.
	storageReadTimeout = 3 * time.Second

	defaultQueryTimeout     = 25 * time.Second
	defaultExpiredRetention = 365 * 24 * time.Hour
	minFetchBudget          = 10 * time.Second
)

// GetServerKeysWithCriteria returns verified keys for serverName. When the
// known document expires before minValidUntil and was fetched long enough
// ago, the server is asked again. If that fails the known document is
// returned: a notary answers with the keys it has.
func (n *Notary) GetServerKeysWithCriteria(ctx context.Context, serverName string, minValidUntil int64) (*ServerKeysResponse, error) {
	keys, fetchedAt, err := n.lookup(ctx, serverName)
	if err != nil {
		return nil, err
	}
	if minValidUntil > 0 && keys.ValidUntilTS < minValidUntil && time.Since(fetchedAt) >= minRefetchInterval {
		recordRefetch(RefetchReasonMinValidUntil)
		fresh, err := n.fetchShared(ctx, serverName, FetchSourceRefetch)
		if err == nil {
			return fresh, nil
		}
		log.Debug("Refetch for minimum_valid_until_ts failed; serving known keys", "server", serverName, "error", err)
	}
	return keys, nil
}

// GetServerKeys returns verified keys for serverName from memory,
// PostgreSQL, or the server itself.
func (n *Notary) GetServerKeys(ctx context.Context, serverName string) (*ServerKeysResponse, error) {
	keys, _, err := n.lookup(ctx, serverName)
	return keys, err
}

// lookup returns the newest verified document for serverName and when it was
// fetched. If the server cannot be reached, the last stored document is
// returned even when expired; it is still needed to check signatures made
// while it was valid.
func (n *Notary) lookup(ctx context.Context, serverName string) (*ServerKeysResponse, time.Time, error) {
	if ctx.Err() != nil {
		return nil, time.Time{}, &KeyError{Op: "get_keys", ServerName: serverName, Err: ErrContextCanceled}
	}

	now := time.Now()
	if cached, ok := n.memoryLookup(serverName, now); ok {
		recordMemoryCacheHit()
		return cloneServerKeysResponse(cached.response), cached.fetchedAt, nil
	}
	recordMemoryCacheMiss()

	stored, storedAt, err := n.storedLookup(ctx, serverName)
	if err != nil {
		log.Warn("Database cache unavailable, continuing with remote fetch", "server", serverName, "error", err)
	}
	if stored != nil && stored.ValidUntilTS > now.UnixMilli() {
		recordDBCacheHit()
		n.storeInMemoryCache(serverName, stored, storedAt)
		return cloneServerKeysResponse(stored), storedAt, nil
	}
	recordDBCacheMiss()

	fetched, err := n.fetchShared(ctx, serverName, FetchSourceDirect)
	if err == nil {
		return fetched, time.Now(), nil
	}
	if stored != nil {
		recordStaleServe()
		log.Warn("Serving last known key document after fetch failure",
			"server", serverName,
			"valid_until", time.UnixMilli(stored.ValidUntilTS).UTC(),
			"error", err,
		)
		return stored, storedAt, nil
	}
	return nil, time.Time{}, err
}

func (n *Notary) storedLookup(ctx context.Context, serverName string) (*ServerKeysResponse, time.Time, error) {
	if n.storage == nil {
		return nil, time.Time{}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, storageReadTimeout)
	defer cancel()
	return n.storage.GetServerResponse(ctx, serverName)
}

// fetchShared fetches serverName once for all concurrent callers. The fetch
// is detached from any single caller and bounded by fetchBudget, so one
// client leaving neither fails the others nor discards the work: the
// result still reaches the caches. Each caller waits within its own
// deadline only.
func (n *Notary) fetchShared(ctx context.Context, serverName, source string) (*ServerKeysResponse, error) {
	ch := n.fetchGroup.DoChan(serverName, func() (interface{}, error) {
		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), n.fetchBudget())
		defer cancel()
		return n.fetchAndStoreWithSource(fetchCtx, serverName, source)
	})
	select {
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}
		return cloneServerKeysResponse(res.Val.(*ServerKeysResponse)), nil
	case <-ctx.Done():
		return nil, &KeyError{Op: "get_keys", ServerName: serverName, Err: fmt.Errorf("%w: %v", ErrContextCanceled, ctx.Err())}
	}
}

func (n *Notary) queryTimeout() time.Duration {
	if n.queryDeadline > 0 {
		return n.queryDeadline
	}
	return defaultQueryTimeout
}

func (n *Notary) fetchBudget() time.Duration {
	if budget := 2 * n.fetchTimeout; budget > minFetchBudget {
		return budget
	}
	return minFetchBudget
}
