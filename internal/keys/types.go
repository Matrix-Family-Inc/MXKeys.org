/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Tue Jan 27 2026 UTC
 * Status: Created
 */

package keys

// ServerKeysResponse response from /_matrix/key/v2/server
type ServerKeysResponse struct {
	ServerName    string                       `json:"server_name"`
	VerifyKeys    map[string]VerifyKeyResponse `json:"verify_keys"`
	OldVerifyKeys map[string]OldKeyResponse    `json:"old_verify_keys,omitempty"`
	ValidUntilTS  int64                        `json:"valid_until_ts"`
	Signatures    map[string]map[string]string `json:"signatures,omitempty"`
}

// VerifyKeyResponse verify key in response
type VerifyKeyResponse struct {
	Key string `json:"key"`
}

// OldKeyResponse old key in response
type OldKeyResponse struct {
	Key       string `json:"key"`
	ExpiredTS int64  `json:"expired_ts"`
}

// KeyQueryRequest request body for /_matrix/key/v2/query
type KeyQueryRequest struct {
	ServerKeys map[string]map[string]KeyCriteria `json:"server_keys"`
}

// KeyCriteria criteria for key query
type KeyCriteria struct {
	MinimumValidUntilTS int64 `json:"minimum_valid_until_ts,omitempty"`
}

// KeyQueryResponse response for /_matrix/key/v2/query
type KeyQueryResponse struct {
	ServerKeys []ServerKeysResponse   `json:"server_keys"`
	Failures   map[string]interface{} `json:"failures,omitempty"`
}
