/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Created
 */

// Minimal WHOIS lookup for /_mxkeys/server-info. Pulls a small
// subset of fields (registrar, registered / updated / expires
// dates, nameservers) from whatever TLD registry answers on
// TCP 43. We deliberately discard the bulk of the record since
// most TLDs ship pages of PII boilerplate that we must not
// re-publish.

package server

import (
	"context"
	"strings"
	"time"

	whoislib "github.com/likexian/whois"
)

const (
	serverInfoWhoisTimeout     = 4 * time.Second
	serverInfoWhoisMaxLines    = 2000
	serverInfoWhoisMaxNS       = 8
)

// whoisAllowed reports whether WHOIS is safe to run for the
// target. We refuse for anything that does not look like a
// public DNS name (IPv6 literal, localhost, unresolved label)
// and for well-known TLDs that do not support port 43 WHOIS at
// all.
func whoisAllowed(host string) bool {
	if host == "" {
		return false
	}
	if strings.HasPrefix(host, "[") {
		return false
	}
	if !strings.Contains(host, ".") {
		return false
	}
	return true
}

// runWhois performs the lookup and returns a sanitised subset
// of the record. Errors are swallowed at the caller; a nil
// return means the record is not worth surfacing.
func runWhois(ctx context.Context, host string) *ServerInfoWhois {
	ctx, cancel := context.WithTimeout(ctx, serverInfoWhoisTimeout)
	defer cancel()
	if !whoisAllowed(host) {
		return nil
	}
	ch := make(chan string, 1)
	go func() {
		client := whoislib.NewClient()
		client.SetTimeout(serverInfoWhoisTimeout)
		raw, err := client.Whois(host)
		if err != nil {
			ch <- ""
			return
		}
		ch <- raw
	}()

	var raw string
	select {
	case <-ctx.Done():
		return nil
	case raw = <-ch:
	}
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return parseWhois(raw)
}

// parseWhois pulls the handful of well-known fields out of a
// WHOIS record, tolerant to the dozens of TLD-specific
// formatting conventions. Field keys are matched case-
// insensitively; values are first-wins (many registries repeat
// fields through multiple referrals).
func parseWhois(raw string) *ServerInfoWhois {
	out := &ServerInfoWhois{}
	seenNS := make(map[string]struct{})
	lines := strings.Split(raw, "\n")
	if len(lines) > serverInfoWhoisMaxLines {
		lines = lines[:serverInfoWhoisMaxLines]
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "%") || strings.HasPrefix(line, "#") {
			continue
		}
		key, value := splitWhoisLine(line)
		if key == "" || value == "" {
			continue
		}
		switch key {
		case "registrar":
			if out.Registrar == "" {
				out.Registrar = value
			}
		case "creation date", "created", "registration date", "registered",
			"domain registration date", "registered on":
			if out.Registered == "" {
				out.Registered = normaliseWhoisDate(value)
			}
		case "updated date", "last updated", "last modified", "changed":
			if out.Updated == "" {
				out.Updated = normaliseWhoisDate(value)
			}
		case "registry expiry date", "expiry date", "paid-till",
			"registrar registration expiration date", "expiration date",
			"expires on", "expire":
			if out.Expires == "" {
				out.Expires = normaliseWhoisDate(value)
			}
		case "name server", "nserver", "nameservers":
			ns := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(value), "."))
			if ns == "" {
				continue
			}
			if _, ok := seenNS[ns]; ok {
				continue
			}
			if len(out.Nameservers) >= serverInfoWhoisMaxNS {
				continue
			}
			seenNS[ns] = struct{}{}
			out.Nameservers = append(out.Nameservers, ns)
		}
	}
	if out.Registrar == "" && out.Registered == "" &&
		out.Updated == "" && out.Expires == "" && len(out.Nameservers) == 0 {
		return nil
	}
	return out
}

// splitWhoisLine splits a "key: value" pair tolerant to the
// varied colon / whitespace conventions across TLD registries.
// Keys are lowercased and trimmed; values keep internal spaces.
func splitWhoisLine(line string) (string, string) {
	idx := strings.IndexByte(line, ':')
	if idx == -1 {
		return "", ""
	}
	key := strings.TrimSpace(strings.ToLower(line[:idx]))
	value := strings.TrimSpace(line[idx+1:])
	return key, value
}

// normaliseWhoisDate keeps only the date portion of common WHOIS
// timestamps so operators see a stable YYYY-MM-DD. Falls back to
// the raw string when nothing obvious parses.
func normaliseWhoisDate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"2006.01.02",
		"02-Jan-2006",
	} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format("2006-01-02")
		}
	}
	if len(raw) >= 10 {
		return raw[:10]
	}
	return raw
}
