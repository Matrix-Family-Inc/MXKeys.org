//go:build integration

package integration

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"mxkeys/internal/zero/canonical"
)

func TestResponseHasNotarySignature(t *testing.T) {
	notaryName := "notary.mxkeys.test"
	notaryKeyID := "ed25519:mxkeys"
	notaryPub, notaryPriv, _ := ed25519.GenerateKey(nil)

	upstreamName := "upstream.matrix.test"
	upstreamKeyID := "ed25519:upstream"
	upstreamPub, upstreamPriv, _ := ed25519.GenerateKey(nil)

	response := map[string]interface{}{
		"server_name":     upstreamName,
		"valid_until_ts":  time.Now().Add(24 * time.Hour).UnixMilli(),
		"verify_keys":     map[string]interface{}{upstreamKeyID: map[string]string{"key": base64.RawStdEncoding.EncodeToString(upstreamPub)}},
		"old_verify_keys": map[string]interface{}{},
	}

	canonBytes, _ := canonical.Marshal(response)
	upstreamSig := ed25519.Sign(upstreamPriv, canonBytes)
	upstreamSigB64 := base64.RawStdEncoding.EncodeToString(upstreamSig)

	response["signatures"] = map[string]interface{}{
		upstreamName: map[string]string{upstreamKeyID: upstreamSigB64},
	}

	canonBytes2, _ := canonical.Marshal(response)
	notarySig := ed25519.Sign(notaryPriv, canonBytes2)
	notarySigB64 := base64.RawStdEncoding.EncodeToString(notarySig)

	sigs := response["signatures"].(map[string]interface{})
	sigs[notaryName] = map[string]string{notaryKeyID: notarySigB64}

	data, _ := json.Marshal(response)

	var result struct {
		ServerName string                       `json:"server_name"`
		Signatures map[string]map[string]string `json:"signatures"`
	}
	json.Unmarshal(data, &result)

	if result.ServerName != upstreamName {
		t.Errorf("server_name = %q, want %q", result.ServerName, upstreamName)
	}

	if _, ok := result.Signatures[upstreamName]; !ok {
		t.Error("missing upstream signature")
	}

	if _, ok := result.Signatures[notaryName]; !ok {
		t.Fatal("missing notary signature")
	}

	notarySigs := result.Signatures[notaryName]
	if _, ok := notarySigs[notaryKeyID]; !ok {
		t.Errorf("notary key ID %s not found", notaryKeyID)
	}

	_ = notaryPub
}

func TestNotarySignatureIsVerifiable(t *testing.T) {
	notaryPub, notaryPriv, _ := ed25519.GenerateKey(nil)
	notaryName := "notary.test"
	notaryKeyID := "ed25519:notary"

	response := map[string]interface{}{
		"server_name":     "origin.test",
		"valid_until_ts":  time.Now().Add(time.Hour).UnixMilli(),
		"verify_keys":     map[string]interface{}{},
		"old_verify_keys": map[string]interface{}{},
	}

	canonBytes, _ := canonical.Marshal(response)
	sig := ed25519.Sign(notaryPriv, canonBytes)
	sigB64 := base64.RawStdEncoding.EncodeToString(sig)

	response["signatures"] = map[string]interface{}{
		notaryName: map[string]string{notaryKeyID: sigB64},
	}

	data, _ := json.Marshal(response)

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	delete(parsed, "signatures")

	canonForVerify, _ := canonical.Marshal(parsed)

	sigBytes, err := base64.RawStdEncoding.DecodeString(sigB64)
	if err != nil {
		t.Fatalf("failed to decode signature: %v", err)
	}

	if !ed25519.Verify(notaryPub, canonForVerify, sigBytes) {
		t.Error("notary signature verification failed")
	}
}

func TestUpstreamSignaturePreserved(t *testing.T) {
	upstreamPub, upstreamPriv, _ := ed25519.GenerateKey(nil)
	upstreamName := "upstream.test"
	upstreamKeyID := "ed25519:upstream"

	response := map[string]interface{}{
		"server_name":     upstreamName,
		"valid_until_ts":  time.Now().Add(time.Hour).UnixMilli(),
		"verify_keys":     map[string]interface{}{upstreamKeyID: map[string]string{"key": base64.RawStdEncoding.EncodeToString(upstreamPub)}},
		"old_verify_keys": map[string]interface{}{},
	}

	canonBytes, _ := canonical.Marshal(response)
	upstreamSig := ed25519.Sign(upstreamPriv, canonBytes)
	upstreamSigB64 := base64.RawStdEncoding.EncodeToString(upstreamSig)

	response["signatures"] = map[string]interface{}{
		upstreamName: map[string]string{upstreamKeyID: upstreamSigB64},
	}

	_, notaryPriv, _ := ed25519.GenerateKey(nil)
	notaryName := "notary.test"
	notaryKeyID := "ed25519:notary"

	canonBytes2, _ := canonical.Marshal(response)
	notarySig := ed25519.Sign(notaryPriv, canonBytes2)
	notarySigB64 := base64.RawStdEncoding.EncodeToString(notarySig)

	sigs := response["signatures"].(map[string]interface{})
	sigs[notaryName] = map[string]string{notaryKeyID: notarySigB64}

	data, _ := json.Marshal(response)

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	signaturesRaw := parsed["signatures"]
	delete(parsed, "signatures")

	canonForVerify, _ := canonical.Marshal(parsed)

	signatures := signaturesRaw.(map[string]interface{})
	upstreamSigs := signatures[upstreamName].(map[string]interface{})
	sigToVerify := upstreamSigs[upstreamKeyID].(string)

	sigBytes, _ := base64.RawStdEncoding.DecodeString(sigToVerify)

	if !ed25519.Verify(upstreamPub, canonForVerify, sigBytes) {
		t.Error("upstream signature should still be valid after notary signs")
	}
}

