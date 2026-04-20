/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Tue Jan 27 2026 UTC
 * Status: Created
 */

package server

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"mxkeys/internal/cluster"
	"mxkeys/internal/config"
	"mxkeys/internal/keys"
	"mxkeys/internal/zero/log"
)

// Server is the HTTP server
type Server struct {
	config                *config.Config
	notary                *keys.Notary
	mux                   *http.ServeMux
	db                    *sql.DB
	rateLimiter           *RateLimiter
	startTime             time.Time
	transparency          *keys.TransparencyLog
	analytics             *keys.Analytics
	trustPolicy           *keys.TrustPolicy
	cluster               *cluster.Cluster
	enterpriseAccessToken string
}

// New creates a new server
func New(cfg *config.Config) (*Server, error) {
	// Connect to database
	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db.SetMaxOpenConns(cfg.Database.MaxConnections)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConnections)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create notary service
	fetchTimeout := time.Duration(cfg.Keys.FetchTimeoutS) * time.Second
	trustedNotaries, err := decodeTrustedNotaries(cfg.Security.TrustedNotaries)
	if err != nil {
		return nil, fmt.Errorf("failed to decode trusted notaries: %w", err)
	}
	notary, err := keys.NewNotary(
		db,
		cfg.Server.Name,
		cfg.Keys.StoragePath,
		cfg.Keys.ValidityHours,
		cfg.Keys.CacheTTLHours,
		cfg.TrustedServers.Fallback,
		fetchTimeout,
		trustedNotaries,
		cfg.Security.MaxSignaturesPerKey,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create notary: %w", err)
	}
	notary.SetBlockPrivateIPs(cfg.TrustPolicy.BlockPrivateIPs)

	// Configure rate limiter from config
	rlConfig := RateLimitConfig{
		GlobalRequestsPerSecond: cfg.RateLimit.RequestsPerSecond,
		GlobalBurst:             cfg.RateLimit.Burst,
		QueryRequestsPerSecond:  cfg.RateLimit.QueryPerSecond,
		QueryBurst:              cfg.RateLimit.QueryBurst,
	}
	if rlConfig.GlobalRequestsPerSecond <= 0 {
		rlConfig = DefaultRateLimitConfig()
	}
	rateLimiter := NewRateLimiter(rlConfig)
	if err := ConfigureClientIPPolicy(cfg.Security.TrustForwardedHeaders, cfg.Security.TrustedProxies); err != nil {
		return nil, fmt.Errorf("failed to configure client IP policy: %w", err)
	}

	s := &Server{
		config:                cfg,
		notary:                notary,
		mux:                   http.NewServeMux(),
		db:                    db,
		rateLimiter:           rateLimiter,
		startTime:             time.Now(),
		enterpriseAccessToken: cfg.Security.EnterpriseAccessToken,
	}

	// Initialize optional enterprise features
	if cfg.TrustPolicy.Enabled {
		s.trustPolicy = keys.NewTrustPolicy(keys.TrustPolicyConfig{
			Enabled:                 cfg.TrustPolicy.Enabled,
			DenyList:                cfg.TrustPolicy.DenyList,
			AllowList:               cfg.TrustPolicy.AllowList,
			RequireNotarySignatures: cfg.TrustPolicy.RequireNotarySignatures,
			MaxKeyAgeHours:          cfg.TrustPolicy.MaxKeyAgeHours,
			RequireWellKnown:        cfg.TrustPolicy.RequireWellKnown,
			RequireValidTLS:         cfg.TrustPolicy.RequireValidTLS,
			BlockPrivateIPs:         cfg.TrustPolicy.BlockPrivateIPs,
		})
		s.notary.SetTrustPolicy(s.trustPolicy)
		log.Info("Trust policy enabled")
	}

	if cfg.Transparency.Enabled {
		transparencyLog, err := keys.NewTransparencyLog(db, keys.TransparencyConfig{
			Enabled:       cfg.Transparency.Enabled,
			LogAllKeys:    cfg.Transparency.LogAllKeys,
			LogKeyChanges: cfg.Transparency.LogKeyChanges,
			LogAnomalies:  cfg.Transparency.LogAnomalies,
			RetentionDays: cfg.Transparency.RetentionDays,
			TableName:     cfg.Transparency.TableName,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize transparency log: %w", err)
		}
		s.transparency = transparencyLog
		s.notary.SetTransparencyLog(transparencyLog)
		log.Info("Transparency log enabled")
	}

	// Analytics is always available (lightweight)
	s.analytics = keys.NewAnalytics(db, keys.AnalyticsConfig{
		Enabled: true,
	})
	s.notary.SetAnalytics(s.analytics)

	if cfg.Cluster.Enabled {
		clusterNodeID := cfg.Cluster.NodeID
		if clusterNodeID == "" {
			clusterNodeID = cfg.Server.Name
		}
		clusterCfg := cluster.ClusterConfig{
			Enabled:          cfg.Cluster.Enabled,
			NodeID:           clusterNodeID,
			BindAddress:      cfg.Cluster.BindAddress,
			BindPort:         cfg.Cluster.BindPort,
			AdvertiseAddress: cfg.Cluster.AdvertiseAddress,
			AdvertisePort:    cfg.Cluster.AdvertisePort,
			Seeds:            cfg.Cluster.Seeds,
			ConsensusMode:    cfg.Cluster.ConsensusMode,
			SyncInterval:     cfg.Cluster.SyncInterval,
			SharedSecret:     cfg.Cluster.SharedSecret,
		}
		c, err := cluster.NewCluster(clusterCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize cluster: %w", err)
		}
		s.cluster = c
		log.Info("Cluster mode enabled",
			"node", clusterNodeID,
			"consensus_mode", cfg.Cluster.ConsensusMode,
		)
	}

	if s.cluster != nil {
		s.notary.SetKeyBroadcastHook(func(serverName, keyID, keyData string, validUntilTS int64) {
			s.cluster.BroadcastKeyUpdate(serverName, keyID, keyData, validUntilTS)
		})
		s.cluster.SetOnKeyReceived(func(serverName string, data []byte) {
			var entry cluster.KeyEntry
			if err := json.Unmarshal(data, &entry); err != nil {
				log.Warn("Failed to decode replicated cluster entry", "error", err)
				return
			}
			if entry.KeyID != keys.ClusterReplicatedResponseKeyID {
				return
			}
			if err := s.notary.ApplyReplicatedServerResponse(serverName, entry.KeyData, entry.ValidUntilTS); err != nil {
				log.Warn("Failed to apply replicated server response", "server", serverName, "error", err)
			}
		})
	}

	s.setupRoutes()

	return s, nil
}

func decodeTrustedNotaries(configured []config.TrustedNotary) ([]keys.TrustedNotaryKey, error) {
	if len(configured) == 0 {
		return nil, nil
	}
	out := make([]keys.TrustedNotaryKey, 0, len(configured))
	for _, item := range configured {
		pub, err := base64.RawStdEncoding.DecodeString(item.PublicKey)
		if err != nil {
			pub, err = base64.StdEncoding.DecodeString(item.PublicKey)
			if err != nil {
				return nil, fmt.Errorf("server=%s key_id=%s: invalid base64 public key", item.ServerName, item.KeyID)
			}
		}
		out = append(out, keys.TrustedNotaryKey{
			ServerName: item.ServerName,
			KeyID:      item.KeyID,
			PublicKey:  pub,
		})
	}
	return out, nil
}
