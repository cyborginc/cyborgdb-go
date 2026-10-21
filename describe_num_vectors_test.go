package cyborgdb

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"
)

// recordedRequest is what recordServer saw of the last request.
type recordedRequest struct {
	Method string
	Path   string
	Body   map[string]interface{}
}

// recordServer returns a test server that answers every request with code and
// body, and records the last request's method, path, and JSON body.
func recordServer(t *testing.T, code int, body string) (*httptest.Server, *recordedRequest) {
	t.Helper()
	rec := &recordedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.Method, rec.Path, rec.Body = r.Method, r.URL.Path, nil
		if raw, _ := io.ReadAll(r.Body); len(raw) > 0 {
			_ = json.Unmarshal(raw, &rec.Body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

// keyedIndex returns an EncryptedIndex named "idx" with an SDK-held key, bound
// to a client for baseURL.
func keyedIndex(t *testing.T, baseURL string) (*EncryptedIndex, string) {
	t.Helper()
	client, err := NewClient(baseURL, "test-key")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	key, err := keyBytesToHex(make([]byte, 32))
	if err != nil {
		t.Fatalf("keyBytesToHex: %v", err)
	}
	return &EncryptedIndex{indexName: "idx", indexKey: key, client: client.internal}, *key
}

const describeBody = `{
	"index_name": "idx",
	"is_trained": true,
	"index_type": "disk_ivf",
	"training_state": "trained",
	"embedding_model": "all-MiniLM-L6-v2",
	"storage_precision": "float16",
	"dimension": 384,
	"metric": "cosine",
	"n_lists": 64,
	"metadata_schema": {"$default": {"filterable": false}, "title": {"filterable": true, "pattern": true}},
	"bm25": {"k1": 1.2, "b": 0.75, "analyzer_version": "v1"}
}`

func TestDescribeMapsEveryField(t *testing.T) {
	srv, _ := recordServer(t, 200, describeBody)
	index, _ := keyedIndex(t, srv.URL)

	got, err := index.Describe(context.Background())
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if got.IndexName != "idx" || got.IndexType != "disk_ivf" || got.Dimension != 384 ||
		got.Metric != "cosine" || got.StoragePrecision != "float16" || got.NLists != 64 ||
		got.TrainingState != "trained" {
		t.Errorf("scalar fields = %+v", got)
	}
	if got.EmbeddingModel == nil || *got.EmbeddingModel != "all-MiniLM-L6-v2" {
		t.Errorf("EmbeddingModel = %v, want all-MiniLM-L6-v2", got.EmbeddingModel)
	}
	if got.BM25 == nil || got.BM25.GetK1() != 1.2 || got.BM25.GetB() != 0.75 {
		t.Errorf("BM25 = %+v, want k1=1.2 b=0.75", got.BM25)
	}
	title := got.MetadataSchema["title"]
	if !title.GetFilterable() || !title.GetPattern() {
		t.Errorf("MetadataSchema[title] = %+v, want filterable and pattern", title)
	}
}

// TestDescribeDefaultPolicyStaysAsSent: the "$default" entry comes back as the
// service sent it, with no pattern/full_text keys added.
func TestDescribeDefaultPolicyStaysAsSent(t *testing.T) {
	srv, _ := recordServer(t, 200, describeBody)
	index, _ := keyedIndex(t, srv.URL)

	got, err := index.Describe(context.Background())
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	raw, err := json.Marshal(got.MetadataSchema["$default"])
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(raw) != `{"filterable":false}` {
		t.Errorf("$default = %s, want {\"filterable\":false}", raw)
	}
}

func TestDescribeOptionalFieldsAbsent(t *testing.T) {
	srv, _ := recordServer(t, 200, `{
		"index_name": "idx", "is_trained": false, "index_type": "disk_ivf",
		"training_state": "untrained", "storage_precision": "float32",
		"dimension": 0, "metric": "euclidean", "n_lists": 1, "bm25": null
	}`)
	index, _ := keyedIndex(t, srv.URL)

	got, err := index.Describe(context.Background())
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if got.EmbeddingModel != nil {
		t.Errorf("EmbeddingModel = %q, want nil", *got.EmbeddingModel)
	}
	if got.BM25 != nil {
		t.Errorf("BM25 = %+v, want nil", got.BM25)
	}
	if got.MetadataSchema == nil || len(got.MetadataSchema) != 0 {
		t.Errorf("MetadataSchema = %v, want empty non-nil map", got.MetadataSchema)
	}

	// Serialized, the description carries exactly the contract's keys: no
	// is_trained, and no embedding_model when the index has none.
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var keys map[string]interface{}
	_ = json.Unmarshal(raw, &keys)
	want := []string{"bm25", "dimension", "index_name", "index_type", "metadata_schema", "metric", "n_lists", "storage_precision", "training_state"}
	if gotKeys := sortedKeys(keys); !reflect.DeepEqual(gotKeys, want) {
		t.Errorf("serialized keys = %v, want %v", gotKeys, want)
	}
}

func TestDescribePassesTrainingStateThrough(t *testing.T) {
	for _, state := range []string{"untrained", "training", "trained"} {
		t.Run(state, func(t *testing.T) {
			srv, _ := recordServer(t, 200, `{
				"index_name": "idx", "is_trained": false, "index_type": "disk_ivf",
				"training_state": "`+state+`", "storage_precision": "float32",
				"dimension": 8, "metric": "euclidean", "n_lists": 1
			}`)
			index, _ := keyedIndex(t, srv.URL)

			got, err := index.Describe(context.Background())
			if err != nil {
				t.Fatalf("Describe: %v", err)
			}
			if got.TrainingState != state {
				t.Errorf("TrainingState = %q, want %q", got.TrainingState, state)
			}
		})
	}
}

func TestDescribeSendsIndexNameAndKey(t *testing.T) {
	srv, rec := recordServer(t, 200, describeBody)
	index, key := keyedIndex(t, srv.URL)

	if _, err := index.Describe(context.Background()); err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/v1/indexes/describe" {
		t.Errorf("request = %s %s, want POST /v1/indexes/describe", rec.Method, rec.Path)
	}
	if rec.Body["index_name"] != "idx" || rec.Body["index_key"] != key {
		t.Errorf("request body = %v, want index_name=idx and the index key", rec.Body)
	}
}

func TestDescribeReturnsTypedError(t *testing.T) {
	srv, _ := recordServer(t, 404, `{"detail":"index not found"}`)
	index, _ := keyedIndex(t, srv.URL)

	_, err := index.Describe(context.Background())
	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("Describe on 404 returned %T, want *NotFoundError", err)
	}
}

func TestGetNumVectorsReturnsResult(t *testing.T) {
	srv, rec := recordServer(t, 200, `{"result": 42}`)
	index, key := keyedIndex(t, srv.URL)

	count, err := index.GetNumVectors(context.Background())
	if err != nil {
		t.Fatalf("GetNumVectors: %v", err)
	}
	if count != 42 {
		t.Errorf("GetNumVectors = %d, want 42", count)
	}
	if rec.Method != http.MethodPost || rec.Path != "/v1/vectors/num_vectors" {
		t.Errorf("request = %s %s, want POST /v1/vectors/num_vectors", rec.Method, rec.Path)
	}
	if rec.Body["index_name"] != "idx" || rec.Body["index_key"] != key {
		t.Errorf("request body = %v, want index_name=idx and the index key", rec.Body)
	}
}

func TestGetNumVectorsReturnsTypedError(t *testing.T) {
	srv, _ := recordServer(t, 401, `{"detail":"invalid api key"}`)
	index, _ := keyedIndex(t, srv.URL)

	_, err := index.GetNumVectors(context.Background())
	var auth *AuthenticationError
	if !errors.As(err, &auth) {
		t.Fatalf("GetNumVectors on 401 returned %T, want *AuthenticationError", err)
	}
}

func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
