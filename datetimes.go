// datetimes.go — epoch-millisecond coercion for metadata and filter values.

package cyborgdb

import (
	"encoding"
	"encoding/json"
	"reflect"
	"time"
)

// coerceDatetimes recursively replaces time.Time values with integer epoch
// milliseconds, the contract's datetime representation (encoding/json would
// otherwise send RFC 3339 strings, which break range filters). Maps and
// slices are rebuilt, not mutated; other types pass through unchanged.
//
// The zero time.Time{} is not special-cased: it becomes -62135596800000
// (year 1), a valid timestamp that sorts before every real date. Callers who
// use the zero value to mean "unset" should omit the field instead. This is
// documented in the package doc rather than rejected, to avoid failing
// requests that worked before coercion existed.
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
// ([]time.Time, []map[string]interface{}, pointers to either, …). Containers that cannot hold a
// time.Time pass through unchanged, as do typed nils and values with custom
// serialization (json.Marshaler / encoding.TextMarshaler), whose MarshalJSON
// or MarshalText output a rebuild would discard.
func coerceTypedContainer(value interface{}) interface{} {
	switch value.(type) {
	case json.Marshaler, encoding.TextMarshaler:
		return value
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Pointer:
		// encoding/json marshals through pointers, so dereferencing keeps the
		// wire shape while reaching any time.Time behind the pointer.
		if rv.IsNil() || !canContainTime(rv.Type().Elem(), 0) {
			return value
		}
		return coerceDatetimes(rv.Elem().Interface())
	case reflect.Slice, reflect.Array:
		if (rv.Kind() == reflect.Slice && rv.IsNil()) || !canContainTime(rv.Type().Elem(), 0) {
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
// preserving nil (the map case of coerceDatetimes returns a nil map as-is).
// Maps with no time.Time anywhere inside are returned as-is, so the common
// case costs one read-only walk instead of a deep copy per upserted item.
func coerceMapDatetimes(m map[string]interface{}) map[string]interface{} {
	if !containsTime(m) {
		return m
	}
	out, ok := coerceDatetimes(m).(map[string]interface{})
	if !ok {
		return m
	}
	return out
}

// containsTime reports whether coerceDatetimes would convert anything in
// value. It must follow exactly the same paths as coerceDatetimes and
// coerceTypedContainer: a time.Time that one reaches and the other skips
// would either be missed or copied for nothing. The time cases come before
// the marshaler case because time.Time itself implements json.Marshaler.
func containsTime(value interface{}) bool {
	switch v := value.(type) {
	case time.Time:
		return true
	case *time.Time:
		return v != nil
	case map[string]interface{}:
		for _, val := range v {
			if containsTime(val) {
				return true
			}
		}
		return false
	case []interface{}:
		for _, val := range v {
			if containsTime(val) {
				return true
			}
		}
		return false
	case json.Marshaler, encoding.TextMarshaler:
		return false
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Pointer:
		return !rv.IsNil() && canContainTime(rv.Type().Elem(), 0) && containsTime(rv.Elem().Interface())
	case reflect.Slice, reflect.Array:
		if !canContainTime(rv.Type().Elem(), 0) {
			return false
		}
		for i := 0; i < rv.Len(); i++ {
			if containsTime(rv.Index(i).Interface()) {
				return true
			}
		}
		return false
	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String || !canContainTime(rv.Type().Elem(), 0) {
			return false
		}
		for iter := rv.MapRange(); iter.Next(); {
			if containsTime(iter.Value().Interface()) {
				return true
			}
		}
		return false
	default:
		return false
	}
}
