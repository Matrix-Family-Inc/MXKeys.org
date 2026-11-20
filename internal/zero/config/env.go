/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 20 Nov 2026 23:15:00 UTC
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

		parsed, err := parseValue(value)
		if err != nil {
			// Environment values are raw strings, not a YAML document.
			// A value this subset rejects is still applied verbatim.
			setPath(m, path, value)
			continue
		}
		setPath(m, path, parsed)
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
