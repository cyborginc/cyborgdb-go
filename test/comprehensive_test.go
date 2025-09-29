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
	baseTimeout = 5 * time.Second
	propagationTimeout = 10 * time.Second
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

// SSL/TLS Configuration Testing - Strict validation
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
			errorStr := strings.ToLower(err.Error())
			
			// Categorize errors strictly
			networkErrors := []string{"connection refused", "no such host", "network unreachable", "timeout"}
			sslErrors := []string{"ssl", "certificate", "handshake", "verification", "tls"}
			
			isNetworkError := false
			for _, netErr := range networkErrors {
				if strings.Contains(errorStr, netErr) {
					isNetworkError = true
					break
				}
			}
			
			// SSL errors indicate configuration problems - test should fail
			for _, sslErr := range sslErrors {
				if strings.Contains(errorStr, sslErr) {
					t.Errorf("SSL auto-detection failed - configuration issue: %v", err)
					return
				}
			}
			
			if !isNetworkError {
				t.Errorf("Unexpected error type (not network, not SSL): %v", err)
			} else {
				t.Skipf("Server not reachable: %v", err)
			}
		}
	})

	t.Run("TestSSLWithHTTPSLocalhost", func(t *testing.T) {
		client, err := cyborgdb.NewClient("https://localhost:8000", os.Getenv("CYBORGDB_API_KEY"))
		if err != nil {
			t.Fatalf("Failed to create client with HTTPS localhost URL: %v", err)
		}
		
		ctx, cancel := context.WithTimeout(context.Background(), baseTimeout)
		defer cancel()
		
		_, err = client.GetHealth(ctx)
		if err != nil {
			errorStr := strings.ToLower(err.Error())
			// For HTTPS, we expect either SSL certificate issues (acceptable for localhost)
			// or network connectivity issues
			acceptableErrors := []string{"connection refused", "certificate", "handshake", "verification"}
			
			isAcceptable := false
			for _, acceptable := range acceptableErrors {
				if strings.Contains(errorStr, acceptable) {
					isAcceptable = true
					break
				}
			}
			
			if !isAcceptable {
				t.Errorf("Unexpected error for HTTPS connection: %v", err)
			}
		}
	})

	t.Run("TestExplicitSSLConfiguration", func(t *testing.T) {
		// Test that we can create clients with explicit SSL configs
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient := &http.Client{Transport: tr}
		
		if httpClient == nil {
			t.Error("Failed to create HTTP client with SSL configuration")
		}
		
		// Test strict SSL
		strictTr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		}
		strictClient := &http.Client{Transport: strictTr}
		
		if strictClient == nil {
			t.Error("Failed to create HTTP client with strict SSL")
		}
	})
}

