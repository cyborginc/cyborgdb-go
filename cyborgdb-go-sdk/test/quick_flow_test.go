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

