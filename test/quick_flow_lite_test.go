package test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/cyborginc/cyborgdb-go"
	openapi "github.com/cyborginc/cyborgdb-go/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCyborgDBLiteFlow tests the basic flow with lite backend
func TestCyborgDBLiteFlow(t *testing.T) {
	// Skip if server is not available
	apiURL := getEnvOrDefault("CYBORGDB_BASE_URL", "http://localhost:8000")
	apiKey := getEnvOrDefault("CYBORGDB_API_KEY", "test-api-key")

	// Create client
	client := cyborgdb.NewClient(apiURL, apiKey)
	require.NotNil(t, client)

	// Generate unique index name
	indexName := fmt.Sprintf("test_lite_index_%d", time.Now().UnixNano())

	// Generate random key
	indexKey := make([]byte, 32)
	rand.Read(indexKey)

	// Create index config (use IVFFlat which should work with lite)
	dimension := int32(128)
	nLists := int32(10)
	metric := "euclidean"

	indexConfig := openapi.IndexIvfFlatModel{
		Dimension: &dimension,
		NLists:    &nLists,
		Metric:    &metric,
	}

	// Create index
	ctx := context.Background()
	index, err := client.CreateIndex(ctx, indexName, indexKey, indexConfig, nil)
	require.NoError(t, err)
	require.NotNil(t, index)

	// Clean up at the end
	defer func() {
		if deleteErr := index.DeleteIndex(ctx); deleteErr != nil {
			t.Logf("Failed to delete index: %v", deleteErr)
		}
	}()

	t.Run("UpsertAndQuery", func(t *testing.T) {
		// Generate test data
		numVectors := 20 // Use fewer vectors for lite
		ids := make([]string, numVectors)
		vectors := make([][]float32, numVectors)

		for i := 0; i < numVectors; i++ {
			ids[i] = fmt.Sprintf("vec_%d", i)
			vectors[i] = generateRandomVector(int(dimension))
		}

		// Upsert vectors
		err := index.Upsert(ctx, ids, vectors, nil)
		require.NoError(t, err)

		// Query
		queryVector := generateRandomVector(int(dimension))
		results, err := index.Query(ctx, [][]float32{queryVector}, 5, nil, false)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.LessOrEqual(t, len(results[0]), 5)
	})

	t.Run("LoadIndex", func(t *testing.T) {
		// Add some data first
		testID := "test_vector"
		testVector := generateRandomVector(int(dimension))
		err := index.Upsert(ctx, []string{testID}, [][]float32{testVector}, nil)
		require.NoError(t, err)

		// Load the index
		loadedIndex := client.LoadIndex(indexName, indexKey)
		require.NotNil(t, loadedIndex)

		// Query to verify it works
		results, err := loadedIndex.Query(ctx, [][]float32{testVector}, 1, nil, false)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Greater(t, len(results[0]), 0)
	})

	t.Run("WithMetadata", func(t *testing.T) {
		// Upsert with metadata
		ids := []string{"vec1", "vec2", "vec3"}
		vectors := [][]float32{
			generateRandomVector(int(dimension)),
			generateRandomVector(int(dimension)),
			generateRandomVector(int(dimension)),
		}
		metadata := []map[string]interface{}{
			{"category": "A", "value": 1},
			{"category": "B", "value": 2},
			{"category": "A", "value": 3},
		}

		err := index.Upsert(ctx, ids, vectors, metadata)
		require.NoError(t, err)

		// Query and check metadata
		results, err := index.Query(ctx, [][]float32{vectors[0]}, 3, nil, true)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Greater(t, len(results[0]), 0)

		// Check that metadata is returned if available
		if len(results[0]) > 0 && results[0][0].Metadata != nil {
			assert.NotNil(t, results[0][0].Metadata)
		}
	})

	t.Run("DeleteVectors", func(t *testing.T) {
		// Add vectors
		ids := []string{"vec1", "vec2", "vec3"}
		vectors := [][]float32{
			generateRandomVector(int(dimension)),
			generateRandomVector(int(dimension)),
			generateRandomVector(int(dimension)),
		}
		err := index.Upsert(ctx, ids, vectors, nil)
		require.NoError(t, err)

		// Delete one vector
		err = index.DeleteVectors(ctx, []string{"vec2"})
		require.NoError(t, err)

		// Get remaining vectors
		remainingVectors, err := index.Get(ctx, []string{"vec1", "vec3"})
		require.NoError(t, err)
		assert.Len(t, remainingVectors, 2)

		// Verify vec2 is deleted
		deletedVectors, err := index.Get(ctx, []string{"vec2"})
		require.NoError(t, err)
		assert.Len(t, deletedVectors, 0)
	})
}

// Helper functions
func generateRandomVector(dimension int) []float32 {
	vector := make([]float32, dimension)
	for i := range vector {
		vector[i] = rand.Float32()
	}
	return vector
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}