/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix.Family Inc. (https://matrix.family)
 * Owner: Matrix.Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 20 Nov 2026 22:39:55 UTC
 * Status: Updated
 */

package keys

import (
	"crypto/ed25519"
	"errors"
	"fmt"

	"mxkeys/internal/zero/log"
	"mxkeys/internal/zero/signedjson"
)

// verifyNotarySignature checks the pinned key of notary on one key document
// that notary returned. Without a pinned key the document rests on the TLS
// connection to the notary and on the origin self-signature, which the
// caller verifies separately.
func (f *Fetcher) verifyNotarySignature(notary string, raw []byte) error {
	trusted, hasPinned := f.trustedNotaries[notary]
	if !hasPinned {
		return nil
	}
	if len(trusted.PublicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid pinned public key length: %d", len(trusted.PublicKey))
	}

	err := signedjson.Verify(raw, notary, trusted.KeyID, ed25519.PublicKey(trusted.PublicKey))
	switch {
	case err == nil:
		log.Debug("Notary signature verified", "notary", notary, "key_id", trusted.KeyID)
		return nil
	case errors.Is(err, signedjson.ErrMissingSignature):
		return fmt.Errorf("notary %s did not sign with pinned key %s", notary, trusted.KeyID)
	default:
		return fmt.Errorf("%w: %v", ErrNotaryKeyMismatch, err)
	}
}
