/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
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

const (
	keyFileName    = "mxkeys_ed25519.key"
	keyFileEncName = "mxkeys_ed25519.key.enc"
)

// FileProvider stores the signing key under a 0700 directory.
//
// Operation modes, selected by the Passphrase field at construction
// time:
//
//   - Passphrase == nil: plaintext ed25519 key at <storage>/mxkeys_ed25519.key,
//     mode 0600. Backwards-compatible with pre-encryption installs.
//
//   - Passphrase != nil: encrypted envelope at <storage>/mxkeys_ed25519.key.enc,
//     mode 0600. Format is documented in file_crypto.go
//     (MXKENC01 / AES-256-GCM / PBKDF2-HMAC-SHA256). A legacy
//     plaintext file, if present, is transparently re-encrypted on
//     first load and the plaintext file is securely replaced.
//
// File and directory permissions are re-enforced on every load so
// operator mistakes ("chmod 0644 keys/ to debug") are self-healing.
type FileProvider struct {
	storagePath string
	passphrase  []byte // nil = plaintext mode

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

// newFileProviderWithPassphrase is used by the Config.Kind == KindFile
// path when encryption is enabled. An empty passphrase is rejected so
// operators cannot accidentally end up in a "claims encrypted but
// isn't" state.
func newFileProviderWithPassphrase(storagePath string, passphrase []byte) (*FileProvider, error) {
	if storagePath == "" {
		return nil, errors.New("keyprovider: file storage path is required")
	}
	if len(passphrase) == 0 {
		return nil, errors.New("keyprovider: empty encryption passphrase")
	}
	return &FileProvider{storagePath: storagePath, passphrase: passphrase}, nil
}

// Kind returns KindFile.
func (f *FileProvider) Kind() Kind { return KindFile }

// LoadOrGenerate loads the existing key from disk or generates a new
// one. When running in encrypted mode, a legacy plaintext file is
// upgraded in-place.
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

	plainPath := filepath.Join(f.storagePath, keyFileName)
	encPath := filepath.Join(f.storagePath, keyFileEncName)

	// Preferred path (encrypted mode): try the .enc file first. If only
	// a legacy plaintext file exists, read, re-encrypt, and atomically
	// replace. In plaintext mode, .enc is ignored (refusing to read
	// encrypted material without a passphrase is safer than guessing).
	if len(f.passphrase) > 0 {
		if priv, err := f.loadEncrypted(encPath); err == nil {
			f.private = priv
			f.loaded = true
			return priv, KeyID, nil
		} else if !os.IsNotExist(err) {
			return nil, "", err
		}
		// Legacy upgrade path.
		if data, err := os.ReadFile(plainPath); err == nil {
			priv, perr := parsePrivateKey(data)
			if perr != nil {
				return nil, "", fmt.Errorf("keyprovider file: legacy parse: %w", perr)
			}
			if werr := f.writeEncrypted(encPath, priv); werr != nil {
				return nil, "", werr
			}
			// Best-effort scrub of the plaintext file; its sensitive
			// contents now live encrypted at rest.
			_ = os.Remove(plainPath)
			f.private = priv
			f.loaded = true
			return priv, KeyID, nil
		} else if !os.IsNotExist(err) {
			return nil, "", fmt.Errorf("keyprovider file: read legacy: %w", err)
		}
		// No existing key: generate one and encrypt it.
		_, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, "", fmt.Errorf("keyprovider file: generate: %w", err)
		}
		if err := f.writeEncrypted(encPath, priv); err != nil {
			return nil, "", err
		}
		f.private = priv
		f.loaded = true
		return priv, KeyID, nil
	}

	// Plaintext mode: legacy path. If an encrypted file exists on disk
	// we refuse to silently overwrite or ignore it; that almost always
	// means the operator forgot the passphrase and would otherwise
	// regenerate (=rotate) their server identity by mistake.
	if _, err := os.Stat(encPath); err == nil {
		return nil, "", errors.New(
			"keyprovider file: encrypted key file exists but no passphrase configured; " +
				"set keys.encryption.passphrase_env or remove the file to regenerate")
	}

	if data, err := os.ReadFile(plainPath); err == nil {
		priv, perr := parsePrivateKey(data)
		if perr != nil {
			return nil, "", fmt.Errorf("keyprovider file: parse: %w", perr)
		}
		if err := os.Chmod(plainPath, 0600); err != nil {
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
	if err := writeAtomic(plainPath, priv, 0600); err != nil {
		return nil, "", fmt.Errorf("keyprovider file: write: %w", err)
	}
	f.private = priv
	f.loaded = true
	return priv, KeyID, nil
}

// PublicKey returns the public component. Panics if LoadOrGenerate has
// not yet succeeded: that indicates a wiring bug in the caller.
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

// loadEncrypted reads and decrypts an MXKENC01 envelope.
func (f *FileProvider) loadEncrypted(path string) (ed25519.PrivateKey, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	plain, err := decryptKey(blob, f.passphrase)
	if err != nil {
		if errors.Is(err, ErrBadMagic) {
			return nil, fmt.Errorf("keyprovider file: %s lacks MXKENC01 magic (corrupted or plaintext-under-wrong-name)", path)
		}
		return nil, fmt.Errorf("keyprovider file: decrypt: %w", err)
	}
	priv, perr := parsePrivateKey(plain)
	if perr != nil {
		return nil, fmt.Errorf("keyprovider file: post-decrypt parse: %w", perr)
	}
	if err := os.Chmod(path, 0600); err != nil {
		return nil, fmt.Errorf("keyprovider file: chmod enc: %w", err)
	}
	return priv, nil
}

// writeEncrypted seals the seed bytes and writes the envelope atomically.
func (f *FileProvider) writeEncrypted(path string, priv ed25519.PrivateKey) error {
	// Store only the 32-byte seed when we can (it reconstructs the full
	// 64-byte private key deterministically); this keeps encrypted file
	// size minimal.
	payload := []byte(priv)
	if len(priv) == ed25519.PrivateKeySize {
		payload = priv.Seed()
	}
	blob, err := encryptKey(payload, f.passphrase)
	if err != nil {
		return fmt.Errorf("keyprovider file: encrypt: %w", err)
	}
	if err := writeAtomic(path, blob, 0600); err != nil {
		return fmt.Errorf("keyprovider file: write enc: %w", err)
	}
	return nil
}

// writeAtomic performs a "write to tmp then rename" sequence so readers
// never see a half-written file.
func writeAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return err
	}
	return nil
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
