// datasets.go
package cyborgdb

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	SampleDatasetsBaseURL = "https://cyborgdb-sample-datasets.s3.amazonaws.com"

	// DefaultSampleDataset is the dataset loaded when LoadSampleDataset is
	// called with an empty name.
	DefaultSampleDataset = "quickstart-75k"

	// defaultSampleDatasetTimeout bounds the dataset download.
	defaultSampleDatasetTimeout = 120 * time.Second

	// maxDecompressedBytes caps the decompressed dataset size, guarding against
	// a decompression bomb: a tiny gzip that expands to many GBs and OOMs the
	// host. The largest shipped dataset is well under this generous cap.
	maxDecompressedBytes = 512 * 1024 * 1024

	// numSampleQueries is how many leading queries are surfaced as
	// SampleQueries for quick demos.
	numSampleQueries = 10

	// sampleCacheDirPerm is the permission used for the local cache directory.
	sampleCacheDirPerm = 0o755
	// sampleCacheFilePerm is the permission used for the cached dataset file.
	sampleCacheFilePerm = 0o644
)

var (
	// ErrUnknownSampleDataset is returned when an unrecognized dataset name is requested.
	ErrUnknownSampleDataset = errors.New("unknown sample dataset")
	// ErrSampleDatasetDownload is returned when the dataset download fails.
	ErrSampleDatasetDownload = errors.New("failed to download sample dataset")
)

// sampleDatasetEntry describes where a dataset lives and how to verify it.
type sampleDatasetEntry struct {
	// objectPath is the dataset's path within the bucket.
	objectPath string
	// sha256 is the hex SHA-256 of the decompressed JSON, pinned so a bucket
	// compromise or a poisoned local cache file can't be trusted silently. The
	// same digest is verified post-download and on cache read.
	sha256 string
}

// sampleDatasets maps a dataset name to its catalog entry.
var sampleDatasets = map[string]sampleDatasetEntry{
	"quickstart-75k": {
		objectPath: "quickstart-75k/v1/dataset.json.gz",
		sha256:     "6e2db96a0932f036698ebf5e25cf0871cc69b649f7fb352f9e3dddcf9af0540f",
	},
}

// sampleDatasetsBaseURLOverride is an unexported test hook to redirect dataset
// downloads at the transport layer. Production code never sets it; tests in
// this package point it at a local httptest server. Unlike an env var, it is
// not a redirect/downgrade surface reachable from outside the package.
var sampleDatasetsBaseURLOverride string

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
	if sampleDatasetsBaseURLOverride != "" {
		return sampleDatasetsBaseURLOverride
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
			if m, ok := ds.Metadata[i].(map[string]interface{}); ok {
				md = m
			}
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

	entry, ok := sampleDatasets[name]
	if !ok {
		known := make([]string, 0, len(sampleDatasets))
		for k := range sampleDatasets {
			known = append(known, k)
		}
		return nil, fmt.Errorf("%w %q; available datasets: %s", ErrUnknownSampleDataset, name, strings.Join(known, ", "))
	}

	cacheDir := opts.CacheDir
	if cacheDir == "" {
		cacheDir = defaultSampleCacheDir()
	}
	// Cache key mirrors the versioned object path so a dataset bump never
	// serves a stale cached copy.
	cacheName := strings.TrimSuffix(strings.ReplaceAll(entry.objectPath, "/", "_"), ".gz")
	cacheFile := filepath.Join(cacheDir, cacheName)

	if !opts.ForceDownload {
		if cached, err := os.ReadFile(cacheFile); err == nil {
			// Verify the cached file against the pinned digest: a poisoned cache
			// must not be trusted. A mismatch falls through to re-download.
			if hex.EncodeToString(sumSHA256(cached)) == entry.sha256 {
				var ds SampleDataset
				if err := json.Unmarshal(cached, &ds); err == nil {
					ds.hydrate()
					return &ds, nil
				}
				// Corrupt cache — fall through and re-download.
			}
		}
	}

	url := fmt.Sprintf("%s/%s", sampleDatasetsBaseURL(), entry.objectPath)
	ctx, cancel := context.WithTimeout(context.Background(), defaultSampleDatasetTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for sample dataset %q: %w", name, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download sample dataset %q from %s: %w", name, url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w %q from %s: HTTP %d", ErrSampleDatasetDownload, name, url, resp.StatusCode)
	}

	// The object is stored as an opaque gzip blob (no Content-Encoding: gzip),
	// so we own the gunzip step.
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read sample dataset %q: %w", name, err)
	}
	defer func() { _ = gz.Close() }()

	// Read one byte past the cap so we can detect (rather than silently
	// truncate) a payload that exceeds the decompression-bomb limit.
	jsonData, err := io.ReadAll(io.LimitReader(gz, maxDecompressedBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to decompress sample dataset %q: %w", name, err)
	}
	if int64(len(jsonData)) > maxDecompressedBytes {
		return nil, fmt.Errorf("sample dataset %q exceeds maximum decompressed size of %d bytes", name, maxDecompressedBytes)
	}

	if got := hex.EncodeToString(sumSHA256(jsonData)); got != entry.sha256 {
		return nil, fmt.Errorf("integrity check failed for sample dataset %q: expected SHA-256 %s, got %s", name, entry.sha256, got)
	}

	var ds SampleDataset
	if err := json.Unmarshal(jsonData, &ds); err != nil {
		return nil, fmt.Errorf("failed to parse sample dataset %q: %w", name, err)
	}

	// Best-effort local cache of the raw payload; a failed write must not break
	// the load. Items/SampleQueries are rebuilt by hydrate() on read.
	if mkErr := os.MkdirAll(cacheDir, sampleCacheDirPerm); mkErr == nil {
		// A failed cache write is non-fatal: the dataset still loads this call.
		if wErr := os.WriteFile(cacheFile, jsonData, sampleCacheFilePerm); wErr != nil {
			_ = wErr
		}
	}

	ds.hydrate()
	return &ds, nil
}

// sumSHA256 returns the SHA-256 digest of data.
func sumSHA256(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}
