/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Tue Apr 21 2026 UTC
 * Status: Created
 */

package keys

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"testing"

	"mxkeys/internal/zero/canonical"
)

// buildSignedOriginPayload mints a canonical /_matrix/key/v2/server
// payload signed by origin with the provided verify key. It returns
// the full canonical JSON (signatures included) plus the canonical
// signable bytes so tests can assert byte-exact preservation.
func buildSignedOriginPayload(
	t *testing.T,
	serverName, keyID string,
	origin ed25519.PrivateKey,
	fields map[string]interface{},
) []byte {
	t.Helper()
	signable := map[string]interface{}{
		"server_name":    serverName,
		"valid_until_ts": int64(2_000_000_000_000),
		"verify_keys": map[string]interface{}{
			keyID: map[string]interface{}{
				"key": base64.RawStdEncoding.EncodeToString(origin.Public().(ed25519.PublicKey)),
			},
		},
	}
	for k, v := range fields {
		signable[k] = v
	}
	signBytes, err := canonical.Marshal(signable)
	if err != nil {
		t.Fatalf("canonicalize signable: %v", err)
	}
	sig := ed25519.Sign(origin, signBytes)
	delivered := make(map[string]interface{}, len(signable)+1)
	for k, v := range signable {
		delivered[k] = v
	}
	delivered["signatures"] = map[string]interface{}{
		serverName: map[string]interface{}{
			keyID: base64.RawStdEncoding.EncodeToString(sig),
		},
	}
	raw, err := canonical.Marshal(delivered)
	if err != nil {
		t.Fatalf("canonicalize delivered: %v", err)
	}
	return raw
}

// verifyOriginSignature mirrors the stripped-canonical verification
// that every Matrix client performs on notary responses: strip
// signatures/unsigned, canonicalize, verify origin signature.
func verifyOriginSignature(
	t *testing.T,
	raw []byte,
	serverName, keyID string,
	pub ed25519.PublicKey,
) bool {
	t.Helper()
	var obj map[string]interface{}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&obj); err != nil {
		t.Fatalf("parse delivered: %v", err)
	}
	sigs, _ := obj["signatures"].(map[string]interface{})
	block, _ := sigs[serverName].(map[string]interface{})
	sigB64, _ := block[keyID].(string)
	sig, err := base64.RawStdEncoding.DecodeString(sigB64)
	if err != nil {
		t.Fatalf("decode origin sig: %v", err)
	}
	delete(obj, "signatures")
	delete(obj, "unsigned")
	signable, err := canonical.Marshal(obj)
	if err != nil {
		t.Fatalf("canonicalize stripped: %v", err)
	}
	return ed25519.Verify(pub, signable, sig)
}

func TestAttachNotarySignaturePreservesOriginWhenOldKeysOmitted(t *testing.T) {
	originPub, originPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	notaryPub, notaryPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	raw := buildSignedOriginPayload(t, "origin.example", "ed25519:o", originPriv, nil)

	out, err := AttachNotarySignature(raw, "notary.example", "ed25519:n", notaryPriv)
	if err != nil {
		t.Fatalf("AttachNotarySignature: %v", err)
	}
	if !verifyOriginSignature(t, out, "origin.example", "ed25519:o", originPub) {
		t.Fatalf("origin self-signature must verify against raw-preserving output")
	}
	_ = notaryPub
}

func TestAttachNotarySignaturePreservesOriginWhenOldKeysEmpty(t *testing.T) {
	originPub, originPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	_, notaryPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	raw := buildSignedOriginPayload(t, "origin.example", "ed25519:o", originPriv, map[string]interface{}{
		"old_verify_keys": map[string]interface{}{},
	})

	out, err := AttachNotarySignature(raw, "notary.example", "ed25519:n", notaryPriv)
	if err != nil {
		t.Fatalf("AttachNotarySignature: %v", err)
	}
	if !verifyOriginSignature(t, out, "origin.example", "ed25519:o", originPub) {
		t.Fatalf("origin self-signature must verify when old_verify_keys is empty")
	}

	// Confirm the empty old_verify_keys{} survived canonical
	// round-trip and was not dropped to satisfy an omitempty rule.
	if !bytes.Contains(out, []byte(`"old_verify_keys":{}`)) {
		t.Fatalf("delivered bytes must keep old_verify_keys:{} intact: %s", out)
	}
}