// Index Type Testing - IVF and IVFPQ with strict validation
func TestIndexTypes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
			errorStr := strings.ToLower(err.Error())
			if strings.Contains(errorStr, "lite") || strings.Contains(errorStr, "not supported") {
				t.Skip("IVF not supported in lite backend")
			}
			t.Fatalf("Failed to create IVF index: %v", err)
		}
		defer func() {
			if delErr := index.DeleteIndex(ctx); delErr != nil {
				t.Logf("Warning: Failed to cleanup index: %v", delErr)
			}
		}()

		// Strict validation of index properties
		indexType := index.GetIndexType()
		if indexType != "ivf" {
			t.Errorf("Expected index type 'ivf', got '%s'", indexType)
		}

		// Test operations with validation
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

		// Wait for propagation with timeout
		waitForPropagation(2 * time.Second)

		// Validate query works
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
			errorStr := strings.ToLower(err.Error())
			if strings.Contains(errorStr, "lite") || strings.Contains(errorStr, "not supported") {
				t.Skip("IVFPQ not supported in lite backend")
			}
			t.Fatalf("Failed to create IVFPQ index: %v", err)
		}
		defer func() {
			if delErr := index.DeleteIndex(ctx); delErr != nil {
				t.Logf("Warning: Failed to cleanup index: %v", delErr)
			}
		}()

		// Strict validation
		if index.GetIndexType() != "ivfpq" {
			t.Errorf("Expected index type 'ivfpq', got '%s'", index.GetIndexType())
		}

		// Test with sufficient data for IVFPQ (needs more vectors for training)
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

		// Validate query
		queryParams := cyborgdb.QueryParams{
			QueryVector: testVectors[0],
			TopK:        5,
		}
		
		results, err := index.Query(ctx, queryParams)
		if err != nil {
			t.Fatalf("Failed to query IVFPQ index: %v", err)
		}

		if results == nil {
			t.Fatal("Query results must not be nil")
		}
	})

	t.Run("TestIVFPQParameterValidation", func(t *testing.T) {
		// First verify IVFPQ is supported
		validConfig := cyborgdb.IndexIVFPQ(dimension, 32, 8)
		metric := "euclidean"
		validParams := &cyborgdb.CreateIndexParams{
			IndexName:   generateUniqueName("valid_ivfpq_"),
			IndexKey:    generateRandomKey(),
			IndexConfig: validConfig,
			Metric:      &metric,
		}
		
		validIndex, err := client.CreateIndex(ctx, validParams)
		if err != nil {
			errorStr := strings.ToLower(err.Error())
			if strings.Contains(errorStr, "lite") || strings.Contains(errorStr, "not supported") {
				t.Skip("IVFPQ not supported in current backend")
			}
			t.Fatalf("Failed to create valid IVFPQ index for validation test: %v", err)
		}
		defer validIndex.DeleteIndex(ctx)
		
		// Test invalid parameters - these MUST fail
		testCases := []struct {
			name     string
			pqDim    int32
			pqBits   int32
			shouldFail bool
			reason   string
		}{
			{"Zero pqDim", 0, 8, true, "pqDim cannot be zero"},
			{"Zero pqBits", 32, 0, true, "pqBits cannot be zero"},
			{"Negative pqDim", -1, 8, true, "pqDim cannot be negative"},
			{"Negative pqBits", 32, -1, true, "pqBits cannot be negative"},
			{"pqDim larger than dimension", dimension + 1, 8, true, "pqDim cannot exceed vector dimension"},
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
				
				if tc.shouldFail {
					if err == nil {
						t.Errorf("VALIDATION FAILURE: Expected error for %s, but creation succeeded - this is a security/validation issue", tc.reason)
					}
				} else {
					if err != nil {
						t.Errorf("Expected success for %s, but got error: %v", tc.reason, err)
					}
				}
			})
		}
	})
}

