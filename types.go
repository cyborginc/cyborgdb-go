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

// QueryMetadataResponse represents the response from QueryMetadata operations:
// the matching items (Results), their IDs, and their count.
//
// Results holds one MetadataResult row per match. On a text (BM25) query each
// row carries a relevance Score, in descending order; on a filter-only query
// each row is just an Id (Score unset). Ids/Count are retained for callers that
// only read IDs.
type QueryMetadataResponse = internal.QueryMetadataResponse

// MetadataResult is one row of a QueryMetadata result: the item Id, plus a BM25
// relevance Score (accessed via GetScore/HasScore) present only on the text
// path. A filter-only query has nothing to score, so Score is unset there —
// the same convention Query uses for distance vs. score. Mirrors the Python
// SDK's MetadataResult.
type MetadataResult = internal.MetadataResult

// BM25Config is the BM25 scorer config an index reports back via
// EncryptedIndex.BM25: K1 (term-frequency saturation) and B (length
// normalization) as supplied at create time or their defaults, plus
// AnalyzerVersion identifying the tokenizer/stemmer pipeline the corpus was
// indexed with. Present only for indexes with at least one full-text field.
type BM25Config = internal.BM25Config

// QueryMetadataParams configures a metadata-only query.
//
// Filters resolve entirely from the encrypted metadata index — there is no
// post-filter stage — so the index's MetadataSchema is enforced here rather
// than being the performance hint it is on Query: $regex/$contains require a
// pattern field, and a field created with Filterable=false cannot be filtered
// on at all. Both come back as an error; use Query with a vector instead.
type QueryMetadataParams struct {
	// Filters is a MongoDB-style filter; nil or empty matches everything.
	Filters map[string]interface{}
	// TopK caps the IDs returned, applied AFTER OrderBy. Zero returns all.
	TopK int32
	// OrderBy sorts matches by a metadata field, post-filter. Empty leaves
	// the result unordered.
	OrderBy string
	// Ascending sets the sort direction when OrderBy is set. The zero value
	// is false, so use QueryMetadataParams{OrderBy: f, Ascending: true} for
	// ascending order; it is ignored when OrderBy is empty.
	Ascending bool

	// Text adds a BM25 full-text leg, ranking matches by relevance (requires an
	// index with at least one full-text field). Results then carry a Score in
	// descending order, and any Filters act as a pre-filter. OrderBy is not
	// supported alongside Text. Nil/empty keeps this a filter-only query.
	Text *string

	// TextFields restricts the text leg to these full-text fields; nil means
	// all of them. Naming a non-full-text field is rejected by the service.
	TextFields []string

	// TextFieldWeights are per-field weights on the summed per-field BM25
	// scores, parallel to the searched fields. Nil means 1.0 each.
	TextFieldWeights []float32

	// RequireAllTerms requires every query term to match (AND) instead of any
	// (OR, the default). Nil uses the server default.
	RequireAllTerms *bool
}

// Storage precision constants for the on-disk rerank-vector dtype.
//
// The float tiers store rerank vectors as IEEE floats. The TurboQuant ("tq")
// tiers quantize each dimension to the given number of bits, trading a small
// recall/latency cost for large storage savings. The precision is chosen at
// index creation and is immutable.
const (
	// StoragePrecisionFloat32 stores rerank vectors as 32-bit floats (default).
	StoragePrecisionFloat32 = "float32"
	// StoragePrecisionFloat16 stores rerank vectors as 16-bit floats, halving disk usage.
	StoragePrecisionFloat16 = "float16"
	// StoragePrecisionTQ8 uses TurboQuant with 8 bits per dimension.
	StoragePrecisionTQ8 = "tq8"
	// StoragePrecisionTQ6 uses TurboQuant with 6 bits per dimension.
	StoragePrecisionTQ6 = "tq6"
	// StoragePrecisionTQ4 uses TurboQuant with 4 bits per dimension. Requires the cosine metric.
	StoragePrecisionTQ4 = "tq4"
)

