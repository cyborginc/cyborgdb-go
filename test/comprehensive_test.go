package test

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
	"github.com/cyborginc/cyborgdb-go/internal"
)

const (
	baseTimeout = 10 * time.Second
	longTimeout = 120 * time.Second
)

var (
	ErrAPIKeyRequired = errors.New("CYBORGDB_API_KEY environment variable is required")
)

func createClient() (*cyborgdb.Client, error) {
	apiKey := os.Getenv("CYBORGDB_API_KEY")
	if apiKey == "" {
		return nil, ErrAPIKeyRequired
	}
	return cyborgdb.NewClient("http://localhost:8000", apiKey)
}

// compTestIndex creates an isolated index for a comprehensive test with cleanup.
func compTestIndex(t *testing.T) *cyborgdb.EncryptedIndex {
	t.Helper()
	client, err := createClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	metric := "euclidean"
	index, err := client.CreateIndex(
		context.Background(),
		&cyborgdb.CreateIndexParams{
			IndexName:   generateUniqueName("comp_"),
			IndexKey:    generateRandomKey(),
			IndexConfig: cyborgdb.IndexIVFFlat(128),
			Metric:      &metric,
		},
	)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = index.DeleteIndex(ctx)
	})
	return index
}

// ---------------------------------------------------------------------------
// Error Handling — validates the SDK and server reject bad input
// ---------------------------------------------------------------------------

func TestInvalidAPIKeyRejected(t *testing.T) {
	// Catches: auth bypass, missing auth enforcement on server.
	// A bad API key must be rejected on any mutating operation.
	ctx, cancel := context.WithTimeout(context.Background(), baseTimeout)
	defer cancel()

	client, err := cyborgdb.NewClient("http://localhost:8000", "definitely-invalid-key-12345")
	if err != nil {
		t.Fatalf("Client creation should not fail with invalid API key: %v", err)
	}

	metric := "euclidean"
	_, createErr := client.CreateIndex(ctx, &cyborgdb.CreateIndexParams{
		IndexName:   generateUniqueName("auth_test_"),
		IndexKey:    generateRandomKey(),
		IndexConfig: cyborgdb.IndexIVFFlat(128),
		Metric:      &metric,
	})
	if createErr == nil {
		t.Fatal("Invalid API key was accepted — authentication is not enforced")
	}
}

func TestMalformedRequestsRejected(t *testing.T) {
	// Catches: server accepting garbage input that would corrupt state.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := createClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	testCases := []struct {
		name   string
		params *cyborgdb.CreateIndexParams
	}{
		{
			"Negative dimension",
			&cyborgdb.CreateIndexParams{
				IndexName:   generateUniqueName("neg_dim_"),
				IndexKey:    generateRandomKey(),
				IndexConfig: cyborgdb.IndexIVFFlat(-1),
				Metric:      strPtr("euclidean"),
			},
		},
		{
			"Invalid metric",
			&cyborgdb.CreateIndexParams{
				IndexName:   generateUniqueName("bad_metric_"),
				IndexKey:    generateRandomKey(),
				IndexConfig: cyborgdb.IndexIVFFlat(128),
				Metric:      strPtr("completely_invalid_metric"),
			},
		},
		{
			"Empty index name",
			&cyborgdb.CreateIndexParams{
				IndexName:   "",
				IndexKey:    generateRandomKey(),
				IndexConfig: cyborgdb.IndexIVFFlat(128),
				Metric:      strPtr("euclidean"),
			},
		},
		{
			"Short key (8 bytes instead of 32)",
			&cyborgdb.CreateIndexParams{
				IndexName:   generateUniqueName("short_key_"),
				IndexKey:    make([]byte, 8),
				IndexConfig: cyborgdb.IndexIVFFlat(128),
				Metric:      strPtr("euclidean"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := client.CreateIndex(ctx, tc.params)
			if err == nil {
				t.Errorf("Server accepted malformed request: %s", tc.name)
			}
		})
	}
}

