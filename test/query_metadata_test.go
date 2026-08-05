package test

import (
	"context"
	"reflect"
	"testing"
	"time"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
)

// Metadata-only query (index.QueryMetadata) and the per-field indexing policy
// it enforces (CreateIndexParams.MetadataSchema).
//
// Mirrors py tests/test_query_metadata.py and js query_metadata.test.ts.
//
// The point is the asymmetry between the two read paths. Query can always fall
// back to a post-filter over the decrypted metadata, so there the policy only
// affects speed. QueryMetadata resolves everything from the index with no
// fallback, so the policy is enforced: $regex/$contains need a pattern field
// and a non-filterable field cannot be filtered at all. Each rejection is
// paired with the same filter succeeding via Query, so a failure points at the
// policy rather than at a broken filter.

const qmDim = 8
const qmCount = 6

func boolPtr(b bool) *bool { return &b }

// qmIndex creates an index whose `color` field opts into the regex dictionary,
// `shape` is indexed but not pattern, and `hidden` opts out of indexing.
// Even-numbered ids are red/square/secret, odd are green/circle/public.
func qmIndex(t *testing.T, schema map[string]cyborgdb.MetadataFieldPolicy) *cyborgdb.EncryptedIndex {
	t.Helper()
	client := newIsolatedClient(t)
	name := generateUniqueName("query_metadata_")
	dim := int32(qmDim)
	metric := "euclidean"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	index, err := client.CreateIndex(ctx, &cyborgdb.CreateIndexParams{
		IndexName:      name,
		IndexKey:       generateRandomKey(),
		Dimension:      &dim,
		Metric:         &metric,
		MetadataSchema: schema,
	})
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanCancel()
		_ = index.DeleteIndex(cleanCtx)
	})

	vectors := generateRandomVectors(qmCount, qmDim)
	ids := make([]string, qmCount)
	metadata := make([]map[string]interface{}, qmCount)
	for i := 0; i < qmCount; i++ {
		ids[i] = idFor(i)
		even := i%2 == 0
		metadata[i] = map[string]interface{}{
			"color":  pick(even, "red", "green"),
			"shape":  pick(even, "square", "circle"),
			"hidden": pick(even, "secret", "public"),
			"rank":   i,
			"loc":    map[string]interface{}{"city": pick(even, "paris", "lyon")},
		}
	}
	if err := index.UpsertVectors(ctx, ids, vectors, metadata); err != nil {
		t.Fatalf("UpsertVectors failed: %v", err)
	}
	waitForPropagation(2 * time.Second)
	return index
}

func idFor(i int) string { return "i" + string(rune('0'+i)) }

func pick(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

func evenIDs() []string {
	var out []string
	for i := 0; i < qmCount; i += 2 {
		out = append(out, idFor(i))
	}
	return out
}

func oddIDs() []string {
	var out []string
	for i := 1; i < qmCount; i += 2 {
		out = append(out, idFor(i))
	}
	return out
}

func sortedSet(ids []string) map[string]bool {
	set := map[string]bool{}
	for _, id := range ids {
		set[id] = true
	}
	return set
}

func assertSameIDs(t *testing.T, got, want []string, label string) {
	t.Helper()
	if !reflect.DeepEqual(sortedSet(got), sortedSet(want)) {
		t.Errorf("%s: got %v, want %v", label, got, want)
	}
}

func queryMeta(t *testing.T, index *cyborgdb.EncryptedIndex, params cyborgdb.QueryMetadataParams) []string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := index.QueryMetadata(ctx, params)
	if err != nil {
		t.Fatalf("QueryMetadata(%v) failed: %v", params.Filters, err)
	}
	if int(resp.Count) != len(resp.Ids) {
		t.Errorf("count %d disagrees with %d ids", resp.Count, len(resp.Ids))
	}
	return resp.Ids
}

// queryIDs runs the same filter through the vector path, for comparison.
func queryIDs(t *testing.T, index *cyborgdb.EncryptedIndex, filters map[string]interface{}) []string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := index.Query(ctx, cyborgdb.QueryParams{
		QueryVector: generateRandomVectors(1, qmDim)[0],
		TopK:        qmCount,
		Filters:     filters,
	})
	if err != nil {
		t.Fatalf("Query(%v) failed: %v", filters, err)
	}
	var ids []string
	for _, item := range getQueryResultItems(&resp.Results) {
		ids = append(ids, item.Id)
	}
	return ids
}

func patternSchema() map[string]cyborgdb.MetadataFieldPolicy {
	return map[string]cyborgdb.MetadataFieldPolicy{
		"color":  {Filterable: boolPtr(true), Pattern: boolPtr(true)},
		"shape":  {Filterable: boolPtr(true), Pattern: boolPtr(false)},
		"hidden": {Filterable: boolPtr(false)},
	}
}

func TestQueryMetadataSchemaRoundTrips(t *testing.T) {
	index := qmIndex(t, patternSchema())
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema, err := index.MetadataSchema(ctx)
	if err != nil {
		t.Fatalf("MetadataSchema failed: %v", err)
	}
	if len(schema) != 3 {
		t.Fatalf("expected 3 fields, got %d: %v", len(schema), schema)
	}
	// GetX has a pointer receiver, so take an addressable copy per field.
	color, shape, hidden := schema["color"], schema["shape"], schema["hidden"]
	if !color.GetPattern() || !color.GetFilterable() {
		t.Errorf("color should be filterable+pattern, got %v", color)
	}
	if shape.GetPattern() {
		t.Errorf("shape should not be a pattern field")
	}
	if hidden.GetFilterable() {
		t.Errorf("hidden should not be filterable")
	}
}

