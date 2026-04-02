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
 * - No escape sequences other than those required for control chars
 */

package canonical

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
)

const maxSafeInteger = 9007199254740991 // 2^53 - 1

// JSON converts a JSON byte slice to canonical form
func JSON(input []byte) ([]byte, error) {
	var v interface{}
	dec := json.NewDecoder(bytes.NewReader(input))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	// Reject trailing non-whitespace tokens to enforce strict single-document parsing.
	var extra interface{}
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("invalid JSON: trailing tokens")
	}
	return Marshal(v)
}

// Marshal marshals a value to canonical JSON
func Marshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := writeValue(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeValue(buf *bytes.Buffer, v interface{}) error {
	switch val := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if val {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case float64:
		if err := writeFloatNumber(buf, val); err != nil {
			return err
		}
	case json.Number:
		if err := writeJSONNumber(buf, val); err != nil {
			return err
		}
	case int:
		return writeInteger(buf, int64(val))
	case int8:
		return writeInteger(buf, int64(val))
	case int16:
		return writeInteger(buf, int64(val))
	case int32:
		return writeInteger(buf, int64(val))
	case int64:
		return writeInteger(buf, val)
	case uint:
		if val > uint(maxSafeInteger) {
			return fmt.Errorf("integer out of canonical range")
		}
		buf.WriteString(strconv.FormatUint(uint64(val), 10))
	case uint8:
		buf.WriteString(strconv.FormatUint(uint64(val), 10))
	case uint16:
		buf.WriteString(strconv.FormatUint(uint64(val), 10))
	case uint32:
		buf.WriteString(strconv.FormatUint(uint64(val), 10))
	case uint64:
		if val > uint64(maxSafeInteger) {
			return fmt.Errorf("integer out of canonical range")
		}
		buf.WriteString(strconv.FormatUint(val, 10))
	case string:
		writeString(buf, val)
	case []interface{}:
		if err := writeArray(buf, val); err != nil {
			return err
		}
	case map[string]interface{}:
		if err := writeObject(buf, val); err != nil {
			return err
		}
	default:
		// Fallback for other types
		b, err := json.Marshal(val)
		if err != nil {
			return err
		}
		buf.Write(b)
	}
	return nil
}

func writeInteger(buf *bytes.Buffer, n int64) error {
	if n < -maxSafeInteger || n > maxSafeInteger {
		return fmt.Errorf("integer out of canonical range")
	}
	buf.WriteString(strconv.FormatInt(n, 10))
	return nil
}

func writeFloatNumber(buf *bytes.Buffer, n float64) error {
	if math.IsNaN(n) || math.IsInf(n, 0) {
		return fmt.Errorf("invalid numeric value")
	}
	if math.Trunc(n) != n {
		return fmt.Errorf("non-integer numbers are not allowed in canonical JSON")
	}
	if n < -maxSafeInteger || n > maxSafeInteger {
		return fmt.Errorf("integer out of canonical range")
	}
	buf.WriteString(strconv.FormatInt(int64(n), 10))
	return nil
}

func writeJSONNumber(buf *bytes.Buffer, n json.Number) error {
	s := n.String()
	if strings.ContainsAny(s, ".eE") {
		return fmt.Errorf("non-integer numbers are not allowed in canonical JSON")
	}

	i := new(big.Int)
	if _, ok := i.SetString(s, 10); !ok {
		return fmt.Errorf("invalid numeric value")
	}

	max := big.NewInt(maxSafeInteger)
	min := big.NewInt(-maxSafeInteger)
	if i.Cmp(max) > 0 || i.Cmp(min) < 0 {
		return fmt.Errorf("integer out of canonical range")
	}

	buf.WriteString(i.String())
	return nil
}

func writeString(buf *bytes.Buffer, s string) {
	buf.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if r < 0x20 {
				// Control character - use \uXXXX
				buf.WriteString(`\u00`)
				buf.WriteByte(hexDigit((r >> 4) & 0xF))
				buf.WriteByte(hexDigit(r & 0xF))
			} else {
				buf.WriteRune(r)
			}
		}
	}
	buf.WriteByte('"')
}

func hexDigit(n rune) byte {
	const digits = "0123456789abcdef"
	if n < 0 || n > 15 {
		return '0'
	}
	return digits[int(n)]
}

func writeArray(buf *bytes.Buffer, arr []interface{}) error {
	buf.WriteByte('[')
	for i, v := range arr {
		if i > 0 {
			buf.WriteByte(',')
		}
		if err := writeValue(buf, v); err != nil {
			return err
		}
	}
	buf.WriteByte(']')
	return nil
}

func writeObject(buf *bytes.Buffer, obj map[string]interface{}) error {
	// Sort keys lexicographically
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	buf.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		writeString(buf, k)
		buf.WriteByte(':')
		if err := writeValue(buf, obj[k]); err != nil {
			return err
		}
	}
	buf.WriteByte('}')
	return nil
}
