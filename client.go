// client.go
package cyborgdb

import (
	"context"

	"github.com/cyborginc/cyborgdb-go/internal"
)

// Client provides a high-level interface to the CyborgDB API, similar to the TypeScript SDK.
// It wraps the internal implementation and exposes user-friendly methods for managing
// encrypted vector indexes and performing vector database operations.
//
// The Client handles authentication, connection management, and provides methods for:
//   - Creating and listing encrypted indexes
//   - Health checking the CyborgDB service
//   - Managing the lifecycle of vector indexes
//
// All operations performed through this client maintain end-to-end encryption of vector data.
type Client struct {
	internal *internal.Client // Embedded internal client
}

// NewClient creates a new CyborgDB client instance.
//
// The client manages the connection to CyborgDB and handles authentication automatically.
// SSL verification can be disabled for development environments with self-signed certificates.
//
// Parameters:
//   - baseURL: Base URL of the CyborgDB service (e.g., "https://api.cyborgdb.com")
//   - apiKey: API key for authentication (required for most operations)
//   - verifySSL: Whether to verify SSL certificates (set false for localhost development)
//
// Returns:
//   - *Client: A new Client instance ready for use
//   - error: Any error that occurred during client creation
func NewClient(baseURL, apiKey string, verifySSL bool) (*Client, error) {
	internalClient, err := internal.NewClient(baseURL, apiKey, verifySSL)
	if err != nil {
		return nil, err
	}

	return &Client{
		internal: internalClient,
	}, nil
}

// ListIndexes retrieves a list of all available encrypted index names from your CyborgDB instance.
//
// This operation queries the CyborgDB service for all indexes that have been created
// under your API key. The returned list contains only the index names, not their
// configurations or contents.
//
// Parameters:
//   - ctx: Context for request cancellation, timeouts, and tracing
//
// Returns:
//   - []string: List of index names (empty slice if no indexes exist)
//   - error: Any error that occurred during the request
func (c *Client) ListIndexes(ctx context.Context) ([]string, error) {
	return c.internal.ListIndexes(ctx)
}

// CreateIndex creates a new encrypted vector index with the specified configuration.
//
// The created index will be empty and ready for vector operations. Different index types
// (IVF, IVFPQ, IVFFlat) offer different trade-offs between speed, accuracy, and memory usage.
// All vector data stored in the index is encrypted using the provided encryption key.
//
// Parameters:
//   - ctx: Context for request cancellation, timeouts, and tracing
//   - indexName: Unique name for the index (must be unique within your CyborgDB instance)
//   - indexKey: 32-byte encryption key (generate using crypto/rand for security)
//   - indexModel: Index configuration specifying type, dimension, and parameters
//   - embeddingModel: Optional name of embedding model to associate with this index
//
// Returns:
//   - *EncryptedIndex: A new EncryptedIndex instance for performing vector operations
//   - error: Any error that occurred during index creation
//
// Note: Store the encryption key securely - it cannot be recovered if lost.
// The index name must be unique; creating an index with an existing name will fail.
func (c *Client) CreateIndex(
	ctx context.Context,
	indexName string,
	indexKey []byte,
	indexModel internal.IndexModel,
	embeddingModel *string,
) (*EncryptedIndex, error) {
	internalIndex, err := c.internal.CreateIndex(ctx, indexName, indexKey, indexModel, embeddingModel)
	if err != nil {
		return nil, err
	}

	// Wrap the internal EncryptedIndex with our public one
	return &EncryptedIndex{
		internal: internalIndex,
	}, nil
}

// GetHealth checks the health status of the CyborgDB service.
//
// This is useful for monitoring, readiness checks, and verifying that the CyborgDB
// service is accessible and operational. The health check typically does not require
// authentication and can be used to test connectivity.
//
// Parameters:
//   - ctx: Context for request cancellation, timeouts, and tracing
//
// Returns:
//   - *HealthResponse: Health status information from the server
//   - error: Any error that occurred during the health check
func (c *Client) GetHealth(ctx context.Context) (*internal.HealthResponse, error) {
	return c.internal.GetHealth(ctx)
}
