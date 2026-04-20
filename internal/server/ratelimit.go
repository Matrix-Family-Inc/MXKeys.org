/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Sat Mar 15 2026 UTC
 * Status: Created
 */

package server

import (
	"net/http"
	"sort"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const maxVisitors = 100000 // Maximum unique IPs to track before aggressive eviction

type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex

	globalRate   rate.Limit
	globalBurst  int
	queryRate    rate.Limit
	queryBurst   int
	cleanupEvery time.Duration
	stopCh       chan struct{}
	stopOnce     sync.Once
}

type visitor struct {
	limiter      *rate.Limiter
	queryLimiter *rate.Limiter
	lastSeen     time.Time
}

type visitorSnapshot struct {
	ip       string
	lastSeen time.Time
}

type RateLimitConfig struct {
	GlobalRequestsPerSecond float64
	GlobalBurst             int
	QueryRequestsPerSecond  float64
	QueryBurst              int
}

func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		GlobalRequestsPerSecond: 100,
		GlobalBurst:             200,
		QueryRequestsPerSecond:  10,
		QueryBurst:              20,
	}
}

func NewRateLimiter(cfg RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		visitors:     make(map[string]*visitor),
		globalRate:   rate.Limit(cfg.GlobalRequestsPerSecond),
		globalBurst:  cfg.GlobalBurst,
		queryRate:    rate.Limit(cfg.QueryRequestsPerSecond),
		queryBurst:   cfg.QueryBurst,
		cleanupEvery: 5 * time.Minute,
		stopCh:       make(chan struct{}),
	}

	go rl.cleanupLoop()

	return rl
}

func (rl *RateLimiter) getVisitor(ip string) *visitor {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if v, exists := rl.visitors[ip]; exists {
		v.lastSeen = time.Now()
		return v
	}

	// Protect against memory exhaustion from too many unique IPs
	if len(rl.visitors) >= maxVisitors {
		rl.evictOldestLocked()
	}

	v := &visitor{
		limiter:      rate.NewLimiter(rl.globalRate, rl.globalBurst),
		queryLimiter: rate.NewLimiter(rl.queryRate, rl.queryBurst),
		lastSeen:     time.Now(),
	}
	rl.visitors[ip] = v
	return v
}

// evictOldestLocked removes oldest entries when map is full. Caller must hold lock.
func (rl *RateLimiter) evictOldestLocked() {
	threshold := time.Now().Add(-1 * time.Minute)
	evicted := 0
	targetEvictions := maxVisitors / 10
	if targetEvictions < 1 {
		targetEvictions = 1
	}

	// First pass: evict entries older than threshold
	for ip, v := range rl.visitors {
		if v.lastSeen.Before(threshold) {
			delete(rl.visitors, ip)
			evicted++
			if evicted >= targetEvictions {
				return
			}
		}
	}

	// If still at capacity, force evict oldest entries
	if len(rl.visitors) >= maxVisitors {
		// Partial sort to find N oldest (selection algorithm)
		toEvict := targetEvictions - evicted
		for _, ip := range oldestVisitorIPs(rl.visitors, toEvict) {
			delete(rl.visitors, ip)
		}
	}
}

func oldestVisitorIPs(visitors map[string]*visitor, limit int) []string {
	if limit <= 0 || len(visitors) == 0 {
		return nil
	}

	entries := make([]visitorSnapshot, 0, len(visitors))
	for ip, v := range visitors {
		if v == nil {
			continue
		}
		entries = append(entries, visitorSnapshot{
			ip:       ip,
			lastSeen: v.lastSeen,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].lastSeen.Before(entries[j].lastSeen)
	})

	if limit > len(entries) {
		limit = len(entries)
	}

	ips := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		ips = append(ips, entries[i].ip)
	}
	return ips
}

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

func (rl *RateLimiter) Stop() {
	rl.stopOnce.Do(func() {
		close(rl.stopCh)
	})
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	threshold := time.Now().Add(-10 * time.Minute)
	for ip, v := range rl.visitors {
		if v.lastSeen.Before(threshold) {
			delete(rl.visitors, ip)
		}
	}
}

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
