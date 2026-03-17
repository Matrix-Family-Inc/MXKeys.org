package canonical

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestCanonicalJSONDeterministic(t *testing.T) {
	input := []byte(`{"b":"2","a":"1","c":"3"}`)

	result1, err := JSON(input)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	result2, err := JSON(input)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if !bytes.Equal(result1, result2) {
		t.Errorf("canonical JSON not deterministic:\n  1: %s\n  2: %s", result1, result2)
	}
}

func TestCanonicalJSONKeyOrder(t *testing.T) {
	input := []byte(`{"z":"last","a":"first","m":"middle"}`)

	result, err := JSON(input)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	expected := `{"a":"first","m":"middle","z":"last"}`
	if string(result) != expected {
		t.Errorf("keys not sorted:\n  got:      %s\n  expected: %s", result, expected)
	}
}

func TestCanonicalJSONNestedObjects(t *testing.T) {
	input := []byte(`{"outer":{"z":"1","a":"2"},"inner":{"b":"3","a":"4"}}`)

	result, err := JSON(input)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	expected := `{"inner":{"a":"4","b":"3"},"outer":{"a":"2","z":"1"}}`
	if string(result) != expected {
		t.Errorf("nested keys not sorted:\n  got:      %s\n  expected: %s", result, expected)
	}
}

func TestCanonicalJSONNoWhitespace(t *testing.T) {
	input := []byte(`{
		"key1": "value1",
		"key2": "value2"
	}`)

	result, err := JSON(input)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	if bytes.Contains(result, []byte(" ")) || bytes.Contains(result, []byte("\n")) || bytes.Contains(result, []byte("\t")) {
		t.Errorf("canonical JSON should not contain whitespace: %s", result)
	}
}

func TestCanonicalJSONArrays(t *testing.T) {
	input := []byte(`{"items":[3,1,2],"name":"test"}`)

	result, err := JSON(input)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	expected := `{"items":[3,1,2],"name":"test"}`
	if string(result) != expected {
		t.Errorf("array order should be preserved:\n  got:      %s\n  expected: %s", result, expected)
	}
}

func TestCanonicalJSONSpecialStrings(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`{"key":"hello world"}`, `{"key":"hello world"}`},
		{`{"key":"hello\nworld"}`, `{"key":"hello\nworld"}`},
		{`{"key":"hello\tworld"}`, `{"key":"hello\tworld"}`},
		{`{"key":"quotes\"here"}`, `{"key":"quotes\"here"}`},
		{`{"key":"backslash\\here"}`, `{"key":"backslash\\here"}`},
	}

	for _, tt := range tests {
		result, err := JSON([]byte(tt.input))
		if err != nil {
			t.Errorf("failed for %s: %v", tt.input, err)
			continue
		}

		if string(result) != tt.expected {
			t.Errorf("special string handling:\n  input:    %s\n  got:      %s\n  expected: %s",
				tt.input, result, tt.expected)
		}
	}
}

func TestCanonicalJSONNumbers(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`{"num":123}`, `{"num":123}`},
		{`{"num":-456}`, `{"num":-456}`},
		{`{"num":0}`, `{"num":0}`},
		{`{"num":9007199254740991}`, `{"num":9007199254740991}`},
	}

	for _, tt := range tests {
		result, err := JSON([]byte(tt.input))
		if err != nil {
			t.Errorf("failed for %s: %v", tt.input, err)
			continue
		}

		if string(result) != tt.expected {
			t.Errorf("number handling:\n  input:    %s\n  got:      %s\n  expected: %s",
				tt.input, result, tt.expected)
		}
	}
}

func TestCanonicalJSONRejectsFloatNumbers(t *testing.T) {
	input := []byte(`{"num":1.5}`)
	if _, err := JSON(input); err == nil {
		t.Fatal("expected float number rejection")
	}
}

func TestCanonicalJSONRejectsOutOfRangeIntegers(t *testing.T) {
	input := []byte(`{"num":9007199254740992}`)
	if _, err := JSON(input); err == nil {
		t.Fatal("expected out-of-range integer rejection")
	}
}

func TestCanonicalJSONPreservesLargeIntegerPrecision(t *testing.T) {
	input := []byte(`{"valid_until_ts":1234567890123456}`)

	result, err := JSON(input)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	expected := `{"valid_until_ts":1234567890123456}`
	if string(result) != expected {
		t.Fatalf("precision mismatch:\n got: %s\nwant: %s", result, expected)
	}
}

func TestCanonicalJSONBooleans(t *testing.T) {
	input := []byte(`{"false_val":false,"true_val":true}`)

	result, err := JSON(input)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	expected := `{"false_val":false,"true_val":true}`
	if string(result) != expected {
		t.Errorf("boolean handling:\n  got:      %s\n  expected: %s", result, expected)
	}
}

func TestCanonicalJSONNull(t *testing.T) {
	input := []byte(`{"null_val":null,"other":"value"}`)

	result, err := JSON(input)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	expected := `{"null_val":null,"other":"value"}`
	if string(result) != expected {
		t.Errorf("null handling:\n  got:      %s\n  expected: %s", result, expected)
	}
}

func TestCanonicalJSONEmptyObject(t *testing.T) {
	input := []byte(`{}`)

	result, err := JSON(input)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	if string(result) != `{}` {
		t.Errorf("empty object: got %s", result)
	}
}

func TestCanonicalJSONEmptyArray(t *testing.T) {
	input := []byte(`{"arr":[]}`)

	result, err := JSON(input)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	if string(result) != `{"arr":[]}` {
		t.Errorf("empty array: got %s", result)
	}
}

func TestMarshalMap(t *testing.T) {
	input := map[string]interface{}{
		"z_key": "last",
		"a_key": "first",
		"m_key": "middle",
	}

	result, err := Marshal(input)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	expected := `{"a_key":"first","m_key":"middle","z_key":"last"}`
	if string(result) != expected {
		t.Errorf("Marshal map:\n  got:      %s\n  expected: %s", result, expected)
	}
}

func TestInvalidJSON(t *testing.T) {
	inputs := []string{
		`{invalid}`,
		`{"key":}`,
		`{"key"`,
		`not json`,
	}

	for _, input := range inputs {
		_, err := JSON([]byte(input))
		if err == nil {
			t.Errorf("invalid JSON should fail: %s", input)
		}
	}
}

func TestCanonicalJSONMatrixExample(t *testing.T) {
	input := []byte(`{
		"server_name": "example.com",
		"valid_until_ts": 1234567890123,
		"verify_keys": {
			"ed25519:key1": {"key": "base64data"}
		},
		"old_verify_keys": {}
	}`)

	result, err := JSON(input)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if bytes.Contains(result, []byte("\n")) || bytes.Contains(result, []byte(" ")) {
		t.Error("result contains whitespace")
	}
}
