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

package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

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

	v := &visitor{
		limiter:      rate.NewLimiter(rl.globalRate, rl.globalBurst),
		queryLimiter: rate.NewLimiter(rl.queryRate, rl.queryBurst),
		lastSeen:     time.Now(),
	}
	rl.visitors[ip] = v
	return v
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
			RecordRateLimited()
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
			RecordRateLimited()
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

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(map[string]interface{}{
		"errcode": "M_LIMIT_EXCEEDED",
		"error":   "Rate limit exceeded",
	})
}
