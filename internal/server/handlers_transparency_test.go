/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 03 Apr 2026 UTC
 * Status: Created
 */

package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mxkeys/internal/keys"
)

func TestTransparencyLogReturns404WhenDisabled(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/transparency/log", nil)
	rr := httptest.NewRecorder()

	s.handleTransparencyLog(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "M_NOT_FOUND") {
		t.Fatalf("expected M_NOT_FOUND body, got %s", rr.Body.String())
	}
}

func TestTransparencyVerifyReturns404WhenDisabled(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/transparency/verify", nil)
	rr := httptest.NewRecorder()

	s.handleTransparencyVerify(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "M_NOT_FOUND") {
		t.Fatalf("expected M_NOT_FOUND body, got %s", rr.Body.String())
	}
}

func TestTransparencyProofReturns404WhenMerkleDisabled(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/transparency/proof?index=0", nil)
	rr := httptest.NewRecorder()

	s.handleTransparencyProof(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "M_NOT_FOUND") {
		t.Fatalf("expected M_NOT_FOUND body, got %s", rr.Body.String())
	}
}

func TestTransparencyVerifyRejectsOversizedLimit(t *testing.T) {
	s := &Server{transparency: &keys.TransparencyLog{}}
	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/transparency/verify?limit=10001", nil)
	rr := httptest.NewRecorder()

	s.handleTransparencyVerify(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "M_INVALID_PARAM") {
		t.Fatalf("expected M_INVALID_PARAM body, got %s", rr.Body.String())
	}
}

func TestTransparencyLogRejectsInvalidServerParameter(t *testing.T) {
	s := &Server{transparency: &keys.TransparencyLog{}}
	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/transparency/log?server=../etc/passwd", nil)
	rr := httptest.NewRecorder()

	s.handleTransparencyLog(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "M_INVALID_PARAM") {
		t.Fatalf("expected M_INVALID_PARAM body, got %s", rr.Body.String())
	}
}

func TestTrustPolicyCheckRejectsInvalidServerParameter(t *testing.T) {
	s := &Server{
		trustPolicy: keys.NewTrustPolicy(keys.TrustPolicyConfig{Enabled: true}),
	}
	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/policy/check?server=../etc/passwd", nil)
	rr := httptest.NewRecorder()

	s.handleTrustPolicyCheck(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "M_INVALID_PARAM") {
		t.Fatalf("expected M_INVALID_PARAM body, got %s", rr.Body.String())
	}
}
