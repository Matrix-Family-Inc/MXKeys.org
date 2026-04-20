/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 08 2026 UTC
 * Status: Created
 */

package keys

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"
)

// SetBlockPrivateIPs enables or disables resolved-address SSRF protection.
func (f *Fetcher) SetBlockPrivateIPs(enabled bool) {
	f.blockPrivateIPs.Store(enabled)
}

func (f *Fetcher) rejectPrivateAddress(ctx context.Context, serverName string, resolved *ResolvedServer) error {
	if !f.blockPrivateIPs.Load() || resolved == nil {
		return nil
	}

	if ip := net.ParseIP(resolved.Host); ip != nil {
		if isPrivateIP(ip) {
			return fmt.Errorf("resolved private IP address %s is blocked for %s", resolved.Host, serverName)
		}
		return nil
	}

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", resolved.Host)
	if err != nil {
		return fmt.Errorf("failed to resolve %s for private IP check: %w", resolved.Host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("failed to resolve %s for private IP check: no addresses returned", resolved.Host)
	}

	pinned := make([]string, 0, len(ips))
	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("resolved private IP address %s is blocked for %s", ip.String(), serverName)
		}
		pinned = append(pinned, ip.String())
	}
	resolved.PinnedIPs = pinned
	return nil
}

func (f *Fetcher) clientForResolved(resolved *ResolvedServer) *http.Client {
	if resolved == nil || len(resolved.PinnedIPs) == 0 {
		return f.client
	}

	baseTransport, ok := f.client.Transport.(*http.Transport)
	if !ok || baseTransport == nil {
		return f.client
	}

	transport := baseTransport.Clone()
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	port := strconv.Itoa(resolved.Port)
	pinnedIPs := append([]string(nil), resolved.PinnedIPs...)
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		var lastErr error
		for _, ip := range pinnedIPs {
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		if lastErr == nil {
			lastErr = fmt.Errorf("no pinned IPs available for %s", resolved.Host)
		}
		return nil, lastErr
	}

	client := *f.client
	client.Transport = transport
	return &client
}
