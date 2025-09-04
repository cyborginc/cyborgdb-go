package cyborgdb

import (
	"github.com/cyborginc/cyborgdb-go/internal"
)

// type Client = internal.Client

type GetResponse = internal.GetResponseModel
type VectorItem = internal.VectorItem
type QueryResponse = internal.QueryResponse
type QueryResultItem = internal.QueryResultItem
type CreateIndexRequest = internal.CreateIndexRequest
type ListIDsResponse = internal.ListIDsResponse

type IndexModel interface {
    ToIndexConfig() *internal.IndexConfig
}

// CreateIndexParams is the public-facing type for creating an index.
// Similar to CreateIndexRequest but accepts IndexModel types for IndexConfig.
type CreateIndexParams struct {
	// Unique index name
	IndexName string `json:"index_name"`
	// 64-char hex string of 32-byte encryption key
	IndexKey string `json:"index_key"`
	// Index configuration - can be IndexIVF, IndexIVFFlat, or IndexIVFPQ
	IndexConfig IndexModel `json:"index_config,omitempty"`
	// Distance metric (e.g., "euclidean", "cosine")
	Metric *string `json:"metric,omitempty"`
	// Embedding model name to associate
	EmbeddingModel *string `json:"embedding_model,omitempty"`
}

type TrainParams struct {
	BatchSize *int32   `json:"batch_size,omitempty"`  // Optional, default: 2048
	MaxIters  *int32   `json:"max_iters,omitempty"`   // Optional, default: 100
	Tolerance *float64 `json:"tolerance,omitempty"`   // Optional, default: 1e-6
	MaxMemory *int32   `json:"max_memory,omitempty"`  // Optional, default: 0
	NLists    *int32   `json:"n_lists,omitempty"`     // Optional: number of IVF clusters
}

type QueryParams struct {
	QueryVector       []float32              `json:"query_vector,omitempty"`
	BatchQueryVectors [][]float32            `json:"query_vectors,omitempty"`
	QueryContents     *string                `json:"query_contents,omitempty"`
	TopK              int32                  `json:"top_k"`                    // Required
	NProbes           *int32                 `json:"n_probes,omitempty"`       // Optional
	Greedy            *bool                  `json:"greedy,omitempty"`         // Optional
	Filters           map[string]interface{} `json:"filters,omitempty"`        // Optional
	Include           []string               `json:"include"`                  // Required
}

// Index model type aliases (unexported)
type indexIVF = internal.IndexIVFModel
type indexIVFFlat = internal.IndexIVFFlatModel
type indexIVFPQ = internal.IndexIVFPQModel

// IndexIVF creates a new IVF index configuration
func IndexIVF(dimension int32) *indexIVF {
	model := &internal.IndexIVFModel{}
	model.SetDimension(dimension)
	model.SetType("ivf") // Hardcoded type
	return model
}

// IndexIVFFlat creates a new IVFFlat index configuration
func IndexIVFFlat(dimension int32) *indexIVFFlat {
	model := &internal.IndexIVFFlatModel{}
	model.SetDimension(dimension)
	model.SetType("ivfflat") // Hardcoded type
	return model
}

// IndexIVFPQ creates a new IVFPQ index configuration
func IndexIVFPQ(dimension int32, pqDim int32, pqBits int32) *indexIVFPQ {
	model := &internal.IndexIVFPQModel{
		PqDim:  pqDim,
		PqBits: pqBits,
	}
	model.SetDimension(dimension)
	model.SetType("ivfpq") // Hardcoded type
	return model
}

// ToIndexConfig converts indexIVF to IndexConfig
func (m *indexIVF) ToIndexConfig() *internal.IndexConfig {
	return &internal.IndexConfig{
		IndexIVFModel: m,
	}
}

// ToIndexConfig converts indexIVFFlat to IndexConfig
func (m *indexIVFFlat) ToIndexConfig() *internal.IndexConfig {
	return &internal.IndexConfig{
		IndexIVFFlatModel: m,
	}
}

// ToIndexConfig converts indexIVFPQ to IndexConfig
func (m *indexIVFPQ) ToIndexConfig() *internal.IndexConfig {
	return &internal.IndexConfig{
		IndexIVFPQModel: m,
	}
}
