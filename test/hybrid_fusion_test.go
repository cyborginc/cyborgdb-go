package test

import (
	"context"
	"reflect"
	"testing"
	"time"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
	"github.com/cyborginc/cyborgdb-go/internal"
)

// Hybrid fusion with hand-chosen vectors, so the fused ranking is a fact
// rather than noise. Core proves the fusion maths; these prove the wiring.
//
// Mirrors py tests/test_bm25.py TestHybridFusionDeterministic, itself ported
// from cyborgdb-core tests/bm25_api_test.py.

const hybridDim = 4

// hybridDoc is (id, vector, title, body, author).
type hybridDoc struct {
	id     string
	vector []float32
	title  string
	body   string
	author string
}

// Distances to hybridQueryVector are strictly ordered — d2 (0.0125) <
// d1 (1.8125) < d0 (1.9125) < d3 (2.0125) — so no assertion rests on a
// tie-break. The euclidean metric and dimension 4 are load-bearing: change
// either and the ordering stops holding.
var hybridDocs = []hybridDoc{
	{"d0", []float32{1.0, 0.0, 0.0, 0.0}, "apple banana", "date date elder", "ann"},
	{"d1", []float32{0.0, 1.0, 0.0, 0.0}, "banana", "date", "bob"},
	{"d2", []float32{0.0, 0.0, 1.0, 0.0}, "cherry", "elder", "ann"},
	{"d3", []float32{0.0, 0.0, 0.0, 1.0}, "apple", "fig", "bob"},
}

var hybridQueryVector = []float32{0.05, 0.1, 1.0, 0.0}

const hybridText = "apple date"

// hybridIndex creates the fusion fixture: two full-text fields and one
// filterable field.
func hybridIndex(t *testing.T) *cyborgdb.EncryptedIndex {
	t.Helper()
	client := newIsolatedClient(t)
	dim := int32(hybridDim)
	metric := "euclidean"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	index, err := client.CreateIndex(ctx, &cyborgdb.CreateIndexParams{
		IndexName: generateUniqueName("hybrid_fusion_"),
		IndexKey:  generateRandomKey(),
		Dimension: &dim,
		Metric:    &metric,
		// filterable spelled out because of cyborgdb-core#2393.
		MetadataSchema: map[string]cyborgdb.MetadataFieldPolicy{
			"title":  {FullText: boolPtr(true), Filterable: boolPtr(false)},
			"body":   {FullText: boolPtr(true), Filterable: boolPtr(false)},
			"author": {Filterable: boolPtr(true)},
		},
	})
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanCancel()
		_ = index.DeleteIndex(cleanCtx)
	})

	items := make(cyborgdb.VectorItems, len(hybridDocs))
	ids := make([]string, len(hybridDocs))
	for i, doc := range hybridDocs {
		ids[i] = doc.id
		items[i] = cyborgdb.VectorItem{
			Id:     doc.id,
			Vector: doc.vector,
			Metadata: map[string]interface{}{
				"title": doc.title, "body": doc.body, "author": doc.author,
			},
		}
	}
	if err := index.Upsert(ctx, items); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	waitForIDs(t, index, ids)
	return index
}

// hybridQuery runs the fusion query, applying opts on top of the defaults
// (the fixture vector, hybridText, topK 4).
func hybridQuery(t *testing.T, index *cyborgdb.EncryptedIndex, opts func(*cyborgdb.QueryParams)) []internal.QueryResultItem {
	t.Helper()
	resp, err := hybridQueryRaw(index, opts)
	if err != nil {
		t.Fatalf("hybrid Query failed: %v", err)
	}
	return getQueryResultItems(&resp.Results)
}

// hybridQueryRaw is hybridQuery without the fatal, for the rejection case.
func hybridQueryRaw(index *cyborgdb.EncryptedIndex, opts func(*cyborgdb.QueryParams)) (*cyborgdb.QueryResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	text := hybridText
	params := cyborgdb.QueryParams{
		QueryVector: hybridQueryVector,
		Text:        &text,
		TopK:        4,
	}
	if opts != nil {
		opts(&params)
	}
	return index.Query(ctx, params)
}

// hybridIDs is the fused ranking as ids.
func hybridIDs(t *testing.T, index *cyborgdb.EncryptedIndex, opts func(*cyborgdb.QueryParams)) []string {
	t.Helper()
	items := hybridQuery(t, index, opts)
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.Id
	}
	return out
}

