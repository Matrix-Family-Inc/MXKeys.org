/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

// Package keyprovider abstracts notary signing key management.
//
// The default FileProvider keeps backward compatibility with the historical
// file-based key on disk (0600, user-only). EnvProvider reads a raw or seed
// key from an environment variable for ephemeral deployments (e.g. CI,
// Kubernetes secrets mounted via env). A KMSProvider stub documents the
// interface for future external-KMS integration without implementing network
// calls in this package.
package keyprovider

import (
	"context"
	"crypto/ed25519"
	"errors"
)

// Kind identifies a provider implementation chosen via configuration.
type Kind string

const (
	KindFile Kind = "file"
	KindEnv  Kind = "env"
	KindKMS  Kind = "kms"
)

// KeyID is the notary key identifier embedded in signatures.
const KeyID = "ed25519:mxkeys"

// ErrNotImplemented is returned by providers that are placeholders for future
// integrations (e.g. external KMS) when the build did not wire the backend.
var ErrNotImplemented = errors.New("keyprovider: not implemented")

// Provider owns the lifecycle of the ed25519 signing key.
// Implementations must be safe for concurrent use after LoadOrGenerate returns.
type Provider interface {
	// LoadOrGenerate loads an existing signing key or generates a new one,
	// depending on the implementation. Returns the private key and its key ID.
	LoadOrGenerate(ctx context.Context) (ed25519.PrivateKey, string, error)

	// PublicKey returns the public component. Must be called after
	// LoadOrGenerate.
	PublicKey() ed25519.PublicKey

	// Sign signs data with the loaded private key. For KMS-backed providers
	// this may be a remote call; the context controls the operation deadline.
	Sign(ctx context.Context, data []byte) ([]byte, error)

	// Kind returns the provider kind for logging and diagnostics.
	Kind() Kind
}

// Config describes how to construct a Provider.
type Config struct {
	// Kind selects the backend.
	Kind Kind

	// File-specific.
	StoragePath string

	// Env-specific.
	EnvVar string

	// KMS-specific (stub fields, reserved for future expansion).
	KMSEndpoint string
	KMSKeyID    string
}

// New returns a Provider for the given configuration. The default (empty Kind)
// is KindFile to preserve backward compatibility.
func New(cfg Config) (Provider, error) {
	switch cfg.Kind {
	case "", KindFile:
		return newFileProvider(cfg.StoragePath)
	case KindEnv:
		return newEnvProvider(cfg.EnvVar)
	case KindKMS:
		return newKMSStub(cfg.KMSEndpoint, cfg.KMSKeyID)
	default:
		return nil, errors.New("keyprovider: unknown kind " + string(cfg.Kind))
	}
}
