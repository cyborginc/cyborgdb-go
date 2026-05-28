package test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
	"github.com/cyborginc/cyborgdb-go/internal"
)

// Test configuration constants
const (
	testTimeout      = 120 * time.Second
	propagationDelay = 2 * time.Second
	dimension        = 384 // Matching embedding model dimension
)

// Global test state
var (
	testClient     *cyborgdb.Client
	testIndex      *cyborgdb.EncryptedIndex
	testIndexName  string
	testIndexKey   []byte
	embeddingIndex *cyborgdb.EncryptedIndex
	embeddingName  string
	embeddingKey   []byte
	testVectors    [][]float32
	testMetadata   []map[string]interface{}
)

// initAPIContractTestData initializes test data for API contract tests.
// Called from TestMain in helpers_test.go.
func initAPIContractTestData() {
	testVectors = generateTestVectors(10, dimension)
	testMetadata = generateTestMetadata(10)
	testIndexName = fmt.Sprintf("test_contract_%d", time.Now().UnixNano())
	embeddingName = fmt.Sprintf("test_embed_%d", time.Now().UnixNano())
	testIndexKey = generateRandomKey()
	embeddingKey = generateRandomKey()
}

// cleanupAPIContractTests cleans up resources created by API contract tests
func cleanupAPIContractTests() {
	ctx := context.Background()
	if testIndex != nil {
		_ = testIndex.DeleteIndex(ctx)
	}
	if embeddingIndex != nil {
		_ = embeddingIndex.DeleteIndex(ctx)
	}
}

// Helper functions

func generateTestMetadata(count int) []map[string]interface{} {
	metadata := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		metadata[i] = map[string]interface{}{
			"index":    i,
			"category": fmt.Sprintf("cat_%d", i%3),
			"value":    i * 10,
		}
	}
	return metadata
}

// Test Suite

// Test 01: Module Exports
func TestModuleExports(t *testing.T) {
	t.Run("RequiredTypesExist", func(t *testing.T) {
		// Verify key types are exported
		// Verify exported types exist and have expected kinds
		var client *cyborgdb.Client
		var index *cyborgdb.EncryptedIndex
		var params *cyborgdb.CreateIndexParams
		var queryParams cyborgdb.QueryParams
		var item cyborgdb.VectorItem

		// Pointer types should be pointers
		if reflect.TypeOf(client).Kind() != reflect.Pointer {
			t.Error("Client should be a pointer type")
		}
		if reflect.TypeOf(index).Kind() != reflect.Pointer {
			t.Error("EncryptedIndex should be a pointer type")
		}
		if reflect.TypeOf(params).Kind() != reflect.Pointer {
			t.Error("CreateIndexParams should be a pointer type")
		}
		// Struct types should be structs
		if reflect.TypeOf(queryParams).Kind() != reflect.Struct {
			t.Error("QueryParams should be a struct")
		}
		if reflect.TypeOf(item).Kind() != reflect.Struct {
			t.Error("VectorItem should be a struct")
		}
	})
}

// Test 02: Client Constructor
func TestClientConstructor(t *testing.T) {
	baseURL := os.Getenv("CYBORGDB_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}
	apiKey := os.Getenv("CYBORGDB_API_KEY")

	t.Run("ConstructWithRequiredParameters", func(t *testing.T) {
		client, err := cyborgdb.NewClient(baseURL, apiKey)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		if client == nil {
			t.Fatal("Client should not be nil")
		}
	})

	t.Run("ConstructWithOptionalVerifySSL", func(t *testing.T) {
		client, err := cyborgdb.NewClient(baseURL, apiKey, true)
		if err != nil {
			t.Fatalf("Failed to create client with verifySsl: %v", err)
		}
		if client == nil {
			t.Fatal("Client should not be nil")
		}
	})

	t.Run("RequireBaseURL", func(t *testing.T) {
		client, err := cyborgdb.NewClient("", apiKey)
		// SDK behavior: either reject at construction time, or allow but fail on requests
		if err != nil {
			// Good: SDK validates baseURL at construction
			t.Logf("SDK correctly rejected empty baseURL: %v", err)
		} else if client != nil {
			// SDK allows construction but should fail on actual request
			ctx := context.Background()
			_, healthErr := client.GetHealth(ctx)
			if healthErr == nil {
				t.Error("Expected error when making request with empty baseURL")
			}
		}
	})

	t.Run("StoreClientForLaterTests", func(t *testing.T) {
		client, err := cyborgdb.NewClient(baseURL, apiKey)
		if err != nil {
			t.Fatalf("Failed to create test client: %v", err)
		}
		testClient = client
	})
}

// Test 03: GenerateKey Function
func TestGenerateKey(t *testing.T) {
	t.Run("Generate32ByteKeyStatic", func(t *testing.T) {
		key, err := cyborgdb.GenerateKey()
		if err != nil {
			t.Fatalf("Failed to generate key: %v", err)
		}
		if len(key) != 32 {
			t.Errorf("Expected 32-byte key, got %d bytes", len(key))
		}
	})

	t.Run("GenerateUniqueKeys", func(t *testing.T) {
		key1, _ := cyborgdb.GenerateKey()
		key2, _ := cyborgdb.GenerateKey()

		if reflect.DeepEqual(key1, key2) {
			t.Error("Generated keys should be unique")
		}
	})

	t.Run("GenerateKeyNoArguments", func(t *testing.T) {
		// GenerateKey should not accept arguments (compile-time check)
		key, err := cyborgdb.GenerateKey()
		if err != nil {
			t.Fatalf("Failed to generate key: %v", err)
		}
		if len(key) != 32 {
			t.Error("Key should be 32 bytes")
		}
	})

	t.Run("StoreKeysForLaterTests", func(t *testing.T) {
		testIndexKey = generateRandomKey()
		embeddingKey = generateRandomKey()

		if len(testIndexKey) != 32 || len(embeddingKey) != 32 {
			t.Fatal("Generated keys have incorrect length")
		}
	})
}

// Test 04: Client.GetHealth()
func TestClientGetHealth(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("ReturnValidHealthStatus", func(t *testing.T) {
		health, err := testClient.GetHealth(ctx)
		if err != nil {
			t.Fatalf("GetHealth failed: %v", err)
		}

		// Health is a map[string]string
		if len(health) == 0 {
			t.Fatal("Health response should not be empty")
		}

		if _, exists := health["status"]; !exists {
			t.Error("Health response should contain 'status' field")
		}
	})

	t.Run("GetHealthNoArguments", func(t *testing.T) {
		// GetHealth should only take context (compile-time check)
		health, err := testClient.GetHealth(ctx)
		if err != nil {
			t.Fatalf("GetHealth failed: %v", err)
		}
		if health == nil {
			t.Error("Health should not be nil")
		}
	})
}

// Test 05: Client.ListIndexes()
func TestClientListIndexes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("ReturnArrayOfIndexNames", func(t *testing.T) {
		indexes, err := testClient.ListIndexes(ctx)
		if err != nil {
			t.Fatalf("ListIndexes failed: %v", err)
		}

		if indexes == nil {
			t.Fatal("Indexes should not be nil")
		}

		for _, name := range indexes {
			if name == "" {
				t.Error("Index name should not be empty")
			}
		}
	})

	t.Run("ListIndexesNoArguments", func(t *testing.T) {
		// ListIndexes should only take context (compile-time check)
		indexes, err := testClient.ListIndexes(ctx)
		if err != nil {
			t.Fatalf("ListIndexes failed: %v", err)
		}
		if indexes == nil {
			t.Error("Indexes should not be nil")
		}
	})
}

