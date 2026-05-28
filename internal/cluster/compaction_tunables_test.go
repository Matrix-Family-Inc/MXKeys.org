/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Tue Apr 22 2026 UTC
 * Status: Created
 */

package cluster

import (
	"testing"
	"time"
)

// TestEffectiveCompactionFallbacks pins the semantics of the new
// ClusterConfig compaction tunables: zero values keep the built-in
// defaults (so existing deployments with no override keep working),
// positive values take effect verbatim.
func TestEffectiveCompactionFallbacks(t *testing.T) {
	t.Run("zero interval falls back to default", func(t *testing.T) {
		c := &Cluster{config: ClusterConfig{RaftCompactionInterval: 0}}
		if got := c.effectiveCompactionInterval(); got != defaultCompactionCheckInterval {
			t.Fatalf("got %s, want default %s", got, defaultCompactionCheckInterval)
		}
	})
	t.Run("positive interval overrides default", func(t *testing.T) {
		c := &Cluster{config: ClusterConfig{RaftCompactionInterval: 5 * time.Second}}
		if got := c.effectiveCompactionInterval(); got != 5*time.Second {
			t.Fatalf("got %s, want 5s override", got)
		}
	})
	t.Run("zero threshold falls back to default", func(t *testing.T) {
		c := &Cluster{config: ClusterConfig{RaftCompactionLogThreshold: 0}}
		if got := c.effectiveCompactionLogThreshold(); got != defaultCompactionLogThreshold {
			t.Fatalf("got %d, want default %d", got, defaultCompactionLogThreshold)
		}
	})
	t.Run("positive threshold overrides default", func(t *testing.T) {
		c := &Cluster{config: ClusterConfig{RaftCompactionLogThreshold: 42}}
		if got := c.effectiveCompactionLogThreshold(); got != 42 {
			t.Fatalf("got %d, want 42 override", got)
		}
	})
}
