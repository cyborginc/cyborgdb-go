package test

import (
	"context"
	"strings"
	"testing"
	"time"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
)

// BM25 analyzer behaviour, scoring properties, tuning parameters and
// lifecycle. Mirrors py tests/test_bm25.py TestBM25Analyzer,
// TestBM25ScoringProperties, TestBM25TuningParameters and TestBM25Lifecycle.
//
// These assert ranking, never an absolute score: scores shift legitimately
// with the analyzer version.

const scoringDim = 4

// seedTextIndex creates an untrained index whose only full-text field is
// `body`, upserts rows, and waits for them to become visible. extra carries
// optional create-time BM25 tuning.
func seedTextIndex(t *testing.T, prefix string, rows [][2]string, extra func(*cyborgdb.CreateIndexParams)) *cyborgdb.EncryptedIndex {
	t.Helper()
	client := newIsolatedClient(t)
	dim := int32(scoringDim)
	metric := "euclidean"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	params := &cyborgdb.CreateIndexParams{
		IndexName:  generateUniqueName(prefix),
		IndexKey:   generateRandomKey(),
		Dimension:  &dim,
		Metric:     &metric,
		TextFields: []string{"body"},
	}
	if extra != nil {
		extra(params)
	}
	index, err := client.CreateIndex(ctx, params)
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanCancel()
		_ = index.DeleteIndex(cleanCtx)
	})

	items := make(cyborgdb.VectorItems, len(rows))
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row[0]
		items[i] = cyborgdb.VectorItem{
			Id:       row[0],
			Vector:   []float32{0.1, 0.2, 0.3, 0.4},
			Metadata: map[string]interface{}{"body": row[1]},
		}
	}
	if err := index.Upsert(ctx, items); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	waitForIDs(t, index, ids)
	return index
}

// textIDs is the set of ids matching a text query.
func textIDs(t *testing.T, index *cyborgdb.EncryptedIndex, text string) map[string]bool {
	t.Helper()
	rows := queryMetaRows(t, index, cyborgdb.QueryMetadataParams{Text: &text})
	out := make(map[string]bool, len(rows))
	for _, row := range rows {
		out[row.Id] = true
	}
	return out
}

// textRanked is the ids matching a text query, in returned (score) order.
func textRanked(t *testing.T, index *cyborgdb.EncryptedIndex, text string) []string {
	t.Helper()
	return metaIDs(queryMetaRows(t, index, cyborgdb.QueryMetadataParams{Text: &text}))
}

// textScores maps id to BM25 score for a text query.
func textScores(t *testing.T, index *cyborgdb.EncryptedIndex, text string) map[string]float32 {
	t.Helper()
	rows := queryMetaRows(t, index, cyborgdb.QueryMetadataParams{Text: &text})
	out := make(map[string]float32, len(rows))
	for _, row := range rows {
		out[row.Id] = row.GetScore()
	}
	return out
}

// indexOf reports the position of id in ranked, or -1.
func indexOf(ranked []string, id string) int {
	for i, got := range ranked {
		if got == id {
			return i
		}
	}
	return -1
}

// -- analyzer ----------------------------------------------------------- //

// analyzerRows exercises the tokenizer/stemmer pipeline. It is not
// configurable from the SDK — only reported as analyzer_version — so a change
// here would otherwise go unnoticed. Every expectation was measured against a
// running service, not assumed.
var analyzerRows = [][2]string{
	{"stem", "running runner runs"},
	{"punct", "mind-killer, fear! (really)"},
	{"accent", "café résumé naïve"},
	{"stop", "the a an and or but of"},
	{"num", "version 42 build 7"},
	{"case", "MixedCase WORD"},
	{"plural", "boxes churches"},
}

func TestBM25AnalyzerBehaviour(t *testing.T) {
	index := seedTextIndex(t, "bm25_analyzer_", analyzerRows, nil)

	t.Run("stems terms to a common root", func(t *testing.T) {
		for _, term := range []string{"run", "runs", "runner", "running"} {
			if !textIDs(t, index, term)["stem"] {
				t.Errorf("%q should match the stemmed document", term)
			}
		}
	})

	t.Run("stems plurals to their singular", func(t *testing.T) {
		for _, term := range []string{"box", "church"} {
			if !textIDs(t, index, term)["plural"] {
				t.Errorf("%q should match the plural document", term)
			}
		}
	})

	t.Run("strips punctuation and splits hyphens", func(t *testing.T) {
		for _, term := range []string{"mind", "killer", "fear"} {
			if !textIDs(t, index, term)["punct"] {
				t.Errorf("%q should survive punctuation stripping", term)
			}
		}
	})

	t.Run("folds case both ways", func(t *testing.T) {
		for _, term := range []string{"mixedcase", "WORD"} {
			if !textIDs(t, index, term)["case"] {
				t.Errorf("%q should match regardless of case", term)
			}
		}
	})

	t.Run("drops stop words", func(t *testing.T) {
		for _, term := range []string{"the", "and", "of"} {
			if got := textIDs(t, index, term); len(got) != 0 {
				t.Errorf("stop word %q matched %v", term, got)
			}
		}
	})

	t.Run("indexes numeric tokens", func(t *testing.T) {
		if !textIDs(t, index, "42")["num"] {
			t.Error("numeric token 42 should be searchable")
		}
	})

	t.Run("does not fold accents", func(t *testing.T) {
		// A limitation, pinned deliberately: adding accent folding is a
		// user-visible search change and should break this test.
		if !textIDs(t, index, "café")["accent"] {
			t.Error("the accented form should match itself")
		}
		if got := textIDs(t, index, "cafe"); len(got) != 0 {
			t.Errorf("unaccented \"cafe\" matched %v; accent folding was added", got)
		}
	})
}

