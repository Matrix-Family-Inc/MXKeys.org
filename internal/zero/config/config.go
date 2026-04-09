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

package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Load loads a YAML config file into a map
func Load(path string) (map[string]interface{}, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	result := make(map[string]interface{})
	scanner := bufio.NewScanner(file)
	var currentPath []string
	var indentStack []int
	var activeListItem map[string]interface{}
	activeListIndent := -1

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines and comments
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Calculate indent
		indent := 0
		for _, c := range line {
			if c == ' ' {
				indent++
			} else if c == '\t' {
				indent += 2
			} else {
				break
			}
		}

		// Adjust current path based on indent
		for len(indentStack) > 0 && indent <= indentStack[len(indentStack)-1] {
			indentStack = indentStack[:len(indentStack)-1]
			if len(currentPath) > 0 {
				currentPath = currentPath[:len(currentPath)-1]
			}
		}
		if activeListItem != nil && indent <= activeListIndent {
			activeListItem = nil
			activeListIndent = -1
		}

		if strings.HasPrefix(trimmed, "- ") {
			item := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			listPath := strings.Join(currentPath, ".")
			existing := getPath(result, listPath)
			var list []interface{}
			if existing != nil {
				if l, ok := existing.([]interface{}); ok {
					list = l
				}
			}

			if item == "" {
				itemMap := make(map[string]interface{})
				list = append(list, itemMap)
				setPath(result, listPath, list)
				activeListItem = itemMap
				activeListIndent = indent
				continue
			}

			if strings.Contains(item, ": ") {
				k, v, ok := parseKeyValue(item)
				if !ok {
					continue
				}
				itemMap := map[string]interface{}{
					k: parseValue(v),
				}
				list = append(list, itemMap)
				setPath(result, listPath, list)
				activeListItem = itemMap
				activeListIndent = indent
				continue
			}

			list = append(list, parseValue(item))
			setPath(result, listPath, list)
			activeListItem = nil
			activeListIndent = -1
			continue
		}

		if activeListItem != nil && indent > activeListIndent {
			k, v, ok := parseKeyValue(trimmed)
			if !ok {
				continue
			}
			if v == "" {
				nested := make(map[string]interface{})
				activeListItem[k] = nested
				continue
			}
			activeListItem[k] = parseValue(v)
			continue
		}

		// Parse key: value
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if value == "" {
			// Nested object
			currentPath = append(currentPath, key)
			indentStack = append(indentStack, indent)
		} else {
			// Simple value
			fullPath := strings.Join(append(currentPath, key), ".")
			setPath(result, fullPath, parseValue(value))
		}
	}

	return result, scanner.Err()
}

func parseValue(s string) interface{} {
	// Remove quotes
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}

	// Try bool
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}

	// Try int
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}

	// Try float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	return s
}

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

// GetString gets a string value from config
func GetString(m map[string]interface{}, path string) string {
	v := getPath(m, path)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// GetInt gets an int value from config
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

// GetFloat gets a float64 value from config
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

// GetBool gets a bool value from config
func GetBool(m map[string]interface{}, path string) bool {
	v := getPath(m, path)
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// GetStringSlice gets a string slice from config
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

// GetMapSlice gets a slice of map objects from config.
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

// Has reports whether a path exists in config map.
func Has(m map[string]interface{}, path string) bool {
	return getPath(m, path) != nil
}

// WithEnvOverride overrides config values with environment variables
// Env format: PREFIX_SECTION_KEY (e.g., MXKEYS_SERVER_PORT)
func WithEnvOverride(m map[string]interface{}, prefix string) {
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]

		if !strings.HasPrefix(key, prefix+"_") {
			continue
		}

		// Convert MXKEYS_SERVER_PORT to server.port
		path := strings.ToLower(strings.TrimPrefix(key, prefix+"_"))
		path = strings.ReplaceAll(path, "_", ".")

		setPath(m, path, parseValue(value))
	}
}

// Validate validates required config fields
func Validate(m map[string]interface{}, required []string) error {
	for _, path := range required {
		if getPath(m, path) == nil {
			return fmt.Errorf("missing required config: %s", path)
		}
	}
	return nil
}
