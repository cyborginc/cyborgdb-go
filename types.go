package cyborgdb

import (
	"github.com/cyborginc/cyborgdb-go/internal"
)

// type Client = internal.Client

type CreateIndexRequest = internal.CreateIndexRequest
type IndexOperationRequest = internal.IndexOperationRequest
type QueryRequest = internal.QueryRequest
type TrainRequest = internal.TrainRequest
type GetResponse = internal.GetResponse
type VectorItem = internal.VectorItem
type QueryResponse = internal.QueryResponse
type IndexIVFFlat = internal.IndexIVFFlatModel
type IndexIVF = internal.IndexIVFModel
type IndexIVFPQ = internal.IndexIVFPQModel
type QueryResultItem = internal.QueryResult
