package cyborgdb_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
)

func generateRandomKey(t *testing.T) []byte {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return key
}

func generateTestIndexName() string {
	random := make([]byte, 4)
	_, _ = rand.Read(random) // ignore error, test name is not security-critical
	return "test_index_" + hex.EncodeToString(random)
}

func TestGetHealth(t *testing.T) {
	apiURL := "http://localhost:8000"
	apiKey := os.Getenv("CYBORGDB_API_KEY")

	if apiURL == "" || apiKey == "" {
		t.Skip("CYBORGDB_API_URL or CYBORGDB_API_KEY environment variable not set")
	}

	client, err := cyborgdb.NewClient(apiURL, apiKey, false)
	require.NoError(t, err)

	health, err := client.GetHealth(context.Background())
	require.NoError(t, err)
	require.NotNil(t, health)
	require.NotNil(t, health.Status)
	require.Greater(t, len(*health.Status), 0)
	require.Contains(t, *health.Status, "healthy") // or "ok" or whatever your service returns
}

func TestCreateIndex_IVFPQ(t *testing.T) {
	apiURL := "http://localhost:8000"
	apiKey := os.Getenv("CYBORGDB_API_KEY")

	if apiURL == "" || apiKey == "" {
		t.Skip("CYBORGDB_API_URL or CYBORGDB_API_KEY environment variable not set")
	}

	client, err := cyborgdb.NewClient(apiURL, apiKey, false)
	require.NoError(t, err)

	indexName := generateTestIndexName()
	indexKey := generateRandomKey(t)
	dim := int32(128)

	model := &cyborgdb.IndexIVFPQModel{
		Dimension: dim,
		Metric:    "euclidean",
		NLists:    32,
		PqDim:     16,
		PqBits:    8,
	}

	resp, err := client.CreateIndex(context.Background(), indexName, indexKey, model, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, indexName, resp.GetIndexName())
	require.Equal(t, "ivfpq", resp.GetIndexType())
	cfg := resp.GetConfig()
	require.Equal(t, dim, cfg.GetDimension())
	require.Equal(t, int32(32), cfg.GetNLists())
	require.Equal(t, int32(16), cfg.GetPqDim())
	require.Equal(t, int32(8), cfg.GetPqBits())
}

func TestListIndexes(t *testing.T) {
	apiURL := "http://localhost:8000"
	apiKey := os.Getenv("CYBORGDB_API_KEY")

	if apiURL == "" || apiKey == "" {
		t.Skip("CYBORGDB_API_URL or CYBORGDB_API_KEY environment variable not set")
	}

	client, err := cyborgdb.NewClient(apiURL, apiKey, false)
	require.NoError(t, err)

	// Create a test index first to ensure at least one known index exists
	indexName := generateTestIndexName()
	indexKey := generateRandomKey(t)
	dim := int32(64)

	model := &cyborgdb.IndexIVFPQModel{
		Dimension: dim,
		Metric:    "cosine",
		NLists:    16,
		PqDim:     8,
		PqBits:    8,
	}

	createdIndex, err := client.CreateIndex(context.Background(), indexName, indexKey, model, nil)
	require.NoError(t, err)
	require.NotNil(t, createdIndex)

	// Now test ListIndexes
	indexes, err := client.ListIndexes(context.Background())
	require.NoError(t, err)
	require.NotNil(t, indexes)
	require.GreaterOrEqual(t, len(indexes), 1)
	// Confirm the test index is present
	found := false
	for _, idx := range indexes {
		if idx == indexName {
			found = true
			break
		}
	}
	require.True(t, found, "expected index %s not found in list: %v", indexName, indexes)
}