// -- scoring properties -------------------------------------------------- //

// scoringDocs differ in exactly one BM25 property per pair:
//
//	IDF     "zeppelin" in one document, "common" in five; both candidates are
//	        the same length and match one query term, so only rarity separates.
//	LENGTH  same term and term-frequency, very different lengths.
//	TF      same length, different term-frequency.
var scoringDocs = [][2]string{
	{"idf_rare", "zeppelin padding padding padding"},
	{"idf_common", "common padding padding padding"},
	{"c1", "common padding padding padding"},
	{"c2", "common padding padding padding"},
	{"c3", "common padding padding padding"},
	{"c4", "common padding padding padding"},
	{"len_short", "target"},
	{"len_long", "target " + strings.Repeat("filler ", 24)},
	{"tf_one", "saturate alpha beta gamma delta"},
	{"tf_many", "saturate saturate saturate saturate saturate"},
}

func TestBM25ScoringProperties(t *testing.T) {
	index := seedTextIndex(t, "bm25_scoring_", scoringDocs, nil)

	t.Run("a rare term outranks a common one", func(t *testing.T) {
		ranked := textRanked(t, index, "zeppelin common")
		rare, common := indexOf(ranked, "idf_rare"), indexOf(ranked, "idf_common")
		if rare < 0 || common < 0 {
			t.Fatalf("both candidates should match; got %v", ranked)
		}
		if rare >= common {
			t.Errorf("a term matching 1 of 10 documents must outrank one matching 5; got %v", ranked)
		}
	})

	t.Run("a shorter document outranks a longer one", func(t *testing.T) {
		ranked := textRanked(t, index, "target")
		if len(ranked) < 2 || ranked[0] != "len_short" || ranked[1] != "len_long" {
			t.Errorf("got %v, want [len_short len_long]", ranked)
		}
	})

	t.Run("higher term frequency scores higher", func(t *testing.T) {
		ranked := textRanked(t, index, "saturate")
		if len(ranked) < 2 || ranked[0] != "tf_many" || ranked[1] != "tf_one" {
			t.Errorf("got %v, want [tf_many tf_one]", ranked)
		}
	})

	t.Run("a term in every document still scores", func(t *testing.T) {
		// IDF shrinks with document frequency but must not reach zero.
		scores := textScores(t, index, "common")
		want := []string{"idf_common", "c1", "c2", "c3", "c4"}
		if len(scores) != len(want) {
			t.Fatalf("got %d matches, want %d: %v", len(scores), len(want), scores)
		}
		for _, id := range want {
			score, ok := scores[id]
			if !ok {
				t.Errorf("%s missing from the results", id)
				continue
			}
			if score <= 0 {
				t.Errorf("%s scored %v; a matching document must score above zero", id, score)
			}
		}
	})
}

// -- tuning parameters ---------------------------------------------------- //

func TestBM25BZeroRemovesTheLengthPenalty(t *testing.T) {
	defaultB := seedTextIndex(t, "bm25_tune_bdefault_", scoringDocs, nil)
	noLength := seedTextIndex(t, "bm25_tune_bzero_", scoringDocs, func(p *cyborgdb.CreateIndexParams) {
		p.Bm25B = f64Ptr(0.0)
	})

	// The default case is asserted alongside so the comparison means something.
	ranked := textRanked(t, defaultB, "target")
	if len(ranked) < 2 || ranked[0] != "len_short" || ranked[1] != "len_long" {
		t.Errorf("at the default b: got %v, want [len_short len_long]", ranked)
	}

	scores := textScores(t, noLength, "target")
	if len(scores) != 2 {
		t.Fatalf("expected both length documents to match; got %v", scores)
	}
	if !approxEqual(scores["len_short"], scores["len_long"]) {
		t.Errorf("with b=0 document length must not affect the score; got short=%v long=%v",
			scores["len_short"], scores["len_long"])
	}
}

