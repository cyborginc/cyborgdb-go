// client.go
package cyborgdb

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/url"

	"github.com/cyborginc/cyborgdb-go/internal"
)

const (
	// KeySize is the required size in bytes for encryption keys (32 bytes for AES-256).
	KeySize = 32
)

var (
	// ErrInvalidKeyLength is returned when an index key is not 32 bytes.
	ErrInvalidKeyLength = fmt.Errorf("index key must be exactly 32 bytes")
	// ErrKeyGeneration is returned when key generation fails.
	ErrKeyGeneration = fmt.Errorf("failed to generate key")
	// ErrInvalidURL is returned when the base URL is invalid.
	ErrInvalidURL = fmt.Errorf("invalid base URL")
)

// Client provides a high-level interface to the CyborgDB API (parallels the TypeScript SDK).
// It wraps the internal client and exposes ergonomic methods for managing encrypted indexes
// and performing vector operations, handling auth and connection details.
//
// The Client supports:
//   - Creating and loading encrypted indexes
//   - Listing indexes
//   - Upserting/querying/deleting vectors via EncryptedIndex
//   - Health checks
//
// All operations maintain end-to-end encryption for vector data.
type Client struct {
	internal *internal.Client // Embedded internal client
}

// GenerateKey returns a cryptographically secure 32-byte key for use with CyborgDB indexes.
//
// The caller must persist this key securely; it cannot be recovered if lost.
//
// Returns:
//   - []byte: A 32-byte encryption key
//   - error: Any error that occurred during key generation
func GenerateKey() ([]byte, error) {
	key := make([]byte, KeySize)
	_, err := rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKeyGeneration, err)
	}
	return key, nil
}

// NewClient constructs a new CyborgDB client.
//
// If verifySSL is omitted, behavior matches the TS SDK:
//   - "http://" URLs -> verifySSL = false
//   - localhost / 127.0.0.1 -> verifySSL = false
//   - otherwise -> verifySSL = true
//
// Usage:
//
//	NewClient(url, apiKey)        // auto-detect verifySSL
//	NewClient(url, apiKey, false) // force off
//	NewClient(url, apiKey, true)  // force on
func NewClient(baseURL, apiKey string, verifySSL ...bool) (*Client, error) {
	// Explicit override wins.
	if len(verifySSL) > 0 {
		v := verifySSL[0]
		internalClient, err := internal.NewClient(baseURL, apiKey, v)
		if err != nil {
			return nil, err
		}
		return &Client{internal: internalClient}, nil
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}
	v := true
	if u.Scheme == "http" {
		v = false
	} else {
		host := u.Hostname()
		if host == "localhost" || host == "127.0.0.1" {
			v = false
		}
	}

	internalClient, err := internal.NewClient(baseURL, apiKey, v)
	if err != nil {
		return nil, err
	}
	return &Client{internal: internalClient}, nil
}

// ListIndexes returns the names of all encrypted indexes in your project.
//
// Parameters:
//   - ctx: Context for cancellation/timeouts
//
// Returns:
//   - []string: Index names (empty slice if none)
//   - error: Any error encountered
func (c *Client) ListIndexes(ctx context.Context) ([]string, error) {
	return c.internal.ListIndexes(ctx)
}

// CreateIndex creates a new encrypted DiskIVF vector index.
//
// The new index is empty and ready for vector operations. DiskIVF performs
// two-stage encrypted ANN search: fast ranking over PQ codes followed by
// reranking with stored full-precision vectors.
//
// Parameters:
//   - ctx: Context for cancellation/timeouts
//   - params: Complete payload containing:
//   - IndexName (required): unique index name
//   - IndexKey  (required): 32-byte encryption key
//   - Dimension (optional): vector dimensionality; auto-detected on first upsert if omitted
//   - Metric (optional): distance metric (e.g., "euclidean", "cosine")
//   - EmbeddingModel (optional): embedding model name to associate
//
// Returns:
//   - *EncryptedIndex: Handle for vector operations
//   - error: Any error encountered
//
// Note: Store the encryption key securely; it cannot be recovered if lost.
// Creating with an existing name will fail.
func (c *Client) CreateIndex(
	ctx context.Context,
	params *CreateIndexParams,
) (*EncryptedIndex, error) {
	// Validate the key length
	if len(params.IndexKey) != KeySize {
		return nil, fmt.Errorf("%w, got %d", ErrInvalidKeyLength, len(params.IndexKey))
	}

	// Convert bytes to hex string
	keyHex := fmt.Sprintf("%x", params.IndexKey)

	req := internal.CreateIndexRequest{
		IndexName: params.IndexName,
		IndexKey:  keyHex,
	}

	if params.Dimension != nil {
		req.Dimension = *internal.NewNullableInt32(params.Dimension)
	}

	if params.Metric != nil {
		req.Metric = *internal.NewNullableString(params.Metric)
	}

	if params.EmbeddingModel != nil {
		req.EmbeddingModel = *internal.NewNullableString(params.EmbeddingModel)
	}

	// Call internal CreateIndex
	_, _, err := c.internal.APIClient.DefaultAPI.CreateIndexV1IndexesCreatePost(ctx).
		CreateIndexRequest(req).
		Execute()
	if err != nil {
		return nil, err
	}

	return &EncryptedIndex{
		indexName: params.IndexName,
		indexKey:  keyHex,
		indexType: "disk_ivf",
		client:    c.internal,
		trained:   false,
	}, nil
}

// LoadIndex loads an existing encrypted index by name and key.
//
// The provided key must match the one used at creation time. Configuration and
// metadata are fetched from the server.
//
// Parameters:
//   - ctx: Context for cancellation/timeouts
//   - indexName: Existing index name
//   - indexKey: 32-byte encryption key
//
// Returns:
//   - *EncryptedIndex: Handle for vector operations
//   - error: Any error encountered
func (c *Client) LoadIndex(ctx context.Context, indexName string, indexKey []byte) (*EncryptedIndex, error) {
	// Validate the key length
	if len(indexKey) != KeySize {
		return nil, fmt.Errorf("%w, got %d", ErrInvalidKeyLength, len(indexKey))
	}

	keyHex := fmt.Sprintf("%x", indexKey)

	describeReq := internal.IndexOperationRequest{
		IndexName: indexName,
		IndexKey:  keyHex,
	}

	indexInfo, _, err := c.internal.APIClient.DefaultAPI.GetIndexInfoV1IndexesDescribePost(ctx).
		IndexOperationRequest(describeReq).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get index info: %w", err)
	}

	return &EncryptedIndex{
		indexName: indexInfo.IndexName,
		indexKey:  keyHex,
		indexType: indexInfo.IndexType,
		client:    c.internal,
		trained:   indexInfo.IsTrained,
	}, nil
}

// GetHealth checks the health status of the CyborgDB service.
//
// Useful for readiness/liveness checks and connectivity diagnostics.
//
// Parameters:
//   - ctx: Context for cancellation/timeouts
//
// Returns:
//   - map[string]string: Health status from the server
//   - error: Any error encountered
func (c *Client) GetHealth(ctx context.Context) (map[string]string, error) {
	return c.internal.GetHealth(ctx)
}
