/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

package config

import (
	"strconv"
	"strings"
)

// parseValue converts a raw YAML scalar into a typed value.
// Precedence: quoted string -> bool -> int64 -> float64 -> string fallback.
func parseValue(s string) interface{} {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}

	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}

	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}

	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	return s
}

// parseKeyValue splits a single "key: value" line.
// Surrounding quotes on value are stripped.
func parseKeyValue(s string) (string, string, bool) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	value = strings.Trim(value, `"`)
	return key, value, key != ""
}

// setPath sets a value at dot-separated path, creating intermediate maps.
func setPath(m map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	current := m
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
		} else {
			if _, ok := current[part]; !ok {
				current[part] = make(map[string]interface{})
			}
			if next, ok := current[part].(map[string]interface{}); ok {
				current = next
			}
		}
	}
}

// getPath reads a value at dot-separated path, nil if missing.
func getPath(m map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = m
	for _, part := range parts {
		if part == "" {
			continue
		}
		if mp, ok := current.(map[string]interface{}); ok {
			current = mp[part]
		} else {
			return nil
		}
	}
	return current
}
