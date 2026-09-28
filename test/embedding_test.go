package test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	cyborgdb "github.com/cyborginc/cyborgdb-go"
	"github.com/cyborginc/cyborgdb-go/internal"
)

// Built-in text embedding (cyborgdb-embed).
//
// cyborgdb-core now embeds in C++ via cyborgdb-embed rather than calling
// sentence-transformers from Python (cyborgdb-core#2422). cyborgdb-service#271
// is the service half: it drops the sentence-transformers gate, adds
// GET /v1/embedding-models, and maps core's EmbeddingModelUnavailable to 503
// instead of letting it fall through as a 500.
//
// These fail until cyborgdb-service#271 merges. They assert the contract that
// PR's own tests assert, so they double as the SDK-side check that it landed.
// Mirrors py tests/test_embedding.py.

const (
	embedModel    = "sentence-transformers/all-MiniLM-L6-v2"
	embedModelDim = 384
)

// embedCorpus holds three distinct topics, so a semantic hit is unambiguous.
// Ported from cyborgdb-core tests/embedding_test.cpp.
var embedCorpus = [][2]string{
	{"fox", "The quick brown fox jumps over the lazy dog."},
	{"revenue", "Quarterly revenue grew eleven percent year over year."},
	{"weather", "Heavy rain and strong winds are expected tomorrow."},
}

