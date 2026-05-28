/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

package canonical

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

// writeInteger emits an int64 if it fits in the safe integer range.
func writeInteger(buf *bytes.Buffer, n int64) error {
	if n < -maxSafeInteger || n > maxSafeInteger {
		return fmt.Errorf("integer out of canonical range")
	}
	buf.WriteString(strconv.FormatInt(n, 10))
	return nil
}

// writeFloatNumber accepts only integer-valued floats inside the safe range.
// NaN, Inf, and non-integer values are rejected.
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

// writeJSONNumber handles json.Number (typed as string). Fractional and
// exponent forms are rejected; integer range enforced via big.Int.
func writeJSONNumber(buf *bytes.Buffer, n json.Number) error {
	s := n.String()
	if strings.ContainsAny(s, ".eE") {
		return fmt.Errorf("non-integer numbers are not allowed in canonical JSON")
	}

	i := new(big.Int)
	if _, ok := i.SetString(s, 10); !ok {
		return fmt.Errorf("invalid numeric value")
	}

	maxBig := big.NewInt(maxSafeInteger)
	minBig := big.NewInt(-maxSafeInteger)
	if i.Cmp(maxBig) > 0 || i.Cmp(minBig) < 0 {
		return fmt.Errorf("integer out of canonical range")
	}

	buf.WriteString(i.String())
	return nil
}
