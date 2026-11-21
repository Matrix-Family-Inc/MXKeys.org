/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix.Family Inc. (https://matrix.family)
 * Owner: Matrix.Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Mon 22 Jun 2026 00:50:40 UTC
 * Status: Updated
 */

package keys

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
	"time"
)

func TestCountersignIsStableAcrossResign(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate notary key: %v", err)
	}
	n := &Notary{serverName: "mxkeys.example.org", serverKeyID: "ed25519:mxkeys", serverKeyPair: priv}

	raw, _ := createSignedKeysResponse(t, "origin.example.org")
	origin, _, err := parseServerKeys(raw)
	if err != nil {
		t.Fatal(err)
	}

	first, err := n.countersign(origin)
	if err != nil {
		t.Fatalf("first countersign: %v", err)
	}
	second, err := n.countersign(first)
	if err != nil {
		t.Fatalf("second countersign: %v", err)
	}
	if string(first.Raw) != string(second.Raw) {
		t.Fatalf("countersigning twice must not change the document")
	}
	if first.Signatures[n.serverName][n.serverKeyID] == "" {
		t.Fatal("countersigned struct must expose the notary signature")
	}
}

func TestCountersignRequiresDocumentBytes(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	n := &Notary{serverName: "mxkeys.example.org", serverKeyID: "ed25519:mxkeys", serverKeyPair: priv}
	legacy := &ServerKeysResponse{
		ServerName:   "origin.example.org",
		ValidUntilTS: time.Now().Add(time.Hour).UnixMilli(),
		VerifyKeys:   map[string]VerifyKeyResponse{"ed25519:origin": {Key: base64.RawStdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize))}},
	}
	if _, err := n.countersign(legacy); err == nil {
		t.Fatal("a document without its delivered bytes must not be countersigned")
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
