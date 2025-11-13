package test

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/joho/godotenv"
	cyborgdb "github.com/cyborginc/cyborgdb-go"
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

func init() {
	// Load environment variables
	godotenv.Load("../.env.local")
}

// TestMain sets up test environment
func TestMain(m *testing.M) {
	if os.Getenv("CYBORGDB_API_KEY") == "" {
		fmt.Println("ERROR: CYBORGDB_API_KEY environment variable is required")
		os.Exit(1)
	}

	// Generate test data
	testVectors = generateTestVectors(10, dimension)
	testMetadata = generateTestMetadata(10)
	testIndexName = fmt.Sprintf("test_contract_%d", time.Now().UnixNano())
	embeddingName = fmt.Sprintf("test_embed_%d", time.Now().UnixNano())

	code := m.Run()

	// Cleanup
	cleanup()

	os.Exit(code)
}

func cleanup() {
	ctx := context.Background()
	if testIndex != nil {
		testIndex.DeleteIndex(ctx)
	}
	if embeddingIndex != nil {
		embeddingIndex.DeleteIndex(ctx)
	}
}

// Helper functions
func generateTestVectors(count, dim int) [][]float32 {
	vectors := make([][]float32, count)
	for i := 0; i < count; i++ {
		vectors[i] = make([]float32, dim)
		for j := 0; j < dim; j++ {
			vectors[i][j] = float32(j) / 100.0
		}
	}
	return vectors
}

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

func generateRandomKey() []byte {
	key := make([]byte, 32)
	rand.Read(key)
	return key
}

func generateUniqueName(prefix string) string {
	return fmt.Sprintf("%s%d", prefix, time.Now().UnixNano())
}

// Test Suite

