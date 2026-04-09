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

package keys

import (
	"context"
	"fmt"
	"net"
)

// SetBlockPrivateIPs enables or disables resolved-address SSRF protection.
func (f *Fetcher) SetBlockPrivateIPs(enabled bool) {
	f.blockPrivateIPs = enabled
}

func (f *Fetcher) rejectPrivateAddress(ctx context.Context, serverName string, resolved *ResolvedServer) error {
	if !f.blockPrivateIPs || resolved == nil {
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
	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("resolved private IP address %s is blocked for %s", ip.String(), serverName)
		}
	}
	return nil
}
