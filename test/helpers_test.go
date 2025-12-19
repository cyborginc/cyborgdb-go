package test

import (
	"crypto/rand"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cyborginc/cyborgdb-go/internal"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// TestMain is the single entry point for all tests in this package
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

	// Cleanup api contract test resources
	cleanupAPIContractTests()

	os.Exit(code)
}

// generateRandomKey generates a cryptographically secure 32-byte key
func generateRandomKey() []byte {
	key := make([]byte, 32)
	rand.Read(key)
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
