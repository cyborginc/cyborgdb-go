package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
)

// Recording fake of the SDK: every call lands in calls with its arguments as passed.

type call struct {
	method string
	args   []any
}

type fake struct {
	calls []call
	fail  map[string]error  // method -> error to return
	reply map[string]string // method -> JSON the SDK's response decodes from
}

func (f *fake) record(method string, args ...any) error {
	f.calls = append(f.calls, call{method, args})
	return f.fail[method]
}

// canned decodes the method's canned reply into the SDK's response type, as the SDK would.
func canned[T any](f *fake, method string) *T {
	var v T
	if r, ok := f.reply[method]; ok {
		if err := json.Unmarshal([]byte(r), &v); err != nil {
			panic(err)
		}
	}
	return &v
}

func (f *fake) NewClient(baseURL, apiKey string) (client, error) {
	if err := f.record("NewClient", baseURL, apiKey); err != nil {
		return nil, err
	}
	return fakeClient{f}, nil
}

type fakeClient struct{ f *fake }

func (c fakeClient) ListIndexes(context.Context) ([]string, error) {
	return []string{}, c.f.record("ListIndexes")
}
func (c fakeClient) CreateIndex(_ context.Context, p *cyborgdb.CreateIndexParams) (index, error) {
	if err := c.f.record("CreateIndex", *p); err != nil {
		return nil, err
	}
	return fakeIndex(c), nil
}
func (c fakeClient) LoadIndex(_ context.Context, name string, key []byte) (index, error) {
	if err := c.f.record("LoadIndex", name, key); err != nil {
		return nil, err
	}
	return fakeIndex(c), nil
}
func (c fakeClient) GetHealth(context.Context) (map[string]string, error) {
	return map[string]string{}, c.f.record("GetHealth")
}
func (c fakeClient) GetTrainingStatus(context.Context) (*cyborgdb.TrainingStatus, error) {
	return &cyborgdb.TrainingStatus{}, c.f.record("GetTrainingStatus")
}
func (c fakeClient) ListEmbeddingModels(context.Context) ([]cyborgdb.EmbeddingModel, error) {
	return nil, c.f.record("ListEmbeddingModels")
}

type fakeIndex struct{ f *fake }

func (x fakeIndex) Describe(context.Context) (*cyborgdb.IndexDescription, error) {
	return canned[cyborgdb.IndexDescription](x.f, "Describe"), x.f.record("Describe")
}
func (x fakeIndex) GetNumVectors(context.Context) (int32, error) {
	return 0, x.f.record("GetNumVectors")
}
func (x fakeIndex) Upsert(_ context.Context, in cyborgdb.UpsertInput) error {
	return x.f.record("Upsert", in)
}
func (x fakeIndex) Query(_ context.Context, in cyborgdb.QueryInput) (*cyborgdb.QueryResponse, error) {
	return canned[cyborgdb.QueryResponse](x.f, "Query"), x.f.record("Query", in)
}
func (x fakeIndex) Get(_ context.Context, ids, include []string) (*cyborgdb.GetResponse, error) {
	return canned[cyborgdb.GetResponse](x.f, "Get"), x.f.record("Get", ids, include)
}
func (x fakeIndex) Delete(_ context.Context, ids []string) error {
	return x.f.record("Delete", ids)
}
func (x fakeIndex) ListIDs(context.Context) (*cyborgdb.ListIDsResponse, error) {
	return canned[cyborgdb.ListIDsResponse](x.f, "ListIDs"), x.f.record("ListIDs")
}
func (x fakeIndex) Train(_ context.Context, p cyborgdb.TrainParams) error {
	return x.f.record("Train", p)
}
func (x fakeIndex) QueryMetadata(_ context.Context, p cyborgdb.QueryMetadataParams) (*cyborgdb.QueryMetadataResponse, error) {
	return canned[cyborgdb.QueryMetadataResponse](x.f, "QueryMetadata"), x.f.record("QueryMetadata", p)
}
func (x fakeIndex) DeleteIndex(context.Context) error { return x.f.record("DeleteIndex") }
func (x fakeIndex) CreateUser(_ context.Context, perms []string) (*cyborgdb.CreatedUser, error) {
	return &cyborgdb.CreatedUser{UserID: "u", APIKey: "k"}, x.f.record("CreateUser", perms)
}
func (x fakeIndex) ListUsers(context.Context) ([]cyborgdb.UserInfo, error) {
	return nil, x.f.record("ListUsers")
}
func (x fakeIndex) DeleteUser(_ context.Context, id string) error {
	return x.f.record("DeleteUser", id)
}

