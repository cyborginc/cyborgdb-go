package test

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
)

// The approximate search path.
//
// Mirrors py tests/test_trained_index.py. Everything else in the suite runs on
// untrained indexes, where search is exhaustive and exact. The service only
// trains past AUTO_TRAIN_MIN_VECTORS (65536 by default), so reaching the
// approximate path at all needs a corpus that size.
//
// The corpus is LoadSampleDataset (quickstart-75k): 75,000 vectors, 100
// queries, and ground-truth neighbors for both the trained and untrained
// cases. Building the index takes a couple of minutes, so the fixture is built
// once and shared, and this file is aimed at the overnight run.
//
// Covers RerankMult, which cannot be tested anywhere else: it widens the
// candidate set before a final exact re-scoring pass, so on an exhaustive index
// it is a no-op by construction.

const (
	trainedUpsertBatch = 5000
	trainedTimeout     = 10 * time.Minute
	trainedHybridText  = "grape cherry"
)

// trainedFixture is the shared 75k trained index.
type trainedFixture struct {
	index   *cyborgdb.EncryptedIndex
	data    *cyborgdb.SampleDataset
	ids     []string
	queries [][]float32
	truth   [][]int32
}

var (
	trainedOnce sync.Once
	trainedShip *trainedFixture
	trainedErr  string
)

// getTrainedIndex builds the fixture on first use. Every test in this file
// reads it, so paying the build cost once matters.
func getTrainedIndex(t *testing.T) *trainedFixture {
	t.Helper()
	trainedOnce.Do(func() {
		f, msg := buildTrainedIndex()
		trainedShip, trainedErr = f, msg
	})
	if trainedErr != "" {
		t.Fatalf("shared trained index unavailable: %s", trainedErr)
	}
	return trainedShip
}

// buildTrainedIndex loads the dataset, upserts it in batches, and waits for
// the background trainer to finish. Returns a message instead of a fixture on
// failure so every test reports the same cause.
func buildTrainedIndex() (*trainedFixture, string) {
	data, err := cyborgdb.LoadSampleDataset("")
	if err != nil {
		return nil, "LoadSampleDataset failed: " + err.Error()
	}

	client, err := cyborgdb.NewClient(testBaseURL(), testAPIKey())
	if err != nil {
		return nil, "NewClient failed: " + err.Error()
	}

	ctx, cancel := context.WithTimeout(context.Background(), trainedTimeout)
	defer cancel()

	dim := int32(data.Dimension)
	metric := data.Metric
	// `fruits` derives from the dataset's `list` field: ten terms, each in ~35%
	// of documents. Marking `string` full_text instead would make it
	// non-filterable and break the example-filter test below.
	index, err := client.CreateIndex(ctx, &cyborgdb.CreateIndexParams{
		IndexName:  generateUniqueName("trained_"),
		IndexKey:   generateRandomKey(),
		Dimension:  &dim,
		Metric:     &metric,
		TextFields: []string{"fruits"},
	})
	if err != nil {
		return nil, "CreateIndex failed: " + err.Error()
	}

	total := len(data.Ids)
	for start := 0; start < total; start += trainedUpsertBatch {
		stop := start + trainedUpsertBatch
		if stop > total {
			stop = total
		}
		items := make(cyborgdb.VectorItems, 0, stop-start)
		for i := start; i < stop; i++ {
			meta := map[string]interface{}{}
			if raw, ok := data.Metadata[i].(map[string]interface{}); ok {
				for k, v := range raw {
					meta[k] = v
				}
			}
			meta["fruits"] = joinListField(meta["list"])
			items = append(items, cyborgdb.VectorItem{
				Id:       data.Ids[i],
				Vector:   data.Vectors[i],
				Metadata: meta,
			})
		}
		if err := index.Upsert(ctx, items); err != nil {
			return nil, "Upsert failed: " + err.Error()
		}
	}

	// Crossing AUTO_TRAIN_MIN_VECTORS queues training on a background worker,
	// so the index is not trained the moment the upsert returns.
	deadline := time.Now().Add(trainedTimeout)
	for time.Now().Before(deadline) {
		trained, err := index.IsTrained(ctx)
		if err == nil && trained {
			return &trainedFixture{
				index:   index,
				data:    data,
				ids:     data.Ids,
				queries: data.Queries,
				truth:   data.TrainedNeighbors,
			}, ""
		}
		time.Sleep(5 * time.Second)
	}
	return nil, "index did not train within the timeout after upserting the full dataset"
}