func TestDeleteIndex(t *testing.T) {
	apiURL := "http://localhost:8000"
	apiKey := os.Getenv("CYBORGDB_API_KEY")

	if apiURL == "" || apiKey == "" {
		t.Skip("CYBORGDB_API_URL or CYBORGDB_API_KEY environment variable not set")
	}

	client, err := cyborgdb.NewClient(apiURL, apiKey, false)
	require.NoError(t, err)

	indexName := generateTestIndexName()
	indexKey := generateRandomKey(t)

	model := &cyborgdb.IndexIVFPQModel{
		Dimension: 128,
		Metric:    "cosine",
		NLists:    16,
		PqDim:     8,
		PqBits:    8,
	}

	// Create the index first
	index, err := client.CreateIndex(context.Background(), indexName, indexKey, model, nil)
	require.NoError(t, err)
	require.NotNil(t, index)

	// Now attempt to delete it
	err = index.DeleteIndex(context.Background())
	require.NoError(t, err)

	// Optionally verify it's no longer in the list
	indexes, err := client.ListIndexes(context.Background())
	require.NoError(t, err)

	for _, name := range indexes {
		require.NotEqual(t, indexName, name, "deleted index %s should not appear in ListIndexes", indexName)
	}
}

func TestUpsertVectors(t *testing.T) {
	apiURL := "http://localhost:8000"
	apiKey := os.Getenv("CYBORGDB_API_KEY")

	if apiURL == "" || apiKey == "" {
		t.Skip("CYBORGDB_API_URL or CYBORGDB_API_KEY environment variable not set")
	}

	client, err := cyborgdb.NewClient(apiURL, apiKey, false)
	require.NoError(t, err)

	// Create a test index
	indexName := generateTestIndexName()
	indexKey := generateRandomKey(t)
	dim := int32(64)

	model := &cyborgdb.IndexIVFPQModel{
		Dimension: dim,
		Metric:    "cosine",
		NLists:    8,
		PqDim:     8,
		PqBits:    8,
	}

	index, err := client.CreateIndex(context.Background(), indexName, indexKey, model, nil)
	require.NoError(t, err)
	require.NotNil(t, index)

	// Generate sample vectors
	vectors := []cyborgdb.VectorItem{
		{
			Id:     "vec_1",
			Vector: make([]float32, dim),
			Metadata: map[string]interface{}{
				"type": "test", // keep flat/simple
			},
			Contents: strPtr("hello world"),
		},
		{
			Id:     "vec_2",
			Vector: make([]float32, dim),
			Metadata: map[string]interface{}{
				"category": "unit-test",
			},
			Contents: strPtr("fallback"),
		},
	}

	// Fill with dummy values
	for i := range vectors {
		for j := range vectors[i].Vector {
			vectors[i].Vector[j] = float32(j)
		}
		// Runtime check: dimension must match
		require.Equal(t, int(dim), len(vectors[i].Vector), "vector length mismatch for %s", vectors[i].Id)

		// Log vector contents
		t.Logf("Vector[%d] ID: %s | Dim: %d | Contents: %v", i, vectors[i].Id, len(vectors[i].Vector), vectors[i].Contents)
	}

	// Call Upsert
	err = index.Upsert(context.Background(), vectors)

	// If error, unwrap and show API body
	if err != nil {
		if apiErr, ok := err.(*cyborgdb.GenericOpenAPIError); ok {
			t.Logf("Raw error body:\n%s", string(apiErr.Body()))
		}
		t.Fatalf("Upsert failed: %v", err)
		t.FailNow()
	}

	t.Log("Upsert successful")
}

func TestTrainIndex(t *testing.T) {
	apiURL := "http://localhost:8000"
	apiKey := os.Getenv("CYBORGDB_API_KEY")

	if apiURL == "" || apiKey == "" {
		t.Skip("CYBORGDB_API_URL or CYBORGDB_API_KEY environment variable not set")
	}

	client, err := cyborgdb.NewClient(apiURL, apiKey, false)
	require.NoError(t, err)

	// Create a new index
	indexName := generateTestIndexName()
	indexKey := generateRandomKey(t)
	dim := int32(64)

	model := &cyborgdb.IndexIVFPQModel{
		Dimension: dim,
		Metric:    "cosine",
		NLists:    8,
		PqDim:     8,
		PqBits:    8,
	}

	index, err := client.CreateIndex(context.Background(), indexName, indexKey, model, nil)
	require.NoError(t, err)
	require.NotNil(t, index)

	// Insert a few vectors first (training requires data)
	vectors := []cyborgdb.VectorItem{}
	for i := 0; i < 10; i++ {
		vec := make([]float32, dim)
		for j := range vec {
			vec[j] = float32(j + i)
		}
		vectors = append(vectors, cyborgdb.VectorItem{
			Id:       fmt.Sprintf("vec_%d", i),
			Vector:   vec,
			Contents: strPtr(fmt.Sprintf("example %d", i)),
		})
	}
	err = index.Upsert(context.Background(), vectors)
	require.NoError(t, err, "upsert before training should succeed")

	// Call Train with explicit values
	batchSize := int32(2048)
	maxIters := int32(100)
	tolerance := 1e-6

	err = index.Train(context.Background(), batchSize, maxIters, tolerance)
	require.NoError(t, err, "training should complete without error")

	t.Logf("Training completed for index: %s", indexName)
}

