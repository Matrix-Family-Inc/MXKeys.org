/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Created
 */

// Wire-facing types for /_mxkeys/server-info. Every field is
// explicitly nullable via `omitempty` so the response stays
// compact when a sub-lookup (WHOIS) is disabled or
// fails to produce a result within the per-task timeout.

package server

import "time"

// ServerInfoResponse is the JSON body returned by
// GET /_mxkeys/server-info?name=<host>. The top-level contract:
// every sub-section is optional; the handler always returns HTTP
// 200 with whatever succeeded within the request budget.
type ServerInfoResponse struct {
	ServerName   string                    `json:"server_name"`
	FetchedAt    time.Time                 `json:"fetched_at"`
	DNS          *ServerInfoDNS            `json:"dns,omitempty"`
	Reachability *ServerInfoReachability   `json:"reachability,omitempty"`
	Whois        *ServerInfoWhois          `json:"whois,omitempty"`
	Errors       map[string]string         `json:"errors,omitempty"`
}

// ServerInfoDNS captures the Matrix server discovery state: the
// well-known delegation target (if any), explicit SRV records,
// and resolved A/AAAA addresses for the final host.
type ServerInfoDNS struct {
	WellKnownServer string                 `json:"well_known_server,omitempty"`
	SRV             []ServerInfoSRVTarget  `json:"srv,omitempty"`
	ResolvedHost    string                 `json:"resolved_host,omitempty"`
	ResolvedPort    int                    `json:"resolved_port,omitempty"`
	A               []string               `json:"a,omitempty"`
	AAAA            []string               `json:"aaaa,omitempty"`
}

// ServerInfoSRVTarget mirrors one record from a Matrix SRV
// lookup (`_matrix-fed._tcp.<host>`).
type ServerInfoSRVTarget struct {
	Target   string `json:"target"`
	Port     int    `json:"port"`
	Priority int    `json:"priority"`
	Weight   int    `json:"weight"`
}

// ServerInfoReachability summarises the federation-port probe:
// did the TCP+TLS handshake succeed, what protocol was negotiated,
// and how many milliseconds did it take.
type ServerInfoReachability struct {
	FederationPort int    `json:"federation_port"`
	Reachable      bool   `json:"reachable"`
	TLSVersion     string `json:"tls_version,omitempty"`
	TLSSNIMatch    bool   `json:"tls_sni_match,omitempty"`
	RTTMS          int64  `json:"rtt_ms,omitempty"`
	Error          string `json:"error,omitempty"`
}

// ServerInfoWhois carries the handful of WHOIS fields worth
// surfacing to the visitor. The raw record is intentionally
// discarded: most registries ship pages of boilerplate and PII
// that should not leak into a public notary response.
type ServerInfoWhois struct {
	Registrar   string   `json:"registrar,omitempty"`
	Registered  string   `json:"registered,omitempty"`
	Expires     string   `json:"expires,omitempty"`
	Updated     string   `json:"updated,omitempty"`
	Nameservers []string `json:"nameservers,omitempty"`
}
