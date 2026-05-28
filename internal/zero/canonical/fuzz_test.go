package canonical

import (
	"bytes"
	stdjson "encoding/json"
	"testing"
)

// FuzzJSON exercises the canonical JSON parser + writer with arbitrary
// byte inputs. Invariants checked:
//
//  1. JSON(data) must never panic. Malformed input returns an error.
//  2. When JSON(data) succeeds, the output is valid standard JSON (the
//     stdlib parser accepts it).
//  3. Idempotence: JSON(JSON(data)) == JSON(data) whenever JSON(data)
//     succeeds. The canonical form is a fixed point of the transformation.
func FuzzJSON(f *testing.F) {
	// Seeds cover the full grammar: scalars, nested objects/arrays, edge
	// cases in integers and strings. The fuzzer mutates these bytes.
	seeds := []string{
		`null`,
		`true`,
		`false`,
		`0`,
		`-1`,
		`42`,
		`9007199254740991`,
		`"hello"`,
		`""`,
		`"with \"escapes\" and \n newlines"`,
		`[]`,
		`{}`,
		`[1,2,3]`,
		`{"a":1,"b":2}`,
		`{"z":{"y":[1,2,3],"x":true},"a":null}`,
		`{"unicode":"\u00e9\u4e2d\u6587"}`,
		`{"nested":[{"k":"v"},[1,[2,[3]]]]}`,
		// Whitespace variants: canonical form must strip all of them.
		"  {   \"a\"   :   1   ,   \"b\"   :   2   }  ",
		`
		{
			"formatted": true
		}`,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		out, err := JSON(data)
		if err != nil {
			// Malformed input is expected; nothing to check. What matters
			// is that we did not panic.
			return
		}

		// Result must be valid standard JSON.
		var anyV interface{}
		if uerr := stdjson.Unmarshal(out, &anyV); uerr != nil {
			t.Fatalf("canonical output is not valid JSON: %v\ninput: %q\noutput: %q", uerr, data, out)
		}

		// Idempotence: running canonicalization twice produces the same
		// bytes. This is the contract Matrix signature verification relies
		// on: two independent nodes must agree on the byte representation.
		again, err := JSON(out)
		if err != nil {
			t.Fatalf("JSON(JSON(data)) failed: %v\nfirst output: %q", err, out)
		}
		if !bytes.Equal(out, again) {
			t.Fatalf("canonical form is not a fixed point:\nfirst : %q\nsecond: %q", out, again)
		}
	})
}

// FuzzMarshalRoundTrip verifies that Marshal(Unmarshal(x)) produces the same
// canonical bytes for values that round-trip through JSON(). Guards against
// divergence between the write path (Marshal) and the parse path (JSON).
func FuzzMarshalRoundTrip(f *testing.F) {
	f.Add([]byte(`{"a":1,"b":["x","y"],"c":{"n":42}}`))
	f.Add([]byte(`[null,true,false,0,1,-9007199254740991]`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`"plain string"`))

	f.Fuzz(func(t *testing.T, data []byte) {
		first, err := JSON(data)
		if err != nil {
			return
		}

		// Decode via stdlib into a generic value, then re-marshal via
		// canonical.Marshal. Must produce the same bytes as JSON did.
		var v interface{}
		if err := stdjson.Unmarshal(first, &v); err != nil {
			t.Fatalf("stdlib cannot parse canonical output: %v\noutput: %q", err, first)
		}

		second, err := Marshal(v)
		if err != nil {
			t.Fatalf("Marshal failed on value decoded from canonical output: %v", err)
		}
		if !bytes.Equal(first, second) {
			t.Fatalf("JSON and Marshal disagree:\nJSON:    %q\nMarshal: %q", first, second)
		}
	})
}