// kindError is a typed SDK error with a chosen kind and status.
type kindError struct {
	kind   string
	status int
}

func (e kindError) Error() string                      { return "boom" }
func (e kindError) StatusCode() int                    { return e.status }
func (e kindError) RequestID() string                  { return "" }
func (e kindError) Detail() string                     { return "" }
func (e kindError) RetryAfter() float64                { return 0 }
func (e kindError) Retryable() bool                    { return false }
func (e kindError) Kind() string                       { return e.kind }
func (e kindError) FieldErrors() []cyborgdb.FieldError { return nil }
func (e kindError) Unwrap() error                      { return nil }

var errUntyped = errors.New("untyped")

// Harness

const keyB64 = "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8=" // bytes 0..31

var rootKey = func() []byte {
	b := make([]byte, 32)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}()

// start opens client c0 and index i0 on a fresh fake for the variant, then forgets those calls.
func start(t *testing.T, variant string) (*handler, *fake) {
	t.Helper()
	f := &fake{fail: map[string]error{}, reply: map[string]string{}}
	h := newHandler(f, variant, func(string) string { return "" })
	mustCall(t, h, "client.new", `{"as":"c0","base_url":"http://localhost:8000","api_key":"root"}`)
	mustCall(t, h, "client.create_index", `{"client":"c0","as":"i0","name":"r","key":{"$bytes":"`+keyB64+`"}}`)
	f.calls = nil
	return h, f
}

func call1(t *testing.T, h *handler, method, params string) (any, *rpcError) {
	t.Helper()
	return h.handle(method, json.RawMessage(params))
}

func mustCall(t *testing.T, h *handler, method, params string) any {
	t.Helper()
	r, err := call1(t, h, method, params)
	if err != nil {
		t.Fatalf("%s: %+v", method, err)
	}
	return r
}

// only returns the single recorded call, which must be method.
func only(t *testing.T, f *fake, method string) []any {
	t.Helper()
	if len(f.calls) != 1 || f.calls[0].method != method {
		t.Fatalf("calls = %+v, want one %s", f.calls, method)
	}
	return f.calls[0].args
}

func eq(t *testing.T, what string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s:\n got  %#v\n want %#v", what, got, want)
	}
}

// asJSON compares a result as the driver sees it.
func asJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Cases

func TestClientNewForwardsOnlyWhatWasGiven(t *testing.T) {
	f := &fake{}
	h := newHandler(f, "json", func(string) string { return "" })
	mustCall(t, h, "client.new", `{"as":"c0","base_url":"http://s"}`)
	mustCall(t, h, "client.new", `{"as":"c1","base_url":"http://s","api_key":""}`)
	mustCall(t, h, "client.new", `{"as":"c2","base_url":"not a url","api_key":"k"}`)
	eq(t, "calls", f.calls, []call{
		{"NewClient", []any{"http://s", ""}},
		{"NewClient", []any{"http://s", ""}},
		{"NewClient", []any{"not a url", "k"}},
	})
}

