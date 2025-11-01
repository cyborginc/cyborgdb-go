package test

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/joho/godotenv"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
)

// Test configuration
const (
	maxRetries         = 3
	baseTimeout        = 10 * time.Second
	propagationTimeout = 15 * time.Second
	longTimeout        = 120 * time.Second
)

var (
	// ErrAPIKeyRequired is returned when the API key is not set
	ErrAPIKeyRequired = errors.New("CYBORGDB_API_KEY environment variable is required")
)

func generateRandomKey() []byte {
	key := make([]byte, 32)
	rand.Read(key)
	return key
}

// Create a CyborgDB client with proper error handling
func createClient() (*cyborgdb.Client, error) {
	apiKey := os.Getenv("CYBORGDB_API_KEY")
	if apiKey == "" {
		return nil, ErrAPIKeyRequired
	}
	return cyborgdb.NewClient("http://localhost:8000", apiKey)
}

// Wait for operations to propagate with timeout
func waitForPropagation(duration time.Duration) {
	time.Sleep(duration)
}

func TestMain(m *testing.M) {
	// Load environment variables
	godotenv.Load("../.env.local")

	// Validate test environment
	if os.Getenv("CYBORGDB_API_KEY") == "" {
		fmt.Println("ERROR: CYBORGDB_API_KEY environment variable is required for testing")
		os.Exit(1)
	}

	// Run tests
	code := m.Run()
	os.Exit(code)
}