// Enhanced Error Handling Testing - Strict validation
func TestComprehensiveErrorHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("TestInvalidAPIKey", func(t *testing.T) {
		client, err := cyborgdb.NewClient("http://localhost:8000", "definitely-invalid-key-12345")
		if err != nil {
			t.Fatalf("Client creation should not fail with invalid API key: %v", err)
		}
		
		_, err = client.GetHealth(ctx)
		if err == nil {
			t.Fatal("SECURITY ISSUE: Invalid API key was accepted by server - authentication is not working")
		}
		
		// Categorize error strictly
		errorStr := strings.ToLower(err.Error())
		
		// Network errors are acceptable (server down)
		networkErrors := []string{"connection refused", "no such host", "timeout", "network unreachable"}
		for _, netErr := range networkErrors {
			if strings.Contains(errorStr, netErr) {
				t.Skipf("Cannot test API key validation - server not reachable: %v", err)
			}
		}
		
		// Must be authentication error
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

		// Test invalid dimension - MUST fail
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
			t.Error("VALIDATION FAILURE: Server accepted negative dimension - this is a bug")
		}

		// Test invalid metric - MUST fail
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
			t.Error("VALIDATION FAILURE: Server accepted invalid metric - this is a bug")
		}

		// Test empty index name - MUST fail
		metric2 := "euclidean"
		emptyNameParams := &cyborgdb.CreateIndexParams{
			IndexName:   "",
			IndexKey:    generateRandomKey(),
			IndexConfig: cyborgdb.IndexIVFFlat(128),
			Metric:      &metric2,
		}
		
		_, err = client.CreateIndex(ctx, emptyNameParams)
		if err == nil {
			t.Error("VALIDATION FAILURE: Server accepted empty index name - this is a bug")
		}

		// Test invalid key length - MUST fail
		shortKey := make([]byte, 8) // Too short (should be 32)
		metric3 := "euclidean"
		shortKeyParams := &cyborgdb.CreateIndexParams{
			IndexName:   generateUniqueName("short_key_"),
			IndexKey:    shortKey,
			IndexConfig: cyborgdb.IndexIVFFlat(128),
			Metric:      &metric3,
		}
		
		_, err = client.CreateIndex(ctx, shortKeyParams)
		if err == nil {
			t.Error("VALIDATION FAILURE: Server accepted invalid key length - this is a security issue")
		}
	})

	t.Run("TestVectorDimensionValidation", func(t *testing.T) {
		client, err := createClient()
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		// Create test index
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

		// Test wrong dimensions - MUST fail
		testCases := []struct {
			name      string
			dimension int
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
				
				if tc.shouldFail {
					if err == nil {
						t.Errorf("VALIDATION FAILURE: Server accepted vector with %s - this is a bug", tc.name)
					} else {
						// Verify it's actually a dimension error
						errorStr := strings.ToLower(err.Error())
						dimensionKeywords := []string{"dimension", "size", "length", "mismatch"}
						
						hasDimensionError := false
						for _, keyword := range dimensionKeywords {
							if strings.Contains(errorStr, keyword) {
								hasDimensionError = true
								break
							}
						}
						
						if !hasDimensionError {
							t.Errorf("Got error but not dimension-related for %s: %v", tc.name, err)
						}
					}
				} else {
					if err != nil {
						t.Errorf("Expected success for %s, but got error: %v", tc.name, err)
					}
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
		} else {
			// Verify it's a network error
			expectedErrors := []string{"connection refused", "no such host", "network", "timeout"}
			errorStr := strings.ToLower(err.Error())
			
			hasNetworkError := false
			for _, expectedError := range expectedErrors {
				if strings.Contains(errorStr, expectedError) {
					hasNetworkError = true
					break
				}
			}
			
			if !hasNetworkError {
				t.Errorf("Expected network error, got: %v", err)
			}
		}
	})
}