// hybridScores maps id to fused score.
func hybridScores(t *testing.T, index *cyborgdb.EncryptedIndex, opts func(*cyborgdb.QueryParams)) map[string]float32 {
	t.Helper()
	out := map[string]float32{}
	for _, item := range hybridQuery(t, index, opts) {
		out[item.Id] = item.GetScore()
	}
	return out
}

// textOnlyIDs is the pure BM25 leg.
func textOnlyIDs(t *testing.T, index *cyborgdb.EncryptedIndex) []string {
	t.Helper()
	text := hybridText
	return metaIDs(queryMetaRows(t, index, cyborgdb.QueryMetadataParams{Text: &text, TopK: 4}))
}

// vectorOnlyIDs is the pure vector leg.
func vectorOnlyIDs(t *testing.T, index *cyborgdb.EncryptedIndex) []string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := index.Query(ctx, cyborgdb.QueryParams{
		QueryVector: hybridQueryVector,
		TopK:        4,
	})
	if err != nil {
		t.Fatalf("vector-only Query failed: %v", err)
	}
	items := getQueryResultItems(&resp.Results)
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.Id
	}
	return out
}

func TestHybridAlphaZeroReproducesPureBM25(t *testing.T) {
	// Anchored as well as compared: agreement alone would hold if both legs
	// returned the same wrong answer. "apple date" matches d0 on both terms,
	// d1 and d3 on one each, and d2 on neither.
	index := hybridIndex(t)
	got := hybridIDs(t, index, func(p *cyborgdb.QueryParams) { p.Alpha = f64Ptr(0.0) })
	if want := textOnlyIDs(t, index); !reflect.DeepEqual(got, want) {
		t.Errorf("alpha=0 ranking %v does not match the pure BM25 ranking %v", got, want)
	}
	assertSameIDs(t, got, []string{"d0", "d1", "d3"}, "alpha=0 match set")
}

func TestHybridAlphaOneReproducesPureVector(t *testing.T) {
	// Distances are strictly ordered, so the expected order is fixed rather
	// than merely consistent.
	index := hybridIndex(t)
	got := hybridIDs(t, index, func(p *cyborgdb.QueryParams) { p.Alpha = f64Ptr(1.0) })
	if want := vectorOnlyIDs(t, index); !reflect.DeepEqual(got, want) {
		t.Errorf("alpha=1 ranking %v does not match the pure vector ranking %v", got, want)
	}
	if want := []string{"d2", "d1", "d0", "d3"}; !reflect.DeepEqual(got, want) {
		t.Errorf("alpha=1 ranking %v, want %v", got, want)
	}
}

func TestHybridAlphaEndpointsDisagree(t *testing.T) {
	// Without this, both tests above would pass vacuously if the two rankings
	// ever coincided.
	index := hybridIndex(t)
	text, vector := textOnlyIDs(t, index), vectorOnlyIDs(t, index)
	if reflect.DeepEqual(text, vector) {
		t.Errorf("the two legs agree (%v); the alpha endpoint tests would be vacuous", text)
	}
}

func TestHybridFusionPromotesADocumentNeitherLegRankedFirst(t *testing.T) {
	// Vector order d2, d1, d0, d3; text order d0, then d1/d3. At the defaults
	// (alpha 0.5, rrf_k 60) d0 wins on agreement across both legs
	// (0.5/61 + 0.5/63) ahead of d1 (0.5/62 + 0.5/62), while d2 — rank 1 on
	// vectors, absent from text — falls to last on 0.5/61 alone. Either
	// ordering of the d1/d3 text tie fuses the same way.
	index := hybridIndex(t)
	got := hybridIDs(t, index, nil)
	if want := []string{"d0", "d1", "d3", "d2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("fused ranking %v, want %v", got, want)
	}
}

