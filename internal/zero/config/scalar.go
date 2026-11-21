/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix.Family Inc. (https://matrix.family)
 * Owner: Matrix.Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 20 Nov 2026 23:15:00 UTC
 * Status: Created
 */

package config

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// stripComment removes a YAML comment. '#' starts a comment only at the
// beginning of the line or after whitespace, and never inside quotes.
func stripComment(line string) (string, error) {
	quote := byte(0)
	escaped := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if quote == '"' && escaped {
			escaped = false
			continue
		}
		if quote == '"' && c == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
			continue
		}
		if c == '#' && (i == 0 || line[i-1] == ' ' || line[i-1] == '\t') {
			return strings.TrimRight(line[:i], " \t"), nil
		}
	}
	if quote != 0 || escaped {
		return "", fmt.Errorf("unclosed quote")
	}
	return line, nil
}

// splitKeyValue splits one "key: value" line. A colon is a separator only
// when it ends the key or is followed by whitespace, so scalars such as
// "ed25519:auto" and "127.0.0.1:7947" stay scalars.
func splitKeyValue(s string) (string, string, error) {
	if s == "" {
		return "", "", fmt.Errorf("expected key: value")
	}
	switch s[0] {
	case '"', '\'':
		return "", "", fmt.Errorf("quoted keys are not supported")
	case '[', '{', '!', '&', '*', '|', '>', '?':
		return "", "", fmt.Errorf("expected key: value")
	}

	quote := byte(0)
	escaped := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote == '"' && escaped {
			escaped = false
			continue
		}
		if quote == '"' && c == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
			continue
		}
		if c != ':' {
			continue
		}
		if i+1 < len(s) && s[i+1] != ' ' && s[i+1] != '\t' {
			continue
		}
		key := strings.TrimSpace(s[:i])
		raw := ""
		if i+1 < len(s) {
			raw = strings.TrimSpace(s[i+1:])
		}
		if key == "" || strings.ContainsAny(key, " \t") {
			return "", "", fmt.Errorf("invalid key %q", key)
		}
		return key, raw, nil
	}
	return "", "", fmt.Errorf("expected key: value")
}

// parseValue types one scalar or one supported flow collection.
// Quoted text is always a string. Anything this subset cannot represent
// is an error.
func parseValue(s string) (interface{}, error) {
	if s == "{}" {
		return map[string]interface{}{}, nil
	}
	if s == "[]" {
		return []interface{}{}, nil
	}
	if s == "" {
		return "", nil
	}
	switch s[0] {
	case '{':
		return nil, fmt.Errorf("only the empty flow mapping {} is supported")
	case '[':
		return parseFlowSequence(s)
	case '"', '\'':
		return parseQuoted(s)
	case '|', '>', '&', '*', '!', '?':
		return nil, fmt.Errorf("unsupported YAML syntax %q", s)
	}
	if s == "true" {
		return true, nil
	}
	if s == "false" {
		return false, nil
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i, nil
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil && !strings.ContainsAny(s, "eE") {
		return f, nil
	}
	if strings.ContainsAny(s, "{}[]") {
		return nil, fmt.Errorf("unsupported YAML syntax %q", s)
	}
	return s, nil
}

func parseFlowSequence(s string) ([]interface{}, error) {
	if len(s) < 2 || s[0] != '[' || s[len(s)-1] != ']' {
		return nil, fmt.Errorf("malformed flow sequence %q", s)
	}
	inner := s[1 : len(s)-1]
	if strings.TrimSpace(inner) == "" {
		return []interface{}{}, nil
	}
	parts, err := splitFlowItems(inner)
	if err != nil {
		return nil, err
	}
	items := make([]interface{}, 0, len(parts))
	for _, part := range parts {
		item, err := parseFlowItem(part)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func parseFlowItem(s string) (interface{}, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty flow sequence item")
	}
	if s[0] == '[' || s[0] == '{' {
		return nil, fmt.Errorf("nested flow collections are not supported")
	}
	if s[0] != '"' && s[0] != '\'' && strings.Contains(s, ": ") {
		return nil, fmt.Errorf("flow mappings are not supported")
	}
	return parseValue(s)
}

// splitFlowItems splits a flow sequence body on commas outside quotes.
// A single trailing comma is ignored.
func splitFlowItems(s string) ([]string, error) {
	var parts []string
	var b strings.Builder
	quote := byte(0)
	escaped := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote == '"' && escaped {
			b.WriteByte(c)
			escaped = false
			continue
		}
		if quote == '"' && c == '\\' {
			b.WriteByte(c)
			escaped = true
			continue
		}
		if quote != 0 {
			b.WriteByte(c)
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			quote = c
			b.WriteByte(c)
		case '[', '{':
			return nil, fmt.Errorf("nested flow collections are not supported")
		case ',':
			parts = append(parts, b.String())
			b.Reset()
		default:
			b.WriteByte(c)
		}
	}
	if quote != 0 || escaped {
		return nil, fmt.Errorf("unclosed quote in flow sequence")
	}
	rest := b.String()
	if strings.TrimSpace(rest) != "" || len(parts) == 0 {
		parts = append(parts, rest)
	}
	return parts, nil
}

func parseQuoted(s string) (string, error) {
	if len(s) < 2 {
		return "", fmt.Errorf("unclosed quote")
	}
	q := s[0]
	if s[len(s)-1] != q || !utf8.ValidString(s) {
		return "", fmt.Errorf("unclosed quote")
	}
	body := s[1 : len(s)-1]
	if q == '\'' {
		if strings.Count(body, "'")%2 != 0 {
			return "", fmt.Errorf("unescaped single quote")
		}
		return strings.ReplaceAll(body, "''", "'"), nil
	}
	var b strings.Builder
	for i := 0; i < len(body); i++ {
		if body[i] != '\\' {
			b.WriteByte(body[i])
			continue
		}
		if i+1 >= len(body) {
			return "", fmt.Errorf("trailing escape")
		}
		i++
		switch body[i] {
		case '\\', '"', '/':
			b.WriteByte(body[i])
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		default:
			return "", fmt.Errorf("unsupported escape \\%c", body[i])
		}
	}
	return b.String(), nil
}
