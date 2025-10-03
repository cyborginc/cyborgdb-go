package test

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
)

// Test configuration
const (
	maxRetries = 3
	baseTimeout = 10 * time.Second
	propagationTimeout = 15 * time.Second
	longTimeout = 120 * time.Second
)

// Test setup helpers
func generateUniqueName(prefix string) string {
	if prefix == "" {
		prefix = "test_"
	}
	return fmt.Sprintf("%s%s", prefix, uuid.New().String())
}

func generateRandomKey() []byte {
	key := make([]byte, 32)
	rand.Read(key)
	return key
}

// Create a CyborgDB client with proper error handling
func createClient() (*cyborgdb.Client, error) {
	apiKey := os.Getenv("CYBORGDB_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("CYBORGDB_API_KEY environment variable is required")
	}
	return cyborgdb.NewClient("http://localhost:8000", apiKey)
}

// Wait for operations to propagate with timeout
func waitForPropagation(duration time.Duration) {
	time.Sleep(duration)
}

// Retry helper for operations that might need time to propagate
func retryOperation(operation func() error, maxRetries int, delay time.Duration) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := operation(); err != nil {
			lastErr = err
			if i < maxRetries-1 {
				time.Sleep(delay)
			}
		} else {
			return nil
		}
	}
	return lastErr
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
		
		resp, err := testClient.Get("https://localhost:8000/v1/health")
		if err != nil {
			errorStr := strings.ToLower(err.Error())
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
		client, err := cyborgdb.NewClient("https://localhost:8000", os.Getenv("CYBORGDB_API_KEY"))
		if err != nil {
			t.Fatalf("Failed to create client with HTTPS localhost URL: %v", err)
		}
		
		ctx, cancel := context.WithTimeout(context.Background(), baseTimeout)
		defer cancel()
		
		_, err = client.GetHealth(ctx)
		if err != nil {
			t.Errorf("HTTPS health check failed: %v", err)
		}
	})

	t.Run("TestExplicitSSLConfiguration", func(t *testing.T) {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient := &http.Client{Transport: tr}
		
		if httpClient == nil {
			t.Error("Failed to create HTTP client with SSL configuration")
		}
		
		strictTr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		}
		strictClient := &http.Client{Transport: strictTr}
		
		if strictClient == nil {
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
		
		index, err := client.CreateIndex(ctx, createParams)
		if err != nil {
			t.Fatalf("Failed to create IVF index: %v", err)
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

		err = index.Upsert(ctx, items)
		if err != nil {
			t.Fatalf("Failed to upsert to IVF index: %v", err)
		}

		waitForPropagation(2 * time.Second)

		queryParams := cyborgdb.QueryParams{
			QueryVector: testVectors[0],
			TopK:        5,
		}
		
		results, err := index.Query(ctx, queryParams)
		if err != nil {
			t.Fatalf("Failed to query IVF index: %v", err)
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
		
		index, err := client.CreateIndex(ctx, createParams)
		if err != nil {
			t.Fatalf("Failed to create IVFPQ index: %v", err)
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

		err = index.Upsert(ctx, items)
		if err != nil {
			t.Fatalf("Failed to upsert to IVFPQ index: %v", err)
		}

		waitForPropagation(3 * time.Second)

		queryParams := cyborgdb.QueryParams{
			QueryVector: testVectors[0],
			TopK:        5,
		}
		
		// Use a longer timeout for IVFPQ queries
		queryCtx, queryCancel := context.WithTimeout(ctx, 60*time.Second)
		defer queryCancel()
		
		results, err := index.Query(queryCtx, queryParams)
		if err != nil {
			t.Fatalf("Failed to query IVFPQ index: %v", err)
		}

		if results == nil {
			t.Fatal("Query results must not be nil")
		}
	})

	t.Run("TestIVFPQParameterValidation", func(t *testing.T) {
		metric := "euclidean"
		
		testCases := []struct {
			name       string
			pqDim      int32
			pqBits     int32
			shouldFail bool
			reason     string
		}{
			{"Zero pqDim", 0, 8, true, "pqDim cannot be zero"},
			{"Zero pqBits", 32, 0, true, "pqBits cannot be zero"},
			{"Negative pqDim", -1, 8, true, "pqDim cannot be negative"},
			{"Negative pqBits", 32, -1, true, "pqBits cannot be negative"},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				invalidConfig := cyborgdb.IndexIVFPQ(dimension, tc.pqDim, tc.pqBits)
				invalidParams := &cyborgdb.CreateIndexParams{
					IndexName:   generateUniqueName("invalid_"),
					IndexKey:    generateRandomKey(),
					IndexConfig: invalidConfig,
					Metric:      &metric,
				}
				
				testIndex, err := client.CreateIndex(ctx, invalidParams)
				if testIndex != nil {
					defer testIndex.DeleteIndex(ctx)
				}
				
				if tc.shouldFail && err == nil {
					t.Errorf("Expected error for %s, but creation succeeded", tc.reason)
				}
			})
		}
	})
}

