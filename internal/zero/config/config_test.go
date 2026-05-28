package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestLoadScalarsAndNesting(t *testing.T) {
	path := writeConfig(t, `server:
  name: mxkeys.test
  port: 8448
  bind_address: "0.0.0.0"

logging:
  level: debug
  format: json

flags:
  enabled: true
  disabled: false
`)
	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got := GetString(m, "server.name"); got != "mxkeys.test" {
		t.Errorf("server.name = %q", got)
	}
	if got := GetInt(m, "server.port"); got != 8448 {
		t.Errorf("server.port = %d", got)
	}
	if got := GetString(m, "server.bind_address"); got != "0.0.0.0" {
		t.Errorf("server.bind_address = %q", got)
	}
	if got := GetString(m, "logging.format"); got != "json" {
		t.Errorf("logging.format = %q", got)
	}
	if got := GetBool(m, "flags.enabled"); !got {
		t.Errorf("flags.enabled must be true")
	}
	if got := GetBool(m, "flags.disabled"); got {
		t.Errorf("flags.disabled must be false")
	}
	if !Has(m, "server.name") {
		t.Errorf("Has must report existence for a known path")
	}
	if Has(m, "server.missing") {
		t.Errorf("Has must report absence for a missing path")
	}
}

func TestLoadStringSlice(t *testing.T) {
	path := writeConfig(t, `trusted_servers:
  fallback:
    - matrix.org
    - example.org
`)
	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := GetStringSlice(m, "trusted_servers.fallback")
	if len(got) != 2 || got[0] != "matrix.org" || got[1] != "example.org" {
		t.Errorf("unexpected slice: %v", got)
	}
}

func TestLoadMappingListItems(t *testing.T) {
	path := writeConfig(t, `trusted_notaries:
  - server_name: matrix.org
    key_id: ed25519:auto
    public_key: cHViMQ
  - server_name: example.org
    key_id: ed25519:ex
    public_key: cHViMg
`)
	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	notaries := GetMapSlice(m, "trusted_notaries")
	if len(notaries) != 2 {
		t.Fatalf("expected 2 notaries, got %d", len(notaries))
	}
	if notaries[0]["server_name"] != "matrix.org" {
		t.Errorf("first server_name: %v", notaries[0]["server_name"])
	}
	if notaries[1]["key_id"] != "ed25519:ex" {
		t.Errorf("second key_id: %v", notaries[1]["key_id"])
	}
}

func TestParseValueFromEnv(t *testing.T) {
	path := writeConfig(t, `server:
  name: file-name
`)
	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	t.Setenv("MXKEYS_SERVER_NAME", "env-name")
	t.Setenv("MXKEYS_SERVER_PORT", "9999")
	t.Setenv("MXKEYS_FLAGS_ON", "true")
	WithEnvOverride(m, "MXKEYS")

	if got := GetString(m, "server.name"); got != "env-name" {
		t.Errorf("env override server.name = %q", got)
	}
	if got := GetInt(m, "server.port"); got != 9999 {
		t.Errorf("env override server.port = %d", got)
	}
	if got := GetBool(m, "flags.on"); !got {
		t.Errorf("env override flags.on must parse to true")
	}
}

func TestValidateReportsMissingPaths(t *testing.T) {
	m := map[string]interface{}{
		"server": map[string]interface{}{
			"name": "notary.example.org",
		},
	}
	if err := Validate(m, []string{"server.name"}); err != nil {
		t.Fatalf("valid path must pass: %v", err)
	}
	err := Validate(m, []string{"server.name", "server.port"})
	if err == nil {
		t.Fatal("expected missing-path error")
	}
}

func TestGetStringSliceEmptyPath(t *testing.T) {
	m := map[string]interface{}{}
	if got := GetStringSlice(m, "missing"); got != nil {
		t.Errorf("missing path must return nil slice, got %v", got)
	}
}

func TestGetFloatAcceptsMultipleNumericShapes(t *testing.T) {
	m := map[string]interface{}{
		"a": 1.5,
		"b": int(2),
		"c": int64(3),
		"d": "4.5",
		"e": "not-a-number",
	}
	if got := GetFloat(m, "a"); got != 1.5 {
		t.Errorf("float a = %v", got)
	}
	if got := GetFloat(m, "b"); got != 2 {
		t.Errorf("int->float b = %v", got)
	}
	if got := GetFloat(m, "c"); got != 3 {
		t.Errorf("int64->float c = %v", got)
	}
	if got := GetFloat(m, "d"); got != 4.5 {
		t.Errorf("string->float d = %v", got)
	}
	if got := GetFloat(m, "e"); got != 0 {
		t.Errorf("unparseable string must yield 0, got %v", got)
	}
}
