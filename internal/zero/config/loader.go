/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 20 Nov 2026 22:39:55 UTC
 * Status: Updated
 */

package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Load reads a YAML subset into a nested map: block mappings, block
// sequences of scalars or of one-level mappings, flow sequences of scalars
// ("[]", "[a, b]"), the empty flow mapping "{}", plain, 'single' and
// "double" quoted scalars, and comments ("#" at line start or after
// whitespace, outside quotes). Anything else is an error, so a file the
// loader cannot represent fails instead of being read differently.
func Load(path string) (map[string]interface{}, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return parse(file)
}

func parse(r io.Reader) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	scanner := bufio.NewScanner(r)
	var currentPath []string
	var indentStack []int
	var activeListItem map[string]interface{}
	activeListIndent := -1

	for lineNo := 1; scanner.Scan(); lineNo++ {
		line, err := stripComment(scanner.Text())
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		indent := lineIndent(line)
		for len(indentStack) > 0 && indent <= indentStack[len(indentStack)-1] {
			indentStack = indentStack[:len(indentStack)-1]
			if len(currentPath) == 0 {
				return nil, fmt.Errorf("line %d: indentation does not match the document structure", lineNo)
			}
			currentPath = currentPath[:len(currentPath)-1]
		}
		if activeListIndent >= 0 && indent <= activeListIndent {
			activeListItem = nil
			activeListIndent = -1
		}

		if trimmed == "-" || strings.HasPrefix(trimmed, "- ") {
			if len(currentPath) == 0 {
				return nil, fmt.Errorf("line %d: sequence item outside of a mapping key", lineNo)
			}
			activeListItem, err = consumeListItem(result, currentPath, strings.TrimSpace(strings.TrimPrefix(trimmed, "-")))
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNo, err)
			}
			activeListIndent = indent
			continue
		}

		key, raw, err := splitKeyValue(trimmed)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}

		if activeListIndent >= 0 && indent > activeListIndent {
			if activeListItem == nil {
				return nil, fmt.Errorf("line %d: nested content under a scalar sequence item", lineNo)
			}
			if raw == "" {
				return nil, fmt.Errorf("line %d: nested mappings inside sequence items are not supported", lineNo)
			}
			value, err := parseValue(raw)
			if err != nil {
				return nil, fmt.Errorf("line %d: %s: %w", lineNo, key, err)
			}
			activeListItem[key] = value
			continue
		}

		if raw == "" {
			currentPath = append(currentPath, key)
			indentStack = append(indentStack, indent)
			continue
		}
		value, err := parseValue(raw)
		if err != nil {
			return nil, fmt.Errorf("line %d: %s: %w", lineNo, key, err)
		}
		setPath(result, strings.Join(append(append([]string(nil), currentPath...), key), "."), value)
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

// consumeListItem appends one sequence item at listPath. An item of the
// form "key: value" starts a mapping that following deeper lines extend;
// it is returned so they can be attached.
func consumeListItem(result map[string]interface{}, currentPath []string, item string) (map[string]interface{}, error) {
	listPath := strings.Join(currentPath, ".")
	var list []interface{}
	switch existing := getPath(result, listPath).(type) {
	case nil:
	case []interface{}:
		list = existing
	default:
		return nil, fmt.Errorf("%s: sequence item mixed with a scalar or mapping", listPath)
	}

	if item == "" {
		itemMap := make(map[string]interface{})
		setPath(result, listPath, append(list, itemMap))
		return itemMap, nil
	}

	if key, raw, err := splitKeyValue(item); err == nil {
		if raw == "" {
			return nil, fmt.Errorf("%s: nested mappings inside sequence items are not supported", listPath)
		}
		value, err := parseValue(raw)
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", listPath, key, err)
		}
		itemMap := map[string]interface{}{key: value}
		setPath(result, listPath, append(list, itemMap))
		return itemMap, nil
	}

	value, err := parseValue(item)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", listPath, err)
	}
	setPath(result, listPath, append(list, value))
	return nil, nil
}
