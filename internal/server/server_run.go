/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Wed Apr 08 2026 UTC
 * Status: Created
 */

package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"mxkeys/internal/zero/log"
)

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

	srv := s.newHTTPServer(addr)

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

func (s *Server) newHTTPServer(addr string) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}
}
