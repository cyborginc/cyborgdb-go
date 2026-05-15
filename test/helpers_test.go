package test

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	mrand "math/rand"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
	"github.com/cyborginc/cyborgdb-go/internal"
)

// TestMain is the single entry point for all tests in this package
func TestMain(m *testing.M) {
	// Load environment variables (ignore error - file may not exist)
	_ = godotenv.Load("../.env.local")

	// Validate test environment
	if os.Getenv("CYBORGDB_API_KEY") == "" {
		fmt.Println("ERROR: CYBORGDB_API_KEY environment variable is required for testing")
		os.Exit(1)
	}

	// Initialize api contract test data
	initAPIContractTestData()

	// Run tests
	code := m.Run()

	// Cleanup api contract test resources
	cleanupAPIContractTests()

	os.Exit(code)
}

// generateRandomKey generates a cryptographically secure 32-byte key
func generateRandomKey() []byte {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	return key
}

// generateUniqueName generates a unique name with the given prefix
func generateUniqueName(prefix string) string {
	if prefix == "" {
		prefix = "test_"
	}
	return fmt.Sprintf("%s%s", prefix, uuid.New().String())
}

// generateTestVectors generates test vectors with varied values
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

// waitForPropagation waits for operations to propagate
//
//nolint:unparam // duration kept as parameter for call-site readability
func waitForPropagation(duration time.Duration) {
	time.Sleep(duration)
}

// pollUntil polls a condition function until it returns true or timeout is reached.
// Returns true if condition was met, false if timeout occurred.
func pollUntil(timeout time.Duration, interval time.Duration, condition func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(interval)
	}
	return false
}

// Polling configuration
const (
	pollTimeout  = 10 * time.Second
	pollInterval = 500 * time.Millisecond
)

// getQueryResultItems extracts query result items from the Results union type.
// For single queries, returns the result array directly.
// For batch queries, returns only the first result set (use getBatchQueryResults for full batch validation).
func getQueryResultItems(results *internal.Results) []internal.QueryResultItem {
	if results == nil {
		return nil
	}
	if results.ArrayOfQueryResultItem != nil {
		return *results.ArrayOfQueryResultItem
	}
	if results.ArrayOfArrayOfQueryResultItem != nil && len(*results.ArrayOfArrayOfQueryResultItem) > 0 {
		return (*results.ArrayOfArrayOfQueryResultItem)[0]
	}
	return nil
}

// getBatchQueryResults extracts all result sets from a batch query.
// Returns a slice of result slices, one per query vector in the batch.
func getBatchQueryResults(results *internal.Results) [][]internal.QueryResultItem {
	if results == nil {
		return nil
	}
	if results.ArrayOfArrayOfQueryResultItem != nil {
		return *results.ArrayOfArrayOfQueryResultItem
	}
	// Single query result - wrap in outer slice for consistent interface
	if results.ArrayOfQueryResultItem != nil {
		return [][]internal.QueryResultItem{*results.ArrayOfQueryResultItem}
	}
	return nil
}

// generateRandomVectors generates random float32 vectors.
func generateRandomVectors(count, dimension int) [][]float32 {
	vectors := make([][]float32, count)
	for i := 0; i < count; i++ {
		vectors[i] = make([]float32, dimension)
		for j := 0; j < dimension; j++ {
			vectors[i][j] = mrand.Float32()
		}
	}
	return vectors
}

// vectorsApproxEqual checks if two float32 vectors are approximately equal.
func vectorsApproxEqual(a, b []float32, rtol float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		diff := math.Abs(float64(a[i]) - float64(b[i]))
		limit := rtol * math.Max(math.Abs(float64(a[i])), math.Abs(float64(b[i])))
		if diff > limit+1e-8 {
			return false
		}
	}
	return true
}

// testBaseURL returns the CyborgDB base URL from env or localhost default.
func testBaseURL() string {
	u := os.Getenv("CYBORGDB_BASE_URL")
	if u == "" {
		return "http://localhost:8000"
	}
	return u
}

// testAPIKey returns the CyborgDB API key from env.
func testAPIKey() string {
	return os.Getenv("CYBORGDB_API_KEY")
}

// newIsolatedClient creates a fresh CyborgDB client. Safe to call from any goroutine.
func newIsolatedClient(t *testing.T) *cyborgdb.Client {
	t.Helper()
	client, err := cyborgdb.NewClient(testBaseURL(), testAPIKey())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	return client
}

// newIsolatedIndex creates a uniquely-named IVFFlat index with its own cleanup.
// Must be called from the test goroutine (uses t.Fatalf). Registers cleanup via t.Cleanup.
//
//nolint:unparam // dimension kept as parameter for call-site readability
func newIsolatedIndex(t *testing.T, client *cyborgdb.Client, prefix string, dimension int32) (*cyborgdb.EncryptedIndex, string) {
	t.Helper()
	name := generateUniqueName(prefix + "_")
	key := generateRandomKey()
	metric := "euclidean"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	index, err := client.CreateIndex(ctx, &cyborgdb.CreateIndexParams{
		IndexName: name,
		IndexKey:  key,
		Dimension: &dimension,
		Metric:    &metric,
	})
	if err != nil {
		t.Fatalf("Failed to create index %s: %v", name, err)
	}
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanCancel()
		_ = index.DeleteIndex(cleanCtx)
	})
	return index, name
}

// concUpsertBatch upserts a batch of random vectors. Returns error instead of calling
// t.Fatal, making it safe to call from any goroutine.
func concUpsertBatch(index *cyborgdb.EncryptedIndex, idPrefix string, count, dimension int) ([]string, error) {
	vectors := generateRandomVectors(count, dimension)
	ids := make([]string, count)
	for i := 0; i < count; i++ {
		ids[i] = fmt.Sprintf("%s_%d", idPrefix, i)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := index.UpsertVectors(ctx, ids, vectors, nil)
	if err != nil {
		return nil, fmt.Errorf("upsertBatch(%s): %w", idPrefix, err)
	}
	return ids, nil
}

// seedIndex upserts seed data from the test goroutine. Calls t.Fatalf on error.
func seedIndex(t *testing.T, index *cyborgdb.EncryptedIndex, prefix string, count, dimension int) {
	t.Helper()
	_, err := concUpsertBatch(index, prefix, count, dimension)
	if err != nil {
		t.Fatalf("seedIndex failed: %v", err)
	}
}