func TestQueryMetadataHappyPaths(t *testing.T) {
	index := qmIndex(t, patternSchema())

	assertSameIDs(t, queryMeta(t, index, cyborgdb.QueryMetadataParams{}),
		append(evenIDs(), oddIDs()...), "no filters")

	assertSameIDs(t, queryMeta(t, index, cyborgdb.QueryMetadataParams{
		Filters: map[string]interface{}{"color": "red"},
	}), evenIDs(), "equality")

	assertSameIDs(t, queryMeta(t, index, cyborgdb.QueryMetadataParams{
		Filters: map[string]interface{}{"loc.city": "paris"},
	}), evenIDs(), "nested dot-path")

	assertSameIDs(t, queryMeta(t, index, cyborgdb.QueryMetadataParams{
		Filters: map[string]interface{}{"color": map[string]interface{}{"$regex": "^r"}},
	}), evenIDs(), "$regex on pattern field")

	assertSameIDs(t, queryMeta(t, index, cyborgdb.QueryMetadataParams{
		Filters: map[string]interface{}{"color": map[string]interface{}{"$contains": "ree"}},
	}), oddIDs(), "$contains on pattern field")

	if ids := queryMeta(t, index, cyborgdb.QueryMetadataParams{
		Filters: map[string]interface{}{"color": "mauve"},
	}); len(ids) != 0 {
		t.Errorf("no-match should be empty, got %v", ids)
	}
}

func TestQueryMetadataOrderingAndPaging(t *testing.T) {
	index := qmIndex(t, patternSchema())
	allRanks := map[string]interface{}{"rank": map[string]interface{}{"$gte": 0}}

	asc := queryMeta(t, index, cyborgdb.QueryMetadataParams{
		Filters: allRanks, OrderBy: "rank", Ascending: true,
	})
	want := []string{}
	for i := 0; i < qmCount; i++ {
		want = append(want, idFor(i))
	}
	if !reflect.DeepEqual(asc, want) {
		t.Errorf("ascending: got %v, want %v", asc, want)
	}

	desc := queryMeta(t, index, cyborgdb.QueryMetadataParams{
		Filters: allRanks, OrderBy: "rank", Ascending: false,
	})
	for i, j := 0, len(want)-1; i < j; i, j = i+1, j-1 {
		want[i], want[j] = want[j], want[i]
	}
	if !reflect.DeepEqual(desc, want) {
		t.Errorf("descending: got %v, want %v", desc, want)
	}

	// TopK applies AFTER the sort, so this is the first 2 of the sorted run.
	if got := (queryMeta(t, index, cyborgdb.QueryMetadataParams{
		Filters: allRanks, OrderBy: "rank", Ascending: true, TopK: 2,
	})); !reflect.DeepEqual(got, []string{idFor(0), idFor(1)}) {
		t.Errorf("top_k after sort: got %v", got)
	}
}

func TestQueryMetadataEnforcesSchema(t *testing.T) {
	index := qmIndex(t, patternSchema())
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	regexOnPlain := map[string]interface{}{"shape": map[string]interface{}{"$regex": "^sq"}}
	if _, err := index.QueryMetadata(ctx, cyborgdb.QueryMetadataParams{Filters: regexOnPlain}); err == nil {
		t.Error("$regex on a non-pattern field should be rejected")
	}
	// ...but the same filter is fine on the vector path, which post-filters.
	assertSameIDs(t, queryIDs(t, index, regexOnPlain), evenIDs(), "$regex via Query")

	nonFilterable := map[string]interface{}{"hidden": "secret"}
	if _, err := index.QueryMetadata(ctx, cyborgdb.QueryMetadataParams{Filters: nonFilterable}); err == nil {
		t.Error("filtering a non-filterable field should be rejected")
	}
	assertSameIDs(t, queryIDs(t, index, nonFilterable), evenIDs(), "non-filterable via Query")

	unsupported := map[string]interface{}{"rank": map[string]interface{}{"$type": "number"}}
	if _, err := index.QueryMetadata(ctx, cyborgdb.QueryMetadataParams{Filters: unsupported}); err == nil {
		t.Error("$type should be rejected")
	}
}

func TestQueryMetadataDefaultPosture(t *testing.T) {
	// No MetadataSchema: every field is filterable, none is a pattern.
	index := qmIndex(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schema, err := index.MetadataSchema(ctx)
	if err != nil {
		t.Fatalf("MetadataSchema failed: %v", err)
	}
	if len(schema) != 0 {
		t.Errorf("default posture should report an empty schema, got %v", schema)
	}

	assertSameIDs(t, queryMeta(t, index, cyborgdb.QueryMetadataParams{
		Filters: map[string]interface{}{"color": "red"},
	}), evenIDs(), "equality without opt-in")

	// Indexed, but with no regex dictionary to resolve against.
	if _, err := index.QueryMetadata(ctx, cyborgdb.QueryMetadataParams{
		Filters: map[string]interface{}{"color": map[string]interface{}{"$regex": "^r"}},
	}); err == nil {
		t.Error("$regex without a pattern field should be rejected")
	}
}