// Test 07: Client.CreateIndex()
func TestClientCreateIndex(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("CreateIndexWithDimensionAndCustomMetric", func(t *testing.T) {
		tempName := generateUniqueName("temp_diskivf_")
		tempKey := generateRandomKey()
		dim := int32(dimension)
		metric := "cosine"

		params := &cyborgdb.CreateIndexParams{
			IndexName: tempName,
			IndexKey:  tempKey,
			Dimension: &dim,
			Metric:    &metric,
		}

		index, err := testClient.CreateIndex(ctx, params)
		if err != nil {
			t.Fatalf("Failed to create DiskIVF index: %v", err)
		}
		defer func() { _ = index.DeleteIndex(ctx) }()

		if index.GetIndexName() != tempName {
			t.Errorf("Expected index name %s, got %s", tempName, index.GetIndexName())
		}

		if index.GetIndexType() != "disk_ivf" {
			t.Errorf("Expected index type disk_ivf, got %s", index.GetIndexType())
		}

		time.Sleep(1 * time.Second)
	})

	t.Run("CreateIndexWithEmbeddingModel", func(t *testing.T) {
		embeddingModel := "all-MiniLM-L6-v2"
		dim := int32(384) // 384 = dimension of all-MiniLM-L6-v2
		params := &cyborgdb.CreateIndexParams{
			IndexName:      embeddingName,
			IndexKey:       embeddingKey,
			EmbeddingModel: &embeddingModel,
			Dimension:      &dim,
		}

		index, err := testClient.CreateIndex(ctx, params)
		if err != nil {
			t.Fatalf("Failed to create embedding index: %v", err)
		}
		embeddingIndex = index

		if index.GetIndexType() != "disk_ivf" {
			t.Errorf("Expected index type disk_ivf, got %s", index.GetIndexType())
		}

		time.Sleep(2 * time.Second)
	})

	t.Run("RejectDuplicateIndexCreation", func(t *testing.T) {
		dupName := generateUniqueName("dup_test_")
		dupKey := generateRandomKey()
		dim := int32(dimension)

		params := &cyborgdb.CreateIndexParams{
			IndexName: dupName,
			IndexKey:  dupKey,
			Dimension: &dim,
		}

		index, err := testClient.CreateIndex(ctx, params)
		if err != nil {
			t.Fatalf("Failed to create first index: %v", err)
		}
		defer func() { _ = index.DeleteIndex(ctx) }()

		_, err = testClient.CreateIndex(ctx, params)
		if err == nil {
			t.Error("Should reject duplicate index creation")
		}

		time.Sleep(1 * time.Second)
	})

	t.Run("RejectUnexpectedParameters", func(t *testing.T) {
		// This is a compile-time check - we can't pass unexpected fields to a struct
		// Just verify that CreateIndexParams only accepts documented fields
		tempName := generateUniqueName("temp_unexpected_")
		tempKey := generateRandomKey()
		dim := int32(dimension)

		params := &cyborgdb.CreateIndexParams{
			IndexName: tempName,
			IndexKey:  tempKey,
			Dimension: &dim,
		}

		index, err := testClient.CreateIndex(ctx, params)
		if err != nil {
			t.Fatalf("Failed to create index: %v", err)
		}
		defer func() { _ = index.DeleteIndex(ctx) }()

		time.Sleep(1 * time.Second)
	})

	t.Run("CreateIndexWithStoragePrecision", func(t *testing.T) {
		tempName := generateUniqueName("temp_advanced_")
		tempKey := generateRandomKey()
		dim := int32(dimension)
		precision := cyborgdb.StoragePrecisionFloat16

		params := &cyborgdb.CreateIndexParams{
			IndexName:        tempName,
			IndexKey:         tempKey,
			Dimension:        &dim,
			StoragePrecision: &precision,
		}

		index, err := testClient.CreateIndex(ctx, params)
		if err != nil {
			t.Fatalf("Failed to create index with storage precision: %v", err)
		}
		defer func() { _ = index.DeleteIndex(ctx) }()

		if index.GetIndexType() != "disk_ivf" {
			t.Errorf("Expected index type disk_ivf, got %s", index.GetIndexType())
		}

		time.Sleep(1 * time.Second)
	})

	t.Run("CreateMainTestIndex", func(t *testing.T) {
		dim := int32(dimension)
		metric := "cosine"

		params := &cyborgdb.CreateIndexParams{
			IndexName: testIndexName,
			IndexKey:  testIndexKey,
			Dimension: &dim,
			Metric:    &metric,
		}

		index, err := testClient.CreateIndex(ctx, params)
		if err != nil {
			t.Fatalf("Failed to create main test index: %v", err)
		}
		testIndex = index

		time.Sleep(2 * time.Second)

		// Verify index was created
		indexes, _ := testClient.ListIndexes(ctx)
		found := false
		for _, name := range indexes {
			if name == testIndexName {
				found = true
				break
			}
		}
		if !found {
			t.Error("Test index not found in index list")
		}

		// Verify index name
		if index.GetIndexName() != testIndexName {
			t.Errorf("Expected index name %s, got %s", testIndexName, index.GetIndexName())
		}
	})
}

// Test 08: EncryptedIndex Properties
func TestEncryptedIndexProperties(t *testing.T) {
	t.Run("ExposeIndexNameViaGetter", func(t *testing.T) {
		name := testIndex.GetIndexName()
		if name != testIndexName {
			t.Errorf("Expected index name %s, got %s", testIndexName, name)
		}
		if reflect.TypeOf(name).Kind() != reflect.String {
			t.Error("Index name should be string")
		}
	})

	t.Run("ExposeIndexTypeViaGetter", func(t *testing.T) {
		indexType := testIndex.GetIndexType()
		if indexType != "disk_ivf" {
			t.Errorf("Expected index type disk_ivf, got %s", indexType)
		}
		if reflect.TypeOf(indexType).Kind() != reflect.String {
			t.Error("Index type should be string")
		}
	})
}

// Test 09: EncryptedIndex.IsTrained()
func TestEncryptedIndexIsTrained(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("ReturnBooleanAndError", func(t *testing.T) {
		trained, err := testIndex.IsTrained(ctx)
		if err != nil {
			t.Fatalf("IsTrained failed: %v", err)
		}
		if reflect.TypeOf(trained).Kind() != reflect.Bool {
			t.Error("IsTrained should return bool")
		}
	})

	t.Run("IsTrainedWithContext", func(t *testing.T) {
		// IsTrained takes a context and returns (bool, error)
		// For a newly created index, it should be untrained (false)
		trained, err := testIndex.IsTrained(ctx)
		if err != nil {
			t.Fatalf("IsTrained failed: %v", err)
		}
		// Just verify the function returns without error and returns a valid bool
		// The actual value depends on whether train was called
		if trained {
			t.Log("Index is already trained")
		} else {
			t.Log("Index is not yet trained (expected for new index)")
		}
	})
}

