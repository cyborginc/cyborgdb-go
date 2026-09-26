package test

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
	"time"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
	"github.com/cyborginc/cyborgdb-go/internal"
)

// Large-batch behavior, binary/JSON path parity, and training boundaries.
//
// Mirrors py tests/test_scale_and_paths.py. These cost more wall-clock than the
// rest of the suite and are aimed at the overnight run rather than per-PR CI.
// They exercise what small fixtures cannot: the binary encoder, the auto-train
// threshold, and the accuracy cost of quantized storage.

const (
	scaleDim = 64
	scaleN   = 2000
)

// Deterministic corpus: every assertion below must be reproducible across runs,
// so the vectors come from a fixed seed rather than fresh randomness.
var scaleRNG = rand.New(rand.NewSource(20260918))

// seededVectors draws count vectors of scaleDim from the shared deterministic
// source.
func seededVectors(count int) [][]float32 {
	out := make([][]float32, count)
	for i := range out {
		out[i] = make([]float32, scaleDim)
		for j := range out[i] {
			out[i][j] = scaleRNG.Float32()
		}
	}
	return out
}

var (
	scaleVectors = seededVectors(scaleN)
	scaleIDs     = paddedIDs("v", scaleN, 5)
)

// paddedIDs builds zero-padded ids so lexical and insertion order agree.
func paddedIDs(prefix string, n, width int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("%s%0*d", prefix, width, i)
	}
	return out
}

// bruteForceNearest is exhaustive ground truth by squared euclidean distance.
// Ties fall back to insertion order, matching the stable sort the Python
// fixture uses.
func bruteForceNearest(query []float32, vectors [][]float32, ids []string, k int) []string {
	type scored struct {
		id string
		i  int
		d  float64
	}
	all := make([]scored, len(vectors))
	for i, v := range vectors {
		var d float64
		for j, x := range v {
			diff := float64(x) - float64(query[j])
			d += diff * diff
		}
		all[i] = scored{ids[i], i, d}
	}
	sort.SliceStable(all, func(a, b int) bool {
		if all[a].d != all[b].d {
			return all[a].d < all[b].d
		}
		return all[a].i < all[b].i
	})
	if k > len(all) {
		k = len(all)
	}
	out := make([]string, k)
	for i := 0; i < k; i++ {
		out[i] = all[i].id
	}
	return out
}

// newScaleIndex creates an untrained index, optionally tweaked by extra.
func newScaleIndex(t *testing.T, prefix string, extra func(*cyborgdb.CreateIndexParams)) *cyborgdb.EncryptedIndex {
	t.Helper()
	client := newIsolatedClient(t)
	dim := int32(scaleDim)
	metric := "euclidean"
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	params := &cyborgdb.CreateIndexParams{
		IndexName: generateUniqueName(prefix),
		IndexKey:  generateRandomKey(),
		Dimension: &dim,
		Metric:    &metric,
	}
	if extra != nil {
		extra(params)
	}
	index, err := client.CreateIndex(ctx, params)
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanCancel()
		_ = index.DeleteIndex(cleanCtx)
	})
	return index
}

// queryIDs runs a query and returns the result ids in rank order.
func rankedQueryIDs(t *testing.T, index *cyborgdb.EncryptedIndex, input cyborgdb.QueryInput) []string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	resp, err := index.Query(ctx, input)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	items := getQueryResultItems(&resp.Results)
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.Id
	}
	return out
}

// -- binary / JSON path parity -------------------------------------------- //

// parityFixture holds the two indexes that must behave identically.
type parityFixture struct {
	jsonIndex   *cyborgdb.EncryptedIndex
	binaryIndex *cyborgdb.EncryptedIndex
	ids         []string
	vectors     [][]float32
}

