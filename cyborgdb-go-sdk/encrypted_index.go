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