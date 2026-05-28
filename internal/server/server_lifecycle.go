/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

package server

import (
	"context"
	"net/http"
	"time"

	"mxkeys/internal/zero/log"
)

// Shutdown timing defaults. Operators may override via config fields
// `server.shutdown_timeout` and `server.predrain_delay`, both ignored
// when zero. Defaults reflect Kubernetes / haproxy operator expectations:
// the instance stays visible on /_mxkeys/livez, serves /_mxkeys/readyz
// with 503 during predrainDelay, then drains open HTTP connections.
const (
	defaultShutdownTimeout = 30 * time.Second
	defaultPredrainDelay   = 5 * time.Second
)

// gracefulShutdown performs a kube-friendly graceful shutdown:
//
//  1. Flip shuttingDown=true so readiness returns 503 immediately.
//     Upstream LBs observe this and drop the instance from rotation.
//  2. Sleep predrainDelay so the LB has time to propagate the state
//     before we start draining HTTP (otherwise clients hit a newly
//     unreachable instance during LB's poll-jitter window).
//  3. srv.Shutdown(shutdownCtx) drains in-flight HTTP requests.
//  4. Stop rate limiter, cluster, notary cleanup.
//  5. Close DB handle last.
//
// All steps are best-effort; errors are logged but do not abort the
// sequence. The function returns the first non-nil error encountered.
func (s *Server) gracefulShutdown(srv *http.Server) error {
	log.Info("Initiating graceful shutdown...")

	// 1. Mark draining; readiness probe starts returning 503.
	s.shuttingDown.Store(true)

	// 2. Predrain delay: let LBs observe readyz=503.
	predrain := s.predrainDelay()
	if predrain > 0 {
		log.Info("Predrain delay before HTTP drain",
			"delay", predrain.String(),
			"purpose", "LB propagation",
		)
		time.Sleep(predrain)
	}

	// 3. HTTP drain.
	timeout := s.shutdownTimeout()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Info("Stopping HTTP server...", "timeout", timeout.String())
	var firstErr error
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error", "error", err)
		firstErr = err
	}

	// 4. Subsystems.
	if s.rateLimiter != nil {
		s.rateLimiter.Stop()
	}

	if s.cluster != nil {
		log.Info("Stopping cluster...")
		if err := s.cluster.Stop(); err != nil {
			log.Error("Cluster shutdown error", "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	if s.notary != nil {
		s.notary.StopCleanupRoutine()
	}

	// 5. Database last.
	log.Info("Closing database connection...")
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			log.Error("Database close error", "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	if firstErr == nil {
		log.Info("Graceful shutdown complete")
	} else {
		log.Warn("Graceful shutdown completed with errors", "first_error", firstErr)
	}
	return firstErr
}

// shutdownTimeout returns the HTTP drain budget.
func (s *Server) shutdownTimeout() time.Duration {
	if s.config != nil && s.config.Server.ShutdownTimeout > 0 {
		return s.config.Server.ShutdownTimeout
	}
	return defaultShutdownTimeout
}

// predrainDelay returns the pause between flipping readyz=503 and
// starting the HTTP drain.
func (s *Server) predrainDelay() time.Duration {
	if s.config != nil && s.config.Server.PredrainDelay > 0 {
		return s.config.Server.PredrainDelay
	}
	return defaultPredrainDelay
}

// Close closes the server out-of-band (not via signal handling).
// Intended for tests and for the deferred cleanup in cmd/mxkeys/main.go
// that runs after Run returns. Idempotent.
func (s *Server) Close() error {
	s.shuttingDown.Store(true)
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