// newParityFixture seeds the same 200 vectors through each encoder.
//
// The binary index goes through BinaryUpsertParams explicitly: passing
// VectorItems routes to the JSON encoder regardless of the vector type, which
// would exercise binary on the query side only.
func newParityFixture(t *testing.T) *parityFixture {
	t.Helper()
	const n = 200
	f := &parityFixture{
		ids:     paddedIDs("b", n, 3),
		vectors: seededVectors(n),
	}
	f.jsonIndex = newScaleIndex(t, "parity_json_", nil)
	f.binaryIndex = newScaleIndex(t, "parity_bin_", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	items := make(cyborgdb.VectorItems, n)
	metadata := make([]map[string]interface{}, n)
	for i, id := range f.ids {
		metadata[i] = map[string]interface{}{"n": i}
		items[i] = cyborgdb.VectorItem{Id: id, Vector: f.vectors[i], Metadata: metadata[i]}
	}
	if err := f.jsonIndex.Upsert(ctx, items); err != nil {
		t.Fatalf("JSON Upsert failed: %v", err)
	}
	err := f.binaryIndex.Upsert(ctx, cyborgdb.BinaryUpsertParams{
		IDs:      f.ids,
		Vectors:  f.vectors,
		Metadata: metadata,
	})
	if err != nil {
		t.Fatalf("binary Upsert failed: %v", err)
	}

	waitForIDs(t, f.jsonIndex, f.ids)
	waitForIDs(t, f.binaryIndex, f.ids)
	return f
}

func TestBinaryAndJSONEncodersRankIdentically(t *testing.T) {
	f := newParityFixture(t)
	jsonIDs := rankedQueryIDs(t, f.jsonIndex, cyborgdb.QueryParams{
		QueryVector: f.vectors[7], TopK: 20,
	})
	binaryIDs := rankedQueryIDs(t, f.binaryIndex, cyborgdb.BinaryQueryParams{
		QueryVectors: [][]float32{f.vectors[7]}, TopK: 20,
	})
	// Anchored as well as compared: identical encoders that were both wrong
	// would agree with each other.
	if !reflect.DeepEqual(jsonIDs, binaryIDs) {
		t.Errorf("encoders disagree:\n  json   %v\n  binary %v", jsonIDs, binaryIDs)
	}
	if len(jsonIDs) == 0 || jsonIDs[0] != f.ids[7] {
		t.Errorf("a vector must be its own nearest; got %v", jsonIDs)
	}
}

func TestBinaryAndJSONEncodersReturnTheSameResultShape(t *testing.T) {
	// The two paths build their result rows differently, so compare which
	// fields are populated and not just the ids.
	f := newParityFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for _, include := range [][]string{nil, {"distance"}, {"metadata"}, {"distance", "metadata"}} {
		t.Run(fmt.Sprintf("include=%v", include), func(t *testing.T) {
			jsonResp, err := f.jsonIndex.Query(ctx, cyborgdb.QueryParams{
				QueryVector: f.vectors[1], TopK: 3, Include: include,
			})
			if err != nil {
				t.Fatalf("JSON Query failed: %v", err)
			}
			binResp, err := f.binaryIndex.Query(ctx, cyborgdb.BinaryQueryParams{
				QueryVectors: [][]float32{f.vectors[1]}, TopK: 3, Include: include,
			})
			if err != nil {
				t.Fatalf("binary Query failed: %v", err)
			}
			jsonRow := getQueryResultItems(&jsonResp.Results)
			binRow := getQueryResultItems(&binResp.Results)
			if len(jsonRow) == 0 || len(binRow) == 0 {
				t.Fatalf("expected rows from both paths; got %d and %d", len(jsonRow), len(binRow))
			}
			if got, want := binRow[0].HasDistance(), jsonRow[0].HasDistance(); got != want {
				t.Errorf("distance present: binary %v, json %v", got, want)
			}
			if got, want := binRow[0].Metadata != nil, jsonRow[0].Metadata != nil; got != want {
				t.Errorf("metadata present: binary %v, json %v", got, want)
			}
		})
	}
}

func TestBinaryAndJSONEncodersRoundTripVectorsIdentically(t *testing.T) {
	f := newParityFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fromJSON, err := f.jsonIndex.Get(ctx, []string{f.ids[3]}, []string{"vector"})
	if err != nil {
		t.Fatalf("JSON Get failed: %v", err)
	}
	fromBinary, err := f.binaryIndex.Get(ctx, []string{f.ids[3]}, []string{"vector"})
	if err != nil {
		t.Fatalf("binary Get failed: %v", err)
	}
	if len(fromJSON.Results) != 1 || len(fromBinary.Results) != 1 {
		t.Fatalf("expected one row from each path")
	}
	jsonVec, binVec := fromJSON.Results[0].Vector, fromBinary.Results[0].Vector
	if !vectorsApproxEqual(jsonVec, binVec) {
		t.Errorf("encoders round-trip differently:\n  json   %v\n  binary %v", jsonVec, binVec)
	}
	if !vectorsApproxEqual(jsonVec, f.vectors[3]) {
		t.Errorf("round-tripped vector does not match the original")
	}
}

func TestBinaryAndJSONEncodersFilterIdentically(t *testing.T) {
	f := newParityFixture(t)
	filters := map[string]interface{}{"n": map[string]interface{}{"$lt": 50}}
	jsonIDs := rankedQueryIDs(t, f.jsonIndex, cyborgdb.QueryParams{
		QueryVector: f.vectors[0], TopK: 200, Filters: filters,
	})
	binaryIDs := rankedQueryIDs(t, f.binaryIndex, cyborgdb.BinaryQueryParams{
		QueryVectors: [][]float32{f.vectors[0]}, TopK: 200, Filters: filters,
	})
	assertSameIDs(t, jsonIDs, binaryIDs, "filtered JSON vs binary")
	assertSameIDs(t, jsonIDs, f.ids[:50], "filtered JSON against the expected prefix")
}

// -- include projection ---------------------------------------------------- //

// includeFixture is a one-row index for probing what `include` accepts.
func includeFixture(t *testing.T) (*cyborgdb.EncryptedIndex, []float32) {
	t.Helper()
	index := newScaleIndex(t, "include_", nil)
	vector := seededVectors(1)[0]
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	item := cyborgdb.VectorItem{
		Id:       "only",
		Vector:   vector,
		Metadata: map[string]interface{}{"n": 1},
	}
	contents := "hello"
	item.SetContents(internal.Contents{String: &contents})
	if err := index.Upsert(ctx, cyborgdb.VectorItems{item}); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	waitForIDs(t, index, []string{"only"})
	return index, vector
}

func TestIncludeSupportedValuesAreHonoured(t *testing.T) {
	index, vector := includeFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := index.Query(ctx, cyborgdb.QueryParams{
		QueryVector: vector, TopK: 1, Include: []string{"distance"},
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	items := getQueryResultItems(&resp.Results)
	if len(items) != 1 || !items[0].HasDistance() {
		t.Errorf("include=[distance] should return a distance; got %+v", items)
	}

	resp, err = index.Query(ctx, cyborgdb.QueryParams{
		QueryVector: vector, TopK: 1, Include: []string{"metadata"},
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	items = getQueryResultItems(&resp.Results)
	if len(items) != 1 || items[0].Metadata == nil {
		t.Fatalf("include=[metadata] should return metadata; got %+v", items)
	}
	if got := items[0].Metadata["n"]; fmt.Sprint(got) != "1" {
		t.Errorf("metadata n = %v, want 1", got)
	}
}

func TestIncludeGetHonoursVectorAndContents(t *testing.T) {
	// The asymmetry in cyborgdb-core#2404: these work on Get and are discarded
	// on Query. Asserted here only for Get, where the contract is documented.
	index, _ := includeFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := index.Get(ctx, []string{"only"}, []string{"vector", "contents"})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected one row, got %d", len(resp.Results))
	}
	if len(resp.Results[0].Vector) == 0 {
		t.Error("include=[vector] should return the vector")
	}
	if got := resp.Results[0].GetContents(); got != "hello" {
		t.Errorf("contents = %q, want \"hello\"", got)
	}
}

func TestIncludeUnknownValuesAreRejected(t *testing.T) {
	// KNOWN BUG — fails today. cyborgdb-core#2404: an unrecognized value is
	// silently discarded on both methods, so a typo such as "metdata" costs the
	// caller the field with no error. Unlike the vector/contents question, this
	// needs no documentation to be wrong.
	index, vector := includeFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if _, err := index.Query(ctx, cyborgdb.QueryParams{
		QueryVector: vector, TopK: 1, Include: []string{"bogus"},
	}); err == nil {
		t.Error("Query should reject an unknown include value")
	}
	if _, err := index.Get(ctx, []string{"only"}, []string{"bogus"}); err == nil {
		t.Error("Get should reject an unknown include value")
	}
}

// -- large batch ----------------------------------------------------------- //

// largeBatchIndex seeds scaleN vectors. Not enough to train:
// AUTO_TRAIN_MIN_VECTORS is 65536, so search here is still exact, which is what
// makes the brute-force comparisons below verifiable.
func largeBatchIndex(t *testing.T) *cyborgdb.EncryptedIndex {
	t.Helper()
	index := newScaleIndex(t, "scale_", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	items := make(cyborgdb.VectorItems, scaleN)
	for i, id := range scaleIDs {
		items[i] = cyborgdb.VectorItem{
			Id:       id,
			Vector:   scaleVectors[i],
			Metadata: map[string]interface{}{"bucket": i % 10},
		}
	}
	if err := index.Upsert(ctx, items); err != nil {
		t.Fatalf("Upsert of %d vectors failed: %v", scaleN, err)
	}
	waitForIDs(t, index, []string{scaleIDs[0], scaleIDs[scaleN-1]})
	return index
}

func TestLargeBatchEveryVectorIsRetrievable(t *testing.T) {
	index := largeBatchIndex(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := index.ListIDs(ctx)
	if err != nil {
		t.Fatalf("ListIDs failed: %v", err)
	}
	present := sortedSet(resp.Ids)
	var missing []string
	for _, id := range scaleIDs {
		if !present[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		t.Errorf("%d of %d vectors missing, first few: %v",
			len(missing), scaleN, missing[:min(5, len(missing))])
	}
}

func TestLargeBatchExactSearchMatchesBruteForce(t *testing.T) {
	index := largeBatchIndex(t)
	query := scaleVectors[123]
	got := rankedQueryIDs(t, index, cyborgdb.QueryParams{QueryVector: query, TopK: 10})
	want := bruteForceNearest(query, scaleVectors, scaleIDs, 10)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("exhaustive search disagrees with brute force:\n  got  %v\n  want %v", got, want)
	}
}

func TestLargeBatchFilteredSearch(t *testing.T) {
	index := largeBatchIndex(t)
	got := rankedQueryIDs(t, index, cyborgdb.QueryParams{
		QueryVector: scaleVectors[0],
		TopK:        scaleN,
		Filters:     map[string]interface{}{"bucket": 3},
	})
	var want []string
	for i, id := range scaleIDs {
		if i%10 == 3 {
			want = append(want, id)
		}
	}
	assertSameIDs(t, got, want, "bucket=3 at scale")
}

func TestLargeBatchTopKPrefixInvariant(t *testing.T) {
	// The first k of a larger result must equal the smaller result. Only
	// meaningful once the corpus exceeds top_k * window_mult, which the small
	// fixtures elsewhere never do.
	index := largeBatchIndex(t)
	query := scaleVectors[500]
	wide := rankedQueryIDs(t, index, cyborgdb.QueryParams{QueryVector: query, TopK: 50})
	narrow := rankedQueryIDs(t, index, cyborgdb.QueryParams{QueryVector: query, TopK: 10})
	if len(wide) < 10 {
		t.Fatalf("top_k=50 returned only %d rows", len(wide))
	}
	if !reflect.DeepEqual(wide[:10], narrow) {
		t.Errorf("top_k is not a prefix:\n  wide[:10] %v\n  narrow    %v", wide[:10], narrow)
	}
}

// -- training boundaries --------------------------------------------------- //

// seedForTraining upserts n vectors into a fresh index and returns their ids.
func seedForTraining(t *testing.T, index *cyborgdb.EncryptedIndex, n int) []string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	vectors := seededVectors(n)
	ids := paddedIDs("t", n, 5)
	items := make(cyborgdb.VectorItems, n)
	for i, id := range ids {
		items[i] = cyborgdb.VectorItem{Id: id, Vector: vectors[i]}
	}
	if err := index.Upsert(ctx, items); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	waitForIDs(t, index, ids[:1])
	return ids
}

func TestTrainingBelowTheMinimumIsASilentNoOp(t *testing.T) {
	// AUTO_TRAIN_MIN_VECTORS is 65536, so Train silently does nothing for any
	// index below it — returns successfully, leaves the index untrained, and
	// IsTrained is the only signal.
	index := newScaleIndex(t, "trainmin_", nil)
	ids := seedForTraining(t, index, 5)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	nLists := int32(64)
	if err := index.Train(ctx, cyborgdb.TrainParams{NLists: &nLists}); err != nil {
		t.Fatalf("Train below the minimum should not error: %v", err)
	}
	trained, err := index.IsTrained(ctx)
	if err != nil {
		t.Fatalf("IsTrained failed: %v", err)
	}
	if trained {
		t.Error("an index of 5 vectors must not report as trained")
	}
	got := rankedQueryIDs(t, index, cyborgdb.QueryParams{
		QueryVector: seededVectors(1)[0], TopK: 3,
	})
	if !isSubset(got, ids) {
		t.Errorf("query returned ids outside the seeded set: %v", got)
	}
}

func TestMoreListsThanVectorsIsASilentNoOp(t *testing.T) {
	// Degenerate case, same contract: no error, no training, correct results.
	index := newScaleIndex(t, "trainmin_", nil)
	ids := seedForTraining(t, index, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	nLists := int32(2)
	if err := index.Train(ctx, cyborgdb.TrainParams{NLists: &nLists}); err != nil {
		t.Fatalf("Train with n_lists == vector count should not error: %v", err)
	}
	trained, err := index.IsTrained(ctx)
	if err != nil {
		t.Fatalf("IsTrained failed: %v", err)
	}
	if trained {
		t.Error("an index of 2 vectors must not report as trained")
	}
	got := rankedQueryIDs(t, index, cyborgdb.QueryParams{
		QueryVector: seededVectors(1)[0], TopK: 2,
	})
	assertSameIDs(t, got, ids, "degenerate training still returns both vectors")
}

func TestUntrainedIndexStillQueries(t *testing.T) {
	// Training is an optimisation, not a prerequisite: an untrained index
	// answers exactly via exhaustive search.
	index := newScaleIndex(t, "trainmin_", nil)
	ids := seedForTraining(t, index, 20)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	got := rankedQueryIDs(t, index, cyborgdb.QueryParams{
		QueryVector: seededVectors(1)[0], TopK: 5,
	})
	if len(got) != 5 {
		t.Errorf("got %d rows, want 5", len(got))
	}
	if !isSubset(got, ids) {
		t.Errorf("query returned ids outside the seeded set: %v", got)
	}
	trained, err := index.IsTrained(ctx)
	if err != nil {
		t.Fatalf("IsTrained failed: %v", err)
	}
	if trained {
		t.Error("index reports trained without Train having taken effect")
	}
}

// -- storage precision accuracy -------------------------------------------- //

// Quantised storage trades accuracy for size — bound the trade.
// storage_precision_test.go covers validation and lifecycle across every tier,
// but nothing asserts that a quantized index still returns sensible results.

const precisionN = 500

// precisionFixture seeds the same corpus at each precision tier.
type precisionFixture struct {
	indexes map[string]*cyborgdb.EncryptedIndex
	ids     []string
	vectors [][]float32
	query   []float32
	truth   []string
}

func newPrecisionFixture(t *testing.T) *precisionFixture {
	t.Helper()
	f := &precisionFixture{
		indexes: map[string]*cyborgdb.EncryptedIndex{},
		ids:     paddedIDs("p", precisionN, 4),
		vectors: seededVectors(precisionN),
	}
	f.query = f.vectors[42]
	f.truth = bruteForceNearest(f.query, f.vectors, f.ids, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	for _, precision := range []string{"float32", "float16", "tq8"} {
		p := precision
		index := newScaleIndex(t, "prec_"+p+"_", func(params *cyborgdb.CreateIndexParams) {
			params.StoragePrecision = &p
		})
		items := make(cyborgdb.VectorItems, precisionN)
		for i, id := range f.ids {
			items[i] = cyborgdb.VectorItem{Id: id, Vector: f.vectors[i]}
		}
		if err := index.Upsert(ctx, items); err != nil {
			t.Fatalf("Upsert at %s failed: %v", p, err)
		}
		f.indexes[p] = index
	}
	for _, index := range f.indexes {
		waitForIDs(t, index, f.ids[:1])
	}
	return f
}

func TestStoragePrecisionFloat32IsExact(t *testing.T) {
	// No quantization, exhaustive search: the result must equal ground truth
	// outright, not merely approximate it.
	f := newPrecisionFixture(t)
	got := rankedQueryIDs(t, f.indexes["float32"], cyborgdb.QueryParams{
		QueryVector: f.query, TopK: 10,
	})
	if !reflect.DeepEqual(got, f.truth) {
		t.Errorf("float32 is not exact:\n  got  %v\n  want %v", got, f.truth)
	}
}

func TestStoragePrecisionQuantisedTiersStayUsable(t *testing.T) {
	// Deliberately loose floors. The purpose is to catch a tier that has become
	// badly wrong, not to police small accuracy movements — a tight threshold
	// here would be a flaky test rather than a useful one.
	f := newPrecisionFixture(t)
	for _, tc := range []struct {
		precision string
		floor     float64
	}{
		{"float16", 0.9},
		{"tq8", 0.5},
	} {
		t.Run(tc.precision, func(t *testing.T) {
			got := rankedQueryIDs(t, f.indexes[tc.precision], cyborgdb.QueryParams{
				QueryVector: f.query, TopK: 10,
			})
			truth := sortedSet(f.truth)
			hits := 0
			for _, id := range got {
				if truth[id] {
					hits++
				}
			}
			recall := float64(hits) / float64(len(f.truth))
			if recall < tc.floor {
				t.Errorf("%s recall@10 was %.2f, want at least %.2f", tc.precision, recall, tc.floor)
			}
		})
	}
}

func TestStoragePrecisionAVectorFindsItself(t *testing.T) {
	// The weakest possible accuracy guarantee, and the one that must hold even
	// at the most aggressive quantization.
	f := newPrecisionFixture(t)
	for precision, index := range f.indexes {
		t.Run(precision, func(t *testing.T) {
			got := rankedQueryIDs(t, index, cyborgdb.QueryParams{
				QueryVector: f.vectors[7], TopK: 1,
			})
			if len(got) != 1 || got[0] != f.ids[7] {
				t.Errorf("%s: nearest to its own vector was %v, want %s", precision, got, f.ids[7])
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
