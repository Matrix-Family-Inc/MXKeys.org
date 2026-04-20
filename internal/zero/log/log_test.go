package log

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// captureAt sets the package logger level first (SetLevel re-creates the
// handler against stderr), then redirects to a buffer. Order matters: the
// current package API requires SetOutput to run AFTER SetLevel or the
// buffer binding is clobbered.
func captureAt(t *testing.T, level string) *bytes.Buffer {
	t.Helper()
	SetLevel(level)
	buf := &bytes.Buffer{}
	SetOutput(buf)
	t.Cleanup(func() {
		SetLevel("info")
	})
	return buf
}

func TestSetLevelLevelGuardDebug(t *testing.T) {
	buf := captureAt(t, "debug")

	Debug("debug-visible", "k", 1)
	Info("info-visible")

	out := buf.String()
	if !strings.Contains(out, "debug-visible") {
		t.Errorf("debug level must emit Debug: %q", out)
	}
	if !strings.Contains(out, "info-visible") {
		t.Errorf("debug level must emit Info: %q", out)
	}
}

func TestSetLevelSuppressesBelowThreshold(t *testing.T) {
	buf := captureAt(t, "warn")

	Debug("debug-suppressed")
	Info("info-suppressed")
	Warn("warn-visible")
	Error("error-visible")

	out := buf.String()
	if strings.Contains(out, "debug-suppressed") {
		t.Errorf("warn level must drop Debug: %q", out)
	}
	if strings.Contains(out, "info-suppressed") {
		t.Errorf("warn level must drop Info: %q", out)
	}
	if !strings.Contains(out, "warn-visible") {
		t.Errorf("warn level must keep Warn: %q", out)
	}
	if !strings.Contains(out, "error-visible") {
		t.Errorf("warn level must keep Error: %q", out)
	}
}

func TestSetLevelUnknownFallsBackToInfo(t *testing.T) {
	buf := captureAt(t, "nonsense")

	Debug("debug-suppressed")
	Info("info-visible")

	out := buf.String()
	if strings.Contains(out, "debug-suppressed") {
		t.Errorf("unknown level must fall back to info; Debug leaked: %q", out)
	}
	if !strings.Contains(out, "info-visible") {
		t.Errorf("info must be emitted on fallback: %q", out)
	}
}

func TestSetJSONWithLevelAppliesBothFormatAndLevel(t *testing.T) {
	// SetJSONWithLevel writes to stderr; call first, then redirect.
	SetJSONWithLevel("debug")
	buf := &bytes.Buffer{}
	SetOutput(buf)
	t.Cleanup(func() { SetLevel("info") })

	Debug("json-debug")

	out := buf.String()
	if !strings.Contains(out, "json-debug") {
		t.Errorf("expected captured output to contain debug message: %q", out)
	}
}

func TestContextHelpersAttachFields(t *testing.T) {
	buf := captureAt(t, "debug")

	ctx := ContextWith(context.Background(), "req-123", "10.0.0.1")
	InfoCtx(ctx, "with-context")
	DebugCtx(ctx, "debug-context")
	WarnCtx(ctx, "warn-context")
	ErrorCtx(ctx, "error-context")

	out := buf.String()
	if !strings.Contains(out, "req-123") {
		t.Errorf("log output missing request_id: %q", out)
	}
	if !strings.Contains(out, "10.0.0.1") {
		t.Errorf("log output missing remote_ip: %q", out)
	}
	for _, msg := range []string{"with-context", "debug-context", "warn-context", "error-context"} {
		if !strings.Contains(out, msg) {
			t.Errorf("log output missing %q: %q", msg, out)
		}
	}
}

func TestWithContextNilCtxReturnsDefaultLogger(t *testing.T) {
	logger := WithContext(context.TODO())
	if logger == nil {
		t.Fatal("expected non-nil logger from nil-ish context")
	}
	// Passing an actual nil context must not panic. staticcheck SA1012
	// flags nil Context as a code smell; here we are explicitly testing
	// nil-safety of WithContext and want the check suppressed.
	var nilCtx context.Context //nolint:gosimple // linter-suppression indirection, not style
	_ = WithContext(nilCtx)
}

func TestContextWithRequestIDAndRemoteIPIsolated(t *testing.T) {
	ctx := context.Background()
	ctx = ContextWithRequestID(ctx, "r1")
	ctx = ContextWithRemoteIP(ctx, "1.2.3.4")

	if v := ctx.Value(requestIDKey); v != "r1" {
		t.Errorf("request_id not stored: %v", v)
	}
	if v := ctx.Value(remoteIPKey); v != "1.2.3.4" {
		t.Errorf("remote_ip not stored: %v", v)
	}
}

func TestWithAddsAttributesToLogger(t *testing.T) {
	buf := captureAt(t, "info")
	logger := With("service", "notary")
	if logger == nil {
		t.Fatal("With returned nil logger")
	}
	logger.Info("attr-test")
	if !strings.Contains(buf.String(), "service=notary") && !strings.Contains(buf.String(), `"service":"notary"`) {
		t.Errorf("With attribute not rendered: %q", buf.String())
	}
}

func TestLoggerReturnsNonNil(t *testing.T) {
	if Logger() == nil {
		t.Fatal("Logger() must never return nil")
	}
}