// SSL/TLS Configuration Testing
func TestSSLVerification(t *testing.T) {
	t.Run("TestSSLAutoDetectionLocalhost", func(t *testing.T) {
		client, err := cyborgdb.NewClient("http://localhost:8000", os.Getenv("CYBORGDB_API_KEY"))
		if err != nil {
			t.Fatalf("Failed to create client with localhost URL: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), baseTimeout)
		defer cancel()

		_, err = client.GetHealth(ctx)
		if err != nil {
			t.Errorf("Health check failed: %v", err)
		}
	})

	t.Run("TestSSLWithHTTPSLocalhost", func(t *testing.T) {
		// First check if HTTPS is available on localhost:8000
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		testClient := &http.Client{
			Transport: tr,
			Timeout:   2 * time.Second,
		}

		// Use context for the request
		ctx := context.Background()
		req, reqErr := http.NewRequestWithContext(ctx, "GET", "https://localhost:8000/v1/health", nil)
		if reqErr != nil {
			t.Fatalf("Failed to create request: %v", reqErr)
		}

		resp, doErr := testClient.Do(req)
		if doErr != nil {
			errorStr := strings.ToLower(doErr.Error())
			// If server gave HTTP response to HTTPS client, HTTPS is not available
			if strings.Contains(errorStr, "http response to https client") ||
				strings.Contains(errorStr, "connection refused") {
				t.Skip("HTTPS not available on localhost:8000 - skipping HTTPS test")
			}
		}
		if resp != nil {
			resp.Body.Close()
		}

		// If we get here, HTTPS is available, so test it
		client, clientErr := cyborgdb.NewClient("https://localhost:8000", os.Getenv("CYBORGDB_API_KEY"))
		if clientErr != nil {
			t.Fatalf("Failed to create client with HTTPS localhost URL: %v", clientErr)
		}

		httpsCtx, cancel := context.WithTimeout(context.Background(), baseTimeout)
		defer cancel()

		_, healthErr := client.GetHealth(httpsCtx)
		if healthErr != nil {
			t.Errorf("HTTPS health check failed: %v", healthErr)
		}
	})

	t.Run("TestExplicitSSLConfiguration", func(t *testing.T) {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient := &http.Client{Transport: tr}

		// httpClient is always non-nil after initialization
		if httpClient.Transport == nil {
			t.Error("Failed to create HTTP client with SSL configuration")
		}

		strictTr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		}
		strictClient := &http.Client{Transport: strictTr}

		// strictClient is always non-nil after initialization
		if strictClient.Transport == nil {
			t.Error("Failed to create HTTP client with strict SSL")
		}
	})
}

// Index Type Testing - IVF and IVFPQ
func TestIndexTypes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), longTimeout)
	defer cancel()

	client, err := createClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	dimension := int32(128)

	t.Run("TestIVFIndexOperations", func(t *testing.T) {
		indexName := generateUniqueName("ivf_test_")
		indexKey := generateRandomKey()

		indexConfig := cyborgdb.IndexIVF(dimension)
		metric := "euclidean"

		createParams := &cyborgdb.CreateIndexParams{
			IndexName:   indexName,
			IndexKey:    indexKey,
			IndexConfig: indexConfig,
			Metric:      &metric,
		}

		index, createErr := client.CreateIndex(ctx, createParams)
		if createErr != nil {
			t.Fatalf("Failed to create IVF index: %v", createErr)
		}
		defer func() {
			if delErr := index.DeleteIndex(ctx); delErr != nil {
				t.Logf("Warning: Failed to cleanup index: %v", delErr)
			}
		}()

		indexType := index.GetIndexType()
		if indexType != "ivf" {
			t.Errorf("Expected index type 'ivf', got '%s'", indexType)
		}

		testVectors := generateTestVectors(10, int(dimension))
		items := make([]cyborgdb.VectorItem, len(testVectors))
		for i, vector := range testVectors {
			items[i] = cyborgdb.VectorItem{
				Id:       fmt.Sprintf("ivf_%d", i),
				Vector:   vector,
				Metadata: map[string]interface{}{"test_id": i},
			}
		}

		upsertErr := index.Upsert(ctx, items)
		if upsertErr != nil {
			t.Fatalf("Failed to upsert to IVF index: %v", upsertErr)
		}

		waitForPropagation(2 * time.Second)

		queryParams := cyborgdb.QueryParams{
			QueryVector: testVectors[0],
			TopK:        5,
		}

		results, queryErr := index.Query(ctx, queryParams)
		if queryErr != nil {
			t.Fatalf("Failed to query IVF index: %v", queryErr)
		}

		if results == nil {
			t.Fatal("Query results must not be nil")
		}
	})

	t.Run("TestIVFPQIndexOperations", func(t *testing.T) {
		indexName := generateUniqueName("ivfpq_test_")
		indexKey := generateRandomKey()

		indexConfig := cyborgdb.IndexIVFPQ(dimension, 32, 8)
		metric := "euclidean"

		createParams := &cyborgdb.CreateIndexParams{
			IndexName:   indexName,
			IndexKey:    indexKey,
			IndexConfig: indexConfig,
			Metric:      &metric,
		}

		index, createErr := client.CreateIndex(ctx, createParams)
		if createErr != nil {
			t.Fatalf("Failed to create IVFPQ index: %v", createErr)
		}
		defer func() {
			if delErr := index.DeleteIndex(ctx); delErr != nil {
				t.Logf("Warning: Failed to cleanup index: %v", delErr)
			}
		}()

		if index.GetIndexType() != "ivfpq" {
			t.Errorf("Expected index type 'ivfpq', got '%s'", index.GetIndexType())
		}

		testVectors := generateTestVectors(50, int(dimension))
		items := make([]cyborgdb.VectorItem, len(testVectors))
		for i, vector := range testVectors {
			items[i] = cyborgdb.VectorItem{
				Id:       fmt.Sprintf("ivfpq_%d", i),
				Vector:   vector,
				Metadata: map[string]interface{}{"test_id": i},
			}
		}

		upsertErr := index.Upsert(ctx, items)
		if upsertErr != nil {
			t.Fatalf("Failed to upsert to IVFPQ index: %v", upsertErr)
		}

		waitForPropagation(3 * time.Second)

		queryParams := cyborgdb.QueryParams{
			QueryVector: testVectors[0],
			TopK:        5,
		}

		// Use a longer timeout for IVFPQ queries
		queryCtx, queryCancel := context.WithTimeout(ctx, 60*time.Second)
		defer queryCancel()

		results, queryErr := index.Query(queryCtx, queryParams)
		if queryErr != nil {
			t.Fatalf("Failed to query IVFPQ index: %v", queryErr)
		}

		if results == nil {
			t.Fatal("Query results must not be nil")
		}
	})
}

