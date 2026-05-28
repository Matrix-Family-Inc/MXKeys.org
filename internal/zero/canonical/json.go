/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

// Package canonical implements Matrix canonical JSON.
// It emits a deterministic byte representation: keys sorted
// lexicographically, no extra whitespace, strict integer-only numbers
// within the safe [-2^53+1, 2^53-1] range, and string escapes limited to
// those required for control characters.
package canonical

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// maxSafeInteger is the IEEE-754 safe integer bound (2^53 - 1).
const maxSafeInteger = 9007199254740991

// JSON parses a JSON byte slice and returns its canonical form.
// Rejects trailing non-whitespace tokens to enforce strict
// single-document parsing.
func JSON(input []byte) ([]byte, error) {
	var v interface{}
	dec := json.NewDecoder(bytes.NewReader(input))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	var extra interface{}
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("invalid JSON: trailing tokens")
	}
	return Marshal(v)
}

// Marshal serializes v to canonical JSON bytes.
func Marshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := writeValue(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
