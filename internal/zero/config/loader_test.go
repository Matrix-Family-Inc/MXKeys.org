/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 20 Nov 2026 23:15:00 UTC
 * Status: Created
 */

package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadStripsInlineCommentsAndFlowCollections(t *testing.T) {
	path := writeConfig(t, `
logging:
  level: info    # debug, info, warn, error
  format: text   # text, json
server:
  bind_address: "0.0.0.0"
  shutdown_timeout: 30s
  quoted: "hash # stays"
  single: 'it''s'
security:
  trusted_proxies: [127.0.0.1/32, 10.0.0.0/8]
  admin_access_token: ""
cluster:
  seeds: []
  tls: {}
  min_version: "1.3" # only "1.3" is accepted
`)
	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := GetString(m, "logging.level"); got != "info" {
		t.Errorf("logging.level = %q", got)
	}
	if got := GetString(m, "logging.format"); got != "text" {
		t.Errorf("logging.format = %q", got)
	}
	if got := GetString(m, "server.bind_address"); got != "0.0.0.0" {
		t.Errorf("bind_address = %q", got)
	}
	if got := GetString(m, "server.shutdown_timeout"); got != "30s" {
		t.Errorf("shutdown_timeout = %q", got)
	}
	if got := GetString(m, "server.quoted"); got != "hash # stays" {
		t.Errorf("quoted = %q", got)
	}
	if got := GetString(m, "server.single"); got != "it's" {
		t.Errorf("single = %q", got)
	}
	proxies := GetStringSlice(m, "security.trusted_proxies")
	if len(proxies) != 2 || proxies[0] != "127.0.0.1/32" || proxies[1] != "10.0.0.0/8" {
		t.Errorf("trusted_proxies = %v", proxies)
	}
	if GetString(m, "security.admin_access_token") != "" {
		t.Errorf("empty quoted token must stay empty")
	}
	if seeds := GetStringSlice(m, "cluster.seeds"); seeds == nil || len(seeds) != 0 {
		t.Errorf("empty flow sequence = %v", seeds)
	}
	if got := GetString(m, "cluster.min_version"); got != "1.3" {
		t.Errorf("quoted version = %q", got)
	}
	tls, ok := getPath(m, "cluster.tls").(map[string]interface{})
	if !ok || len(tls) != 0 {
		t.Errorf("empty flow mapping = %#v", getPath(m, "cluster.tls"))
	}
}

func TestLoadExampleConfig(t *testing.T) {
	path := filepath.Join("..", "..", "..", "config.example.yaml")
	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load example: %v", err)
	}
	if got := GetString(m, "logging.level"); got != "info" {
		t.Errorf("example logging.level = %q", got)
	}
	if got := GetString(m, "logging.format"); got != "text" {
		t.Errorf("example logging.format = %q", got)
	}
	if got := GetString(m, "cluster.consensus_mode"); got != "crdt" {
		t.Errorf("example consensus_mode = %q", got)
	}
	if got := GetInt(m, "keys.expired_retention_days"); got != 365 {
		t.Errorf("example expired_retention_days = %d", got)
	}
	if GetBool(m, "analytics.enabled") {
		t.Errorf("example analytics.enabled must be false")
	}
	if got := GetString(m, "cluster.tls.min_version"); got != "1.3" {
		t.Errorf("example min_version = %q", got)
	}
	if seeds := GetStringSlice(m, "cluster.seeds"); seeds == nil || len(seeds) != 0 {
		t.Errorf("example seeds = %v", seeds)
	}
	if proxies := GetStringSlice(m, "security.trusted_proxies"); proxies == nil || len(proxies) != 0 {
		t.Errorf("example trusted_proxies = %v", proxies)
	}
}

func TestLoadRejectsUnsupportedYAML(t *testing.T) {
	cases := []string{
		"a: |\n  hello\n",
		"a: {b: c}\n",
		"a: [1, [2]]\n",
		"a: \"unterminated\n",
		"list:\n  - scalar\n    nested: bad\n",
		"a: &anchor value\n",
		"a: *alias\n",
	}
	for _, body := range cases {
		path := writeConfig(t, body)
		if _, err := Load(path); err == nil {
			t.Errorf("expected error for %q", body)
		}
	}
}

func TestLoadCommentDoesNotSwallowSibling(t *testing.T) {
	path := writeConfig(t, `
cluster:
  node_id: # empty
  bind_port: 7946
`)
	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if GetInt(m, "cluster.bind_port") != 7946 {
		t.Fatalf("sibling was swallowed: %#v", m["cluster"])
	}
}

func TestParseValueKeepsPlainColons(t *testing.T) {
	value, err := parseValue("ed25519:auto")
	if err != nil {
		t.Fatal(err)
	}
	if value != "ed25519:auto" {
		t.Fatalf("got %#v", value)
	}
	if _, err := parseValue("info"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(mustString(t, "\"a\\nb\""), "\\") {
		t.Fatal("double-quoted newline was not unescaped")
	}
}

func mustString(t *testing.T, raw string) string {
	t.Helper()
	value, err := parseValue(raw)
	if err != nil {
		t.Fatal(err)
	}
	text, ok := value.(string)
	if !ok {
		t.Fatalf("got %#v", value)
	}
	return text
}
