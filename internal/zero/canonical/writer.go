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
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
)

// writeValue dispatches on dynamic type and emits canonical bytes.
// Unsupported floats (NaN, Inf, fractional) are rejected.
func writeValue(buf *bytes.Buffer, v interface{}) error {
	switch val := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if val {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case float64:
		return writeFloatNumber(buf, val)
	case json.Number:
		return writeJSONNumber(buf, val)
	case int:
		return writeInteger(buf, int64(val))
	case int8:
		return writeInteger(buf, int64(val))
	case int16:
		return writeInteger(buf, int64(val))
	case int32:
		return writeInteger(buf, int64(val))
	case int64:
		return writeInteger(buf, val)
	case uint:
		if val > uint(maxSafeInteger) {
			return fmt.Errorf("integer out of canonical range")
		}
		buf.WriteString(strconv.FormatUint(uint64(val), 10))
	case uint8:
		buf.WriteString(strconv.FormatUint(uint64(val), 10))
	case uint16:
		buf.WriteString(strconv.FormatUint(uint64(val), 10))
	case uint32:
		buf.WriteString(strconv.FormatUint(uint64(val), 10))
	case uint64:
		if val > uint64(maxSafeInteger) {
			return fmt.Errorf("integer out of canonical range")
		}
		buf.WriteString(strconv.FormatUint(val, 10))
	case string:
		writeString(buf, val)
	case []interface{}:
		return writeArray(buf, val)
	case map[string]interface{}:
		return writeObject(buf, val)
	default:
		return writeReflected(buf, val)
	}
	return nil
}

// writeReflected handles typed maps, structs, pointers, and typed slices.
// Keys of maps must be strings. Structs are serialized via structToMap.
func writeReflected(buf *bytes.Buffer, val interface{}) error {
	rv := reflect.ValueOf(val)
	switch rv.Kind() {
	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("unsupported map key type %s for canonical JSON marshaling", rv.Type().Key())
		}
		converted := make(map[string]interface{}, rv.Len())
		for _, k := range rv.MapKeys() {
			converted[k.String()] = rv.MapIndex(k).Interface()
		}
		return writeObject(buf, converted)
	case reflect.Struct:
		converted, err := structToMap(rv)
		if err != nil {
			return err
		}
		return writeObject(buf, converted)
	case reflect.Ptr:
		if rv.IsNil() {
			buf.WriteString("null")
			return nil
		}
		return writeValue(buf, rv.Elem().Interface())
	case reflect.Slice:
		arr := make([]interface{}, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			arr[i] = rv.Index(i).Interface()
		}
		return writeArray(buf, arr)
	default:
		return fmt.Errorf("unsupported type %T for canonical JSON marshaling", val)
	}
}

// writeArray emits a JSON array with no separator whitespace.
func writeArray(buf *bytes.Buffer, arr []interface{}) error {
	buf.WriteByte('[')
	for i, v := range arr {
		if i > 0 {
			buf.WriteByte(',')
		}
		if err := writeValue(buf, v); err != nil {
			return err
		}
	}
	buf.WriteByte(']')
	return nil
}

// writeObject emits a JSON object with keys sorted lexicographically.
func writeObject(buf *bytes.Buffer, obj map[string]interface{}) error {
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	buf.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		writeString(buf, k)
		buf.WriteByte(':')
		if err := writeValue(buf, obj[k]); err != nil {
			return err
		}
	}
	buf.WriteByte('}')
	return nil
}
