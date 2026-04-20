/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

package nettls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// generateSelfSignedCA produces a CA certificate + key, writes both to
// disk, and returns the paths. The CA is valid for 1 hour, which is
// plenty for the lifetime of the test.
func generateSelfSignedCA(t *testing.T, dir, name string) (certPath, keyPath string, cert *x509.Certificate, key *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}
	tpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             time.Now().Add(-1 * time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate CA: %v", err)
	}
	cert, err = x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate CA: %v", err)
	}
	certPath = filepath.Join(dir, name+".crt")
	keyPath = filepath.Join(dir, name+".key")
	writePEM(t, certPath, "CERTIFICATE", der)
	kb, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("MarshalECPrivateKey: %v", err)
	}
	writePEM(t, keyPath, "EC PRIVATE KEY", kb)
	return
}

func generateLeaf(t *testing.T, dir, name string, caCert *x509.Certificate, caKey *ecdsa.PrivateKey) (certPath, keyPath string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate leaf key: %v", err)
	}
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: name},
		NotBefore:    time.Now().Add(-1 * time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		t.Fatalf("CreateCertificate leaf: %v", err)
	}
	certPath = filepath.Join(dir, name+".crt")
	keyPath = filepath.Join(dir, name+".key")
	writePEM(t, certPath, "CERTIFICATE", der)
	kb, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("MarshalECPrivateKey leaf: %v", err)
	}
	writePEM(t, keyPath, "EC PRIVATE KEY", kb)
	return
}

func writePEM(t *testing.T, path, typ string, der []byte) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	defer f.Close()
	if err := pem.Encode(f, &pem.Block{Type: typ, Bytes: der}); err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
}

// TestDisabledFallsBackToPlainTCP validates the critical backward-compat
// property: an empty TLSConfig produces a plain TCP listener, and
// DialTimeout returns a plain TCP connection. No surprise encryption.
func TestDisabledFallsBackToPlainTCP(t *testing.T) {
	ln, err := Listen("tcp", "127.0.0.1:0", Config{})
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer ln.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		_, _ = io.WriteString(conn, "hello")
		_ = conn.Close()
	}()

	c, err := DialTimeout("tcp", ln.Addr().String(), 2*time.Second, Config{})
	if err != nil {
		t.Fatalf("DialTimeout: %v", err)
	}
	buf := make([]byte, 5)
	if _, err := io.ReadFull(c, buf); err != nil {
		t.Fatalf("ReadFull: %v", err)
	}
	_ = c.Close()
	<-done
	if string(buf) != "hello" {
		t.Fatalf("got %q, want hello", string(buf))
	}
}

// TestMutualTLSHandshakeSucceeds wires a CA-signed cert on both sides
// and verifies that a round-trip works end-to-end.
func TestMutualTLSHandshakeSucceeds(t *testing.T) {
	dir := t.TempDir()
	caCertPath, _, caCert, caKey := generateSelfSignedCA(t, dir, "ca")
	srvCert, srvKey := generateLeaf(t, dir, "server", caCert, caKey)
	cliCert, cliKey := generateLeaf(t, dir, "client", caCert, caKey)

	srvCfg := Config{
		Enabled:           true,
		CertFile:          srvCert,
		KeyFile:           srvKey,
		CAFile:            caCertPath,
		RequireClientCert: true,
		MinVersion:        "1.3",
	}
	cliCfg := Config{
		Enabled:           true,
		CertFile:          cliCert,
		KeyFile:           cliKey,
		CAFile:            caCertPath,
		RequireClientCert: true,
		MinVersion:        "1.3",
	}

	ln, err := Listen("tcp", "127.0.0.1:0", srvCfg)
	if err != nil {
		t.Fatalf("Listen TLS: %v", err)
	}
	defer ln.Close()

	done := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		if tlsConn, ok := conn.(*tls.Conn); ok {
			if err := tlsConn.Handshake(); err != nil {
				done <- err
				return
			}
		}
		_, err = io.WriteString(conn, "encrypted-hello")
		done <- err
	}()

	c, err := DialTimeout("tcp", ln.Addr().String(), 3*time.Second, cliCfg)
	if err != nil {
		t.Fatalf("DialTimeout TLS: %v", err)
	}
	buf := make([]byte, len("encrypted-hello"))
	if _, err := io.ReadFull(c, buf); err != nil {
		t.Fatalf("ReadFull: %v", err)
	}
	_ = c.Close()

	if err := <-done; err != nil {
		t.Fatalf("server side: %v", err)
	}
	if string(buf) != "encrypted-hello" {
		t.Fatalf("got %q, want encrypted-hello", string(buf))
	}
}

