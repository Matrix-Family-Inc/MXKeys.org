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
	"crypto/ed25519"
	"encoding/base64"
	"testing"
	"time"
)

func TestAddNotarySignatureIsStableAcrossResign(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate notary key: %v", err)
	}

	n := &Notary{
		serverName:    "mxkeys.example.org",
		serverKeyID:   "ed25519:mxkeys",
		serverKeyPair: priv,
	}

	resp := &ServerKeysResponse{
		ServerName:   "origin.example.org",
		ValidUntilTS: time.Now().Add(time.Hour).UnixMilli(),
		VerifyKeys: map[string]VerifyKeyResponse{
			"ed25519:origin": {Key: base64.RawStdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize))},
		},
		OldVerifyKeys: map[string]OldKeyResponse{},
		Signatures: map[string]map[string]string{
			"origin.example.org": {"ed25519:origin": "origin-signature"},
		},
	}

	if err := n.addNotarySignature(resp); err != nil {
		t.Fatalf("addNotarySignature() first call failed: %v", err)
	}
	sigFirst := resp.Signatures[n.serverName][n.serverKeyID]
	if sigFirst == "" {
		t.Fatal("expected notary signature after first signing")
	}

	if err := n.addNotarySignature(resp); err != nil {
		t.Fatalf("addNotarySignature() second call failed: %v", err)
	}
	sigSecond := resp.Signatures[n.serverName][n.serverKeyID]
	if sigSecond == "" {
		t.Fatal("expected notary signature after second signing")
	}

	if sigFirst != sigSecond {
		t.Fatalf("notary signature changed between re-signs")
	}
}

func TestCloneServerKeysResponseDeepCopy(t *testing.T) {
	src := &ServerKeysResponse{
		ServerName:   "origin.example.org",
		ValidUntilTS: time.Now().Add(time.Hour).UnixMilli(),
		VerifyKeys: map[string]VerifyKeyResponse{
			"ed25519:origin": {Key: "origin-key"},
		},
		OldVerifyKeys: map[string]OldKeyResponse{
			"ed25519:old": {Key: "old-key", ExpiredTS: 123},
		},
		Signatures: map[string]map[string]string{
			"origin.example.org": {"ed25519:origin": "sig"},
		},
	}

	clone := cloneServerKeysResponse(src)
	clone.VerifyKeys["ed25519:origin"] = VerifyKeyResponse{Key: "modified"}
	clone.OldVerifyKeys["ed25519:old"] = OldKeyResponse{Key: "modified-old", ExpiredTS: 456}
	clone.Signatures["origin.example.org"]["ed25519:origin"] = "modified-sig"

	if src.VerifyKeys["ed25519:origin"].Key != "origin-key" {
		t.Fatalf("source verify_keys was mutated")
	}
	if src.OldVerifyKeys["ed25519:old"].Key != "old-key" {
		t.Fatalf("source old_verify_keys was mutated")
	}
	if src.Signatures["origin.example.org"]["ed25519:origin"] != "sig" {
		t.Fatalf("source signatures were mutated")
	}
}
