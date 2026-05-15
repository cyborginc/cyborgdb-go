// Package cyborgdb provides a Go client for CyborgDB, the confidential vector database.
// This file defines public types used throughout the client API.
package cyborgdb

import (
	"github.com/cyborginc/cyborgdb-go/internal"
)

// Re-export commonly used internal types for public API convenience.
// These maintain compatibility with the internal OpenAPI generated models.

// GetResponse represents the response from Get operations, containing retrieved vectors and metadata.
type GetResponse = internal.GetResponseModel

// VectorItem represents a single vector with ID, vector data, and optional metadata.
type VectorItem = internal.VectorItem

// VectorItems is a slice of VectorItem used for type-safe upsert operations.
type VectorItems []VectorItem

// UpsertInput is implemented by types that can be passed to Upsert.
// Valid types are VectorItems and BinaryUpsertParams.
type UpsertInput interface {
	isUpsertInput()
}

// QueryInput is implemented by types that can be passed to Query.
// Valid types are QueryParams and BinaryQueryParams.
type QueryInput interface {
	isQueryInput()
}

// isUpsertInput implements UpsertInput for VectorItems.
func (VectorItems) isUpsertInput() {}

// isUpsertInput implements UpsertInput for BinaryUpsertParams.
func (BinaryUpsertParams) isUpsertInput() {}

// isQueryInput implements QueryInput for QueryParams.
func (QueryParams) isQueryInput() {}

// isQueryInput implements QueryInput for BinaryQueryParams.
func (BinaryQueryParams) isQueryInput() {}

// QueryResponse represents the response from similarity search operations.
type QueryResponse = internal.QueryResponse

// QueryResultItem represents a single result from a similarity search query.
type QueryResultItem = internal.QueryResultItem

// ListIDsResponse represents the response from ListIDs operations.
type ListIDsResponse = internal.ListIDsResponse

// CreateIndexParams defines the parameters for creating a new encrypted vector index.
//
// All indexes are DiskIVF: a two-stage encrypted ANN index that uses PQ codes for
// fast stage-1 ranking and reranks with the stored full-precision vectors.
//
// Fields:
//   - IndexName: Unique identifier for the index (required)
//   - IndexKey: 32-byte encryption key (required)
//   - Dimension: Vector dimensionality. Auto-detected from the first upsert if omitted.
//   - Metric: Distance metric for similarity calculations (optional, defaults to "euclidean")
//   - EmbeddingModel: Name of embedding model to associate with the index (optional)
type CreateIndexParams struct {
	// IndexName is the unique identifier for this index.
	// Must be unique within your project and contain only alphanumeric characters,
	// hyphens, and underscores.
	IndexName string `json:"index_name"`

	// IndexKey is the 32-byte encryption key used for end-to-end encryption of vector data.
	// Generate using GenerateKey() or provide your own 32-byte key.
	IndexKey []byte `json:"index_key"`

	// Dimension is the vector dimensionality for this index.
	// If nil, the server auto-detects dimensionality from the first upsert.
	Dimension *int32 `json:"dimension,omitempty"`

	// Metric specifies the distance metric for similarity calculations.
	// Supported values include "euclidean", "cosine", "dot_product".
	// Defaults to "euclidean" if not specified.
	Metric *string `json:"metric,omitempty"`

	// EmbeddingModel optionally associates an embedding model name with this index.
	// This is for metadata purposes and doesn't affect index behavior.
	EmbeddingModel *string `json:"embedding_model,omitempty"`
}

// TrainParams defines the parameters for training an encrypted vector index.
//
// Training optimizes the index for better performance by clustering vectors
// and building internal data structures. All parameters are optional and have
// sensible defaults.
//
// Parameters:
//   - BatchSize: Number of vectors processed per training batch (default: 2048)
//   - MaxIters: Maximum training iterations (default: 100)
//   - Tolerance: Convergence tolerance for training (default: 1e-6)
//   - MaxMemory: Maximum memory usage in MB, 0 = no limit (default: 0)
//   - NLists: Number of IVF clusters, 0 = auto-determine (default: 0)
type TrainParams struct {
	// BatchSize controls how many vectors are processed in each training batch.
	// Larger batches may train faster but use more memory. Default: 2048.
	BatchSize *int32 `json:"batch_size,omitempty"`

	// MaxIters sets the maximum number of training iterations.
	// Training may stop early if convergence is reached. Default: 100.
	MaxIters *int32 `json:"max_iters,omitempty"`

	// Tolerance defines the convergence threshold for training.
	// Lower values mean more precise training but longer time. Default: 1e-6.
	Tolerance *float64 `json:"tolerance,omitempty"`

	// MaxMemory limits memory usage during training in MB.
	// Set to 0 for no limit. Default: 0 (unlimited).
	MaxMemory *int32 `json:"max_memory,omitempty"`

	// NLists specifies the number of IVF clusters for index partitioning.
	// Set to 0 for automatic determination based on data size. Default: 0 (auto).
	NLists *int32 `json:"n_lists,omitempty"`
}

