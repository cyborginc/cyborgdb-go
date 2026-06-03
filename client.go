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
	// ErrMissingKeyOrKMS is returned when CreateIndex is called with neither
	// IndexKey nor KmsName set.
	ErrMissingKeyOrKMS = fmt.Errorf("create_index requires IndexKey, KmsName, or both")
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

// keyBytesToHex validates an optional 32-byte index key and returns its hex
// encoding. Returns (nil, nil) for an empty key — callers decide whether
// that's acceptable for the specific operation (it is for KMS-backed
// indexes; it isn't for legacy keyed indexes).
func keyBytesToHex(key []byte) (*string, error) {
	if len(key) == 0 {
		return nil, nil
	}
	if len(key) != KeySize {
		return nil, fmt.Errorf("%w, got %d", ErrInvalidKeyLength, len(key))
	}
	h := fmt.Sprintf("%x", key)
	return &h, nil
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
// At least one of params.IndexKey or params.KmsName must be supplied
// (ErrMissingKeyOrKMS otherwise). The service accepts exactly one of them:
//
//   - IndexKey only — the SDK supplies the 32-byte wrapping key; the service
//     records the index as provider="none" and performs no KMS round-trip. The
//     same key must be re-supplied to LoadIndex.
//   - KmsName only — the service generates the key and wraps it under the named
//     kms.registry entry ("aws-kms" or "aws"); the SDK never sees the plaintext
//     key, and LoadIndex needs no key.
//
// Supplying both is forwarded as-is and rejected by the service with a 400, for
// every provider: the named slot already determines the key source, so an
// SDK-supplied key is contradictory. Note that "none" is not a registry slot
// type — the no-KMS path is reached by omitting KmsName, not by naming a
// "none" slot.
//
// Returns:
//   - *EncryptedIndex: Handle for vector operations
//   - error: Any error encountered (ErrMissingKeyOrKMS if neither is set;
//     ErrInvalidKeyLength if IndexKey is set but not 32 bytes)
//
// Note: Store the encryption key securely; it cannot be recovered if lost.
// Creating with an existing name will fail.
func (c *Client) CreateIndex(
	ctx context.Context,
	params *CreateIndexParams,
) (*EncryptedIndex, error) {
	if len(params.IndexKey) == 0 && params.KmsName == nil {
		return nil, ErrMissingKeyOrKMS
	}

	keyHex, err := keyBytesToHex(params.IndexKey)
	if err != nil {
		return nil, err
	}

	req := internal.CreateIndexRequest{
		IndexName: params.IndexName,
	}

	if keyHex != nil {
		req.IndexKey = *internal.NewNullableString(keyHex)
	}

	if params.KmsName != nil {
		req.KmsName = *internal.NewNullableString(params.KmsName)
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

	if params.StoragePrecision != nil {
		req.StoragePrecision = *internal.NewNullableString(params.StoragePrecision)
	}

	_, _, err = c.internal.APIClient.DefaultAPI.CreateIndexV1IndexesCreatePost(ctx).
		CreateIndexRequest(req).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}

	return &EncryptedIndex{
		indexName: params.IndexName,
		indexKey:  keyHex,
		client:    c.internal,
	}, nil
}

// LoadIndex loads an existing encrypted index by name.
//
// indexKey is required for legacy / "provider: none" indexes where the SDK
// owns the KEK. For KMS-backed indexes the service resolves the DEK via the
// stored KMSBlob, so indexKey may be nil (or zero-length).
//
// Parameters:
//   - ctx: Context for cancellation/timeouts
//   - indexName: Existing index name
//   - indexKey: 32-byte encryption key, or nil for KMS-backed indexes
//
// Returns:
//   - *EncryptedIndex: Handle for vector operations
//   - error: Any error encountered
func (c *Client) LoadIndex(ctx context.Context, indexName string, indexKey []byte) (*EncryptedIndex, error) {
	keyHex, err := keyBytesToHex(indexKey)
	if err != nil {
		return nil, err
	}

	var key internal.NullableString
	if keyHex != nil {
		key = *internal.NewNullableString(keyHex)
	}

	indexInfo, err := describeIndex(ctx, c.internal, indexName, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get index info: %w", err)
	}

	return &EncryptedIndex{
		indexName: indexInfo.IndexName,
		indexKey:  keyHex,
		client:    c.internal,
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
