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
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"mxkeys/internal/zero/log"
)

type contextKey string

const (
	requestIDKey       contextKey = "request_id"
	maxRequestIDLength            = 128
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._:/-]{1,128}$`)

// RequestIDMiddleware adds X-Request-ID header and logging context
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID, err := normalizeRequestID(r.Header.Get("X-Request-ID"))
		if err != nil || requestID == "" {
			requestID = generateRequestID()
		}

		// Set in response header
		w.Header().Set("X-Request-ID", requestID)

		// Extract remote IP
		remoteIP := extractClientIP(r)

		// Add to context for local use
		ctx := context.WithValue(r.Context(), requestIDKey, requestID)

		// Add to log context
		ctx = log.ContextWith(ctx, requestID, remoteIP)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDRequirementMiddleware rejects requests without X-Request-ID when required.
func RequestIDRequirementMiddleware(required bool, next http.Handler) http.Handler {
	if !required {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := normalizeRequestID(r.Header.Get("X-Request-ID")); err != nil {
			reason := RejectReasonInvalidRequestID
			if strings.TrimSpace(r.Header.Get("X-Request-ID")) == "" {
				reason = RejectReasonMissingRequestID
			}
			RecordRequestRejection(reason)
			w.Header().Set("Content-Type", "application/json")
			writeMatrixError(w, http.StatusBadRequest, "M_INVALID_PARAM", err.Error())
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SecurityHeadersMiddleware adds security headers to responses
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// XSS protection (legacy but still useful)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Cache control for API responses
		if isAPIPath(r.URL.Path) {
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
		}

		next.ServeHTTP(w, r)
	})
}

// GetRequestID returns the request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}

func normalizeRequestID(raw string) (string, error) {
	requestID := strings.TrimSpace(raw)
	if requestID == "" {
		return "", fmt.Errorf("X-Request-ID header is required")
	}
	if len(requestID) > maxRequestIDLength {
		return "", fmt.Errorf("X-Request-ID header is too long")
	}
	if !requestIDPattern.MatchString(requestID) {
		return "", fmt.Errorf("X-Request-ID header contains invalid characters")
	}
	return requestID, nil
}

func isAPIPath(path string) bool {
	return len(path) > 0 && (path[0] == '/' && (len(path) < 2 || path[1] != '/'))
}
