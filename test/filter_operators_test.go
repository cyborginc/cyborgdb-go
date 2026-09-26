package test

import (
	"context"
	"testing"
	"time"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
)

// Every documented filter operator, on both read paths.
//
// Mirrors py tests/test_query_metadata.py TestFilterOperators and js
// query_metadata.test.ts "filter operators on both read paths". Four operators
// were exercised anywhere before this.

const opDim = 8

// o2 and o4 omit `author` entirely, and o3's `tags` is empty — the two cases
// that make operator semantics ambiguous.
type opRow struct {
	id     string
	color  string
	rank   int
	tags   []string
	author string // "" means the field is omitted entirely
}

var opRows = []opRow{
	{"o0", "red", 0, []string{"design", "search"}, "ada"},
	{"o1", "green", 10, []string{"design"}, "bob"},
	{"o2", "blue", 20, []string{"search"}, ""},
	{"o3", "red", 30, []string{}, "ada"},
	{"o4", "green", 40, []string{"design", "search", "ml"}, ""},
}

var allOpIDs = []string{"o0", "o1", "o2", "o3", "o4"}

// Each expected answer is a proper subset of the corpus, so a filter that
// silently matched everything or nothing fails rather than passing by luck.
type opCase struct {
	name    string
	filters map[string]interface{}
	want    []string
}

func operatorCases() []opCase {
	return []opCase{
		{"$eq", map[string]interface{}{"color": map[string]interface{}{"$eq": "red"}}, []string{"o0", "o3"}},
		{"$ne", map[string]interface{}{"color": map[string]interface{}{"$ne": "red"}}, []string{"o1", "o2", "o4"}},
		{"$in", map[string]interface{}{"color": map[string]interface{}{"$in": []string{"red", "blue"}}}, []string{"o0", "o2", "o3"}},
		{"$nin", map[string]interface{}{"color": map[string]interface{}{"$nin": []string{"red"}}}, []string{"o1", "o2", "o4"}},
		{"$gt", map[string]interface{}{"rank": map[string]interface{}{"$gt": 20}}, []string{"o3", "o4"}},
		{"$gte", map[string]interface{}{"rank": map[string]interface{}{"$gte": 20}}, []string{"o2", "o3", "o4"}},
		{"$lt", map[string]interface{}{"rank": map[string]interface{}{"$lt": 20}}, []string{"o0", "o1"}},
		{"$lte", map[string]interface{}{"rank": map[string]interface{}{"$lte": 20}}, []string{"o0", "o1", "o2"}},
		{"$exists true", map[string]interface{}{"author": map[string]interface{}{"$exists": true}}, []string{"o0", "o1", "o3"}},
		{"$exists false", map[string]interface{}{"author": map[string]interface{}{"$exists": false}}, []string{"o2", "o4"}},
		{"$and", map[string]interface{}{"$and": []interface{}{
			map[string]interface{}{"color": "red"},
			map[string]interface{}{"rank": map[string]interface{}{"$gte": 30}},
		}}, []string{"o3"}},
		{"$or", map[string]interface{}{"$or": []interface{}{
			map[string]interface{}{"color": "blue"},
			map[string]interface{}{"rank": map[string]interface{}{"$lt": 10}},
		}}, []string{"o0", "o2"}},
		{"$nor", map[string]interface{}{"$nor": []interface{}{
			map[string]interface{}{"color": "red"},
			map[string]interface{}{"color": "green"},
		}}, []string{"o2"}},
		// `$not` is deliberately absent — openapi.json documents it, but the
		// engine rejects it on both read paths. See cyborgdb-core#2395.
		{"$regex", map[string]interface{}{"color": map[string]interface{}{"$regex": "^r"}}, []string{"o0", "o3"}},
		{"$contains", map[string]interface{}{"color": map[string]interface{}{"$contains": "ree"}}, []string{"o1", "o4"}},
	}
}