func TestDeleteVectors(t *testing.T) {
	apiURL := "http://localhost:8000"
	apiKey := os.Getenv("CYBORGDB_API_KEY")

	if apiURL == "" || apiKey == "" {
		t.Skip("CYBORGDB_API_URL or CYBORGDB_API_KEY environment variable not set")
	}

	client, err := cyborgdb.NewClient(apiURL, apiKey, false)
	require.NoError(t, err)

	indexName := generateTestIndexName()
	indexKey := generateRandomKey(t)
	dim := int32(64)

	model := &cyborgdb.IndexIVFPQModel{
		Dimension: dim,
		Metric:    "cosine",
		NLists:    8,
		PqDim:     8,
		PqBits:    8,
	}

	index, err := client.CreateIndex(context.Background(), indexName, indexKey, model, nil)
	require.NoError(t, err)

	// Upsert two vectors
	vectors := []cyborgdb.VectorItem{
		{
			Id:       "vec_delete_1",
			Vector:   make([]float32, dim),
			Contents: strPtr("to be deleted"),
		},
		{
			Id:       "vec_keep_2",
			Vector:   make([]float32, dim),
			Contents: strPtr("to be kept"),
		},
	}

	for i := range vectors {
		for j := range vectors[i].Vector {
			vectors[i].Vector[j] = float32(j + i)
		}
	}

	err = index.Upsert(context.Background(), vectors)
	require.NoError(t, err, "Upsert failed before delete")

	// Delete vec_delete_1
	err = index.Delete(context.Background(), []string{"vec_delete_1"})
	require.NoError(t, err, "Delete failed")

	t.Log("Delete succeeded")
}

func strPtr(s string) *string {
	return &s
}