// BinaryUpsertParams defines the parameters for binary format vector upserts.
//
// This is more efficient than regular Upsert for large batches as vectors are
// sent as base64-encoded binary data instead of JSON arrays.
//
// Parameters:
//   - IDs: Slice of unique identifiers for each vector (required)
//   - Vectors: 2D slice of float32 vectors, shape [n_vectors][dimension] (required)
//   - Metadata: Optional metadata for each vector (must match IDs length if provided)
//   - Contents: Optional contents for each vector (must match IDs length if provided)
type BinaryUpsertParams struct {
	// IDs contains unique identifiers for each vector.
	// Length must match the number of vectors.
	IDs []string

	// Vectors contains the vector data as a 2D slice.
	// Shape should be [n_vectors][dimension].
	Vectors [][]float32

	// Metadata contains optional metadata for each vector.
	// If provided, length must match IDs length.
	// Use nil for vectors without metadata.
	Metadata []map[string]interface{}

	// Contents contains optional content strings for each vector.
	// If provided, length must match IDs length.
	// Use nil for vectors without contents.
	Contents []string
}

// BinaryQueryParams defines the parameters for binary format similarity search.
//
// This is more efficient than regular Query for batch queries as vectors are
// sent as base64-encoded binary data instead of JSON arrays.
//
// Parameters:
//   - QueryVectors: 2D slice of query vectors, shape [n_queries][dimension] (required)
//   - TopK: Number of nearest neighbors to return (optional, defaults to 100)
//   - NProbes: Number of IVF lists to probe (optional)
//   - Greedy: Enable greedy search mode (optional)
//   - Filters: Metadata filters to apply (optional)
//   - Include: Fields to include in response (optional)
type BinaryQueryParams struct {
	// QueryVectors contains the query vectors as a 2D slice.
	// Shape should be [n_queries][dimension].
	QueryVectors [][]float32

	// TopK specifies the number of nearest neighbors to return.
	// Defaults to 100 if not specified.
	TopK int32

	// NProbes controls the search accuracy vs speed trade-off for IVF indexes.
	// Higher values = more accurate but slower.
	NProbes *int32

	// Greedy enables greedy search mode for potentially faster results.
	Greedy *bool

	// Filters applies metadata-based filtering to search results.
	Filters map[string]interface{}

	// Include specifies which fields to return in results.
	// Common values: ["metadata"], ["vector"], ["metadata", "vector"].
	Include []string
}

// QueryParams defines the parameters for similarity search queries.
//
// Supports both single vector queries and batch queries. Exactly one of
// QueryVector, BatchQueryVectors, or QueryContents must be provided.
//
// Query Types:
//   - Vector query: Provide QueryVector for single query or BatchQueryVectors for batch
//   - Content query: Provide QueryContents for text-based search (if supported)
//
// Required fields: TopK, Include.
// Optional fields: NProbes, Greedy, Filters (and one query input).
type QueryParams struct {
	// QueryVector contains the query vector for single vector similarity search.
	// Mutually exclusive with BatchQueryVectors and QueryContents.
	QueryVector []float32 `json:"query_vector,omitempty"`

	// BatchQueryVectors contains multiple query vectors for batch similarity search.
	// Results will be returned for each query vector in the same order.
	// Mutually exclusive with QueryVector and QueryContents.
	BatchQueryVectors [][]float32 `json:"query_vectors,omitempty"`

	// QueryContents enables content-based search using text input (if supported).
	// The server will embed the text and perform similarity search.
	// Mutually exclusive with QueryVector and BatchQueryVectors.
	QueryContents *string `json:"query_contents,omitempty"`

	// TopK specifies the number of nearest neighbors to return (required).
	// Must be > 0. Server may have maximum limits.
	TopK int32 `json:"top_k"`

	// NProbes controls the search accuracy vs speed trade-off for IVF indexes.
	// Higher values = more accurate but slower. If not set, uses index default.
	NProbes *int32 `json:"n_probes,omitempty"`

	// Greedy enables greedy search mode for potentially faster results.
	// May affect result quality. If not set, uses index default.
	Greedy *bool `json:"greedy,omitempty"`

	// Filters applies metadata-based filtering to search results.
	// Map keys are metadata field names, values are filter criteria.
	// Exact filter syntax depends on server implementation.
	Filters map[string]interface{} `json:"filters,omitempty"`

	// Include specifies which fields to return in results (required).
	// Common values: ["metadata"], ["vector"], ["metadata", "vector"].
	// An empty slice may return only IDs and distances.
	Include []string `json:"include"`
}