func TestCreateIndexForwardsOnlyWhatWasGiven(t *testing.T) {
	h, f := start(t, "json")
	mustCall(t, h, "client.create_index", `{"client":"c0","as":"i1","name":"s","kms_name":"slot"}`)
	eq(t, "omitted stay unset", only(t, f, "CreateIndex")[0], cyborgdb.CreateIndexParams{IndexName: "s", KmsName: ptr("slot")})

	f.calls = nil
	mustCall(t, h, "client.create_index", `{"client":"c0","as":"i1","name":"s","key":{"$bytes":"AAE="},"dimension":0,"metric":"cosine",
		"storage_precision":"tq4","embedding_model":"m","text_fields":[],"bm25_k1":0,"bm25_b":0.5,
		"metadata_schema":{"t":{"full_text":true},"p":{"filterable":false,"pattern":true}}}`)
	eq(t, "given, zeros and wrong-length key included", only(t, f, "CreateIndex")[0], cyborgdb.CreateIndexParams{
		IndexName: "s", IndexKey: []byte{0, 1}, Dimension: ptr(int32(0)), Metric: ptr("cosine"),
		StoragePrecision: ptr("tq4"), EmbeddingModel: ptr("m"), TextFields: []string{}, Bm25K1: ptr(0.0), Bm25B: ptr(0.5),
		MetadataSchema: map[string]cyborgdb.MetadataFieldPolicy{
			"t": {FullText: ptr(true)}, "p": {Filterable: ptr(false), Pattern: ptr(true)},
		},
	})
}

func TestLoadIndexWithAndWithoutKey(t *testing.T) {
	h, f := start(t, "json")
	mustCall(t, h, "client.load_index", `{"client":"c0","as":"i1","name":"s"}`)
	mustCall(t, h, "client.load_index", `{"client":"c0","as":"i1","name":"s","key":{"$bytes":"`+keyB64+`"}}`)
	eq(t, "calls", f.calls, []call{{"LoadIndex", []any{"s", []byte(nil)}}, {"LoadIndex", []any{"s", rootKey}}})
}

func TestQueryJSONKeepsTheShapeAndZeros(t *testing.T) {
	h, f := start(t, "json")
	f.reply["Query"] = `{"results":[]}`
	mustCall(t, h, "index.query", `{"index":"i0","vectors":{"$f32":[1,2]}}`)
	eq(t, "one vector, nothing else set", only(t, f, "Query")[0], cyborgdb.QueryParams{QueryVector: []float32{1, 2}})

	f.calls = nil
	mustCall(t, h, "index.query", `{"index":"i0","vectors":{"$f32":[[1,2],[3]]},"top_k":0,"n_probes":-1,"greedy":false,
		"rerank_mult":0,"window_mult":0,"alpha":0,"rrf_k":0,"include":[],"text":"","text_fields":[],"text_field_weights":[],
		"require_all_terms":false,"filters":{"n":{"$i64":"9007199254740993"}}}`)
	eq(t, "rows (ragged too), zeros and empties as given", only(t, f, "Query")[0], cyborgdb.QueryParams{
		BatchQueryVectors: [][]float32{{1, 2}, {3}}, TopK: 0, NProbes: ptr(int32(-1)), Greedy: ptr(false),
		RerankMult: ptr(int32(0)), WindowMult: ptr(int32(0)), Alpha: ptr(0.0), RrfK: ptr(0.0), Include: []string{},
		Text: ptr(""), TextFields: []string{}, TextFieldWeights: []float32{}, RequireAllTerms: ptr(false),
		Filters: map[string]any{"n": int64(9007199254740993)},
	})

	f.calls = nil
	mustCall(t, h, "index.query", `{"index":"i0","contents":"hello"}`)
	eq(t, "contents", only(t, f, "Query")[0], cyborgdb.QueryParams{QueryContents: ptr("hello")})
}

func TestQueryBinaryTakesRows(t *testing.T) {
	h, f := start(t, "binary")
	f.reply["Query"] = `{"results":[[]]}`
	mustCall(t, h, "index.query", `{"index":"i0","vectors":{"$f32":[1,2]},"top_k":3}`)
	eq(t, "one vector is one row", only(t, f, "Query")[0], cyborgdb.BinaryQueryParams{QueryVectors: [][]float32{{1, 2}}, TopK: 3})
}

