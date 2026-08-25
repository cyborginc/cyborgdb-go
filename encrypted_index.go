// encrypted_index.go provides the EncryptedIndex type for encrypted vector operations.
// This file implements the main interface for working with encrypted vector indexes,
// including CRUD operations, similarity search, and index management.
package cyborgdb

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/cyborginc/cyborgdb-go/internal"
)

const (
	// float32ByteSize is the number of bytes in a float32.
	float32ByteSize = 4
)

var (
	// ErrQueryVectorsInvalidType is returned when QueryParams contains invalid query vector types.
	// This occurs when query vectors are not properly formatted as []float32 or [][]float32.
	ErrQueryVectorsInvalidType = errors.New("queryVectors must be []float32 for single vector queries or [][]float32 for batch queries")

	// ErrMissingQueryInput is returned when no query input is provided in QueryParams.
	// At least one of QueryVector, BatchQueryVectors, or QueryContents must be specified.
	ErrMissingQueryInput = errors.New("either queryVectors or queryContents must be provided")

	// ErrUnexpectedTrainingStatus is returned when the training status response format is unexpected.
	ErrUnexpectedTrainingStatus = errors.New("unexpected training status response format")

	// ErrEmptyIDs is returned when IDs slice is empty.
	ErrEmptyIDs = errors.New("IDs cannot be empty")

	// ErrEmptyVectors is returned when vectors slice is empty.
	ErrEmptyVectors = errors.New("vectors cannot be empty")

	// ErrEmptyQueryVectors is returned when query vectors slice is empty.
	ErrEmptyQueryVectors = errors.New("queryVectors cannot be empty")

	// ErrInconsistentDimension is returned when a batch of vectors does not all
	// share the same dimension. Binary encoding assumes a uniform dimension.
	ErrInconsistentDimension = errors.New("all vectors must have the same dimension")

	// ErrIDsVectorsLengthMismatch is returned when IDs and vectors have different lengths.
	ErrIDsVectorsLengthMismatch = errors.New("IDs length must match vectors length")

	// ErrMetadataLengthMismatch is returned when metadata length doesn't match IDs length.
	ErrMetadataLengthMismatch = errors.New("metadata length must match IDs length")

	// ErrContentsLengthMismatch is returned when contents length doesn't match IDs length.
	ErrContentsLengthMismatch = errors.New("contents length must match IDs length")

	// ErrUnsupportedUpsertType is returned when Upsert receives an unsupported input type.
	ErrUnsupportedUpsertType = errors.New("unsupported upsert input type")

	// ErrUnsupportedQueryType is returned when Query receives an unsupported params type.
	ErrUnsupportedQueryType = errors.New("unsupported query input type")
)

// EncryptedIndex provides a handle for performing operations on an encrypted vector index.
//
// This type encapsulates all the information needed to interact with a specific index,
// including authentication credentials and cached metadata. It provides methods for:
//
//   - Vector operations: Upsert, Query, Get, Delete
//   - Index management: Train, DeleteIndex, ListIDs
//   - Metadata access: GetIndexName, IsTrained
//
// All vector data is encrypted end-to-end using the provided encryption key
// (or, for KMS-backed indexes, resolved server-side from the index's KMSBlob).
// The index maintains a persistent connection to the CyborgDB service and
// caches metadata to avoid unnecessary API calls.
//
// Instances should be created via Client.CreateIndex() or Client.LoadIndex().
type EncryptedIndex struct {
	// indexName is the unique identifier for this index
	indexName string

	// indexKey is the hex-encoded encryption key for end-to-end encryption.
	// nil for KMS-backed indexes where the service resolves the DEK from
	// the stored KMSBlob.
	indexKey *string

	// client provides access to the underlying API client
	client *internal.Client
}

