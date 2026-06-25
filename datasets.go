// datasets.go
package cyborgdb

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// SampleDatasetsBaseURL is the base URL for hosted sample datasets
	// (public-read S3 bucket). Datasets live at versioned per-dataset paths
	// ("<name>/v<n>/dataset.json.gz"), so the dataset can be iterated without an
	// SDK release: re-upload under a new version path and bump sampleDatasets.
	// It can be overridden via the CYBORGDB_SAMPLE_DATASETS_BASE_URL env var.
	SampleDatasetsBaseURL = "https://cyborgdb-sample-datasets.s3.amazonaws.com"

	// DefaultSampleDataset is the dataset loaded when LoadSampleDataset is
	// called with an empty name.
	DefaultSampleDataset = "quickstart-75k"

	// defaultSampleDatasetTimeout bounds the dataset download.
	defaultSampleDatasetTimeout = 120 * time.Second

	// numSampleQueries is how many leading queries are surfaced as
	// SampleQueries for quick demos.
	numSampleQueries = 10
)

// sampleDatasets maps a dataset name to its object path within the bucket.
var sampleDatasets = map[string]string{
	"quickstart-75k": "quickstart-75k/v1/dataset.json.gz",
}

// SampleFilter is a curated, named metadata filter guaranteed to match rows.
type SampleFilter struct {
	Name         string                 `json:"name"`
	Filter       map[string]interface{} `json:"filter"`
	Demonstrates string                 `json:"demonstrates"`
}

// SampleDataset is a fully-loaded sample dataset, ready to upsert and query.
//
// It combines dataset metadata, loader-derived convenience fields (Items,
// SampleQueries), the raw parallel arrays (Ids/Vectors/Metadata), and the
// ground-truth fixture data (Queries, *Neighbors, *Recall, ...) used to
// validate recall/accuracy. Arrays are aligned by index.
type SampleDataset struct {
	Name        string `json:"name"`
	Version     int    `json:"version"`
	Description string `json:"description"`
	Dimension   int    `json:"dimension"`
	Metric      string `json:"metric"`
	Count       int    `json:"count"`

	// Convenience fields built by the loader (absent from the raw artifact).
	Items         VectorItems `json:"-"`
	SampleQueries [][]float32 `json:"-"`

	// Raw parallel arrays.
	Ids            []string       `json:"ids"`
	Vectors        [][]float32    `json:"vectors"`
	Metadata       []interface{}  `json:"metadata"`
	ExampleFilters []SampleFilter `json:"exampleFilters"`

	// Ground-truth fixture data for recall/accuracy validation.
	Queries                    [][]float32   `json:"queries"`
	MetadataQueries            []interface{} `json:"metadata_queries"`
	MetadataQueryNames         []string      `json:"metadata_query_names"`
	UntrainedNeighbors         [][]int32     `json:"untrained_neighbors"`
	TrainedNeighbors           [][]int32     `json:"trained_neighbors"`
	UntrainedMetadataMatches   [][]int32     `json:"untrained_metadata_matches"`
	TrainedMetadataMatches     [][]int32     `json:"trained_metadata_matches"`
	UntrainedMetadataNeighbors [][][]int32   `json:"untrained_metadata_neighbors"`
	TrainedMetadataNeighbors   [][][]int32   `json:"trained_metadata_neighbors"`
	UntrainedRecall            float64       `json:"untrained_recall"`
	TrainedRecall              float64       `json:"trained_recall"`
	NumUntrainedVectors        int           `json:"num_untrained_vectors"`
	NumTrainedVectors          int           `json:"num_trained_vectors"`
}

// LoadSampleDatasetOptions configures LoadSampleDatasetWithOptions.
type LoadSampleDatasetOptions struct {
	// CacheDir is where the decompressed dataset is cached. If empty, defaults
	// to a "cyborgdb" subdirectory of the OS user cache directory.
	CacheDir string
	// ForceDownload re-downloads even if a cached copy exists.
	ForceDownload bool
}

func sampleDatasetsBaseURL() string {
	if v := os.Getenv("CYBORGDB_SAMPLE_DATASETS_BASE_URL"); v != "" {
		return v
	}
	return SampleDatasetsBaseURL
}