// joinListField renders the dataset's `list` field as space-separated text for
// the BM25 leg.
func joinListField(raw interface{}) string {
	values, ok := raw.([]interface{})
	if !ok {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, v := range values {
		if s, ok := v.(string); ok {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, " ")
}

// recallAtK is the mean recall@k across every query in the dataset.
func (f *trainedFixture) recallAtK(t *testing.T, k int32, opts func(*cyborgdb.QueryParams)) float64 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	params := cyborgdb.QueryParams{BatchQueryVectors: f.queries, TopK: k}
	if opts != nil {
		opts(&params)
	}
	resp, err := f.index.Query(ctx, params)
	if err != nil {
		t.Fatalf("batch Query failed: %v", err)
	}
	rows := getBatchQueryResults(&resp.Results)
	if len(rows) == 0 {
		t.Fatal("batch query returned no result sets")
	}

	var total float64
	for q, row := range rows {
		expected := map[string]bool{}
		for _, idx := range f.truth[q][:k] {
			expected[f.ids[idx]] = true
		}
		hits := 0
		for _, item := range row {
			if expected[item.Id] {
				hits++
			}
		}
		total += float64(hits) / float64(k)
	}
	return total / float64(len(rows))
}

// firstQueryIDs runs a single-vector query against query 0.
func (f *trainedFixture) firstQueryIDs(t *testing.T, opts func(*cyborgdb.QueryParams)) []string {
	t.Helper()
	params := cyborgdb.QueryParams{QueryVector: f.queries[0], TopK: 10}
	if opts != nil {
		opts(&params)
	}
	return rankedQueryIDs(t, f.index, params)
}

func TestTrainedIndexIsTrained(t *testing.T) {
	f := getTrainedIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	trained, err := f.index.IsTrained(ctx)
	if err != nil {
		t.Fatalf("IsTrained failed: %v", err)
	}
	if !trained {
		t.Error("the fixture index should be trained")
	}
	nLists, err := f.index.NLists(ctx)
	if err != nil {
		t.Fatalf("NLists failed: %v", err)
	}
	if nLists <= 0 {
		t.Errorf("n_lists = %d, want a positive value on a trained index", nLists)
	}
}

func TestTrainedIndexRecallMeetsTheDatasetExpectation(t *testing.T) {
	// The dataset ships the recall its authors measured for the trained case.
	// Compared against that rather than a number invented here, with headroom
	// so ordinary index-build variation does not trip it.
	f := getTrainedIndex(t)
	want := f.data.TrainedRecall * 0.98
	got := f.recallAtK(t, 100, nil)
	if got < want {
		t.Errorf("recall@100 was %.3f, dataset expects ~%.3f (floor %.3f)",
			got, f.data.TrainedRecall, want)
	}
}

func TestTrainedIndexEveryVectorIsStillRetrievable(t *testing.T) {
	// Training rebuilds the index; nothing may be lost in the process.
	f := getTrainedIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var sample []string
	for i := 0; i < len(f.ids); i += 5000 {
		sample = append(sample, f.ids[i])
	}
	resp, err := f.index.Get(ctx, sample, []string{"vector"})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	got := make([]string, len(resp.Results))
	for i, row := range resp.Results {
		got[i] = row.Id
	}
	assertSameIDs(t, got, sample, "sampled ids after training")
}

// -- rerank_mult: only assertable where search is approximate -------------- //

func TestTrainedIndexWiderRerankingDoesNotReduceRecall(t *testing.T) {
	// The real contract: a wider candidate set cannot make results worse.
	// Averaged over all 100 queries — on any single query the two can
	// legitimately tie, so a per-query strict inequality would be flaky.
	f := getTrainedIndex(t)
	narrow := f.recallAtK(t, 10, func(p *cyborgdb.QueryParams) { p.RerankMult = int32Ptr(1) })
	wide := f.recallAtK(t, 10, func(p *cyborgdb.QueryParams) { p.RerankMult = int32Ptr(8) })

	// Measured sweep of recall@10: 0.783 / 0.944 / 0.971 / 0.977 / 0.977 for
	// rerank_mult 1 / 2 / 4 / 8 / 16. The effect is large and saturates around
	// 8, so assert a real improvement rather than merely "not worse" — the
	// latter would pass if rerank_mult were ignored entirely.
	if wide < narrow+0.05 {
		t.Errorf("recall@10 only moved %.3f -> %.3f between rerank_mult 1 and 8", narrow, wide)
	}
	if narrow <= 0.7 {
		t.Errorf("baseline recall@10 was only %.3f", narrow)
	}
}

func TestTrainedIndexRerankMultDoesNotChangeTheResultCount(t *testing.T) {
	f := getTrainedIndex(t)
	for _, mult := range []int32{1, 4, 16} {
		m := mult
		t.Run(itoa(m), func(t *testing.T) {
			got := f.firstQueryIDs(t, func(p *cyborgdb.QueryParams) { p.RerankMult = &m })
			if len(got) != 10 {
				t.Errorf("rerank_mult=%d returned %d rows, want 10", m, len(got))
			}
		})
	}
}

func TestTrainedIndexResultsStayOrderedByDistance(t *testing.T) {
	f := getTrainedIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for _, mult := range []int32{1, 8} {
		m := mult
		t.Run(itoa(m), func(t *testing.T) {
			resp, err := f.index.Query(ctx, cyborgdb.QueryParams{
				QueryVector: f.queries[0],
				TopK:        20,
				RerankMult:  &m,
				Include:     []string{"distance"},
			})
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}
			items := getQueryResultItems(&resp.Results)
			distances := make([]float32, len(items))
			for i, item := range items {
				distances[i] = item.GetDistance()
			}
			if !sort.SliceIsSorted(distances, func(a, b int) bool { return distances[a] < distances[b] }) {
				t.Errorf("rerank_mult=%d returned unsorted distances: %v", m, distances)
			}
		})
	}
}

