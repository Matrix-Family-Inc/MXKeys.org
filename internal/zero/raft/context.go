/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Tue Apr 21 2026 UTC
 * Status: Created
 */

package raft

import (
	"context"
	"time"
)

// ctxWithStop derives a context that cancels when the given parent
// is cancelled, when n.stopCh is closed, or after timeout elapses.
//
// It is the right context to hand to long-running background
// operations such as SendInstallSnapshot or Submit-on-behalf-of so
// that Node.Stop() unblocks them deterministically instead of
// letting them burn through a full timeout after shutdown.
//
// Caller owns the returned cancel func and MUST invoke it to
// release the watcher goroutine promptly when the operation ends
// early. The watcher goroutine exits on either stopCh close or
// ctx.Done; cancel waits for that exit so no goroutine is left
// dangling after the caller returns.
func (n *Node) ctxWithStop(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	done := make(chan struct{})
	go func() {
		select {
		case <-n.stopCh:
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
