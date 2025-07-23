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
	apiItems := make([]APIVectorItem, len(items))
	
	for i, item := range items {
		// Create API VectorItem with proper field mapping
		apiItem:= APIVectorItem{
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
		IndexName: *e.IndexName,
		IndexKey:  e.IndexKey,
		Items:     []APIVectorItem(apiItems),
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