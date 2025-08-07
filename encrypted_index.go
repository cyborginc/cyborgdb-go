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

// GetIndexType returns the type of the encrypted index.
//
// The index type determines the underlying algorithm used for similarity search:
//   - "ivf": Inverted File index (good balance of speed and accuracy)
//   - "ivfpq": IVF with Product Quantization (memory efficient, compressed vectors)
//   - "ivfflat": IVF with flat vectors (highest accuracy, more memory usage)
//
// Returns:
//   - string: The index type ("ivf", "ivfpq", or "ivfflat")
func (e *EncryptedIndex) GetIndexType() string {
	return e.internal.GetIndexType()
}

// GetIndexConfig returns the configuration of the encrypted index.
//
// The configuration includes parameters like dimension, metric, number of lists,
// and type-specific settings such as PQ parameters for IVFPQ indexes.
//
// Returns:
//   - internal.IndexConfig: The complete index configuration
func (e *EncryptedIndex) GetIndexConfig() internal.IndexConfig {
	return e.internal.GetIndexConfig()
}

// IsTrained returns whether the index has been trained.
//
// Training optimizes the index structure for better query performance, especially
// for large datasets. Trained indexes typically provide faster and more accurate
// similarity searches.
//
// Returns:
//   - bool: true if the index has been trained, false otherwise
func (e *EncryptedIndex) IsTrained() bool {
	return e.internal.IsTrained()
}

// Upsert adds or updates vectors in the encrypted index.
//
// Vectors with the same ID will be updated, while new IDs will be inserted.
// All vector data is encrypted before storage. Each vector can include:
//   - Vector: The high-dimensional embedding (required)
//   - Metadata: Structured data for filtering and retrieval
//   - Contents: Text or binary content associated with the vector
//
// Parameters:
//   - ctx: Context for request cancellation, timeouts, and tracing
//   - items: Slice of VectorItem structs containing vectors and associated data
//
// Returns:
//   - error: nil on success, or an error describing what went wrong
//
// Note: Vector dimensions must match the index configuration.
// Large batches are more efficient than individual upserts.
func (e *EncryptedIndex) Upsert(ctx context.Context, items []VectorItem) error {
	return e.internal.Upsert(ctx, items)
}

// Query searches for nearest neighbors in the encrypted index.
// This method supports multiple calling patterns for flexibility and ease of use:
// 1. Direct parameters: Query(ctx, queryVectors, topK, nProbes, greedy, filters, include)
// 2. QueryRequest struct: Query(ctx, queryRequest)
// 3. BatchQueryRequest struct: Query(ctx, batchQueryRequest)
//
// The query performs similarity search using the index's configured distance metric
// and returns the most similar vectors along with their distances and metadata.
//
// Parameters:
//   - ctx: Context for request cancellation, timeouts, and tracing
//   - args: Variable arguments supporting multiple patterns (see examples below)
//
// Returns:
//   - *QueryResponse: Search results with nearest neighbors, distances, and metadata
//   - error: nil on success, or an error describing what went wrong
//
// Query Parameters:
//   - queryVectors: Single vector []float32 or batch [][]float32
//   - topK: Number of nearest neighbors to return (default: 100)
//   - nProbes: Number of clusters to search (higher = more accurate, slower)
//   - greedy: Use greedy search for potentially faster results
//   - filters: Metadata filters for narrowing results
//   - include: Fields to include in response ("metadata", "distance", "contents", "vector")
func (e *EncryptedIndex) Query(ctx context.Context, args ...interface{}) (*QueryResponse, error) {
	// Delegate to the internal implementation which has the full logic
	return e.internal.Query(ctx, args...)
}

// Get retrieves specific vectors from the encrypted index by their IDs.
//
// This method allows you to fetch vectors by their unique identifiers and specify
// which fields to include in the response. It's useful for retrieving known vectors
// or getting full details after a similarity search.
//
// Parameters:
//   - ctx: Context for request cancellation, timeouts, and tracing
//   - ids: Slice of string IDs identifying the vectors to retrieve
//   - include: Fields to include in response (e.g., ["vector", "metadata", "contents"])
//
// Returns:
//   - *GetResponse: Response containing retrieved vectors with requested fields
//   - error: nil on success, or an error describing what went wrong
//
//   // Retrieve only metadata for efficiency
//   metadataOnly, err := index.Get(ctx, ids, []string{"metadata"})
//
// Available include fields:
//   - "vector": The vector embeddings
//   - "metadata": Associated metadata
//   - "contents": Text/binary content
//
// Note: If a requested ID doesn't exist, it won't appear in the results.
// For trained IVFPQ indexes, retrieved vectors may be compressed.
func (e *EncryptedIndex) Get(ctx context.Context, ids []string, include []string) (*GetResponse, error) {
	return e.internal.Get(ctx, ids, include)
}

// Delete removes specific vectors from the encrypted index by their IDs.
//
// This operation permanently removes the specified vectors and all their associated
// data (metadata, contents) from the index. The deletion cannot be undone.
//
// Parameters:
//   - ctx: Context for request cancellation, timeouts, and tracing
//   - ids: Slice of string IDs identifying the vectors to delete
//
// Returns:
//   - error: nil on success, or an error describing what went wrong
//
// Note: Deleting non-existent IDs will not cause an error.
// Large batches are more efficient than individual deletions.
func (e *EncryptedIndex) Delete(ctx context.Context, ids []string) error {
	return e.internal.Delete(ctx, ids)
}

// Train optimizes the encrypted index for better search performance.
//
// Training uses machine learning techniques to analyze the vector distribution and
// optimize the index structure. This typically improves query speed and accuracy,
// especially for large datasets. Training is recommended when you have a significant
// number of vectors (typically 10x the number of clusters).
//
// Parameters:
//   - ctx: Context for request cancellation, timeouts, and tracing
//   - batchSize: Number of vectors to process in each training batch (affects memory usage)
//   - maxIters: Maximum number of training iterations to perform
//   - tolerance: Convergence tolerance for training (smaller = more precise training)
//
// Returns:
//   - error: nil on success, or an error describing what went wrong
//
// Training Guidelines:
//   - Recommended after upserting significant amounts of data
//   - Larger batchSize = more memory usage but potentially faster training
//   - More maxIters = potentially better optimization but longer training time
//   - Smaller tolerance = more precise but longer training
//   - Training time scales with dataset size and index complexity
//
// Note: Training is resource-intensive and may take considerable time for large datasets.
// The index remains available for queries during training, but performance may vary.
func (e *EncryptedIndex) Train(ctx context.Context, batchSize int32, maxIters int32, tolerance float64) error {
	return e.internal.Train(ctx, batchSize, maxIters, tolerance)
}

// DeleteIndex permanently removes the entire encrypted index from CyborgDB.
//
// This operation is irreversible and will delete ALL vectors, metadata, and
// configuration associated with the index. Use with extreme caution as there
// is no way to recover the data once the index is deleted.
//
// Parameters:
//   - ctx: Context for request cancellation, timeouts, and tracing
//
// Returns:
//   - error: nil on success, or an error describing what went wrong
//
// WARNING: This operation cannot be undone. All data in the index will be permanently lost.
// Consider exporting important data before deletion if recovery might be needed.
//
// Note: After successful deletion, this EncryptedIndex instance should not be used
// for further operations as the underlying index no longer exists.
func (e *EncryptedIndex) DeleteIndex(ctx context.Context) error {
	return e.internal.DeleteIndex(ctx)
}