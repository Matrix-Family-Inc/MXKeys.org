/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Fri 03 Apr 2026 UTC
 * Status: Created
 */

package server

import (
	"context"
	"net/http"
	"time"

	"mxkeys/internal/zero/log"
)

// gracefulShutdown performs graceful shutdown.
func (s *Server) gracefulShutdown(srv *http.Server) error {
	log.Info("Initiating graceful shutdown...")

	// Create shutdown context with timeout.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop accepting new requests.
	log.Info("Stopping HTTP server...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error", "error", err)
	}

	// Flush any pending metrics.
	log.Info("Flushing metrics...")

	if s.rateLimiter != nil {
		s.rateLimiter.Stop()
	}

	if s.cluster != nil {
		log.Info("Stopping cluster...")
		if err := s.cluster.Stop(); err != nil {
			log.Error("Cluster shutdown error", "error", err)
		}
	}

	if s.notary != nil {
		s.notary.StopCleanupRoutine()
	}

	// Close database connection.
	log.Info("Closing database connection...")
	if s.db == nil {
		log.Info("Graceful shutdown complete")
		return nil
	}
	if err := s.db.Close(); err != nil {
		log.Error("Database close error", "error", err)
		return err
	}

	log.Info("Graceful shutdown complete")
	return nil
}

// Close closes the server.
func (s *Server) Close() error {
	if s.cluster != nil {
		if err := s.cluster.Stop(); err != nil {
			return err
		}
	}
	if s.rateLimiter != nil {
		s.rateLimiter.Stop()
	}
	if s.notary != nil {
		s.notary.StopCleanupRoutine()
	}
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}