func TestVectorDimensionValidation(t *testing.T) {
	// Catches: server silently accepting wrong-dimension vectors, which would
	// corrupt the index or produce garbage query results.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	index := compTestIndex(t)

	testCases := []struct {
		name       string
		dimension  int
		shouldFail bool
	}{
		{"Wrong dimension (64)", 64, true},
		{"Wrong dimension (256)", 256, true},
		{"Empty vector (0)", 0, true},
		{"Correct dimension (128)", 128, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vector := make([]float32, tc.dimension)
			for i := range vector {
				vector[i] = float32(i) / 100.0
			}
			items := cyborgdb.VectorItems{{
				Id:     fmt.Sprintf("dimtest_%d", tc.dimension),
				Vector: vector,
			}}

			err := index.Upsert(ctx, items)
			if tc.shouldFail && err == nil {
				t.Errorf("Server accepted vector with dimension %d on a 128-dim index", tc.dimension)
			} else if !tc.shouldFail && err != nil {
				t.Errorf("Rejected valid vector: %v", err)
			}
		})
	}
}

func TestNetworkErrorHandling(t *testing.T) {
	// Catches: SDK panicking or hanging on unreachable server instead of
	// returning a clean error.
	client, err := cyborgdb.NewClient("http://non-existent-server-12345.invalid:8000", "test-key")
	if err != nil {
		t.Fatalf("Client creation should not fail: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, healthErr := client.GetHealth(ctx)
	if healthErr == nil {
		t.Error("Expected network error for non-existent server, got nil")
	}
}

// ---------------------------------------------------------------------------
// Data Integrity — validates data survives round-trips correctly
// ---------------------------------------------------------------------------

func TestVectorAndMetadataRoundTrip(t *testing.T) {
	// Catches: encryption/encoding bugs that corrupt vector data or metadata
	// during upsert→get round-trip. Checks element-by-element precision.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	index := compTestIndex(t)

	originalVector := generateTestVectors(1, 128)[0]
	originalMetadata := map[string]interface{}{
		"string_val": "hello world",
		"number_val": float64(42),
		"bool_val":   true,
		"array_val":  []interface{}{float64(1), float64(2), float64(3)},
		"nested_val": map[string]interface{}{"inner": "value"},
	}

	items := cyborgdb.VectorItems{{
		Id:       "roundtrip_test",
		Vector:   originalVector,
		Metadata: originalMetadata,
	}}
	if err := index.Upsert(ctx, items); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	waitForPropagation(2 * time.Second)

	results, err := index.Get(ctx, []string{"roundtrip_test"}, []string{"vector", "metadata"})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(results.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results.Results))
	}

	retrieved := results.Results[0]

	// Verify vector element-by-element
	retrievedVec := retrieved.GetVector()
	if len(retrievedVec) != 128 {
		t.Fatalf("Vector length mismatch: expected 128, got %d", len(retrievedVec))
	}
	for i, expected := range originalVector {
		diff := math.Abs(float64(retrievedVec[i]) - float64(expected))
		if diff > 1e-6 {
			t.Errorf("Vector corruption at index %d: expected %f, got %f", i, expected, retrievedVec[i])
		}
	}

	// Verify metadata fields
	meta := retrieved.GetMetadata()
	if meta["string_val"] != "hello world" {
		t.Errorf("Metadata string_val: expected 'hello world', got '%v'", meta["string_val"])
	}
	if meta["number_val"] != float64(42) {
		t.Errorf("Metadata number_val: expected 42, got '%v'", meta["number_val"])
	}
	if meta["bool_val"] != true {
		t.Errorf("Metadata bool_val: expected true, got '%v'", meta["bool_val"])
	}
	nested, ok := meta["nested_val"].(map[string]interface{})
	if !ok || nested["inner"] != "value" {
		t.Errorf("Metadata nested_val corrupted: got '%v'", meta["nested_val"])
	}
}

func TestUpsertOverwritePreservesLatestData(t *testing.T) {
	// Catches: stale cache, write-behind bugs where an earlier upsert's data
	// is returned instead of the latest, or metadata from version 1 leaks
	// into version 2.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	index := compTestIndex(t)

	vecV1 := generateTestVectors(1, 128)[0]
	metaV1 := map[string]interface{}{"version": float64(1), "old_field": "should_disappear"}
	if err := index.Upsert(ctx, cyborgdb.VectorItems{{
		Id: "overwrite_test", Vector: vecV1, Metadata: metaV1,
	}}); err != nil {
		t.Fatalf("Upsert v1 failed: %v", err)
	}

	waitForPropagation(2 * time.Second)

	// Overwrite with different vector and metadata
	vecV2 := generateTestVectors(1, 128)[0]
	// Make v2 distinct from v1
	for i := range vecV2 {
		vecV2[i] += 10.0
	}
	metaV2 := map[string]interface{}{"version": float64(2), "new_field": "present"}
	if err := index.Upsert(ctx, cyborgdb.VectorItems{{
		Id: "overwrite_test", Vector: vecV2, Metadata: metaV2,
	}}); err != nil {
		t.Fatalf("Upsert v2 failed: %v", err)
	}

	waitForPropagation(2 * time.Second)

	results, err := index.Get(ctx, []string{"overwrite_test"}, []string{"vector", "metadata"})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(results.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results.Results))
	}

	retrieved := results.Results[0]
	retrievedVec := retrieved.GetVector()

	// The retrieved vector must match v2, not v1
	for i := range vecV2 {
		diff := math.Abs(float64(retrievedVec[i]) - float64(vecV2[i]))
		if diff > 1e-6 {
			t.Errorf("Vector at index %d matches v1 instead of v2: got %f, want %f",
				i, retrievedVec[i], vecV2[i])
			break
		}
	}

	// Metadata must be v2's metadata
	meta := retrieved.GetMetadata()
	if meta["version"] != float64(2) {
		t.Errorf("Expected metadata version=2, got %v — stale data returned", meta["version"])
	}
	if meta["new_field"] != "present" {
		t.Errorf("Expected new_field='present', got %v", meta["new_field"])
	}
}

