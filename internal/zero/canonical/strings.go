/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Mon 22 Jun 2026 00:50:40 UTC
 * Status: Updated
 */

package canonical

import "bytes"

// writeString emits a JSON string. Only escapes required for control
// characters or structural characters are used; other runes are emitted
// as-is (valid UTF-8 bytes).
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

// hexDigit returns the lowercase hex digit byte for n in [0, 15].
func hexDigit(n rune) byte {
	const digits = "0123456789abcdef"
	if n < 0 || n > 15 {
		return '0'
	}
	return digits[int(n)]
}
