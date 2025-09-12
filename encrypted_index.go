// encrypted_index.go provides the EncryptedIndex type for encrypted vector operations.
// This file implements the main interface for working with encrypted vector indexes,
// including CRUD operations, similarity search, and index management.
package cyborgdb

import (
	"context"
	"fmt"

	"github.com/cyborginc/cyborgdb-go/internal"
)

var (
	// ErrQueryVectorsInvalidType is returned when QueryParams contains invalid query vector types.
	// This occurs when query vectors are not properly formatted as []float32 or [][]float32.
	ErrQueryVectorsInvalidType = fmt.Errorf("queryVectors must be []float32 for single vector queries or [][]float32 for batch queries")

	// ErrMissingQueryInput is returned when no query input is provided in QueryParams.
	// At least one of QueryVector, BatchQueryVectors, or QueryContents must be specified.
	ErrMissingQueryInput = fmt.Errorf("either queryVectors or queryContents must be provided")
)

// EncryptedIndex provides a handle for performing operations on an encrypted vector index.
//
// This type encapsulates all the information needed to interact with a specific index,
// including authentication credentials and cached metadata. It provides methods for:
//
//   - Vector operations: Upsert, Query, Get, Delete
//   - Index management: Train, DeleteIndex, ListIDs
//   - Metadata access: GetIndexName, GetIndexType, IsTrained, GetIndexConfig
//
// All vector data is encrypted end-to-end using the provided encryption key.
// The index maintains a persistent connection to the CyborgDB service and
// caches metadata to avoid unnecessary API calls.
//
// Instances should be created via Client.CreateIndex() or Client.LoadIndex().
type EncryptedIndex struct {
	// indexName is the unique identifier for this index
	indexName string

	// indexKey is the hex-encoded encryption key for end-to-end encryption
	indexKey string

	// indexType indicates the index algorithm ("ivf", "ivfflat", "ivfpq")
	indexType string

	// config holds the detailed index configuration, may be nil for loaded indexes
	config *internal.IndexConfig

	// trained indicates whether the index has been optimized via training
	trained bool

	// client provides access to the underlying API client
	client *internal.Client
}

// GetIndexName returns the unique name of this index.
//
// This is a cached value that doesn't require an API call.
//
// Returns:
//   - string: The index name as specified during creation
func (e *EncryptedIndex) GetIndexName() string { return e.indexName }

// GetIndexType returns the algorithm type of this index.
//
// This is a cached value that doesn't require an API call.
//
// Returns:
//   - string: Index type ("ivf", "ivfflat", or "ivfpq")
func (e *EncryptedIndex) GetIndexType() string { return e.indexType }

// GetIndexConfig returns the detailed configuration of this index.
//
// This is a cached value that doesn't require an API call. For indexes
// loaded via LoadIndex(), the configuration may be incomplete.
//
// Returns:
//   - internal.IndexConfig: The index configuration, or empty if not available
func (e *EncryptedIndex) GetIndexConfig() internal.IndexConfig {
	if e.config != nil {
		return *e.config
	}
	return internal.IndexConfig{}
}

// IsTrained reports whether this index has been optimized through training.
//
// This is a cached value that doesn't require an API call. The value is
// updated automatically when Train() completes successfully.
//
// Returns:
//   - bool: true if the index has been trained, false otherwise
func (e *EncryptedIndex) IsTrained() bool { return e.trained }

// CheckTrainingStatus queries the server to check if this index is currently being trained
// and updates the cached training status if training has completed.
//
// Returns:
//   - bool: true if the index is currently being trained, false otherwise
//   - error: Any error encountered during the status check
func (e *EncryptedIndex) CheckTrainingStatus(ctx context.Context) (bool, error) {
	// Get training status from server
	result, _, err := e.client.APIClient.DefaultAPI.GetTrainingStatusV1IndexesTrainingStatusGet(ctx).Execute()
	if err != nil {
		return false, fmt.Errorf("failed to get training status: %w", err)
	}

	// Parse the result to check if this index is being trained
	if statusMap, ok := result.(map[string]interface{}); ok {
		if trainingIndexes, ok := statusMap["training_indexes"].([]interface{}); ok {
			isTraining := false
			for _, idx := range trainingIndexes {
				if idxName, ok := idx.(string); ok && idxName == e.indexName {
					isTraining = true
					break
				}
			}
			
			// If not training anymore but was previously untrained, update the cached status
			if !isTraining && !e.trained {
				// Check if the index is actually trained by querying its info
				describeReq := internal.IndexOperationRequest{
					IndexName: e.indexName,
					IndexKey:  e.indexKey,
				}
				
				resp, _, err := e.client.APIClient.DefaultAPI.GetIndexInfoV1IndexesDescribePost(ctx).
					IndexOperationRequest(describeReq).
					Execute()
				if err == nil && resp != nil {
					e.trained = resp.GetIsTrained()
				}
			}
			
			return isTraining, nil
		}
	}
	
	return false, fmt.Errorf("unexpected training status response format")
}

