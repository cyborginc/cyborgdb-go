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
	indexName string
	indexKey  string
	indexType string
	config    *internal.IndexConfig
	trained   bool
	client    *internal.Client
}

// GetIndexName returns the index name.
func (e *EncryptedIndex) GetIndexName() string { return e.indexName }

// GetIndexType returns the index type ("ivf", "ivfpq", or "ivfflat").
func (e *EncryptedIndex) GetIndexType() string { return e.indexType }

// GetIndexConfig returns the full index configuration.
func (e *EncryptedIndex) GetIndexConfig() internal.IndexConfig { 
	if e.config != nil {
		return *e.config
	}
	return internal.IndexConfig{}
}

// IsTrained reports whether the index has been trained.
func (e *EncryptedIndex) IsTrained() bool { return e.trained }

// Upsert inserts or updates vectors (IDs that already exist will be updated).
func (e *EncryptedIndex) Upsert(ctx context.Context, items []VectorItem) error {
	req := internal.UpsertRequest{
		IndexName: e.indexName,
		IndexKey:  e.indexKey,
		Vectors:   items,
	}
	_, _, err := e.client.APIClient.DefaultAPI.UpsertVectorsV1VectorsUpsertPost(ctx).
		UpsertRequest(req).
		Execute()
	return err
}

// Query performs a similarity search.
//
// Provide a single request object:
//   - QueryParams
//
// Behavior:
//   - If params.BatchQueryVectors is non-empty, a batch query is executed.
//   - Otherwise, a single-vector or contents-based query is executed.
//
// Returns:
//   - *QueryResponse with nearest neighbors, distances, and metadata
//   - error on failure
func (e *EncryptedIndex) Query(ctx context.Context, params QueryParams) (*QueryResponse, error) {
	req := internal.QueryRequest{
		IndexName:         e.indexName,
		IndexKey:          e.indexKey,
		QueryVector:       params.QueryVector,
		BatchQueryVectors: params.BatchQueryVectors,
		QueryContents:     params.QueryContents,
		TopK:              params.TopK,
		NProbes:           params.NProbes,
		Greedy:            params.Greedy,
		Filters:           params.Filters,
		Include:           params.Include,
	}
	result, _, err := e.client.APIClient.DefaultAPI.QueryVectorsV1VectorsQueryPost(ctx).
		QueryRequest(req).
		Execute()
	return result, err
}

// Get fetches vectors by ID with selected fields.
func (e *EncryptedIndex) Get(ctx context.Context, ids []string, include []string) (*GetResponse, error) {
	req := internal.GetRequest{
		IndexName: e.indexName,
		IndexKey:  e.indexKey,
		Ids:       ids,
		Include:   include,
	}
	result, _, err := e.client.APIClient.DefaultAPI.GetVectorsV1VectorsGetPost(ctx).
		GetRequest(req).
		Execute()
	if err != nil {
		return nil, err
	}
	// Convert GetResponseModel to GetResponse
	return (*GetResponse)(result), nil
}

// Delete removes vectors by ID.
func (e *EncryptedIndex) Delete(ctx context.Context, ids []string) error {
	req := internal.DeleteRequest{
		IndexName: e.indexName,
		IndexKey:  e.indexKey,
		Ids:       ids,
	}
	_, _, err := e.client.APIClient.DefaultAPI.DeleteVectorsV1VectorsDeletePost(ctx).
		DeleteRequest(req).
		Execute()
	return err
}

// Train optimizes the index (see internal.TrainRequest for options).
func (e *EncryptedIndex) Train(ctx context.Context, params TrainParams) error {
	req := internal.TrainRequest{
		IndexName: e.indexName,
		IndexKey:  e.indexKey,
		BatchSize: params.BatchSize,
		MaxIters:  params.MaxIters,
		Tolerance: params.Tolerance,
		MaxMemory: params.MaxMemory,
		NLists:    params.NLists,
	}
	_, _, err := e.client.APIClient.DefaultAPI.TrainIndexV1IndexesTrainPost(ctx).
		TrainRequest(req).
		Execute()
	if err == nil {
		e.trained = true
	}
	return err
}

// DeleteIndex permanently removes the index.
func (e *EncryptedIndex) DeleteIndex(ctx context.Context) error {
	req := internal.IndexOperationRequest{
		IndexName: e.indexName,
		IndexKey:  e.indexKey,
	}
	_, _, err := e.client.APIClient.DefaultAPI.DeleteIndexV1IndexesDeletePost(ctx).
		IndexOperationRequest(req).
		Execute()
	return err
}

// ListIDs returns all IDs in the index.
//
// Returns:
//   - *ListIDsResponse containing the IDs and count
//   - error on failure
func (e *EncryptedIndex) ListIDs(ctx context.Context) (*ListIDsResponse, error) {
	req := internal.ListIDsRequest{
		IndexName: e.indexName,
		IndexKey:  e.indexKey,
	}
	result, _, err := e.client.APIClient.DefaultAPI.ListIdsV1VectorsListIdsPost(ctx).
		ListIDsRequest(req).
		Execute()
	return result, err
}
