/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Tue Apr 21 2026 UTC
 * Status: Created
 */

package cluster

import (
	"context"
	"time"
)

// proposeTimeout is the upper bound BroadcastKeyUpdate waits for a
// raft-mode write to commit. It matches raft's default
// CommitTimeout so a Propose cannot block the application hot path
// longer than a single commit round-trip.
const proposeTimeout = 5 * time.Second

// proposeCtx returns a context bounded by timeout that also cancels
// the moment c.stopCh closes. This is the right context for
// Propose/Submit calls issued from application hot paths
// (BroadcastKeyUpdate): it guarantees a shutdown can evict an
// in-flight proposal without waiting for the full raft
// CommitTimeout.
//
// Caller owns the returned cancel func and MUST invoke it to
// release the watcher goroutine. The watcher exits on either
// stopCh close or ctx.Done; cancel waits for that exit so no
// goroutine is left dangling after the caller returns.
func (c *Cluster) proposeCtx(timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	done := make(chan struct{})
	go func() {
		select {
		case <-c.stopCh:
			cancel()
		case <-ctx.Done():
		}
		close(done)
	}()
	return ctx, func() {
		cancel()
		<-done
	}
}