func TestDeleteActuallyRemovesData(t *testing.T) {
	// Catches: soft-delete bugs where data lingers in queries or Get after
	// deletion, or ListIDs still shows deleted vectors.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	index := compTestIndex(t)

	// Upsert 10 vectors
	vectors := generateTestVectors(10, 128)
	items := make(cyborgdb.VectorItems, 10)
	for i := 0; i < 10; i++ {
		items[i] = cyborgdb.VectorItem{
			Id:     fmt.Sprintf("del_test_%d", i),
			Vector: vectors[i],
		}
	}
	if err := index.Upsert(ctx, items); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	waitForPropagation(2 * time.Second)

	// Delete first 5
	deleteIDs := []string{"del_test_0", "del_test_1", "del_test_2", "del_test_3", "del_test_4"}
	if err := index.Delete(ctx, deleteIDs); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	waitForPropagation(2 * time.Second)

	// Get deleted IDs — should return empty results
	getResp, err := index.Get(ctx, deleteIDs, []string{"vector"})
	if err != nil {
		t.Fatalf("Get deleted IDs failed: %v", err)
	}
	if len(getResp.Results) != 0 {
		returnedIDs := make([]string, len(getResp.Results))
		for i, r := range getResp.Results {
			returnedIDs[i] = r.GetId()
		}
		t.Errorf("Deleted vectors still returned by Get: %v", returnedIDs)
	}

	// ListIDs — should only contain the surviving 5
	listResp, err := index.ListIDs(ctx)
	if err != nil {
		t.Fatalf("ListIDs failed: %v", err)
	}
	for _, id := range listResp.Ids {
		for _, deleted := range deleteIDs {
			if id == deleted {
				t.Errorf("Deleted ID '%s' still appears in ListIDs", id)
			}
		}
	}
	if len(listResp.Ids) != 5 {
		t.Errorf("Expected 5 surviving IDs, got %d", len(listResp.Ids))
	}

	// Query with a deleted vector — it must not appear in results
	queryResp, err := index.Query(ctx, cyborgdb.QueryParams{
		QueryVector: vectors[0], // vector for del_test_0
		TopK:        10,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	resultItems := getQueryResultItems(&queryResp.Results)
	for _, item := range resultItems {
		for _, deleted := range deleteIDs {
			if item.GetId() == deleted {
				t.Errorf("Deleted ID '%s' still appears in query results", item.GetId())
			}
		}
	}
}

func TestDuplicateIndexNameRejected(t *testing.T) {
	// Catches: silent overwrite of an existing index when creating a new one
	// with the same name, which would destroy production data.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := createClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	name := generateUniqueName("dup_test_")
	key := generateRandomKey()
	metric := "euclidean"
	params := &cyborgdb.CreateIndexParams{
		IndexName:   name,
		IndexKey:    key,
		IndexConfig: cyborgdb.IndexIVFFlat(128),
		Metric:      &metric,
	}

	index, err := client.CreateIndex(ctx, params)
	if err != nil {
		t.Fatalf("First CreateIndex failed: %v", err)
	}
	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = index.DeleteIndex(cleanCtx)
	})

	// Second create with same name must fail
	_, dupErr := client.CreateIndex(ctx, &cyborgdb.CreateIndexParams{
		IndexName:   name,
		IndexKey:    generateRandomKey(),
		IndexConfig: cyborgdb.IndexIVFFlat(128),
		Metric:      &metric,
	})
	if dupErr == nil {
		t.Fatal("Server accepted duplicate index name — would silently overwrite existing data")
	}
}