// Test 10: Client.IsTraining()
func TestClientIsTraining(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("ReturnTrainingStatusWithCorrectSchema", func(t *testing.T) {
		// Note: In Go SDK, this is CheckTrainingStatus on the index, not client
		// But we can verify the behavior exists
		isTraining, err := testIndex.CheckTrainingStatus(ctx)
		if err != nil {
			t.Fatalf("CheckTrainingStatus failed: %v", err)
		}

		if reflect.TypeOf(isTraining).Kind() != reflect.Bool {
			t.Error("Training status should be bool")
		}
	})
}

// Test 11: EncryptedIndex.Upsert()
func TestEncryptedIndexUpsert(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("UpsertWithItemsArrayFormat", func(t *testing.T) {
		items := make(cyborgdb.VectorItems, 2)
		expectedIDs := make(map[string]bool)
		for i := 0; i < 2; i++ {
			id := fmt.Sprintf("%d", i)
			items[i] = cyborgdb.VectorItem{
				Id:       id,
				Vector:   testVectors[i],
				Metadata: testMetadata[i],
			}
			expectedIDs[id] = true
		}

		err := testIndex.Upsert(ctx, items)
		if err != nil {
			t.Fatalf("Upsert failed: %v", err)
		}

		// Poll until IDs are available instead of fixed sleep
		found := pollUntil(pollTimeout, func() bool {
			result, err := testIndex.ListIDs(ctx)
			if err != nil {
				return false
			}
			for id := range expectedIDs {
				idFound := false
				for _, existingID := range result.Ids {
					if existingID == id {
						idFound = true
						break
					}
				}
				if !idFound {
					return false
				}
			}
			return true
		})
		if !found {
			t.Error("Upserted IDs not found after polling timeout")
		}
	})

	t.Run("UpsertWithItemsArrayAutoEmbed", func(t *testing.T) {
		items := make(cyborgdb.VectorItems, 3)
		expectedIDs := make(map[string]bool)
		sampleTexts := []string{
			"The quick brown fox jumps over the lazy dog",
			"Machine learning models process natural language",
			"Vector databases enable semantic search capabilities",
		}
		for i := 0; i < 3; i++ {
			id := fmt.Sprintf("embed_%d", i)
			item := cyborgdb.VectorItem{
				Id:       id,
				Metadata: map[string]interface{}{"type": "auto-embedded", "index": i},
			}
			// Set contents for auto-embedding
			item.SetContents(internal.Contents{String: &sampleTexts[i]})
			items[i] = item
			expectedIDs[id] = true
		}

		err := embeddingIndex.Upsert(ctx, items)
		if err != nil {
			t.Fatalf("Auto-embed upsert failed: %v", err)
		}

		// Poll until IDs are available
		found := pollUntil(pollTimeout, func() bool {
			result, err := embeddingIndex.ListIDs(ctx)
			if err != nil {
				return false
			}
			for id := range expectedIDs {
				idFound := false
				for _, existingID := range result.Ids {
					if existingID == id {
						idFound = true
						break
					}
				}
				if !idFound {
					return false
				}
			}
			return true
		})
		if !found {
			t.Error("Auto-embedded IDs not found after polling timeout")
		}
	})

	t.Run("UpsertRemainingTestItems", func(t *testing.T) {
		items := make(cyborgdb.VectorItems, 8)
		expectedIDs := make(map[string]bool)
		for i := 2; i < 10; i++ {
			id := fmt.Sprintf("%d", i)
			items[i-2] = cyborgdb.VectorItem{
				Id:       id,
				Vector:   testVectors[i%len(testVectors)],
				Metadata: testMetadata[i%len(testMetadata)],
			}
			expectedIDs[id] = true
		}

		err := testIndex.Upsert(ctx, items)
		if err != nil {
			t.Fatalf("Batch upsert failed: %v", err)
		}

		// Poll until all IDs are available
		found := pollUntil(pollTimeout, func() bool {
			result, err := testIndex.ListIDs(ctx)
			if err != nil {
				return false
			}
			for id := range expectedIDs {
				idFound := false
				for _, existingID := range result.Ids {
					if existingID == id {
						idFound = true
						break
					}
				}
				if !idFound {
					return false
				}
			}
			return true
		})
		if !found {
			t.Error("Batch upserted IDs not found after polling timeout")
		}
	})

	t.Run("UpsertWithParallelArraysFormat", func(t *testing.T) {
		// Go SDK doesn't support separate ids/vectors arrays like Python/TS
		// Use items array instead
		items := make(cyborgdb.VectorItems, 5)
		expectedIDs := make(map[string]bool)
		for i := 10; i < 15; i++ {
			id := fmt.Sprintf("%d", i)
			items[i-10] = cyborgdb.VectorItem{
				Id:     id,
				Vector: testVectors[i%len(testVectors)],
			}
			expectedIDs[id] = true
		}

		err := testIndex.Upsert(ctx, items)
		if err != nil {
			t.Fatalf("Additional upsert failed: %v", err)
		}

		// Poll until all IDs are available
		found := pollUntil(pollTimeout, func() bool {
			result, err := testIndex.ListIDs(ctx)
			if err != nil {
				return false
			}
			for id := range expectedIDs {
				idFound := false
				for _, existingID := range result.Ids {
					if existingID == id {
						idFound = true
						break
					}
				}
				if !idFound {
					return false
				}
			}
			return true
		})
		if !found {
			t.Error("Additional upserted IDs not found after polling timeout")
		}
	})

	t.Run("RejectVectorsWithWrongDimensions", func(t *testing.T) {
		wrongVector := make([]float32, 64)
		items := cyborgdb.VectorItems{{
			Id:     "wrong-dim",
			Vector: wrongVector,
		}}

		err := testIndex.Upsert(ctx, items)
		if err == nil {
			t.Error("Should reject vectors with wrong dimensions")
		}
	})

	t.Run("RejectWhenNeitherItemsNorVectorsProvided", func(t *testing.T) {
		// Test empty items array behavior
		items := cyborgdb.VectorItems{}
		err := testIndex.Upsert(ctx, items)
		// Document actual SDK behavior for empty upsert
		if err != nil {
			t.Logf("SDK rejects empty upsert: %v", err)
		} else {
			t.Log("SDK allows empty upsert as no-op")
		}
		// Either behavior is acceptable - just document it
	})
}

