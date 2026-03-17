/*
 * Project: MXKeys - Matrix Federation Trust Infrastructure
 * Company: Matrix.Family Inc. - Delaware C-Corp
 * Dev: Brabus
 * Date: Sat Mar 15 2026 UTC
 * Status: Created
 * Contact: @support:matrix.family
 */

package keys

import (
	"sync"
	"time"
)

// CircuitState represents the circuit breaker state
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// CircuitBreaker prevents repeated calls to failing upstreams
type CircuitBreaker struct {
	mu sync.RWMutex

	// Configuration
	failureThreshold int
	resetTimeout     time.Duration
	halfOpenMax      int

	// Per-server state
	servers map[string]*serverCircuit
}

type serverCircuit struct {
	state        CircuitState
	failures     int
	lastFailure  time.Time
	halfOpenReqs int
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	if failureThreshold <= 0 {
		failureThreshold = 5
	}
	if resetTimeout <= 0 {
		resetTimeout = 60 * time.Second
	}
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		halfOpenMax:      1,
		servers:          make(map[string]*serverCircuit),
	}
}

// Allow checks if a request to the server should be allowed
func (cb *CircuitBreaker) Allow(serverName string) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	sc, exists := cb.servers[serverName]
	if !exists {
		return true
	}

	now := time.Now()

	switch sc.state {
	case CircuitClosed:
		return true

	case CircuitOpen:
		// Check if we should transition to half-open
		if now.Sub(sc.lastFailure) > cb.resetTimeout {
			sc.state = CircuitHalfOpen
			sc.halfOpenReqs = 0
			return true
		}
		return false

	case CircuitHalfOpen:
		// Allow limited requests in half-open state
		if sc.halfOpenReqs < cb.halfOpenMax {
			sc.halfOpenReqs++
			return true
		}
		return false
	}

	return true
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess(serverName string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	sc, exists := cb.servers[serverName]
	if !exists {
		return
	}

	// Reset on success
	if sc.state == CircuitHalfOpen {
		sc.state = CircuitClosed
		sc.failures = 0
	} else if sc.state == CircuitClosed {
		sc.failures = 0
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure(serverName string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	sc, exists := cb.servers[serverName]
	if !exists {
		sc = &serverCircuit{state: CircuitClosed}
		cb.servers[serverName] = sc
	}

	sc.failures++
	sc.lastFailure = time.Now()

	switch sc.state {
	case CircuitClosed:
		if sc.failures >= cb.failureThreshold {
			sc.state = CircuitOpen
		}

	case CircuitHalfOpen:
		// Failure in half-open state → back to open
		sc.state = CircuitOpen
	}
}

// State returns the current state for a server
func (cb *CircuitBreaker) State(serverName string) CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	sc, exists := cb.servers[serverName]
	if !exists {
		return CircuitClosed
	}
	return sc.state
}

// Reset resets the circuit for a server
func (cb *CircuitBreaker) Reset(serverName string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	delete(cb.servers, serverName)
}

// Stats returns circuit breaker statistics
func (cb *CircuitBreaker) Stats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	open := 0
	halfOpen := 0
	for _, sc := range cb.servers {
		switch sc.state {
		case CircuitOpen:
			open++
		case CircuitHalfOpen:
			halfOpen++
		}
	}

	return map[string]interface{}{
		"total_servers": len(cb.servers),
		"open":          open,
		"half_open":     halfOpen,
	}
}
