/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 08 2026 UTC
 * Status: Created
 */

package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"mxkeys/internal/keys"
)

// liveBaseURL returns the base URL for live federation tests.
// Tests skip when MXKEYS_LIVE_TEST != "1" or MXKEYS_LIVE_BASE_URL is not set.
// No hardcoded fallback: live tests must target an explicitly configured deployment.
func liveBaseURL(t *testing.T) string {
	t.Helper()
	if os.Getenv("MXKEYS_LIVE_TEST") != "1" {
		t.Skip("set MXKEYS_LIVE_TEST=1 to run live federation checks")
	}
	baseURL := os.Getenv("MXKEYS_LIVE_BASE_URL")
	if baseURL == "" {
		t.Skip("set MXKEYS_LIVE_BASE_URL to run live federation checks")
	}
	return baseURL
}

func TestLiveFederationQueryStrictness(t *testing.T) {
	baseURL := liveBaseURL(t)

	client := &http.Client{Timeout: 15 * time.Second}

	reqBody := `{"server_keys":{"s-a.mxtest.tech":{},"s-b.mxtest.tech":{}}}`
	req, err := http.NewRequest(http.MethodPost, baseURL+"/_matrix/key/v2/query", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("live query failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("live query returned %d: %s", resp.StatusCode, string(body))
	}

	var queryResp struct {
		ServerKeys []map[string]interface{} `json:"server_keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
		t.Fatalf("failed to decode live response: %v", err)
	}
	if len(queryResp.ServerKeys) < 1 {
		t.Fatalf("expected at least 1 server_keys entry, got %d", len(queryResp.ServerKeys))
	}

	trailingReqBody := `{"server_keys":{"s-a.mxtest.tech":{}}}{"x":1}`
	trailingReq, err := http.NewRequest(http.MethodPost, baseURL+"/_matrix/key/v2/query", strings.NewReader(trailingReqBody))
	if err != nil {
		t.Fatalf("failed to build trailing request: %v", err)
	}
	trailingReq.Header.Set("Content-Type", "application/json")

	trailingResp, err := client.Do(trailingReq)
	if err != nil {
		t.Fatalf("live trailing query failed: %v", err)
	}
	defer trailingResp.Body.Close()
	if trailingResp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(trailingResp.Body)
		t.Fatalf("expected non-200 for trailing JSON, got 200: %s", string(body))
	}
}

func TestLiveQueryCompatibility(t *testing.T) {
	baseURL := liveBaseURL(t)

	client := &http.Client{Timeout: 15 * time.Second}
	reqBody := `{"server_keys":{"s-a.mxtest.tech":{},"s-b.mxtest.tech":{}}}`
	req, err := http.NewRequest(http.MethodPost, baseURL+"/_matrix/key/v2/query", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("live query failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("live query returned %d: %s", resp.StatusCode, string(body))
	}

	var queryResp struct {
		ServerKeys []struct {
			ServerName string                            `json:"server_name"`
			VerifyKeys map[string]keys.VerifyKeyResponse `json:"verify_keys"`
			Signatures map[string]map[string]string      `json:"signatures"`
		} `json:"server_keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
		t.Fatalf("failed to decode live response: %v", err)
	}
	if len(queryResp.ServerKeys) < 1 {
		t.Fatalf("expected at least 1 server_keys entry, got %d", len(queryResp.ServerKeys))
	}
	for _, entry := range queryResp.ServerKeys {
		if entry.ServerName == "" {
			t.Fatal("server_name must not be empty")
		}
		if len(entry.VerifyKeys) == 0 {
			t.Fatalf("verify_keys must not be empty for %s", entry.ServerName)
		}
		if len(entry.Signatures) == 0 {
			t.Fatalf("signatures must not be empty for %s", entry.ServerName)
		}
	}
}

func TestLiveNotaryFailurePath(t *testing.T) {
	baseURL := liveBaseURL(t)

	client := &http.Client{Timeout: 20 * time.Second}
	reqBody := `{"server_keys":{"s-a.mxtest.tech":{},"no-such-server.invalid":{}}}`
	req, err := http.NewRequest(http.MethodPost, baseURL+"/_matrix/key/v2/query", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("live query failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("live query returned %d: %s", resp.StatusCode, string(body))
	}

	var queryResp struct {
		ServerKeys []struct {
			ServerName string `json:"server_name"`
		} `json:"server_keys"`
		Failures map[string]interface{} `json:"failures"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
		t.Fatalf("failed to decode live response: %v", err)
	}
	if len(queryResp.ServerKeys) == 0 {
		t.Fatal("expected at least one successful server_keys entry")
	}
	if _, ok := queryResp.Failures["no-such-server.invalid"]; !ok {
		t.Fatalf("expected failure entry for no-such-server.invalid, got: %#v", queryResp.Failures)
	}
}