// Error Handling Testing
func TestComprehensiveErrorHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("TestInvalidAPIKey", func(t *testing.T) {
		client, clientErr := cyborgdb.NewClient("http://localhost:8000", "definitely-invalid-key-12345")
		if clientErr != nil {
			t.Fatalf("Client creation should not fail with invalid API key: %v", clientErr)
		}

		// Try to create an index - this should require authentication
		indexConfig := cyborgdb.IndexIVFFlat(128)
		metric := "euclidean"
		createParams := &cyborgdb.CreateIndexParams{
			IndexName:   generateUniqueName("auth_test_"),
			IndexKey:    generateRandomKey(),
			IndexConfig: indexConfig,
			Metric:      &metric,
		}

		_, createErr := client.CreateIndex(ctx, createParams)
		if createErr == nil {
			t.Fatal("Invalid API key was accepted - authentication is not working")
		}

		errorStr := strings.ToLower(createErr.Error())
		authErrors := []string{"unauthorized", "401", "forbidden", "403", "invalid", "key", "auth"}
		hasAuthError := false
		for _, authErr := range authErrors {
			if strings.Contains(errorStr, authErr) {
				hasAuthError = true
				break
			}
		}

		if !hasAuthError {
			t.Errorf("Expected authentication error for invalid API key, got: %v", createErr)
		}
	})

	t.Run("TestMalformedRequests", func(t *testing.T) {
		client, clientErr := createClient()
		if clientErr != nil {
			t.Fatalf("Failed to create client: %v", clientErr)
		}

		// Test invalid dimension
		invalidConfig := cyborgdb.IndexIVFFlat(-1)
		metric := "euclidean"
		createParams := &cyborgdb.CreateIndexParams{
			IndexName:   generateUniqueName("invalid_dim_"),
			IndexKey:    generateRandomKey(),
			IndexConfig: invalidConfig,
			Metric:      &metric,
		}

		_, createErr := client.CreateIndex(ctx, createParams)
		if createErr == nil {
			t.Error("Server accepted negative dimension")
		}

		// Test invalid metric
		validConfig := cyborgdb.IndexIVFFlat(128)
		invalidMetric := "completely_invalid_metric"
		invalidParams := &cyborgdb.CreateIndexParams{
			IndexName:   generateUniqueName("invalid_metric_"),
			IndexKey:    generateRandomKey(),
			IndexConfig: validConfig,
			Metric:      &invalidMetric,
		}

		_, metricErr := client.CreateIndex(ctx, invalidParams)
		if metricErr == nil {
			t.Error("Server accepted invalid metric")
		}

		// Test empty index name
		metric2 := "euclidean"
		emptyNameParams := &cyborgdb.CreateIndexParams{
			IndexName:   "",
			IndexKey:    generateRandomKey(),
			IndexConfig: cyborgdb.IndexIVFFlat(128),
			Metric:      &metric2,
		}

		_, emptyErr := client.CreateIndex(ctx, emptyNameParams)
		if emptyErr == nil {
			t.Error("Server accepted empty index name")
		}

		// Test invalid key length
		shortKey := make([]byte, 8)
		metric3 := "euclidean"
		shortKeyParams := &cyborgdb.CreateIndexParams{
			IndexName:   generateUniqueName("short_key_"),
			IndexKey:    shortKey,
			IndexConfig: cyborgdb.IndexIVFFlat(128),
			Metric:      &metric3,
		}

		_, keyErr := client.CreateIndex(ctx, shortKeyParams)
		if keyErr == nil {
			t.Error("Server accepted invalid key length")
		}
	})

	t.Run("TestVectorDimensionValidation", func(t *testing.T) {
		client, clientErr := createClient()
		if clientErr != nil {
			t.Fatalf("Failed to create client: %v", clientErr)
		}

		indexConfig := cyborgdb.IndexIVFFlat(128)
		indexName := generateUniqueName("dim_validation_")
		indexKey := generateRandomKey()
		metric := "euclidean"

		createParams := &cyborgdb.CreateIndexParams{
			IndexName:   indexName,
			IndexKey:    indexKey,
			IndexConfig: indexConfig,
			Metric:      &metric,
		}

		index, createErr := client.CreateIndex(ctx, createParams)
		if createErr != nil {
			t.Fatalf("Failed to create test index: %v", createErr)
		}
		defer index.DeleteIndex(ctx)

		testCases := []struct {
			name       string
			dimension  int
			shouldFail bool
		}{
			{"Wrong dimension (64)", 64, true},
			{"Wrong dimension (256)", 256, true},
			{"Empty vector", 0, true},
			{"Correct dimension", 128, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				vector := make([]float32, tc.dimension)
				for i := range vector {
					vector[i] = float32(i) / 100.0
				}

				items := []cyborgdb.VectorItem{{
					Id:       fmt.Sprintf("test_%s", strings.ReplaceAll(tc.name, " ", "_")),
					Vector:   vector,
					Metadata: map[string]interface{}{},
				}}

				upsertErr := index.Upsert(ctx, items)

				if tc.shouldFail && upsertErr == nil {
					t.Errorf("Server accepted vector with %s", tc.name)
				} else if !tc.shouldFail && upsertErr != nil {
					t.Errorf("Expected success for %s, but got error: %v", tc.name, upsertErr)
				}
			})
		}
	})

	t.Run("TestNetworkConnectivity", func(t *testing.T) {
		client, clientErr := cyborgdb.NewClient("http://non-existent-server-12345.invalid:8000", "test-key")
		if clientErr != nil {
			t.Fatalf("Client creation should not fail: %v", clientErr)
		}

		networkCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, healthErr := client.GetHealth(networkCtx)
		if healthErr == nil {
			t.Error("Expected network connectivity error for non-existent server")
		}
	})
}

