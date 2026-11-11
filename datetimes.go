// datetimes.go — epoch-millisecond coercion for metadata and filter values.

package cyborgdb

import (
	"reflect"
	"time"
)

// coerceDatetimes recursively replaces time.Time values with integer epoch
// milliseconds, the contract's datetime representation (encoding/json would
// otherwise send RFC 3339 strings, which break range filters). Maps and
// slices are rebuilt, not mutated; other types pass through unchanged.
func coerceDatetimes(value interface{}) interface{} {
	switch v := value.(type) {
	case time.Time:
		return v.UnixMilli()
	case *time.Time:
		if v == nil {
			return v
		}
		return v.UnixMilli()
	case map[string]interface{}:
		if v == nil {
			return v
		}
		out := make(map[string]interface{}, len(v))
		for k, val := range v {
			out[k] = coerceDatetimes(val)
		}
		return out
	case []interface{}:
		if v == nil {
			return v
		}
		out := make([]interface{}, len(v))
		for i, val := range v {
			out[i] = coerceDatetimes(val)
		}
		return out
	}
	return coerceTypedContainer(value)
}

// coerceTypedContainer rebuilds typed containers the type switch misses
// ([]time.Time, []map[string]interface{}, …). Containers that cannot hold a
// time.Time pass through unchanged, as do typed nils.
func coerceTypedContainer(value interface{}) interface{} {
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Slice:
		if rv.IsNil() || !canContainTime(rv.Type().Elem(), 0) {
			return value
		}
		out := make([]interface{}, rv.Len())
		for i := range out {
			out[i] = coerceDatetimes(rv.Index(i).Interface())
		}
		return out
	case reflect.Array:
		if !canContainTime(rv.Type().Elem(), 0) {
			return value
		}
		out := make([]interface{}, rv.Len())
		for i := range out {
			out[i] = coerceDatetimes(rv.Index(i).Interface())
		}
		return out
	case reflect.Map:
		if rv.IsNil() || rv.Type().Key().Kind() != reflect.String || !canContainTime(rv.Type().Elem(), 0) {
			return value
		}
		out := make(map[string]interface{}, rv.Len())
		for iter := rv.MapRange(); iter.Next(); {
			out[iter.Key().String()] = coerceDatetimes(iter.Value().Interface())
		}
		return out
	default:
		return value
	}
}

// maxTypeDepth bounds canContainTime's descent so self-referential types
// (type T []T) cannot recurse forever.
const maxTypeDepth = 10

var timeType = reflect.TypeOf(time.Time{})

// canContainTime reports whether a value of type t could hold a time.Time
// somewhere inside it. Struct fields are not inspected: arbitrary structs
// marshal via their own json tags and are out of scope for coercion.
func canContainTime(t reflect.Type, depth int) bool {
	if t == timeType {
		return true
	}
	if depth >= maxTypeDepth {
		return false
	}
	switch t.Kind() {
	case reflect.Interface:
		return true
	case reflect.Pointer, reflect.Slice, reflect.Array:
		return canContainTime(t.Elem(), depth+1)
	case reflect.Map:
		return t.Key().Kind() == reflect.String && canContainTime(t.Elem(), depth+1)
	default:
		return false
	}
}

// coerceMapDatetimes applies coerceDatetimes to a metadata or filter map,
// preserving nil.
func coerceMapDatetimes(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	out := make(map[string]interface{}, len(m))
	for k, val := range m {
		out[k] = coerceDatetimes(val)
	}
	return out
}
