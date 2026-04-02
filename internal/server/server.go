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

package server

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"mxkeys/internal/cluster"
	"mxkeys/internal/config"
	"mxkeys/internal/keys"
	"mxkeys/internal/zero/log"
	"mxkeys/internal/zero/merkle"
	"mxkeys/internal/zero/metrics"
)

// Server is the HTTP server
type Server struct {
	config       *config.Config
	notary       *keys.Notary
	mux          *http.ServeMux
	db           *sql.DB
	rateLimiter  *RateLimiter
	startTime    time.Time
	transparency *keys.TransparencyLog
	analytics    *keys.Analytics
	trustPolicy  *keys.TrustPolicy
	cluster      *cluster.Cluster
	merkleTree   *merkle.Tree
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
	notary, err := keys.NewNotary(
		db,
		cfg.Server.Name,
		cfg.Keys.StoragePath,
		cfg.Keys.ValidityHours,
		cfg.Keys.CacheTTLHours,
		cfg.TrustedServers.Fallback,
		fetchTimeout,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create notary: %w", err)
	}

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

	s := &Server{
		config:      cfg,
		notary:      notary,
		mux:         http.NewServeMux(),
		db:          db,
		rateLimiter: rateLimiter,
		startTime:   time.Now(),
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
			log.Error("Failed to initialize transparency log", "error", err)
		} else {
			s.transparency = transparencyLog
			s.merkleTree = merkle.New()
			log.Info("Transparency log enabled")
		}
	}

	// Analytics is always available (lightweight)
	s.analytics = keys.NewAnalytics(db, keys.AnalyticsConfig{
		Enabled: true,
	})

	if cfg.Cluster.Enabled {
		clusterCfg := cluster.ClusterConfig{
			NodeID:       cfg.Server.Name,
			BindAddress:  cfg.Cluster.BindAddress,
			BindPort:     cfg.Cluster.BindPort,
			Seeds:        cfg.Cluster.Seeds,
			SyncInterval: cfg.Cluster.SyncInterval,
		}
		c, err := cluster.NewCluster(clusterCfg)
		if err != nil {
			log.Error("Failed to initialize cluster", "error", err)
		} else {
			s.cluster = c
			log.Info("Cluster mode enabled", "node", cfg.Server.Name)
		}
	}

	s.setupRoutes()

	return s, nil
}

// setupRoutes sets up the HTTP routes
func (s *Server) setupRoutes() {
	// Health checks, status and metrics
	s.mux.HandleFunc("GET /_mxkeys/health", s.handleHealth)
	s.mux.HandleFunc("GET /_mxkeys/live", s.handleLiveness)
	s.mux.HandleFunc("GET /_mxkeys/ready", s.handleReadiness)
	s.mux.HandleFunc("GET /_mxkeys/status", s.handleStatus)
	s.mux.Handle("GET /_mxkeys/metrics", metrics.Handler())

	// Matrix Key Server API v2
	// GET /_matrix/key/v2/server - own keys (no keyID)
	s.mux.HandleFunc("GET /_matrix/key/v2/server", s.handleServerKeys)
	// GET /_matrix/key/v2/server/{keyID} - own keys with keyID (Go 1.22+ path params)
	s.mux.HandleFunc("GET /_matrix/key/v2/server/{keyID}", s.handleServerKeys)

	// POST /_matrix/key/v2/query - notary query (stricter rate limit)
	s.mux.HandleFunc("POST /_matrix/key/v2/query", s.withQueryRateLimit(s.handleKeyQuery))

	// Version endpoint
	s.mux.HandleFunc("GET /_matrix/federation/v1/version", s.handleVersion)

	// Enterprise API endpoints
	s.registerTransparencyRoutes()
}

// withQueryRateLimit wraps a handler with query-specific rate limiting
func (s *Server) withQueryRateLimit(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		v := s.rateLimiter.getVisitor(ip)

		if !v.queryLimiter.Allow() {
			RecordRateLimited()
			writeRateLimitError(w)
			return
		}

		h(w, r)
	}
}

// Handler returns the HTTP handler with all middleware applied
func (s *Server) Handler() http.Handler {
	// Chain middleware: request ID -> security headers -> logging -> rate limiting -> routes
	handler := http.Handler(s.mux)
	handler = s.rateLimiter.Middleware(handler)
	handler = loggingMiddleware(handler)
	handler = SecurityHeadersMiddleware(handler)
	handler = RequestIDMiddleware(handler)
	return handler
}

// Run starts the server
func (s *Server) Run(ctx context.Context) error {
	// Run initial cleanup
	s.notary.RunCleanup()

	// Start cleanup routine
	cleanupInterval := time.Duration(s.config.Keys.CleanupHours) * time.Hour
	s.notary.StartCleanupRoutine(ctx, cleanupInterval)

	// Start cluster if enabled
	if s.cluster != nil {
		if err := s.cluster.Start(ctx); err != nil {
			log.Error("Failed to start cluster", "error", err)
		}
	}

	addr := fmt.Sprintf("%s:%d", s.config.Server.BindAddress, s.config.Server.Port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      s.Handler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Info("Starting MXKeys notary server", "address", addr, "server", s.config.Server.Name)

	errChan := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		return s.gracefulShutdown(srv)
	case err := <-errChan:
		return err
	}
}

// gracefulShutdown performs graceful shutdown
func (s *Server) gracefulShutdown(srv *http.Server) error {
	log.Info("Initiating graceful shutdown...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop accepting new requests
	log.Info("Stopping HTTP server...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error", "error", err)
	}

	// Flush any pending metrics
	log.Info("Flushing metrics...")

	// Close database connection
	log.Info("Closing database connection...")
	if err := s.db.Close(); err != nil {
		log.Error("Database close error", "error", err)
		return err
	}

	log.Info("Graceful shutdown complete")
	return nil
}

// Close closes the server
func (s *Server) Close() error {
	if s.cluster != nil {
		s.cluster.Stop()
	}
	return s.db.Close()
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		IncInFlightRequests()
		defer DecInFlightRequests()

		start := time.Now()

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		route := NormalizeRoute(r.URL.Path)

		RecordHTTPRequest(r.Method, route, strconv.Itoa(rw.statusCode), duration.Seconds())

		log.Debug("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration", duration,
			"remote", r.RemoteAddr,
		)
	})
}
