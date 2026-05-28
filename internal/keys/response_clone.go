/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 08 2026 UTC
 * Status: Created
 */

package keys

func cloneServerKeysResponse(src *ServerKeysResponse) *ServerKeysResponse {
	if src == nil {
		return nil
	}

	dst := &ServerKeysResponse{
		ServerName:    src.ServerName,
		ValidUntilTS:  src.ValidUntilTS,
		VerifyKeys:    make(map[string]VerifyKeyResponse, len(src.VerifyKeys)),
		OldVerifyKeys: make(map[string]OldKeyResponse, len(src.OldVerifyKeys)),
	}

	for keyID, key := range src.VerifyKeys {
		dst.VerifyKeys[keyID] = key
	}
	for keyID, oldKey := range src.OldVerifyKeys {
		dst.OldVerifyKeys[keyID] = oldKey
	}

	if len(src.Signatures) > 0 {
		dst.Signatures = make(map[string]map[string]string, len(src.Signatures))
		for signer, signerSigs := range src.Signatures {
			copied := make(map[string]string, len(signerSigs))
			for keyID, sig := range signerSigs {
				copied[keyID] = sig
			}
			dst.Signatures[signer] = copied
		}
	}

	if len(src.Raw) > 0 {
		dst.Raw = append([]byte(nil), src.Raw...)
	}

	return dst
}