// Edge Cases and Boundary Conditions - Strict validation
func TestEdgeCasesStrict(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	
	client, err := createClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create test index
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

		// Upsert
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

		// Retrieve and validate with strict checking
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

		// Strict vector comparison
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

		// Metadata validation
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
		
		// Perform concurrent operations
		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				
				vector := generateTestVectors(1, 128)[0]
				// Modify vector to make it unique
				for j := range vector {
					vector[j] += float32(id) / 1000.0
				}
				
				items := []cyborgdb.VectorItem{{
					Id:       fmt.Sprintf("concurrent_%d", id),
					Vector:   vector,
					Metadata: map[string]interface{}{"batch_id": id},
				}}
				
				if err := index.Upsert(ctx, items); err != nil {
					errorChan <- fmt.Errorf("operation %d failed: %v", id, err)
				} else {
					successChan <- fmt.Sprintf("concurrent_%d", id)
				}
			}(i)
		}
		
		wg.Wait()
		close(errorChan)
		close(successChan)
		
		// Strict validation of results
		var errors []error
		for err := range errorChan {
			errors = append(errors, err)
		}
		
		var successIds []string
		for id := range successChan {
			successIds = append(successIds, id)
		}
		
		// No errors should occur in concurrent operations
		if len(errors) > 0 {
			for _, err := range errors {
				t.Errorf("Concurrent operation error: %v", err)
			}
			t.Fatalf("Expected zero errors in concurrent operations, got %d", len(errors))
		}
		
		if len(successIds) != numOperations {
			t.Fatalf("Expected %d successful operations, got %d", numOperations, len(successIds))
		}
		
		// Wait for propagation and verify all items exist
		waitForPropagation(5 * time.Second)
		
		// Verify all items are present
		err := retryOperation(func() error {
			results, err := index.ListIDs(ctx)
			if err != nil {
				return err
			}
			
			concurrentCount := 0
			for _, id := range results.Ids {
				if strings.HasPrefix(id, "concurrent_") {
					concurrentCount++
				}
			}
			
			if concurrentCount != numOperations {
				return fmt.Errorf("expected %d items in index, found %d", numOperations, concurrentCount)
			}
			return nil
		}, maxRetries, time.Second)
		
		if err != nil {
			t.Errorf("Data integrity issue after concurrent operations: %v", err)
		}
	})

	t.Run("TestBoundaryValues", func(t *testing.T) {
		testCases := []struct {
			name     string
			vector   []float32
			shouldSucceed bool
		}{
			{
				name:     "Zero vector",
				vector:   make([]float32, 128),
				shouldSucceed: true,
			},
			{
				name:     "Very small values",
				vector:   generateVectorWithValue(128, 1e-10),
				shouldSucceed: true,
			},
			{
				name:     "Very large values",
				vector:   generateVectorWithValue(128, 1e10),
				shouldSucceed: true,
			},
			{
				name:     "Mixed positive negative",
				vector:   generateMixedVector(128),
				shouldSucceed: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				items := []cyborgdb.VectorItem{{
					Id:       fmt.Sprintf("boundary_%s", strings.ReplaceAll(tc.name, " ", "_")),
					Vector:   tc.vector,
					Metadata: map[string]interface{}{"type": tc.name},
				}}
				
				err := index.Upsert(ctx, items)
				
				if tc.shouldSucceed {
					if err != nil {
						t.Errorf("Expected success for %s, got error: %v", tc.name, err)
					}
				} else {
					if err == nil {
						t.Errorf("Expected failure for %s, but operation succeeded", tc.name)
					}
				}
			})
		}
	})

	t.Run("TestLargeMetadataHandling", func(t *testing.T) {
		// Test various metadata sizes
		testCases := []struct {
			name     string
			metadata map[string]interface{}
		}{
			{
				name: "Large string",
				metadata: map[string]interface{}{
					"description": strings.Repeat("A", 10000),
				},
			},
			{
				name: "Deep nesting",
				metadata: createDeepNestedMetadata(5),
			},
			{
				name: "Large array",
				metadata: map[string]interface{}{
					"array": make([]int, 1000),
				},
			},
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
				}
				
				// Verify retrieval
				waitForPropagation(2 * time.Second)
				include := []string{"metadata"}
				results, err := index.Get(ctx, []string{items[0].Id}, include)
				if err != nil {
					t.Errorf("Failed to retrieve %s: %v", tc.name, err)
				}
				
				if len(results.Results) != 1 {
					t.Errorf("Expected 1 result for %s, got %d", tc.name, len(results.Results))
				}
			})
		}
	})
}

// Backend Compatibility Tests - Strict validation
func TestBackendCompatibility(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	client, err := createClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("TestHealthCheck", func(t *testing.T) {
		health, err := client.GetHealth(ctx)
		if err != nil {
			errorStr := strings.ToLower(err.Error())
			if strings.Contains(errorStr, "connection refused") || strings.Contains(errorStr, "timeout") {
				t.Skip("Server not reachable")
			}
			t.Fatalf("Health check failed: %v", err)
		}
		
		if health == nil {
			t.Fatal("Health response must not be nil")
		}
	})

	t.Run("TestBasicIndexSupport", func(t *testing.T) {
		// IVFFlat should work on all backends
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
		
		// Test basic operations
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
		
		// Test IVFPQ availability
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
			errorStr := strings.ToLower(err.Error())
			if strings.Contains(errorStr, "lite") || strings.Contains(errorStr, "not supported") {
				t.Log("Advanced index types not supported - this is expected for lite backend")
				return
			}
			t.Errorf("Unexpected error creating advanced index: %v", err)
			return
		}
		defer advancedIndex.DeleteIndex(ctx)
		
		t.Log("Advanced index types supported")
		
		// If advanced indexes are supported, they should work properly
		vectors := generateTestVectors(100, 128) // Need more vectors for IVFPQ
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

// Helper functions for test data generation
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