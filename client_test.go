package cyborgdb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// statusServer returns a test server that answers every request with code and
// records the request paths it saw.
func statusServer(t *testing.T, code int, body string) (*httptest.Server, *[]string) {
	t.Helper()
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &paths
}

func TestDeleteUserReturnsTypedError(t *testing.T) {
	srv, _ := statusServer(t, 404, `{"detail":"user not found"}`)
	client, err := NewClient(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	index := &EncryptedIndex{indexName: "idx", client: client.internal}

	err = index.DeleteUser(context.Background(), "abc")
	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("DeleteUser on 404 returned %T, want *NotFoundError", err)
	}
	if got := notFound.Detail(); got != "user not found" {
		t.Errorf("Detail() = %q, want \"user not found\"", got)
	}
}

func TestCreateIndexNilParams(t *testing.T) {
	client, err := NewClient("http://localhost:8000", "test-key")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.CreateIndex(context.Background(), nil)
	if !errors.Is(err, ErrNilParams) {
		t.Errorf("errors.Is(err, ErrNilParams) = false for %v", err)
	}
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Errorf("CreateIndex(nil) returned %T, want *ValidationError", err)
	}
}

// TestBaseURLPathPrefixIsKept: a service mounted under a path prefix must
// receive requests under that prefix.
func TestBaseURLPathPrefixIsKept(t *testing.T) {
	for _, prefix := range []string{"/cyborgdb", "/cyborgdb/", ""} {
		t.Run(prefix, func(t *testing.T) {
			srv, paths := statusServer(t, 200, `{"indexes":[]}`)
			client, err := NewClient(srv.URL+prefix, "test-key")
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			if _, err := client.ListIndexes(context.Background()); err != nil {
				t.Fatalf("ListIndexes: %v", err)
			}
			want := "/cyborgdb/v1/indexes/list"
			if prefix == "" {
				want = "/v1/indexes/list"
			}
			if len(*paths) != 1 || (*paths)[0] != want {
				t.Errorf("request paths = %v, want [%s]", *paths, want)
			}
		})
	}
}

func TestLoopbackHostsSkipTLSVerification(t *testing.T) {
	for _, tc := range []struct {
		url        string
		skipVerify bool
	}{
		{"https://localhost:8000", true},
		{"https://127.0.0.1:8000", true},
		{"https://[::1]:8000", true},
		{"https://api.example.com", false},
	} {
		client, err := NewClient(tc.url, "test-key")
		if err != nil {
			t.Fatalf("NewClient(%q): %v", tc.url, err)
		}
		transport := client.internal.APIClient.GetConfig().HTTPClient.Transport.(*http.Transport)
		if got := transport.TLSClientConfig.InsecureSkipVerify; got != tc.skipVerify {
			t.Errorf("%s: InsecureSkipVerify = %v, want %v", tc.url, got, tc.skipVerify)
		}
	}
}

// TestQueryMetadataDescendingOnTheWire: the default must omit ascending so the
// service default (ascending) applies; Descending must send ascending=false.
func TestQueryMetadataDescendingOnTheWire(t *testing.T) {
	for _, tc := range []struct {
		name       string
		descending bool
		want       any // nil means the key must be absent
	}{
		{"default", false, nil},
		{"descending", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body map[string]any
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewDecoder(r.Body).Decode(&body)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"results":[],"ids":[],"count":0}`))
			}))
			t.Cleanup(srv.Close)
			client, err := NewClient(srv.URL, "test-key")
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			index := &EncryptedIndex{indexName: "idx", client: client.internal}
			if _, err := index.QueryMetadata(context.Background(), QueryMetadataParams{
				OrderBy: "rank", Descending: tc.descending,
			}); err != nil {
				t.Fatalf("QueryMetadata: %v", err)
			}
			got, present := body["ascending"]
			if tc.want == nil {
				if present {
					t.Errorf("ascending = %v on the wire, want it omitted", got)
				}
			} else if got != tc.want {
				t.Errorf("ascending = %v on the wire, want %v", got, tc.want)
			}
		})
	}
}

func TestBinaryQuerySendsRerankMult(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	t.Cleanup(srv.Close)
	client, err := NewClient(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	index := &EncryptedIndex{indexName: "idx", client: client.internal}
	_, _ = index.Query(context.Background(), BinaryQueryParams{
		QueryVectors: [][]float32{{0.1, 0.2}},
		RerankMult:   Int32(4),
	})
	if got := body["rerank_mult"]; got != float64(4) {
		t.Errorf("rerank_mult = %v on the wire, want 4", got)
	}
}

func TestSetContentsString(t *testing.T) {
	var item VectorItem
	item.SetContentsString("hello")
	got, err := json.Marshal(item.Contents)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(got) != `"hello"` {
		t.Errorf("contents = %s, want \"hello\"", got)
	}
}
