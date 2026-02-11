// client_wrapper.go
// Custom wrapper for the generated APIClient - DO NOT DELETE

package internal

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
)

// Client wraps the generated APIClient with convenience methods
type Client struct {
	APIClient *APIClient
	baseURL   string
	apiKey    string
}

// NewClient creates a new internal client wrapper
func NewClient(baseURL, apiKey string, verifySSL bool) (*Client, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	cfg := NewConfiguration()
	cfg.Scheme = parsedURL.Scheme
	cfg.Host = parsedURL.Host

	cfg.Servers = []ServerConfiguration{
		{
			URL:         fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host),
			Description: "CyborgDB API",
		},
	}

	if apiKey != "" {
		cfg.AddDefaultHeader("X-API-Key", apiKey)
	}

	cfg.HTTPClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: !verifySSL},
		},
	}

	apiClient := NewAPIClient(cfg)
	return &Client{
		APIClient: apiClient,
		baseURL:   baseURL,
		apiKey:    apiKey,
	}, nil
}

// ListIndexes returns all encrypted index names
func (c *Client) ListIndexes(ctx context.Context) ([]string, error) {
	resp, _, err := c.APIClient.DefaultAPI.ListIndexesV1IndexesListGet(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}
	return resp.Indexes, nil
}

// GetHealth checks the health status of the service
func (c *Client) GetHealth(ctx context.Context) (map[string]string, error) {
	health, _, err := c.APIClient.DefaultAPI.HealthCheckV1HealthGet(ctx).Execute()
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}
	return health, nil
}