func TestQueryResultsAreOneListPerRowWithFieldsAsReturned(t *testing.T) {
	h, f := start(t, "json")
	f.reply["Query"] = `{"results":[{"id":"a","distance":0.5,"metadata":{"k":1}},{"id":"b","score":2}]}`
	got := asJSON(t, mustCall(t, h, "index.query", `{"index":"i0","vectors":{"$f32":[1,2]}}`))
	eq(t, "flat single query", got, `[[{"distance":0.5,"id":"a","metadata":{"k":1}},{"id":"b","score":2}]]`)

	f.reply["Query"] = `{"results":[[{"id":"a"}],[]]}`
	got = asJSON(t, mustCall(t, h, "index.query", `{"index":"i0","vectors":{"$f32":[[1,2],[3,4]]}}`))
	eq(t, "batch", got, `[[{"id":"a"}],[]]`)
}

func TestGetIncludeOmittedAndEmpty(t *testing.T) {
	h, f := start(t, "json")
	f.reply["Get"] = `{"results":[{"id":"a","contents":"hi","vector":[1.5],"metadata":{}},{"id":"b","contents":null}]}`
	got := asJSON(t, mustCall(t, h, "index.get", `{"index":"i0","ids":["a","b"]}`))
	mustCall(t, h, "index.get", `{"index":"i0","ids":["a"],"include":[]}`)
	eq(t, "omitted is nil", f.calls[0].args[1], []string(nil))
	eq(t, "empty stays empty", f.calls[1].args[1], []string{})
	eq(t, "rows", got, `[{"contents":{"$bytes":"aGk="},"id":"a","metadata":{},"vector":{"$f32":[1.5]}},{"contents":null,"id":"b"}]`)
}

func TestUpsertJSONItemsAndParallelForm(t *testing.T) {
	h, f := start(t, "json")
	mustCall(t, h, "index.upsert", `{"index":"i0","items":[
		{"id":"a","vector":{"$f32":[1,2]},"metadata":{"at":{"$datetime":"2026-01-02T03:04:05Z"}}},
		{"id":"b","contents":"text"},
		{"id":"c","contents":{"$bytes":"AAE="},"metadata":null},
		{"id":"d","contents":null}]}`)
	items := only(t, f, "Upsert")[0].(cyborgdb.VectorItems)
	text, raw := "text", []byte{0, 1}
	want := make(cyborgdb.VectorItems, 4)
	want[0].Id, want[0].Vector = "a", []float32{1, 2}
	want[0].Metadata = map[string]any{"at": time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}
	want[1].Id = "b"
	want[1].SetContents(cyborgdb.Contents{String: &text})
	want[2].Id = "c"
	want[2].SetContents(cyborgdb.Contents{Bytes: &raw})
	want[3].Id = "d"
	want[3].SetContentsNil()
	eq(t, "items", items, want)

	f.calls = nil
	mustCall(t, h, "index.upsert", `{"index":"i0","ids":["a","b"],"vectors":{"$f32":[[1,2],[3]]},"metadata":[{"x":1},null]}`)
	want = make(cyborgdb.VectorItems, 2)
	want[0].Id, want[0].Vector, want[0].Metadata = "a", []float32{1, 2}, map[string]any{"x": json.Number("1")}
	want[1].Id, want[1].Vector = "b", []float32{3}
	eq(t, "parallel form as items", only(t, f, "Upsert")[0], want)
}

func TestUpsertBinaryPassesColumnsAsSent(t *testing.T) {
	h, f := start(t, "binary")
	mustCall(t, h, "index.upsert", `{"index":"i0","ids":["a","b"],"vectors":{"$f32":[[1,2]]},"metadata":[{"x":1},null,{}]}`)
	eq(t, "counts left for the SDK to check", only(t, f, "Upsert")[0], cyborgdb.BinaryUpsertParams{
		IDs: []string{"a", "b"}, Vectors: [][]float32{{1, 2}},
		Metadata: []map[string]any{{"x": json.Number("1")}, nil, {}},
	})

	f.calls = nil
	mustCall(t, h, "index.upsert", `{"index":"i0","items":[{"id":"a","vector":{"$f32":[1]},"contents":"t"},{"id":"b","vector":{"$f32":[2]}}]}`)
	eq(t, "items as columns", only(t, f, "Upsert")[0], cyborgdb.BinaryUpsertParams{
		IDs: []string{"a", "b"}, Vectors: [][]float32{{1}, {2}}, Contents: []string{"t", ""},
	})
}

