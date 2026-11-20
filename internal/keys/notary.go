/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Mon 22 Jun 2026 00:50:40 UTC
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
	"mxkeys/internal/zero/signedjson"
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
	fetchTimeout     time.Duration
	queryDeadline    time.Duration
	expiredRetention time.Duration
	trustPolicy      *TrustPolicy
	transparency     *TransparencyLog
	analytics        *Analytics
	keyBroadcastHook func(serverName, keyID, keyData string, validUntilTS int64)

	configMu sync.RWMutex // protects runtime configuration setters
}

type cachedResponse struct {
	response   *ServerKeysResponse
	validUntil time.Time
	fetchedAt  time.Time
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
	// QueryTimeout bounds one POST /_matrix/key/v2/query. It must stay
	// below the HTTP write timeout so the response can still be sent.
	QueryTimeout time.Duration
	// ExpiredRetention is how long a key document is kept after expiry.
	ExpiredRetention time.Duration
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
		serverName:       cfg.ServerName,
		storage:          storage,
		fetcher:          fetcher,
		cache:            make(map[string]*cachedResponse),
		validityHours:    cfg.ValidityHours,
		cacheTTLHours:    cfg.CacheTTLHours,
		fetchTimeout:     cfg.FetchTimeout,
		queryDeadline:    cfg.QueryTimeout,
		expiredRetention: cfg.ExpiredRetention,
	}
	if n.expiredRetention <= 0 {
		n.expiredRetention = defaultExpiredRetention
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

// GetOwnKeys returns this notary's own key document, signed by itself. Raw
// holds the signed canonical bytes, which are what the handler serves.
func (n *Notary) GetOwnKeys() (*ServerKeysResponse, error) {
	publicKey := n.serverKeyPair.Public().(ed25519.PublicKey)
	validUntil := time.Now().Add(time.Duration(n.validityHours) * time.Hour)

	unsigned, err := canonical.Marshal(map[string]interface{}{
		"server_name":    n.serverName,
		"valid_until_ts": validUntil.UnixMilli(),
		"verify_keys": map[string]interface{}{
			n.serverKeyID: map[string]interface{}{"key": base64.RawStdEncoding.EncodeToString(publicKey)},
		},
		"old_verify_keys": map[string]interface{}{},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode own keys document: %w", err)
	}
	signed, err := signedjson.Sign(unsigned, n.serverName, n.serverKeyID, n.serverKeyPair)
	if err != nil {
		return nil, fmt.Errorf("failed to sign own keys document: %w", err)
	}
	response, _, err := parseServerKeys(signed)
	if err != nil {
		return nil, fmt.Errorf("own keys document is malformed: %w", err)
	}
	return response, nil
}