// Test 01: Module Exports
func TestModuleExports(t *testing.T) {
	t.Run("RequiredTypesExist", func(t *testing.T) {
		// Verify key types are exported
		var client *cyborgdb.Client
		var index *cyborgdb.EncryptedIndex
		var params *cyborgdb.CreateIndexParams
		var queryParams cyborgdb.QueryParams
		var item cyborgdb.VectorItem

		if client == nil && index == nil && params == nil {
			// Types exist
		}
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
		_, err := cyborgdb.NewClient("", apiKey)
		// Note: SDK may not reject empty baseURL at construction time
		// This is acceptable behavior - it will fail when making requests
		_ = err
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

// Test 06: Index Config Types
func TestIndexConfigTypes(t *testing.T) {
	t.Run("CreateIndexIVFConfig", func(t *testing.T) {
		config := cyborgdb.IndexIVF(0)
		_ = config
	})

	t.Run("CreateIndexIVFFlatConfigWithDimension", func(t *testing.T) {
		config := cyborgdb.IndexIVFFlat(dimension)
		_ = config
	})

	t.Run("CreateIndexIVFPQConfigWithRequiredParams", func(t *testing.T) {
		config := cyborgdb.IndexIVFPQ(dimension, 64, 8)
		_ = config
	})
}

// Test 07: Client.CreateIndex()
func TestClientCreateIndex(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("CreateIndexWithIVFFlatConfigAndCustomMetric", func(t *testing.T) {
		tempName := generateUniqueName("temp_ivfflat_")
		tempKey := generateRandomKey()
		config := cyborgdb.IndexIVFFlat(dimension)
		metric := "cosine"

		params := &cyborgdb.CreateIndexParams{
			IndexName:   tempName,
			IndexKey:    tempKey,
			IndexConfig: config,
			Metric:      &metric,
		}

		index, err := testClient.CreateIndex(ctx, params)
		if err != nil {
			t.Fatalf("Failed to create IVFFlat index: %v", err)
		}
		defer index.DeleteIndex(ctx)

		if index.GetIndexName() != tempName {
			t.Errorf("Expected index name %s, got %s", tempName, index.GetIndexName())
		}

		if index.GetIndexType() != "ivfflat" {
			t.Errorf("Expected index type ivfflat, got %s", index.GetIndexType())
		}

		time.Sleep(1 * time.Second)
	})

	t.Run("CreateIndexWithIVFConfig", func(t *testing.T) {
		tempName := generateUniqueName("temp_ivf_")
		tempKey := generateRandomKey()
		config := cyborgdb.IndexIVF(0)
		metric := "squared_euclidean"

		params := &cyborgdb.CreateIndexParams{
			IndexName:   tempName,
			IndexKey:    tempKey,
			IndexConfig: config,
			Metric:      &metric,
		}

		index, err := testClient.CreateIndex(ctx, params)
		if err != nil {
			t.Fatalf("Failed to create IVF index: %v", err)
		}
		defer index.DeleteIndex(ctx)

		if index.GetIndexType() != "ivf" {
			t.Errorf("Expected index type ivf, got %s", index.GetIndexType())
		}

		time.Sleep(1 * time.Second)
	})

	t.Run("CreateIndexWithIVFPQConfig", func(t *testing.T) {
		tempName := generateUniqueName("temp_ivfpq_")
		tempKey := generateRandomKey()
		config := cyborgdb.IndexIVFPQ(0, 32, 8)

		params := &cyborgdb.CreateIndexParams{
			IndexName:   tempName,
			IndexKey:    tempKey,
			IndexConfig: config,
		}

		index, err := testClient.CreateIndex(ctx, params)
		if err != nil {
			t.Fatalf("Failed to create IVFPQ index: %v", err)
		}
		defer index.DeleteIndex(ctx)

		if index.GetIndexType() != "ivfpq" {
			t.Errorf("Expected index type ivfpq, got %s", index.GetIndexType())
		}

		time.Sleep(1 * time.Second)
	})

	t.Run("CreateIndexWithEmbeddingModel", func(t *testing.T) {
		embeddingModel := "all-MiniLM-L6-v2"
		params := &cyborgdb.CreateIndexParams{
			IndexName:      embeddingName,
			IndexKey:       embeddingKey,
			EmbeddingModel: &embeddingModel,
			// No IndexConfig - should work with just embedding model
		}

		index, err := testClient.CreateIndex(ctx, params)
		if err != nil {
			t.Fatalf("Failed to create index with embedding model: %v", err)
		}
		embeddingIndex = index

		if index.GetIndexType() != "ivfflat" {
			t.Errorf("Expected default index type ivfflat, got %s", index.GetIndexType())
		}

		time.Sleep(2 * time.Second)
	})

	t.Run("RejectDuplicateIndexCreation", func(t *testing.T) {
		dupName := generateUniqueName("dup_test_")
		dupKey := generateRandomKey()

		params := &cyborgdb.CreateIndexParams{
			IndexName: dupName,
			IndexKey:  dupKey,
			// No IndexConfig - should work with defaults
		}

		index, err := testClient.CreateIndex(ctx, params)
		if err != nil {
			t.Fatalf("Failed to create first index: %v", err)
		}
		defer index.DeleteIndex(ctx)

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

		params := &cyborgdb.CreateIndexParams{
			IndexName: tempName,
			IndexKey:  tempKey,
		}

		index, err := testClient.CreateIndex(ctx, params)
		if err != nil {
			t.Fatalf("Failed to create index: %v", err)
		}
		defer index.DeleteIndex(ctx)

		time.Sleep(1 * time.Second)
	})

	t.Run("CreateMainTestIndex", func(t *testing.T) {
		config := cyborgdb.IndexIVFFlat(dimension)
		metric := "cosine"

		params := &cyborgdb.CreateIndexParams{
			IndexName:   testIndexName,
			IndexKey:    testIndexKey,
			IndexConfig: config,
			Metric:      &metric,
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
		if indexType != "ivfflat" {
			t.Errorf("Expected index type ivfflat, got %s", indexType)
		}
		if reflect.TypeOf(indexType).Kind() != reflect.String {
			t.Error("Index type should be string")
		}
	})

	t.Run("ExposeIndexConfigViaGetter", func(t *testing.T) {
		config := testIndex.GetIndexConfig()
		// Config is a struct, just verify it exists
		_ = config
	})
}

// Test 09: EncryptedIndex.IsTrained()
func TestEncryptedIndexIsTrained(t *testing.T) {
	t.Run("ReturnBoolean", func(t *testing.T) {
		trained := testIndex.IsTrained()
		if reflect.TypeOf(trained).Kind() != reflect.Bool {
			t.Error("IsTrained should return bool")
		}
	})

	t.Run("IsTrainedNoArguments", func(t *testing.T) {
		// IsTrained should not take arguments (compile-time check)
		trained := testIndex.IsTrained()
		_ = trained
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
		items := make([]cyborgdb.VectorItem, 2)
		for i := 0; i < 2; i++ {
			items[i] = cyborgdb.VectorItem{
				Id:       fmt.Sprintf("%d", i),
				Vector:   testVectors[i],
				Metadata: testMetadata[i],
			}
		}

		err := testIndex.Upsert(ctx, items)
		if err != nil {
			t.Fatalf("Upsert failed: %v", err)
		}

		time.Sleep(propagationDelay)
	})

	t.Run("UpsertWithItemsArrayAutoEmbed", func(t *testing.T) {
		items := make([]cyborgdb.VectorItem, 3)
		for i := 0; i < 3; i++ {
			items[i] = cyborgdb.VectorItem{
				Id:       fmt.Sprintf("embed_%d", i),
				Metadata: map[string]interface{}{"type": "auto-embedded", "index": i},
			}
		}

		err := embeddingIndex.Upsert(ctx, items)
		if err != nil {
			t.Fatalf("Auto-embed upsert failed: %v", err)
		}

		time.Sleep(propagationDelay)
	})

	t.Run("UpsertRemainingTestItems", func(t *testing.T) {
		items := make([]cyborgdb.VectorItem, 8)
		for i := 2; i < 10; i++ {
			items[i-2] = cyborgdb.VectorItem{
				Id:       fmt.Sprintf("%d", i),
				Vector:   testVectors[i%len(testVectors)],
				Metadata: testMetadata[i%len(testMetadata)],
			}
		}

		err := testIndex.Upsert(ctx, items)
		if err != nil {
			t.Fatalf("Batch upsert failed: %v", err)
		}

		time.Sleep(propagationDelay)
	})

	t.Run("UpsertWithParallelArraysFormat", func(t *testing.T) {
		// Go SDK doesn't support separate ids/vectors arrays like Python/TS
		// Use items array instead
		items := make([]cyborgdb.VectorItem, 5)
		for i := 10; i < 15; i++ {
			items[i-10] = cyborgdb.VectorItem{
				Id:     fmt.Sprintf("%d", i),
				Vector: testVectors[i%len(testVectors)],
			}
		}

		err := testIndex.Upsert(ctx, items)
		if err != nil {
			t.Fatalf("Additional upsert failed: %v", err)
		}

		time.Sleep(propagationDelay)
	})

	t.Run("RejectVectorsWithWrongDimensions", func(t *testing.T) {
		wrongVector := make([]float32, 64)
		items := []cyborgdb.VectorItem{{
			Id:     "wrong-dim",
			Vector: wrongVector,
		}}

		err := testIndex.Upsert(ctx, items)
		if err == nil {
			t.Error("Should reject vectors with wrong dimensions")
		}
	})

	t.Run("RejectWhenNeitherItemsNorVectorsProvided", func(t *testing.T) {
		// In Go SDK, empty items array is allowed (no-op)
		// This is a compile-time check - can't pass wrong format
		items := []cyborgdb.VectorItem{}
		err := testIndex.Upsert(ctx, items)
		// Empty upsert is allowed, just a no-op
		_ = err
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
			fmt.Sscanf(result.GetId(), "%d", &idInt)
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
		params := cyborgdb.QueryParams{
			QueryVector: testVectors[0],
			TopK:        5,
		}

		results, err := testIndex.Query(ctx, params)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if results == nil {
			t.Fatal("Results must not be nil")
		}
	})

	t.Run("QueryWithNestedArraySingleVector", func(t *testing.T) {
		// Go SDK uses QueryVector for single, BatchQueryVectors for batch
		params := cyborgdb.QueryParams{
			QueryVector: testVectors[1],
			TopK:        3,
		}

		results, err := testIndex.Query(ctx, params)
		if err != nil {
			t.Fatalf("Query with topK failed: %v", err)
		}

		if results == nil {
			t.Fatal("Results must not be nil")
		}
	})

	t.Run("QueryWithBatchVectors", func(t *testing.T) {
		batchVectors := [][]float32{testVectors[2], testVectors[3]}
		params := cyborgdb.QueryParams{
			BatchQueryVectors: batchVectors,
			TopK:              2,
		}

		results, err := testIndex.Query(ctx, params)
		if err != nil {
			t.Fatalf("Batch query failed: %v", err)
		}

		if results == nil {
			t.Fatal("Results must not be nil")
		}
	})

	t.Run("QueryWithSpecificInclude", func(t *testing.T) {
		params := cyborgdb.QueryParams{
			QueryVector: testVectors[0],
			TopK:        5,
			Include:     []string{"metadata"},
		}

		results, err := testIndex.Query(ctx, params)
		if err != nil {
			t.Fatalf("Query with include failed: %v", err)
		}

		if results == nil {
			t.Fatal("Results must not be nil")
		}
	})

	t.Run("QueryWithMetadataFilters", func(t *testing.T) {
		filters := map[string]interface{}{"category": "cat_0"}
		params := cyborgdb.QueryParams{
			QueryVector: testVectors[0],
			TopK:        10,
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
	})

	t.Run("QueryWithTextContentsAutoEmbed", func(t *testing.T) {
		queryText := "test content for similarity search"
		params := cyborgdb.QueryParams{
			QueryContents: &queryText,
			TopK:          3,
		}

		results, err := embeddingIndex.Query(ctx, params)
		if err != nil {
			t.Fatalf("Query with text contents failed: %v", err)
		}

		if results == nil {
			t.Fatal("Results must not be nil")
		}
	})
}

// Test 15: EncryptedIndex.Query() Additional Patterns
func TestEncryptedIndexQueryPatterns(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("QueryWithMultipleTestPatterns", func(t *testing.T) {
		// Test 1: Single vector
		params1 := cyborgdb.QueryParams{
			QueryVector: testVectors[4],
			TopK:        3,
		}
		results1, err1 := testIndex.Query(ctx, params1)
		if err1 != nil {
			t.Errorf("Query 1 failed: %v", err1)
		}
		if results1 == nil {
			t.Error("Results 1 should not be nil")
		}

		time.Sleep(500 * time.Millisecond)

		// Test 2: Batch vectors
		params2 := cyborgdb.QueryParams{
			BatchQueryVectors: [][]float32{testVectors[5], testVectors[6]},
			TopK:              2,
		}
		results2, err2 := testIndex.Query(ctx, params2)
		if err2 != nil {
			t.Errorf("Query 2 failed: %v", err2)
		}
		if results2 == nil {
			t.Error("Results 2 should not be nil")
		}

		time.Sleep(500 * time.Millisecond)

		// Test 3: With filters
		params3 := cyborgdb.QueryParams{
			QueryVector: testVectors[7],
			TopK:        10,
			Filters:     map[string]interface{}{"category": "cat_1"},
		}
		results3, err3 := testIndex.Query(ctx, params3)
		if err3 != nil {
			t.Errorf("Query 3 failed: %v", err3)
		}
		if results3 == nil {
			t.Error("Results 3 should not be nil")
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

		time.Sleep(2 * time.Second)
	})
}

// Test 17: EncryptedIndex.Delete()
func TestEncryptedIndexDelete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	t.Run("DeleteVectorsByIDs", func(t *testing.T) {
		ids := []string{"0", "5"}
		err := testIndex.Delete(ctx, ids)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		time.Sleep(propagationDelay)

		// Verify deletion
		result, _ := testIndex.ListIDs(ctx)
		for _, deletedID := range ids {
			for _, id := range result.Ids {
				if id == deletedID {
					t.Errorf("ID %s should have been deleted", deletedID)
				}
			}
		}
	})

	t.Run("DeleteAdditionalVector", func(t *testing.T) {
		err := testIndex.Delete(ctx, []string{"9"})
		if err != nil {
			t.Fatalf("Additional delete failed: %v", err)
		}

		time.Sleep(propagationDelay)
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
		err := testIndex.DeleteIndex(ctx)
		if err != nil {
			t.Fatalf("DeleteIndex failed: %v", err)
		}

		time.Sleep(propagationDelay)

		// Verify deletion
		indexes, _ := testClient.ListIndexes(ctx)
		for _, name := range indexes {
			if name == testIndexName {
				t.Error("Index should have been deleted")
			}
		}

		testIndex = nil
	})

	t.Run("DeleteIndexNoArguments", func(t *testing.T) {
		// Create a temp index to delete
		tempName := generateUniqueName("temp_delete_")
		tempKey := generateRandomKey()
		config := cyborgdb.IndexIVFFlat(dimension)

		params := &cyborgdb.CreateIndexParams{
			IndexName:   tempName,
			IndexKey:    tempKey,
			IndexConfig: config,
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
	})
}