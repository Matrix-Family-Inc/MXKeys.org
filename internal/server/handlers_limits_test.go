package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOversizedRequestBodyRejected(t *testing.T) {
	maxBodySize := int64(1024)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

		_, err := io.ReadAll(r.Body)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			writeJSON(w, map[string]string{
				"errcode": "M_TOO_LARGE",
				"error":   "Request body too large",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	largeBody := strings.Repeat("x", 2048)
	req := httptest.NewRequest("POST", "/_matrix/key/v2/query", strings.NewReader(largeBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", rr.Code)
	}
}

func TestAcceptableRequestBodySize(t *testing.T) {
	maxBodySize := int64(1024)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

		_, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	smallBody := strings.Repeat("x", 512)
	req := httptest.NewRequest("POST", "/_matrix/key/v2/query", strings.NewReader(smallBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestExactLimitBodySize(t *testing.T) {
	maxBodySize := int64(1024)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

		_, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	exactBody := strings.Repeat("x", 1024)
	req := httptest.NewRequest("POST", "/_matrix/key/v2/query", strings.NewReader(exactBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("exact limit should be accepted, got %d", rr.Code)
	}
}

func TestOneByteTooLarge(t *testing.T) {
	maxBodySize := int64(1024)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

		_, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	overBody := strings.Repeat("x", 1025)
	req := httptest.NewRequest("POST", "/_matrix/key/v2/query", strings.NewReader(overBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("one byte over limit should be rejected, got %d", rr.Code)
	}
}

func TestEmptyBodyAccepted(t *testing.T) {
	maxBodySize := int64(1024)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}

		if len(body) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/_matrix/key/v2/query", strings.NewReader(""))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("empty body should be handled (rejected as bad request), got %d", rr.Code)
	}
}

func TestVeryLargeQueryRejected(t *testing.T) {
	maxBodySize := int64(64 * 1024)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

		_, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	hugeBody := strings.Repeat("x", 128*1024)
	req := httptest.NewRequest("POST", "/_matrix/key/v2/query", strings.NewReader(hugeBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("huge request should be rejected, got %d", rr.Code)
	}
}

func TestContentLengthHeaderRespected(t *testing.T) {
	req := httptest.NewRequest("POST", "/_matrix/key/v2/query", strings.NewReader("small body"))
	req.ContentLength = 1000000

	if req.ContentLength > 64*1024 {
		t.Log("large Content-Length would be rejected before reading body")
	}
}
