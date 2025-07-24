package cyborgdb

import (
	"context"
	"fmt"
)

// Upsert adds or updates vectors in the index
func (e *EncryptedIndex) Upsert(ctx context.Context, items []VectorItem) error {
    if e.client == nil {
        return fmt.Errorf("cannot upsert vectors: client reference is nil")
    }
    if e.IndexName == nil || *e.IndexName == "" {
        return fmt.Errorf("index name is required")
    }
    if e.IndexKey == "" {
        return fmt.Errorf("index key is required")
    }

	

    // Map user-facing VectorItems to API VectorItems
    apiItems := make([]VectorItem, len(items))
    for i, item := range items {
        apiItem := VectorItem{
            Id:       item.Id,
            Vector:   item.Vector,
            Metadata: item.Metadata,
        }
        if item.Contents != nil {
            apiItem.Contents = item.Contents
        }
        apiItems[i] = apiItem
    }

    // Construct request with ALL required fields
    upsertRequest := UpsertRequest{
        IndexKey:  e.IndexKey,
        IndexName: *e.IndexName,
        Items:     apiItems,
    }

    // Call without XIndexKey header since it's now in the body
    _, err := e.client.apiClient.DefaultAPI.
        UpsertVectors(ctx, *e.IndexName).
		XIndexKey(e.IndexKey).
        UpsertRequest(upsertRequest).
        Execute()

    if err != nil {
        return fmt.Errorf("failed to upsert vectors: %w", err)
    }
    return nil
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

// Train trains the encrypted index using the specified parameters.
//
// It performs a training pass over existing data to enable efficient querying.
// Parameters are optional and default to commonly used values:
//   - batchSize: 2048
//   - maxIters: 100
//   - tolerance: 1e-6
func (e *EncryptedIndex) Train(ctx context.Context, batchSize int32, maxIters int32, tolerance float64) error {
	if e.client == nil {
		return fmt.Errorf("cannot train index: client reference is nil")
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

	// Prepare the train request
	trainReq := TrainRequest{
		IndexName: *e.IndexName,
		IndexKey:  e.IndexKey,
		BatchSize: &batchSize,
		MaxIters:  &maxIters,
		Tolerance: &tolerance,
	}

	// Call the low-level API
	_, err := e.client.apiClient.DefaultAPI.
		TrainIndex(ctx, *e.IndexName).
		XIndexKey(e.IndexKey).
		TrainRequest(trainReq).
		Execute()

	if err != nil {
		return fmt.Errorf("failed to train index '%s': %w", *e.IndexName, err)
	}

	return nil
}