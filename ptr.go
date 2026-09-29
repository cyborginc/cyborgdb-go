// ptr.go — helpers for setting optional (pointer) fields.

package cyborgdb

// Optional fields in this SDK are pointers, where nil means "use the service
// default". These helpers let a literal value be set inline:
//
//	params := cyborgdb.QueryParams{
//	    QueryVector: vec,
//	    NProbes:     cyborgdb.Int32(8),
//	    Greedy:      cyborgdb.Bool(true),
//	}

// Bool returns a pointer to v.
func Bool(v bool) *bool { return &v }

// Int32 returns a pointer to v.
func Int32(v int32) *int32 { return &v }

// String returns a pointer to v.
func String(v string) *string { return &v }

// Float64 returns a pointer to v.
func Float64(v float64) *float64 { return &v }