// Upsert inserts new vectors or updates existing ones in the index.
//
// Vector data is encrypted end-to-end before transmission. If a vector ID
// already exists, it will be updated with the new vector data and metadata.
// This operation is idempotent.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - items: Slice of VectorItem containing ID, vector, and optional metadata
//
// Returns:
//   - error: Any error encountered during the operation
//
// Example:
//
//	items := []VectorItem{
//		{Id: "doc1", Vector: []float32{0.1, 0.2, 0.3}, Metadata: map[string]interface{}{"type": "document"}},
//		{Id: "doc2", Vector: []float32{0.4, 0.5, 0.6}},
//	}
//	err := index.Upsert(ctx, items)
func (e *EncryptedIndex) Upsert(ctx context.Context, items []VectorItem) error {
	req := internal.UpsertRequest{
		IndexName: e.indexName,
		IndexKey:  e.indexKey,
		Items:     items,
	}
	resp, _, err := e.client.APIClient.DefaultAPI.UpsertVectorsV1VectorsUpsertPost(ctx).
		UpsertRequest(req).
		Execute()
	if err != nil {
		return err
	}
	
	// If training was triggered, we can note that the index is no longer trained
	// (it will be retrained automatically)
	if resp != nil && resp.HasTrainingTriggered() && resp.GetTrainingTriggered() {
		e.trained = false
	}
	
	return nil
}

// Query performs similarity search to find the nearest neighbors to query vector(s).
//
// This method supports three types of queries:
//   - Single vector query: Set QueryParams.QueryVector
//   - Batch vector query: Set QueryParams.BatchQueryVectors
//   - Content-based query: Set QueryParams.QueryContents (if supported by server)
//
// The search uses the distance metric specified during index creation.
// Results are ordered by similarity (closest first) and can be filtered
// by metadata using the Filters parameter.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - params: QueryParams specifying query vectors, filters, and result preferences
//
// Returns:
//   - *QueryResponse: Search results with IDs, distances, and requested fields
//   - error: Any error encountered during the search
//
// Example:
//
//	params := QueryParams{
//		QueryVector: []float32{0.1, 0.2, 0.3},
//		TopK: 10,
//		Include: []string{"metadata"},
//		Filters: map[string]interface{}{"category": "document"},
//	}
//	results, err := index.Query(ctx, params)
func (e *EncryptedIndex) Query(ctx context.Context, params QueryParams) (*QueryResponse, error) {
	// Handle batch queries separately
	if len(params.BatchQueryVectors) > 0 {
		batchReq := internal.BatchQueryRequest{
			IndexName:    e.indexName,
			IndexKey:     e.indexKey,
			QueryVectors: params.BatchQueryVectors,
			Filters:      params.Filters,
			Include:      params.Include,
		}

		// Handle nullable fields for batch request
		if params.TopK != 0 {
			batchReq.TopK = *internal.NewNullableInt32(&params.TopK)
		}

		if params.NProbes != nil {
			batchReq.NProbes = *internal.NewNullableInt32(params.NProbes)
		}

		if params.Greedy != nil {
			batchReq.Greedy = *internal.NewNullableBool(params.Greedy)
		}

		request := internal.Request{
			BatchQueryRequest: &batchReq,
		}
		result, _, err := e.client.APIClient.DefaultAPI.QueryVectorsV1VectorsQueryPost(ctx).
			Request(request).
			Execute()
		return result, err
	}

	// Handle single query
	req := internal.QueryRequest{
		IndexName: e.indexName,
		IndexKey:  e.indexKey,
		Filters:   params.Filters,
		Include:   params.Include,
	}

	if params.QueryVector != nil {
		req.QueryVectors = params.QueryVector
	}

	// Handle nullable fields
	if params.QueryContents != nil {
		req.QueryContents = *internal.NewNullableString(params.QueryContents)
	}

	if params.TopK != 0 {
		req.TopK = *internal.NewNullableInt32(&params.TopK)
	}

	if params.NProbes != nil {
		req.NProbes = *internal.NewNullableInt32(params.NProbes)
	}

	if params.Greedy != nil {
		req.Greedy = *internal.NewNullableBool(params.Greedy)
	}
	request := internal.Request{
		QueryRequest: &req,
	}
	result, _, err := e.client.APIClient.DefaultAPI.QueryVectorsV1VectorsQueryPost(ctx).
		Request(request).
		Execute()
	return result, err
}

