// demo.go
package cyborgdb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	// DefaultDemoEndpoint is the default endpoint for generating demo API keys.
	DefaultDemoEndpoint = "https://api.cyborgdb.co/v1/api-key/manage/create-demo-key"
	// DefaultDemoDescription is the default description for demo API keys.
	DefaultDemoDescription = "Temporary demo API key"
	// DefaultDemoTimeout is the default timeout for demo API key requests.
	DefaultDemoTimeout = 30 * time.Second
)

var (
	// ErrDemoAPIKeyNotFound is returned when the demo API key is not found in the response.
	ErrDemoAPIKeyNotFound = errors.New("demo API key not found in response")
	// ErrDemoAPIKeyGeneration is returned when demo API key generation fails.
	ErrDemoAPIKeyGeneration = errors.New("demo API key generation failed")
)

// DemoAPIKeyResponse represents the response from the demo API key endpoint.
type DemoAPIKeyResponse struct {
	APIKey    string `json:"apiKey"`
	ExpiresAt *int64 `json:"expiresAt,omitempty"`
}

// GetDemoAPIKey generates a temporary demo API key from the CyborgDB demo API service.
//
// This function generates a temporary API key that can be used for demo purposes.
// The endpoint can be configured via the CYBORGDB_DEMO_ENDPOINT environment variable.
//
// Parameters:
//   - description: Optional description for the demo API key.
//     If empty, defaults to "Temporary demo API key"
//
// Returns:
//   - string: The generated demo API key
//   - error: Any error encountered during generation
//
// Example:
//
//	demoKey, err := cyborgdb.GetDemoAPIKey("")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	client, err := cyborgdb.NewClient("https://your-instance.com", demoKey)
func GetDemoAPIKey(description string) (string, error) {
	// Use environment variable if set, otherwise use default endpoint
	endpoint := os.Getenv("CYBORGDB_DEMO_ENDPOINT")
	if endpoint == "" {
		endpoint = DefaultDemoEndpoint
	}

	// Set default description if not provided
	if description == "" {
		description = DefaultDemoDescription
	}

	// Prepare the request payload
	payload := map[string]string{
		"description": description,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request payload: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), DefaultDemoTimeout)
	defer cancel()

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Create HTTP client
	client := &http.Client{}

	// Make the POST request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to generate demo API key: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check if request was successful
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w with status %d: %s", ErrDemoAPIKeyGeneration, resp.StatusCode, string(body))
	}

	// Parse the response
	var result DemoAPIKeyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Validate the API key
	if result.APIKey == "" {
		return "", ErrDemoAPIKeyNotFound
	}

	// Log expiration info if available
	if result.ExpiresAt != nil {
		expiresAt := time.Unix(*result.ExpiresAt, 0)
		timeLeft := time.Until(expiresAt).Round(time.Second)
		fmt.Printf("Demo API key will expire in %s\n", timeLeft)
	}

	return result.APIKey, nil
}
