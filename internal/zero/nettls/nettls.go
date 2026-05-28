/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

// Package nettls provides TLS configuration and listen/dial helpers
// shared by the cluster (CRDT and Raft) transports.
//
// Design goals:
//
//   - Opt-in: a zero TLSConfig yields plain TCP listen/dial (backward
//     compatible with existing clear-text cluster deployments).
//   - Mutual authentication by default when TLS is enabled: the server
//     verifies the client cert against the configured CA bundle, and
//     vice versa. Operators who need one-way TLS (rare in cluster
//     deployments) can set RequireClientCert=false.
//   - TLS 1.3 by default, TLS 1.2 opt-in for unusual environments.
//   - Zero third-party dependencies: uses crypto/tls and crypto/x509
//     only.
//
// The package does not implement certificate rotation
// watchers (SIGHUP-driven cert reload, inotify, ...). Rolling restart
// with Kubernetes / systemd is the supported rotation path; see
// docs/runbook/cluster-tls-rotation.md.
package nettls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"time"
)

// Config describes a cluster-transport TLS policy.
type Config struct {
	// Enabled toggles TLS. When false all fields are ignored and both
	// listen/dial operate over plain TCP.
	Enabled bool

	// CertFile and KeyFile are the PEM-encoded server certificate and
	// private key presented on listen and (when RequireClientCert is
	// true) on dial. Both must be set when Enabled=true.
	CertFile string
	KeyFile  string

	// CAFile is the PEM bundle used to verify the peer certificate on
	// the opposite side of the connection. Required on both sides.
	CAFile string

	// RequireClientCert turns on mutual TLS. Listeners refuse clients
	// without a valid certificate; dialers present their cert to the
	// server. Recommended for every production deployment.
	RequireClientCert bool

	// MinVersion is kept for forward compatibility but is ignored:
	// the cluster transport is TLS 1.3 only. Operators running
	// legacy peers that cannot speak TLS 1.3 must upgrade them
	// before enabling cluster TLS here.
	MinVersion string

	// ServerName is the SNI / expected CN for dial-side verification.
	// When empty, the host portion of the dial address is used.
	ServerName string
}

// IsEnabled reports whether Config activates TLS.
func (c Config) IsEnabled() bool { return c.Enabled }

// Validate checks that mandatory fields are present when TLS is enabled.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.CertFile == "" {
		return errors.New("nettls: cert_file is required when tls.enabled=true")
	}
	if c.KeyFile == "" {
		return errors.New("nettls: key_file is required when tls.enabled=true")
	}
	if c.CAFile == "" {
		return errors.New("nettls: ca_file is required when tls.enabled=true")
	}
	if _, err := os.Stat(c.CertFile); err != nil {
		return fmt.Errorf("nettls: cert_file: %w", err)
	}
	if _, err := os.Stat(c.KeyFile); err != nil {
		return fmt.Errorf("nettls: key_file: %w", err)
	}
	if _, err := os.Stat(c.CAFile); err != nil {
		return fmt.Errorf("nettls: ca_file: %w", err)
	}
	return nil
}

// tlsMinVersion is the single TLS version floor this package
// configures. Cluster transport is greenfield: there is no legacy
// peer to preserve, and TLS 1.2 adds a materially larger attack
// surface (ciphersuite selection, renegotiation, ...) for no
// interop benefit.
const tlsMinVersion = tls.VersionTLS13

// loadCAPool reads a PEM bundle and returns a CertPool ready for use
// as ClientCAs or RootCAs.
func loadCAPool(path string) (*x509.CertPool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("nettls: read ca: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(raw) {
		return nil, errors.New("nettls: ca bundle contains no valid PEM certificates")
	}
	return pool, nil
}

// ServerConfig returns a *tls.Config suitable for wrapping a listener.
// Returns nil and no error when Config.Enabled is false (caller then
// uses the plain TCP listener).
func ServerConfig(c Config) (*tls.Config, error) {
	if !c.Enabled {
		return nil, nil
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("nettls: server cert/key: %w", err)
	}
	pool, err := loadCAPool(c.CAFile)
	if err != nil {
		return nil, err
	}
	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tlsMinVersion,
		ClientCAs:    pool,
	}
	if c.RequireClientCert {
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	} else {
		cfg.ClientAuth = tls.VerifyClientCertIfGiven
	}
	return cfg, nil
}

// ClientConfig returns a *tls.Config for dial-side use. Returns nil
// and no error when Config.Enabled is false.
func ClientConfig(c Config) (*tls.Config, error) {
	if !c.Enabled {
		return nil, nil
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	pool, err := loadCAPool(c.CAFile)
	if err != nil {
		return nil, err
	}
	cfg := &tls.Config{
		MinVersion: tlsMinVersion,
		RootCAs:    pool,
	}
	if c.RequireClientCert {
		cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("nettls: client cert/key: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	if c.ServerName != "" {
		cfg.ServerName = c.ServerName
	}
	return cfg, nil
}

// Listen returns a net.Listener optionally wrapped in TLS.
func Listen(network, address string, c Config) (net.Listener, error) {
	base, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}
	scfg, err := ServerConfig(c)
	if err != nil {
		_ = base.Close()
		return nil, err
	}
	if scfg == nil {
		return base, nil
	}
	return tls.NewListener(base, scfg), nil
}

// DialTimeout returns a net.Conn optionally wrapped in TLS. The
// serverName default is the host portion of address.
//
// Equivalent to DialContext with a background context and the
// given timeout. Kept for legacy call sites that have no
// context; every path that does carry a context should call
// DialContext directly so cancellation actually interrupts the
// connect + TLS handshake stages.
func DialTimeout(network, address string, timeout time.Duration, c Config) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return DialContext(ctx, network, address, timeout, c)
}

// DialContext dials network:address, optionally wraps the
// connection in TLS, and honours ctx cancellation at BOTH the
// TCP-connect stage and the TLS-handshake stage. timeout is the
// hard per-attempt cap applied via net.Dialer.Timeout on top of
// ctx; pass zero to rely on ctx alone.
//
// Cancellation contract: if ctx fires before net.Dial returns,
// the dial is aborted immediately (net.Dialer.DialContext). If
// ctx fires after TCP is up but before the TLS handshake
// finishes, HandshakeContext tears the connection down and
// returns ctx.Err(). Every internal goroutine exits before
// DialContext returns; no watcher is leaked.
func DialContext(ctx context.Context, network, address string, timeout time.Duration, c Config) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: timeout}
	if !c.Enabled {
		return dialer.DialContext(ctx, network, address)
	}
	ccfg, err := ClientConfig(c)
	if err != nil {
		return nil, err
	}
	if ccfg.ServerName == "" {
		host, _, splitErr := net.SplitHostPort(address)
		if splitErr == nil && host != "" {
			ccfg = ccfg.Clone()
			ccfg.ServerName = host
		}
	}
	raw, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	tlsConn := tls.Client(raw, ccfg)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = tlsConn.Close()
		return nil, err
	}
	return tlsConn, nil
}
