package cyborgdb

import (
	"context"
	"encoding/base64"
	"fmt"
)

// Upsert adds or updates vectors in the index
func (e *EncryptedIndex) Upsert(ctx context.Context, items []VectorItem) error {

    // Convert user-facing VectorItem to API VectorItem
    // Note: The API VectorItem is the one from your generated code
    apiItems := make([]VectorItem, len(items))
    
    for i, item := range items {
        // Create API VectorItem with proper field mapping
        apiItem:= VectorItem{
            Id:       item.Id,
            Vector:   item.Vector, // No conversion needed - already []float32
            Metadata: item.Metadata,
        }
        
        if item.Contents != nil {
            encoded := base64.StdEncoding.EncodeToString([]byte(*item.Contents))
            apiItem.Contents = &encoded
        }
        
        apiItems[i] = apiItem
    }

    // Create the upsert request
    upsertRequest := UpsertRequest{
        Items:     []VectorItem(apiItems),
    }

    _, err := e.client.apiClient.DefaultAPI.
        UpsertVectors(ctx,*e.IndexName).
        XIndexKey(*e.IndexName).                // Set the encryption key header
        UpsertRequest(upsertRequest).
        Execute()
    if err != nil {
        return fmt.Errorf("failed to upsert vectors: %w", err)
    }

    return nil
}

// Helper function for type conversion
func convertFloat64ToFloat32(vec []float64) []float32 {
	if vec == nil {
		return nil
	}
	result := make([]float32, len(vec))
	for i, v := range vec {
		result[i] = float32(v)
	}
	return result
}

// DeleteIndex deletes the current encrypted index from the CyborgDB service.
//
// This method uses the stored index name and key to issue the delete request.
// Returns an error if the key is missing, malformed, or if the delete operation fails.
func (e *EncryptedIndex) DeleteIndex(ctx context.Context) error {
	if e.client == nil {
		return fmt.Errorf("cannot delete index: client reference is nil")
	}
	if e.IndexName == nil || *e.IndexName == "" {
		return fmt.Errorf("index name is required")
	}
	if e.IndexKey == "" {
		return fmt.Errorf("index key is required")
	}
	if len(e.IndexKey) != 64 {
		return fmt.Errorf("index key must be 64-character hex string (32 bytes), got %d", len(e.IndexKey))
	}

	// Call the low-level API
	_, err := e.client.apiClient.DefaultAPI.
		DeleteIndex(ctx, *e.IndexName).
		XIndexKey(e.IndexKey).
		Execute()

	if err != nil {
		return fmt.Errorf("failed to delete index '%s': %w", *e.IndexName, err)
	}

	return nil
}