func TestTrainedIndexTopKTimesRerankMultCeilingIsEnforced(t *testing.T) {
	// Nothing anywhere asserted the 10000 ceiling, or that the error names the
	// parameter responsible.
	f := getTrainedIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_, err := f.index.Query(ctx, cyborgdb.QueryParams{
		QueryVector: f.queries[0], TopK: 5000, RerankMult: int32Ptr(4),
	})
	if err == nil {
		t.Fatal("top_k=5000 with rerank_mult=4 exceeds the 10000 ceiling and should be rejected")
	}
	message := err.Error()
	if !strings.Contains(message, "10000") {
		t.Errorf("the error should state the limit, got: %s", message)
	}
	// KNOWN BUG — this assertion fails today. cyborgdb-core#2401: the message
	// says "top_k exceeds kMaxTopK" even though top_k=5000 is itself under the
	// limit; it is the product with rerank_mult that breaches it. A caller
	// reducing top_k to 2500 still fails.
	if !strings.Contains(message, "rerank_mult") {
		t.Errorf("the error should name the parameter responsible, got: %s", message)
	}
}

func TestTrainedIndexTheCeilingIsInclusive(t *testing.T) {
	// Exactly 10000 is accepted; only above it is rejected. Without this the
	// test above would still pass if the limit were off by one.
	f := getTrainedIndex(t)
	for _, tc := range []struct{ topK, rerankMult int32 }{
		{2000, 5}, {1000, 10}, {100, 100},
	} {
		tc := tc
		t.Run(itoa(tc.topK)+"x"+itoa(tc.rerankMult), func(t *testing.T) {
			got := rankedQueryIDs(t, f.index, cyborgdb.QueryParams{
				QueryVector: f.queries[0], TopK: tc.topK, RerankMult: &tc.rerankMult,
			})
			if len(got) == 0 {
				t.Errorf("top_k=%d rerank_mult=%d returned nothing at exactly the ceiling",
					tc.topK, tc.rerankMult)
			}
		})
	}
}

