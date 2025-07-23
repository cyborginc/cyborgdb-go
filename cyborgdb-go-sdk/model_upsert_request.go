package cyborgdb

type UpsertRequest struct {
    IndexName string          `json:"index_name"`
    IndexKey  string          `json:"index_key"`
    Items     []APIVectorItem `json:"items"`  // <-- Find this type
}

type APIVectorItem struct {
    Id       string                 `json:"id"`       // Note: Id not ID
    Vector   []float32              `json:"vector"`   // Note: float32
    Contents *string                `json:"contents"`
    Metadata map[string]interface{} `json:"metadata"`
}