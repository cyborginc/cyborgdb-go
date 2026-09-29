// client_wrapper.go
// Custom wrapper for the generated APIClient - DO NOT DELETE

package internal

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
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

	// Keep any path prefix (e.g. a reverse-proxy mount at https://host/cyborgdb);
	// generated operations append "/v1/..." to this URL.
	cfg.Servers = []ServerConfiguration{
		{
			URL:         fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, strings.TrimRight(parsedURL.EscapedPath(), "/")),
			Description: "CyborgDB API",
		},
	}

	if apiKey != "" {
		cfg.AddDefaultHeader("X-API-Key", apiKey)
	}

	cfg.HTTPClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: !verifySSL},
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			MaxConnsPerHost:     100,
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

// SetContentsString sets the item's contents to a text string. Contents is an
// anyOf(bytes, string) wrapper, so this saves callers building one by hand.
// Kept here, not in the generated model file, so regeneration preserves it.
func (o *VectorItem) SetContentsString(s string) {
	o.SetContents(Contents{String: &s})
}