func TestWrongKeyCannotAccessData(t *testing.T) {
	// Catches: encryption key bypass, key validation bugs where any key
	// can read another index's data.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := createClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	name := generateUniqueName("wrongkey_")
	correctKey := generateRandomKey()
	metric := "euclidean"

	index, err := client.CreateIndex(ctx, &cyborgdb.CreateIndexParams{
		IndexName:   name,
		IndexKey:    correctKey,
		IndexConfig: cyborgdb.IndexIVFFlat(128),
		Metric:      &metric,
	})
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}
	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = index.DeleteIndex(cleanCtx)
	})

	// Upsert data with the correct key
	vector := generateTestVectors(1, 128)[0]
	if err := index.Upsert(ctx, cyborgdb.VectorItems{{
		Id: "secret_data", Vector: vector,
	}}); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	waitForPropagation(2 * time.Second)

	// Try to load the same index with a wrong key
	wrongKey := generateRandomKey()
	_, loadErr := client.LoadIndex(ctx, name, wrongKey)
	if loadErr == nil {
		t.Fatal("LoadIndex with wrong key succeeded — encryption key validation is broken")
	}
}

func TestGetNonExistentIDs(t *testing.T) {
	// Catches: server crashing or returning errors when asked for IDs that
	// don't exist, instead of gracefully returning empty results.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	index := compTestIndex(t)

	// Upsert one vector so the index isn't completely empty
	if err := index.Upsert(ctx, cyborgdb.VectorItems{{
		Id:     "exists",
		Vector: generateTestVectors(1, 128)[0],
	}}); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	waitForPropagation(2 * time.Second)

	// Get a mix of existing and non-existing IDs
	resp, err := index.Get(ctx, []string{"exists", "ghost_1", "ghost_2"}, []string{"vector"})
	if err != nil {
		t.Fatalf("Get with non-existent IDs should not error: %v", err)
	}

	// Should return only the existing vector
	if len(resp.Results) != 1 {
		t.Errorf("Expected 1 result for the existing ID, got %d", len(resp.Results))
	}
	if len(resp.Results) > 0 && resp.Results[0].GetId() != "exists" {
		t.Errorf("Expected ID 'exists', got '%s'", resp.Results[0].GetId())
	}
}

// ---------------------------------------------------------------------------
// Index Type — validates IVFSQ and IVFPQ produce correct results
// ---------------------------------------------------------------------------

func TestIVFSQQueryCorrectness(t *testing.T) {
	// Catches: scalar quantization corrupting vector data enough to return
	// wrong nearest neighbors, broken distance computation, unsorted results.
	ctx, cancel := context.WithTimeout(context.Background(), longTimeout)
	defer cancel()

	client, err := createClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	metric := "euclidean"
	index, err := client.CreateIndex(ctx, &cyborgdb.CreateIndexParams{
		IndexName:   generateUniqueName("ivfsq_"),
		IndexKey:    generateRandomKey(),
		IndexConfig: cyborgdb.IndexIVFSQ(128, 8),
		Metric:      &metric,
	})
	if err != nil {
		t.Fatalf("Failed to create IVFSQ index: %v", err)
	}
	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = index.DeleteIndex(cleanCtx)
	})

	if index.GetIndexType() != "ivfsq" {
		t.Errorf("Expected index type 'ivfsq', got '%s'", index.GetIndexType())
	}

	vectors := generateTestVectors(50, 128)
	items := make(cyborgdb.VectorItems, len(vectors))
	for i, v := range vectors {
		items[i] = cyborgdb.VectorItem{Id: fmt.Sprintf("sq_%d", i), Vector: v}
	}
	if err := index.Upsert(ctx, items); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	queryParams := cyborgdb.QueryParams{QueryVector: vectors[0], TopK: 5}
	var resultItems []internal.QueryResultItem
	ok := pollUntil(30*time.Second, 2*time.Second, func() bool {
		results, err := index.Query(ctx, queryParams)
		if err != nil || results == nil {
			return false
		}
		resultItems = getQueryResultItems(&results.Results)
		return len(resultItems) > 0
	})
	if !ok {
		t.Fatal("IVFSQ query returned no results within 30s")
	}

	// Self-match: querying with vectors[0] must return sq_0 first
	if resultItems[0].GetId() != "sq_0" {
		t.Errorf("Expected nearest neighbor 'sq_0', got '%s' — SQ compression corrupting lookups", resultItems[0].GetId())
	}
	if resultItems[0].GetDistance() > 1.0 {
		t.Errorf("Self-distance should be near zero, got %f", resultItems[0].GetDistance())
	}

	// Distances must be non-negative and ordered
	for i, item := range resultItems {
		if item.GetDistance() < 0 {
			t.Errorf("Result %d: negative distance %f", i, item.GetDistance())
		}
		if i > 0 && item.GetDistance() < resultItems[i-1].GetDistance() {
			t.Errorf("Results not sorted: [%d]=%f < [%d]=%f",
				i, item.GetDistance(), i-1, resultItems[i-1].GetDistance())
		}
	}
}