func TestHybridRrfKReachesTheFusion(t *testing.T) {
	// RRF contributes 1/(k + rank) per leg, so a smaller k raises every score.
	// Asserted on scores, not order: on four documents the order margins are
	// under 1% and would flake.
	index := hybridIndex(t)
	small := hybridScores(t, index, func(p *cyborgdb.QueryParams) { p.RrfK = f64Ptr(1.0) })
	large := hybridScores(t, index, func(p *cyborgdb.QueryParams) { p.RrfK = f64Ptr(60.0) })

	if len(small) != len(large) {
		t.Fatalf("rrf_k must not change matches: small %v, large %v", small, large)
	}
	identical := true
	for id, smallScore := range small {
		largeScore, ok := large[id]
		if !ok {
			t.Fatalf("rrf_k changed the match set: %s missing at rrf_k=60", id)
		}
		if smallScore != largeScore {
			identical = false
		}
		if smallScore <= largeScore {
			t.Errorf("%s: smaller rrf_k must raise the fused score; got %v at k=1 vs %v at k=60",
				id, smallScore, largeScore)
		}
	}
	if identical {
		t.Error("rrf_k had no effect on any score; it is not reaching the fusion")
	}
}

func TestHybridWindowMultBelowOneIsRejected(t *testing.T) {
	// Only the bound is assertable: search is exhaustive on an untrained
	// index, so the candidate window cannot affect the ranking.
	index := hybridIndex(t)
	zero := int32(0)
	if _, err := hybridQueryRaw(index, func(p *cyborgdb.QueryParams) { p.WindowMult = &zero }); err == nil {
		t.Error("window_mult=0 should be rejected")
	}
}

func TestHybridIncludeMetadataReturnsTheFusedWinner(t *testing.T) {
	index := hybridIndex(t)
	items := hybridQuery(t, index, func(p *cyborgdb.QueryParams) {
		p.Include = []string{"metadata"}
		p.TopK = 2
	})
	if len(items) != 2 {
		t.Fatalf("got %d rows, want 2", len(items))
	}
	for _, item := range items {
		if item.Metadata == nil {
			t.Fatalf("%s carries no metadata", item.Id)
		}
		if _, ok := item.Metadata["title"]; !ok {
			t.Errorf("%s metadata is missing title: %v", item.Id, item.Metadata)
		}
	}
	if items[0].Id != "d0" {
		t.Errorf("fused winner %s, want d0", items[0].Id)
	}
	if got := items[0].Metadata["author"]; got != "ann" {
		t.Errorf("d0 author %v, want ann", got)
	}
}

func TestHybridBatchFusesEachRowIndependently(t *testing.T) {
	// Two identical vectors must fuse identically and match the single-vector
	// result: the text leg applies per row, not per batch.
	index := hybridIndex(t)
	expected := hybridIDs(t, index, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	text := hybridText
	resp, err := index.Query(ctx, cyborgdb.QueryParams{
		BatchQueryVectors: [][]float32{hybridQueryVector, hybridQueryVector},
		Text:              &text,
		TopK:              4,
	})
	if err != nil {
		t.Fatalf("batch hybrid Query failed: %v", err)
	}
	rows := getBatchQueryResults(&resp.Results)
	if len(rows) != 2 {
		t.Fatalf("got %d result sets, want 2", len(rows))
	}
	for i, row := range rows {
		got := make([]string, len(row))
		for j, item := range row {
			got[j] = item.Id
		}
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("batch row %d fused to %v, want %v", i, got, expected)
		}
	}
}

func TestHybridFilterPrefiltersBothLegs(t *testing.T) {
	// author=ann keeps d0 and d2; the order must be the fused order restricted
	// to them, not an arbitrary subset.
	index := hybridIndex(t)
	got := hybridIDs(t, index, func(p *cyborgdb.QueryParams) {
		p.Filters = map[string]interface{}{"author": "ann"}
	})
	if want := []string{"d0", "d2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("filtered fusion %v, want %v", got, want)
	}
}

func TestHybridRepeatedQueriesAreIdentical(t *testing.T) {
	// Precondition for every order assertion above.
	index := hybridIndex(t)
	first := hybridQuery(t, index, nil)
	second := hybridQuery(t, index, nil)
	if len(first) != len(second) {
		t.Fatalf("repeated queries returned %d and %d rows", len(first), len(second))
	}
	for i := range first {
		if first[i].Id != second[i].Id {
			t.Errorf("row %d: ids %s then %s", i, first[i].Id, second[i].Id)
		}
		if first[i].GetScore() != second[i].GetScore() {
			t.Errorf("row %d (%s): scores %v then %v", i, first[i].Id, first[i].GetScore(), second[i].GetScore())
		}
	}
}