// indexKeyField builds the NullableString IndexKey field expected by the
// generated request models. Returns the unset zero value when this index has
// no SDK-held key (KMS-backed indexes), which serializes as "field omitted"
// and lets the service resolve the DEK from the stored KMSBlob.
func (e *EncryptedIndex) indexKeyField() internal.NullableString {
	if e.indexKey == nil {
		return internal.NullableString{}
	}
	return *internal.NewNullableString(e.indexKey)
}

// describeIndex calls the describe endpoint for indexName. key carries the
// NullableString IndexKey (unset for KMS-backed indexes). Callers add their own
// error context.
func describeIndex(ctx context.Context, client *internal.Client, indexName string, key internal.NullableString) (*internal.IndexInfoResponseModel, error) {
	req := internal.IndexOperationRequest{IndexName: indexName, IndexKey: key}
	resp, _, err := client.APIClient.DefaultAPI.GetIndexInfoV1IndexesDescribePost(ctx).
		IndexOperationRequest(req).
		Execute()
	return resp, err
}

// nullableQueryOpts converts the optional TopK/NProbes/Greedy query knobs into
// the Nullable trio shared by QueryRequest, BatchQueryRequest, and
// BinaryQueryRequest. A zero TopK is left unset so the server applies its
// default; nil NProbes/Greedy are likewise left unset.
func nullableQueryOpts(topK int32, nProbes *int32, greedy *bool) (internal.NullableInt32, internal.NullableInt32, internal.NullableBool) {
	var nTopK, nNProbes internal.NullableInt32
	var nGreedy internal.NullableBool
	if topK != 0 {
		nTopK = *internal.NewNullableInt32(&topK)
	}
	if nProbes != nil {
		nNProbes = *internal.NewNullableInt32(nProbes)
	}
	if greedy != nil {
		nGreedy = *internal.NewNullableBool(greedy)
	}
	return nTopK, nNProbes, nGreedy
}

// hybridNullables holds the BM25/hybrid text-leg knobs converted into the
// Nullable/slice wire types shared by QueryRequest, BatchQueryRequest, and
// BinaryQueryRequest. Each unset knob stays at its Nullable zero value, which
// serializes as "field omitted" — so an index without full-text fields keeps
// seeing text-free requests.
type hybridNullables struct {
	Text             internal.NullableString
	TextFields       []string
	TextFieldWeights []float32
	RequireAllTerms  internal.NullableBool
	Alpha            internal.NullableFloat32
	RrfK             internal.NullableFloat32
	WindowMult       internal.NullableInt32
}

// buildHybrid converts the optional hybrid text-leg knobs (shared by
// QueryParams and BinaryQueryParams) into their wire form. Alpha/RrfK/Bm25 are
// float64 in the public API for ergonomics but float32 on the wire.
func buildHybrid(text *string, textFields []string, textFieldWeights []float32, requireAllTerms *bool, alpha, rrfK *float64, windowMult *int32) hybridNullables {
	h := hybridNullables{TextFields: textFields, TextFieldWeights: textFieldWeights}
	if text != nil {
		h.Text = *internal.NewNullableString(text)
	}
	if requireAllTerms != nil {
		h.RequireAllTerms = *internal.NewNullableBool(requireAllTerms)
	}
	if alpha != nil {
		a := float32(*alpha)
		h.Alpha = *internal.NewNullableFloat32(&a)
	}
	if rrfK != nil {
		r := float32(*rrfK)
		h.RrfK = *internal.NewNullableFloat32(&r)
	}
	if windowMult != nil {
		h.WindowMult = *internal.NewNullableInt32(windowMult)
	}
	return h
}

// encodeVectorBatch base64-encodes a batch of equal-length vectors and returns
// the encoding together with the shared dimension. Ragged batches yield
// ErrInconsistentDimension (from vectorsToBase64).
func encodeVectorBatch(vectors [][]float32) (string, int32, error) {
	b64, err := vectorsToBase64(vectors)
	if err != nil {
		return "", 0, err
	}
	if len(vectors) == 0 {
		return b64, 0, nil
	}
	return b64, int32(len(vectors[0])), nil
}

