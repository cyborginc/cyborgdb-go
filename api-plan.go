// Package cyborgdb provides a Go client for interacting with the CyborgDB vector database service.
//
// CyborgDB is a secure, encrypted vector database that allows you to store and query
// high-dimensional vectors with end-to-end encryption. This package provides a
// comprehensive Go client for all CyborgDB operations.

package cyborgdb

import (
	"context"
)

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

// GenerateKey generates a cryptographically secure 32-byte key for use with CyborgDB indexes.
//
// This function uses the system's cryptographically secure random number generator
// to create a key suitable for encrypting vector indexes. The returned key should
// be stored securely and used consistently for all operations on a given index.
//
// Returns:
//   - A 32-byte slice containing the generated key
//   - An error if the key generation fails
func GenerateKey() ([]byte, error)

// =============================================================================
// TYPE DEFINITIONS
// =============================================================================

// IndexConfig represents the configuration for creating a vector index.
type IndexConfig struct {
	// Dimension is the dimensionality of the vectors to be stored.
	Dimension int `json:"dimension"`

	// Metric specifies the distance metric to use for similarity calculations.
	// Valid values: "l2", "ip", "cosine"
	Metric string `json:"metric"`

	// IndexType specifies the type of index to create.
	// Valid values: "ivf", "ivf_flat", "ivf_pq"
	IndexType string `json:"index_type"`

	// NLists is the number of clusters for IVF-based indexes.
	NLists int `json:"n_lists"`

	// PQDimension is the dimension for product quantization (only for ivf_pq).
	PQDimension *int `json:"pq_dim,omitempty"`

	// PQBits is the number of bits for product quantization (only for ivf_pq).
	PQBits *int `json:"pq_bits,omitempty"`
}