// CreateIndexParams defines the parameters for creating a new encrypted DiskIVF index.
//
// At least one of IndexKey or KmsName must be provided:
//
//   - IndexKey only — the SDK supplies the 32-byte DEK directly (legacy path).
//   - KmsName only — the service generates a fresh DEK and wraps it under the
//     named kms.registry entry; the SDK never sees the plaintext DEK.
//   - IndexKey + KmsName — only valid when KmsName references a "provider: none"
//     registry entry, in which case IndexKey is the wrapping KEK rather than
//     the DEK itself. Passing both against a real-KMS slot
//     ("provider: aws-kms" or "aws") is rejected by the service with a 400.
type CreateIndexParams struct {
	// IndexName is the unique identifier for this index.
	// Must be unique within your project and contain only alphanumeric characters,
	// hyphens, and underscores.
	IndexName string `json:"index_name"`

	// IndexKey is the 32-byte encryption key. Optional when KmsName is set
	// against a real KMS entry (the service generates and wraps the DEK).
	// Required when KmsName is unset (legacy) or references a
	// "provider: none" entry (the KEK).
	IndexKey []byte `json:"index_key,omitempty"`

	// KmsName is the name of a kms.registry entry in the service YAML.
	// When supplied, the service handles DEK generation and wrapping.
	KmsName *string `json:"kms_name,omitempty"`

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

	// StoragePrecision selects the on-disk rerank-vector dtype, chosen at
	// create time and immutable. Use StoragePrecisionFloat32 (default),
	// StoragePrecisionFloat16, or a TurboQuant tier (StoragePrecisionTQ8,
	// StoragePrecisionTQ6, StoragePrecisionTQ4). StoragePrecisionTQ4 requires
	// the cosine metric.
	StoragePrecision *string `json:"storage_precision,omitempty"`

	// MetadataSchema is the per-field metadata indexing policy, fixed at
	// create time and immutable. Fields left out are filterable (opt-out
	// posture); Pattern requires Filterable. On Query this only decides how a
	// filter resolves — index vs. post-filter, same rows either way — but
	// QueryMetadata enforces it.
	//
	//	MetadataSchema: map[string]cyborgdb.MetadataFieldPolicy{
	//		"title": {Filterable: ptr(true), Pattern: ptr(true)},
	//	}
	//
	// A field can also opt into BM25 full-text search with FullText=true (which
	// implies Filterable=false and is incompatible with Pattern=true); the
	// TextFields shorthand below marks fields FullText for you.
	MetadataSchema map[string]MetadataFieldPolicy `json:"metadata_schema,omitempty"`

	// TextFields marks these metadata fields FullText=true — routing their
	// string values through the BM25 analyzer so they are searchable by
	// EncryptedIndex.QueryMetadata (Text) and hybrid Query (Text). Shorthand
	// for the FullText policy in MetadataSchema. BM25 is opt-in: an index with
	// no full-text field writes no BM25 config at all.
	TextFields []string `json:"text_fields,omitempty"`

	// Bm25K1 tunes term-frequency saturation for the BM25 scorer (default 1.2).
	// Requires at least one full-text field. Nil uses the default.
	Bm25K1 *float64 `json:"bm25_k1,omitempty"`

	// Bm25B tunes length-normalization strength for the BM25 scorer (default
	// 0.75). Requires at least one full-text field. Nil uses the default.
	Bm25B *float64 `json:"bm25_b,omitempty"`
}

