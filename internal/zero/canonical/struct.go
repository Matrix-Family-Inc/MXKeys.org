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
	"reflect"
	"strings"
)

// structToMap converts a struct reflection value to a map[string]interface{}
// using JSON struct tag conventions. Unexported fields are skipped. The
// "omitempty" tag option is honored.
func structToMap(rv reflect.Value) (map[string]interface{}, error) {
	t := rv.Type()
	result := make(map[string]interface{}, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		name := field.Name
		if tag := field.Tag.Get("json"); tag != "" {
			parts := strings.Split(tag, ",")
			if parts[0] == "-" {
				continue
			}
			if parts[0] != "" {
				name = parts[0]
			}
			if len(parts) > 1 && parts[1] == "omitempty" {
				if isEmptyValue(rv.Field(i)) {
					continue
				}
			}
		}

		result[name] = rv.Field(i).Interface()
	}

	return result, nil
}

// isEmptyValue reports whether v is the zero value of its type, for purposes
// of honoring the json:"omitempty" tag.
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}