// GetIndexName returns the unique name of this index.
//
// This is a cached value that doesn't require an API call.
//
// Returns:
//   - string: The index name as specified during creation
func (e *EncryptedIndex) GetIndexName() string { return e.indexName }

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
	resp, err := describeIndex(ctx, e.client, e.indexName, e.indexKeyField())
	if err != nil {
		return false, fmt.Errorf("failed to get index training status: %w", err)
	}
	return resp.GetIsTrained(), nil
}

// IsTraining queries the server to check whether this index is currently being
// trained (e.g. by an auto-training trigger). It reflects live server state on
// every call and holds no cached status, matching the Python SDK's is_training.
//
// Returns:
//   - bool: true if the index is currently being trained, false otherwise
//   - error: Any error encountered during the status check
func (e *EncryptedIndex) IsTraining(ctx context.Context) (bool, error) {
	result, _, err := e.client.APIClient.DefaultAPI.GetTrainingStatusV1IndexesTrainingStatusGet(ctx).Execute()
	if err != nil {
		return false, fmt.Errorf("failed to get training status: %w", err)
	}

	for _, idx := range result.TrainingIndexes {
		if idx == e.indexName {
			return true, nil
		}
	}
	return false, nil
}

// Dimension returns the vector dimensionality of this index.
//
// The value is 0 if the index was created without an explicit dimension and
// the first upsert hasn't happened yet; otherwise it is the real dimension.
// Each call queries the describe endpoint for live state.
//
// Returns:
//   - int32: The vector dimension
//   - error: Any error encountered during the lookup
func (e *EncryptedIndex) Dimension(ctx context.Context) (int32, error) {
	resp, err := describeIndex(ctx, e.client, e.indexName, e.indexKeyField())
	if err != nil {
		return 0, fmt.Errorf("failed to get index dimension: %w", err)
	}
	return resp.GetDimension(), nil
}

// Metric returns the distance metric used by this index
// ("euclidean", "cosine", or "squared_euclidean").
//
// Each call queries the describe endpoint for live state.
//
// Returns:
//   - string: The distance metric
//   - error: Any error encountered during the lookup
func (e *EncryptedIndex) Metric(ctx context.Context) (string, error) {
	resp, err := describeIndex(ctx, e.client, e.indexName, e.indexKeyField())
	if err != nil {
		return "", fmt.Errorf("failed to get index metric: %w", err)
	}
	return resp.GetMetric(), nil
}

// NLists returns the number of inverted lists for this index.
//
// The value is 1 for untrained indexes and the trained cluster count after
// Train. Each call queries the describe endpoint so post-training callers see
// the updated value.
//
// Returns:
//   - int32: The number of inverted lists
//   - error: Any error encountered during the lookup
func (e *EncryptedIndex) NLists(ctx context.Context) (int32, error) {
	resp, err := describeIndex(ctx, e.client, e.indexName, e.indexKeyField())
	if err != nil {
		return 0, fmt.Errorf("failed to get index n_lists: %w", err)
	}
	return resp.GetNLists(), nil
}

// MetadataSchema returns the per-field metadata indexing policy recorded at
// create time, keyed by field name. Empty when the index uses the default
// index-everything posture, or when the service predates the feature.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//
// Returns:
//   - map[string]MetadataFieldPolicy: The recorded policy, never nil
//   - error: Any error encountered during the operation
func (e *EncryptedIndex) MetadataSchema(ctx context.Context) (map[string]MetadataFieldPolicy, error) {
	resp, err := describeIndex(ctx, e.client, e.indexName, e.indexKeyField())
	if err != nil {
		return nil, fmt.Errorf("failed to get index metadata_schema: %w", err)
	}
	schema := resp.GetMetadataSchema()
	if schema == nil {
		schema = map[string]MetadataFieldPolicy{}
	}
	return schema, nil
}

