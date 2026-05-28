/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

package server

import (
	"container/list"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// defaultMaxVisitors is the fallback capacity when config does not specify one.
const defaultMaxVisitors = 100000

// RateLimiter enforces per-IP request rate limits with an LRU cache of
// visitor entries. The LRU provides O(1) get, insert, and eviction under
// bounded memory when the number of unique client IPs exceeds capacity.
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*list.Element
	order    *list.List // front = most recently used, back = oldest

	globalRate   rate.Limit
	globalBurst  int
	queryRate    rate.Limit
	queryBurst   int
	maxVisitors  int
	cleanupEvery time.Duration
	idleTTL      time.Duration
	stopCh       chan struct{}
	stopOnce     sync.Once
}

type visitor struct {
	ip           string
	limiter      *rate.Limiter
	queryLimiter *rate.Limiter
	lastSeen     time.Time
}

// RateLimitConfig describes the rate-limit bounds for both the global limiter
// (applied to all routes) and the tighter query limiter (applied to the
// POST /_matrix/key/v2/query hot path).
type RateLimitConfig struct {
	GlobalRequestsPerSecond float64
	GlobalBurst             int
	QueryRequestsPerSecond  float64
	QueryBurst              int
	// MaxVisitors caps the number of distinct client IPs tracked
	// simultaneously. Exceeding it evicts the LRU tail in O(1).
	// Zero or negative means use defaultMaxVisitors.
	MaxVisitors int
	// IdleTTL is the cutoff for the background cleanup loop: visitors not
	// seen for IdleTTL are dropped. Zero means 10 minutes.
	IdleTTL time.Duration
}

// DefaultRateLimitConfig returns a conservative default tuned for small
// operators; production deployments should override via config.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		GlobalRequestsPerSecond: 100,
		GlobalBurst:             200,
		QueryRequestsPerSecond:  10,
		QueryBurst:              20,
		MaxVisitors:             defaultMaxVisitors,
		IdleTTL:                 10 * time.Minute,
	}
}

// NewRateLimiter constructs a RateLimiter and starts its background cleanup
// goroutine. Call Stop to release it.
func NewRateLimiter(cfg RateLimitConfig) *RateLimiter {
	if cfg.MaxVisitors <= 0 {
		cfg.MaxVisitors = defaultMaxVisitors
	}
	if cfg.IdleTTL <= 0 {
		cfg.IdleTTL = 10 * time.Minute
	}
	rl := &RateLimiter{
		visitors:     make(map[string]*list.Element, cfg.MaxVisitors/8+1),
		order:        list.New(),
		globalRate:   rate.Limit(cfg.GlobalRequestsPerSecond),
		globalBurst:  cfg.GlobalBurst,
		queryRate:    rate.Limit(cfg.QueryRequestsPerSecond),
		queryBurst:   cfg.QueryBurst,
		maxVisitors:  cfg.MaxVisitors,
		cleanupEvery: 5 * time.Minute,
		idleTTL:      cfg.IdleTTL,
		stopCh:       make(chan struct{}),
	}

	go rl.cleanupLoop()

	return rl
}

// getVisitor returns the visitor for ip, creating it on first use. Promoted
// to the LRU front in O(1). Inserts that exceed capacity evict the LRU tail.
func (rl *RateLimiter) getVisitor(ip string) *visitor {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if el, ok := rl.visitors[ip]; ok {
		v := el.Value.(*visitor)
		v.lastSeen = time.Now()
		rl.order.MoveToFront(el)
		return v
	}

	if rl.order.Len() >= rl.maxVisitors {
		rl.evictOldestLocked()
	}

	v := &visitor{
		ip:           ip,
		limiter:      rate.NewLimiter(rl.globalRate, rl.globalBurst),
		queryLimiter: rate.NewLimiter(rl.queryRate, rl.queryBurst),
		lastSeen:     time.Now(),
	}
	el := rl.order.PushFront(v)
	rl.visitors[ip] = el
	return v
}

// evictOldestLocked drops the LRU tail entry. Caller holds rl.mu.
func (rl *RateLimiter) evictOldestLocked() {
	el := rl.order.Back()
	if el == nil {
		return
	}
	v := el.Value.(*visitor)
	delete(rl.visitors, v.ip)
	rl.order.Remove(el)
}

// cleanupLoop periodically drops idle visitors older than idleTTL.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupEvery)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stopCh:
			return
		case <-ticker.C:
			rl.cleanup()
		}
	}
}

// Stop releases the background cleanup goroutine.
func (rl *RateLimiter) Stop() {
	rl.stopOnce.Do(func() {
		close(rl.stopCh)
	})
}

// cleanup sweeps visitors not seen for longer than idleTTL. Walks the LRU
// from tail forward and stops as soon as it hits an entry younger than the
// cutoff (LRU order implies everything ahead is newer).
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-rl.idleTTL)
	for el := rl.order.Back(); el != nil; {
		v := el.Value.(*visitor)
		if !v.lastSeen.Before(cutoff) {
			break
		}
		prev := el.Prev()
		delete(rl.visitors, v.ip)
		rl.order.Remove(el)
		el = prev
	}
}

// Middleware applies the global per-IP rate limit.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		v := rl.getVisitor(ip)

		if !v.limiter.Allow() {
			RecordRateLimited("global")
			writeRateLimitError(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// QueryMiddleware applies the tighter query-path per-IP rate limit on top of
// the global limiter; both must admit the request.
func (rl *RateLimiter) QueryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		v := rl.getVisitor(ip)

		if !v.queryLimiter.Allow() {
			RecordRateLimited("query")
			writeRateLimitError(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeRateLimitError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "1")
	w.WriteHeader(http.StatusTooManyRequests)

	writeJSON(w, map[string]interface{}{
		"errcode": "M_LIMIT_EXCEEDED",
		"error":   "Rate limit exceeded",
	})
}
