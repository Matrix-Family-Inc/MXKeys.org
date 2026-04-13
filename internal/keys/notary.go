/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Tue Jan 27 2026 UTC
 * Status: Created
 */

package keys

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"mxkeys/internal/zero/canonical"
	"mxkeys/internal/zero/log"
)

// Notary is the main key notary service
type Notary struct {
	serverName    string
	serverKeyID   string
	serverKeyPair ed25519.PrivateKey

	storage *Storage
	fetcher *Fetcher

	cache      map[string]*cachedResponse
	cacheMu    sync.RWMutex
	fetchGroup singleflight.Group
	cleanupMu  sync.Mutex
	cleanupWG  sync.WaitGroup

	cleanupCancel context.CancelFunc

	validityHours    int
	cacheTTLHours    int
	trustPolicy      *TrustPolicy
	transparency     *TransparencyLog
	analytics        *Analytics
	keyBroadcastHook func(serverName, keyID, keyData string, validUntilTS int64)

	configMu sync.RWMutex // protects runtime configuration setters
}

type cachedResponse struct {
	response   *ServerKeysResponse
	validUntil time.Time
}

// NewNotary creates new notary service
func NewNotary(
	db *sql.DB,
	serverName string,
	keyStoragePath string,
	validityHours, cacheTTLHours int,
	fallbackServers []string,
	fetchTimeout time.Duration,
	trustedNotaries []TrustedNotaryKey,
	maxSignaturesPerKey int,
) (*Notary, error) {
	storage, err := NewStorage(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	fetcher := NewFetcherWithConfig(FetcherConfig{
		FallbackServers: fallbackServers,
		Timeout:         fetchTimeout,
		TrustedNotaries: trustedNotaries,
		MaxSignatures:   maxSignaturesPerKey,
	})

	n := &Notary{
		serverName:    serverName,
		storage:       storage,
		fetcher:       fetcher,
		cache:         make(map[string]*cachedResponse),
		validityHours: validityHours,
		cacheTTLHours: cacheTTLHours,
	}

	// Load or generate server signing key
	if err := n.initSigningKey(keyStoragePath); err != nil {
		return nil, fmt.Errorf("failed to init signing key: %w", err)
	}

	return n, nil
}

// initSigningKey loads existing key or generates new one
func (n *Notary) initSigningKey(keyStoragePath string) error {
	keyPath := filepath.Join(keyStoragePath, "mxkeys_ed25519.key")

	if err := os.MkdirAll(keyStoragePath, 0700); err != nil {
		return fmt.Errorf("failed to create key storage directory: %w", err)
	}
	if err := os.Chmod(keyStoragePath, 0700); err != nil {
		return fmt.Errorf("failed to enforce key storage directory permissions: %w", err)
	}

	// Try to load existing key
	if data, err := os.ReadFile(keyPath); err == nil {
		switch len(data) {
		case ed25519.PrivateKeySize:
			n.serverKeyPair = ed25519.PrivateKey(data)
		case ed25519.SeedSize:
			n.serverKeyPair = ed25519.NewKeyFromSeed(data)
		default:
			return fmt.Errorf("existing notary signing key has invalid length %d", len(data))
		}
		n.serverKeyID = "ed25519:mxkeys"
		if err := os.Chmod(keyPath, 0600); err != nil {
			return fmt.Errorf("failed to enforce key file permissions: %w", err)
		}
		log.Info("Loaded existing notary signing key", "key_id", n.serverKeyID)
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read existing notary signing key: %w", err)
	}

	// Generate new key
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	n.serverKeyPair = privateKey
	n.serverKeyID = "ed25519:mxkeys"

	// Save key
	if err := os.WriteFile(keyPath, privateKey, 0600); err != nil {
		return fmt.Errorf("failed to save notary signing key: %w", err)
	}
	if err := os.Chmod(keyPath, 0600); err != nil {
		return fmt.Errorf("failed to enforce key file permissions: %w", err)
	}
	log.Info("Generated and saved new notary signing key", "key_id", n.serverKeyID)

	return nil
}

// GetOwnKeys returns this notary's own public keys
func (n *Notary) GetOwnKeys() (*ServerKeysResponse, error) {
	publicKey := n.serverKeyPair.Public().(ed25519.PublicKey)
	publicKeyB64 := base64.RawStdEncoding.EncodeToString(publicKey)

	validUntil := time.Now().Add(time.Duration(n.validityHours) * time.Hour)

	response := &ServerKeysResponse{
		ServerName:   n.serverName,
		ValidUntilTS: validUntil.UnixMilli(),
		VerifyKeys: map[string]VerifyKeyResponse{
			n.serverKeyID: {Key: publicKeyB64},
		},
		OldVerifyKeys: make(map[string]OldKeyResponse),
	}

	if err := n.signResponse(response); err != nil {
		return nil, fmt.Errorf("failed to sign own keys response: %w", err)
	}

	return response, nil
}

// signResponse signs a response with this notary's key
func (n *Notary) signResponse(response *ServerKeysResponse) error {
	toSign := map[string]interface{}{
		"server_name":     response.ServerName,
		"valid_until_ts":  response.ValidUntilTS,
		"verify_keys":     response.VerifyKeys,
		"old_verify_keys": response.OldVerifyKeys,
	}

	canonicalBytes, err := canonical.Marshal(toSign)
	if err != nil {
		return fmt.Errorf("canonical JSON marshaling failed: %w", err)
	}

	signature := ed25519.Sign(n.serverKeyPair, canonicalBytes)
	signatureB64 := base64.RawStdEncoding.EncodeToString(signature)

	if response.Signatures == nil {
		response.Signatures = make(map[string]map[string]string)
	}
	response.Signatures[n.serverName] = map[string]string{
		n.serverKeyID: signatureB64,
	}
	return nil
}

// addNotarySignature adds notary's perspective signature to response
func (n *Notary) addNotarySignature(response *ServerKeysResponse) error {
	toSign := map[string]interface{}{
		"server_name":     response.ServerName,
		"valid_until_ts":  response.ValidUntilTS,
		"verify_keys":     response.VerifyKeys,
		"old_verify_keys": response.OldVerifyKeys,
	}

	// Include signatures from other signers only.
	// Our own prior signature must never be part of the payload we sign.
	if response.Signatures != nil {
		signatures := make(map[string]map[string]string, len(response.Signatures))
		for signer, signerSigs := range response.Signatures {
			if signer == n.serverName {
				continue
			}
			copied := make(map[string]string, len(signerSigs))
			for keyID, value := range signerSigs {
				copied[keyID] = value
			}
			signatures[signer] = copied
		}
		if len(signatures) > 0 {
			toSign["signatures"] = signatures
		}
	}

	canonicalBytes, err := canonical.Marshal(toSign)
	if err != nil {
		return fmt.Errorf("canonical JSON marshaling failed: %w", err)
	}

	signature := ed25519.Sign(n.serverKeyPair, canonicalBytes)
	signatureB64 := base64.RawStdEncoding.EncodeToString(signature)

	if response.Signatures == nil {
		response.Signatures = make(map[string]map[string]string)
	}
	if response.Signatures[n.serverName] == nil {
		response.Signatures[n.serverName] = make(map[string]string)
	}
	response.Signatures[n.serverName][n.serverKeyID] = signatureB64
	return nil
}