// Edge Cases and Boundary Conditions
func TestEdgeCasesStrict(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), longTimeout)
	defer cancel()

	client, err := createClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	indexConfig := cyborgdb.IndexIVFFlat(128)
	indexName := generateUniqueName("edge_test_")
	indexKey := generateRandomKey()
	metric := "euclidean"

	createParams := &cyborgdb.CreateIndexParams{
		IndexName:   indexName,
		IndexKey:    indexKey,
		IndexConfig: indexConfig,
		Metric:      &metric,
	}

	index, err := client.CreateIndex(ctx, createParams)
	if err != nil {
		t.Fatalf("Failed to create test index: %v", err)
	}
	defer index.DeleteIndex(ctx)

	t.Run("TestEmptyIndexQuery", func(t *testing.T) {
		queryVector := generateTestVectors(1, 128)[0]
		queryParams := cyborgdb.QueryParams{
			QueryVector: queryVector,
			TopK:        10,
		}

		results, queryErr := index.Query(ctx, queryParams)
		if queryErr != nil {
			t.Fatalf("Failed to query empty index: %v", queryErr)
		}

		if results == nil {
			t.Fatal("Results must not be nil, even for empty index")
		}
	})

	t.Run("TestDataIntegrityThroughOperations", func(t *testing.T) {
		originalVector := generateTestVectors(1, 128)[0]
		originalMetadata := map[string]interface{}{
			"test_key": "test_value",
			"number":   42,
			"array":    []int{1, 2, 3, 4, 5},
		}

		items := []cyborgdb.VectorItem{{
			Id:       "integrity_test",
			Vector:   originalVector,
			Metadata: originalMetadata,
		}}

		upsertErr := index.Upsert(ctx, items)
		if upsertErr != nil {
			t.Fatalf("Failed to upsert: %v", upsertErr)
		}

		waitForPropagation(2 * time.Second)

		include := []string{"vector", "metadata"}
		results, getErr := index.Get(ctx, []string{"integrity_test"}, include)
		if getErr != nil {
			t.Fatalf("Failed to get vector: %v", getErr)
		}

		if len(results.Results) != 1 {
			t.Fatalf("Expected exactly 1 result, got %d", len(results.Results))
		}

		retrieved := results.Results[0]
		if retrieved.GetId() != "integrity_test" {
			t.Errorf("ID mismatch: expected 'integrity_test', got '%s'", retrieved.GetId())
		}

		retrievedVector := retrieved.GetVector()
		if len(retrievedVector) != len(originalVector) {
			t.Fatalf("Vector length mismatch: expected %d, got %d", len(originalVector), len(retrievedVector))
		}

		for i, expected := range originalVector {
			actual := retrievedVector[i]
			diff := actual - expected
			if diff < 0 {
				diff = -diff
			}
			if diff > 1e-6 {
				t.Errorf("Vector data corruption at index %d: expected %f, got %f", i, expected, actual)
			}
		}

		retrievedMetadata := retrieved.GetMetadata()
		if retrievedMetadata["test_key"] != originalMetadata["test_key"] {
			t.Errorf("Metadata corruption: test_key expected %v, got %v",
				originalMetadata["test_key"], retrievedMetadata["test_key"])
		}
	})

	t.Run("TestConcurrentOperationsValidation", func(t *testing.T) {
		numOperations := 20
		var wg sync.WaitGroup
		errorChan := make(chan error, numOperations)
		successChan := make(chan string, numOperations)

		// Use a longer timeout for concurrent operations
		concurrentCtx, concurrentCancel := context.WithTimeout(ctx, 90*time.Second)
		defer concurrentCancel()

		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				vector := generateTestVectors(1, 128)[0]
				for j := range vector {
					vector[j] += float32(id) / 1000.0
				}

				items := []cyborgdb.VectorItem{{
					Id:       fmt.Sprintf("concurrent_%d", id),
					Vector:   vector,
					Metadata: map[string]interface{}{"batch_id": id},
				}}

				if upsertErr := index.Upsert(concurrentCtx, items); upsertErr != nil {
					errorChan <- fmt.Errorf("operation %d failed: %w", id, upsertErr)
				} else {
					successChan <- fmt.Sprintf("concurrent_%d", id)
				}
			}(i)
		}

		wg.Wait()
		close(errorChan)
		close(successChan)

		var errs []error
		for opErr := range errorChan {
			errs = append(errs, opErr)
		}

		var successIds []string
		for id := range successChan {
			successIds = append(successIds, id)
		}

		if len(errs) > 0 {
			for _, opErr := range errs {
				t.Errorf("Concurrent operation error: %v", opErr)
			}
		}

		if len(successIds) != numOperations {
			t.Errorf("Expected %d successful operations, got %d", numOperations, len(successIds))
		}

		waitForPropagation(5 * time.Second)

		results, listErr := index.ListIDs(concurrentCtx)
		if listErr != nil {
			t.Fatalf("Failed to list IDs: %v", listErr)
		}

		concurrentCount := 0
		for _, id := range results.Ids {
			if strings.HasPrefix(id, "concurrent_") {
				concurrentCount++
			}
		}

		if concurrentCount != numOperations {
			t.Errorf("Expected %d items in index, found %d", numOperations, concurrentCount)
		}
	})

	t.Run("TestBoundaryValues", func(t *testing.T) {
		testCases := []struct {
			name          string
			vector        []float32
			shouldSucceed bool
		}{
			{"Zero vector", make([]float32, 128), true},
			{"Very small values", generateVectorWithValue(128, 1e-10), true},
			{"Very large values", generateVectorWithValue(128, 1e10), true},
			{"Mixed positive negative", generateMixedVector(128), true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				items := []cyborgdb.VectorItem{{
					Id:       fmt.Sprintf("boundary_%s", strings.ReplaceAll(tc.name, " ", "_")),
					Vector:   tc.vector,
					Metadata: map[string]interface{}{"type": tc.name},
				}}

				upsertErr := index.Upsert(ctx, items)

				if tc.shouldSucceed && upsertErr != nil {
					t.Errorf("Expected success for %s, got error: %v", tc.name, upsertErr)
				} else if !tc.shouldSucceed && upsertErr == nil {
					t.Errorf("Expected failure for %s, but operation succeeded", tc.name)
				}
			})
		}
	})

	t.Run("TestLargeMetadataHandling", func(t *testing.T) {
		testCases := []struct {
			name     string
			metadata map[string]interface{}
		}{
			{"Large string", map[string]interface{}{"description": strings.Repeat("A", 1000)}},
			{"Deep nesting", createDeepNestedMetadata(5)},
			{"Large array", map[string]interface{}{"array": make([]int, 50)}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				vector := generateTestVectors(1, 128)[0]
				items := []cyborgdb.VectorItem{{
					Id:       fmt.Sprintf("metadata_%s", strings.ReplaceAll(tc.name, " ", "_")),
					Vector:   vector,
					Metadata: tc.metadata,
				}}

				metadataErr := index.Upsert(ctx, items)
				if metadataErr != nil {
					t.Errorf("Failed to upsert %s: %v", tc.name, metadataErr)
					return
				}

				waitForPropagation(2 * time.Second)
				include := []string{"metadata"}
				results, getErr := index.Get(ctx, []string{items[0].Id}, include)
				if getErr != nil {
					t.Errorf("Failed to retrieve %s: %v", tc.name, getErr)
					return
				}

				if results == nil {
					t.Errorf("Results is nil for %s", tc.name)
					return
				}

				if len(results.Results) != 1 {
					t.Errorf("Expected 1 result for %s, got %d", tc.name, len(results.Results))
				}
			})
		}
	})
}

