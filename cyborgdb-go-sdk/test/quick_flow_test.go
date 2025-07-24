package cyborgdb_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

func strPtr(s string) *string {
	return &s
}
