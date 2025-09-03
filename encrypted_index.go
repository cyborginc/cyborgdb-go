// encrypted_index.go
package cyborgdb

import (
	"context"
	"fmt"

	"github.com/cyborginc/cyborgdb-go/internal"
)

var (
	// ErrQueryVectorsInvalidType is returned when QueryOptions.QueryVectors has an invalid type.
	ErrQueryVectorsInvalidType = fmt.Errorf("queryVectors must be []float32 for single vector queries or [][]float32 for batch queries")
	// ErrMissingQueryInput is returned when neither QueryVectors nor QueryContents is provided.
	ErrMissingQueryInput = fmt.Errorf("either queryVectors or queryContents must be provided")
)

// EncryptedIndex is the public handle for an encrypted vector index.
// It wraps the internal implementation and exposes friendly methods.
type EncryptedIndex struct {
	internal *internal.EncryptedIndex
}

// GetIndexName returns the index name.
func (e *EncryptedIndex) GetIndexName() string { return e.internal.GetIndexName() }

// GetIndexType returns the index type ("ivf", "ivfpq", or "ivfflat").
func (e *EncryptedIndex) GetIndexType() string { return e.internal.GetIndexType() }

// GetIndexConfig returns the full index configuration.
func (e *EncryptedIndex) GetIndexConfig() internal.IndexConfig { return e.internal.GetIndexConfig() }

// IsTrained reports whether the index has been trained.
func (e *EncryptedIndex) IsTrained() bool { return e.internal.IsTrained() }

// Upsert inserts or updates vectors (IDs that already exist will be updated).
func (e *EncryptedIndex) Upsert(ctx context.Context, items []VectorItem) error {
	return e.internal.Upsert(ctx, items)
}

// Query performs a similarity search.
//
// Provide a single request object:
//   - *internal.QueryRequest
//
// Behavior:
//   - If req.BatchQueryVectors is non-empty, a batch query is executed.
//   - Otherwise, a single-vector or contents-based query is executed.
//
// Returns:
//   - *QueryResponse with nearest neighbors, distances, and metadata
//   - error on failure
func (e *EncryptedIndex) Query(ctx context.Context, req *internal.QueryRequest) (*QueryResponse, error) {
	return e.internal.Query(ctx, req)
}

// Get fetches vectors by ID with selected fields.
func (e *EncryptedIndex) Get(ctx context.Context, ids []string, include []string) (*GetResponse, error) {
	return e.internal.Get(ctx, ids, include)
}

// Delete removes vectors by ID.
func (e *EncryptedIndex) Delete(ctx context.Context, ids []string) error {
	return e.internal.Delete(ctx, ids)
}

// Train optimizes the index (see internal.TrainRequest for options).
func (e *EncryptedIndex) Train(ctx context.Context, req *internal.TrainRequest) error {
	return e.internal.Train(ctx, req)
}

// DeleteIndex permanently removes the index.
func (e *EncryptedIndex) DeleteIndex(ctx context.Context) error {
	return e.internal.DeleteIndex(ctx)
}