func TestIVFPQQueryCorrectness(t *testing.T) {
	// Catches: PQ compression/decompression corrupting lookups, broken
	// distance computation in quantized space, unsorted results.
	ctx, cancel := context.WithTimeout(context.Background(), longTimeout)
	defer cancel()

	client, err := createClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	metric := "euclidean"
	index, err := client.CreateIndex(ctx, &cyborgdb.CreateIndexParams{
		IndexName:   generateUniqueName("ivfpq_"),
		IndexKey:    generateRandomKey(),
		IndexConfig: cyborgdb.IndexIVFPQ(128, 32, 8),
		Metric:      &metric,
	})
	if err != nil {
		t.Fatalf("Failed to create IVFPQ index: %v", err)
	}
	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = index.DeleteIndex(cleanCtx)
	})

	if index.GetIndexType() != "ivfpq" {
		t.Errorf("Expected index type 'ivfpq', got '%s'", index.GetIndexType())
	}

	vectors := generateTestVectors(50, 128)
	items := make(cyborgdb.VectorItems, len(vectors))
	for i, v := range vectors {
		items[i] = cyborgdb.VectorItem{Id: fmt.Sprintf("pq_%d", i), Vector: v}
	}
	if err := index.Upsert(ctx, items); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	queryParams := cyborgdb.QueryParams{QueryVector: vectors[0], TopK: 5}
	var resultItems []internal.QueryResultItem
	ok := pollUntil(30*time.Second, 2*time.Second, func() bool {
		results, err := index.Query(ctx, queryParams)
		if err != nil || results == nil {
			return false
		}
		resultItems = getQueryResultItems(&results.Results)
		return len(resultItems) > 0
	})
	if !ok {
		t.Fatal("IVFPQ query returned no results within 30s")
	}

	if resultItems[0].GetId() != "pq_0" {
		t.Errorf("Expected nearest neighbor 'pq_0', got '%s' — PQ compression corrupting lookups", resultItems[0].GetId())
	}
	if resultItems[0].GetDistance() > 1.0 {
		t.Errorf("Self-distance should be near zero, got %f", resultItems[0].GetDistance())
	}

	for i, item := range resultItems {
		if item.GetDistance() < 0 {
			t.Errorf("Result %d: negative distance %f", i, item.GetDistance())
		}
		if i > 0 && item.GetDistance() < resultItems[i-1].GetDistance() {
			t.Errorf("Results not sorted: [%d]=%f < [%d]=%f",
				i, item.GetDistance(), i-1, resultItems[i-1].GetDistance())
		}
	}
}

// ---------------------------------------------------------------------------
// Edge Cases — boundary values and large metadata round-trips
// ---------------------------------------------------------------------------

func TestBoundaryVectorValuesRoundTrip(t *testing.T) {
	// Catches: float overflow/underflow in encryption, NaN propagation,
	// encoding bugs for extreme values. Verifies vectors survive round-trip,
	// not just that upsert succeeds.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	index := compTestIndex(t)

	testCases := []struct {
		name   string
		vector []float32
	}{
		{"Very small values (1e-10)", generateVectorWithValue(128, 1e-10)},
		{"Very large values (1e10)", generateVectorWithValue(128, 1e10)},
		{"Mixed positive and negative", generateMixedVector(128)},
	}

	// Upsert all
	for i, tc := range testCases {
		id := fmt.Sprintf("boundary_%d", i)
		if err := index.Upsert(ctx, cyborgdb.VectorItems{{
			Id: id, Vector: tc.vector,
		}}); err != nil {
			t.Fatalf("Upsert failed for %s: %v", tc.name, err)
		}
	}

	waitForPropagation(2 * time.Second)

	// Verify round-trip
	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id := fmt.Sprintf("boundary_%d", i)
			resp, err := index.Get(ctx, []string{id}, []string{"vector"})
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}
			if len(resp.Results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(resp.Results))
			}

			retrieved := resp.Results[0].GetVector()
			if len(retrieved) != 128 {
				t.Fatalf("Vector length mismatch: got %d", len(retrieved))
			}

			for j, expected := range tc.vector {
				diff := math.Abs(float64(retrieved[j]) - float64(expected))
				if diff > 1e-4 {
					t.Errorf("Element %d: expected %e, got %e (diff %e)",
						j, expected, retrieved[j], diff)
					break
				}
			}
		})
	}
}

