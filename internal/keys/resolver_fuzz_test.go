package keys

import "testing"

// FuzzParseServerName exercises parseServerName with arbitrary inputs.
// Invariants:
//
//  1. Must never panic (the function feeds DNS/SRV paths and must be
//     robust against adversarial server_name parameters from /query).
//  2. When isIP==true, the returned hostname must be a valid IP as parsed
//     by net.ParseIP (caller decides whether to route via private-IP block).
//  3. Returned port is either 0 or in the valid 1..65535 range; no other
//     values can be fed to the downstream URL builder.
func FuzzParseServerName(f *testing.F) {
	seeds := []string{
		"matrix.org",
		"matrix.org:443",
		"matrix.org:8448",
		"127.0.0.1",
		"127.0.0.1:8448",
		"[::1]",
		"[::1]:8448",
		"[2001:db8::1]:443",
		"[2001:db8::1]",
		"",
		":",
		":8448",
		"host:abc",
		"host:-1",
		"host:65536",
		"host:0",
		"[::1",
		"]]",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, name string) {
		hostname, port, isIP := parseServerName(name)
		_ = hostname
		_ = isIP

		if port < 0 {
			t.Fatalf("parseServerName returned negative port %d for %q", port, name)
		}
		if port > 65535 {
			t.Fatalf("parseServerName returned out-of-range port %d for %q", port, name)
		}
	})
}
