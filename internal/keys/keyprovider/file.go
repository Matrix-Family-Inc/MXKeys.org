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
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const keyFileName = "mxkeys_ed25519.key"

// FileProvider stores the signing key as raw bytes at 0600 under a directory
// locked down to 0700. This matches the pre-keyprovider behavior for operators
// migrating from older builds without reconfiguration.
type FileProvider struct {
	storagePath string

	mu      sync.RWMutex
	loaded  bool
	private ed25519.PrivateKey
}

func newFileProvider(storagePath string) (*FileProvider, error) {
	if storagePath == "" {
		return nil, errors.New("keyprovider: file storage path is required")
	}
	return &FileProvider{storagePath: storagePath}, nil
}

// Kind returns KindFile.
func (f *FileProvider) Kind() Kind { return KindFile }

// LoadOrGenerate loads the existing key from disk or generates a new one and
// persists it. File/directory permissions are enforced on every call to
// recover from operators accidentally loosening them out-of-band.
func (f *FileProvider) LoadOrGenerate(ctx context.Context) (ed25519.PrivateKey, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := os.MkdirAll(f.storagePath, 0700); err != nil {
		return nil, "", fmt.Errorf("keyprovider file: mkdir: %w", err)
	}
	if err := os.Chmod(f.storagePath, 0700); err != nil {
		return nil, "", fmt.Errorf("keyprovider file: chmod dir: %w", err)
	}

	keyPath := filepath.Join(f.storagePath, keyFileName)

	if data, err := os.ReadFile(keyPath); err == nil {
		priv, perr := parsePrivateKey(data)
		if perr != nil {
			return nil, "", fmt.Errorf("keyprovider file: parse: %w", perr)
		}
		if err := os.Chmod(keyPath, 0600); err != nil {
			return nil, "", fmt.Errorf("keyprovider file: chmod key: %w", err)
		}
		f.private = priv
		f.loaded = true
		return priv, KeyID, nil
	} else if !os.IsNotExist(err) {
		return nil, "", fmt.Errorf("keyprovider file: read: %w", err)
	}

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, "", fmt.Errorf("keyprovider file: generate: %w", err)
	}
	if err := os.WriteFile(keyPath, priv, 0600); err != nil {
		return nil, "", fmt.Errorf("keyprovider file: write: %w", err)
	}
	if err := os.Chmod(keyPath, 0600); err != nil {
		return nil, "", fmt.Errorf("keyprovider file: chmod key: %w", err)
	}
	f.private = priv
	f.loaded = true
	return priv, KeyID, nil
}

// PublicKey returns the public component. Panics if LoadOrGenerate has not
// yet succeeded: that indicates a wiring bug in the caller.
func (f *FileProvider) PublicKey() ed25519.PublicKey {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if !f.loaded {
		panic("keyprovider file: PublicKey called before LoadOrGenerate")
	}
	return f.private.Public().(ed25519.PublicKey)
}

// Sign delegates to ed25519.Sign on the in-memory key.
func (f *FileProvider) Sign(_ context.Context, data []byte) ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if !f.loaded {
		return nil, errors.New("keyprovider file: not loaded")
	}
	return ed25519.Sign(f.private, data), nil
}

// parsePrivateKey accepts either a full 64-byte ed25519 private key or a
// 32-byte seed, matching the legacy on-disk format.
func parsePrivateKey(data []byte) (ed25519.PrivateKey, error) {
	switch len(data) {
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(data), nil
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(data), nil
	default:
		return nil, fmt.Errorf("invalid ed25519 key length %d (expected %d or %d)",
			len(data), ed25519.PrivateKeySize, ed25519.SeedSize)
	}
}