func TestLargeMetadataRoundTrip(t *testing.T) {
	// Catches: metadata truncation, JSON serialization bugs for deep nesting,
	// large strings being silently dropped. Verifies content matches, not just
	// that the server accepted it.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	index := compTestIndex(t)

	largeString := strings.Repeat("A", 1000)
	testCases := []struct {
		name     string
		metadata map[string]interface{}
		checkKey string
		checkVal interface{}
	}{
		{
			"1KB string value",
			map[string]interface{}{"description": largeString},
			"description", largeString,
		},
		{
			"Deep nesting (5 levels)",
			createDeepNestedMetadata(5),
			"level", float64(5), // JSON unmarshals numbers as float64
		},
		{
			"50-element array",
			map[string]interface{}{"tags": makeIntSlice(50)},
			"tags", nil, // checked separately
		},
	}

	for i, tc := range testCases {
		id := fmt.Sprintf("meta_%d", i)
		if err := index.Upsert(ctx, cyborgdb.VectorItems{{
			Id: id, Vector: generateTestVectors(1, 128)[0], Metadata: tc.metadata,
		}}); err != nil {
			t.Fatalf("Upsert failed for %s: %v", tc.name, err)
		}
	}

	waitForPropagation(2 * time.Second)

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id := fmt.Sprintf("meta_%d", i)
			resp, err := index.Get(ctx, []string{id}, []string{"metadata"})
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}
			if len(resp.Results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(resp.Results))
			}

			meta := resp.Results[0].GetMetadata()

			if tc.checkKey == "tags" {
				arr, ok := meta["tags"].([]interface{})
				if !ok {
					t.Fatalf("Expected array for 'tags', got %T", meta["tags"])
				}
				if len(arr) != 50 {
					t.Errorf("Array truncated: expected 50 elements, got %d", len(arr))
				}
			} else if tc.checkVal != nil {
				if meta[tc.checkKey] != tc.checkVal {
					got := meta[tc.checkKey]
					if s, ok := got.(string); ok && len(s) > 50 {
						got = s[:50] + "..."
					}
					t.Errorf("Metadata '%s': expected '%v', got '%v'", tc.checkKey, tc.checkVal, got)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Demo API Key
// ---------------------------------------------------------------------------

func TestGetDemoAPIKey(t *testing.T) {
	// Catches: demo key generation endpoint broken, returned key not usable.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	apiKey, err := cyborgdb.GetDemoAPIKey("")
	if err != nil {
		t.Fatalf("Failed to get demo API key: %v", err)
	}
	if apiKey == "" {
		t.Fatal("Demo API key is empty")
	}

	// Verify the key actually works
	client, err := cyborgdb.NewClient("http://localhost:8000", apiKey)
	if err != nil {
		t.Fatalf("Failed to create client with demo key: %v", err)
	}

	health, err := client.GetHealth(ctx)
	if err != nil {
		t.Fatalf("Health check failed with demo key: %v", err)
	}
	if health == nil {
		t.Fatal("Health response is nil")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func strPtr(s string) *string { return &s }

func generateVectorWithValue(dimension int, value float32) []float32 {
	vector := make([]float32, dimension)
	for i := range vector {
		vector[i] = value
	}
	return vector
}

func generateMixedVector(dimension int) []float32 {
	vector := make([]float32, dimension)
	for i := range vector {
		if i%2 == 0 {
			vector[i] = float32(i) / 100.0
		} else {
			vector[i] = -float32(i) / 100.0
		}
	}
	return vector
}

func createDeepNestedMetadata(depth int) map[string]interface{} {
	if depth <= 0 {
		return map[string]interface{}{"value": "leaf"}
	}
	return map[string]interface{}{
		"nested": createDeepNestedMetadata(depth - 1),
		"level":  depth,
	}
}

func makeIntSlice(n int) []interface{} {
	s := make([]interface{}, n)
	for i := range s {
		s[i] = float64(i)
	}
	return s
}
