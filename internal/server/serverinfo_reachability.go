/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Created
 */

// DNS / well-known / SRV / TLS reachability probe for
// /_mxkeys/server-info. Purely informational: the probe does
// NOT consult the notary's signing-key cache and never triggers
// a federation key fetch. It only describes what the visitor
// would see if they tried to federate with the target right now.

package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// serverInfoWellKnownTimeout bounds the well-known HTTP fetch
	// end-to-end; the probe must not hold the enrichment budget
	// on a slow third-party web server.
	serverInfoWellKnownTimeout = 2 * time.Second

	// serverInfoTLSTimeout bounds the TCP connect + TLS handshake
	// probe against the resolved federation port.
	serverInfoTLSTimeout = 2 * time.Second

	// serverInfoWellKnownMaxBytes is the hard cap for the body we
	// read from /.well-known/matrix/server. The Matrix spec
	// bounds the JSON document to a handful of fields; anything
	// larger is either misconfigured or hostile.
	serverInfoWellKnownMaxBytes = 64 * 1024
)

// probeReachability runs the DNS + well-known + SRV + TLS chain
// for serverName and returns what each stage discovered plus an
// optional error string. It never calls the notary key fetcher
// and never consumes the signing-key cache.
func probeReachability(ctx context.Context, serverName string) (*ServerInfoDNS, *ServerInfoReachability) {
	host, explicitPort := splitHostPort(serverName)

	dns := &ServerInfoDNS{}
	if explicitPort == 0 {
		if wk, ok := fetchWellKnown(ctx, host); ok {
			dns.WellKnownServer = wk
		}
		dns.SRV = lookupSRV(ctx, host)
	}

	resolvedHost, resolvedPort := resolveFederationTarget(host, explicitPort, dns)
	dns.ResolvedHost = resolvedHost
	dns.ResolvedPort = resolvedPort
	dns.A, dns.AAAA = lookupHostAddresses(ctx, resolvedHost)

	reach := probeTLS(ctx, resolvedHost, resolvedPort, host)
	return dns, reach
}

// splitHostPort separates a Matrix server_name into host and
// explicit port (0 when absent). Bracketed IPv6 literals are
// preserved intact.
func splitHostPort(serverName string) (string, int) {
	if strings.HasPrefix(serverName, "[") {
		end := strings.Index(serverName, "]")
		if end == -1 {
			return serverName, 0
		}
		host := serverName[1:end]
		rest := serverName[end+1:]
		if strings.HasPrefix(rest, ":") {
			if p, err := strconv.Atoi(rest[1:]); err == nil && p > 0 && p <= 65535 {
				return host, p
			}
		}
		return host, 0
	}
	idx := strings.LastIndex(serverName, ":")
	if idx == -1 {
		return serverName, 0
	}
	if p, err := strconv.Atoi(serverName[idx+1:]); err == nil && p > 0 && p <= 65535 {
		return serverName[:idx], p
	}
	return serverName, 0
}

// fetchWellKnown tries `https://<host>/.well-known/matrix/server`
// and returns the `m.server` delegation target when the document
// is well-formed.
func fetchWellKnown(ctx context.Context, host string) (string, bool) {
	ctx, cancel := context.WithTimeout(ctx, serverInfoWellKnownTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://"+host+"/.well-known/matrix/server", nil)
	if err != nil {
		return "", false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, serverInfoWellKnownMaxBytes))
	if err != nil {
		return "", false
	}
	var wk struct {
		Server string `json:"m.server"`
	}
	if err := json.Unmarshal(body, &wk); err != nil {
		return "", false
	}
	wk.Server = strings.TrimSpace(wk.Server)
	if wk.Server == "" {
		return "", false
	}
	return wk.Server, true
}

