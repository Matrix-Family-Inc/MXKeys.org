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
	"bufio"
	"os"
	"strings"
)

// Load reads a minimal-subset YAML config file into a nested map.
// Supports: scalars, nested mappings, string/scalar list items, and one-level
// mapping list items (e.g. `- key_id: value`). Comments start with `#`.
// Not a full YAML parser: suitable for simple operator config only.
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

	for scanner.Scan() {
		line := scanner.Text()

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent := lineIndent(line)

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
			activeListItem, activeListIndent = consumeListItem(result, currentPath, trimmed, indent)
			continue
		}

		if activeListItem != nil && indent > activeListIndent {
			k, v, ok := parseKeyValue(trimmed)
			if !ok {
				continue
			}
			if v == "" {
				activeListItem[k] = make(map[string]interface{})
				continue
			}
			activeListItem[k] = parseValue(v)
			continue
		}

		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if value == "" {
			currentPath = append(currentPath, key)
			indentStack = append(indentStack, indent)
		} else {
			fullPath := strings.Join(append(currentPath, key), ".")
			setPath(result, fullPath, parseValue(value))
		}
	}

	return result, scanner.Err()
}

// lineIndent treats tabs as 2 spaces (common YAML convention).
func lineIndent(line string) int {
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
	return indent
}

// consumeListItem parses a list element line and returns the mapping head and
// its indent so subsequent nested keys can be attached.
func consumeListItem(result map[string]interface{}, currentPath []string, trimmed string, indent int) (map[string]interface{}, int) {
	item := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
	listPath := strings.Join(currentPath, ".")
	existing := getPath(result, listPath)
	var list []interface{}
	if l, ok := existing.([]interface{}); ok {
		list = l
	}

	if item == "" {
		itemMap := make(map[string]interface{})
		list = append(list, itemMap)
		setPath(result, listPath, list)
		return itemMap, indent
	}

	if strings.Contains(item, ": ") {
		k, v, ok := parseKeyValue(item)
		if !ok {
			return nil, -1
		}
		itemMap := map[string]interface{}{k: parseValue(v)}
		list = append(list, itemMap)
		setPath(result, listPath, list)
		return itemMap, indent
	}

	list = append(list, parseValue(item))
	setPath(result, listPath, list)
	return nil, -1
}
