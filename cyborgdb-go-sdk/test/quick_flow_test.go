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

func TestHealth(t *testing.T) {
	apiURL := "http://localhost:8000"
	apiKey := "cyborg_e9n8t7e6r5p4r3i2s1e0987654321abc"

	if apiURL == "" || apiKey == "" {
		t.Skip("CYBORGDB_API_URL or CYBORGDB_API_KEY environment variable not set")
	}

	client, err := cyborgdb.NewClient(apiURL, apiKey, false)
	require.NoError(t, err)

	t.Log("testing")

	resp, err := client.GetHealth(context.Background())
	require.NoError(t, err)
	require.NotNil(t, resp)
}