func defaultSampleCacheDir() string {
	if dir, err := os.UserCacheDir(); err == nil {
		return filepath.Join(dir, "cyborgdb")
	}
	return filepath.Join(os.TempDir(), "cyborgdb")
}

// hydrate builds the loader-derived convenience fields (Items, SampleQueries)
// from the raw parallel arrays. The hosted artifact and the local cache store
// only the raw arrays (no duplicated vectors), so this runs on every load.
func (ds *SampleDataset) hydrate() {
	items := make(VectorItems, len(ds.Ids))
	for i, id := range ds.Ids {
		var md map[string]interface{}
		if i < len(ds.Metadata) {
			md, _ = ds.Metadata[i].(map[string]interface{})
		}
		var vec []float32
		if i < len(ds.Vectors) {
			vec = ds.Vectors[i]
		}
		items[i] = VectorItem{Id: id, Vector: vec, Metadata: md}
	}
	ds.Items = items

	if len(ds.Queries) > numSampleQueries {
		ds.SampleQueries = ds.Queries[:numSampleQueries]
	} else {
		ds.SampleQueries = ds.Queries
	}
}

// LoadSampleDataset loads a hosted sample dataset, fetching from S3 on first
// use and caching the decompressed copy locally for subsequent calls.
//
// An empty name loads DefaultSampleDataset.
//
// Example:
//
//	dataset, err := cyborgdb.LoadSampleDataset("")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	err = index.Upsert(ctx, dataset.Items)
func LoadSampleDataset(name string) (*SampleDataset, error) {
	return LoadSampleDatasetWithOptions(name, LoadSampleDatasetOptions{})
}

// LoadSampleDatasetWithOptions is LoadSampleDataset with explicit cache control.
func LoadSampleDatasetWithOptions(name string, opts LoadSampleDatasetOptions) (*SampleDataset, error) {
	if name == "" {
		name = DefaultSampleDataset
	}

	objectPath, ok := sampleDatasets[name]
	if !ok {
		known := make([]string, 0, len(sampleDatasets))
		for k := range sampleDatasets {
			known = append(known, k)
		}
		return nil, fmt.Errorf("unknown sample dataset %q; available datasets: %s", name, strings.Join(known, ", "))
	}

	cacheDir := opts.CacheDir
	if cacheDir == "" {
		cacheDir = defaultSampleCacheDir()
	}
	// Cache key mirrors the versioned object path so a dataset bump never
	// serves a stale cached copy.
	cacheName := strings.TrimSuffix(strings.ReplaceAll(objectPath, "/", "_"), ".gz")
	cacheFile := filepath.Join(cacheDir, cacheName)

	if !opts.ForceDownload {
		if cached, err := os.ReadFile(cacheFile); err == nil {
			var ds SampleDataset
			if err := json.Unmarshal(cached, &ds); err == nil {
				ds.hydrate()
				return &ds, nil
			}
			// Corrupt cache — fall through and re-download.
		}
	}

	url := fmt.Sprintf("%s/%s", sampleDatasetsBaseURL(), objectPath)
	client := &http.Client{Timeout: defaultSampleDatasetTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download sample dataset %q from %s: %w", name, url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download sample dataset %q from %s: HTTP %d", name, url, resp.StatusCode)
	}

	// The object is stored as an opaque gzip blob (no Content-Encoding: gzip),
	// so we own the gunzip step.
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read sample dataset %q: %w", name, err)
	}
	defer func() { _ = gz.Close() }()

	jsonData, err := io.ReadAll(gz)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress sample dataset %q: %w", name, err)
	}

	var ds SampleDataset
	if err := json.Unmarshal(jsonData, &ds); err != nil {
		return nil, fmt.Errorf("failed to parse sample dataset %q: %w", name, err)
	}

	// Best-effort local cache of the raw payload; a failed write must not break
	// the load. Items/SampleQueries are rebuilt by hydrate() on read.
	if err := os.MkdirAll(cacheDir, 0o755); err == nil {
		_ = os.WriteFile(cacheFile, jsonData, 0o644)
	}

	ds.hydrate()
	return &ds, nil
}
