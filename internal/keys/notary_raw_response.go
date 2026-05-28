/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Tue Apr 21 2026 UTC
 * Status: Created
 */

// This file owns the raw-preserving notary reply pipeline. It wraps
// an origin-delivered /_matrix/key/v2/server payload in canonical
// JSON and appends this notary's perspective signature without
// reshaping any other field of the origin payload. The origin
// self-signature remains verifiable end-to-end because every byte of
// canonical(payload - signatures) stays identical to what origin
// canonicalized when it signed.

package keys

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"mxkeys/internal/zero/canonical"
)

// errRawDecode marks any failure parsing origin JSON prior to
// perspective-signing. Callers must fall back to the struct-based
// path when they see this error.
var errRawDecode = errors.New("notary raw: decode origin payload")

// AttachNotarySignature takes origin-delivered JSON bytes (one
// /_matrix/key/v2/server object), canonicalizes them, attaches this
// notary's perspective signature under signatures[notaryName][keyID]
// (without overwriting any other signer), and returns canonical JSON
// bytes of the resulting object.
//
// The payload that this notary signs matches the historical
// addNotarySignature contract: canonical over
// {server_name, valid_until_ts, verify_keys, old_verify_keys
// (when present in origin payload), signatures (non-self signers
// only)}. Origin signatures from `signatures` are preserved byte-
// for-byte in the returned bytes.
func AttachNotarySignature(
	rawOrigin []byte,
	notaryName, notaryKeyID string,
	notaryPriv ed25519.PrivateKey,
) ([]byte, error) {
	if len(rawOrigin) == 0 {
		return nil, fmt.Errorf("%w: empty input", errRawDecode)
	}
	if notaryName == "" || notaryKeyID == "" {
		return nil, fmt.Errorf("notary raw: notary name and key id are required")
	}
	if len(notaryPriv) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("notary raw: invalid notary private key length")
	}

	obj, err := decodeOriginObject(rawOrigin)
	if err != nil {
		return nil, err
	}

	existingSigs := extractSignatures(obj)

	signable := buildSignable(obj, existingSigs, notaryName)
	signBytes, err := canonical.Marshal(signable)
	if err != nil {
		return nil, fmt.Errorf("notary raw: canonicalize signable: %w", err)
	}
	sigB64 := base64.RawStdEncoding.EncodeToString(ed25519.Sign(notaryPriv, signBytes))

	finalSigs := mergeNotarySig(existingSigs, notaryName, notaryKeyID, sigB64)
	obj["signatures"] = finalSigs

	out, err := canonical.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("notary raw: canonicalize final: %w", err)
	}
	return out, nil
}

// decodeOriginObject parses raw into a canonical-friendly generic
// map, preserving integer precision via json.Number so that values
// like valid_until_ts round-trip exactly.
func decodeOriginObject(raw []byte) (map[string]interface{}, error) {
	var obj map[string]interface{}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&obj); err != nil {
		return nil, fmt.Errorf("%w: %v", errRawDecode, err)
	}
	if obj == nil {
		return nil, fmt.Errorf("%w: top-level must be an object", errRawDecode)
	}
	return obj, nil
}

// extractSignatures returns the `signatures` sub-object as a
// nested map keyed by signer. Missing or malformed returns an empty
// map. The returned map is safe to mutate; values are never shared
// with the input.
func extractSignatures(obj map[string]interface{}) map[string]map[string]string {
	raw, ok := obj["signatures"].(map[string]interface{})
	if !ok {
		return map[string]map[string]string{}
	}
	out := make(map[string]map[string]string, len(raw))
	for signer, v := range raw {
		inner, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		block := make(map[string]string, len(inner))
		for keyID, val := range inner {
			if s, ok := val.(string); ok {
				block[keyID] = s
			}
		}
		out[signer] = block
	}
	return out
}

// buildSignable creates the payload that this notary signs over.
// It includes every origin field except `signatures` and `unsigned`,
// and re-attaches non-self signers' signatures (matching the
// existing perspective contract used by our fetcher verifier).
func buildSignable(
	obj map[string]interface{},
	existing map[string]map[string]string,
	notaryName string,
) map[string]interface{} {
	signable := make(map[string]interface{}, len(obj))
	for k, v := range obj {
		if k == "signatures" || k == "unsigned" {
			continue
		}
		signable[k] = v
	}
	others := make(map[string]interface{}, len(existing))
	for signer, block := range existing {
		if signer == notaryName {
			continue
		}
		copied := make(map[string]interface{}, len(block))
		for keyID, val := range block {
			copied[keyID] = val
		}
		others[signer] = copied
	}
	if len(others) > 0 {
		signable["signatures"] = others
	}
	return signable
}

// mergeNotarySig returns a signatures sub-object including every
// existing signer plus this notary's newly produced signature under
// notaryKeyID. Existing signers other than this notary are preserved
// verbatim; another key-id under the same notary is replaced.
func mergeNotarySig(
	existing map[string]map[string]string,
	notaryName, notaryKeyID, sigB64 string,
) map[string]interface{} {
	out := make(map[string]interface{}, len(existing)+1)
	for signer, block := range existing {
		if signer == notaryName {
			continue
		}
		inner := make(map[string]interface{}, len(block))
		for keyID, val := range block {
			inner[keyID] = val
		}
		out[signer] = inner
	}
	notaryBlock := make(map[string]interface{})
	if existingSelf, ok := existing[notaryName]; ok {
		for keyID, val := range existingSelf {
			notaryBlock[keyID] = val
		}
	}
	notaryBlock[notaryKeyID] = sigB64
	out[notaryName] = notaryBlock
	return out
}
