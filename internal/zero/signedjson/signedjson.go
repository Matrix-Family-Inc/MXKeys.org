/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix.Family Inc. (https://matrix.family)
 * Owner: Matrix.Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 20 Nov 2026 22:39:55 UTC
 * Status: Created
 */

// Package signedjson implements the Matrix "Signing JSON" and "Checking
// for a signature" algorithms (Matrix specification, Appendices). A
// signature covers the canonical JSON of the object with the top-level
// "signatures" and "unsigned" members removed, so signatures added by
// other parties never change what an earlier signer signed.
package signedjson

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"mxkeys/internal/zero/canonical"
)

var (
	// ErrNotObject is returned when the input is not a single JSON object.
	ErrNotObject = errors.New("signedjson: input must be a single JSON object")
	// ErrMissingSignature is returned when no signature exists for the
	// requested signer and key ID.
	ErrMissingSignature = errors.New("signedjson: signature not present")
	// ErrBadSignature is returned when a signature is malformed or does
	// not verify.
	ErrBadSignature = errors.New("signedjson: signature invalid")
)

// Decode parses raw into a generic object. Numbers are kept as
// json.Number so integers round-trip exactly through canonical JSON.
func Decode(raw []byte) (map[string]interface{}, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var obj map[string]interface{}
	if err := dec.Decode(&obj); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotObject, err)
	}
	if obj == nil {
		return nil, ErrNotObject
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("%w: trailing data", ErrNotObject)
	}
	return obj, nil
}

// SigningBytes returns the canonical JSON that a signature over obj covers.
func SigningBytes(obj map[string]interface{}) ([]byte, error) {
	signable := make(map[string]interface{}, len(obj))
	for k, v := range obj {
		if k == "signatures" || k == "unsigned" {
			continue
		}
		signable[k] = v
	}
	return canonical.Marshal(signable)
}

// Sign adds a signature by signer/keyID to raw and returns the canonical
// JSON of the signed object. Existing signatures are kept; a previous
// signature by the same signer and key ID is replaced.
func Sign(raw []byte, signer, keyID string, priv ed25519.PrivateKey) ([]byte, error) {
	if signer == "" || keyID == "" {
		return nil, errors.New("signedjson: signer and key ID are required")
	}
	if len(priv) != ed25519.PrivateKeySize {
		return nil, errors.New("signedjson: invalid ed25519 private key")
	}
	obj, err := Decode(raw)
	if err != nil {
		return nil, err
	}
	msg, err := SigningBytes(obj)
	if err != nil {
		return nil, err
	}
	sig := base64.RawStdEncoding.EncodeToString(ed25519.Sign(priv, msg))

	signatures, err := signatureMap(obj)
	if err != nil {
		return nil, err
	}
	block, _ := signatures[signer].(map[string]interface{})
	if block == nil {
		block = make(map[string]interface{}, 1)
	}
	block[keyID] = sig
	signatures[signer] = block
	obj["signatures"] = signatures

	return canonical.Marshal(obj)
}

// Verify checks the signature by signer/keyID on raw against pub.
func Verify(raw []byte, signer, keyID string, pub ed25519.PublicKey) error {
	obj, err := Decode(raw)
	if err != nil {
		return err
	}
	return VerifyObject(obj, signer, keyID, pub)
}

// VerifyObject is Verify for an already decoded object.
func VerifyObject(obj map[string]interface{}, signer, keyID string, pub ed25519.PublicKey) error {
	if len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("%w: public key length %d", ErrBadSignature, len(pub))
	}
	signatures, err := signatureMap(obj)
	if err != nil {
		return err
	}
	block, _ := signatures[signer].(map[string]interface{})
	encoded, ok := block[keyID].(string)
	if !ok {
		return fmt.Errorf("%w: %s %s", ErrMissingSignature, signer, keyID)
	}
	sig, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("%w: malformed signature for %s %s", ErrBadSignature, signer, keyID)
	}
	msg, err := SigningBytes(obj)
	if err != nil {
		return err
	}
	if !ed25519.Verify(pub, msg, sig) {
		return fmt.Errorf("%w: %s %s", ErrBadSignature, signer, keyID)
	}
	return nil
}

// signatureMap returns the "signatures" member, creating an empty one when
// absent. A present member of the wrong type is an error.
func signatureMap(obj map[string]interface{}) (map[string]interface{}, error) {
	raw, present := obj["signatures"]
	if !present {
		return make(map[string]interface{}), nil
	}
	signatures, ok := raw.(map[string]interface{})
	if !ok {
		return nil, errors.New("signedjson: signatures must be an object")
	}
	return signatures, nil
}
