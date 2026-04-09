/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Mon Mar 16 2026 UTC
 * Status: Created
 */

package server

import (
	"database/sql"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"mxkeys/internal/config"
)

func TestGracefulShutdownStopsServerAndClosesDB(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://mxkeys:mxkeys@127.0.0.1:1/mxkeys?sslmode=disable")
	if err != nil {
		t.Fatalf("failed to open db handle: %v", err)
	}
	rl := NewRateLimiter(DefaultRateLimitConfig())

	s := &Server{db: db, rateLimiter: rl}
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(20 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}),
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()
	go func() {
		_ = srv.Serve(ln)
	}()

	if err := s.gracefulShutdown(srv); err != nil {
		t.Fatalf("gracefulShutdown returned error: %v", err)
	}

	if err := db.Ping(); err == nil {
		t.Fatalf("database should be closed after gracefulShutdown")
	} else if !strings.Contains(strings.ToLower(err.Error()), "closed") {
		t.Fatalf("expected closed-db error after shutdown, got: %v", err)
	}

	// Must be idempotent after gracefulShutdown.
	rl.Stop()
}

func TestCloseHandlesNilComponentsAndIsIdempotentForRateLimiter(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://mxkeys:mxkeys@127.0.0.1:1/mxkeys?sslmode=disable")
	if err != nil {
		t.Fatalf("failed to open db handle: %v", err)
	}

	s := &Server{
		db:          db,
		rateLimiter: NewRateLimiter(DefaultRateLimitConfig()),
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close() returned error: %v", err)
	}
	s.rateLimiter.Stop()
	s.rateLimiter.Stop()
}

func TestNewHTTPServerAppliesHeaderHardening(t *testing.T) {
	rl := NewRateLimiter(DefaultRateLimitConfig())
	defer rl.Stop()

	s := &Server{
		config:      &config.Config{},
		mux:         http.NewServeMux(),
		rateLimiter: rl,
	}

	srv := s.newHTTPServer("127.0.0.1:8448")
	if srv.ReadHeaderTimeout != 10*time.Second {
		t.Fatalf("ReadHeaderTimeout = %v, want 10s", srv.ReadHeaderTimeout)
	}
	if srv.MaxHeaderBytes != 1<<16 {
		t.Fatalf("MaxHeaderBytes = %d, want %d", srv.MaxHeaderBytes, 1<<16)
	}
}

func TestDeploymentGuideContainsRestartPolicy(t *testing.T) {
	data, err := os.ReadFile("../../docs/deployment.md")
	if err != nil {
		t.Fatalf("failed to read deployment guide: %v", err)
	}
	content := string(data)

	required := []string{
		"Restart=always",
		"RestartSec=5",
		"NoNewPrivileges=true",
		"X-Request-ID $request_id",
	}
	for _, needle := range required {
		if !strings.Contains(content, needle) {
			t.Fatalf("deployment guide is missing required restart policy setting: %s", needle)
		}
	}
}