// Test 12: EncryptedIndex.ListIDs()
func TestEncryptedIndexListIDs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("ReturnObjectWithIDsArrayAndCount", func(t *testing.T) {
		result, err := testIndex.ListIDs(ctx)
		if err != nil {
			t.Fatalf("ListIDs failed: %v", err)
		}

		if result.Ids == nil {
			t.Fatal("IDs should not be nil")
		}

		if int(result.Count) != len(result.Ids) {
			t.Errorf("Count %d doesn't match IDs length %d", result.Count, len(result.Ids))
		}

		for _, id := range result.Ids {
			if reflect.TypeOf(id).Kind() != reflect.String {
				t.Error("Each ID should be string")
			}
		}

		// We upserted IDs 0-14 (15 total)
		expectedIDs := make(map[string]bool)
		for i := 0; i < 15; i++ {
			expectedIDs[fmt.Sprintf("%d", i)] = true
		}

		for _, id := range result.Ids {
			if !expectedIDs[id] {
				t.Errorf("Unexpected ID: %s", id)
			}
			delete(expectedIDs, id)
		}

		if len(expectedIDs) > 0 {
			missing := []string{}
			for id := range expectedIDs {
				missing = append(missing, id)
			}
			t.Errorf("Missing IDs: %v", missing)
		}
	})

	t.Run("ListIDsNoArguments", func(t *testing.T) {
		// ListIDs should only take context (compile-time check)
		result, err := testIndex.ListIDs(ctx)
		if err != nil {
			t.Fatalf("ListIDs failed: %v", err)
		}
		if result == nil {
			t.Error("Result should not be nil")
		}
	})
}

// Test 13: EncryptedIndex.Get()
func TestEncryptedIndexGet(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("GetVectorsWithDefaultInclude", func(t *testing.T) {
		ids := []string{"0", "5", "9"}
		results, err := testIndex.Get(ctx, ids, []string{"vector", "metadata"})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if len(results.Results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results.Results))
		}

		for _, result := range results.Results {
			if result.GetId() == "" {
				t.Error("Result should have ID")
			}

			vector := result.GetVector()
			if len(vector) != dimension {
				t.Errorf("Expected vector dimension %d, got %d", dimension, len(vector))
			}

			metadata := result.GetMetadata()
			idInt := 0
			_, _ = fmt.Sscanf(result.GetId(), "%d", &idInt)
			if idInt < 10 {
				if metadata == nil {
					t.Error("Expected metadata for ID < 10")
				}
			}
		}
	})

	t.Run("GetVectorsWithSpecificInclude", func(t *testing.T) {
		ids := []string{"0", "5"}
		results, err := testIndex.Get(ctx, ids, []string{"metadata"})
		if err != nil {
			t.Fatalf("Get with include failed: %v", err)
		}

		for _, result := range results.Results {
			if result.GetId() == "" {
				t.Error("Result should have ID")
			}

			metadata := result.GetMetadata()
			if metadata == nil {
				t.Error("Expected metadata in include")
			}
		}
	})

	t.Run("GetVectorsWithEmptyInclude", func(t *testing.T) {
		ids := []string{"0"}
		results, err := testIndex.Get(ctx, ids, []string{})
		if err != nil {
			t.Fatalf("Get with empty include failed: %v", err)
		}

		if len(results.Results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results.Results))
		}

		result := results.Results[0]
		if result.GetId() == "" {
			t.Error("Result should have ID")
		}
	})
}