func TestQuery(t *testing.T) {
	apiURL := "http://localhost:8000"
	apiKey := os.Getenv("CYBORGDB_API_KEY")

	if apiURL == "" || apiKey == "" {
		t.Skip("CYBORGDB_API_URL or CYBORGDB_API_KEY environment variable not set")
	}

	client, err := cyborgdb.NewClient(apiURL, apiKey, false)
	require.NoError(t, err)

	// Create a test index
	indexName := generateTestIndexName()
	indexKey := generateRandomKey(t)
	dim := int32(64)

	model := &cyborgdb.IndexIVFPQModel{
		Dimension: dim,
		Metric:    "cosine",
		NLists:    8,
		PqDim:     8,
		PqBits:    8,
	}

	index, err := client.CreateIndex(context.Background(), indexName, indexKey, model, nil)
	require.NoError(t, err)
	require.NotNil(t, index)

	// Insert test vectors with metadata
	vectors := []cyborgdb.VectorItem{
		{
			Id:     "vec_1",
			Vector: make([]float32, dim),
			Metadata: map[string]interface{}{
				"category": "A",
				"score":    0.9,
			},
			Contents: strPtr("First test vector"),
		},
		{
			Id:     "vec_2",
			Vector: make([]float32, dim),
			Metadata: map[string]interface{}{
				"category": "B",
				"score":    0.8,
			},
			Contents: strPtr("Second test vector"),
		},
		{
			Id:     "vec_3",
			Vector: make([]float32, dim),
			Metadata: map[string]interface{}{
				"category": "A",
				"score":    0.7,
			},
			Contents: strPtr("Third test vector"),
		},
		{
			Id:     "vec_4",
			Vector: make([]float32, dim),
			Metadata: map[string]interface{}{
				"category": "B",
				"score":    0.6,
			},
			Contents: strPtr("Fourth test vector"),
		},
	}

	err = index.Upsert(context.Background(), vectors)
	require.NoError(t, err, "upsert should succeed")

	// Train the index (required for IVF indexes before querying)
	err = index.Train(context.Background(), 2048, 100, 1e-6)
	require.NoError(t, err, "training should succeed")

	// Test 1: Single vector query with default parameters
	t.Run("SingleVectorQuery", func(t *testing.T) {
		queryVector := make([]float32, dim)
		for j := range queryVector {
			queryVector[j] = float32(j)
		}
		
		result, err := index.Query(
			context.Background(),
			queryVector,      // single vector
			int32(2),         // topK
			int32(1),         // nProbes
			false,            // greedy
			nil,              // no filters
			[]string{"distance", "metadata"},
		)
		
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Results)
		require.Len(t, result.Results, 1, "single query should return 1 result set")
		require.LessOrEqual(t, len(result.Results[0]), 2, "should return at most topK results")
		
		// Check that results have the requested fields
		if len(result.Results[0]) > 0 {
			firstResult := result.Results[0][0]
			require.NotEmpty(t, firstResult.Id)
			require.NotNil(t, firstResult.Distance)
			require.NotNil(t, firstResult.Metadata)
			t.Logf("Top result: ID=%s, Distance=%f", firstResult.Id, *firstResult.Distance)
		}
	})

	// Test 2: Batch query with multiple vectors
	t.Run("BatchVectorQuery", func(t *testing.T) {
		queryVectors := [][]float32{
			make([]float32, dim),
			make([]float32, dim),
		}
		
		result, err := index.Query(
			context.Background(),
			queryVectors,     // batch of vectors
			int32(3),         // topK
			int32(2),         // nProbes
			true,             // greedy
			nil,              // no filters
			[]string{"distance", "metadata", "contents"},
		)
		
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Results)
		require.Len(t, result.Results, 2, "batch query should return 2 result sets")
		
		for i, resultSet := range result.Results {
			require.LessOrEqual(t, len(resultSet), 3, "should return at most topK results")
			t.Logf("Query %d returned %d results", i, len(resultSet))
			
		}
	})

	// Clean up
	err = index.DeleteIndex(context.Background())
	require.NoError(t, err)
}

func TestGetVectors(t *testing.T) {
	apiURL := "http://localhost:8000"
	apiKey := os.Getenv("CYBORGDB_API_KEY")

	if apiURL == "" || apiKey == "" {
		t.Skip("CYBORGDB_API_URL or CYBORGDB_API_KEY environment variable not set")
	}

	client, err := cyborgdb.NewClient(apiURL, apiKey, false)
	require.NoError(t, err)

	indexName := generateTestIndexName()
	indexKey := generateRandomKey(t)
	dim := int32(32)

	model := &cyborgdb.IndexIVFPQModel{
		Dimension: dim,
		Metric:    "cosine",
		NLists:    4,
		PqDim:     8,
		PqBits:    8,
	}

	index, err := client.CreateIndex(context.Background(), indexName, indexKey, model, nil)
	require.NoError(t, err)
	require.NotNil(t, index)

	// Insert vectors
	vectors := []cyborgdb.VectorItem{
		{
			Id:       "vec_get_1",
			Vector:   make([]float32, dim),
			Contents: strPtr("retrievable one"),
		},
		{
			Id:       "vec_get_2",
			Vector:   make([]float32, dim),
			Contents: strPtr("retrievable two"),
		},
	}
	for i := range vectors {
		for j := range vectors[i].Vector {
			vectors[i].Vector[j] = float32(i + j)
		}
	}

	err = index.Upsert(context.Background(), vectors)
	require.NoError(t, err)

	// Retrieve them using Get
	retrieved, err := index.Get(context.Background(), []string{"vec_get_1", "vec_get_2"}, []string{"contents"})
	require.NoError(t, err)
	require.Len(t, retrieved, 2)

	ids := map[string]bool{}
	for _, item := range retrieved {
		ids[item.Id] = true
		require.NotNil(t, item.Contents)
		t.Logf("Retrieved ID: %s | Contents: %s", item.Id, *item.Contents)
	}
	require.True(t, ids["vec_get_1"])
	require.True(t, ids["vec_get_2"])
}