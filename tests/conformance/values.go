package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"strconv"
	"time"
)

// params is an op's params object; values are decoded on use, so an absent one stays absent.
type params map[string]json.RawMessage

func (p params) opt(k string) (json.RawMessage, bool) {
	v, ok := p[k]
	return v, ok
}

func (p params) need(k string) json.RawMessage {
	v, ok := p[k]
	if !ok {
		failf(codeBadParams, "missing %s", k)
	}
	return v
}

func isNull(v json.RawMessage) bool { return string(bytes.TrimSpace(v)) == "null" }

// tagged returns the payload of a one-key {"$tag": payload} object.
func tagged(v json.RawMessage, tag string) (json.RawMessage, bool) {
	var o map[string]json.RawMessage
	if json.Unmarshal(v, &o) != nil || len(o) != 1 {
		return nil, false
	}
	x, ok := o[tag]
	return x, ok
}

func str(v json.RawMessage, what string) string {
	var s string
	if json.Unmarshal(v, &s) != nil {
		failf(codeUnsupported, "%s must be a string to be expressible", what)
	}
	return s
}

func flag(v json.RawMessage, what string) bool {
	var b bool
	if json.Unmarshal(v, &b) != nil {
		failf(codeUnsupported, "%s must be a bool to be expressible", what)
	}
	return b
}

// count is a non-negative integer that fits a Go int.
func count(v json.RawMessage, what string) int {
	var n uint64
	if json.Unmarshal(v, &n) != nil || n > math.MaxInt {
		failf(codeUnsupported, "%s must be a non-negative integer to be expressible", what)
	}
	return int(n)
}

func intOf(v json.RawMessage, what string) int64 {
	var n int64
	if json.Unmarshal(v, &n) == nil {
		return n
	}
	if x, ok := tagged(v, "$i64"); ok {
		n, err := strconv.ParseInt(str(x, "$i64"), 10, 64)
		if err != nil {
			failf(codeBadParams, "invalid $i64 for %s", what)
		}
		return n
	}
	failf(codeUnsupported, "%s must be an integer to be expressible", what)
	return 0
}

// int32Of is any integer an int32 holds; negative ones too, so the SDK's handling of them is under test.
func int32Of(v json.RawMessage, what string) int32 {
	n := intOf(v, what)
	if n < math.MinInt32 || n > math.MaxInt32 {
		failf(codeUnsupported, "%s %d doesn't fit the SDK's int32", what, n)
	}
	return int32(n)
}

func ptr[T any](v T) *T { return &v }

func num(v json.RawMessage, what string) float64 {
	var f float64
	if json.Unmarshal(v, &f) == nil {
		return f
	}
	if x, ok := tagged(v, "$float"); ok {
		switch str(x, "$float") {
		case "nan":
			return math.NaN()
		case "inf":
			return math.Inf(1)
		case "-inf":
			return math.Inf(-1)
		}
		failf(codeBadParams, "invalid $float for %s", what)
	}
	failf(codeUnsupported, "%s must be a number to be expressible", what)
	return 0
}

func bytesOf(v json.RawMessage, what string) []byte {
	x, ok := tagged(v, "$bytes")
	if !ok {
		failf(codeBadParams, "%s: expected $bytes", what)
	}
	b, err := base64.StdEncoding.DecodeString(str(x, "$bytes"))
	if err != nil {
		failf(codeBadParams, "invalid base64 in $bytes")
	}
	return b
}

// stringsOf is a list of strings, or $range [lo, hi) as decimal ids. Never nil.
func stringsOf(v json.RawMessage, what string) []string {
	if r, ok := tagged(v, "$range"); ok {
		var bounds []json.RawMessage
		if json.Unmarshal(r, &bounds) != nil || len(bounds) != 2 {
			failf(codeBadParams, "$range must be [lo, hi]")
		}
		lo, hi := intOf(bounds[0], "$range"), intOf(bounds[1], "$range")
		out := []string{}
		for i := lo; i < hi; i++ {
			out = append(out, strconv.FormatInt(i, 10))
		}
		return out
	}
	var items []json.RawMessage
	if json.Unmarshal(v, &items) != nil {
		failf(codeUnsupported, "%s must be a list of strings", what)
	}
	out := make([]string, len(items))
	for i, e := range items {
		out[i] = str(e, what)
	}
	return out
}

// floats is a list of numbers. Never nil.
func floats(v json.RawMessage, what string) []float32 {
	var items []json.RawMessage
	if json.Unmarshal(v, &items) != nil {
		failf(codeUnsupported, "%s must be a list of numbers", what)
	}
	out := make([]float32, len(items))
	for i, e := range items {
		out[i] = float32(num(e, what))
	}
	return out
}

// matrixRank is a $tensor shape's length when it holds rows.
const matrixRank = 2