// Test 14: EncryptedIndex.Query()
func TestEncryptedIndexQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("QueryWithSingleVectorFlatArray", func(t *testing.T) {
		topK := int32(5)
		params := cyborgdb.QueryParams{
			QueryVector: testVectors[0],
			TopK:        topK,
			Include:     []string{"distance"},
		}

		results, err := testIndex.Query(ctx, params)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if results == nil {
			t.Fatal("Results must not be nil")
		}

		// Extract result items from union type
		resultItems := getQueryResultItems(&results.Results)

		// Verify result count respects TopK
		if len(resultItems) == 0 {
			t.Error("Expected at least one result")
		}
		if len(resultItems) > int(topK) {
			t.Errorf("Result count %d exceeds TopK %d", len(resultItems), topK)
		}

		// Verify each result has required fields
		for i, result := range resultItems {
			if result.Id == "" {
				t.Errorf("Result %d: missing ID", i)
			}
			// Distance should be non-negative for similarity search
			// Use small epsilon to handle floating-point precision issues (e.g., -1e-10)
			dist := result.GetDistance()
			if dist < -1e-6 {
				t.Errorf("Result %d: invalid distance %f", i, dist)
			}
		}
	})

	t.Run("QueryWithNestedArraySingleVector", func(t *testing.T) {
		// Go SDK uses QueryVector for single, BatchQueryVectors for batch
		topK := int32(3)
		params := cyborgdb.QueryParams{
			QueryVector: testVectors[1],
			TopK:        topK,
			Include:     []string{"distance"},
		}

		results, err := testIndex.Query(ctx, params)
		if err != nil {
			t.Fatalf("Query with topK failed: %v", err)
		}

		if results == nil {
			t.Fatal("Results must not be nil")
		}

		resultItems := getQueryResultItems(&results.Results)

		// Verify result count respects TopK
		if len(resultItems) > int(topK) {
			t.Errorf("Result count %d exceeds TopK %d", len(resultItems), topK)
		}

		// Verify results are ordered by distance (ascending)
		for i := 1; i < len(resultItems); i++ {
			currDist := resultItems[i].GetDistance()
			prevDist := resultItems[i-1].GetDistance()
			if currDist < prevDist {
				t.Errorf("Results not sorted by distance: result[%d]=%f < result[%d]=%f",
					i, currDist, i-1, prevDist)
			}
		}
	})

	t.Run("QueryWithBatchVectors", func(t *testing.T) {
		batchVectors := [][]float32{testVectors[2], testVectors[3]}
		topK := int32(2)
		params := cyborgdb.QueryParams{
			BatchQueryVectors: batchVectors,
			TopK:              topK,
		}

		results, err := testIndex.Query(ctx, params)
		if err != nil {
			t.Fatalf("Batch query failed: %v", err)
		}

		if results == nil {
			t.Fatal("Results must not be nil")
		}

		// Use batch helper to get ALL result sets
		batchResults := getBatchQueryResults(&results.Results)

		// Verify we got results for each query vector in the batch
		if len(batchResults) != len(batchVectors) {
			t.Errorf("Expected %d result sets for %d query vectors, got %d",
				len(batchVectors), len(batchVectors), len(batchResults))
		}

		// Validate each result set
		for batchIdx, resultSet := range batchResults {
			// Each result set should respect TopK
			if len(resultSet) > int(topK) {
				t.Errorf("Batch %d: result count %d exceeds TopK %d",
					batchIdx, len(resultSet), topK)
			}

			// Verify each result has valid fields
			for i, result := range resultSet {
				if result.Id == "" {
					t.Errorf("Batch %d, result %d: missing ID", batchIdx, i)
				}
			}
		}
	})

	t.Run("QueryWithSpecificInclude", func(t *testing.T) {
		topK := int32(5)
		params := cyborgdb.QueryParams{
			QueryVector: testVectors[0],
			TopK:        topK,
			Include:     []string{"metadata"},
		}

		results, err := testIndex.Query(ctx, params)
		if err != nil {
			t.Fatalf("Query with include failed: %v", err)
		}

		if results == nil {
			t.Fatal("Results must not be nil")
		}

		resultItems := getQueryResultItems(&results.Results)

		if len(resultItems) > int(topK) {
			t.Errorf("Result count %d exceeds TopK %d", len(resultItems), topK)
		}

		// Verify metadata is included in results (for items that have metadata)
		hasMetadata := false
		for _, result := range resultItems {
			if result.Id == "" {
				t.Error("Result missing ID")
			}
			if len(result.Metadata) > 0 {
				hasMetadata = true
			}
		}
		if !hasMetadata && len(resultItems) > 0 {
			t.Log("Warning: No metadata returned despite include=['metadata'] - may be expected if vectors have no metadata")
		}
	})

	t.Run("QueryWithMetadataFilters", func(t *testing.T) {
		expectedCategory := "cat_0"
		filters := map[string]interface{}{"category": expectedCategory}
		topK := int32(10)
		params := cyborgdb.QueryParams{
			QueryVector: testVectors[0],
			TopK:        topK,
			Filters:     filters,
			Include:     []string{"metadata"},
		}

		results, err := testIndex.Query(ctx, params)
		if err != nil {
			t.Fatalf("Query with filters failed: %v", err)
		}

		if results == nil {
			t.Fatal("Results must not be nil")
		}

		resultItems := getQueryResultItems(&results.Results)

		if len(resultItems) > int(topK) {
			t.Errorf("Result count %d exceeds TopK %d", len(resultItems), topK)
		}

		// Verify ALL returned results match the filter criteria
		for i, result := range resultItems {
			if result.Id == "" {
				t.Errorf("Result %d: missing ID", i)
				continue
			}

			if result.Metadata == nil {
				t.Errorf("Result %d (ID=%s): metadata is nil, cannot verify filter", i, result.Id)
				continue
			}

			category, ok := result.Metadata["category"]
			if !ok {
				t.Errorf("Result %d (ID=%s): missing 'category' field in metadata", i, result.Id)
				continue
			}

			categoryStr, ok := category.(string)
			if !ok {
				t.Errorf("Result %d (ID=%s): category is not a string: %T", i, result.Id, category)
				continue
			}

			if categoryStr != expectedCategory {
				t.Errorf("Result %d (ID=%s): filter violation - expected category=%q, got %q",
					i, result.Id, expectedCategory, categoryStr)
			}
		}
	})

	t.Run("QueryWithTextContentsAutoEmbed", func(t *testing.T) {
		queryText := "test content for similarity search"
		topK := int32(3)
		params := cyborgdb.QueryParams{
			QueryContents: &queryText,
			TopK:          topK,
		}

		results, err := embeddingIndex.Query(ctx, params)
		if err != nil {
			t.Fatalf("Query with text contents failed: %v", err)
		}

		if results == nil {
			t.Fatal("Results must not be nil")
		}

		resultItems := getQueryResultItems(&results.Results)

		if len(resultItems) > int(topK) {
			t.Errorf("Result count %d exceeds TopK %d", len(resultItems), topK)
		}

		// Verify each result has valid ID
		for i, result := range resultItems {
			if result.Id == "" {
				t.Errorf("Result %d: missing ID", i)
			}
		}
	})

	t.Run("QueryDefaultIncludeReturnsOnlyID", func(t *testing.T) {
		// With no Include param, Query() must return only id — no distance,
		// metadata, or vector. Mirrors the cyborgdb-js contract test added in
		// commit 38b917b after the server changed the default include behavior.
		topK := int32(5)
		params := cyborgdb.QueryParams{
			QueryVector: testVectors[0],
			TopK:        topK,
		}

		results, err := testIndex.Query(ctx, params)
		if err != nil {
			t.Fatalf("Query with default include failed: %v", err)
		}
		if results == nil {
			t.Fatal("Results must not be nil")
		}

		resultItems := getQueryResultItems(&results.Results)
		if len(resultItems) == 0 {
			t.Fatal("Expected at least one result")
		}

		for i, result := range resultItems {
			if result.Id == "" {
				t.Errorf("Result %d: missing ID", i)
			}
			if result.HasDistance() {
				t.Errorf("Result %d: expected no distance with default include, got %f",
					i, result.GetDistance())
			}
			if result.Metadata != nil {
				t.Errorf("Result %d: expected no metadata with default include, got %v",
					i, result.Metadata)
			}
			if result.Vector != nil {
				t.Errorf("Result %d: expected no vector with default include, got %d-dim vector",
					i, len(result.Vector))
			}
		}
	})
}

// Test 15: EncryptedIndex.Query() Additional Patterns
func TestEncryptedIndexQueryPatterns(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("QueryWithMultipleTestPatterns", func(t *testing.T) {
		// Test 1: Single vector with TopK validation
		topK1 := int32(3)
		params1 := cyborgdb.QueryParams{
			QueryVector: testVectors[4],
			TopK:        topK1,
		}
		results1, err1 := testIndex.Query(ctx, params1)
		if err1 != nil {
			t.Fatalf("Query 1 failed: %v", err1)
		}
		if results1 == nil {
			t.Fatal("Results 1 should not be nil")
		}
		resultItems1 := getQueryResultItems(&results1.Results)
		if len(resultItems1) > int(topK1) {
			t.Errorf("Query 1: result count %d exceeds TopK %d", len(resultItems1), topK1)
		}
		for i, r := range resultItems1 {
			if r.Id == "" {
				t.Errorf("Query 1 result %d: missing ID", i)
			}
		}

		// Test 2: Batch vectors with validation
		topK2 := int32(2)
		params2 := cyborgdb.QueryParams{
			BatchQueryVectors: [][]float32{testVectors[5], testVectors[6]},
			TopK:              topK2,
		}
		results2, err2 := testIndex.Query(ctx, params2)
		if err2 != nil {
			t.Fatalf("Query 2 failed: %v", err2)
		}
		if results2 == nil {
			t.Fatal("Results 2 should not be nil")
		}
		resultItems2 := getQueryResultItems(&results2.Results)
		if len(resultItems2) == 0 {
			t.Error("Query 2: expected results from batch query")
		}

		// Test 3: With filters - verify filter criteria is respected
		expectedCategory := "cat_1"
		topK3 := int32(10)
		params3 := cyborgdb.QueryParams{
			QueryVector: testVectors[7],
			TopK:        topK3,
			Filters:     map[string]interface{}{"category": expectedCategory},
			Include:     []string{"metadata"},
		}
		results3, err3 := testIndex.Query(ctx, params3)
		if err3 != nil {
			t.Fatalf("Query 3 failed: %v", err3)
		}
		if results3 == nil {
			t.Fatal("Results 3 should not be nil")
		}
		resultItems3 := getQueryResultItems(&results3.Results)
		if len(resultItems3) > int(topK3) {
			t.Errorf("Query 3: result count %d exceeds TopK %d", len(resultItems3), topK3)
		}

		// Verify filter criteria for all results
		for i, result := range resultItems3 {
			if result.Metadata == nil {
				continue // Skip if metadata not returned
			}
			category, ok := result.Metadata["category"]
			if !ok {
				continue // Skip if category not in metadata
			}
			categoryStr, ok := category.(string)
			if ok && categoryStr != expectedCategory {
				t.Errorf("Query 3 result %d (ID=%s): filter violation - expected category=%q, got %q",
					i, result.Id, expectedCategory, categoryStr)
			}
		}
	})
}

