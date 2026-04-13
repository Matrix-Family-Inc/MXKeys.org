/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Sat Mar 15 2026 UTC
 * Status: Created
 */

package keys

import (
	"sync"
	"time"

	"mxkeys/internal/zero/metrics"
)

// CircuitState represents the circuit breaker state
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

var (
	circuitStateGauge = metrics.NewGaugeVec(
		metrics.GaugeOpts{
			Namespace: "mxkeys",
			Subsystem: "circuit_breaker",
			Name:      "servers",
			Help:      "Number of tracked upstream servers by circuit state",
		},
		[]string{"state"},
	)
	circuitTripsTotal = metrics.NewCounter(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Subsystem: "circuit_breaker",
			Name:      "trips_total",
			Help:      "Total number of circuit breaker trips (closed -> open)",
		},
	)
	circuitRecoveriesTotal = metrics.NewCounter(
		metrics.CounterOpts{
			Namespace: "mxkeys",
			Subsystem: "circuit_breaker",
			Name:      "recoveries_total",
			Help:      "Total number of circuit breaker recoveries (half-open -> closed)",
		},
	)
)

// CircuitBreaker prevents repeated calls to failing upstreams
type CircuitBreaker struct {
	mu sync.RWMutex

	// Configuration
	failureThreshold int
	resetTimeout     time.Duration
	halfOpenMax      int
	serverTTL        time.Duration
	maxTracked       int

	// Per-server state
	servers map[string]*serverCircuit
}

type serverCircuit struct {
	state        CircuitState
	failures     int
	lastFailure  time.Time
	lastTouched  time.Time
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
		serverTTL:        maxDuration(15*time.Minute, resetTimeout*10),
		maxTracked:       4096,
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
	if cb.isExpiredLocked(sc, now) {
		delete(cb.servers, serverName)
		return true
	}
	sc.lastTouched = now

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
	sc.lastTouched = time.Now()

	if sc.state == CircuitHalfOpen {
		sc.state = CircuitClosed
		sc.failures = 0
		sc.halfOpenReqs = 0
		circuitRecoveriesTotal.Inc()
	} else if sc.state == CircuitClosed {
		sc.failures = 0
	}

	cb.updateMetricsLocked()
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure(serverName string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	cb.cleanupLocked(now)
	sc, exists := cb.servers[serverName]
	if !exists {
		if len(cb.servers) >= cb.maxTracked {
			cb.evictOldestLocked()
		}
		sc = &serverCircuit{
			state:       CircuitClosed,
			lastTouched: now,
		}
		cb.servers[serverName] = sc
	}

	sc.failures++
	sc.lastFailure = now
	sc.lastTouched = now

	switch sc.state {
	case CircuitClosed:
		if sc.failures >= cb.failureThreshold {
			sc.state = CircuitOpen
			circuitTripsTotal.Inc()
		}

	case CircuitHalfOpen:
		sc.state = CircuitOpen
		circuitTripsTotal.Inc()
	}

	cb.updateMetricsLocked()
}

// State returns the current state for a server
func (cb *CircuitBreaker) State(serverName string) CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	sc, exists := cb.servers[serverName]
	if !exists {
		return CircuitClosed
	}
	if cb.isExpiredLocked(sc, time.Now()) {
		delete(cb.servers, serverName)
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
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.cleanupLocked(time.Now())

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

func (cb *CircuitBreaker) updateMetricsLocked() {
	var closed, open, halfOpen int
	for _, sc := range cb.servers {
		switch sc.state {
		case CircuitClosed:
			closed++
		case CircuitOpen:
			open++
		case CircuitHalfOpen:
			halfOpen++
		}
	}
	circuitStateGauge.WithLabelValues("closed").Set(float64(closed))
	circuitStateGauge.WithLabelValues("open").Set(float64(open))
	circuitStateGauge.WithLabelValues("half_open").Set(float64(halfOpen))
}

func (cb *CircuitBreaker) cleanupLocked(now time.Time) {
	for serverName, sc := range cb.servers {
		if cb.isExpiredLocked(sc, now) {
			delete(cb.servers, serverName)
		}
	}
}

func (cb *CircuitBreaker) isExpiredLocked(sc *serverCircuit, now time.Time) bool {
	return sc == nil || now.Sub(sc.lastTouched) > cb.serverTTL
}

func (cb *CircuitBreaker) evictOldestLocked() {
	var (
		oldestName string
		oldestTime time.Time
	)
	for serverName, sc := range cb.servers {
		if oldestName == "" || sc.lastTouched.Before(oldestTime) {
			oldestName = serverName
			oldestTime = sc.lastTouched
		}
	}
	if oldestName != "" {
		delete(cb.servers, oldestName)
	}
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}