func TestQueryMetadataForwardsOnlyWhatWasGiven(t *testing.T) {
	h, f := start(t, "json")
	f.reply["QueryMetadata"] = `{"results":[{"id":"a"},{"id":"b","score":1.5}],"ids":["a","b"],"count":2}`
	got := asJSON(t, mustCall(t, h, "index.query_metadata", `{"index":"i0"}`))
	eq(t, "omitted stay unset", only(t, f, "QueryMetadata")[0], cyborgdb.QueryMetadataParams{})
	eq(t, "rows", got, `[{"id":"a"},{"id":"b","score":1.5}]`)

	f.calls = nil
	mustCall(t, h, "index.query_metadata", `{"index":"i0","filters":{},"top_k":0,"order_by":"","ascending":false}`)
	eq(t, "zeros as given", only(t, f, "QueryMetadata")[0], cyborgdb.QueryMetadataParams{
		Filters: map[string]any{}, OrderBy: "", Ascending: ptr(false),
	})
}

func TestTrainForwardsOnlyWhatWasGiven(t *testing.T) {
	h, f := start(t, "json")
	mustCall(t, h, "index.train", `{"index":"i0","n_lists":0,"tolerance":0}`)
	eq(t, "train", only(t, f, "Train")[0], cyborgdb.TrainParams{NLists: ptr(int32(0)), Tolerance: ptr(0.0)})
}

func TestUsers(t *testing.T) {
	h, f := start(t, "json")
	got := asJSON(t, mustCall(t, h, "index.create_user", `{"index":"i0","permissions":["read"]}`))
	eq(t, "create result", got, `{"user_id":"u","api_key":"k"}`)
	mustCall(t, h, "index.delete_user", `{"index":"i0","user_id":"abc"}`)
	mustCall(t, h, "index.delete_user", `{"index":"i0","user_id":{"$bytes":"AQI="}}`)
	eq(t, "calls", f.calls, []call{
		{"CreateUser", []any{[]string{"read"}}}, {"DeleteUser", []any{"abc"}}, {"DeleteUser", []any{"0102"}},
	})
}

func TestDescribeRenamesIndexName(t *testing.T) {
	h, f := start(t, "json")
	f.reply["Describe"] = `{"index_name":"r","dimension":4,"metadata_schema":{"t":{"full_text":true}},"bm25":null}`
	got := asJSON(t, mustCall(t, h, "index.describe", `{"index":"i0"}`))
	if !strings.Contains(got, `"name":"r"`) || strings.Contains(got, "index_name") || !strings.Contains(got, `"metadata_schema":{"t":{"full_text":true}}`) {
		t.Errorf("describe: %s", got)
	}
}

func TestNativeErrorsKeepTheirKind(t *testing.T) {
	h, f := start(t, "json")
	f.fail["Delete"] = kindError{kind: "wrong_key", status: 401}
	_, err := call1(t, h, "index.delete", `{"index":"i0","ids":["a"]}`)
	if err == nil || err.Code != codeOpFailed || err.Data.Kind != "wrong_key" || err.Data.Native != "main.kindError 401" {
		t.Errorf("typed: got %+v", err)
	}
	f.fail["Delete"] = errUntyped
	_, err = call1(t, h, "index.delete", `{"index":"i0","ids":["a"]}`)
	if err == nil || err.Code != codeOpFailed || err.Data.Kind != "internal" {
		t.Errorf("untyped: got %+v", err)
	}
}

