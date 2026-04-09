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
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"mxkeys/internal/zero/canonical"
	"mxkeys/internal/zero/log"
)

// verifyNotarySignature verifies the notary's signature if we have a pinned key
func (f *Fetcher) verifyNotarySignature(notary string, resp *ServerKeysResponse) error {
	trusted, hasPinned := f.trustedNotaries[notary]
	if !hasPinned {
		// No pinned key, trust based on TLS
		return nil
	}

	// Check if notary signed this response
	notarySigs, ok := resp.Signatures[notary]
	if !ok {
		return fmt.Errorf("notary %s did not sign the response", notary)
	}

	sig, ok := notarySigs[trusted.KeyID]
	if !ok {
		return fmt.Errorf("notary %s did not sign with pinned key %s", notary, trusted.KeyID)
	}

	// Decode signature
	sigBytes, err := base64.RawStdEncoding.DecodeString(sig)
	if err != nil {
		return fmt.Errorf("failed to decode notary signature: %w", err)
	}

	if len(sigBytes) != ed25519.SignatureSize {
		return fmt.Errorf("invalid notary signature length: %d", len(sigBytes))
	}

	if len(trusted.PublicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid pinned public key length: %d", len(trusted.PublicKey))
	}

	toVerify := map[string]interface{}{
		"server_name":     resp.ServerName,
		"valid_until_ts":  resp.ValidUntilTS,
		"verify_keys":     resp.VerifyKeys,
		"old_verify_keys": resp.OldVerifyKeys,
	}

	// The notary signs the object that includes all existing signatures
	// except its own just-added signature.
	if resp.Signatures != nil {
		signatures := make(map[string]map[string]string, len(resp.Signatures))
		for signer, signerSigs := range resp.Signatures {
			if signer == notary {
				continue
			}
			copied := make(map[string]string, len(signerSigs))
			for keyID, value := range signerSigs {
				copied[keyID] = value
			}
			signatures[signer] = copied
		}
		if len(signatures) > 0 {
			toVerify["signatures"] = signatures
		}
	}

	canonicalBytes, err := canonical.Marshal(toVerify)
	if err != nil {
		return fmt.Errorf("failed to canonicalize notary payload: %w", err)
	}

	if !ed25519.Verify(ed25519.PublicKey(trusted.PublicKey), canonicalBytes, sigBytes) {
		return ErrNotaryKeyMismatch
	}

	log.Debug("Notary signature verified",
		"notary", notary,
		"key_id", trusted.KeyID,
	)

	return nil
}

// verifySelfSignature verifies that server signed its own keys
// using Matrix canonical JSON for signature verification.
func (f *Fetcher) verifySelfSignature(resp *ServerKeysResponse, rawJSON []byte) error {
	// Verify required fields
	if resp.ServerName == "" {
		return fmt.Errorf("server_name is empty")
	}
	if len(resp.VerifyKeys) == 0 {
		return fmt.Errorf("verify_keys is empty")
	}
	if resp.ValidUntilTS <= time.Now().UnixMilli() {
		return fmt.Errorf("valid_until_ts is in the past")
	}
	if resp.Signatures == nil {
		return fmt.Errorf("no signatures in response")
	}

	serverSigs, ok := resp.Signatures[resp.ServerName]
	if !ok {
		return fmt.Errorf("no self-signature found for %s", resp.ServerName)
	}
	if len(serverSigs) > f.maxSignatures {
		return fmt.Errorf("too many signatures for %s: %d > %d", resp.ServerName, len(serverSigs), f.maxSignatures)
	}

	// Remove signatures and unsigned for canonical JSON
	var parsed map[string]interface{}
	dec := json.NewDecoder(bytes.NewReader(rawJSON))
	dec.UseNumber()
	if err := dec.Decode(&parsed); err != nil {
		return fmt.Errorf("failed to parse JSON for verification: %w", err)
	}
	delete(parsed, "signatures")
	delete(parsed, "unsigned")

	// Re-encode without signatures
	stripped, err := json.Marshal(parsed)
	if err != nil {
		return fmt.Errorf("failed to re-encode JSON: %w", err)
	}

	// Convert to Matrix canonical JSON (sorted keys, compact)
	canonicalBytes, err := canonical.JSON(stripped)
	if err != nil {
		return fmt.Errorf("failed to canonicalize JSON: %w", err)
	}

	// Verify at least one signature matches
	for keyID, sigBase64 := range serverSigs {
		verifyKey, ok := resp.VerifyKeys[keyID]
		if !ok {
			// Check old_verify_keys
			if oldKey, ok := resp.OldVerifyKeys[keyID]; ok {
				pubKeyBytes, err := base64.RawStdEncoding.DecodeString(oldKey.Key)
				if err != nil {
					continue
				}
				// Verify ed25519 public key length (32 bytes)
				if len(pubKeyBytes) != ed25519.PublicKeySize {
					continue
				}
				sig, err := base64.RawStdEncoding.DecodeString(sigBase64)
				if err != nil {
					continue
				}
				// Verify ed25519 signature length (64 bytes)
				if len(sig) != ed25519.SignatureSize {
					continue
				}
				if ed25519.Verify(ed25519.PublicKey(pubKeyBytes), canonicalBytes, sig) {
					return nil
				}
			}
			continue
		}

		pubKeyBytes, err := base64.RawStdEncoding.DecodeString(verifyKey.Key)
		if err != nil {
			continue
		}

		// Verify ed25519 public key length (32 bytes)
		if len(pubKeyBytes) != ed25519.PublicKeySize {
			continue
		}

		sig, err := base64.RawStdEncoding.DecodeString(sigBase64)
		if err != nil {
			continue
		}

		// Verify ed25519 signature length (64 bytes)
		if len(sig) != ed25519.SignatureSize {
			continue
		}

		if ed25519.Verify(ed25519.PublicKey(pubKeyBytes), canonicalBytes, sig) {
			return nil // Valid signature found
		}
	}

	return fmt.Errorf("no valid self-signature for %s", resp.ServerName)
}
