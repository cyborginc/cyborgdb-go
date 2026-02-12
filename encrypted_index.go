// encrypted_index.go provides the EncryptedIndex type for encrypted vector operations.
// This file implements the main interface for working with encrypted vector indexes,
// including CRUD operations, similarity search, and index management.
package cyborgdb

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/cyborginc/cyborgdb-go/internal"
)

var (
	// ErrQueryVectorsInvalidType is returned when QueryParams contains invalid query vector types.
	// This occurs when query vectors are not properly formatted as []float32 or [][]float32.
	ErrQueryVectorsInvalidType = fmt.Errorf("queryVectors must be []float32 for single vector queries or [][]float32 for batch queries")

	// ErrMissingQueryInput is returned when no query input is provided in QueryParams.
	// At least one of QueryVector, BatchQueryVectors, or QueryContents must be specified.
	ErrMissingQueryInput = fmt.Errorf("either queryVectors or queryContents must be provided")

	// ErrUnexpectedTrainingStatus is returned when the training status response format is unexpected.
	ErrUnexpectedTrainingStatus = fmt.Errorf("unexpected training status response format")
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

	// indexType indicates the index algorithm ("ivfflat", "ivfpq", "ivfsq")
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
//   - string: Index type ("ivfflat", "ivfpq", or "ivfsq")
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

// IsTrained checks whether this index has been optimized through training.
//
// This method calls the describe endpoint to get the current training status,
// matching the behavior of the Python and JavaScript SDKs.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//
// Returns:
//   - bool: true if the index has been trained, false otherwise
//   - error: Any error encountered during the status check
func (e *EncryptedIndex) IsTrained(ctx context.Context) (bool, error) {
	describeReq := internal.IndexOperationRequest{
		IndexName: e.indexName,
		IndexKey:  e.indexKey,
	}
	resp, _, err := e.client.APIClient.DefaultAPI.GetIndexInfoV1IndexesDescribePost(ctx).
		IndexOperationRequest(describeReq).
		Execute()
	if err != nil {
		return false, fmt.Errorf("failed to get index training status: %w", err)
	}
	e.trained = resp.GetIsTrained()
	return e.trained, nil
}

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

	// Check if this index is being trained
	isTraining := false
	for _, idx := range result.TrainingIndexes {
		if idx == e.indexName {
			isTraining = true
			break
		}
	}

	// If not training anymore but was previously untrained, update the cached status
	if !isTraining && !e.trained {
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
	_, _, err := e.client.APIClient.DefaultAPI.UpsertVectorsV1VectorsUpsertPost(ctx).
		UpsertRequest(req).
		Execute()
	return err
}

// UpsertVectors inserts vectors using separate ID and vector arrays.
//
// This method automatically uses binary format for efficient transfer when
// vectors are provided as a 2D slice. This is more efficient than regular
// Upsert for large batches.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - ids: Slice of unique identifiers for each vector
//   - vectors: 2D slice of float32 vectors, shape [n_vectors][dimension]
//   - metadata: Optional metadata for each vector (can be nil)
//
// Returns:
//   - error: Any error encountered during the operation
//
// Example:
//
//	ids := []string{"doc1", "doc2", "doc3"}
//	vectors := [][]float32{
//		{0.1, 0.2, 0.3},
//		{0.4, 0.5, 0.6},
//		{0.7, 0.8, 0.9},
//	}
//	err := index.UpsertVectors(ctx, ids, vectors, nil)
func (e *EncryptedIndex) UpsertVectors(ctx context.Context, ids []string, vectors [][]float32, metadata []map[string]interface{}) error {
	// Use binary format for efficiency
	params := BinaryUpsertParams{
		IDs:      ids,
		Vectors:  vectors,
		Metadata: metadata,
	}
	return e.UpsertBinary(ctx, params)
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
	// Handle batch queries using BatchQueryRequest (non-binary format)
	// For binary format with large batches, use QueryBinary() directly
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
	return result, nil
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

	// Set optional fields only if provided by the caller
	// Let the server handle default values

	if params.BatchSize != nil {
		req.BatchSize = *internal.NewNullableInt32(params.BatchSize)
	}

	if params.MaxIters != nil {
		req.MaxIters = *internal.NewNullableInt32(params.MaxIters)
	}

	if params.Tolerance != nil {
		tolerance := float32(*params.Tolerance)
		req.Tolerance = *internal.NewNullableFloat32(&tolerance)
	}

	if params.MaxMemory != nil {
		req.MaxMemory = *internal.NewNullableInt32(params.MaxMemory)
	}

	if params.NLists != nil {
		req.NLists = *internal.NewNullableInt32(params.NLists)
	}

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

// vectorsToBase64 converts a 2D slice of float32 vectors to a base64-encoded string.
// The vectors are flattened and encoded as little-endian float32 bytes.
func vectorsToBase64(vectors [][]float32) string {
	if len(vectors) == 0 {
		return ""
	}

	// Calculate total number of floats
	numVectors := len(vectors)
	dimension := len(vectors[0])
	totalFloats := numVectors * dimension

	// Create byte buffer (4 bytes per float32)
	buf := make([]byte, totalFloats*4)

	// Convert each float32 to little-endian bytes
	offset := 0
	for _, vec := range vectors {
		for _, val := range vec {
			binary.LittleEndian.PutUint32(buf[offset:], math.Float32bits(val))
			offset += 4
		}
	}

	return base64.StdEncoding.EncodeToString(buf)
}

// UpsertBinary inserts new vectors or updates existing ones using binary format.
//
// This method is more efficient than regular Upsert for large batches as vectors
// are sent as base64-encoded binary data instead of JSON arrays. This reduces
// payload size and improves performance for large datasets.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - params: BinaryUpsertParams containing IDs, vectors, and optional metadata
//
// Returns:
//   - error: Any error encountered during the operation
//
// Example:
//
//	params := BinaryUpsertParams{
//		IDs: []string{"doc1", "doc2", "doc3"},
//		Vectors: [][]float32{
//			{0.1, 0.2, 0.3},
//			{0.4, 0.5, 0.6},
//			{0.7, 0.8, 0.9},
//		},
//		Metadata: []map[string]interface{}{
//			{"type": "document"},
//			{"type": "article"},
//			nil, // No metadata for doc3
//		},
//	}
//	err := index.UpsertBinary(ctx, params)
func (e *EncryptedIndex) UpsertBinary(ctx context.Context, params BinaryUpsertParams) error {
	if len(params.IDs) == 0 {
		return fmt.Errorf("IDs cannot be empty")
	}
	if len(params.Vectors) == 0 {
		return fmt.Errorf("vectors cannot be empty")
	}
	if len(params.IDs) != len(params.Vectors) {
		return fmt.Errorf("IDs length (%d) must match vectors length (%d)", len(params.IDs), len(params.Vectors))
	}
	if len(params.Metadata) > 0 && len(params.Metadata) != len(params.IDs) {
		return fmt.Errorf("metadata length (%d) must match IDs length (%d)", len(params.Metadata), len(params.IDs))
	}
	if len(params.Contents) > 0 && len(params.Contents) != len(params.IDs) {
		return fmt.Errorf("contents length (%d) must match IDs length (%d)", len(params.Contents), len(params.IDs))
	}

	// Get dimension from first vector
	dimension := int32(len(params.Vectors[0]))

	// Convert vectors to base64
	vectorsB64 := vectorsToBase64(params.Vectors)

	// Build the batch
	batch := internal.BinaryVectorBatch{
		Ids:        params.IDs,
		VectorsB64: vectorsB64,
		Dimension:  dimension,
	}

	// Add metadata if provided
	if len(params.Metadata) > 0 {
		metadata := make([]*map[string]interface{}, len(params.Metadata))
		for i, m := range params.Metadata {
			if m != nil {
				metadata[i] = &m
			}
		}
		batch.Metadata = metadata
	}

	// Add contents if provided
	if len(params.Contents) > 0 {
		contents := make([]internal.BinaryVectorBatchContentsInner, len(params.Contents))
		for i, c := range params.Contents {
			if c != "" {
				contents[i] = internal.BinaryVectorBatchContentsInner{String: &c}
			}
		}
		batch.Contents = contents
	}

	req := internal.BinaryUpsertRequest{
		IndexName: e.indexName,
		IndexKey:  e.indexKey,
		Batch:     batch,
	}

	_, _, err := e.client.APIClient.DefaultAPI.UpsertVectorsBinaryV1VectorsUpsertBinaryPost(ctx).
		BinaryUpsertRequest(req).
		Execute()
	return err
}

// QueryBinary performs similarity search using binary format for query vectors.
//
// This method is more efficient than regular Query for batch queries as vectors
// are sent as base64-encoded binary data instead of JSON arrays. This reduces
// payload size and improves performance for large batch queries.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - params: BinaryQueryParams containing query vectors and search options
//
// Returns:
//   - *QueryResponse: Search results with IDs, distances, and requested fields
//   - error: Any error encountered during the search
//
// Example:
//
//	params := BinaryQueryParams{
//		QueryVectors: [][]float32{
//			{0.1, 0.2, 0.3},
//			{0.4, 0.5, 0.6},
//		},
//		TopK: 10,
//		Include: []string{"metadata"},
//		Filters: map[string]interface{}{"category": "document"},
//	}
//	results, err := index.QueryBinary(ctx, params)
func (e *EncryptedIndex) QueryBinary(ctx context.Context, params BinaryQueryParams) (*QueryResponse, error) {
	if len(params.QueryVectors) == 0 {
		return nil, fmt.Errorf("queryVectors cannot be empty")
	}

	// Get dimension from first vector
	dimension := int32(len(params.QueryVectors[0]))

	// Convert vectors to base64
	vectorsB64 := vectorsToBase64(params.QueryVectors)

	// Build the batch
	batch := internal.BinaryQueryBatch{
		VectorsB64: vectorsB64,
		Dimension:  dimension,
	}

	req := internal.BinaryQueryRequest{
		IndexName: e.indexName,
		IndexKey:  e.indexKey,
		Batch:     batch,
		Filters:   params.Filters,
		Include:   params.Include,
	}

	// Handle optional fields
	if params.TopK != 0 {
		req.TopK = *internal.NewNullableInt32(&params.TopK)
	}

	if params.NProbes != nil {
		req.NProbes = *internal.NewNullableInt32(params.NProbes)
	}

	if params.Greedy != nil {
		req.Greedy = *internal.NewNullableBool(params.Greedy)
	}

	result, _, err := e.client.APIClient.DefaultAPI.QueryVectorsBinaryV1VectorsQueryBinaryPost(ctx).
		BinaryQueryRequest(req).
		Execute()
	return result, err
}
