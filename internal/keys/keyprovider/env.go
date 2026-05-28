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
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"sync"
)

// EnvProvider loads a notary signing key from a base64-encoded environment
// variable. Intended for short-lived/ephemeral deployments that inject secrets
// via orchestrator (Kubernetes, systemd) rather than files on disk.
//
// The envelope is base64 of either a 32-byte seed or a 64-byte full ed25519
// key. RawStdEncoding is preferred; StdEncoding is accepted as fallback.
type EnvProvider struct {
	envVar string

	mu      sync.RWMutex
	loaded  bool
	private ed25519.PrivateKey
}

func newEnvProvider(envVar string) (*EnvProvider, error) {
	if envVar == "" {
		return nil, errors.New("keyprovider: env variable name is required")
	}
	return &EnvProvider{envVar: envVar}, nil
}

// Kind returns KindEnv.
func (e *EnvProvider) Kind() Kind { return KindEnv }

// LoadOrGenerate reads the configured environment variable and decodes the
// embedded key. Does not generate a new key: the env backend assumes the
// operator is responsible for key provisioning outside the process.
func (e *EnvProvider) LoadOrGenerate(ctx context.Context) (ed25519.PrivateKey, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	raw := os.Getenv(e.envVar)
	if raw == "" {
		return nil, "", fmt.Errorf("keyprovider env: %s is not set", e.envVar)
	}

	data, err := decodeEnvKey(raw)
	if err != nil {
		return nil, "", fmt.Errorf("keyprovider env: decode %s: %w", e.envVar, err)
	}

	priv, err := parsePrivateKey(data)
	if err != nil {
		return nil, "", fmt.Errorf("keyprovider env: parse %s: %w", e.envVar, err)
	}

	e.private = priv
	e.loaded = true
	return priv, KeyID, nil
}

// PublicKey returns the public component.
func (e *EnvProvider) PublicKey() ed25519.PublicKey {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if !e.loaded {
		panic("keyprovider env: PublicKey called before LoadOrGenerate")
	}
	return e.private.Public().(ed25519.PublicKey)
}

// Sign delegates to ed25519.Sign.
func (e *EnvProvider) Sign(_ context.Context, data []byte) ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if !e.loaded {
		return nil, errors.New("keyprovider env: not loaded")
	}
	return ed25519.Sign(e.private, data), nil
}

// decodeEnvKey accepts base64 in either RawStdEncoding (preferred) or
// StdEncoding form.
func decodeEnvKey(raw string) ([]byte, error) {
	if b, err := base64.RawStdEncoding.DecodeString(raw); err == nil {
		return b, nil
	}
	return base64.StdEncoding.DecodeString(raw)
}
