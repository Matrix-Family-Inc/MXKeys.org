/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix.Family Inc. (https://matrix.family)
 * Owner: Matrix.Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Mon 22 Jun 2026 00:50:40 UTC
 * Status: Updated
 */

package keys

import (
	"context"
	"testing"
	"time"
)

func TestHalfOpenAdmitsOneProbeAtATime(t *testing.T) {
	cb := NewCircuitBreaker(1, 10*time.Millisecond)
	cb.RecordFailure("s")
	time.Sleep(20 * time.Millisecond)

	if !cb.Allow("s") {
		t.Fatal("the first request after the reset timeout must be admitted as a probe")
	}
	if cb.Allow("s") {
		t.Fatal("a second probe must wait for the first one's outcome")
	}
	cb.Release("s")
	if !cb.Allow("s") {
		t.Fatal("a released probe slot must admit the next probe")
	}
}

func TestAbandonedProbesDoNotWedgeTheCircuit(t *testing.T) {
	f := NewFetcherWithConfig(FetcherConfig{Timeout: time.Second})
	f.circuitBreaker = NewCircuitBreaker(1, 10*time.Millisecond)
	f.circuitBreaker.RecordFailure("busy.example")
	time.Sleep(20 * time.Millisecond)

	if err := f.fetchSem.Acquire(context.Background(), maxConcurrentFetches); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		_, _ = f.FetchServerKeys(ctx, "busy.example")
		cancel()
	}
	f.fetchSem.Release(maxConcurrentFetches)

	if !f.circuitBreaker.Allow("busy.example") {
		t.Fatal("probes abandoned on the fetch semaphore must not keep the circuit closed to new probes")
	}
}

func TestNewBreakerIsClosed(t *testing.T) {
	cb := NewCircuitBreaker(5, time.Minute)
	if state := cb.State("test.server"); state != CircuitClosed {
		t.Errorf("new breaker should be closed, got %v", state)
	}
}

func TestBreakerAllowsWhenClosed(t *testing.T) {
	cb := NewCircuitBreaker(5, time.Minute)
	if !cb.Allow("test.server") {
		t.Error("closed breaker should allow requests")
	}
}

func TestBreakerOpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Minute)
	server := "failing.server"

	for i := 0; i < 3; i++ {
		cb.RecordFailure(server)
	}

	if state := cb.State(server); state != CircuitOpen {
		t.Errorf("breaker should be open after %d failures, got %v", 3, state)
	}
}

func TestOpenBreakerRejectsRequests(t *testing.T) {
	cb := NewCircuitBreaker(2, time.Minute)
	server := "failing.server"

	cb.RecordFailure(server)
	cb.RecordFailure(server)

	if cb.Allow(server) {
		t.Error("open breaker should reject requests")
	}
}

func TestBreakerTransitionsToHalfOpenAfterTimeout(t *testing.T) {
	cb := NewCircuitBreaker(2, 10*time.Millisecond)
	server := "failing.server"

	cb.RecordFailure(server)
	cb.RecordFailure(server)

	if cb.State(server) != CircuitOpen {
		t.Fatal("breaker should be open")
	}

	time.Sleep(20 * time.Millisecond)

	if !cb.Allow(server) {
		t.Error("breaker should allow request after timeout (half-open)")
	}

	if cb.State(server) != CircuitHalfOpen {
		t.Errorf("breaker should be half-open, got %v", cb.State(server))
	}
}

func TestSuccessResetsBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Minute)
	server := "test.server"

	cb.RecordFailure(server)
	cb.RecordFailure(server)

	cb.RecordSuccess(server)

	if state := cb.State(server); state != CircuitClosed {
		t.Errorf("success should reset breaker to closed, got %v", state)
	}

	for i := 0; i < 3; i++ {
		cb.RecordFailure(server)
	}
	if cb.State(server) != CircuitOpen {
		t.Error("after reset, should need full threshold again")
	}
}

func TestSuccessInHalfOpenCloses(t *testing.T) {
	cb := NewCircuitBreaker(2, 10*time.Millisecond)
	server := "test.server"

	cb.RecordFailure(server)
	cb.RecordFailure(server)

	time.Sleep(20 * time.Millisecond)
	cb.Allow(server)

	cb.RecordSuccess(server)

	if state := cb.State(server); state != CircuitClosed {
		t.Errorf("success in half-open should close breaker, got %v", state)
	}
}

func TestFailureInHalfOpenReopens(t *testing.T) {
	cb := NewCircuitBreaker(2, 10*time.Millisecond)
	server := "test.server"

	cb.RecordFailure(server)
	cb.RecordFailure(server)

	time.Sleep(20 * time.Millisecond)
	cb.Allow(server)

	cb.RecordFailure(server)

	if state := cb.State(server); state != CircuitOpen {
		t.Errorf("failure in half-open should reopen breaker, got %v", state)
	}
}

func TestBreakerIsolatesServers(t *testing.T) {
	cb := NewCircuitBreaker(2, time.Minute)
	failing := "failing.server"
	healthy := "healthy.server"

	cb.RecordFailure(failing)
	cb.RecordFailure(failing)

	if !cb.Allow(healthy) {
		t.Error("healthy server should not be affected by failing server")
	}

	if cb.Allow(failing) {
		t.Error("failing server should be blocked")
	}
}

func TestResetClearsState(t *testing.T) {
	cb := NewCircuitBreaker(2, time.Minute)
	server := "test.server"

	cb.RecordFailure(server)
	cb.RecordFailure(server)

	cb.Reset(server)

	if state := cb.State(server); state != CircuitClosed {
		t.Errorf("reset should return to closed, got %v", state)
	}
	if !cb.Allow(server) {
		t.Error("reset server should allow requests")
	}
}

func TestStats(t *testing.T) {
	cb := NewCircuitBreaker(2, time.Minute)

	cb.RecordFailure("server1")
	cb.RecordFailure("server1")

	cb.RecordFailure("server2")

	stats := cb.Stats()

	if stats["total_servers"].(int) != 2 {
		t.Errorf("expected 2 tracked servers, got %v", stats["total_servers"])
	}
	if stats["open"].(int) != 1 {
		t.Errorf("expected 1 open circuit, got %v", stats["open"])
	}
}