func operatorIndex(t *testing.T) *cyborgdb.EncryptedIndex {
	t.Helper()
	client := newIsolatedClient(t)
	dim := int32(opDim)
	metric := "euclidean"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	index, err := client.CreateIndex(ctx, &cyborgdb.CreateIndexParams{
		IndexName: generateUniqueName("operators_"),
		IndexKey:  generateRandomKey(),
		Dimension: &dim,
		Metric:    &metric,
		MetadataSchema: map[string]cyborgdb.MetadataFieldPolicy{
			"color":  {Filterable: boolPtr(true), Pattern: boolPtr(true)},
			"rank":   {Filterable: boolPtr(true)},
			"tags":   {Filterable: boolPtr(true)},
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

	vectors := generateRandomVectors(len(opRows), opDim)
	ids := make([]string, len(opRows))
	metadata := make([]map[string]interface{}, len(opRows))
	for i, row := range opRows {
		ids[i] = row.id
		m := map[string]interface{}{"color": row.color, "rank": row.rank, "tags": row.tags}
		// Omitted rather than empty: these exercise absence.
		if row.author != "" {
			m["author"] = row.author
		}
		metadata[i] = m
	}
	if err := index.UpsertVectors(ctx, ids, vectors, metadata); err != nil {
		t.Fatalf("UpsertVectors failed: %v", err)
	}
	waitForIDs(t, index, ids)
	return index
}

// opVectorIDs runs the same filter through the vector path.
func opVectorIDs(t *testing.T, index *cyborgdb.EncryptedIndex, filters map[string]interface{}) []string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := index.Query(ctx, cyborgdb.QueryParams{
		QueryVector: generateRandomVectors(1, opDim)[0],
		TopK:        int32(len(allOpIDs)),
		Filters:     filters,
	})
	if err != nil {
		t.Fatalf("Query(%v) failed: %v", filters, err)
	}
	items := getQueryResultItems(&resp.Results)
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.Id)
	}
	return ids
}

func TestFilterOperatorsOnBothPaths(t *testing.T) {
	index := operatorIndex(t)
	for _, tc := range operatorCases() {
		t.Run(tc.name, func(t *testing.T) {
			meta := queryMeta(t, index, cyborgdb.QueryMetadataParams{Filters: tc.filters})
			assertSameIDs(t, meta, tc.want, tc.name+" via QueryMetadata")

			// Query post-filters over decrypted metadata rather than resolving
			// from the index; the answers must still match.
			vector := opVectorIDs(t, index, tc.filters)
			assertSameIDs(t, vector, tc.want, tc.name+" via Query")

			// Anchored as well as compared: a bug in the shared filter parser
			// would break both paths identically and slip past an
			// agreement-only check.
			assertSameIDs(t, meta, vector, tc.name+" paths agree")
		})
	}
}

func TestFilterMissingFieldSemantics(t *testing.T) {
	index := operatorIndex(t)

	// `$ne` drops documents lacking the field, `$nin` keeps them. Both are
	// defensible; the point is that the contract is pinned, not inferred.
	assertSameIDs(t,
		queryMeta(t, index, cyborgdb.QueryMetadataParams{
			Filters: map[string]interface{}{"author": map[string]interface{}{"$ne": "ada"}},
		}),
		[]string{"o1"}, "$ne excludes a missing field")

	assertSameIDs(t,
		queryMeta(t, index, cyborgdb.QueryMetadataParams{
			Filters: map[string]interface{}{"author": map[string]interface{}{"$nin": []string{"ada"}}},
		}),
		[]string{"o1", "o2", "o4"}, "$nin includes a missing field")

	assertSameIDs(t,
		queryMeta(t, index, cyborgdb.QueryMetadataParams{
			Filters: map[string]interface{}{"$nor": []interface{}{
				map[string]interface{}{"author": "ada"},
			}},
		}),
		[]string{"o1", "o2", "o4"}, "$nor includes a missing field")
}

