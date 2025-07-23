package cyborgdb

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
)

// Client provides a high-level interface to CyborgDB
type Client struct {
	apiClient *APIClient
	baseURL   string
	apiKey    string
}

// NewClient creates a new CyborgDB client
func NewClient(baseURL, apiKey string, verifySSL bool) (*Client, error) {
	// Parse the baseURL to configure the APIClient
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	// Auto-detect localhost SSL bypass
	if !verifySSL && (parsedURL.Hostname() == "localhost" || parsedURL.Hostname() == "127.0.0.1") {
		fmt.Println("SSL verification is disabled for localhost (development mode)")
	}
	// Create configuration
	cfg := NewConfiguration()
	cfg.Scheme = parsedURL.Scheme
	cfg.Host = parsedURL.Host

	// Set API key in default headers if provided
	if apiKey != "" {
		cfg.AddDefaultHeader("X-API-Key", apiKey)
	}
	// Create custom HTTP client that respects verifySSL
	cfg.HTTPClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: !verifySSL},
		},
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
func (c *Client) CreateIndex(
	ctx context.Context,
	indexName string,
	indexKey []byte,
	indexModel IndexModel,
	embeddingModel *string,
) (*EncryptedIndex, error) {
	if len(indexKey) != 32 {
		return nil, fmt.Errorf("index key must be exactly 32 bytes, got %d", len(indexKey))
	}

	keyHex := hex.EncodeToString(indexKey)

	indexCfg, err := ConvertToIndexConfig(indexModel)
	if err != nil {
		return nil, fmt.Errorf("failed to convert index model: %w", err)
	}

	createReq := CreateIndexRequest{
		IndexName:      indexName,
		IndexKey:       keyHex,
		IndexConfig:    indexCfg, // this is now of type *IndexConfig
		EmbeddingModel: embeddingModel,
	}

	_, _, err = c.apiClient.DefaultAPI.CreateIndex(ctx).
		CreateIndexRequest(createReq).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}

	return &EncryptedIndex{
		IndexName: &indexName,
		IndexType: &createReq.IndexConfig.IndexType,
		Config:    &createReq.IndexConfig,
	}, nil
}

// GetHealth checks the health status of the CyborgDB service
func (c *Client) GetHealth(ctx context.Context) (*HealthResponse, error) {
	health, _, err := c.apiClient.DefaultAPI.GetHealth(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}
	return health, nil
}
