/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Wed Apr 08 2026 UTC
 * Status: Created
 */

package keys

import (
	"errors"
	"testing"
)

func TestSanitizeQueryFailure(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantErrCode string
		wantMessage string
	}{
		{
			name:        "resolve",
			err:         NewResolveError("example.org", errors.New("dial tcp 10.0.0.1: no route to host")),
			wantErrCode: "M_NOT_FOUND",
			wantMessage: "Unable to resolve remote server",
		},
		{
			name:        "fetch",
			err:         NewFetchError("example.org", errors.New("tls handshake timeout")),
			wantErrCode: "M_UNKNOWN",
			wantMessage: "Unable to fetch remote server keys",
		},
		{
			name:        "signature",
			err:         NewSignatureError("example.org", errors.New("wrong key")),
			wantErrCode: "M_INVALID_PARAM",
			wantMessage: "Remote server keys failed verification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeQueryFailure(tt.err)
			if got["errcode"] != tt.wantErrCode {
				t.Fatalf("errcode = %v, want %s", got["errcode"], tt.wantErrCode)
			}
			if got["error"] != tt.wantMessage {
				t.Fatalf("error = %v, want %s", got["error"], tt.wantMessage)
			}
		})
	}
}