// MetadataFieldPolicy is one field's entry in a CreateIndexParams.MetadataSchema:
// Filterable builds inverted-index postings, Pattern additionally builds the
// regex dictionary that $regex / $contains need. Both default to the shipped
// posture when nil (filterable, not pattern).
type MetadataFieldPolicy = internal.MetadataFieldPolicy

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

	// Text turns this into a hybrid (BM25 + vector) query against an index with
	// at least one full-text field. Hybrid results carry a fused Score instead
	// of a distance. Nil/empty leaves the query text-free (pure vector search).
	// See QueryParams for the text-leg / fusion knobs below.
	Text *string

	// TextFields restricts the text leg to these full-text fields; nil means all.
	TextFields []string

	// TextFieldWeights are per-field weights, parallel to the searched fields.
	TextFieldWeights []float32

	// RequireAllTerms requires every query term to match (AND vs OR).
	RequireAllTerms *bool

	// Alpha blends the two legs in [0, 1] (0 = BM25, 1 = vector; default 0.5).
	Alpha *float64

	// RrfK is the RRF rank-smoothing constant (> 0; default 60).
	RrfK *float64

	// WindowMult sets per-leg candidate depth as a multiple of TopK (>= 1; default 3).
	WindowMult *int32
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

	// RerankMult is the stage-1 retrieval multiplier for reranking indexes.
	// Higher values retrieve more candidates before reranking, trading speed
	// for recall. If not set, the server applies its default (10).
	RerankMult *int32 `json:"rerank_mult,omitempty"`

	// Filters applies metadata-based filtering to search results.
	// Map keys are metadata field names, values are filter criteria.
	// Exact filter syntax depends on server implementation.
	Filters map[string]interface{} `json:"filters,omitempty"`

	// Include specifies which fields to return in results (required).
	// Common values: ["metadata"], ["vector"], ["metadata", "vector"].
	// An empty slice may return only IDs and distances.
	Include []string `json:"include"`

	// Text turns this into a hybrid (BM25 + vector) query against an index with
	// at least one full-text field; a query vector is still required. Hybrid
	// results carry a fused Score (larger = more relevant, descending) instead
	// of a distance. Nil/empty leaves the query text-free (pure vector search).
	Text *string `json:"text,omitempty"`

	// TextFields restricts the text leg to these full-text fields; nil means
	// all of them. Naming a non-full-text field is rejected by the service.
	TextFields []string `json:"text_fields,omitempty"`

	// TextFieldWeights are per-field weights on the summed per-field BM25
	// scores, parallel to the searched fields. Nil means 1.0 each.
	TextFieldWeights []float32 `json:"text_field_weights,omitempty"`

	// RequireAllTerms requires every query term to match (AND) instead of any
	// (OR, the default). Nil uses the server default.
	RequireAllTerms *bool `json:"require_all_terms,omitempty"`

	// Alpha blends the two legs in [0, 1]: 0 = pure BM25, 1 = pure vector.
	// Nil uses the server default (0.5).
	Alpha *float64 `json:"alpha,omitempty"`

	// RrfK is the RRF rank-smoothing constant (> 0). Nil uses the server
	// default (60).
	RrfK *float64 `json:"rrf_k,omitempty"`

	// WindowMult sets per-leg candidate depth as a multiple of TopK (>= 1).
	// Nil uses the server default (3).
	WindowMult *int32 `json:"window_mult,omitempty"`
}

// CreatedUser holds the credentials minted by EncryptedIndex.CreateUser.
//
// The APIKey is returned exactly once and is never stored by the service —
// capture it immediately, as it cannot be recovered. Hand it to the user;
// they authenticate by passing it as the apiKey to NewClient and need no
// index key of their own.
type CreatedUser struct {
	// UserID is the hex-encoded identifier for the new user.
	UserID string `json:"user_id"`

	// APIKey is the cdbk_ user API key. Returned once; never recoverable.
	APIKey string `json:"api_key"`
}

// UserInfo describes a user provisioned for an index, as returned by
// EncryptedIndex.ListUsers.
type UserInfo struct {
	// UserID is the hex-encoded identifier for the user.
	UserID string `json:"user_id"`

	// Permissions is the granted subset of {"read", "write"}, derived from
	// which wrapped keys exist for the user (the cryptographic source of
	// truth), not a stored field.
	Permissions []string `json:"permissions"`
}
