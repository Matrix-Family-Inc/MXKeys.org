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
	"crypto/ed25519"

	"mxkeys/internal/zero/signedjson"
)

// AttachNotarySignature adds this notary's perspective signature to one
// verified /_matrix/key/v2/server document and returns its canonical JSON.
// The signature follows the Matrix signing algorithm: it covers the
// document without "signatures" and "unsigned", so the origin signature and
// any other notary signature already present stay valid.
func AttachNotarySignature(raw []byte, notaryName, notaryKeyID string, notaryPriv ed25519.PrivateKey) ([]byte, error) {
	return signedjson.Sign(raw, notaryName, notaryKeyID, notaryPriv)
}