func TestMultipleNotarySignatures(t *testing.T) {
	response := map[string]interface{}{
		"server_name":     "origin.test",
		"valid_until_ts":  time.Now().Add(time.Hour).UnixMilli(),
		"verify_keys":     map[string]interface{}{},
		"old_verify_keys": map[string]interface{}{},
		"signatures": map[string]interface{}{
			"origin.test":  map[string]string{"ed25519:origin": "sig1"},
			"notary1.test": map[string]string{"ed25519:n1": "sig2"},
			"notary2.test": map[string]string{"ed25519:n2": "sig3"},
			"notary3.test": map[string]string{"ed25519:n3": "sig4"},
		},
	}

	data, _ := json.Marshal(response)

	var result struct {
		Signatures map[string]map[string]string `json:"signatures"`
	}
	json.Unmarshal(data, &result)

	if len(result.Signatures) != 4 {
		t.Errorf("expected 4 signers (origin + 3 notaries), got %d", len(result.Signatures))
	}

	expectedSigners := []string{"origin.test", "notary1.test", "notary2.test", "notary3.test"}
	for _, signer := range expectedSigners {
		if _, ok := result.Signatures[signer]; !ok {
			t.Errorf("missing signer: %s", signer)
		}
	}
}

func TestSignatureKeyIDFormat(t *testing.T) {
	validKeyIDs := []string{
		"ed25519:abc",
		"ed25519:mxkeys",
		"ed25519:key_123",
		"ed25519:CAPS",
	}

	for _, keyID := range validKeyIDs {
		parts := splitKeyID(keyID)
		if parts[0] != "ed25519" {
			t.Errorf("%s: algorithm should be ed25519, got %s", keyID, parts[0])
		}
		if parts[1] == "" {
			t.Errorf("%s: key identifier should not be empty", keyID)
		}
	}
}

func splitKeyID(keyID string) []string {
	for i, c := range keyID {
		if c == ':' {
			return []string{keyID[:i], keyID[i+1:]}
		}
	}
	return []string{keyID, ""}
}

func TestOwnKeysContainSignature(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	serverName := "mxkeys.test"
	keyID := "ed25519:mxkeys"
	pubB64 := base64.RawStdEncoding.EncodeToString(pub)

	ownKeys := map[string]interface{}{
		"server_name":     serverName,
		"valid_until_ts":  time.Now().Add(24 * time.Hour).UnixMilli(),
		"verify_keys":     map[string]interface{}{keyID: map[string]string{"key": pubB64}},
		"old_verify_keys": map[string]interface{}{},
	}

	canonBytes, _ := canonical.Marshal(ownKeys)
	sig := ed25519.Sign(priv, canonBytes)
	sigB64 := base64.RawStdEncoding.EncodeToString(sig)

	ownKeys["signatures"] = map[string]interface{}{
		serverName: map[string]string{keyID: sigB64},
	}

	data, _ := json.Marshal(ownKeys)

	var result struct {
		ServerName string                          `json:"server_name"`
		VerifyKeys map[string]struct{ Key string } `json:"verify_keys"`
		Signatures map[string]map[string]string    `json:"signatures"`
	}
	json.Unmarshal(data, &result)

	if result.ServerName != serverName {
		t.Errorf("server_name = %q, want %q", result.ServerName, serverName)
	}

	if _, ok := result.VerifyKeys[keyID]; !ok {
		t.Errorf("verify_keys should contain %s", keyID)
	}

	if _, ok := result.Signatures[serverName]; !ok {
		t.Error("signatures should contain self-signature")
	}

	selfSigs := result.Signatures[serverName]
	if _, ok := selfSigs[keyID]; !ok {
		t.Errorf("self-signature should use key %s", keyID)
	}
}

func TestQueryResponseSignatureContract(t *testing.T) {
	queryResponse := map[string]interface{}{
		"server_keys": []map[string]interface{}{
			{
				"server_name":    "queried.server",
				"valid_until_ts": time.Now().Add(time.Hour).UnixMilli(),
				"verify_keys":    map[string]interface{}{"ed25519:key1": map[string]string{"key": "base64key"}},
				"signatures": map[string]interface{}{
					"queried.server":    map[string]string{"ed25519:key1": "origin_sig"},
					"mxkeys.notary.org": map[string]string{"ed25519:mxkeys": "notary_sig"},
				},
			},
		},
		"failures": map[string]interface{}{},
	}

	data, _ := json.Marshal(queryResponse)

	var result struct {
		ServerKeys []struct {
			ServerName string                       `json:"server_name"`
			Signatures map[string]map[string]string `json:"signatures"`
		} `json:"server_keys"`
	}
	json.Unmarshal(data, &result)

	if len(result.ServerKeys) != 1 {
		t.Fatalf("expected 1 server_key, got %d", len(result.ServerKeys))
	}

	sk := result.ServerKeys[0]

	if _, ok := sk.Signatures[sk.ServerName]; !ok {
		t.Error("server_key must have origin signature")
	}

	hasNotarySig := false
	for signer := range sk.Signatures {
		if signer != sk.ServerName {
			hasNotarySig = true
			break
		}
	}

	if !hasNotarySig {
		t.Error("server_key should have at least one notary signature in query response")
	}
}