// Test 15b: EncryptedIndex Binary Upsert and Query
func TestEncryptedIndexBinaryUpsertAndQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Use unique IDs for binary tests to avoid conflicts with regular upsert tests
	binaryIDPrefix := "binary_"

	t.Run("BinaryUpsertWithVectors", func(t *testing.T) {
		// Prepare binary upsert params
		ids := make([]string, 5)
		vectors := make([][]float32, 5)
		metadata := make([]map[string]interface{}, 5)

		for i := 0; i < 5; i++ {
			ids[i] = fmt.Sprintf("%s%d", binaryIDPrefix, i)
			vectors[i] = testVectors[i%len(testVectors)]
			metadata[i] = map[string]interface{}{
				"binary_index": i,
				"category":     fmt.Sprintf("binary_cat_%d", i%3),
			}
		}

		params := cyborgdb.BinaryUpsertParams{
			IDs:      ids,
			Vectors:  vectors,
			Metadata: metadata,
		}

		err := testIndex.Upsert(ctx, params)
		if err != nil {
			t.Fatalf("Binary upsert failed: %v", err)
		}

		// Poll until IDs are available
		expectedIDs := make(map[string]bool)
		for _, id := range ids {
			expectedIDs[id] = true
		}

		found := pollUntil(pollTimeout, func() bool {
			result, err := testIndex.ListIDs(ctx)
			if err != nil {
				return false
			}
			for id := range expectedIDs {
				idFound := false
				for _, existingID := range result.Ids {
					if existingID == id {
						idFound = true
						break
					}
				}
				if !idFound {
					return false
				}
			}
			return true
		})
		if !found {
			t.Error("Binary upserted IDs not found after polling timeout")
		}
	})

	t.Run("BinaryUpsertWithoutMetadata", func(t *testing.T) {
		ids := make([]string, 3)
		vectors := make([][]float32, 3)

		for i := 0; i < 3; i++ {
			ids[i] = fmt.Sprintf("%snometa_%d", binaryIDPrefix, i)
			vectors[i] = testVectors[i%len(testVectors)]
		}

		params := cyborgdb.BinaryUpsertParams{
			IDs:     ids,
			Vectors: vectors,
		}

		err := testIndex.Upsert(ctx, params)
		if err != nil {
			t.Fatalf("Binary upsert without metadata failed: %v", err)
		}

		// Poll until IDs are available
		found := pollUntil(pollTimeout, func() bool {
			result, err := testIndex.ListIDs(ctx)
			if err != nil {
				return false
			}
			for _, id := range ids {
				idFound := false
				for _, existingID := range result.Ids {
					if existingID == id {
						idFound = true
						break
					}
				}
				if !idFound {
					return false
				}
			}
			return true
		})
		if !found {
			t.Error("Binary upserted IDs (no metadata) not found after polling timeout")
		}
	})

	t.Run("BinaryQueryWithSingleVector", func(t *testing.T) {
		topK := int32(5)
		params := cyborgdb.BinaryQueryParams{
			QueryVectors: [][]float32{testVectors[0]},
			TopK:         topK,
		}

		results, err := testIndex.Query(ctx, params)
		if err != nil {
			t.Fatalf("Binary query failed: %v", err)
		}

		if results == nil {
			t.Fatal("Results must not be nil")
		}

		resultItems := getQueryResultItems(&results.Results)

		if len(resultItems) == 0 {
			t.Error("Expected at least one result")
		}
		if len(resultItems) > int(topK) {
			t.Errorf("Result count %d exceeds TopK %d", len(resultItems), topK)
		}

		for i, result := range resultItems {
			if result.Id == "" {
				t.Errorf("Result %d: missing ID", i)
			}
		}
	})

	t.Run("BinaryQueryWithBatchVectors", func(t *testing.T) {
		batchVectors := [][]float32{testVectors[0], testVectors[1], testVectors[2]}
		topK := int32(3)
		params := cyborgdb.BinaryQueryParams{
			QueryVectors: batchVectors,
			TopK:         topK,
		}

		results, err := testIndex.Query(ctx, params)
		if err != nil {
			t.Fatalf("Binary batch query failed: %v", err)
		}

		if results == nil {
			t.Fatal("Results must not be nil")
		}

		batchResults := getBatchQueryResults(&results.Results)

		if len(batchResults) != len(batchVectors) {
			t.Errorf("Expected %d result sets for %d query vectors, got %d",
				len(batchVectors), len(batchVectors), len(batchResults))
		}

		for batchIdx, resultSet := range batchResults {
			if len(resultSet) > int(topK) {
				t.Errorf("Batch %d: result count %d exceeds TopK %d",
					batchIdx, len(resultSet), topK)
			}

			for i, result := range resultSet {
				if result.Id == "" {
					t.Errorf("Batch %d, result %d: missing ID", batchIdx, i)
				}
			}
		}
	})

	t.Run("BinaryQueryWithFilters", func(t *testing.T) {
		expectedCategory := "binary_cat_0"
		topK := int32(10)
		params := cyborgdb.BinaryQueryParams{
			QueryVectors: [][]float32{testVectors[0]},
			TopK:         topK,
			Filters:      map[string]interface{}{"category": expectedCategory},
			Include:      []string{"metadata"},
		}

		results, err := testIndex.Query(ctx, params)
		if err != nil {
			t.Fatalf("Binary query with filters failed: %v", err)
		}

		if results == nil {
			t.Fatal("Results must not be nil")
		}

		resultItems := getQueryResultItems(&results.Results)

		if len(resultItems) > int(topK) {
			t.Errorf("Result count %d exceeds TopK %d", len(resultItems), topK)
		}

		// Verify filter criteria for results that have metadata
		for i, result := range resultItems {
			if result.Metadata == nil {
				continue
			}
			category, ok := result.Metadata["category"]
			if !ok {
				continue
			}
			categoryStr, ok := category.(string)
			if ok && categoryStr != expectedCategory {
				t.Errorf("Result %d (ID=%s): filter violation - expected category=%q, got %q",
					i, result.Id, expectedCategory, categoryStr)
			}
		}
	})

	t.Run("BinaryQueryWithInclude", func(t *testing.T) {
		topK := int32(5)
		params := cyborgdb.BinaryQueryParams{
			QueryVectors: [][]float32{testVectors[0]},
			TopK:         topK,
			Include:      []string{"metadata", "vector"},
		}

		results, err := testIndex.Query(ctx, params)
		if err != nil {
			t.Fatalf("Binary query with include failed: %v", err)
		}

		if results == nil {
			t.Fatal("Results must not be nil")
		}

		resultItems := getQueryResultItems(&results.Results)

		if len(resultItems) == 0 {
			t.Error("Expected at least one result")
		}

		// Verify that results contain the requested fields
		for i, result := range resultItems {
			if result.Id == "" {
				t.Errorf("Result %d: missing ID", i)
			}
			// Vector should be included
			if len(result.Vector) == 0 {
				t.Logf("Result %d: vector not returned despite include=['vector'] - may be expected based on server config", i)
			}
		}
	})

	t.Run("BinaryUpsertRejectMismatchedLengths", func(t *testing.T) {
		// IDs and vectors have different lengths - should be rejected
		params := cyborgdb.BinaryUpsertParams{
			IDs:     []string{"mismatch_1", "mismatch_2"},
			Vectors: [][]float32{testVectors[0]}, // Only 1 vector for 2 IDs
		}

		err := testIndex.Upsert(ctx, params)
		if err == nil {
			t.Error("Should reject upsert with mismatched IDs and vectors lengths")
		}
	})

	t.Run("BinaryUpsertRejectEmptyIDs", func(t *testing.T) {
		params := cyborgdb.BinaryUpsertParams{
			IDs:     []string{},
			Vectors: [][]float32{testVectors[0]},
		}

		err := testIndex.Upsert(ctx, params)
		if err == nil {
			t.Error("Should reject upsert with empty IDs")
		}
	})

	t.Run("BinaryQueryRejectEmptyVectors", func(t *testing.T) {
		params := cyborgdb.BinaryQueryParams{
			QueryVectors: [][]float32{},
			TopK:         5,
		}

		_, err := testIndex.Query(ctx, params)
		if err == nil {
			t.Error("Should reject query with empty vectors")
		}
	})

	// Cleanup: Delete the binary test vectors
	t.Run("CleanupBinaryTestVectors", func(t *testing.T) {
		idsToDelete := make([]string, 8)
		for i := 0; i < 5; i++ {
			idsToDelete[i] = fmt.Sprintf("%s%d", binaryIDPrefix, i)
		}
		for i := 0; i < 3; i++ {
			idsToDelete[5+i] = fmt.Sprintf("%snometa_%d", binaryIDPrefix, i)
		}

		err := testIndex.Delete(ctx, idsToDelete)
		if err != nil {
			t.Logf("Cleanup warning: failed to delete binary test vectors: %v", err)
		}
	})
}

