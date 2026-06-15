package test

import (
	"context"
	"testing"
	"time"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
)

// TestQueryWithRerankMult exercises the rerank_mult query knob added in the
// 0.17.0 API. rerank_mult is the stage-1 retrieval multiplier for reranking
// indexes; it is optional and the server applies a default when unset. This
// verifies the SDK threads the value into the request and the server accepts
// it on a standard query.
func TestQueryWithRerankMult(t *testing.T) {
	const dim = 8

	client := newIsolatedClient(t)
	index, _ := newIsolatedIndex(t, client, "rerank", int32(dim))
	seedIndex(t, index, "rerank", 20, dim)
	waitForPropagation(2 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rerankMult := int32(4)
	resp, err := index.Query(ctx, cyborgdb.QueryParams{
		QueryVector: generateRandomVectors(1, dim)[0],
		TopK:        5,
		RerankMult:  &rerankMult,
		Include:     []string{"distance"},
	})
	if err != nil {
		t.Fatalf("Query with RerankMult failed: %v", err)
	}
	if len(getQueryResultItems(&resp.Results)) < 1 {
		t.Error("expected at least one result from rerank_mult query")
	}
}