// VectorItem represents a single vector item with its associated data.
type VectorItem struct {
	// ID is the unique identifier for this vector item.
	ID string `json:"id"`

	// Vector is the vector embedding for this item.
	Vector []float64 `json:"vector,omitempty"`

	// Contents is the optional text content associated with this vector.
	Contents *string `json:"contents,omitempty"`

	// Metadata contains optional key-value metadata for this vector.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// QueryRequest represents a request to query vectors from an index.
type QueryRequest struct {
	// IndexName is the name of the index to query.
	IndexName string `json:"index_name"`

	// IndexKey is the hex-encoded encryption key for the index.
	IndexKey string `json:"index_key"`

	// QueryVector is a single vector to search for (mutually exclusive with QueryVectors).
	QueryVector []float64 `json:"query_vector,omitempty"`

	// QueryVectors is a batch of vectors to search for (mutually exclusive with QueryVector).
	QueryVectors [][]float64 `json:"query_vectors,omitempty"`

	// QueryContents is text content to search for using the embedding model.
	QueryContents *string `json:"query_contents,omitempty"`

	// TopK is the number of nearest neighbors to return per query.
	TopK int `json:"top_k"`

	// NProbes is the number of clusters to search (affects speed vs accuracy).
	NProbes int `json:"n_probes"`

	// Greedy enables greedy search mode for potentially better results.
	Greedy bool `json:"greedy"`

	// Filters contains metadata filters to apply to the search.
	Filters map[string]interface{} `json:"filters,omitempty"`

	// Include specifies which fields to include in the response.
	// Valid values: "distance", "metadata", "vector", "contents"
	Include []string `json:"include"`
}

// QueryResult represents a single result from a vector query.
type QueryResult struct {
	// ID is the unique identifier of the result vector.
	ID string `json:"id"`

	// Distance is the distance between the query and result vector.
	Distance *float64 `json:"distance,omitempty"`

	// Metadata contains the metadata associated with this result.
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Vector is the vector embedding of this result.
	Vector []float64 `json:"vector,omitempty"`

	// Contents is the text content associated with this result.
	Contents *string `json:"contents,omitempty"`
}

// QueryResponse represents the response from a vector query operation.
type QueryResponse struct {
	// Results contains the query results. For single vector queries, this will
	// contain one slice. For batch queries, this will contain one slice per query vector.
	Results [][]QueryResult `json:"results"`
}

// HealthResponse represents the response from a health check operation.
type HealthResponse struct {
	// Status indicates the health status of the service.
	Status string `json:"status"`
}

// =============================================================================
// CLIENT API
// =============================================================================

// Client provides access to the CyborgDB vector database service.
//
// The Client handles authentication, request routing, and response parsing
// for all CyborgDB operations. It maintains an HTTP client for network
// communication and provides methods for managing encrypted vector indexes.
type Client struct {
	baseURL string
	apiKey  string
}

// NewClient creates a new CyborgDB client with the specified configuration.
//
// The client will use the provided base URL and API key for all requests.
//
// Parameters:
//   - baseURL: The base URL of the CyborgDB service (e.g., "https://api.cyborgdb.co")
//   - apiKey: The API key for authentication (can be empty for unauthenticated requests)
//
// Returns:
//   - A configured Client instance ready for use
func NewClient(baseURL, apiKey string) *Client

// ListIndexes retrieves a list of all available encrypted indexes.
//
// This method returns the names of all indexes that are accessible with the
// current API key. The returned names can be used to create EncryptedIndex
// instances for existing indexes.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout control
//
// Returns:
//   - A slice of index names
//   - An error if the request fails or the response cannot be parsed
func (c *Client) ListIndexes(ctx context.Context) ([]string, error)

// CreateIndex creates a new encrypted vector index with the specified configuration.
//
// This method creates a new index on the CyborgDB service and returns an
// EncryptedIndex instance for interacting with it. The index will be encrypted
// using the provided key, which must be exactly 32 bytes long.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout control
//   - indexName: The name of the index to create (must be unique)
//   - indexKey: A 32-byte encryption key for the index
//   - config: Configuration specifying the index type and parameters
//   - embeddingModel: Optional name of the embedding model to use for text queries
//
// Returns:
//   - An EncryptedIndex instance for the newly created index
//   - An error if the index creation fails or the parameters are invalid
func (c *Client) CreateIndex(ctx context.Context, indexName string, indexKey []byte, config IndexConfig, embeddingModel *string) (*EncryptedIndex, error)

// GetHealth checks the health status of the CyborgDB service.
//
// This method performs a health check against the CyborgDB service to verify
// that it is available and responding correctly. It can be used for monitoring
// and diagnostic purposes.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout control
//
// Returns:
//   - A HealthResponse containing the service status
//   - An error if the health check fails or the response cannot be parsed
func (c *Client) GetHealth(ctx context.Context) (*HealthResponse, error)

// =============================================================================
// ENCRYPTED INDEX API
// =============================================================================

// EncryptedIndex provides access to an encrypted vector index in CyborgDB.
//
// An EncryptedIndex represents a single encrypted vector index and provides
// methods for adding, querying, and managing vectors within that index.
// All operations are performed using the encryption key provided during
// index creation or loading.
type EncryptedIndex struct {
	client    *Client
	indexName string
	indexKey  []byte
	config    IndexConfig
}

// IndexName returns the name of this encrypted index.
//
// Returns:
//   - The string name of the index
func (e *EncryptedIndex) IndexName() string

// IndexType returns the type of this encrypted index.
//
// Returns:
//   - The string type of the index (e.g., "ivf_flat", "ivf_pq")
func (e *EncryptedIndex) IndexType() string

// Train trains the index for efficient approximate nearest neighbor search.
//
// Training builds internal data structures that enable fast similarity search.
// This method must be called before performing queries on IVF-based indexes.
// The index must contain at least 2 * nLists vectors before training can succeed.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout control
//   - batchSize: Number of vectors to process in each training batch
//   - maxIterations: Maximum number of training iterations to perform
//   - tolerance: Convergence tolerance for training termination
//
// Returns:
//   - An error if training fails or the index doesn't have enough vectors
func (e *EncryptedIndex) Train(ctx context.Context, batchSize, maxIterations int, tolerance float64) error

// Upsert adds or updates vectors in the encrypted index.
//
// This method adds new vectors to the index or updates existing vectors
// if they have the same ID. Vectors are encrypted before storage using
// the index's encryption key.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout control
//   - items: Slice of VectorItem objects to add or update
//
// Returns:
//   - An error if the upsert operation fails
func (e *EncryptedIndex) Upsert(ctx context.Context, items []VectorItem) error

// Query searches for the nearest neighbors of the specified query vectors.
//
// This method performs similarity search against the encrypted index and
// returns the most similar vectors. The search can be performed using
// vector embeddings or text content (if an embedding model is configured).
//
// Parameters:
//   - ctx: Context for request cancellation and timeout control
//   - request: QueryRequest containing the search parameters and query data
//
// Returns:
//   - A QueryResponse containing the search results
//   - An error if the query fails
func (e *EncryptedIndex) Query(ctx context.Context, request QueryRequest) (*QueryResponse, error)

// Get retrieves vectors from the index by their IDs.
//
// This method fetches specific vectors from the encrypted index using their
// unique identifiers. The returned vectors are decrypted automatically.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout control
//   - ids: Slice of string IDs to retrieve
//   - include: Slice of field names to include in the response
//
// Returns:
//   - A slice of VectorItem objects containing the requested data
//   - An error if the retrieval fails
func (e *EncryptedIndex) Get(ctx context.Context, ids []string, include []string) ([]VectorItem, error)

// Delete removes vectors from the encrypted index by their IDs.
//
// This method permanently removes the specified vectors from the index.
// The operation cannot be undone.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout control
//   - ids: Slice of string IDs to delete
//
// Returns:
//   - An error if the deletion fails
func (e *EncryptedIndex) Delete(ctx context.Context, ids []string) error

// DeleteIndex permanently deletes the entire encrypted index and all its data.
//
// This method removes the index and all vectors contained within it from
// the CyborgDB service. The operation cannot be undone.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout control
//
// Returns:
//   - An error if the index deletion fails
func (e *EncryptedIndex) DeleteIndex(ctx context.Context) error

// UpsertVectors adds or updates vectors using separate slices (Structure of Arrays pattern).
//
// This method provides an alternative to Upsert() for cases where you have
// vector data in columnar format. It's particularly useful for bulk operations
// and when you don't need to associate metadata with every vector.
//
// Parameters:
//   - ctx: Context for request cancellation and timeout control
//   - ids: Slice of unique identifiers for the vectors
//   - vectors: Slice of vector embeddings (must match length of ids)
//
// Returns:
//   - An error if the upsert operation fails or slice lengths don't match
func (e *EncryptedIndex) UpsertVectors(ctx context.Context, ids []string, vectors [][]float64) error
