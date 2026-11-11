// datetimes.go — epoch-millisecond coercion for metadata and filter values.

package cyborgdb

import "time"

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
		out := make(map[string]interface{}, len(v))
		for k, val := range v {
			out[k] = coerceDatetimes(val)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, val := range v {
			out[i] = coerceDatetimes(val)
		}
		return out
	default:
		return value
	}
}

// coerceMapDatetimes applies coerceDatetimes to a metadata or filter map,
// preserving nil.
func coerceMapDatetimes(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	return coerceDatetimes(m).(map[string]interface{})
}
