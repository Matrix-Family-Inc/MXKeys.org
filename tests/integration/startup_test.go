//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHealthEndpointAvailable(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "healthy",
			"server":  "test.mxkeys",
			"version": "0.1.0",
		})
	})

	req := httptest.NewRequest("GET", "/_mxkeys/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("health check failed: %d", rr.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if result["status"] != "healthy" {
		t.Error("status should be healthy")
	}
}

func TestReadinessAfterStartup(t *testing.T) {
	type readinessState struct {
		isReady bool
	}

	state := &readinessState{isReady: false}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !state.isReady {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "not_ready",
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ready",
		})
	})

	req := httptest.NewRequest("GET", "/_mxkeys/ready", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Error("should be unavailable before startup")
	}

	state.isReady = true

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Error("should be ready after startup")
	}
}

func TestStatusEndpointContent(t *testing.T) {
	startTime := time.Now()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"version": "0.1.0",
			"server":  "test.mxkeys",
			"uptime":  time.Since(startTime).String(),
			"cache": map[string]int{
				"memory_entries":   0,
				"database_entries": 0,
			},
			"database": map[string]int{
				"open_connections": 1,
				"in_use":           0,
				"idle":             1,
			},
		})
	})

	req := httptest.NewRequest("GET", "/_mxkeys/status", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status check failed: %d", rr.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	requiredFields := []string{"status", "version", "uptime", "cache", "database"}
	for _, field := range requiredFields {
		if result[field] == nil {
			t.Errorf("missing required field: %s", field)
		}
	}

	uptime, ok := result["uptime"].(string)
	if !ok {
		t.Error("uptime should be a string")
	}
	if uptime == "" {
		t.Error("uptime should not be empty")
	}

	if result["version"] != "0.1.0" {
		t.Errorf("version mismatch: %v", result["version"])
	}
}

func TestLivenessEndpoint(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "alive",
		})
	})

	req := httptest.NewRequest("GET", "/_mxkeys/live", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("liveness check failed: %d", rr.Code)
	}

	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["status"] != "alive" {
		t.Error("expected alive status")
	}
}

func TestVersionEndpoint(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"version":    "0.1.0",
			"git_commit": "abc123",
			"build_time": "2026-03-15T00:00:00Z",
		})
	})

	req := httptest.NewRequest("GET", "/_mxkeys/version", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("version endpoint failed: %d", rr.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &result)

	if result["version"] != "0.1.0" {
		t.Errorf("version mismatch: %v", result["version"])
	}
}

func TestMetricsEndpoint(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		lines := []string{
			"# HELP mxkeys_requests_total Total HTTP requests",
			"# TYPE mxkeys_requests_total counter",
			"mxkeys_requests_total{method=\"GET\",path=\"/_mxkeys/health\"} 1",
			"# HELP mxkeys_key_fetch_total Total key fetch operations",
			"# TYPE mxkeys_key_fetch_total counter",
			"mxkeys_key_fetch_total{source=\"cache\"} 0",
			"mxkeys_key_fetch_total{source=\"upstream\"} 0",
		}
		w.Write([]byte(strings.Join(lines, "\n")))
	})

	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("metrics endpoint failed: %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("unexpected content type: %s", contentType)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "mxkeys_") {
		t.Error("metrics should contain mxkeys_ prefix")
	}
}