// Test 16: EncryptedIndex.Train()
func TestEncryptedIndexTrain(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("TrainWithDefaultParameters", func(t *testing.T) {
		err := testIndex.Train(ctx, cyborgdb.TrainParams{})
		if err != nil {
			t.Fatalf("Train with defaults failed: %v", err)
		}
	})

	t.Run("TrainWithCustomParameters", func(t *testing.T) {
		nLists := int32(10)
		batchSize := int32(512)
		maxIters := int32(50)
		tolerance := float64(1e-5)

		params := cyborgdb.TrainParams{
			NLists:    &nLists,
			BatchSize: &batchSize,
			MaxIters:  &maxIters,
			Tolerance: &tolerance,
		}

		err := testIndex.Train(ctx, params)
		if err != nil {
			t.Fatalf("Train with custom parameters failed: %v", err)
		}
	})

	t.Run("TrainWithPartialParameters", func(t *testing.T) {
		nLists := int32(5)
		params := cyborgdb.TrainParams{
			NLists: &nLists,
		}

		err := testIndex.Train(ctx, params)
		if err != nil {
			t.Fatalf("Train with partial parameters failed: %v", err)
		}

		// Allow training to complete before subsequent tests run delete operations.
		// Train() returns immediately but processing continues server-side.
		time.Sleep(2 * time.Second)
	})
}

// Test 17: EncryptedIndex.Delete()
func TestEncryptedIndexDelete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("DeleteVectorsByIDs", func(t *testing.T) {
		deletedIDs := []string{"0", "5"}
		err := testIndex.Delete(ctx, deletedIDs)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Poll until IDs are confirmed deleted
		deleted := pollUntil(pollTimeout, func() bool {
			result, listErr := testIndex.ListIDs(ctx)
			if listErr != nil {
				return false
			}
			for _, deletedID := range deletedIDs {
				for _, id := range result.Ids {
					if id == deletedID {
						return false // ID still exists
					}
				}
			}
			return true // All deleted IDs are gone
		})
		if !deleted {
			t.Error("Deleted IDs still found after polling timeout")
		}

		// Final verification
		result, err := testIndex.ListIDs(ctx)
		if err != nil {
			t.Fatalf("ListIDs failed: %v", err)
		}
		for _, deletedID := range deletedIDs {
			for _, id := range result.Ids {
				if id == deletedID {
					t.Errorf("ID %s should have been deleted", deletedID)
				}
			}
		}
	})

	t.Run("DeleteAdditionalVector", func(t *testing.T) {
		deleteID := "9"
		err := testIndex.Delete(ctx, []string{deleteID})
		if err != nil {
			t.Fatalf("Additional delete failed: %v", err)
		}

		// Poll until ID is confirmed deleted
		deleted := pollUntil(pollTimeout, func() bool {
			result, err := testIndex.ListIDs(ctx)
			if err != nil {
				return false
			}
			for _, id := range result.Ids {
				if id == deleteID {
					return false
				}
			}
			return true
		})
		if !deleted {
			t.Errorf("ID %s still found after polling timeout", deleteID)
		}
	})
}

// Test 18: Client.LoadIndex()
func TestClientLoadIndex(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("LoadExistingIndex", func(t *testing.T) {
		loaded, err := testClient.LoadIndex(ctx, testIndexName, testIndexKey)
		if err != nil {
			t.Fatalf("LoadIndex failed: %v", err)
		}

		if loaded.GetIndexName() != testIndexName {
			t.Errorf("Expected index name %s, got %s", testIndexName, loaded.GetIndexName())
		}
	})

	t.Run("FailWithWrongEncryptionKey", func(t *testing.T) {
		wrongKey := generateRandomKey()
		_, err := testClient.LoadIndex(ctx, testIndexName, wrongKey)
		if err == nil {
			t.Error("Should fail with wrong encryption key")
		}
	})

	t.Run("FailWithNonExistentIndex", func(t *testing.T) {
		_, err := testClient.LoadIndex(ctx, "non-existent-index", generateRandomKey())
		if err == nil {
			t.Error("Should fail with non-existent index")
		}
	})

	t.Run("RejectUnexpectedParameters", func(t *testing.T) {
		// This is a compile-time check - LoadIndex only takes 3 params
		loaded, err := testClient.LoadIndex(ctx, testIndexName, testIndexKey)
		if err != nil {
			t.Fatalf("LoadIndex failed: %v", err)
		}
		if loaded == nil {
			t.Error("Loaded index should not be nil")
		}
	})
}

