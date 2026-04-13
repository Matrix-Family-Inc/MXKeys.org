/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Wed Apr 08 2026 UTC
 * Status: Created
 */

package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func decodeStrictJSON(r io.Reader, dst interface{}, maxDepth int) error {
	body, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	if err := validateJSONDepth(body, maxDepth); err != nil {
		return err
	}

	dec := json.NewDecoder(bytes.NewReader(body))
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var trailing interface{}
	if err := dec.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("trailing JSON data")
	}
	return nil
}

const defaultMaxJSONDepth = 64

func validateJSONDepth(body []byte, maxDepth int) error {
	if maxDepth <= 0 {
		maxDepth = defaultMaxJSONDepth
	}

	dec := json.NewDecoder(bytes.NewReader(body))
	depth := 0
	maxSeen := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		delim, ok := tok.(json.Delim)
		if !ok {
			continue
		}

		switch delim {
		case '{', '[':
			depth++
			if depth > maxSeen {
				maxSeen = depth
			}
			if maxSeen > maxDepth {
				return fmt.Errorf("JSON exceeds maximum depth of %d", maxDepth)
			}
		case '}', ']':
			if depth > 0 {
				depth--
			}
		}
	}

	return nil
}
