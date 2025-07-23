package cyborgdb

import (
	"context"
	"encoding/base64"
	"fmt"
)
// Upsert adds or updates vectors in the index
func (ei *EncryptedIndex) Upsert(ctx context.Context, items []VectorItem) (map[string]interface{}, error) {

	// Convert VectorItem slice to match API expectations
	vectors := make([]VectorItem, len(items))
	for i, item := range items {
		vector := VectorItem{
			Id:       item.Id,
			Vector:   item.Vector,
			Metadata: item.Metadata,
		}

		// Handle contents encoding
		if item.Contents != nil {
			// If contents is already a string, use it directly
			// Otherwise, encode as base64
			if strContent, ok := (*item.Contents).(string); ok {
				vector.Contents = &strContent
			} else if byteContent, ok := (*item.Contents).([]byte); ok {
				encoded := base64.StdEncoding.EncodeToString(byteContent)
				vector.Contents = &encoded
			} else {
				// Convert to string if possible
				str := fmt.Sprintf("%v", *item.Contents)
				vector.Contents = &str
			}
		}

		vectors[i] = vector
	}

	upsertRequest := UpsertRequest{
		IndexName: ei.IndexName,
		IndexKey:  ei.IndexKey,
		Items:     vectors,
	}

	resp, _, err := ei.apiClient.DefaultAPI.UpsertVectors(ctx).
		UpsertRequest(upsertRequest).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to upsert vectors: %w", err)
	}

	result := map[string]interface{}{
		"status": resp.Status,
	}
	if resp.UpsertedCount != nil {
		result["upserted_count"] = *resp.UpsertedCount
	}

	return result, nil
}