func TestFilterArrayFieldSemantics(t *testing.T) {
	index := operatorIndex(t)

	assertSameIDs(t,
		queryMeta(t, index, cyborgdb.QueryMetadataParams{
			Filters: map[string]interface{}{"tags": "design"},
		}),
		[]string{"o0", "o1", "o4"}, "a bare value means contains")

	assertSameIDs(t,
		queryMeta(t, index, cyborgdb.QueryMetadataParams{
			Filters: map[string]interface{}{"tags": map[string]interface{}{"$in": []string{"ml", "search"}}},
		}),
		[]string{"o0", "o2", "o4"}, "$in means any-of")

	// "Contains all" has no dedicated operator; it is $and of two memberships.
	assertSameIDs(t,
		queryMeta(t, index, cyborgdb.QueryMetadataParams{
			Filters: map[string]interface{}{"$and": []interface{}{
				map[string]interface{}{"tags": "design"},
				map[string]interface{}{"tags": "search"},
			}},
		}),
		[]string{"o0", "o4"}, "has-all via $and")

	// o3's tags are empty, so it can never satisfy a membership condition.
	for _, id := range queryMeta(t, index, cyborgdb.QueryMetadataParams{
		Filters: map[string]interface{}{"tags": "design"},
	}) {
		if id == "o3" {
			t.Error("an empty array should match no membership condition")
		}
	}
}

func TestFilterDegenerateOperands(t *testing.T) {
	index := operatorIndex(t)

	assertSameIDs(t,
		queryMeta(t, index, cyborgdb.QueryMetadataParams{Filters: map[string]interface{}{}}),
		allOpIDs, "an empty filter matches everything")

	assertSameIDs(t,
		queryMeta(t, index, cyborgdb.QueryMetadataParams{
			Filters: map[string]interface{}{"color": map[string]interface{}{"$in": []string{}}},
		}),
		[]string{}, "an empty $in list matches nothing")

	assertSameIDs(t,
		queryMeta(t, index, cyborgdb.QueryMetadataParams{
			Filters: map[string]interface{}{"color": map[string]interface{}{"$nin": []string{}}},
		}),
		allOpIDs, "an empty $nin list matches everything")

	// $and over nothing is vacuously true, $or vacuously false.
	assertSameIDs(t,
		queryMeta(t, index, cyborgdb.QueryMetadataParams{
			Filters: map[string]interface{}{"$and": []interface{}{}},
		}),
		allOpIDs, "$and over nothing")

	assertSameIDs(t,
		queryMeta(t, index, cyborgdb.QueryMetadataParams{
			Filters: map[string]interface{}{"$or": []interface{}{}},
		}),
		[]string{}, "$or over nothing")
}

func TestFilterIntAndFloatAreTheSameKey(t *testing.T) {
	index := operatorIndex(t)

	// Anchored to the expected answer, not just compared to each other: two
	// empty results would otherwise satisfy the comparison.
	assertSameIDs(t,
		queryMeta(t, index, cyborgdb.QueryMetadataParams{
			Filters: map[string]interface{}{"rank": 20},
		}),
		[]string{"o2"}, "int 20")

	assertSameIDs(t,
		queryMeta(t, index, cyborgdb.QueryMetadataParams{
			Filters: map[string]interface{}{"rank": 20.0},
		}),
		[]string{"o2"}, "float 20.0")

	assertSameIDs(t,
		queryMeta(t, index, cyborgdb.QueryMetadataParams{
			Filters: map[string]interface{}{"rank": map[string]interface{}{"$gte": 20.0}},
		}),
		[]string{"o2", "o3", "o4"}, "float range bound")
}

func TestFilterNotOperatorIsDocumentedButRejected(t *testing.T) {
	// KNOWN BUG — fails today. cyborgdb-core#2395: the engine rejects `$not` on
	// both read paths although openapi.json documents it.
	index := operatorIndex(t)
	filters := map[string]interface{}{
		"color": map[string]interface{}{"$not": map[string]interface{}{"$eq": "red"}},
	}
	assertSameIDs(t,
		queryMeta(t, index, cyborgdb.QueryMetadataParams{Filters: filters}),
		[]string{"o1", "o2", "o4"}, "$not via QueryMetadata")
	assertSameIDs(t, opVectorIDs(t, index, filters),
		[]string{"o1", "o2", "o4"}, "$not via Query")
}
