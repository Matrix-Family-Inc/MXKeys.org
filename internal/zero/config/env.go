/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

package config

import (
	"fmt"
	"os"
	"strings"
)

// WithEnvOverride overrides config values with environment variables.
// Env format: PREFIX_SECTION_KEY (e.g., MXKEYS_SERVER_PORT becomes server.port).
func WithEnvOverride(m map[string]interface{}, prefix string) {
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]

		if !strings.HasPrefix(key, prefix+"_") {
			continue
		}

		path := strings.ToLower(strings.TrimPrefix(key, prefix+"_"))
		path = strings.ReplaceAll(path, "_", ".")

		setPath(m, path, parseValue(value))
	}
}

// Validate reports missing required paths.
func Validate(m map[string]interface{}, required []string) error {
	for _, path := range required {
		if getPath(m, path) == nil {
			return fmt.Errorf("missing required config: %s", path)
		}
	}
	return nil
}
