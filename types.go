package cyborgdb

import (
	"github.com/cyborginc/cyborgdb-go/internal"
)

// type Client = internal.Client

type GetResponse = internal.GetResponse
type VectorItem = internal.VectorItem
type QueryResponse = internal.QueryResponse
type QueryResultItem = internal.QueryResult
type CreateIndexRequest = internal.CreateIndexRequest

type IndexModel interface {
    GetType() string
    GetDimension() int
    ToIndexConfig() *internal.IndexConfig
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

var (
	IndexIVF     = internal.IndexIVF
	IndexIVFFlat = internal.IndexIVFFlat  
	IndexIVFPQ   = internal.IndexIVFPQ
)
