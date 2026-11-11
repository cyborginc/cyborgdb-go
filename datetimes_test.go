package cyborgdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// 2024-05-01T12:00:00.123Z
var testTime = time.Date(2024, 5, 1, 12, 0, 0, 123_000_000, time.UTC)

const testTimeMs = int64(1714564800123)

func TestCoerceDatetimes(t *testing.T) {
	plus2 := time.FixedZone("UTC+2", 2*60*60)
	for _, tc := range []struct {
		name string
		in   interface{}
		want interface{}
	}{
		{"time", testTime, testTimeMs},
		{"time pointer", &testTime, testTimeMs},
		{"offset normalized to UTC", testTime.In(plus2), testTimeMs},
		{"string passes through", "2024-05-01T12:00:00Z", "2024-05-01T12:00:00Z"},
		{"int passes through", 42, 42},
		{"nil passes through", nil, nil},
		{"nested map", map[string]interface{}{"a": map[string]interface{}{"b": testTime}},
			map[string]interface{}{"a": map[string]interface{}{"b": testTimeMs}}},
		{"array", []interface{}{testTime, "x", 1},
			[]interface{}{testTimeMs, "x", 1}},
		{"filter operand", map[string]interface{}{"created": map[string]interface{}{"$gt": testTime}},
			map[string]interface{}{"created": map[string]interface{}{"$gt": testTimeMs}}},
		{"$in operands", map[string]interface{}{"created": map[string]interface{}{"$in": []interface{}{testTime, testTime}}},
			map[string]interface{}{"created": map[string]interface{}{"$in": []interface{}{testTimeMs, testTimeMs}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := coerceDatetimes(tc.in)
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(tc.want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("coerceDatetimes(%v) = %s, want %s", tc.in, gotJSON, wantJSON)
			}
		})
	}
}

func TestCoerceMapDatetimesPreservesNilAndInput(t *testing.T) {
	if got := coerceMapDatetimes(nil); got != nil {
		t.Errorf("coerceMapDatetimes(nil) = %v, want nil", got)
	}
	in := map[string]interface{}{"created": testTime, "tags": []interface{}{testTime}}
	_ = coerceMapDatetimes(in)
	if _, ok := in["created"].(time.Time); !ok {
		t.Errorf("caller's map was mutated: created = %T", in["created"])
	}
	if _, ok := in["tags"].([]interface{})[0].(time.Time); !ok {
		t.Errorf("caller's nested slice was mutated: tags[0] = %T", in["tags"].([]interface{})[0])
	}
}

// captureIndex returns an index wired to a test server that records the
// request body and answers with respBody.
func captureIndex(t *testing.T, respBody string) (*EncryptedIndex, *map[string]any) {
	t.Helper()
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body = nil
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(respBody))
	}))
	t.Cleanup(srv.Close)
	client, err := NewClient(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return &EncryptedIndex{indexName: "idx", client: client.internal}, &body
}

var wantMs = float64(testTimeMs) // JSON decodes numbers as float64

func TestUpsertItemsSendsEpochMs(t *testing.T) {
	index, body := captureIndex(t, `{"message":"ok"}`)
	meta := map[string]interface{}{"created": testTime}
	err := index.Upsert(context.Background(), VectorItems{
		{Id: "doc1", Vector: []float32{0.1, 0.2}, Metadata: meta},
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	items := (*body)["items"].([]any)
	got := items[0].(map[string]any)["metadata"].(map[string]any)["created"]
	if got != wantMs {
		t.Errorf("metadata.created = %v (%T) on the wire, want %v", got, got, wantMs)
	}
	if _, ok := meta["created"].(time.Time); !ok {
		t.Errorf("caller's metadata was mutated: created = %T", meta["created"])
	}
}

func TestUpsertBinarySendsEpochMs(t *testing.T) {
	index, body := captureIndex(t, `{"message":"ok"}`)
	meta := map[string]interface{}{"created": testTime}
	err := index.Upsert(context.Background(), BinaryUpsertParams{
		IDs:      []string{"doc1"},
		Vectors:  [][]float32{{0.1, 0.2}},
		Metadata: []map[string]interface{}{meta},
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	batch := (*body)["batch"].(map[string]any)
	got := batch["metadata"].([]any)[0].(map[string]any)["created"]
	if got != wantMs {
		t.Errorf("batch.metadata[0].created = %v (%T) on the wire, want %v", got, got, wantMs)
	}
	if _, ok := meta["created"].(time.Time); !ok {
		t.Errorf("caller's metadata was mutated: created = %T", meta["created"])
	}
}

func wireFilters(t *testing.T, body *map[string]any) map[string]any {
	t.Helper()
	filters, ok := (*body)["filters"].(map[string]any)
	if !ok {
		t.Fatalf("no filters object on the wire: %v", *body)
	}
	return filters
}

func timeFilters() map[string]interface{} {
	return map[string]interface{}{"created": map[string]interface{}{"$gt": testTime}}
}

func assertGtEpochMs(t *testing.T, filters map[string]any) {
	t.Helper()
	got := filters["created"].(map[string]any)["$gt"]
	if got != wantMs {
		t.Errorf("filters.created.$gt = %v (%T) on the wire, want %v", got, got, wantMs)
	}
}

func TestQuerySendsFilterEpochMs(t *testing.T) {
	index, body := captureIndex(t, `{"results":[]}`)
	_, err := index.Query(context.Background(), QueryParams{
		QueryVector: []float32{0.1, 0.2},
		Filters:     timeFilters(),
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	assertGtEpochMs(t, wireFilters(t, body))
}

func TestBatchQuerySendsFilterEpochMs(t *testing.T) {
	index, body := captureIndex(t, `{"results":[]}`)
	_, err := index.Query(context.Background(), QueryParams{
		BatchQueryVectors: [][]float32{{0.1, 0.2}, {0.3, 0.4}},
		Filters:           timeFilters(),
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	assertGtEpochMs(t, wireFilters(t, body))
}

func TestBinaryQuerySendsFilterEpochMs(t *testing.T) {
	index, body := captureIndex(t, `{"results":[]}`)
	_, err := index.Query(context.Background(), BinaryQueryParams{
		QueryVectors: [][]float32{{0.1, 0.2}},
		Filters:      timeFilters(),
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	assertGtEpochMs(t, wireFilters(t, body))
}

func TestQueryMetadataSendsFilterEpochMs(t *testing.T) {
	index, body := captureIndex(t, `{"results":[],"ids":[],"count":0}`)
	_, err := index.QueryMetadata(context.Background(), QueryMetadataParams{
		Filters: timeFilters(),
	})
	if err != nil {
		t.Fatalf("QueryMetadata: %v", err)
	}
	assertGtEpochMs(t, wireFilters(t, body))
}
