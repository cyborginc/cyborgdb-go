package test

import (
	"context"
	"sort"
	"testing"
	"time"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
)

// BM25 full-text search: the FullText metadata policy (via TextFields), the
// BM25 scorer config (index.BM25), and the Text leg on QueryMetadata (pure
// BM25) and Query (hybrid BM25 + vector).
//
// Mirrors py tests/test_bm25.py.
//
// BM25 is opt-in and derived: an index with at least one full-text field
// reports a BM25 config and accepts the Text legs; an index with none reports
// nil and rejects them server-side. Full-text search resolves from the metadata
// index and needs no training, so these run on small untrained indexes.

const bm25Dim = 8

// bm25Doc is (id, body, topic). body is analyzed by BM25; topic stays an
// exact-match filterable field so we can pre-filter the text leg. Docs 0/2/4
// are about quantum computing to differing degrees; 1/3/5 are unrelated noise.
type bm25Doc struct {
	id, body, topic string
}

var bm25Docs = []bm25Doc{
	{"d0", "quantum computing breakthroughs in error correction", "physics"},
	{"d1", "classical machine learning models for tabular data", "ml"},
	{"d2", "quantum entanglement and superposition explained", "physics"},
	{"d3", "cooking pasta with fresh tomatoes and basil", "food"},
	{"d4", "advances in quantum computing hardware and qubits", "physics"},
	{"d5", "financial markets and stock trading strategies", "finance"},
}

// "quantum computing" — both terms in d0/d4, only "quantum" in d2.
var bm25BothTerms = []string{"d0", "d4"}
var bm25AnyTerm = []string{"d0", "d2", "d4"}

func f64Ptr(f float64) *float64 { return &f }

// approxEqual compares two float32 values within a small tolerance.
func approxEqual(a, b float32) bool {
	const tol = 1e-4
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= tol
}

// metaIDs pulls the ids out of QueryMetadata's Results rows.
func metaIDs(rows []cyborgdb.MetadataResult) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.Id
	}
	return out
}

// isSubset reports whether every id in got is present in want.
func isSubset(got, want []string) bool {
	set := sortedSet(want)
	for _, id := range got {
		if !set[id] {
			return false
		}
	}
	return true
}

// bm25Index creates an untrained index with a full-text `body` field, a
// filterable `topic` field, and custom BM25 tuning (k1=1.5, b=0.7), then
// upserts bm25Docs. Mirrors test_bm25.py's TestBM25.setUp.
func bm25Index(t *testing.T) *cyborgdb.EncryptedIndex {
	t.Helper()
	client := newIsolatedClient(t)
	name := generateUniqueName("bm25_")
	dim := int32(bm25Dim)
	metric := "euclidean"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	index, err := client.CreateIndex(ctx, &cyborgdb.CreateIndexParams{
		IndexName:      name,
		IndexKey:       generateRandomKey(),
		Dimension:      &dim,
		Metric:         &metric,
		MetadataSchema: map[string]cyborgdb.MetadataFieldPolicy{"topic": {Filterable: boolPtr(true)}},
		TextFields:     []string{"body"},
		Bm25K1:         f64Ptr(1.5),
		Bm25B:          f64Ptr(0.7),
	})
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanCancel()
		_ = index.DeleteIndex(cleanCtx)
	})

	vectors := generateRandomVectors(len(bm25Docs), bm25Dim)
	items := make(cyborgdb.VectorItems, len(bm25Docs))
	for i, doc := range bm25Docs {
		items[i] = cyborgdb.VectorItem{
			Id:       doc.id,
			Vector:   vectors[i],
			Metadata: map[string]interface{}{"body": doc.body, "topic": doc.topic},
		}
	}
	if err := index.Upsert(ctx, items); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	waitForPropagation(2 * time.Second)
	return index
}

// queryMetaRows runs QueryMetadata and returns the Results rows, failing on error.
func queryMetaRows(t *testing.T, index *cyborgdb.EncryptedIndex, params cyborgdb.QueryMetadataParams) []cyborgdb.MetadataResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := index.QueryMetadata(ctx, params)
	if err != nil {
		t.Fatalf("QueryMetadata failed: %v", err)
	}
	return resp.Results
}

// -- schema / config round-trip ---------------------------------------- //

