/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Fri 03 Apr 2026 UTC
 * Status: Created
 */

package config

import (
	"path/filepath"
	"strings"

	zeroconfig "mxkeys/internal/zero/config"
)

func parseTrustedNotariesFromYAML(configPath string) []TrustedNotary {
	if configPath == "" {
		return nil
	}
	ext := strings.ToLower(filepath.Ext(configPath))
	if ext != ".yaml" && ext != ".yml" {
		return nil
	}
	m, err := zeroconfig.Load(configPath)
	if err != nil {
		return nil
	}
	return parseTrustedNotariesFromMap(m)
}

func parseTrustedNotariesFromMap(m map[string]interface{}) []TrustedNotary {
	items := zeroconfig.GetMapSlice(m, "trusted_notaries")
	if len(items) == 0 {
		return nil
	}

	result := make([]TrustedNotary, 0, len(items))
	for _, item := range items {
		entry := TrustedNotary{
			ServerName: strings.TrimSpace(zeroconfig.GetString(item, "server_name")),
			KeyID:      strings.TrimSpace(zeroconfig.GetString(item, "key_id")),
			PublicKey:  strings.TrimSpace(zeroconfig.GetString(item, "public_key")),
		}
		if entry.ServerName == "" || entry.KeyID == "" || entry.PublicKey == "" {
			continue
		}
		result = append(result, entry)
	}
	return result
}
