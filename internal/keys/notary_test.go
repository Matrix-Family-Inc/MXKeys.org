/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 20 Nov 2026 22:39:55 UTC
 * Status: Updated
 */

package keys

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"testing"
	"time"
)

func testNotary(t *testing.T) (*Notary, ed25519.PublicKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return &Notary{
		serverName:    "notary.example",
		serverKeyID:   "ed25519:n",
		serverKeyPair: priv,
		validityHours: 24,
		cacheTTLHours: 1,
		cache:         make(map[string]*cachedResponse),
	}, pub
}

// encodeLikeHandler encodes v the way server.writeJSON does.
func encodeLikeHandler(t *testing.T, v interface{}) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		t.Fatal(err)
	}
	return bytes.TrimSpace(buf.Bytes())
}

func TestGetOwnKeysIsSelfSignedPerSpec(t *testing.T) {
	n, pub := testNotary(t)
	own, err := n.GetOwnKeys()
	if err != nil {
		t.Fatal(err)
	}
	served := encodeLikeHandler(t, own)

	if !verifyOriginSignature(t, served, "notary.example", "ed25519:n", pub) {
		t.Fatalf("own key document must verify with the Matrix algorithm: %s", served)
	}
	if !bytes.Contains(served, []byte(`"old_verify_keys":{}`)) {
		t.Fatalf("served document must carry the old_verify_keys it was signed with: %s", served)
	}
}

func TestQueryKeysServesSpecVerifiableSignatures(t *testing.T) {
	n, notaryPub := testNotary(t)
	originPub, originPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	raw := buildSignedOriginPayload(t, "origin.example", "ed25519:o", originPriv, map[string]interface{}{
		"old_verify_keys": map[string]interface{}{},
		"valid_until_ts":  time.Now().Add(time.Hour).UnixMilli(),
	})
	origin, _, err := parseServerKeys(raw)
	if err != nil {
		t.Fatal(err)
	}
	n.cache["origin.example"] = &cachedResponse{response: origin, validUntil: time.Now().Add(time.Hour)}

	resp := n.QueryKeys(context.Background(), &KeyQueryRequest{
		ServerKeys: map[string]map[string]KeyCriteria{"origin.example": {}},
	})
	if len(resp.ServerKeys) != 1 {
		t.Fatalf("expected one key document, got %d (failures: %v)", len(resp.ServerKeys), resp.Failures)
	}

	var wire struct {
		ServerKeys []json.RawMessage `json:"server_keys"`
	}
	if err := json.Unmarshal(encodeLikeHandler(t, resp), &wire); err != nil {
		t.Fatal(err)
	}
	doc := wire.ServerKeys[0]
	if !verifyOriginSignature(t, doc, "origin.example", "ed25519:o", originPub) {
		t.Fatalf("origin signature must survive the notary reply: %s", doc)
	}
	if !verifyOriginSignature(t, doc, "notary.example", "ed25519:n", notaryPub) {
		t.Fatalf("notary signature must verify with the Matrix algorithm: %s", doc)
	}
}

func TestQueryKeysNeverServesDocumentsWithoutBytes(t *testing.T) {
	n, _ := testNotary(t)
	n.cache["legacy.example"] = &cachedResponse{
		response: &ServerKeysResponse{
			ServerName:   "legacy.example",
			ValidUntilTS: time.Now().Add(time.Hour).UnixMilli(),
			VerifyKeys:   map[string]VerifyKeyResponse{"ed25519:a": {Key: "AAAA"}},
		},
		validUntil: time.Now().Add(time.Hour),
	}

	resp := n.QueryKeys(context.Background(), &KeyQueryRequest{
		ServerKeys: map[string]map[string]KeyCriteria{"legacy.example": {}},
	})
	if len(resp.ServerKeys) != 0 {
		t.Fatalf("a re-marshalled document must not be served")
	}
	if _, ok := resp.Failures["legacy.example"]; !ok {
		t.Fatalf("expected a failure entry, got %v", resp.Failures)
	}
}
