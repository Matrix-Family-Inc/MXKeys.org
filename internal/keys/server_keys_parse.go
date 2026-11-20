/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 20 Nov 2026 22:39:55 UTC
 * Status: Created
 */

package keys

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"mxkeys/internal/zero/signedjson"
)

// parseServerKeys decodes one /_matrix/key/v2/server object. Every field
// is read from the decoded object by its exact name, so the struct view
// and the bytes that signatures cover cannot disagree. The returned
// response carries a copy of raw in Raw.
func parseServerKeys(raw []byte) (*ServerKeysResponse, map[string]interface{}, error) {
	obj, err := signedjson.Decode(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	resp := &ServerKeysResponse{
		VerifyKeys:    make(map[string]VerifyKeyResponse),
		OldVerifyKeys: make(map[string]OldKeyResponse),
		Signatures:    make(map[string]map[string]string),
		Raw:           append([]byte(nil), raw...),
	}

	name, ok := obj["server_name"].(string)
	if !ok || name == "" {
		return nil, nil, fmt.Errorf("%w: server_name must be a non-empty string", ErrInvalidResponse)
	}
	resp.ServerName = name

	if resp.ValidUntilTS, err = integerField(obj["valid_until_ts"]); err != nil {
		return nil, nil, fmt.Errorf("%w: valid_until_ts: %v", ErrInvalidResponse, err)
	}

	verifyKeys, ok := obj["verify_keys"].(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("%w: verify_keys must be an object", ErrInvalidResponse)
	}
	for keyID, value := range verifyKeys {
		entry, ok := value.(map[string]interface{})
		key, keyOK := entry["key"].(string)
		if !ok || !keyOK {
			return nil, nil, fmt.Errorf("%w: verify_keys[%s].key must be a string", ErrInvalidResponse, keyID)
		}
		resp.VerifyKeys[keyID] = VerifyKeyResponse{Key: key}
	}

	if value, present := obj["old_verify_keys"]; present {
		oldKeys, ok := value.(map[string]interface{})
		if !ok {
			return nil, nil, fmt.Errorf("%w: old_verify_keys must be an object", ErrInvalidResponse)
		}
		for keyID, value := range oldKeys {
			entry, ok := value.(map[string]interface{})
			key, keyOK := entry["key"].(string)
			if !ok || !keyOK {
				return nil, nil, fmt.Errorf("%w: old_verify_keys[%s].key must be a string", ErrInvalidResponse, keyID)
			}
			expired, err := integerField(entry["expired_ts"])
			if err != nil {
				return nil, nil, fmt.Errorf("%w: old_verify_keys[%s].expired_ts: %v", ErrInvalidResponse, keyID, err)
			}
			resp.OldVerifyKeys[keyID] = OldKeyResponse{Key: key, ExpiredTS: expired}
		}
	}

	if value, present := obj["signatures"]; present {
		signers, ok := value.(map[string]interface{})
		if !ok {
			return nil, nil, fmt.Errorf("%w: signatures must be an object", ErrInvalidResponse)
		}
		for signer, value := range signers {
			block, ok := value.(map[string]interface{})
			if !ok {
				return nil, nil, fmt.Errorf("%w: signatures[%s] must be an object", ErrInvalidResponse, signer)
			}
			sigs := make(map[string]string, len(block))
			for keyID, value := range block {
				sig, ok := value.(string)
				if !ok {
					return nil, nil, fmt.Errorf("%w: signatures[%s][%s] must be a string", ErrInvalidResponse, signer, keyID)
				}
				sigs[keyID] = sig
			}
			resp.Signatures[signer] = sigs
		}
	}

	return resp, obj, nil
}

// checkSelfSignature requires the document to be current and signed by its
// own server with one of the keys in verify_keys. A key that appears only
// in old_verify_keys has expired and cannot vouch for a current document.
func (f *Fetcher) checkSelfSignature(resp *ServerKeysResponse, obj map[string]interface{}) error {
	if len(resp.VerifyKeys) == 0 {
		return fmt.Errorf("verify_keys is empty")
	}
	if resp.ValidUntilTS <= time.Now().UnixMilli() {
		return fmt.Errorf("valid_until_ts is in the past")
	}
	serverSigs := resp.Signatures[resp.ServerName]
	if len(serverSigs) == 0 {
		return fmt.Errorf("no self-signature found for %s", resp.ServerName)
	}
	if len(serverSigs) > f.maxSignatures {
		return fmt.Errorf("too many signatures for %s: %d > %d", resp.ServerName, len(serverSigs), f.maxSignatures)
	}

	keyIDs := make([]string, 0, len(serverSigs))
	for keyID := range serverSigs {
		keyIDs = append(keyIDs, keyID)
	}
	sort.Strings(keyIDs)
	for _, keyID := range keyIDs {
		verifyKey, ok := resp.VerifyKeys[keyID]
		if !ok {
			continue
		}
		pub, err := base64.RawStdEncoding.DecodeString(verifyKey.Key)
		if err != nil || len(pub) != ed25519.PublicKeySize {
			continue
		}
		if signedjson.VerifyObject(obj, resp.ServerName, keyID, ed25519.PublicKey(pub)) == nil {
			return nil
		}
	}
	return fmt.Errorf("no valid self-signature for %s", resp.ServerName)
}

// verifySelfSignedServerKeys parses raw and checks its self-signature.
func (f *Fetcher) verifySelfSignedServerKeys(raw []byte) (*ServerKeysResponse, error) {
	resp, obj, err := parseServerKeys(raw)
	if err != nil {
		return nil, err
	}
	if err := f.checkSelfSignature(resp, obj); err != nil {
		return nil, err
	}
	return resp, nil
}

func integerField(value interface{}) (int64, error) {
	number, ok := value.(json.Number)
	if !ok {
		return 0, fmt.Errorf("must be an integer")
	}
	n, err := number.Int64()
	if err != nil {
		return 0, fmt.Errorf("must be an integer")
	}
	return n, nil
}