// Backend Compatibility Tests
func TestBackendCompatibility(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), longTimeout)
	defer cancel()

	client, err := createClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("TestHealthCheck", func(t *testing.T) {
		health, healthErr := client.GetHealth(ctx)
		if healthErr != nil {
			t.Fatalf("Health check failed: %v", healthErr)
		}

		if health == nil {
			t.Fatal("Health response must not be nil")
		}
	})

	t.Run("TestBasicIndexSupport", func(t *testing.T) {
		indexConfig := cyborgdb.IndexIVFFlat(128)
		indexName := generateUniqueName("compatibility_")
		indexKey := generateRandomKey()
		metric := "euclidean"

		createParams := &cyborgdb.CreateIndexParams{
			IndexName:   indexName,
			IndexKey:    indexKey,
			IndexConfig: indexConfig,
			Metric:      &metric,
		}

		index, createErr := client.CreateIndex(ctx, createParams)
		if createErr != nil {
			t.Fatalf("Basic IVFFlat index creation failed: %v", createErr)
		}
		defer index.DeleteIndex(ctx)

		vector := generateTestVectors(1, 128)[0]
		items := []cyborgdb.VectorItem{{
			Id:     "compatibility_test",
			Vector: vector,
		}}

		if upsertErr := index.Upsert(ctx, items); upsertErr != nil {
			t.Fatalf("Basic upsert failed: %v", upsertErr)
		}

		waitForPropagation(2 * time.Second)

		queryParams := cyborgdb.QueryParams{
			QueryVector: vector,
			TopK:        1,
		}

		results, queryErr := index.Query(ctx, queryParams)
		if queryErr != nil {
			t.Fatalf("Basic query failed: %v", queryErr)
		}

		if results == nil {
			t.Fatal("Query results must not be nil")
		}
	})

	t.Run("TestAdvancedIndexSupport", func(t *testing.T) {
		dimension := int32(128)

		indexConfig := cyborgdb.IndexIVFPQ(dimension, 32, 8)
		metric := "euclidean"
		createParams := &cyborgdb.CreateIndexParams{
			IndexName:   generateUniqueName("advanced_"),
			IndexKey:    generateRandomKey(),
			IndexConfig: indexConfig,
			Metric:      &metric,
		}

		advancedIndex, createErr := client.CreateIndex(ctx, createParams)
		if createErr != nil {
			t.Fatalf("Failed to create advanced index: %v", createErr)
		}
		defer advancedIndex.DeleteIndex(ctx)

		vectors := generateTestVectors(100, 128)
		items := make([]cyborgdb.VectorItem, len(vectors))
		for i, vector := range vectors {
			items[i] = cyborgdb.VectorItem{
				Id:     fmt.Sprintf("advanced_%d", i),
				Vector: vector,
			}
		}

		if upsertErr := advancedIndex.Upsert(ctx, items); upsertErr != nil {
			t.Errorf("Advanced index upsert failed: %v", upsertErr)
		}

		waitForPropagation(5 * time.Second)

		queryParams := cyborgdb.QueryParams{
			QueryVector: vectors[0],
			TopK:        5,
		}

		results, queryErr := advancedIndex.Query(ctx, queryParams)
		if queryErr != nil {
			t.Errorf("Advanced index query failed: %v", queryErr)
		}

		if results == nil {
			t.Error("Advanced index query results must not be nil")
		}
	})
}

