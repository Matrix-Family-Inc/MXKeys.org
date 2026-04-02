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

package log

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	remoteIPKey  contextKey = "remote_ip"
)

var (
	defaultLogger *slog.Logger
	currentLevel  slog.Level
	mu            sync.RWMutex
)

func init() {
	currentLevel = slog.LevelInfo
	defaultLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: currentLevel,
	}))
}

// SetLevel sets the log level
func SetLevel(level string) {
	mu.Lock()
	defer mu.Unlock()

	switch strings.ToLower(level) {
	case "debug":
		currentLevel = slog.LevelDebug
	case "info":
		currentLevel = slog.LevelInfo
	case "warn", "warning":
		currentLevel = slog.LevelWarn
	case "error":
		currentLevel = slog.LevelError
	default:
		currentLevel = slog.LevelInfo
	}

	defaultLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: currentLevel,
	}))
}

// SetOutput sets the log output
func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	defaultLogger = slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: currentLevel,
	}))
}

// SetJSON enables JSON format
func SetJSON() {
	mu.Lock()
	defer mu.Unlock()
	defaultLogger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: currentLevel,
	}))
}

// SetJSONWithLevel enables JSON format with specific level
func SetJSONWithLevel(level string) {
	mu.Lock()
	defer mu.Unlock()

	switch strings.ToLower(level) {
	case "debug":
		currentLevel = slog.LevelDebug
	case "info":
		currentLevel = slog.LevelInfo
	case "warn", "warning":
		currentLevel = slog.LevelWarn
	case "error":
		currentLevel = slog.LevelError
	default:
		currentLevel = slog.LevelInfo
	}

	defaultLogger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: currentLevel,
	}))
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// Info logs an info message
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// With returns a logger with additional attributes
func With(args ...any) *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return defaultLogger.With(args...)
}

// WithContext returns a logger with context values (request_id, remote_ip)
func WithContext(ctx context.Context) *slog.Logger {
	mu.RLock()
	logger := defaultLogger
	mu.RUnlock()

	if ctx == nil {
		return logger
	}

	var attrs []any

	if reqID, ok := ctx.Value(requestIDKey).(string); ok && reqID != "" {
		attrs = append(attrs, "request_id", reqID)
	}

	if remoteIP, ok := ctx.Value(remoteIPKey).(string); ok && remoteIP != "" {
		attrs = append(attrs, "remote_ip", remoteIP)
	}

	if len(attrs) > 0 {
		return logger.With(attrs...)
	}

	return logger
}

// Logger returns the default logger
func Logger() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return defaultLogger
}

// ContextWithRequestID adds request ID to context
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// ContextWithRemoteIP adds remote IP to context
func ContextWithRemoteIP(ctx context.Context, remoteIP string) context.Context {
	return context.WithValue(ctx, remoteIPKey, remoteIP)
}

// ContextWith adds both request ID and remote IP to context
func ContextWith(ctx context.Context, requestID, remoteIP string) context.Context {
	ctx = context.WithValue(ctx, requestIDKey, requestID)
	ctx = context.WithValue(ctx, remoteIPKey, remoteIP)
	return ctx
}

// InfoCtx logs info with context
func InfoCtx(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Info(msg, args...)
}

// DebugCtx logs debug with context
func DebugCtx(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Debug(msg, args...)
}

// WarnCtx logs warning with context
func WarnCtx(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Warn(msg, args...)
}

// ErrorCtx logs error with context
func ErrorCtx(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Error(msg, args...)
}