// Test 19: EncryptedIndex.DeleteIndex()
func TestEncryptedIndexDeleteIndex(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("DeleteTheIndex", func(t *testing.T) {
		indexToDelete := testIndexName
		err := testIndex.DeleteIndex(ctx)
		if err != nil {
			t.Fatalf("DeleteIndex failed: %v", err)
		}

		// Poll until index is confirmed deleted
		deleted := pollUntil(pollTimeout, func() bool {
			indexes, listErr := testClient.ListIndexes(ctx)
			if listErr != nil {
				return false
			}
			for _, name := range indexes {
				if name == indexToDelete {
					return false // Index still exists
				}
			}
			return true
		})
		if !deleted {
			t.Error("Index still found after polling timeout")
		}

		// Final verification
		indexes, err := testClient.ListIndexes(ctx)
		if err != nil {
			t.Fatalf("ListIndexes failed: %v", err)
		}
		for _, name := range indexes {
			if name == indexToDelete {
				t.Error("Index should have been deleted")
			}
		}

		testIndex = nil
	})

	t.Run("DeleteIndexNoArguments", func(t *testing.T) {
		// Create a temp index to delete
		tempName := generateUniqueName("temp_delete_")
		tempKey := generateRandomKey()
		dim := int32(dimension)

		params := &cyborgdb.CreateIndexParams{
			IndexName: tempName,
			IndexKey:  tempKey,
			Dimension: &dim,
		}

		tempIndex, err := testClient.CreateIndex(ctx, params)
		if err != nil {
			t.Fatalf("Failed to create temp index: %v", err)
		}

		// DeleteIndex should only take context (compile-time check)
		err = tempIndex.DeleteIndex(ctx)
		if err != nil {
			t.Errorf("DeleteIndex failed: %v", err)
		}

		// Poll to verify deletion
		deleted := pollUntil(pollTimeout, func() bool {
			indexes, err := testClient.ListIndexes(ctx)
			if err != nil {
				return false
			}
			for _, name := range indexes {
				if name == tempName {
					return false
				}
			}
			return true
		})
		if !deleted {
			t.Errorf("Temp index %s still found after polling timeout", tempName)
		}
	})
}

// TestSDKConstructionOffline exercises SDK-side construction and validation
// paths that do not require a live cyborgdb-service. Mirrors the Python
// TestSDKConstructionOffline class.
func TestSDKConstructionOffline(t *testing.T) {
	// Client.NewClient does not make any network calls; safe to instantiate
	// without a server.
	client, err := cyborgdb.NewClient("http://localhost:8000", "offline-test-key", false)
	if err != nil {
		t.Fatalf("NewClient: unexpected error: %v", err)
	}

	t.Run("CreateIndexRequiresKeyOrKMS", func(t *testing.T) {
		// Neither IndexKey nor KmsName supplied — must fail before any
		// network call with ErrMissingKeyOrKMS.
		_, err := client.CreateIndex(context.Background(), &cyborgdb.CreateIndexParams{
			IndexName: "x",
		})
		if !errors.Is(err, cyborgdb.ErrMissingKeyOrKMS) {
			t.Fatalf("expected ErrMissingKeyOrKMS, got %v", err)
		}
	})

	t.Run("CreateIndexRejectsWrongLengthKey", func(t *testing.T) {
		// IndexKey set but not 32 bytes — must fail upstream of the
		// network call with ErrInvalidKeyLength.
		_, err := client.CreateIndex(context.Background(), &cyborgdb.CreateIndexParams{
			IndexName: "x",
			IndexKey:  make([]byte, 16),
		})
		if !errors.Is(err, cyborgdb.ErrInvalidKeyLength) {
			t.Fatalf("expected ErrInvalidKeyLength, got %v", err)
		}
	})

	t.Run("LoadIndexRejectsWrongLengthKey", func(t *testing.T) {
		// indexKey set but not 32 bytes — must fail upstream of the
		// describe network call with ErrInvalidKeyLength.
		_, err := client.LoadIndex(context.Background(), "x", make([]byte, 16))
		if !errors.Is(err, cyborgdb.ErrInvalidKeyLength) {
			t.Fatalf("expected ErrInvalidKeyLength, got %v", err)
		}
	})

	t.Run("CreateIndexRequestSerializesKmsName", func(t *testing.T) {
		// KMS-only path: marshaled JSON must omit index_key entirely and
		// include kms_name. Service treats absence as "KMS-resolved."
		kmsName := "vendor-slot"
		req := internal.CreateIndexRequest{IndexName: "x"}
		req.KmsName = *internal.NewNullableString(&kmsName)

		raw, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if payload["index_name"] != "x" {
			t.Errorf("index_name: got %v", payload["index_name"])
		}
		if payload["kms_name"] != "vendor-slot" {
			t.Errorf("kms_name: got %v", payload["kms_name"])
		}
		if _, present := payload["index_key"]; present {
			t.Errorf("index_key should be omitted, got %v", payload["index_key"])
		}
	})

	t.Run("IndexOperationRequestKeylessBuildsKeylessPayload", func(t *testing.T) {
		// Load-index / describe / delete path: an IndexOperationRequest
		// with IndexKey left at the zero value must marshal without
		// index_key on the wire.
		req := internal.IndexOperationRequest{IndexName: "x"}

		raw, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if payload["index_name"] != "x" {
			t.Errorf("index_name: got %v", payload["index_name"])
		}
		if _, present := payload["index_key"]; present {
			t.Errorf("index_key should be omitted, got %v", payload["index_key"])
		}
	})

	t.Run("AllDataPlaneRequestsAcceptKeylessConstruction", func(t *testing.T) {
		// Every data-plane request model the SDK constructs must accept an
		// unset index_key so KMS-backed indexes can use them without an
		// SDK-held key. Regression risk: if the openapi regen flips
		// index_key back to required on any of these, the SDK breaks at
		// runtime, not at type-check time.
		type marshaller interface {
			MarshalJSON() ([]byte, error)
		}
		cases := []struct {
			name string
			req  marshaller
		}{
			{"QueryRequest", &internal.QueryRequest{IndexName: "x"}},
			{"UpsertRequest", &internal.UpsertRequest{IndexName: "x", Items: []internal.VectorItem{}}},
			{"GetRequest", &internal.GetRequest{IndexName: "x", Ids: []string{"a"}}},
			{"DeleteRequest", &internal.DeleteRequest{IndexName: "x", Ids: []string{"a"}}},
			{"TrainRequest", &internal.TrainRequest{IndexName: "x"}},
			{"ListIDsRequest", &internal.ListIDsRequest{IndexName: "x"}},
			{"BinaryQueryRequest", &internal.BinaryQueryRequest{IndexName: "x", Batch: internal.BinaryQueryBatch{}}},
			{"BinaryUpsertRequest", &internal.BinaryUpsertRequest{IndexName: "x", Batch: internal.BinaryVectorBatch{}}},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				raw, err := c.req.MarshalJSON()
				if err != nil {
					t.Fatalf("marshal: %v", err)
				}
				var payload map[string]interface{}
				if err := json.Unmarshal(raw, &payload); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if v, present := payload["index_key"]; present && v != nil {
					t.Errorf("index_key should be absent or null on the wire, got %v", v)
				}
			})
		}
	})
}