func TestUnexpressibleInputIsUnsupportedAndNeverCalled(t *testing.T) {
	for name, c := range map[string][3]string{
		"per-call key":               {"json", "index.num_vectors", `{"index":"i0","key":{"$bytes":"` + keyB64 + `"}}`},
		"load as user":               {"json", "client.load_index", `{"client":"c0","as":"i1","name":"s","user_id":{"$bytes":"AA=="}}`},
		"core storage":               {"json", "client.new", `{"as":"c9","base_url":"http://s","storage":{"kind":"disk"}}`},
		"item without id":            {"json", "index.upsert", `{"index":"i0","items":[{"vector":{"$f32":[1]}}]}`},
		"unknown item field":         {"json", "index.upsert", `{"index":"i0","items":[{"id":"a","extra":1}]}`},
		"non-object metadata":        {"json", "index.upsert", `{"index":"i0","items":[{"id":"a","metadata":[1]}]}`},
		"unpairable parallel lists":  {"json", "index.upsert", `{"index":"i0","ids":["a"],"vectors":{"$f32":[[1],[2]]}}`},
		"schema key the SDK lacks":   {"json", "client.create_index", `{"client":"c0","as":"i1","name":"s","metadata_schema":{"f":{"sortable":true}}}`},
		"filter string":              {"json", "index.query", `{"index":"i0","vectors":{"$f32":[1]},"filters":"{}"}`},
		"contents list":              {"json", "index.query", `{"index":"i0","contents":["a","b"]}`},
		"order_by dict":              {"json", "index.query_metadata", `{"index":"i0","order_by":{"f":-1}}`},
		"top_k beyond int32":         {"json", "index.query", `{"index":"i0","vectors":{"$f32":[1]},"top_k":4294967296}`},
		"binary query contents":      {"binary", "index.query", `{"index":"i0","contents":"a"}`},
		"binary item without vector": {"binary", "index.upsert", `{"index":"i0","items":[{"id":"a","contents":"t"}]}`},
		"binary bytes contents":      {"binary", "index.upsert", `{"index":"i0","items":[{"id":"a","vector":{"$f32":[1]},"contents":{"$bytes":"AA=="}}]}`},
		"binary empty contents":      {"binary", "index.upsert", `{"index":"i0","items":[{"id":"a","vector":{"$f32":[1]},"contents":""}]}`},
	} {
		h, f := start(t, c[0])
		_, err := call1(t, h, c[1], c[2])
		if err == nil || err.Code != codeUnsupported {
			t.Errorf("%s: got %+v, want unsupported", name, err)
		}
		eq(t, name+": no SDK calls", len(f.calls), 0)
	}
}

func TestAdapterBugsAreNotProductErrors(t *testing.T) {
	h, _ := start(t, "json")
	if _, err := call1(t, h, "index.get", `{"index":"nope","ids":[]}`); err == nil || err.Code != codeBadParams {
		t.Errorf("unknown handle: %+v", err)
	}
	if _, err := call1(t, h, "client.kms_push", `{"client":"c0"}`); err == nil || err.Code != codeUnsupported {
		t.Errorf("core-only op: %+v", err)
	}
}

func TestHelloCapabilitiesFollowTheVariant(t *testing.T) {
	caps := func(variant string, env map[string]string) []string {
		h := newHandler(&fake{}, variant, func(k string) string { return env[k] })
		return h.hello().(map[string]any)["capabilities"].([]string)
	}
	has := func(list []string, c string) bool {
		for _, x := range list {
			if x == c {
				return true
			}
		}
		return false
	}
	j, b := caps("json", nil), caps("binary", map[string]string{"CYBORGDB_KMS_NAME_REAL": "slot"})
	eq(t, "json", []bool{has(j, "ragged_batches"), has(j, "parallel_metadata"), has(j, "kms_provider")}, []bool{true, false, false})
	eq(t, "binary", []bool{has(b, "ragged_batches"), has(b, "parallel_metadata"), has(b, "kms_provider")}, []bool{false, true, true})
}

