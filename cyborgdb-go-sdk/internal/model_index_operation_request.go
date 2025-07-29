package internal

// IndexOperationRequest represents a request to perform an operation
// (such as describe or delete) on an existing encrypted index.
//
// This struct maps to the JSON payload expected by the CyborgDB API.
// Fields use snake_case JSON tags to match OpenAPI spec.
//
// Fields:
//   - IndexKey: A 32-byte encryption key encoded as a hex string.
//   - IndexName: The name or identifier of the index to operate on.
type IndexOperationRequest struct {
	IndexKey  string `json:"index_key"`  // Hex-encoded 32-byte encryption key
	IndexName string `json:"index_name"` // Index name or identifier
}