// BM25 returns the BM25 scorer config the index reports back — the K1/B tuning
// parameters and the AnalyzerVersion — or nil when the index has no full-text
// field. BM25 is opt-in and derived: an index with at least one FullText field
// (see CreateIndexParams.TextFields) reports a config; one with none reports
// nil. Mirrors the Python SDK's bm25 property.
//
// Each call queries the describe endpoint for live state.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//
// Returns:
//   - *BM25Config: The scorer config, or nil when BM25 is not configured
//   - error: Any error encountered during the lookup
func (e *EncryptedIndex) BM25(ctx context.Context) (*BM25Config, error) {
	resp, err := describeIndex(ctx, e.client, e.indexName, e.indexKeyField())
	if err != nil {
		return nil, fmt.Errorf("failed to get index bm25 config: %w", err)
	}
	config, ok := resp.GetBm25Ok()
	if !ok || config == nil {
		return nil, nil
	}
	return config, nil
}

// CreateUser mints a user API key scoped to this index.
//
// permissions must be a non-empty subset of {"read", "write"}. The grant is
// enforced cryptographically by the service, not by a checked policy field.
//
// The returned CreatedUser.APIKey is provided exactly once and is never stored
// by the service — capture it now, it cannot be recovered. The new user
// authenticates by passing it as the apiKey to NewClient and needs no index
// key of their own.
//
// Requires the client to be using the index's root key.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - permissions: Non-empty subset of {"read", "write"}
//
// Returns:
//   - *CreatedUser: The new user's ID and one-time API key
//   - error: Any error encountered (e.g. not using the root key, or invalid permissions)
func (e *EncryptedIndex) CreateUser(ctx context.Context, permissions []string) (*CreatedUser, error) {
	// SDK-supplied-KEK indexes: the service needs the index key to unwrap the
	// root DEK and re-wrap it under the new user's key. KMS-backed indexes
	// resolve it server-side, so IndexKey is left unset.
	req := internal.CreateUserRequest{
		Permissions: permissions,
		IndexKey:    e.indexKeyField(),
	}
	resp, _, err := e.client.APIClient.DefaultAPI.CreateUserV1IndexesIndexNameUsersPost(ctx, e.indexName).
		CreateUserRequest(req).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &CreatedUser{UserID: resp.GetUserId(), APIKey: resp.GetApiKey()}, nil
}

// ListUsers lists the users provisioned for this index.
//
// Permissions are derived from which wrapped keys exist for each user (the
// cryptographic source of truth), not a stored field. Requires the client to
// be using the index's root key.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//
// Returns:
//   - []UserInfo: The index's users and their permissions
//   - error: Any error encountered (e.g. not using the root key)
func (e *EncryptedIndex) ListUsers(ctx context.Context) ([]UserInfo, error) {
	request := e.client.APIClient.DefaultAPI.ListUsersV1IndexesIndexNameUsersGet(ctx, e.indexName)
	if e.indexKey != nil {
		request = request.XIndexKey(*e.indexKey)
	}
	resp, _, err := request.Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	users := make([]UserInfo, 0, len(resp.GetUsers()))
	for _, u := range resp.GetUsers() {
		users = append(users, UserInfo{UserID: u.GetUserId(), Permissions: u.GetPermissions()})
	}
	return users, nil
}

