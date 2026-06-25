package test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
)

// fakeRaw is the *raw* hosted shape (no items/sample_queries); those
// convenience fields are rebuilt by the loader's hydrate step.
var fakeRaw = map[string]interface{}{
	"name":        "quickstart-75k",
	"version":     1,
	"description": "test fixture",
	"dimension":   3,
	"metric":      "euclidean",
	"count":       2,
	"exampleFilters": []map[string]interface{}{
		{"name": "eq", "filter": map[string]interface{}{"string": "a"}, "demonstrates": "equality"},
	},
	"ids":      []string{"item_0", "item_1"},
	"vectors":  [][]float32{{1, 2, 3}, {4, 5, 6}},
	"metadata": []map[string]interface{}{{"number": 0, "string": "a"}, {"number": 1, "string": "b"}},
	"queries":  [][]float32{{1, 2, 3}},

	"metadata_queries":             []map[string]interface{}{{"string": "a"}},
	"metadata_query_names":         []string{"eq string a"},
	"untrained_neighbors":          [][]int32{{0}},
	"trained_neighbors":            [][]int32{{0}},
	"untrained_metadata_matches":   [][]int32{{1}},
	"trained_metadata_matches":     [][]int32{{1}},
	"untrained_metadata_neighbors": [][][]int32{{{0}}},
	"trained_metadata_neighbors":   [][][]int32{{{0}}},
	"untrained_recall":             1.0,
	"trained_recall":               0.94,
	"num_untrained_vectors":        1,
	"num_trained_vectors":          1,
}

// gzipBytes marshals v to JSON and gzips it (matching the opaque-blob hosting).
func gzipBytes(t *testing.T, v interface{}) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(raw); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

// newServer serves the gzipped payload and returns a pointer to the request
// count. The server is closed automatically via t.Cleanup.
func newServer(t *testing.T, payload []byte, status int) *int {
	t.Helper()
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(payload)
	}))
	t.Cleanup(ts.Close)
	t.Setenv("CYBORGDB_SAMPLE_DATASETS_BASE_URL", ts.URL)
	return &calls
}

func TestLoadSampleDataset_DownloadAndHydrate(t *testing.T) {
	calls := newServer(t, gzipBytes(t, fakeRaw), http.StatusOK)

	ds, err := cyborgdb.LoadSampleDatasetWithOptions("quickstart-75k",
		cyborgdb.LoadSampleDatasetOptions{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if *calls != 1 {
		t.Errorf("expected 1 request, got %d", *calls)
	}
	if ds.Count != 2 || ds.Dimension != 3 {
		t.Errorf("metadata mismatch: count=%d dim=%d", ds.Count, ds.Dimension)
	}
	if len(ds.Items) != 2 || ds.Items[0].Id != "item_0" {
		t.Fatalf("items not hydrated: %+v", ds.Items)
	}
	if got := ds.Items[0].Vector; len(got) != 3 || got[0] != 1 {
		t.Errorf("item vector mismatch: %v", got)
	}
	if ds.Items[1].Metadata["string"] != "b" {
		t.Errorf("item metadata mismatch: %v", ds.Items[1].Metadata)
	}
	if len(ds.SampleQueries) != 1 {
		t.Errorf("sample queries mismatch: %v", ds.SampleQueries)
	}
	if len(ds.ExampleFilters) != 1 || ds.ExampleFilters[0].Filter["string"] != "a" {
		t.Errorf("example filters mismatch: %v", ds.ExampleFilters)
	}
	// raw ground-truth fields pass through
	if ds.TrainedRecall != 0.94 {
		t.Errorf("trained recall mismatch: %v", ds.TrainedRecall)
	}
	if len(ds.UntrainedNeighbors) != 1 || ds.UntrainedNeighbors[0][0] != 0 {
		t.Errorf("ground-truth neighbors mismatch: %v", ds.UntrainedNeighbors)
	}
}

func TestLoadSampleDataset_SecondCallUsesCache(t *testing.T) {
	calls := newServer(t, gzipBytes(t, fakeRaw), http.StatusOK)
	cacheDir := t.TempDir()
	opts := cyborgdb.LoadSampleDatasetOptions{CacheDir: cacheDir}

	if _, err := cyborgdb.LoadSampleDatasetWithOptions("", opts); err != nil {
		t.Fatalf("load 1: %v", err)
	}
	if _, err := cyborgdb.LoadSampleDatasetWithOptions("", opts); err != nil {
		t.Fatalf("load 2: %v", err)
	}
	if *calls != 1 {
		t.Errorf("expected 1 request (cache hit), got %d", *calls)
	}
}

func TestLoadSampleDataset_ForceDownloadRefetches(t *testing.T) {
	calls := newServer(t, gzipBytes(t, fakeRaw), http.StatusOK)
	cacheDir := t.TempDir()

	if _, err := cyborgdb.LoadSampleDatasetWithOptions("",
		cyborgdb.LoadSampleDatasetOptions{CacheDir: cacheDir}); err != nil {
		t.Fatalf("load 1: %v", err)
	}
	if _, err := cyborgdb.LoadSampleDatasetWithOptions("",
		cyborgdb.LoadSampleDatasetOptions{CacheDir: cacheDir, ForceDownload: true}); err != nil {
		t.Fatalf("load 2: %v", err)
	}
	if *calls != 2 {
		t.Errorf("expected 2 requests, got %d", *calls)
	}
}

func TestLoadSampleDataset_UnknownDatasetErrors(t *testing.T) {
	calls := newServer(t, gzipBytes(t, fakeRaw), http.StatusOK)
	if _, err := cyborgdb.LoadSampleDatasetWithOptions("does-not-exist",
		cyborgdb.LoadSampleDatasetOptions{CacheDir: t.TempDir()}); err == nil {
		t.Fatal("expected error for unknown dataset")
	}
	if *calls != 0 {
		t.Errorf("expected no request for unknown dataset, got %d", *calls)
	}
}

func TestLoadSampleDataset_DownloadFailureErrors(t *testing.T) {
	newServer(t, nil, http.StatusNotFound)
	if _, err := cyborgdb.LoadSampleDatasetWithOptions("",
		cyborgdb.LoadSampleDatasetOptions{CacheDir: t.TempDir()}); err == nil {
		t.Fatal("expected error on HTTP 404")
	}
}
