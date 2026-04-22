/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Created
 */

package server

import "testing"

// golden record trimmed to the subset of fields the parser must
// accept; the real WHOIS responses are much longer and contain
// boilerplate we deliberately ignore.
const whoisVerisign = `
Domain Name: EXAMPLE.ORG
Registry Domain ID: 2336799_DOMAIN_ORG-IANA
Registrar WHOIS Server: whois.iana.org
Registrar URL: http://res-dom.iana.org
Updated Date: 2024-08-14T07:01:31Z
Creation Date: 1995-08-14T04:00:00Z
Registry Expiry Date: 2025-08-13T04:00:00Z
Registrar: Internet Assigned Numbers Authority
Name Server: A.IANA-SERVERS.NET
Name Server: B.IANA-SERVERS.NET
DNSSEC: signedDelegation
`

func TestParseWhoisExtractsCoreFields(t *testing.T) {
	got := parseWhois(whoisVerisign)
	if got == nil {
		t.Fatal("parseWhois returned nil for a populated record")
	}
	if got.Registrar != "Internet Assigned Numbers Authority" {
		t.Errorf("registrar = %q", got.Registrar)
	}
	if got.Registered != "1995-08-14" {
		t.Errorf("registered = %q", got.Registered)
	}
	if got.Expires != "2025-08-13" {
		t.Errorf("expires = %q", got.Expires)
	}
	if got.Updated != "2024-08-14" {
		t.Errorf("updated = %q", got.Updated)
	}
	if len(got.Nameservers) != 2 ||
		got.Nameservers[0] != "a.iana-servers.net" ||
		got.Nameservers[1] != "b.iana-servers.net" {
		t.Errorf("nameservers = %v", got.Nameservers)
	}
}

func TestParseWhoisReturnsNilForEmpty(t *testing.T) {
	if got := parseWhois(""); got != nil {
		t.Fatalf("empty input should return nil, got %#v", got)
	}
	if got := parseWhois("% no match found\n% more comments"); got != nil {
		t.Fatalf("comments-only input should return nil, got %#v", got)
	}
}

func TestWhoisAllowed(t *testing.T) {
	cases := map[string]bool{
		"":              false,
		"[::1]":         false,
		"localhost":     false,
		"matrix.org":    true,
		"example.co.uk": true,
	}
	for in, want := range cases {
		if got := whoisAllowed(in); got != want {
			t.Errorf("whoisAllowed(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestNormaliseWhoisDate(t *testing.T) {
	cases := map[string]string{
		"2024-08-14T07:01:31Z": "2024-08-14",
		"2024-08-14":           "2024-08-14",
		"2024.08.14":           "2024-08-14",
		"14-Aug-2024":          "2024-08-14",
		"malformed":            "malformed",
		"":                     "",
	}
	for in, want := range cases {
		if got := normaliseWhoisDate(in); got != want {
			t.Errorf("normalise(%q) = %q, want %q", in, got, want)
		}
	}
}
