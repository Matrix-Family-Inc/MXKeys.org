/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

package config

import "strconv"

// GetString returns a string value at path, or "" if missing/wrong type.
func GetString(m map[string]interface{}, path string) string {
	v := getPath(m, path)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// GetInt returns an int value at path, or 0 if missing/wrong type.
// Numeric types (int, int64, float64) are coerced.
func GetInt(m map[string]interface{}, path string) int {
	v := getPath(m, path)
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	}
	return 0
}

// GetFloat returns a float64 value at path, or 0 if missing/wrong type.
// Numeric types and parseable strings are coerced.
func GetFloat(m map[string]interface{}, path string) float64 {
	v := getPath(m, path)
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return 0
}

// GetBool returns a bool value at path, or false if missing/wrong type.
func GetBool(m map[string]interface{}, path string) bool {
	v := getPath(m, path)
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// GetStringSlice returns a []string at path. Non-string items are skipped.
func GetStringSlice(m map[string]interface{}, path string) []string {
	v := getPath(m, path)
	if arr, ok := v.([]interface{}); ok {
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// GetMapSlice returns a []map at path. Non-map items are skipped.
func GetMapSlice(m map[string]interface{}, path string) []map[string]interface{} {
	v := getPath(m, path)
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	result := make([]map[string]interface{}, 0, len(arr))
	for _, item := range arr {
		if mp, ok := item.(map[string]interface{}); ok {
			result = append(result, mp)
		}
	}
	return result
}

// Has reports whether a value exists at path.
func Has(m map[string]interface{}, path string) bool {
	return getPath(m, path) != nil
}