// writeTensor writes junk then values as little-endian float32, and returns the file and where the values start.
func writeTensor(t *testing.T, values []float32) (path string, offset int) {
	t.Helper()
	junk := []byte{0xde, 0xad, 0xbe, 0xef, 0, 0, 0, 0} // so a wrong offset reads the wrong numbers
	buf := append([]byte(nil), junk...)
	for _, v := range values {
		buf = binary.LittleEndian.AppendUint32(buf, math.Float32bits(v))
	}
	path = filepath.Join(t.TempDir(), "tensor.f32")
	if err := os.WriteFile(path, buf, 0o600); err != nil {
		t.Fatal(err)
	}
	return path, len(junk)
}

func tensorJSON(path string, offset int, dtype string, shape ...int) string {
	b, _ := json.Marshal(map[string]any{"$tensor": map[string]any{"path": path, "dtype": dtype, "offset": offset, "shape": shape}})
	return string(b)
}

// Datasets send vectors as $tensor files and ids as $range; a bug in either would break every dataset-backed spec.
func TestTensorsAndRangesDecode(t *testing.T) {
	values := []float32{1.5, -2, 0, 3.25, float32(math.Inf(1)), -0.5}
	path, off := writeTensor(t, values)

	h, f := start(t, "json")
	f.reply["Query"] = `{"results":[[],[]]}`
	mustCall(t, h, "index.query", `{"index":"i0","vectors":`+tensorJSON(path, off, "f32", 2, 3)+`}`)
	eq(t, "2-D tensor is rows, so a batch query", only(t, f, "Query")[0], cyborgdb.QueryParams{
		BatchQueryVectors: [][]float32{{1.5, -2, 0}, {3.25, float32(math.Inf(1)), -0.5}},
	})

	f.calls = nil
	f.reply["Query"] = `{"results":[]}`
	mustCall(t, h, "index.query", `{"index":"i0","vectors":`+tensorJSON(path, off+4*3, "f32", 3)+`}`)
	eq(t, "1-D tensor at an offset is one vector", only(t, f, "Query")[0], cyborgdb.QueryParams{
		QueryVector: []float32{3.25, float32(math.Inf(1)), -0.5},
	})

	h, f = start(t, "binary")
	mustCall(t, h, "index.upsert", `{"index":"i0","ids":{"$range":[7,9]},"vectors":`+tensorJSON(path, off, "f32", 2, 3)+`}`)
	eq(t, "$range ids with tensor rows", only(t, f, "Upsert")[0], cyborgdb.BinaryUpsertParams{
		IDs: []string{"7", "8"}, Vectors: [][]float32{{1.5, -2, 0}, {3.25, float32(math.Inf(1)), -0.5}},
	})
	eq(t, "an empty $range is no ids", stringsOf(json.RawMessage(`{"$range":[5,5]}`), "ids"), []string{})
}

// A $tensor that doesn't fit its file, or isn't f32, is refused before any memory is set aside for it.
func TestBadTensorsAreRefusedNotRead(t *testing.T) {
	path, off := writeTensor(t, []float32{1, 2, 3, 4})
	for name, c := range map[string]struct {
		vectors string
		code    int
	}{
		"rows past the end":     {tensorJSON(path, off, "f32", 2, 3), codeBadParams},
		"offset past the end":   {tensorJSON(path, off+1000, "f32", 1), codeBadParams},
		"a huge shape":          {tensorJSON(path, off, "f32", 1<<40, 1<<30), codeBadParams},
		"a billion empty rows":  {tensorJSON(path, off, "f32", 1_000_000_000, 0), codeBadParams},
		"not f32":               {tensorJSON(path, off, "f16", 4), codeUnsupported},
		"a shape of rank three": {tensorJSON(path, off, "f32", 1, 2, 2), codeBadParams},
	} {
		h, f := start(t, "json")
		_, err := call1(t, h, "index.query", `{"index":"i0","vectors":`+c.vectors+`}`)
		if err == nil || err.Code != c.code {
			t.Errorf("%s: got %+v, want code %d", name, err, c.code)
		}
		eq(t, name+": no SDK calls", len(f.calls), 0)
	}
}
