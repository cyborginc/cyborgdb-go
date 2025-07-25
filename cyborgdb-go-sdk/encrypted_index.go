// encrypted_index.go
package cyborgdb

import (
	"context"
	"github.com/cyborginc/cyborgdb-go/internal"
)

// EncryptedIndex represents an encrypted vector index, similar to the TypeScript EncryptedIndex class.
// It provides methods for vector operations like upsert, query, get, delete, and train.
type EncryptedIndex struct {
	internal *internal.EncryptedIndex // Embedded internal implementation
}

// GetIndexName returns the name of the encrypted index.
// This corresponds to the TypeScript method: index.getIndexName()
//
// Returns:
//   - string: The index name
func (e *EncryptedIndex) GetIndexName() string {
	return e.internal.GetIndexName()
}

// GetIndexType returns the type of the encrypted index (e.g., "ivf", "ivfpq", "ivfflat").
// This corresponds to the TypeScript method: index.getIndexType()
//
// Returns:
//   - string: The index type
func (e *EncryptedIndex) GetIndexType() string {
	return e.internal.GetIndexType()
}

// GetIndexConfig returns the configuration of the encrypted index.
// This corresponds to the TypeScript method: index.getIndexConfig()
//
// Returns:
//   - IndexConfig: The index configuration
func (e *EncryptedIndex) GetIndexConfig() internal.IndexConfig {
	return e.internal.GetIndexConfig()
}

// IsTrained returns whether the index has been trained.
// This corresponds to the TypeScript method: index.isTrained()
//
// Returns:
//   - bool: true if the index is trained, false otherwise
func (e *EncryptedIndex) IsTrained() bool {
	return e.internal.IsTrained()
}

// Upsert adds or updates vectors in the encrypted index.
// This corresponds to the TypeScript method: index.upsert(items)
//
// Parameters:
//   - ctx: Context for request cancellation and timeouts
//   - items: Slice of VectorItem structs containing vectors and metadata to upsert
//
// Returns:
//   - error: nil on success, or an error describing what went wrong
//
// Example:
//   items := []cyborgdb.VectorItem{
//       {
//           Id:     "vec1",
//           Vector: []float32{1.0, 2.0, 3.0, 4.0},
//           Metadata: map[string]interface{}{
//               "category": "example",
//               "tags":     []string{"test", "demo"},
//           },
//           Contents: stringPtr("This is example content"),
//       },
//   }
//   err := index.Upsert(ctx, items)
func (e *EncryptedIndex) Upsert(ctx context.Context, items []VectorItem) error {
	return e.internal.Upsert(ctx, items)
}

// Query searches for nearest neighbors in the encrypted index.
// This method supports multiple calling patterns:
// 1. Direct parameters: Query(ctx, queryVectors, topK, nProbes, greedy, filters, include)
// 2. QueryRequest struct: Query(ctx, queryRequest)
// 3. BatchQueryRequest struct: Query(ctx, batchQueryRequest)
//
// Parameters:
//   - ctx: Context for request cancellation and timeouts
//   - args: Variable arguments supporting multiple patterns:
//     * (queryVectors, topK, nProbes, greedy, filters, include) - Direct parameter style
//     * (QueryRequest) - Single query request struct
//     * (BatchQueryRequest) - Batch query request struct
//
// Returns:
//   - *QueryResponse: Search results with nearest neighbors and metadata
//   - error: nil on success, or an error describing what went wrong
//
// Examples:
//   // Direct parameters - single vector
//   results, err := index.Query(ctx, []float32{1.0, 2.0, 3.0}, 10, 5, false, nil, []string{"metadata"})
//
//   // Direct parameters - batch vectors
//   batch := [][]float32{{1.0, 2.0}, {3.0, 4.0}}
//   results, err := index.Query(ctx, batch, 5, 3, false, nil, []string{"distance", "metadata"})
//
//   // Using QueryRequest struct
//   queryReq := &QueryRequest{
//       QueryVector: []float32{1.0, 2.0, 3.0},
//       TopK: 10,
//       NProbes: 5,
//       Include: []string{"metadata"},
//   }
//   results, err := index.Query(ctx, queryReq)
//
//   // Using BatchQueryRequest struct
//   batchReq := &BatchQueryRequest{
//       QueryVectors: [][]float32{{1.0, 2.0}, {3.0, 4.0}},
//       TopK: &topK,
//       NProbes: &nProbes,
//       Include: []string{"distance", "metadata"},
//   }
//   results, err := index.Query(ctx, batchReq)
func (e *EncryptedIndex) Query(ctx context.Context, args ...interface{}) (*QueryResponse, error) {
	// Delegate to the internal implementation which has the full logic
	return e.internal.Query(ctx, args...)
}