func TestAttachNotarySignaturePreservesOriginWhenOldKeysPopulated(t *testing.T) {
	originPub, originPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	_, notaryPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	oldPriv := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x07}, 32))
	oldPubB64 := base64.RawStdEncoding.EncodeToString(oldPriv.Public().(ed25519.PublicKey))
	raw := buildSignedOriginPayload(t, "origin.example", "ed25519:o", originPriv, map[string]interface{}{
		"old_verify_keys": map[string]interface{}{
			"ed25519:retired": map[string]interface{}{
				"key":        oldPubB64,
				"expired_ts": int64(1_500_000_000_000),
			},
		},
	})

	out, err := AttachNotarySignature(raw, "notary.example", "ed25519:n", notaryPriv)
	if err != nil {
		t.Fatalf("AttachNotarySignature: %v", err)
	}
	if !verifyOriginSignature(t, out, "origin.example", "ed25519:o", originPub) {
		t.Fatalf("origin self-signature must verify when old_verify_keys is populated")
	}
}

func TestAttachNotarySignatureAddsVerifiableNotarySig(t *testing.T) {
	_, originPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	notaryPub, notaryPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	raw := buildSignedOriginPayload(t, "origin.example", "ed25519:o", originPriv, nil)

	out, err := AttachNotarySignature(raw, "notary.example", "ed25519:n", notaryPriv)
	if err != nil {
		t.Fatalf("AttachNotarySignature: %v", err)
	}

	var obj map[string]interface{}
	dec := json.NewDecoder(bytes.NewReader(out))
	dec.UseNumber()
	if err := dec.Decode(&obj); err != nil {
		t.Fatalf("parse out: %v", err)
	}
	sigs, _ := obj["signatures"].(map[string]interface{})
	notaryBlock, _ := sigs["notary.example"].(map[string]interface{})
	sigB64, _ := notaryBlock["ed25519:n"].(string)
	sig, err := base64.RawStdEncoding.DecodeString(sigB64)
	if err != nil {
		t.Fatalf("decode notary sig: %v", err)
	}

	// Rebuild the signable payload exactly as AttachNotarySignature
	// would: strip signatures/unsigned, then re-attach non-self
	// signers.
	existing := extractSignatures(obj)
	signable := buildSignable(obj, existing, "notary.example")
	signBytes, err := canonical.Marshal(signable)
	if err != nil {
		t.Fatalf("canonicalize signable for verify: %v", err)
	}
	if !ed25519.Verify(notaryPub, signBytes, sig) {
		t.Fatalf("notary perspective signature must verify against canonical(payload + non-self signers)")
	}
}

func TestAttachNotarySignatureRejectsEmptyAndBadInputs(t *testing.T) {
	_, notaryPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := AttachNotarySignature(nil, "n", "k", notaryPriv); err == nil {
		t.Fatal("empty input must error")
	}
	if _, err := AttachNotarySignature([]byte(`"not-an-object"`), "n", "k", notaryPriv); err == nil {
		t.Fatal("non-object input must error")
	}
	if _, err := AttachNotarySignature([]byte(`{`), "n", "k", notaryPriv); err == nil {
		t.Fatal("malformed JSON must error")
	}
	if _, err := AttachNotarySignature([]byte(`{}`), "", "k", notaryPriv); err == nil {
		t.Fatal("missing notary name must error")
	}
	if _, err := AttachNotarySignature([]byte(`{}`), "n", "k", nil); err == nil {
		t.Fatal("missing notary private key must error")
	}
}