// DeleteUser revokes a user, erasing their wrapped keys for this index.
//
// After this returns, the user's API key is rejected on the next request — the
// service can no longer unwrap any key for them. Requires the client to be
// using the index's root key.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - userID: The hex user_id returned by CreateUser (also surfaced by ListUsers)
//
// Returns:
//   - error: Any error encountered during deletion
func (e *EncryptedIndex) DeleteUser(ctx context.Context, userID string) error {
	request := e.client.APIClient.DefaultAPI.DeleteUserV1IndexesIndexNameUsersUserIdDelete(ctx, e.indexName, userID)
	if e.indexKey != nil {
		request = request.XIndexKey(*e.indexKey)
	}
	_, err := request.Execute()
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// Upsert inserts new vectors or updates existing ones in the index.
//
// Vector data is encrypted end-to-end before transmission. If a vector ID
// already exists, it will be updated with the new vector data and metadata.
// This operation is idempotent.
//
// The input type determines the format used:
//   - VectorItems ([]VectorItem): Standard JSON format, suitable for small batches
//   - BinaryUpsertParams: Binary format, more efficient for large batches
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - input: Either VectorItems or BinaryUpsertParams
//
// Returns:
//   - error: Any error encountered during the operation
//
// Example with VectorItems:
//
//	items := VectorItems{
//		{Id: "doc1", Vector: []float32{0.1, 0.2, 0.3}, Metadata: map[string]interface{}{"type": "document"}},
//		{Id: "doc2", Vector: []float32{0.4, 0.5, 0.6}},
//	}
//	err := index.Upsert(ctx, items)
//
// Example with BinaryUpsertParams:
//
//	params := BinaryUpsertParams{
//		IDs:     []string{"doc1", "doc2"},
//		Vectors: [][]float32{{0.1, 0.2, 0.3}, {0.4, 0.5, 0.6}},
//	}
//	err := index.Upsert(ctx, params)
func (e *EncryptedIndex) Upsert(ctx context.Context, input UpsertInput) error {
	switch v := input.(type) {
	case VectorItems:
		return e.upsertItems(ctx, v)
	case BinaryUpsertParams:
		return e.upsertBinary(ctx, v)
	default:
		// This should never happen due to the sealed interface.
		return ErrUnsupportedUpsertType
	}
}

// upsertItems handles standard JSON format upserts.
func (e *EncryptedIndex) upsertItems(ctx context.Context, items VectorItems) error {
	req := internal.UpsertRequest{
		IndexName: e.indexName,
		IndexKey:  e.indexKeyField(),
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
	return e.upsertBinary(ctx, params)
}

// Query performs similarity search to find the nearest neighbors to query vector(s).
//
// The input type determines the query format:
//   - QueryParams: Standard format supporting single vector, batch vectors, or content queries
//   - BinaryQueryParams: Binary format, more efficient for large batch queries
//
// QueryParams supports:
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
//   - input: Either QueryParams or BinaryQueryParams
//
// Returns:
//   - *QueryResponse: Search results with IDs, distances, and requested fields
//   - error: Any error encountered during the search
//
// Example with QueryParams:
//
//	params := QueryParams{
//		QueryVector: []float32{0.1, 0.2, 0.3},
//		TopK: 10,
//		Include: []string{"metadata"},
//		Filters: map[string]interface{}{"category": "document"},
//	}
//	results, err := index.Query(ctx, params)
//
// Example with BinaryQueryParams:
//
//	params := BinaryQueryParams{
//		QueryVectors: [][]float32{{0.1, 0.2, 0.3}, {0.4, 0.5, 0.6}},
//		TopK: 10,
//	}
//	results, err := index.Query(ctx, params)
func (e *EncryptedIndex) Query(ctx context.Context, input QueryInput) (*QueryResponse, error) {
	switch v := input.(type) {
	case QueryParams:
		return e.queryParams(ctx, v)
	case BinaryQueryParams:
		return e.queryBinary(ctx, v)
	default:
		// This should never happen due to the sealed interface.
		return nil, ErrUnsupportedQueryType
	}
}

// queryParams handles standard format queries.
func (e *EncryptedIndex) queryParams(ctx context.Context, params QueryParams) (*QueryResponse, error) {
	h := buildHybrid(params.Text, params.TextFields, params.TextFieldWeights, params.RequireAllTerms, params.Alpha, params.RrfK, params.WindowMult)

	// Handle batch queries using BatchQueryRequest (non-binary format)
	// For binary format with large batches, use QueryBinary() directly
	if len(params.BatchQueryVectors) > 0 {
		batchReq := internal.BatchQueryRequest{
			IndexName:    e.indexName,
			IndexKey:     e.indexKeyField(),
			QueryVectors: params.BatchQueryVectors,
			Filters:      params.Filters,
			Include:      params.Include,
		}

		batchReq.TopK, batchReq.NProbes, batchReq.Greedy = nullableQueryOpts(params.TopK, params.NProbes, params.Greedy)
		if params.RerankMult != nil {
			batchReq.RerankMult = *internal.NewNullableInt32(params.RerankMult)
		}
		batchReq.Text, batchReq.TextFields, batchReq.TextFieldWeights = h.Text, h.TextFields, h.TextFieldWeights
		batchReq.RequireAllTerms, batchReq.Alpha, batchReq.RrfK, batchReq.WindowMult = h.RequireAllTerms, h.Alpha, h.RrfK, h.WindowMult

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
		IndexKey:  e.indexKeyField(),
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

	req.TopK, req.NProbes, req.Greedy = nullableQueryOpts(params.TopK, params.NProbes, params.Greedy)
	if params.RerankMult != nil {
		req.RerankMult = *internal.NewNullableInt32(params.RerankMult)
	}
	req.Text, req.TextFields, req.TextFieldWeights = h.Text, h.TextFields, h.TextFieldWeights
	req.RequireAllTerms, req.Alpha, req.RrfK, req.WindowMult = h.RequireAllTerms, h.Alpha, h.RrfK, h.WindowMult

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
		IndexKey:  e.indexKeyField(),
		Ids:       ids,
		Include:   include,
	}
	result, _, err := e.client.APIClient.DefaultAPI.GetVectorsV1VectorsGetPost(ctx).
		GetRequest(req).
		Execute()
	if err != nil {
		return nil, err
	}
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
		IndexKey:  e.indexKeyField(),
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
// All parameters are optional with sensible defaults. Use IsTrained or
// IsTraining to observe training state, which is read live from the
// server.
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
		IndexKey:  e.indexKeyField(),
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
		IndexKey:  e.indexKeyField(),
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
//
// QueryMetadata finds items by metadata alone — no query vector, no distances.
//
// The filter is resolved entirely from the encrypted metadata index, so this
// works on untrained indexes and never decrypts vectors. It is also the one
// read path where the index's per-field MetadataSchema is enforced instead of
// merely steering performance: $regex/$contains need a pattern field, and a
// Filterable=false field cannot be filtered on. Both return an error — run the
// same filter through Query with a vector if you need those.
//
// Passing params.Text adds a BM25 full-text leg (requires an index with at
// least one full-text field). Results are then ranked by relevance and each
// row in the response's Results carries a Score, in descending order; Filters
// given alongside act as a pre-filter, and OrderBy is not supported with Text.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - params: Filters, plus optional TopK / OrderBy / Ascending, or the Text
//     (BM25) knobs
//
// Returns:
//   - *QueryMetadataResponse: Matching items (Results), their IDs, and count
//   - error: Any error encountered, including a filter the index cannot resolve
//
// Example:
//
//	res, err := index.QueryMetadata(ctx, cyborgdb.QueryMetadataParams{
//		Filters:   map[string]interface{}{"title": map[string]interface{}{"$regex": "^intro"}},
//		OrderBy:   "rank",
//		Ascending: true,
//		TopK:      10,
//	})
func (e *EncryptedIndex) QueryMetadata(ctx context.Context, params QueryMetadataParams) (*QueryMetadataResponse, error) {
	filters := params.Filters
	if filters == nil {
		filters = map[string]interface{}{}
	}

	req := internal.QueryMetadataRequest{
		IndexName: e.indexName,
		IndexKey:  e.indexKeyField(),
		Filters:   filters,
		Ascending: &params.Ascending,
	}
	// TopK and OrderBy are optional on the wire; sending their zero values
	// would mean "return nothing" and "sort by the empty field name".
	if params.TopK > 0 {
		req.TopK = *internal.NewNullableInt32(&params.TopK)
	}
	if params.OrderBy != "" {
		// order_by is an anyOf(str, {field: 1|-1}); a plain field name is the
		// string arm, with direction carried by Ascending.
		orderBy := params.OrderBy
		req.OrderBy = *internal.NewNullableOrderBy(&internal.OrderBy{String: &orderBy})
	}

	// BM25 full-text leg: ranks matches by relevance and populates a Score on
	// each returned row. Unset knobs stay at their Nullable zero value (omitted).
	if params.Text != nil {
		req.Text = *internal.NewNullableString(params.Text)
	}
	req.TextFields = params.TextFields
	req.TextFieldWeights = params.TextFieldWeights
	if params.RequireAllTerms != nil {
		req.RequireAllTerms = *internal.NewNullableBool(params.RequireAllTerms)
	}

	result, _, err := e.client.APIClient.DefaultAPI.QueryMetadataV1VectorsQueryMetadataPost(ctx).
		QueryMetadataRequest(req).
		Execute()
	return result, err
}

func (e *EncryptedIndex) ListIDs(ctx context.Context) (*ListIDsResponse, error) {
	req := internal.ListIDsRequest{
		IndexName: e.indexName,
		IndexKey:  e.indexKeyField(),
	}
	result, _, err := e.client.APIClient.DefaultAPI.ListIdsV1VectorsListIdsPost(ctx).
		ListIDsRequest(req).
		Execute()
	return result, err
}

// vectorsToBase64 converts a 2D slice of float32 vectors to a base64-encoded string.
// The vectors are flattened and encoded as little-endian float32 bytes.
//
// All vectors must share the same dimension (that of vectors[0]); the flattened
// buffer is sized from it, so a ragged batch would otherwise overflow the buffer
// or silently disagree with the Dimension declared on the wire. A mismatch
// returns ErrInconsistentDimension.
func vectorsToBase64(vectors [][]float32) (string, error) {
	if len(vectors) == 0 {
		return "", nil
	}

	// Calculate total number of floats
	numVectors := len(vectors)
	dimension := len(vectors[0])

	// Reject ragged batches before sizing the buffer.
	for i, vec := range vectors {
		if len(vec) != dimension {
			return "", fmt.Errorf("%w: vector %d has dimension %d, expected %d", ErrInconsistentDimension, i, len(vec), dimension)
		}
	}

	totalFloats := numVectors * dimension

	// Create byte buffer (float32ByteSize bytes per float32)
	buf := make([]byte, totalFloats*float32ByteSize)

	// Convert each float32 to little-endian bytes
	offset := 0
	for _, vec := range vectors {
		for _, val := range vec {
			binary.LittleEndian.PutUint32(buf[offset:], math.Float32bits(val))
			offset += float32ByteSize
		}
	}

	return base64.StdEncoding.EncodeToString(buf), nil
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
//	err := index.upsertBinary(ctx, params)
func (e *EncryptedIndex) upsertBinary(ctx context.Context, params BinaryUpsertParams) error {
	if len(params.IDs) == 0 {
		return ErrEmptyIDs
	}
	if len(params.Vectors) == 0 {
		return ErrEmptyVectors
	}
	if len(params.IDs) != len(params.Vectors) {
		return fmt.Errorf("%w: got %d IDs and %d vectors", ErrIDsVectorsLengthMismatch, len(params.IDs), len(params.Vectors))
	}
	if len(params.Metadata) > 0 && len(params.Metadata) != len(params.IDs) {
		return fmt.Errorf("%w: got %d metadata and %d IDs", ErrMetadataLengthMismatch, len(params.Metadata), len(params.IDs))
	}
	if len(params.Contents) > 0 && len(params.Contents) != len(params.IDs) {
		return fmt.Errorf("%w: got %d contents and %d IDs", ErrContentsLengthMismatch, len(params.Contents), len(params.IDs))
	}

	// Encode vectors and read the shared dimension (validates equal lengths)
	vectorsB64, dimension, err := encodeVectorBatch(params.Vectors)
	if err != nil {
		return err
	}

	// Build the batch
	batch := internal.BinaryVectorBatch{
		Ids:        params.IDs,
		VectorsB64: vectorsB64,
		Dimension:  dimension,
	}

	// Add metadata if provided. Index the slice directly rather than taking the
	// address of the range variable, which is shared across iterations under
	// this module's go directive and would alias every entry to the last item.
	if len(params.Metadata) > 0 {
		metadata := make([]*map[string]interface{}, len(params.Metadata))
		for i := range params.Metadata {
			if params.Metadata[i] != nil {
				metadata[i] = &params.Metadata[i]
			}
		}
		batch.Metadata = metadata
	}

	// Add contents if provided. Copy into a per-iteration local before taking
	// its address (see the metadata note above).
	if len(params.Contents) > 0 {
		contents := make([]internal.BinaryVectorBatchContentsInner, len(params.Contents))
		for i := range params.Contents {
			if params.Contents[i] != "" {
				c := params.Contents[i]
				contents[i] = internal.BinaryVectorBatchContentsInner{String: &c}
			}
		}
		batch.Contents = contents
	}

	req := internal.BinaryUpsertRequest{
		IndexName: e.indexName,
		IndexKey:  e.indexKeyField(),
		Batch:     batch,
	}

	_, _, err = e.client.APIClient.DefaultAPI.UpsertVectorsBinaryV1VectorsUpsertBinaryPost(ctx).
		BinaryUpsertRequest(req).
		Execute()
	return err
}

// QueryBinary performs similarity search using binary format for query vectors.
//
// This method is more efficient than Query for large batch queries as vectors
// are encoded in binary format, reducing payload size.
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
//		QueryVectors: [][]float32{{0.1, 0.2, 0.3}, {0.4, 0.5, 0.6}},
//		TopK: 10,
//	}
//	results, err := index.queryBinary(ctx, params)
func (e *EncryptedIndex) queryBinary(ctx context.Context, params BinaryQueryParams) (*QueryResponse, error) {
	if len(params.QueryVectors) == 0 {
		return nil, ErrEmptyQueryVectors
	}

	// Encode vectors and read the shared dimension (validates equal lengths)
	vectorsB64, dimension, err := encodeVectorBatch(params.QueryVectors)
	if err != nil {
		return nil, err
	}

	// Build the batch
	batch := internal.BinaryQueryBatch{
		VectorsB64: vectorsB64,
		Dimension:  dimension,
	}

	req := internal.BinaryQueryRequest{
		IndexName: e.indexName,
		IndexKey:  e.indexKeyField(),
		Batch:     batch,
		Filters:   params.Filters,
		Include:   params.Include,
	}

	req.TopK, req.NProbes, req.Greedy = nullableQueryOpts(params.TopK, params.NProbes, params.Greedy)
	h := buildHybrid(params.Text, params.TextFields, params.TextFieldWeights, params.RequireAllTerms, params.Alpha, params.RrfK, params.WindowMult)
	req.Text, req.TextFields, req.TextFieldWeights = h.Text, h.TextFields, h.TextFieldWeights
	req.RequireAllTerms, req.Alpha, req.RrfK, req.WindowMult = h.RequireAllTerms, h.Alpha, h.RrfK, h.WindowMult

	result, _, err := e.client.APIClient.DefaultAPI.QueryVectorsBinaryV1VectorsQueryBinaryPost(ctx).
		BinaryQueryRequest(req).
		Execute()
	return result, err
}
