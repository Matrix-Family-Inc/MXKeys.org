/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Sat Mar 15 2026 UTC
 * Status: Created
 */

package server

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	maxServerNameLength = 255
	maxKeyIDLength      = 128
)

var (
	serverNameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-\.]*[a-zA-Z0-9])?(\:[0-9]{1,5})?$`)
	keyIDRegex      = regexp.MustCompile(`^ed25519:[a-zA-Z0-9_]+$`)
)

// ValidationConfig holds validation settings
type ValidationConfig struct {
	MaxServerNameLength int
	MaxServersPerQuery  int
	MaxJSONDepth        int
	MaxSignaturesPerKey int
}

// DefaultValidationConfig returns default validation config
func DefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		MaxServerNameLength: 255,
		MaxServersPerQuery:  100,
		MaxJSONDepth:        10,
		MaxSignaturesPerKey: 10,
	}
}

// ValidateServerName validates a Matrix server name
func ValidateServerName(name string, maxLen int) error {
	if name == "" {
		return fmt.Errorf("server name is empty")
	}

	if maxLen <= 0 {
		maxLen = maxServerNameLength
	}

	if utf8.RuneCountInString(name) > maxLen {
		return fmt.Errorf("server name too long: %d > %d", utf8.RuneCountInString(name), maxLen)
	}

	if !utf8.ValidString(name) {
		return fmt.Errorf("server name contains invalid UTF-8")
	}

	// Check for control characters
	for _, r := range name {
		if r < 32 || r == 127 {
			return fmt.Errorf("server name contains control characters")
		}
	}

	// Basic format check: hostname[:port] or IP[:port]
	if !isValidServerNameFormat(name) {
		return fmt.Errorf("invalid server name format: %s", name)
	}

	return nil
}

// ValidateKeyID validates a Matrix key ID format
func ValidateKeyID(keyID string) error {
	if keyID == "" {
		return fmt.Errorf("key ID is empty")
	}

	if len(keyID) > maxKeyIDLength {
		return fmt.Errorf("key ID too long: %d > %d", len(keyID), maxKeyIDLength)
	}

	// Matrix key IDs must be in format: algorithm:key_id
	// Currently only ed25519 is supported
	if !strings.HasPrefix(keyID, "ed25519:") {
		return fmt.Errorf("unsupported key algorithm: %s", keyID)
	}

	// Validate key_id part
	parts := strings.SplitN(keyID, ":", 2)
	if len(parts) != 2 || parts[1] == "" {
		return fmt.Errorf("invalid key ID format: %s", keyID)
	}

	// Key ID part should only contain alphanumeric and underscore
	for _, c := range parts[1] {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return fmt.Errorf("key ID contains invalid character: %c", c)
		}
	}

	return nil
}

func isValidServerNameFormat(name string) bool {
	// Check for IP literal [IPv6]:port
	if strings.HasPrefix(name, "[") {
		closeBracket := strings.Index(name, "]")
		if closeBracket == -1 {
			return false
		}
		// IPv6 literal
		ipv6 := name[1:closeBracket]
		if !isValidIPv6(ipv6) {
			return false
		}
		// Check port if present
		if len(name) > closeBracket+1 {
			if name[closeBracket+1] != ':' {
				return false
			}
			port := name[closeBracket+2:]
			return isValidPort(port)
		}
		return true
	}

	// Split hostname and port
	var hostname, port string
	if colonIdx := strings.LastIndex(name, ":"); colonIdx != -1 {
		// Check if this is IPv4:port or hostname:port
		potentialHost := name[:colonIdx]
		potentialPort := name[colonIdx+1:]
		if isValidPort(potentialPort) {
			hostname = potentialHost
			port = potentialPort
		} else {
			hostname = name
		}
	} else {
		hostname = name
	}

	// Validate hostname
	if !isValidHostname(hostname) && !isValidIPv4(hostname) {
		return false
	}

	// Validate port if present
	if port != "" && !isValidPort(port) {
		return false
	}

	return true
}

func isValidHostname(h string) bool {
	if len(h) == 0 {
		return false
	}

	if !isASCII(h) {
		// Matrix server names are expected to be ASCII hostnames.
		// IDNA names must be provided in punycode form (xn--...).
		return false
	}

	if len(h) > 253 {
		return false
	}

	// Cannot start or end with hyphen or dot
	if h[0] == '-' || h[0] == '.' || h[len(h)-1] == '-' || h[len(h)-1] == '.' {
		return false
	}

	// Check each label
	labels := strings.Split(h, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-') {
				return false
			}
		}
	}

	return true
}

func isValidIPv4(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		if len(part) == 0 || len(part) > 3 {
			return false
		}
		num := 0
		for _, c := range part {
			if c < '0' || c > '9' {
				return false
			}
			num = num*10 + int(c-'0')
		}
		if num > 255 {
			return false
		}
		// Leading zeros not allowed (except for 0 itself)
		if len(part) > 1 && part[0] == '0' {
			return false
		}
	}
	return true
}

func isValidIPv6(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed != nil && strings.Contains(ip, ":")
}

func isValidPort(p string) bool {
	if len(p) == 0 || len(p) > 5 {
		return false
	}
	num := 0
	for _, c := range p {
		if c < '0' || c > '9' {
			return false
		}
		num = num*10 + int(c-'0')
	}
	return num > 0 && num <= 65535
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}
