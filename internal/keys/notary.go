/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

package keys

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"mxkeys/internal/keys/keyprovider"
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

// NotaryConfig holds the configuration required to build a Notary.
// Use this struct over the legacy positional NewNotary constructor.
type NotaryConfig struct {
	ServerName          string
	KeyProvider         keyprovider.Provider
	ValidityHours       int
	CacheTTLHours       int
	FallbackServers     []string
	FetchTimeout        time.Duration
	TrustedNotaries     []TrustedNotaryKey
	MaxSignaturesPerKey int
}

// NewNotaryWithConfig constructs a Notary from the richer config struct and
// a pluggable signing-key Provider.
func NewNotaryWithConfig(ctx context.Context, db *sql.DB, cfg NotaryConfig) (*Notary, error) {
	if cfg.KeyProvider == nil {
		return nil, fmt.Errorf("notary: key provider is required")
	}

	storage, err := NewStorage(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	fetcher := NewFetcherWithConfig(FetcherConfig{
		FallbackServers: cfg.FallbackServers,
		Timeout:         cfg.FetchTimeout,
		TrustedNotaries: cfg.TrustedNotaries,
		MaxSignatures:   cfg.MaxSignaturesPerKey,
	})

	n := &Notary{
		serverName:    cfg.ServerName,
		storage:       storage,
		fetcher:       fetcher,
		cache:         make(map[string]*cachedResponse),
		validityHours: cfg.ValidityHours,
		cacheTTLHours: cfg.CacheTTLHours,
	}

	priv, keyID, err := cfg.KeyProvider.LoadOrGenerate(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load signing key: %w", err)
	}
	n.serverKeyPair = priv
	n.serverKeyID = keyID
	log.Info("Notary signing key loaded", "key_id", keyID, "provider", string(cfg.KeyProvider.Kind()))

	return n, nil
}

// NewNotary is a backwards-compatible wrapper that uses the default file-based
// key provider. New call sites should prefer NewNotaryWithConfig.
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
	provider, err := keyprovider.New(keyprovider.Config{
		Kind:        keyprovider.KindFile,
		StoragePath: keyStoragePath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build key provider: %w", err)
	}
	return NewNotaryWithConfig(context.Background(), db, NotaryConfig{
		ServerName:          serverName,
		KeyProvider:         provider,
		ValidityHours:       validityHours,
		CacheTTLHours:       cacheTTLHours,
		FallbackServers:     fallbackServers,
		FetchTimeout:        fetchTimeout,
		TrustedNotaries:     trustedNotaries,
		MaxSignaturesPerKey: maxSignaturesPerKey,
	})
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
