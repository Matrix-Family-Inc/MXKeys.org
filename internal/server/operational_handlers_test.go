/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Mon 22 Jun 2026 21:12:00 UTC
 * Status: Created
 */

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDisabledOperationalHandlersReturnDisabledJSON(t *testing.T) {
	s := newHelperServer()
	tests := []struct {
		name    string
		path    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{"analytics summary", "/_mxkeys/analytics/summary", s.handleAnalyticsSummary},
		{"analytics servers", "/_mxkeys/analytics/servers?limit=bad", s.handleAnalyticsServers},
		{"analytics anomalies", "/_mxkeys/analytics/anomalies", s.handleAnalyticsAnomalies},
		{"analytics rotators", "/_mxkeys/analytics/rotators?limit=9999", s.handleAnalyticsTopRotators},
		{"cluster status", "/_mxkeys/cluster/status", s.handleClusterStatus},
		{"cluster nodes", "/_mxkeys/cluster/nodes", s.handleClusterNodes},
		{"trust policy status", "/_mxkeys/policy/status", s.handleTrustPolicyStatus},
		{"trust policy check", "/_mxkeys/policy/check?server=matrix.org", s.handleTrustPolicyCheck},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			tt.handler(rr, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rr.Code)
			}
			var payload map[string]bool
			if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			if payload["enabled"] {
				t.Fatalf("expected enabled=false payload, got %s", rr.Body.String())
			}
		})
	}
}

func TestNormalizeRouteCases(t *testing.T) {
	tests := map[string]string{
		"/_matrix/key/v2/query":          "/_matrix/key/v2/query",
		"/_matrix/key/v2/server/a":       "/_matrix/key/v2/server",
		"/_matrix/federation/v1/version": "/_matrix/federation/v1/version",
		"/_mxkeys/health":                "/_mxkeys/health",
		"/_mxkeys/live":                  "/_mxkeys/live",
		"/_mxkeys/ready":                 "/_mxkeys/ready",
		"/_mxkeys/metrics":               "/_mxkeys/metrics",
		"/unknown":                       "/other",
	}
	for in, want := range tests {
		if got := NormalizeRoute(in); got != want {
			t.Fatalf("NormalizeRoute(%q) = %q, want %q", in, got, want)
		}
	}
}
