package cyborgdb

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
)

func TestListEmbeddingModelsMapsEveryFieldInOrder(t *testing.T) {
	srv, rec := recordServer(t, 200, `{"models": [
		{"name": "all-MiniLM-L6-v2", "revision": "abc123", "dimension": 384, "max_seq_length": 256, "normalize": true},
		{"name": "bge-base-en-v1.5", "revision": "def456", "dimension": 768, "max_seq_length": 512, "normalize": false}
	]}`)
	client, err := NewClient(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	models, err := client.ListEmbeddingModels(context.Background())
	if err != nil {
		t.Fatalf("ListEmbeddingModels: %v", err)
	}
	want := []EmbeddingModel{
		{Name: "all-MiniLM-L6-v2", Revision: "abc123", Dimension: 384, MaxSeqLength: 256, Normalize: true},
		{Name: "bge-base-en-v1.5", Revision: "def456", Dimension: 768, MaxSeqLength: 512, Normalize: false},
	}
	if !reflect.DeepEqual(models, want) {
		t.Errorf("models = %+v, want %+v", models, want)
	}
	if rec.Method != http.MethodGet || rec.Path != "/v1/embedding-models" {
		t.Errorf("request = %s %s, want GET /v1/embedding-models", rec.Method, rec.Path)
	}
}

func TestListEmbeddingModelsEmptyIsNotNil(t *testing.T) {
	srv, _ := recordServer(t, 200, `{"models": []}`)
	client, err := NewClient(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	models, err := client.ListEmbeddingModels(context.Background())
	if err != nil {
		t.Fatalf("ListEmbeddingModels: %v", err)
	}
	if models == nil || len(models) != 0 {
		t.Errorf("models = %v, want empty non-nil slice", models)
	}
}

func TestListEmbeddingModelsReturnsTypedError(t *testing.T) {
	srv, _ := recordServer(t, 503, `{"detail":"unavailable"}`)
	client, err := NewClient(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.ListEmbeddingModels(context.Background())
	var service *ServiceError
	if !errors.As(err, &service) {
		t.Fatalf("ListEmbeddingModels on 503 returned %T, want *ServiceError", err)
	}
}