// -- metadata filtering against the approximate path ----------------------- //

func TestTrainedIndexExampleFiltersAllResolve(t *testing.T) {
	// The dataset ships filters its authors consider representative. Each must
	// return something and every row must satisfy the filter.
	f := getTrainedIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	for _, example := range f.data.ExampleFilters {
		example := example
		t.Run(example.Name, func(t *testing.T) {
			resp, err := f.index.Query(ctx, cyborgdb.QueryParams{
				QueryVector: f.queries[0],
				TopK:        50,
				Filters:     example.Filter,
				Include:     []string{"metadata"},
			})
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}
			items := getQueryResultItems(&resp.Results)
			if len(items) == 0 {
				t.Fatalf("%s matched nothing", example.Name)
			}
			for _, item := range items {
				if !metadataMatchesFilter(item.Metadata, example.Filter) {
					t.Errorf("%s does not satisfy %v (metadata %v)",
						item.Id, example.Filter, item.Metadata)
				}
			}
		})
	}
}

// metadataMatchesFilter evaluates the dataset's example filters locally, as an
// oracle against the service's own answer.
func metadataMatchesFilter(metadata, filters map[string]interface{}) bool {
	for field, condition := range filters {
		value := metadata[field]
		cond, isOperator := condition.(map[string]interface{})
		if !isOperator {
			if list, ok := value.([]interface{}); ok {
				if !containsValue(list, condition) {
					return false
				}
				continue
			}
			if !equalValues(value, condition) {
				return false
			}
			continue
		}
		for op, operand := range cond {
			switch op {
			case "$lt":
				if !(toFloat(value) < toFloat(operand)) {
					return false
				}
			case "$lte":
				if !(toFloat(value) <= toFloat(operand)) {
					return false
				}
			case "$gte":
				if !(toFloat(value) >= toFloat(operand)) {
					return false
				}
			case "$in":
				wanted, ok := operand.([]interface{})
				if !ok {
					return false
				}
				candidates, isList := value.([]interface{})
				if !isList {
					candidates = []interface{}{value}
				}
				matched := false
				for _, c := range candidates {
					if containsValue(wanted, c) {
						matched = true
						break
					}
				}
				if !matched {
					return false
				}
			}
		}
	}
	return true
}

// -- hybrid on the approximate path ---------------------------------------- //
//
// Not relevance tests: every term sits in ~35% of documents, so the ranking is
// mostly ties. These assert the wiring only, and are differential, so the weak
// text does not matter.

func TestTrainedIndexHybridReturnsFusedScores(t *testing.T) {
	f := getTrainedIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	text := trainedHybridText
	resp, err := f.index.Query(ctx, cyborgdb.QueryParams{
		QueryVector: f.queries[0], Text: &text, TopK: 10,
	})
	if err != nil {
		t.Fatalf("hybrid Query failed: %v", err)
	}
	items := getQueryResultItems(&resp.Results)
	if len(items) == 0 {
		t.Fatal("hybrid query returned nothing")
	}
	for _, item := range items {
		if !item.HasScore() {
			t.Errorf("%s carries no fused score", item.Id)
		}
		if item.HasDistance() {
			t.Errorf("%s carries a distance; hybrid rows are scored", item.Id)
		}
	}
}

