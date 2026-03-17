/*
Project: MXKeys - Matrix Federation Trust Infrastructure
Company: Matrix.Family Inc. - Delaware C-Corp
Dev: Brabus
Date: Mon Mar 16 2026 UTC
Status: Created
Contact: @support:matrix.family
*/

package server

import (
	"database/sql"
	"net/http"
	"os"
	"strings"
	"testing"

	_ "github.com/lib/pq"
)

func TestGracefulShutdownCompletesWithoutError(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://mxkeys:mxkeys@127.0.0.1:1/mxkeys?sslmode=disable")
	if err != nil {
		t.Fatalf("failed to open db handle: %v", err)
	}

	s := &Server{db: db}
	srv := &http.Server{}

	if err := s.gracefulShutdown(srv); err != nil {
		t.Fatalf("gracefulShutdown returned error: %v", err)
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
	}
	for _, needle := range required {
		if !strings.Contains(content, needle) {
			t.Fatalf("deployment guide is missing required restart policy setting: %s", needle)
		}
	}
}