func TestBM25FullTextReportedInSchema(t *testing.T) {
	index := bm25Index(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema, err := index.MetadataSchema(ctx)
	if err != nil {
		t.Fatalf("MetadataSchema failed: %v", err)
	}
	body := schema["body"]
	// full_text implies filterable=false, pattern=false.
	if !body.GetFullText() {
		t.Error("body should be a full_text field")
	}
	if body.GetFilterable() {
		t.Error("full_text field should not be filterable")
	}
	if body.GetPattern() {
		t.Error("full_text field should not be a pattern field")
	}
}

func TestBM25ConfigReportsTuningParams(t *testing.T) {
	index := bm25Index(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config, err := index.BM25(ctx)
	if err != nil {
		t.Fatalf("BM25 failed: %v", err)
	}
	if config == nil {
		t.Fatal("expected a BM25 config for an index with a full_text field")
	}
	if got := config.GetK1(); !approxEqual(got, 1.5) {
		t.Errorf("k1: got %v, want 1.5", got)
	}
	if got := config.GetB(); !approxEqual(got, 0.7) {
		t.Errorf("b: got %v, want 0.7", got)
	}
	if !config.HasAnalyzerVersion() {
		t.Error("expected an analyzer_version")
	}
}

// -- QueryMetadata(Text) : pure BM25 ----------------------------------- //

func TestBM25TextSearchReturnsScoredRowsRanked(t *testing.T) {
	index := bm25Index(t)
	rows := queryMetaRows(t, index, cyborgdb.QueryMetadataParams{Text: strPtr("quantum computing")})
	if len(rows) == 0 {
		t.Fatal("expected at least one match")
	}
	// Scored rows, not bare IDs, sorted by descending score.
	scores := make([]float64, len(rows))
	for i, r := range rows {
		if !r.HasScore() {
			t.Errorf("row %s missing score", r.Id)
		}
		scores[i] = float64(r.GetScore())
	}
	if !sort.SliceIsSorted(scores, func(a, b int) bool { return scores[a] > scores[b] }) {
		t.Errorf("scores not in descending order: %v", scores)
	}
	// Every hit is a quantum doc; the top hit contains both query terms.
	if !isSubset(metaIDs(rows), bm25AnyTerm) {
		t.Errorf("got non-quantum hit: %v", metaIDs(rows))
	}
	if !sortedSet(bm25BothTerms)[rows[0].Id] {
		t.Errorf("top hit %s should contain both query terms", rows[0].Id)
	}
}

func TestBM25RequireAllTermsNarrowsToAND(t *testing.T) {
	index := bm25Index(t)
	rows := queryMetaRows(t, index, cyborgdb.QueryMetadataParams{
		Text:            strPtr("quantum computing"),
		RequireAllTerms: boolPtr(true),
	})
	assertSameIDs(t, metaIDs(rows), bm25BothTerms, "require_all_terms AND")
}

func TestBM25TextSearchTopKCapsResults(t *testing.T) {
	index := bm25Index(t)
	rows := queryMetaRows(t, index, cyborgdb.QueryMetadataParams{Text: strPtr("quantum"), TopK: 1})
	if len(rows) != 1 {
		t.Errorf("top_k=1: got %d rows, want 1", len(rows))
	}
}

func TestBM25TextFieldsRestrictsToNamedField(t *testing.T) {
	index := bm25Index(t)
	// body is the only full_text field; naming it explicitly is a no-op but
	// must be accepted.
	rows := queryMetaRows(t, index, cyborgdb.QueryMetadataParams{
		Text: strPtr("quantum"), TextFields: []string{"body"},
	})
	if !isSubset(metaIDs(rows), bm25AnyTerm) {
		t.Errorf("got %v, want subset of %v", metaIDs(rows), bm25AnyTerm)
	}
}

func TestBM25FilterPrefiltersTheTextLeg(t *testing.T) {
	index := bm25Index(t)
	// topic=food excludes every quantum doc, so the text leg scores nothing.
	rows := queryMetaRows(t, index, cyborgdb.QueryMetadataParams{
		Text: strPtr("quantum"), Filters: map[string]interface{}{"topic": "food"},
	})
	if len(rows) != 0 {
		t.Errorf("expected empty, got %v", metaIDs(rows))
	}
}

func TestBM25FilterOperatorPrefiltersTheTextLeg(t *testing.T) {
	index := bm25Index(t)
	// An operator filter ($in) must pre-filter the text leg the same way an
	// equality filter does: only physics docs survive, so only quantum docs
	// can score.
	rows := queryMetaRows(t, index, cyborgdb.QueryMetadataParams{
		Text:    strPtr("quantum"),
		Filters: map[string]interface{}{"topic": map[string]interface{}{"$in": []string{"physics"}}},
	})
	assertSameIDs(t, metaIDs(rows), bm25AnyTerm, "operator pre-filter")
	for _, r := range rows {
		if !r.HasScore() {
			t.Errorf("row %s should carry a score", r.Id)
		}
	}
}

func TestBM25RequireAllTermsWithFilterComposes(t *testing.T) {
	index := bm25Index(t)
	// AND-matching and the pre-filter apply together: require_all_terms narrows
	// to {d0, d4}, and topic=physics keeps both (they are physics).
	rows := queryMetaRows(t, index, cyborgdb.QueryMetadataParams{
		Text:            strPtr("quantum computing"),
		RequireAllTerms: boolPtr(true),
		Filters:         map[string]interface{}{"topic": "physics"},
	})
	assertSameIDs(t, metaIDs(rows), bm25BothTerms, "AND + pre-filter")
}

func TestBM25EmptyTextIsFilterOnly(t *testing.T) {
	index := bm25Index(t)
	// An empty Text keeps this a filter-only query — {id} rows with no score —
	// even though the SDK still forwards the empty string to the service.
	rows := queryMetaRows(t, index, cyborgdb.QueryMetadataParams{
		Text: strPtr(""), Filters: map[string]interface{}{"topic": "physics"},
	})
	assertSameIDs(t, metaIDs(rows), []string{"d0", "d2", "d4"}, "empty text filter-only")
	for _, r := range rows {
		if r.HasScore() {
			t.Errorf("filter-only row %s should have no score", r.Id)
		}
	}
}

func TestBM25TextMatchingNoDocumentReturnsEmpty(t *testing.T) {
	index := bm25Index(t)
	rows := queryMetaRows(t, index, cyborgdb.QueryMetadataParams{Text: strPtr("zzzznonexistent")})
	if len(rows) != 0 {
		t.Errorf("expected empty, got %v", metaIDs(rows))
	}
}

func TestBM25TopKLargerThanMatchesReturnsAll(t *testing.T) {
	index := bm25Index(t)
	// top_k above the match count is a cap, not a floor.
	rows := queryMetaRows(t, index, cyborgdb.QueryMetadataParams{Text: strPtr("quantum"), TopK: 100})
	assertSameIDs(t, metaIDs(rows), bm25AnyTerm, "top_k as cap")
}

func TestBM25TextSearchIsCaseInsensitive(t *testing.T) {
	index := bm25Index(t)
	upper := queryMetaRows(t, index, cyborgdb.QueryMetadataParams{Text: strPtr("QUANTUM COMPUTING")})
	lower := queryMetaRows(t, index, cyborgdb.QueryMetadataParams{Text: strPtr("quantum computing")})
	assertSameIDs(t, metaIDs(upper), metaIDs(lower), "case-insensitive")
	assertSameIDs(t, metaIDs(lower), bm25AnyTerm, "lower-case matches quantum docs")
}

func TestBM25OrderByWithTextIsRejected(t *testing.T) {
	index := bm25Index(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Text results are relevance-ranked, so OrderBy alongside Text is
	// unsupported and must be rejected rather than silently ignored.
	if _, err := index.QueryMetadata(ctx, cyborgdb.QueryMetadataParams{
		Text: strPtr("quantum"), OrderBy: "topic",
	}); err == nil {
		t.Error("order_by with text should be rejected")
	}
}

func TestBM25NonFilterableFieldRejectedEvenWithText(t *testing.T) {
	index := bm25Index(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// The metadata schema is enforced on the text path too: a pre-filter on a
	// non-filterable field (body is full_text, so filterable=false) is rejected.
	if _, err := index.QueryMetadata(ctx, cyborgdb.QueryMetadataParams{
		Text: strPtr("quantum"), Filters: map[string]interface{}{"body": "quantum"},
	}); err == nil {
		t.Error("pre-filter on a non-filterable field should be rejected")
	}
}

// -- Query(Text, Filters) : hybrid + pre-filter ------------------------ //

func TestBM25HybridQueryAppliesMetadataFilter(t *testing.T) {
	index := bm25Index(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// topic=food pre-filters the hybrid candidate set: no quantum doc survives,
	// so only food docs can appear — never a quantum doc, and none carry a
	// distance (hybrid rows are scored).
	resp, err := index.Query(ctx, cyborgdb.QueryParams{
		QueryVector: generateRandomVectors(1, bm25Dim)[0],
		Text:        strPtr("quantum computing"),
		Filters:     map[string]interface{}{"topic": "food"},
		TopK:        6,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	items := getQueryResultItems(&resp.Results)
	if len(items) == 0 {
		t.Fatal("expected the food doc d3 to survive the pre-filter, got no rows")
	}
	for _, item := range items {
		if item.Id != "d3" {
			t.Errorf("only food doc d3 may appear, got %s", item.Id)
		}
		if item.HasDistance() {
			t.Errorf("hybrid row %s should not carry a distance", item.Id)
		}
	}
}

func TestBM25NoTextReturnsUnscoredIDRows(t *testing.T) {
	index := bm25Index(t)
	// Without Text this stays a filter-only query: {id} rows, no score.
	rows := queryMetaRows(t, index, cyborgdb.QueryMetadataParams{
		Filters: map[string]interface{}{"topic": "physics"},
	})
	assertSameIDs(t, metaIDs(rows), []string{"d0", "d2", "d4"}, "filter-only ids")
	for _, r := range rows {
		if r.HasScore() {
			t.Errorf("filter-only row %s should have no score", r.Id)
		}
	}
}

// -- Query(Text) : hybrid BM25 + vector -------------------------------- //

func TestBM25HybridQueryVectorCarriesScore(t *testing.T) {
	index := bm25Index(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := index.Query(ctx, cyborgdb.QueryParams{
		QueryVector: generateRandomVectors(1, bm25Dim)[0],
		Text:        strPtr("quantum computing"),
		TopK:        6,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	items := getQueryResultItems(&resp.Results)
	if len(items) == 0 {
		t.Fatal("expected hybrid results")
	}
	// Hybrid rows are scored (fused), not distance-ranked, and non-increasing.
	prev := float32(0)
	for i, item := range items {
		if !item.HasScore() {
			t.Errorf("row %s missing score", item.Id)
		}
		if item.HasDistance() {
			t.Errorf("hybrid row %s should not carry a distance", item.Id)
		}
		if i > 0 && item.GetScore() > prev {
			t.Errorf("scores not descending: %f > %f", item.GetScore(), prev)
		}
		prev = item.GetScore()
	}
}

// The batch path forwards the text leg too (encrypted_index.go's
// BatchQueryRequest branch): every query in the batch comes back hybrid-scored,
// not distance-ranked.
func TestBM25BatchHybridQueryCarriesScore(t *testing.T) {
	index := bm25Index(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := index.Query(ctx, cyborgdb.QueryParams{
		BatchQueryVectors: generateRandomVectors(2, bm25Dim),
		Text:              strPtr("quantum computing"),
		TopK:              6,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	batches := getBatchQueryResults(&resp.Results)
	if len(batches) != 2 {
		t.Fatalf("expected 2 result sets for a 2-vector batch, got %d", len(batches))
	}
	for b, items := range batches {
		if len(items) == 0 {
			t.Fatalf("batch %d: expected hybrid results", b)
		}
		// Hybrid rows are fused-scored, not distance-ranked, and non-increasing.
		prev := float32(0)
		for i, item := range items {
			if !item.HasScore() {
				t.Errorf("batch %d row %s missing score", b, item.Id)
			}
			if item.HasDistance() {
				t.Errorf("batch %d hybrid row %s should not carry a distance", b, item.Id)
			}
			if i > 0 && item.GetScore() > prev {
				t.Errorf("batch %d scores not descending: %f > %f", b, item.GetScore(), prev)
			}
			prev = item.GetScore()
		}
	}
}

func TestBM25HybridBinaryVectorCarriesScore(t *testing.T) {
	index := bm25Index(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// The binary query path must forward the text leg too.
	resp, err := index.Query(ctx, cyborgdb.BinaryQueryParams{
		QueryVectors: generateRandomVectors(1, bm25Dim),
		Text:         strPtr("quantum computing"),
		TopK:         6,
		Alpha:        f64Ptr(0.5),
	})
	if err != nil {
		t.Fatalf("Binary query failed: %v", err)
	}
	items := getQueryResultItems(&resp.Results)
	if len(items) == 0 {
		t.Fatal("expected hybrid results")
	}
	for _, item := range items {
		if !item.HasScore() {
			t.Errorf("row %s missing score", item.Id)
		}
	}
}

func TestBM25HybridAlphaForwardedToService(t *testing.T) {
	index := bm25Index(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// alpha must reach the service: an out-of-[0, 1] value is rejected there,
	// proving the SDK forwards it rather than dropping it.
	if _, err := index.Query(ctx, cyborgdb.QueryParams{
		QueryVector: generateRandomVectors(1, bm25Dim)[0],
		Text:        strPtr("quantum computing"),
		Alpha:       f64Ptr(5.0),
		TopK:        6,
	}); err == nil {
		t.Error("out-of-range alpha should be rejected by the service")
	}
}

func TestBM25HybridTextFieldsForwardedToService(t *testing.T) {
	index := bm25Index(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// text_fields must reach the service: naming a non-full-text field (topic)
	// is rejected there, proving forwarding on the hybrid path.
	if _, err := index.Query(ctx, cyborgdb.QueryParams{
		QueryVector: generateRandomVectors(1, bm25Dim)[0],
		Text:        strPtr("quantum"),
		TextFields:  []string{"topic"},
		TopK:        6,
	}); err == nil {
		t.Error("naming a non-full-text field in text_fields should be rejected")
	}
}

func TestBM25PureVectorQueryStillUsesDistance(t *testing.T) {
	index := bm25Index(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// A pure vector query (no Text) with include=[distance] carries distances,
	// not scores.
	resp, err := index.Query(ctx, cyborgdb.QueryParams{
		QueryVector: generateRandomVectors(1, bm25Dim)[0],
		TopK:        6,
		Include:     []string{"distance"},
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	items := getQueryResultItems(&resp.Results)
	if len(items) == 0 {
		t.Fatal("expected results")
	}
	for _, item := range items {
		if !item.HasDistance() {
			t.Errorf("row %s should carry a distance", item.Id)
		}
		if item.HasScore() {
			t.Errorf("pure vector row %s should not carry a score", item.Id)
		}
	}
}

// -- Two full_text fields: metadata filter narrows to a proper subset --- //

// bm25FilterRow is (id, title, body, lang). "quantum" appears in different
// fields per doc; lang splits the matches.
type bm25FilterRow struct {
	id, title, body, lang string
}

var bm25FilterRows = []bm25FilterRow{
	{"a", "quantum theory", "notes on physics", "en"},        // title
	{"b", "kitchen recipes", "a quantum leap forward", "en"}, // body only
	{"c", "quantum hardware", "qubit fabrication", "fr"},     // title
	{"d", "sourdough bread", "baking at home", "en"},         // no match
}

var quantumAnyField = []string{"a", "b", "c"}
var quantumInTitle = []string{"a", "c"}

func bm25FilterIndex(t *testing.T) *cyborgdb.EncryptedIndex {
	t.Helper()
	client := newIsolatedClient(t)
	name := generateUniqueName("bm25_filter_")
	dim := int32(bm25Dim)
	metric := "euclidean"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	index, err := client.CreateIndex(ctx, &cyborgdb.CreateIndexParams{
		IndexName:      name,
		IndexKey:       generateRandomKey(),
		Dimension:      &dim,
		Metric:         &metric,
		MetadataSchema: map[string]cyborgdb.MetadataFieldPolicy{"lang": {Filterable: boolPtr(true)}},
		TextFields:     []string{"title", "body"},
	})
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanCancel()
		_ = index.DeleteIndex(cleanCtx)
	})

	vectors := generateRandomVectors(len(bm25FilterRows), bm25Dim)
	items := make(cyborgdb.VectorItems, len(bm25FilterRows))
	for i, row := range bm25FilterRows {
		items[i] = cyborgdb.VectorItem{
			Id:       row.id,
			Vector:   vectors[i],
			Metadata: map[string]interface{}{"title": row.title, "body": row.body, "lang": row.lang},
		}
	}
	if err := index.Upsert(ctx, items); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	waitForPropagation(2 * time.Second)
	return index
}

func TestBM25TextMatchesAcrossBothFields(t *testing.T) {
	index := bm25FilterIndex(t)
	rows := queryMetaRows(t, index, cyborgdb.QueryMetadataParams{Text: strPtr("quantum")})
	assertSameIDs(t, metaIDs(rows), quantumAnyField, "match across both fields")
}

func TestBM25FilterNarrowsTextMatchesToProperSubset(t *testing.T) {
	index := bm25FilterIndex(t)
	// text matches {a, b, c}; lang=en drops the French doc c, leaving a strict
	// subset — proving the pre-filter intersects rather than replaces.
	rows := queryMetaRows(t, index, cyborgdb.QueryMetadataParams{
		Text: strPtr("quantum"), Filters: map[string]interface{}{"lang": "en"},
	})
	assertSameIDs(t, metaIDs(rows), []string{"a", "b"}, "filter narrows to subset")
}

func TestBM25TextFieldsExcludesMatchInUnsearchedField(t *testing.T) {
	index := bm25FilterIndex(t)
	// Restricting to title drops b, whose only "quantum" is in body.
	rows := queryMetaRows(t, index, cyborgdb.QueryMetadataParams{
		Text: strPtr("quantum"), TextFields: []string{"title"},
	})
	assertSameIDs(t, metaIDs(rows), quantumInTitle, "text_fields excludes unsearched field")
}

func TestBM25TextFieldsAndFilterCompose(t *testing.T) {
	index := bm25FilterIndex(t)
	// Both narrowings apply: title-only → {a, c}, then lang=en drops the French
	// c, leaving just {a}.
	rows := queryMetaRows(t, index, cyborgdb.QueryMetadataParams{
		Text:       strPtr("quantum"),
		TextFields: []string{"title"},
		Filters:    map[string]interface{}{"lang": "en"},
	})
	assertSameIDs(t, metaIDs(rows), []string{"a"}, "text_fields + filter compose")
}

func TestBM25FieldWeightsAcceptedAndRankStable(t *testing.T) {
	index := bm25FilterIndex(t)
	// Per-field weights (parallel to the searched fields) are forwarded and
	// accepted; the matched set is unchanged by re-weighting.
	rows := queryMetaRows(t, index, cyborgdb.QueryMetadataParams{
		Text:             strPtr("quantum"),
		TextFields:       []string{"title", "body"},
		TextFieldWeights: []float32{2.0, 1.0},
	})
	assertSameIDs(t, metaIDs(rows), quantumAnyField, "field weights accepted")
}

// -- An index with no full_text field: BM25 is absent, not empty ------- //

func bm25NoneIndex(t *testing.T) *cyborgdb.EncryptedIndex {
	t.Helper()
	client := newIsolatedClient(t)
	index, _ := newIsolatedIndex(t, client, "bm25_none", int32(bm25Dim))
	seedIndex(t, index, "n", 4, bm25Dim)
	waitForPropagation(2 * time.Second)
	return index
}

func TestBM25IsNilWhenNotConfigured(t *testing.T) {
	index := bm25NoneIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	config, err := index.BM25(ctx)
	if err != nil {
		t.Fatalf("BM25 failed: %v", err)
	}
	if config != nil {
		t.Errorf("expected nil BM25 config for an index with no full_text field, got %+v", config)
	}
}

func TestBM25TextQueryRejectedWithoutFullTextField(t *testing.T) {
	index := bm25NoneIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := index.QueryMetadata(ctx, cyborgdb.QueryMetadataParams{Text: strPtr("quantum")}); err == nil {
		t.Error("text query without a full_text field should be rejected")
	}
}

// -- MetadataResult shape (no service needed) -------------------------- //

// TestBM25MetadataResultShape pins the public MetadataResult row type: an Id
// plus an optional Score. Mirrors py TestMetadataResultContract, adapted to
// Go's compile-time struct shape.
func TestBM25MetadataResultShape(t *testing.T) {
	// Filter-only row: Id set, Score unset.
	filterOnly := cyborgdb.MetadataResult{Id: "x"}
	if filterOnly.GetId() != "x" {
		t.Errorf("Id: got %q, want x", filterOnly.GetId())
	}
	if filterOnly.HasScore() {
		t.Error("a filter-only row should have no score")
	}

	// Text row: Id and Score both set.
	scored := cyborgdb.MetadataResult{Id: "y"}
	scored.SetScore(1.5)
	if !scored.HasScore() || !approxEqual(scored.GetScore(), 1.5) {
		t.Errorf("scored row: HasScore=%v Score=%v", scored.HasScore(), scored.GetScore())
	}
}
