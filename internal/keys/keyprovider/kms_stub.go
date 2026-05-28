/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

package keyprovider

import (
	"context"
	"crypto/ed25519"
	"errors"
)

// KMSStub is a placeholder provider that documents the future external-KMS
// contract. It intentionally does not dial any network; attempting to use it
// returns ErrNotImplemented. Production builds that integrate a real KMS
// replace this file with a concrete implementation.
type KMSStub struct {
	endpoint string
	keyID    string
}

func newKMSStub(endpoint, keyID string) (*KMSStub, error) {
	if endpoint == "" {
		return nil, errors.New("keyprovider kms: endpoint is required")
	}
	return &KMSStub{endpoint: endpoint, keyID: keyID}, nil
}

// Kind returns KindKMS.
func (k *KMSStub) Kind() Kind { return KindKMS }

// LoadOrGenerate returns ErrNotImplemented.
func (k *KMSStub) LoadOrGenerate(_ context.Context) (ed25519.PrivateKey, string, error) {
	return nil, "", ErrNotImplemented
}

// PublicKey panics: the stub has no material.
func (k *KMSStub) PublicKey() ed25519.PublicKey {
	panic("keyprovider kms: stub has no key material")
}

// Sign returns ErrNotImplemented.
func (k *KMSStub) Sign(_ context.Context, _ []byte) ([]byte, error) {
	return nil, ErrNotImplemented
}