// Called directly — the SDK has no wrapper for this endpoint yet.
func TestEmbeddingModelCatalog(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, testBaseURL()+"/v1/embedding-models", nil)
	if err != nil {
		t.Fatalf("building the request failed: %v", err)
	}
	req.Header.Set("X-API-Key", testAPIKey())

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("GET /v1/embedding-models failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected the catalog endpoint, got HTTP %d", resp.StatusCode)
	}

	var body struct {
		Models []struct {
			Name         string `json:"name"`
			Dimension    int    `json:"dimension"`
			MaxSeqLength int    `json:"max_seq_length"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decoding the catalog failed: %v", err)
	}

	var found bool
	for _, m := range body.Models {
		if m.Name == embedModel {
			found = true
			if m.Dimension != embedModelDim {
				t.Errorf("%s dimension = %d, want %d", m.Name, m.Dimension, embedModelDim)
			}
		}
		if m.Dimension <= 0 {
			t.Errorf("%s has dimension %d", m.Name, m.Dimension)
		}
		if m.MaxSeqLength <= 0 {
			t.Errorf("%s has max_seq_length %d", m.Name, m.MaxSeqLength)
		}
	}
	if !found {
		t.Errorf("%s missing from the catalog", embedModel)
	}
}

func newEmbeddingIndex(t *testing.T, model string, dimension *int32) (*cyborgdb.EncryptedIndex, error) {
	t.Helper()
	client := newIsolatedClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	index, err := client.CreateIndex(ctx, &cyborgdb.CreateIndexParams{
		IndexName:      generateUniqueName("embed_"),
		IndexKey:       generateRandomKey(),
		EmbeddingModel: &model,
		Dimension:      dimension,
	})
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanCancel()
		_ = index.DeleteIndex(cleanCtx)
	})
	return index, nil
}

// assertClientError requires a 400-class rejection. Before #271 an unsupported
// model came back as a 500, which tells a caller to retry something that can
// never succeed.
func assertClientError(t *testing.T, err error, what string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s should be rejected", what)
	}
	var cerr cyborgdb.Error
	if !errors.As(err, &cerr) {
		t.Fatalf("%s: error is not a cyborgdb.Error: %T (%v)", what, err, err)
	}
	if code := cerr.StatusCode(); code != http.StatusBadRequest {
		t.Errorf("%s: HTTP %d, want 400 — a bad model name is the caller's mistake", what, code)
	}
}

func TestEmbeddingBareMixedCaseNameAccepted(t *testing.T) {
	// cyborgdb-embed accepts the name with or without the org prefix.
	index, err := newEmbeddingIndex(t, "ALL-MINILM-L6-V2", nil)
	if err != nil {
		t.Fatalf("a bare mixed-case name should be accepted: %v", err)
	}
	assertIndexDimension(t, index, embedModelDim)
}

func TestEmbeddingFullNameAccepted(t *testing.T) {
	index, err := newEmbeddingIndex(t, embedModel, nil)
	if err != nil {
		t.Fatalf("the full model name should be accepted: %v", err)
	}
	assertIndexDimension(t, index, embedModelDim)
}

func TestEmbeddingUnknownModelIsAClientError(t *testing.T) {
	_, err := newEmbeddingIndex(t, "not-a-real-model", nil)
	assertClientError(t, err, "an unsupported model name")
}

func TestEmbeddingOpenAIStyleNameRejected(t *testing.T) {
	_, err := newEmbeddingIndex(t, "text-embedding-3-small", nil)
	assertClientError(t, err, "an openai-style model name")
}

func TestEmbeddingDimensionContradictingTheModelRejected(t *testing.T) {
	wrong := int32(embedModelDim + 1)
	_, err := newEmbeddingIndex(t, embedModel, &wrong)
	assertClientError(t, err, "a dimension contradicting the model")
}

func TestEmbeddingDimensionMatchingTheModelAccepted(t *testing.T) {
	// Anchors the test above: the rejection is about the contradiction, not
	// about passing a dimension at all.
	right := int32(embedModelDim)
	index, err := newEmbeddingIndex(t, embedModel, &right)
	if err != nil {
		t.Fatalf("a matching dimension should be accepted: %v", err)
	}
	assertIndexDimension(t, index, embedModelDim)
}

func assertIndexDimension(t *testing.T, index *cyborgdb.EncryptedIndex, want int32) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	got, err := index.Dimension(ctx)
	if err != nil {
		t.Fatalf("Dimension failed: %v", err)
	}
	if got != want {
		t.Errorf("dimension = %d, want %d", got, want)
	}
}

// Text in, semantically-related text finds it again.
func TestEmbeddingRoundTrip(t *testing.T) {
	index, err := newEmbeddingIndex(t, embedModel, nil)
	if err != nil {
		t.Fatalf("CreateIndex with an embedding model failed: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	items := make(cyborgdb.VectorItems, len(embedCorpus))
	ids := make([]string, len(embedCorpus))
	for i, row := range embedCorpus {
		ids[i] = row[0]
		text := row[1]
		item := cyborgdb.VectorItem{Id: row[0]}
		item.SetContents(internal.Contents{String: &text})
		items[i] = item
	}
	if err := index.Upsert(ctx, items); err != nil {
		t.Fatalf("Upsert of text contents failed: %v", err)
	}
	waitForIDs(t, index, ids)

	topHit := func(text string) string {
		resp, qErr := index.Query(ctx, cyborgdb.QueryParams{
			QueryContents: &text,
			TopK:          1,
		})
		if qErr != nil {
			t.Fatalf("query by contents failed: %v", qErr)
		}
		got := getQueryResultItems(&resp.Results)
		if len(got) == 0 {
			return ""
		}
		return got[0].Id
	}

	// Neither query shares a distinctive word with its target, so a match has
	// to come from the embedding rather than lexical overlap.
	for _, tc := range [][2]string{
		{"a fox leaping over a sleepy dog", "fox"},
		{"company earnings this quarter", "revenue"},
		{"a storm is coming", "weather"},
	} {
		if got := topHit(tc[0]); got != tc[1] {
			t.Errorf("%q matched %q, want %q", tc[0], got, tc[1])
		}
	}

	resp, err := index.Get(ctx, []string{"fox"}, []string{"contents", "vector"})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected one row, got %d", len(resp.Results))
	}
	if got := resp.Results[0].GetContents(); got != embedCorpus[0][1] {
		t.Errorf("contents = %q, want %q", got, embedCorpus[0][1])
	}
	if got := len(resp.Results[0].Vector); got != embedModelDim {
		t.Errorf("stored vector has %d dimensions, want %d", got, embedModelDim)
	}
}
