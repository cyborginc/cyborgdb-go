// client.go
package cyborgdb

import (
	"context"
	"github.com/cyborginc/cyborgdb-go/internal"
)

// Client provides a high-level interface to the CyborgDB API, similar to the TypeScript SDK.
// It wraps the internal implementation and exposes user-friendly methods.
type Client struct {
	internal *internal.Client // Embedded internal client
}

// NewClient creates a new CyborgDB client instance.
// This is equivalent to the TypeScript constructor: new CyborgDB(baseUrl, apiKey).
//
// Parameters:
//   - baseURL: Base URL of the CyborgDB service (e.g., "https://api.cyborgdb.com")
//   - apiKey: API key for authentication (optional, can be empty string)
//   - verifySSL: Whether to verify SSL certificates (set false for localhost dev)
//
// Returns:
//   - *Client: A new Client instance
//   - error: Any error that occurred during client creation
//
// Example:
//   client, err := cyborgdb.NewClient("https://api.cyborgdb.com", "your-api-key", true)
//   if err != nil {
//       log.Fatal(err)
//   }
func NewClient(baseURL, apiKey string, verifySSL bool) (*Client, error) {
	internalClient, err := internal.NewClient(baseURL, apiKey, verifySSL)
	if err != nil {
		return nil, err
	}
	
	return &Client{
		internal: internalClient,
	}, nil
}

// ListIndexes retrieves a list of all available encrypted index names.
// This method corresponds to the TypeScript method: client.listIndexes()
//
// Parameters:
//   - ctx: Context for request cancellation and timeouts
//
// Returns:
//   - []string: List of index names
//   - error: Any error that occurred during the request
//
// Example:
//   indexes, err := client.ListIndexes(context.Background())
//   if err != nil {
//       log.Printf("Failed to list indexes: %v", err)
//   }
//   for _, indexName := range indexes {
//       fmt.Printf("Found index: %s\n", indexName)
//   }
func (c *Client) ListIndexes(ctx context.Context) ([]string, error) {
	return c.internal.ListIndexes(ctx)
}

// CreateIndex creates a new encrypted vector index.
// This method corresponds to the TypeScript method: client.createIndex()
//
// Parameters:
//   - ctx: Context for request cancellation and timeouts
//   - indexName: Unique name for the index
//   - indexKey: 32-byte encryption key (must be exactly 32 bytes)
//   - indexModel: Index configuration (IndexIVF, IndexIVFPQ, or IndexIVFFlat)
//   - embeddingModel: Optional name of embedding model to associate
//
// Returns:
//   - *EncryptedIndex: A new EncryptedIndex instance for performing operations
//   - error: Any error that occurred during index creation
//
// Example:
//   // Create IVFPQ index
//   indexModel := &cyborgdb.IndexIVFPQ{
//       Dimension: 768,
//       Metric:    "euclidean",
//       NLists:    100,
//       PqDim:     32,
//       PqBits:    8,
//   }
//   
//   key := make([]byte, 32)
//   rand.Read(key) // Generate random key
//   
//   index, err := client.CreateIndex(ctx, "my-index", key, indexModel, nil)
//   if err != nil {
//       log.Printf("Failed to create index: %v", err)
//   }
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
// This method corresponds to the TypeScript method: client.getHealth()
//
// Parameters:
//   - ctx: Context for request cancellation and timeouts
//
// Returns:
//   - *HealthResponse: Health status information from the server
//   - error: Any error that occurred during the health check
//
// Example:
//   health, err := client.GetHealth(context.Background())
//   if err != nil {
//       log.Printf("Health check failed: %v", err)
//   } else {
//       fmt.Printf("Service status: %s\n", *health.Status)
//   }
func (c *Client) GetHealth(ctx context.Context) (*internal.HealthResponse, error) {
	return c.internal.GetHealth(ctx)
}

// handleAPIError provides consistent error handling (private method).
// This corresponds to the TypeScript private method: handleApiError()
func (c *Client) handleAPIError(err error) error {
	// Add any Go-specific error handling/formatting here
	// For now, just return the error as-is, but you could enhance this
	return err
}