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
		apiItem := VectorItem{
			Id: item.Id, // Map ID to Id
		}
		
		// Convert float64 to float32 for vector
		if item.Vector != nil && len(item.Vector) > 0 {
			apiItem.Vector = convertFloat64ToFloat32(item.Vector)
		}
		
		// Handle contents - it's already *string in both types
		if item.Contents != nil {
			// Since Contents is *string in both types, we can encode it directly
			encoded := base64.StdEncoding.EncodeToString([]byte(*item.Contents))
			apiItem.Contents = &encoded
		}
		
		// Metadata is the same type in both
		if item.Metadata != nil {
			apiItem.Metadata = item.Metadata
		}
		
		apiItems[i] = apiItem
	}

	// Create the upsert request
	upsertRequest := UpsertRequest{
		IndexName: *e.IndexName,
		IndexKey:  e.IndexKey,
		Items:     apiItems,
	}

	_, _, err := e.client.apiClient.DefaultAPI.UpsertVectors(ctx).
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