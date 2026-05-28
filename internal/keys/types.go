/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Tue Apr 21 2026 UTC
 * Status: Updated
 */

package keys

import "encoding/json"

// ServerKeysResponse response from /_matrix/key/v2/server.
//
// Raw holds the canonical JSON bytes delivered by origin (or already
// carrying this notary's perspective signature) when known. When Raw
// is non-empty, MarshalJSON returns those bytes verbatim so every
// field delivered by origin (including presence/absence of
// `old_verify_keys` and any future schema extension) is preserved
// byte-for-byte. Origin signatures computed over
// canonical(payload - signatures) stay verifiable end-to-end.
type ServerKeysResponse struct {
	ServerName    string                       `json:"server_name"`
	VerifyKeys    map[string]VerifyKeyResponse `json:"verify_keys"`
	OldVerifyKeys map[string]OldKeyResponse    `json:"old_verify_keys,omitempty"`
	ValidUntilTS  int64                        `json:"valid_until_ts"`
	Signatures    map[string]map[string]string `json:"signatures,omitempty"`

	Raw []byte `json:"-"`
}

// serverKeysResponseAlias mirrors ServerKeysResponse for struct-based
// marshaling (bypassing the custom MarshalJSON) when Raw is unset.
type serverKeysResponseAlias struct {
	ServerName    string                       `json:"server_name"`
	VerifyKeys    map[string]VerifyKeyResponse `json:"verify_keys"`
	OldVerifyKeys map[string]OldKeyResponse    `json:"old_verify_keys,omitempty"`
	ValidUntilTS  int64                        `json:"valid_until_ts"`
	Signatures    map[string]map[string]string `json:"signatures,omitempty"`
}

// MarshalJSON returns Raw bytes verbatim when populated; otherwise
// falls back to struct-based marshaling. This is the single guarantee
// that lets the raw-preserving notary pipeline deliver origin payload
// byte-for-byte.
func (r ServerKeysResponse) MarshalJSON() ([]byte, error) {
	if len(r.Raw) > 0 {
		out := make([]byte, len(r.Raw))
		copy(out, r.Raw)
		return out, nil
	}
	return json.Marshal(serverKeysResponseAlias{
		ServerName:    r.ServerName,
		VerifyKeys:    r.VerifyKeys,
		OldVerifyKeys: r.OldVerifyKeys,
		ValidUntilTS:  r.ValidUntilTS,
		Signatures:    r.Signatures,
	})
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
