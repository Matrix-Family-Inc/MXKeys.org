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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteMatrixError_ConsistentShapeAndStatus(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		errCode    string
		message    string
	}{
		{name: "bad json", statusCode: http.StatusBadRequest, errCode: "M_BAD_JSON", message: "invalid json"},
		{name: "invalid param", statusCode: http.StatusBadRequest, errCode: "M_INVALID_PARAM", message: "invalid param"},
		{name: "not found", statusCode: http.StatusNotFound, errCode: "M_NOT_FOUND", message: "not found"},
		{name: "too large", statusCode: http.StatusRequestEntityTooLarge, errCode: "M_TOO_LARGE", message: "payload too large"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeMatrixError(rec, tc.statusCode, tc.errCode, tc.message)

			if rec.Code != tc.statusCode {
				t.Fatalf("expected status %d, got %d", tc.statusCode, rec.Code)
			}

			var payload map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("expected valid JSON payload, got error: %v", err)
			}

			if payload["errcode"] != tc.errCode {
				t.Fatalf("expected errcode %q, got %q", tc.errCode, payload["errcode"])
			}
			if payload["error"] != tc.message {
				t.Fatalf("expected error message %q, got %q", tc.message, payload["error"])
			}
		})
	}
}