func TestBM25K1ZeroMakesScoringBinary(t *testing.T) {
	// At k1=0 the tf component collapses to presence/absence.
	defaultK1 := seedTextIndex(t, "bm25_tune_kdefault_", scoringDocs, nil)
	binary := seedTextIndex(t, "bm25_tune_kzero_", scoringDocs, func(p *cyborgdb.CreateIndexParams) {
		p.Bm25K1 = f64Ptr(0.0)
	})

	defaults := textScores(t, defaultK1, "saturate")
	if defaults["tf_many"] <= defaults["tf_one"] {
		t.Errorf("at the default k1, repeating a term must raise the score; got many=%v one=%v",
			defaults["tf_many"], defaults["tf_one"])
	}

	flat := textScores(t, binary, "saturate")
	if !approxEqual(flat["tf_many"], flat["tf_one"]) {
		t.Errorf("with k1=0 term frequency must stop mattering; got many=%v one=%v",
			flat["tf_many"], flat["tf_one"])
	}
}

// -- lifecycle ------------------------------------------------------------ //

// lifecycleRows: "alpha" matches m0, m1 and m3 but not m2.
var lifecycleRows = [][2]string{
	{"m0", "alpha beta"},
	{"m1", "alpha gamma"},
	{"m2", "delta epsilon"},
	{"m3", "alpha zeta"},
}

// lifecycleIndex seeds lifecycleRows with basis vectors so each document is
// independently addressable.
func lifecycleIndex(t *testing.T) *cyborgdb.EncryptedIndex {
	t.Helper()
	return seedTextIndex(t, "bm25_lifecycle_", lifecycleRows, nil)
}

// BM25 after mutation. Scores depend on corpus-wide statistics (document
// count, total length) that feed IDF and length normalisation. CEI tests those
// hard at its own layer; nothing checked they are wired through core ->
// service -> SDK, where stale statistics would skew every score with no error
// surface.

func TestBM25DeleteRemovesADocumentFromTextResults(t *testing.T) {
	index := lifecycleIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if got := textIDs(t, index, "alpha"); len(got) != 3 || !got["m0"] || !got["m1"] || !got["m3"] {
		t.Fatalf("precondition: got %v, want m0, m1, m3", got)
	}
	if err := index.Delete(ctx, []string{"m1"}); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	waitUntilIDsGone(t, index, []string{"m1"})

	got := textIDs(t, index, "alpha")
	if len(got) != 2 || !got["m0"] || !got["m3"] {
		t.Errorf("after deleting m1: got %v, want m0 and m3", got)
	}
}

func TestBM25DeletedDocumentNeverComesBack(t *testing.T) {
	index := lifecycleIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := index.Delete(ctx, []string{"m0"}); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	waitUntilIDsGone(t, index, []string{"m0"})

	for _, text := range []string{"alpha", "alpha beta", "beta"} {
		if textIDs(t, index, text)["m0"] {
			t.Errorf("m0 resurfaced for %q", text)
		}
	}
}

func TestBM25UpdatingATextFieldMovesTheDocument(t *testing.T) {
	index := lifecycleIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if textIDs(t, index, "alpha")["m2"] {
		t.Fatal("precondition: m2 should not match \"alpha\" before the rewrite")
	}
	err := index.Upsert(ctx, cyborgdb.VectorItems{{
		Id:       "m2",
		Vector:   []float32{0.0, 0.0, 1.0, 0.0},
		Metadata: map[string]interface{}{"body": "alpha omega"},
	}})
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	waitForCondition(t, "m2 becomes searchable for \"alpha\" after its body was rewritten", func() bool {
		return textIDs(t, index, "alpha")["m2"]
	})

	// ...and the old term no longer matches it: the update replaced the
	// document's postings rather than adding to them.
	if textIDs(t, index, "delta")["m2"] {
		t.Error("m2 still matches its old body; the update added postings instead of replacing them")
	}
}

func TestBM25ReupsertDoesNotDoubleCount(t *testing.T) {
	index := lifecycleIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Double-counted corpus statistics would shift IDF and the length
	// normaliser, moving every score.
	before := textScores(t, index, "alpha")

	// `marker` rides along only to give the poll below something to observe;
	// `body` is byte-identical, so BM25 must be unaffected.
	err := index.Upsert(ctx, cyborgdb.VectorItems{{
		Id:       "m0",
		Vector:   []float32{1.0, 0.0, 0.0, 0.0},
		Metadata: map[string]interface{}{"body": "alpha beta", "marker": "reupserted"},
	}})
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	waitForCondition(t, "the re-upserted m0 carries its new marker", func() bool {
		rows := queryMetaRows(t, index, cyborgdb.QueryMetadataParams{
			Filters: map[string]interface{}{"marker": "reupserted"},
		})
		return len(rows) == 1 && rows[0].Id == "m0"
	})

	after := textScores(t, index, "alpha")
	if len(after) != len(before) {
		t.Fatalf("match set changed: before %v, after %v", before, after)
	}
	for id, want := range before {
		got, ok := after[id]
		if !ok {
			t.Errorf("%s disappeared after the re-upsert", id)
			continue
		}
		if !approxEqual(got, want) {
			t.Errorf("%s score moved from %v to %v; corpus statistics were double-counted", id, want, got)
		}
	}
}