// Get retrieves specific vectors from the encrypted index by their IDs.
// This corresponds to the TypeScript method: index.get(ids, include)
//
// Parameters:
//   - ctx: Context for request cancellation and timeouts
//   - ids: Slice of string IDs identifying the vectors to retrieve
//   - include: Fields to include in response (e.g., ["vector", "metadata", "contents"])
//
// Returns:
//   - []VectorItem: Slice of retrieved vectors with requested fields
//   - error: nil on success, or an error describing what went wrong
//
// Example:
//   ids := []string{"vec1", "vec2", "vec3"}
//   include := []string{"vector", "metadata", "contents"}
//   vectors, err := index.Get(ctx, ids, include)
//   if err != nil {
//       log.Printf("Failed to retrieve vectors: %v", err)
//   }
//   for _, vec := range vectors {
//       fmt.Printf("Retrieved vector %s with %d dimensions\n", vec.Id, len(vec.Vector))
//   }
func (e *EncryptedIndex) Get(ctx context.Context, ids []string, include []string) ([]VectorItem, error) {
	return e.internal.Get(ctx, ids, include)
}

// Delete removes specific vectors from the encrypted index by their IDs.
// This corresponds to the TypeScript method: index.delete(ids)
//
// Parameters:
//   - ctx: Context for request cancellation and timeouts
//   - ids: Slice of string IDs identifying the vectors to delete
//
// Returns:
//   - error: nil on success, or an error describing what went wrong
//
// Example:
//   idsToDelete := []string{"vec1", "vec2", "vec3"}
//   err := index.Delete(ctx, idsToDelete)
//   if err != nil {
//       log.Printf("Failed to delete vectors: %v", err)
//   }
func (e *EncryptedIndex) Delete(ctx context.Context, ids []string) error {
	return e.internal.Delete(ctx, ids)
}

// Train optimizes the encrypted index for better search performance.
// This corresponds to the TypeScript method: index.train(batchSize, maxIters, tolerance)
//
// Parameters:
//   - ctx: Context for request cancellation and timeouts
//   - batchSize: Number of vectors to process in each training batch
//   - maxIters: Maximum number of training iterations
//   - tolerance: Convergence tolerance for training
//
// Returns:
//   - error: nil on success, or an error describing what went wrong
//
// Example:
//   err := index.Train(ctx, 100, 50, 1e-5)
//   if err != nil {
//       log.Printf("Failed to train index: %v", err)
//   } else {
//       fmt.Println("Index training completed successfully")
//   }
func (e *EncryptedIndex) Train(ctx context.Context, batchSize int32, maxIters int32, tolerance float64) error {
	return e.internal.Train(ctx, batchSize, maxIters, tolerance)
}

// DeleteIndex permanently removes the entire encrypted index from CyborgDB.
// This corresponds to the TypeScript method: index.deleteIndex()
//
// Parameters:
//   - ctx: Context for request cancellation and timeouts
//
// Returns:
//   - error: nil on success, or an error describing what went wrong
//
// Example:
//   err := index.DeleteIndex(ctx)
//   if err != nil {
//       log.Printf("Failed to delete index: %v", err)
//   } else {
//       fmt.Println("Index deleted successfully")
//   }
func (e *EncryptedIndex) DeleteIndex(ctx context.Context) error {
	return e.internal.DeleteIndex(ctx)
}