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

// QueryResponse represents the response from similarity search operations.
type QueryResponse = internal.QueryResponse

// QueryResultItem represents a single result from a similarity search query.
type QueryResultItem = internal.QueryResultItem

// CreateIndexRequest represents the low-level request structure for index creation.
type CreateIndexRequest = internal.CreateIndexRequest

// ListIDsResponse represents the response from ListIDs operations.
type ListIDsResponse = internal.ListIDsResponse

// IndexModel is the interface implemented by all index configuration types.
// It allows type-safe creation of different index configurations (IVF, IVFFlat, IVFPQ)
// while maintaining compatibility with the internal OpenAPI models.
type IndexModel interface {
	// ToIndexConfig converts the public type to the internal IndexConfig structure.
	ToIndexConfig() *internal.IndexConfig
}

// CreateIndexParams defines the parameters for creating a new encrypted vector index.
//
// This type provides a more ergonomic interface than the internal CreateIndexRequest,
// accepting IndexModel types for type-safe index configuration.
//
// Fields:
//   - IndexName: Unique identifier for the index (required)
//   - IndexKey: 64-character hex string of the 32-byte encryption key (required)
//   - IndexConfig: Index configuration specifying the index type and parameters (optional)
//   - Metric: Distance metric for similarity calculations (optional, defaults to "euclidean")
//   - EmbeddingModel: Name of embedding model to associate with the index (optional)
type CreateIndexParams struct {
	// IndexName is the unique identifier for this index.
	// Must be unique within your project and contain only alphanumeric characters,
	// hyphens, and underscores.
	IndexName string `json:"index_name"`
	
	// IndexKey is the 64-character hex string representation of a 32-byte encryption key.
	// This key is used for end-to-end encryption of vector data.
	// Generate using GenerateKey() and convert to hex, or use hex.EncodeToString().
	IndexKey string `json:"index_key"`
	
	// IndexConfig specifies the index type and configuration.
	// Can be created using IndexIVF(), IndexIVFFlat(), or IndexIVFPQ() functions.
	// If nil, the server will use default configuration.
	IndexConfig IndexModel `json:"index_config,omitempty"`
	
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

// QueryParams defines the parameters for similarity search queries.
//
// Supports both single vector queries and batch queries. Exactly one of
// QueryVector, BatchQueryVectors, or QueryContents must be provided.
//
// Query Types:
//   - Vector query: Provide QueryVector for single query or BatchQueryVectors for batch
//   - Content query: Provide QueryContents for text-based search (if supported)
//
// Required fields: TopK, Include
// Optional fields: NProbes, Greedy, Filters (and one query input)
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

// Index model wrapper types provide type-safe access to different index configurations.
// These types wrap the internal OpenAPI generated models and implement the IndexModel interface.

// indexIVF wraps the IVF (Inverted File) index configuration.
// IVF indexes provide fast approximate search by partitioning vectors into clusters.
type indexIVF struct {
	*internal.IndexIVFModel
}

// indexIVFFlat wraps the IVFFlat index configuration.
// IVFFlat combines IVF clustering with flat (exact) search within clusters.
type indexIVFFlat struct {
	*internal.IndexIVFFlatModel
}

// indexIVFPQ wraps the IVFPQ (IVF with Product Quantization) index configuration.
// IVFPQ provides the most compact storage by using product quantization compression.
type indexIVFPQ struct {
	*internal.IndexIVFPQModel
}

// IndexIVF creates a new IVF (Inverted File) index configuration.
//
// IVF indexes partition vectors into clusters for fast approximate search.
// They offer a good balance of speed and accuracy for most use cases.
//
// Parameters:
//   - dimension: The dimensionality of vectors that will be stored (e.g., 768 for many embedding models)
//
// Returns:
//   - *indexIVF: IVF index configuration implementing IndexModel
//
// Usage:
//   config := IndexIVF(768) // For 768-dimensional vectors
func IndexIVF(dimension int32) *indexIVF {
	model := &internal.IndexIVFModel{}
	model.SetDimension(dimension)
	model.SetType("ivf")
	return &indexIVF{IndexIVFModel: model}
}

// IndexIVFFlat creates a new IVFFlat index configuration.
//
// IVFFlat combines IVF clustering with flat (exact) search within each cluster.
// This provides higher accuracy than plain IVF but uses more memory and compute.
//
// Parameters:
//   - dimension: The dimensionality of vectors that will be stored
//
// Returns:
//   - *indexIVFFlat: IVFFlat index configuration implementing IndexModel
//
// Usage:
//   config := IndexIVFFlat(768) // For 768-dimensional vectors
func IndexIVFFlat(dimension int32) *indexIVFFlat {
	model := &internal.IndexIVFFlatModel{}
	model.SetDimension(dimension)
	model.SetType("ivfflat")
	return &indexIVFFlat{IndexIVFFlatModel: model}
}

// IndexIVFPQ creates a new IVFPQ (IVF with Product Quantization) index configuration.
//
// IVFPQ provides the most memory-efficient storage by using product quantization
// to compress vectors. This enables handling of very large datasets but with
// some accuracy trade-off.
//
// Parameters:
//   - dimension: The dimensionality of vectors that will be stored
//   - pqDim: Product quantization dimension (typically dimension/8 or dimension/16)
//   - pqBits: Bits per PQ code (typically 8, higher = more accurate but larger)
//
// Returns:
//   - *indexIVFPQ: IVFPQ index configuration implementing IndexModel
//
// Usage:
//   config := IndexIVFPQ(768, 96, 8) // 768-dim vectors, 96 PQ dim, 8 bits per code
func IndexIVFPQ(dimension int32, pqDim int32, pqBits int32) *indexIVFPQ {
	model := &internal.IndexIVFPQModel{
		PqDim:  pqDim,
		PqBits: pqBits,
	}
	model.SetDimension(dimension)
	model.SetType("ivfpq")
	return &indexIVFPQ{IndexIVFPQModel: model}
}

// ToIndexConfig converts the IVF index configuration to the internal IndexConfig format.
// This method implements the IndexModel interface.
func (m *indexIVF) ToIndexConfig() *internal.IndexConfig {
	return &internal.IndexConfig{
		IndexIVFModel: m.IndexIVFModel,
	}
}

// ToIndexConfig converts the IVFFlat index configuration to the internal IndexConfig format.
// This method implements the IndexModel interface.
func (m *indexIVFFlat) ToIndexConfig() *internal.IndexConfig {
	return &internal.IndexConfig{
		IndexIVFFlatModel: m.IndexIVFFlatModel,
	}
}

// ToIndexConfig converts the IVFPQ index configuration to the internal IndexConfig format.
// This method implements the IndexModel interface.
func (m *indexIVFPQ) ToIndexConfig() *internal.IndexConfig {
	return &internal.IndexConfig{
		IndexIVFPQModel: m.IndexIVFPQModel,
	}
}