// Get retrieves specific vectors from the index by their IDs.
//
// This method allows efficient retrieval of vectors and their metadata
// without performing similarity search. Useful for reconstructing original
// data or examining specific vectors.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - ids: Slice of vector IDs to retrieve
//   - include: Fields to include in response ("vector", "metadata", or both)
//
// Returns:
//   - *GetResponse: Retrieved vectors with requested fields
//   - error: Any error encountered, including IDs not found
//
// Example:
//
//	ids := []string{"doc1", "doc2", "doc3"}
//	include := []string{"vector", "metadata"}
//	results, err := index.Get(ctx, ids, include)
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

// Delete removes vectors from the index by their IDs.
//
// This operation is irreversible. Deleted vectors are permanently removed
// from the index and cannot be recovered. The operation succeeds even if
// some IDs don't exist in the index.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - ids: Slice of vector IDs to delete
//
// Returns:
//   - error: Any error encountered during deletion
//
// Example:
//
//	ids := []string{"doc1", "doc2"}
//	err := index.Delete(ctx, ids)
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

// Train optimizes the index for better query performance and accuracy.
//
// Training analyzes the existing vectors to build internal data structures
// that accelerate similarity search. This process can significantly improve
// query speed and accuracy, especially for large datasets.
//
// Training is typically performed after upserting a substantial number of
// vectors. The index remains usable during training, but performance may
// be suboptimal until training completes.
//
// All parameters are optional with sensible defaults. The trained flag is
// automatically updated upon successful completion.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts (training can take time)
//   - params: TrainParams specifying training options like batch size and iterations
//
// Returns:
//   - error: Any error encountered during training
//
// Example:
//
//	params := TrainParams{
//		BatchSize: &[]int32{1024}[0],  // Process 1024 vectors per batch
//		MaxIters: &[]int32{200}[0],   // Allow up to 200 iterations
//	}
//	err := index.Train(ctx, params)
func (e *EncryptedIndex) Train(ctx context.Context, params TrainParams) error {
	// Create request with required fields
	req := internal.TrainRequest{
		IndexKey:  e.indexKey,
		IndexName: e.indexName,
	}

	// Set optional fields, using server defaults when not provided
	// This works around a Python server issue where missing fields become None

	// BatchSize: default 2048
	batchSize := int32(2048)
	if params.BatchSize != nil {
		batchSize = *params.BatchSize
	}
	req.BatchSize = *internal.NewNullableInt32(&batchSize)

	// MaxIters: default 100
	maxIters := int32(100)
	if params.MaxIters != nil {
		maxIters = *params.MaxIters
	}
	req.MaxIters = *internal.NewNullableInt32(&maxIters)

	// Tolerance: default 1e-6
	tolerance := float32(1e-6)
	if params.Tolerance != nil {
		tolerance = float32(*params.Tolerance)
	}
	req.Tolerance = *internal.NewNullableFloat32(&tolerance)

	// MaxMemory: default 0 (no limit)
	maxMemory := int32(0)
	if params.MaxMemory != nil {
		maxMemory = *params.MaxMemory
	}
	req.MaxMemory = *internal.NewNullableInt32(&maxMemory)

	// NLists: default 0 (auto)
	nLists := int32(0)
	if params.NLists != nil {
		nLists = *params.NLists
	}
	req.NLists = *internal.NewNullableInt32(&nLists)
	_, _, err := e.client.APIClient.DefaultAPI.TrainIndexV1IndexesTrainPost(ctx).
		TrainRequest(req).
		Execute()
	if err == nil {
		e.trained = true
	}
	return err
}

// DeleteIndex permanently destroys this index and all its data.
//
// This operation is irreversible and will delete all vectors, metadata,
// and index structures. The index cannot be recovered after deletion.
// The EncryptedIndex handle becomes invalid after this operation.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//
// Returns:
//   - error: Any error encountered during deletion
//
// Warning: This operation cannot be undone. Ensure you have backups if needed.
//
// Example:
//
//	err := index.DeleteIndex(ctx)
//	// index is now invalid and should not be used
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

// ListIDs retrieves all vector IDs currently stored in the index.
//
// This method provides a way to enumerate all vectors without retrieving
// their actual vector data or metadata. Useful for administrative tasks,
// data exploration, or building processing pipelines.
//
// For large indexes, this operation may take considerable time and return
// a large response. Consider implementing pagination if needed.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//
// Returns:
//   - *ListIDsResponse: Contains all vector IDs and total count
//   - error: Any error encountered during the operation
//
// Example:
//
//	response, err := index.ListIDs(ctx)
//	if err == nil {
//		fmt.Printf("Index contains %d vectors\n", len(response.Ids))
//		for _, id := range response.Ids {
//			fmt.Printf("Vector ID: %s\n", id)
//		}
//	}
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