// Helper functions
func generateTestVectors(count, dimension int) [][]float32 {
	vectors := make([][]float32, count)
	for i := 0; i < count; i++ {
		vectors[i] = make([]float32, dimension)
		for j := 0; j < dimension; j++ {
			vectors[i][j] = float32(i*dimension+j) / 1000.0
		}
	}
	return vectors
}

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

// TestGetDemoAPIKey tests the GetDemoAPIKey function
func TestGetDemoAPIKey(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("TestGetDemoAPIKeySuccess", func(t *testing.T) {
		// Get a demo API key
		apiKey, err := cyborgdb.GetDemoAPIKey("")
		if err != nil {
			t.Fatalf("Failed to get demo API key: %v", err)
		}

		if apiKey == "" {
			t.Fatal("Demo API key is empty")
		}

		// Verify the API key works by creating a client and checking health
		client, err := cyborgdb.NewClient("http://localhost:8000", apiKey)
		if err != nil {
			t.Fatalf("Failed to create client with demo API key: %v", err)
		}

		health, err := client.GetHealth(ctx)
		if err != nil {
			t.Fatalf("Health check failed with demo API key: %v", err)
		}

		if health == nil {
			t.Fatal("Health response is nil")
		}
	})

	t.Run("TestGetDemoAPIKeyWithCustomDescription", func(t *testing.T) {
		// Get a demo API key with a custom description
		customDescription := "Test API key for Go SDK"
		apiKey, err := cyborgdb.GetDemoAPIKey(customDescription)
		if err != nil {
			t.Fatalf("Failed to get demo API key with custom description: %v", err)
		}

		if apiKey == "" {
			t.Fatal("Demo API key is empty")
		}

		// Verify the API key works
		client, err := cyborgdb.NewClient("http://localhost:8000", apiKey)
		if err != nil {
			t.Fatalf("Failed to create client with demo API key: %v", err)
		}

		health, err := client.GetHealth(ctx)
		if err != nil {
			t.Fatalf("Health check failed with demo API key: %v", err)
		}

		if health == nil {
			t.Fatal("Health response is nil")
		}
	})
}