// lookupSRV returns `_matrix-fed._tcp.<host>` records first and
// falls back to the deprecated `_matrix._tcp.<host>` form.
func lookupSRV(ctx context.Context, host string) []ServerInfoSRVTarget {
	if v := srvLookup(ctx, "matrix-fed", host); len(v) > 0 {
		return v
	}
	return srvLookup(ctx, "matrix", host)
}

func srvLookup(ctx context.Context, service, host string) []ServerInfoSRVTarget {
	var r net.Resolver
	_, srvs, err := r.LookupSRV(ctx, service, "tcp", host)
	if err != nil {
		return nil
	}
	out := make([]ServerInfoSRVTarget, 0, len(srvs))
	for _, s := range srvs {
		out = append(out, ServerInfoSRVTarget{
			Target:   strings.TrimSuffix(s.Target, "."),
			Port:     int(s.Port),
			Priority: int(s.Priority),
			Weight:   int(s.Weight),
		})
	}
	return out
}

// resolveFederationTarget folds well-known + SRV into a single
// (host, port) per Matrix S2S discovery. Fallback port is 8448
// when nothing else applies.
func resolveFederationTarget(host string, explicitPort int, dns *ServerInfoDNS) (string, int) {
	if explicitPort != 0 {
		return host, explicitPort
	}
	if dns.WellKnownServer != "" {
		h, p := splitHostPort(dns.WellKnownServer)
		if p == 0 {
			p = 8448
		}
		return h, p
	}
	if len(dns.SRV) > 0 {
		return dns.SRV[0].Target, dns.SRV[0].Port
	}
	return host, 8448
}

// lookupHostAddresses returns A and AAAA records for host,
// split by family so the UI can show them in two columns.
func lookupHostAddresses(ctx context.Context, host string) ([]string, []string) {
	var r net.Resolver
	addrs, err := r.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, nil
	}
	var v4, v6 []string
	for _, a := range addrs {
		if a.IP.To4() != nil {
			v4 = append(v4, a.IP.String())
		} else {
			v6 = append(v6, a.IP.String())
		}
	}
	return v4, v6
}

// probeTLS dials the federation port with a full TLS handshake
// and reports the negotiated version, the elapsed time, and
// whether SNI matched the expected Matrix server_name.
func probeTLS(ctx context.Context, host string, port int, sni string) *ServerInfoReachability {
	out := &ServerInfoReachability{FederationPort: port}
	ctx, cancel := context.WithTimeout(ctx, serverInfoTLSTimeout)
	defer cancel()
	dialer := &net.Dialer{}
	target := net.JoinHostPort(host, strconv.Itoa(port))
	start := time.Now()
	rawConn, err := dialer.DialContext(ctx, "tcp", target)
	if err != nil {
		out.Error = classifyReachabilityError(err)
		return out
	}
	defer rawConn.Close()
	tlsConn := tls.Client(rawConn, &tls.Config{
		ServerName: sni,
		MinVersion: tls.VersionTLS12,
	})
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		out.Error = classifyReachabilityError(err)
		return out
	}
	state := tlsConn.ConnectionState()
	out.Reachable = true
	out.RTTMS = time.Since(start).Milliseconds()
	out.TLSVersion = tlsVersionName(state.Version)
	out.TLSSNIMatch = state.ServerName == sni
	return out
}

func tlsVersionName(v uint16) string {
	switch v {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("0x%04x", v)
	}
}

// classifyReachabilityError turns noisy low-level errors into a
// single short phrase safe to render verbatim in a public UI
// panel. Avoids leaking internal IPs or resolver state.
func classifyReachabilityError(err error) string {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "connection refused"):
		return "connection refused"
	case strings.Contains(msg, "no such host"), strings.Contains(msg, "NXDOMAIN"):
		return "DNS lookup failed"
	case strings.Contains(msg, "unreachable"):
		return "network unreachable"
	case strings.Contains(msg, "handshake"), strings.Contains(msg, "tls:"), strings.Contains(msg, "x509"):
		return "TLS handshake failed"
	default:
		return "unreachable"
	}
}
