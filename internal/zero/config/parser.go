/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix.Family Inc. (https://matrix.family)
 * Owner: Matrix.Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 20 Nov 2026 23:15:00 UTC
 * Status: Updated
 */

package config

import (
	"strings"
)

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
