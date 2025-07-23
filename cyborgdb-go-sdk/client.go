package cyborgdb

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/url"
)

// Client provides a high-level interface to CyborgDB
type Client struct {
	apiClient *APIClient
	baseURL   string
	apiKey    string
}

// NewClient creates a new CyborgDB client
func NewClient(baseURL, apiKey string) (*Client, error) {
	// Parse the baseURL to configure the APIClient
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	// Create configuration
	cfg := NewConfiguration()
	cfg.Scheme = parsedURL.Scheme
	cfg.Host = parsedURL.Host
	
	// Set API key in default headers if provided
	if apiKey != "" {
		cfg.AddDefaultHeader("X-API-Key", apiKey)
	}

	// Create the low-level API client
	apiClient := NewAPIClient(cfg)

	return &Client{
		apiClient: apiClient,
		baseURL:   baseURL,
		apiKey:    apiKey,
	}, nil
}

// GenerateKey is available from utils.go (OpenAPI generated)

// ListIndexes retrieves a list of all available encrypted indexes
func (c *Client) ListIndexes(ctx context.Context) ([]string, error) {
	indexes, _, err := c.apiClient.DefaultAPI.ListIndexes(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}
	return indexes, nil
}

// CreateIndex creates a new encrypted vector index with the specified configuration
func (c *Client) CreateIndex(ctx context.Context, indexName string, indexKey []byte, config IndexConfig, embeddingModel *string) (*IndexWrapper, error) {
	// Validate index key
	if len(indexKey) != 32 {
		return nil, fmt.Errorf("index key must be exactly 32 bytes, got %d", len(indexKey))
	}

	// Convert key to hex string for API
	keyHex := hex.EncodeToString(indexKey)

	// Create the request - using correct field names from generated code
	createReq := CreateIndexRequest{
		IndexName:      PtrString(indexName),
		IndexKey:       PtrString(keyHex),
		IndexConfig:    &config,
		EmbeddingModel: embeddingModel,
	}

	// Call the API
	_, _, err := c.apiClient.DefaultAPI.CreateIndex(ctx).CreateIndexRequest(createReq).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}

	// Return an IndexWrapper instance
	return &IndexWrapper{
		client:    c,
		indexName: indexName,
		indexKey:  indexKey,
		config:    config,
	}, nil
}

// LoadIndex creates an IndexWrapper instance for an existing index
func (c *Client) LoadIndex(indexName string, indexKey []byte) *IndexWrapper {
	return &IndexWrapper{
		client:    c,
		indexName: indexName,
		indexKey:  indexKey,
		// config will be loaded lazily when needed
	}
}

// GetHealth checks the health status of the CyborgDB service
func (c *Client) GetHealth(ctx context.Context) (*HealthResponse, error) {
	health, _, err := c.apiClient.DefaultAPI.GetHealth(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}
	return health, nil
}