func TestTrainedIndexAlphaOneReproducesTheApproximateVectorRanking(t *testing.T) {
	// The new ground covered here: at alpha=1 the fused result must match the
	// plain vector query, which on a trained index is the *approximate*
	// ranking. Nothing else checks that fusion leaves it intact.
	f := getTrainedIndex(t)
	text := trainedHybridText
	vectorOnly := f.firstQueryIDs(t, nil)
	fused := f.firstQueryIDs(t, func(p *cyborgdb.QueryParams) {
		p.Text = &text
		p.Alpha = f64Ptr(1.0)
	})
	if !reflect.DeepEqual(fused, vectorOnly) {
		t.Errorf("alpha=1 fused ranking %v does not match the vector ranking %v", fused, vectorOnly)
	}
}

func TestTrainedIndexAlphaZeroReproducesThePureBM25Ranking(t *testing.T) {
	f := getTrainedIndex(t)
	text := trainedHybridText
	textOnly := metaIDs(queryMetaRows(t, f.index, cyborgdb.QueryMetadataParams{
		Text: &text, TopK: 10,
	}))
	fused := f.firstQueryIDs(t, func(p *cyborgdb.QueryParams) {
		p.Text = &text
		p.Alpha = f64Ptr(0.0)
	})
	if !reflect.DeepEqual(fused, textOnly) {
		t.Errorf("alpha=0 fused ranking %v does not match the BM25 ranking %v", fused, textOnly)
	}
}

func TestTrainedIndexAlphaEndpointsDisagree(t *testing.T) {
	// Guards the two above: if the vector and text rankings coincided they
	// would both pass while proving nothing.
	f := getTrainedIndex(t)
	text := trainedHybridText
	vectorOnly := f.firstQueryIDs(t, nil)
	textOnly := metaIDs(queryMetaRows(t, f.index, cyborgdb.QueryMetadataParams{
		Text: &text, TopK: 10,
	}))
	if reflect.DeepEqual(vectorOnly, textOnly) {
		t.Errorf("the two legs agree (%v); the alpha endpoint tests would be vacuous", vectorOnly)
	}
}

func TestTrainedIndexHybridFilterPrefiltersBothLegs(t *testing.T) {
	f := getTrainedIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	text := trainedHybridText
	resp, err := f.index.Query(ctx, cyborgdb.QueryParams{
		QueryVector: f.queries[0],
		Text:        &text,
		Filters:     map[string]interface{}{"number": map[string]interface{}{"$lt": 100}},
		TopK:        20,
		Include:     []string{"metadata"},
	})
	if err != nil {
		t.Fatalf("filtered hybrid Query failed: %v", err)
	}
	items := getQueryResultItems(&resp.Results)
	if len(items) == 0 {
		t.Fatal("filtered hybrid query returned nothing")
	}
	for _, item := range items {
		if got := toFloat(item.Metadata["number"]); got >= 100 {
			t.Errorf("%s has number=%v, which the filter should have excluded", item.Id, got)
		}
	}
}

// -- small helpers for the filter oracle ----------------------------------- //

// itoa renders an int32 for a subtest name.
func itoa(v int32) string { return strconv.FormatInt(int64(v), 10) }

// toFloat coerces a JSON number (always float64 through encoding/json) to a
// float64, returning NaN for anything that is not numeric.
func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return math.NaN()
		}
		return f
	default:
		return math.NaN()
	}
}

// equalValues compares two decoded JSON scalars, treating numbers numerically
// so an int operand matches a float64 value.
func equalValues(a, b interface{}) bool {
	af, bf := toFloat(a), toFloat(b)
	if !math.IsNaN(af) && !math.IsNaN(bf) {
		return af == bf
	}
	return a == b
}

// containsValue reports whether list holds an element equal to want.
func containsValue(list []interface{}, want interface{}) bool {
	for _, item := range list {
		if equalValues(item, want) {
			return true
		}
	}
	return false
}