// TestMutualTLSRejectsUntrustedClient confirms the RequireClientCert
// path: a client whose cert is not issued by the server's CA must be
// rejected.
func TestMutualTLSRejectsUntrustedClient(t *testing.T) {
	dir := t.TempDir()
	serverCA, _, serverCACert, serverCAKey := generateSelfSignedCA(t, dir, "ca-server")
	srvCert, srvKey := generateLeaf(t, dir, "server", serverCACert, serverCAKey)

	// A DIFFERENT CA signs the client: the server must not trust it.
	_, _, otherCACert, otherCAKey := generateSelfSignedCA(t, dir, "ca-other")
	cliCert, cliKey := generateLeaf(t, dir, "client", otherCACert, otherCAKey)

	srvCfg := Config{
		Enabled:           true,
		CertFile:          srvCert,
		KeyFile:           srvKey,
		CAFile:            serverCA,
		RequireClientCert: true,
		MinVersion:        "1.3",
	}
	cliCfg := Config{
		Enabled:           true,
		CertFile:          cliCert,
		KeyFile:           cliKey,
		CAFile:            serverCA,
		RequireClientCert: true,
		MinVersion:        "1.3",
	}

	ln, err := Listen("tcp", "127.0.0.1:0", srvCfg)
	if err != nil {
		t.Fatalf("Listen TLS: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		if tlsConn, ok := conn.(*tls.Conn); ok {
			_ = tlsConn.Handshake()
		}
		_ = conn.Close()
	}()

	// In TLS 1.3 the untrusted-client failure can surface either on
	// the Dial itself (server alert during handshake) or on the first
	// I/O operation after dial (server closed the connection with
	// bad_certificate). Both are acceptable; "none of the above" is
	// the regression we guard against.
	c, err := DialTimeout("tcp", ln.Addr().String(), 3*time.Second, cliCfg)
	if err == nil {
		// Server must have rejected; a read should surface the error.
		_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
		buf := make([]byte, 1)
		_, rerr := c.Read(buf)
		_ = c.Close()
		if rerr == nil {
			t.Fatal("expected failure reading from rejected TLS connection")
		}
	}
}

// TestValidateRejectsMissingFiles covers the early-startup-fail
// behaviour: a config that names non-existent files must not pass
// Validate, saving operators from discovering the problem only when
// cluster traffic first tries to connect.
func TestValidateRejectsMissingFiles(t *testing.T) {
	cfg := Config{
		Enabled:  true,
		CertFile: "/does/not/exist.crt",
		KeyFile:  "/does/not/exist.key",
		CAFile:   "/does/not/exist.pem",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation failure for missing files")
	}
}

// TestConfigMinVersionDefaulting accepts blank or recognised values
// and rejects nothing at the config layer (defaults kick in later).
func TestConfigMinVersionDefaulting(t *testing.T) {
	for _, in := range []string{"", "1.2", "1.3"} {
		if minVersion(in) == 0 {
			t.Fatalf("minVersion(%q) = 0", in)
		}
	}
	// Unknown strings fall back to 1.3, never to an insecure level.
	if got := minVersion("sslv3"); got != tls.VersionTLS13 {
		t.Fatalf("expected unknown to default to TLS 1.3, got %x", got)
	}
}

func TestLoadCAPoolRejectsGarbage(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bogus.pem")
	if err := os.WriteFile(p, []byte("this is not a pem bundle"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := loadCAPool(p); err == nil {
		t.Fatal("expected loadCAPool to reject non-PEM input")
	}
}
