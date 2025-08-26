package cyborgdb

import (
	"github.com/cyborginc/cyborgdb-go/internal"
)

// type Client = internal.Client

type CreateIndexRequest = internal.CreateIndexRequest
type IndexOperationRequest = internal.IndexOperationRequest
type UpsertRequest = internal.UpsertRequest
type QueryRequest = internal.QueryRequest
type BatchQueryRequest = internal.BatchQueryRequest
type TrainRequest = internal.TrainRequest
type DeleteRequest = internal.DeleteRequest
type GetRequest = internal.GetRequest
type GetResponse = internal.GetResponse
type VectorItem = internal.VectorItem
type QueryResponse = internal.QueryResponse
type IndexIVFFlat = internal.IndexIVFFlatModel
type IndexIVF = internal.IndexIVFModel
type IndexIVFPQ = internal.IndexIVFPQModel
type QueryResultItem = internal.QueryResult

// QueryOptions provides a cleaner interface for query parameters that matches the Python SDK.
type QueryOptions struct {
	// QueryVectors can be a single vector ([]float32) or multiple vectors ([][]float32)
	QueryVectors interface{}
	// QueryContents is text content to be embedded and searched (requires embedding model)
	QueryContents string
	// TopK is the number of nearest neighbors to return (default: 100)
	TopK int32
	// NProbes is the number of clusters to search (default: 1)
	NProbes int32
	// Filters are metadata filters for narrowing results
	Filters map[string]interface{}
	// Include specifies which fields to include in response (default: ["distance", "metadata"])
	Include []string
	// Greedy enables greedy search for potentially faster results (default: false)
	Greedy bool
}