// vectorsOf decodes a $f32 vector or rows, or a $tensor file reference. nested reports whether
// the spec sent rows (a list of vectors, or a 2-D tensor): the SDK has a separate field for each.
// Rows of different lengths stay as sent; the SDK's [][]float32 can carry them.
func vectorsOf(v json.RawMessage) (rows [][]float32, nested bool) {
	if x, ok := tagged(v, "$f32"); ok {
		var items []json.RawMessage
		if json.Unmarshal(x, &items) != nil {
			failf(codeBadParams, "$f32 must be a list")
		}
		if len(items) > 0 && bytes.HasPrefix(bytes.TrimSpace(items[0]), []byte("[")) {
			rows = make([][]float32, len(items))
			for i, r := range items {
				rows[i] = floats(r, "$f32 row")
			}
			return rows, true
		}
		return [][]float32{floats(x, "$f32")}, false
	}
	if x, ok := tagged(v, "$tensor"); ok {
		var t struct {
			Path   string            `json:"path"`
			Dtype  string            `json:"dtype"`
			Shape  []json.RawMessage `json:"shape"`
			Offset uint64            `json:"offset"`
		}
		if json.Unmarshal(x, &t) != nil || len(t.Shape) == 0 || len(t.Shape) > matrixRank {
			failf(codeBadParams, "malformed $tensor")
		}
		if t.Dtype != "f32" {
			failf(codeUnsupported, "the adapter reads f32 tensors only")
		}
		n, cols := 1, count(t.Shape[len(t.Shape)-1], "shape")
		if len(t.Shape) == matrixRank {
			n = count(t.Shape[0], "shape")
		}
		f, err := os.Open(t.Path)
		if err != nil {
			failf(codeBadParams, "cannot open tensor file")
		}
		defer f.Close()
		raw := make([]byte, 4*n*cols)
		if _, err := f.ReadAt(raw, int64(t.Offset)); err != nil {
			failf(codeBadParams, "short tensor file")
		}
		rows = make([][]float32, n)
		for r := range rows {
			rows[r] = make([]float32, cols)
			for c := range rows[r] {
				rows[r][c] = math.Float32frombits(binary.LittleEndian.Uint32(raw[4*(r*cols+c):]))
			}
		}
		return rows, len(t.Shape) == matrixRank
	}
	failf(codeUnsupported, "vectors must be $f32 or $tensor")
	return nil, false
}

// vectorOf is one vector, as an item carries it.
func vectorOf(v json.RawMessage) []float32 {
	rows, nested := vectorsOf(v)
	if nested {
		failf(codeUnsupported, "an item's vector must be one vector, not rows")
	}
	return rows[0]
}

// document decodes a JSON document with typed values: $literal passes through verbatim, $i64
// becomes an int64 and $datetime a time.Time, which the SDK encodes however it does. Numbers stay
// exact as json.Number.
func document(v json.RawMessage) any {
	dec := json.NewDecoder(bytes.NewReader(v))
	dec.UseNumber()
	var raw any
	if dec.Decode(&raw) != nil {
		failf(codeBadParams, "malformed JSON document")
	}
	return plain(raw)
}

func plain(v any) any {
	switch t := v.(type) {
	case []any:
		for i, e := range t {
			t[i] = plain(e)
		}
		return t
	case map[string]any:
		if len(t) == 1 {
			for k, x := range t {
				switch k {
				case "$literal":
					b, err := json.Marshal(x)
					if err != nil {
						failf(codeAdapter, "re-encode $literal: %v", err)
					}
					return json.RawMessage(b)
				case "$datetime":
					s, ok := x.(string)
					if !ok {
						failf(codeBadParams, "$datetime must be a string")
					}
					at, err := time.Parse(time.RFC3339Nano, s)
					if err != nil {
						failf(codeBadParams, "$datetime must be RFC 3339")
					}
					return at
				case "$i64":
					s, ok := x.(string)
					if !ok {
						failf(codeBadParams, "$i64 must be a string")
					}
					n, err := strconv.ParseInt(s, 10, 64)
					if err != nil {
						failf(codeBadParams, "invalid $i64")
					}
					return n
				case "$bytes", "$f32", "$float", "$tensor":
					failf(codeUnsupported, "typed value not expressible inside a JSON document")
				}
			}
		}
		for k, x := range t {
			t[k] = plain(x)
		}
		return t
	}
	return v
}

// Results

func f32Out(v []float32) any {
	out := make([]any, len(v))
	for i, x := range v {
		switch {
		case math.IsNaN(float64(x)):
			out[i] = map[string]string{"$float": "nan"}
		case math.IsInf(float64(x), 1):
			out[i] = map[string]string{"$float": "inf"}
		case math.IsInf(float64(x), -1):
			out[i] = map[string]string{"$float": "-inf"}
		default:
			out[i] = float64(x)
		}
	}
	return map[string]any{"$f32": out}
}

func bytesOut(b []byte) any { return map[string]string{"$bytes": base64.StdEncoding.EncodeToString(b)} }

// number keeps NaN and infinities encodable (as null), as the other adapters do.
func number(f float32) any {
	if math.IsNaN(float64(f)) || math.IsInf(float64(f), 0) {
		return nil
	}
	return float64(f)
}