// Error Handling Testing
func TestComprehensiveErrorHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("TestInvalidAPIKey", func(t *testing.T) {
		client, err := cyborgdb.NewClient("http://localhost:8000", "definitely-invalid-key-12345")
		if err != nil {
			t.Fatalf("Client creation should not fail with invalid API key: %v", err)
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
		
		_, err = client.CreateIndex(ctx, createParams)
		if err == nil {
			t.Fatal("Invalid API key was accepted - authentication is not working")
		}
		
		errorStr := strings.ToLower(err.Error())
		authErrors := []string{"unauthorized", "401", "forbidden", "403", "invalid", "key", "auth"}
		hasAuthError := false
		for _, authErr := range authErrors {
			if strings.Contains(errorStr, authErr) {
				hasAuthError = true
				break
			}
		}
		
		if !hasAuthError {
			t.Errorf("Expected authentication error for invalid API key, got: %v", err)
		}
	})

	t.Run("TestMalformedRequests", func(t *testing.T) {
		client, err := createClient()
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
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
		
		_, err = client.CreateIndex(ctx, createParams)
		if err == nil {
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
		
		_, err = client.CreateIndex(ctx, invalidParams)
		if err == nil {
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
		
		_, err = client.CreateIndex(ctx, emptyNameParams)
		if err == nil {
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
		
		_, err = client.CreateIndex(ctx, shortKeyParams)
		if err == nil {
			t.Error("Server accepted invalid key length")
		}
	})

	t.Run("TestVectorDimensionValidation", func(t *testing.T) {
		client, err := createClient()
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
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
		
		index, err := client.CreateIndex(ctx, createParams)
		if err != nil {
			t.Fatalf("Failed to create test index: %v", err)
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

				err := index.Upsert(ctx, items)
				
				if tc.shouldFail && err == nil {
					t.Errorf("Server accepted vector with %s", tc.name)
				} else if !tc.shouldFail && err != nil {
					t.Errorf("Expected success for %s, but got error: %v", tc.name, err)
				}
			})
		}
	})

	t.Run("TestNetworkConnectivity", func(t *testing.T) {
		client, err := cyborgdb.NewClient("http://non-existent-server-12345.invalid:8000", "test-key")
		if err != nil {
			t.Fatalf("Client creation should not fail: %v", err)
		}
		
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		_, err = client.GetHealth(ctx)
		if err == nil {
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
		
		results, err := index.Query(ctx, queryParams)
		if err != nil {
			t.Fatalf("Failed to query empty index: %v", err)
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
		
		err := index.Upsert(ctx, items)
		if err != nil {
			t.Fatalf("Failed to upsert: %v", err)
		}

		waitForPropagation(2 * time.Second)

		include := []string{"vector", "metadata"}
		results, err := index.Get(ctx, []string{"integrity_test"}, include)
		if err != nil {
			t.Fatalf("Failed to get vector: %v", err)
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
				
				if err := index.Upsert(concurrentCtx, items); err != nil {
					errorChan <- fmt.Errorf("operation %d failed: %v", id, err)
				} else {
					successChan <- fmt.Sprintf("concurrent_%d", id)
				}
			}(i)
		}
		
		wg.Wait()
		close(errorChan)
		close(successChan)
		
		var errors []error
		for err := range errorChan {
			errors = append(errors, err)
		}
		
		var successIds []string
		for id := range successChan {
			successIds = append(successIds, id)
		}
		
		if len(errors) > 0 {
			for _, err := range errors {
				t.Errorf("Concurrent operation error: %v", err)
			}
		}
		
		if len(successIds) != numOperations {
			t.Errorf("Expected %d successful operations, got %d", numOperations, len(successIds))
		}
		
		waitForPropagation(5 * time.Second)
		
		results, err := index.ListIDs(concurrentCtx)
		if err != nil {
			t.Fatalf("Failed to list IDs: %v", err)
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
				
				err := index.Upsert(ctx, items)
				
				if tc.shouldSucceed && err != nil {
					t.Errorf("Expected success for %s, got error: %v", tc.name, err)
				} else if !tc.shouldSucceed && err == nil {
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
				
				err := index.Upsert(ctx, items)
				if err != nil {
					t.Errorf("Failed to upsert %s: %v", tc.name, err)
					return
				}
				
				waitForPropagation(2 * time.Second)
				include := []string{"metadata"}
				results, err := index.Get(ctx, []string{items[0].Id}, include)
				if err != nil {
					t.Errorf("Failed to retrieve %s: %v", tc.name, err)
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
		health, err := client.GetHealth(ctx)
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
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
		
		index, err := client.CreateIndex(ctx, createParams)
		if err != nil {
			t.Fatalf("Basic IVFFlat index creation failed: %v", err)
		}
		defer index.DeleteIndex(ctx)
		
		vector := generateTestVectors(1, 128)[0]
		items := []cyborgdb.VectorItem{{
			Id:     "compatibility_test",
			Vector: vector,
		}}
		
		if err := index.Upsert(ctx, items); err != nil {
			t.Fatalf("Basic upsert failed: %v", err)
		}
		
		waitForPropagation(2 * time.Second)
		
		queryParams := cyborgdb.QueryParams{
			QueryVector: vector,
			TopK:        1,
		}
		
		results, err := index.Query(ctx, queryParams)
		if err != nil {
			t.Fatalf("Basic query failed: %v", err)
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
		
		advancedIndex, err := client.CreateIndex(ctx, createParams)
		if err != nil {
			t.Fatalf("Failed to create advanced index: %v", err)
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
		
		if err := advancedIndex.Upsert(ctx, items); err != nil {
			t.Errorf("Advanced index upsert failed: %v", err)
		}
		
		waitForPropagation(5 * time.Second)
		
		queryParams := cyborgdb.QueryParams{
			QueryVector: vectors[0],
			TopK:        5,
		}
		
		results, err := advancedIndex.Query(ctx, queryParams)
		if err != nil {
			t.Errorf("Advanced index query failed: %v", err)
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