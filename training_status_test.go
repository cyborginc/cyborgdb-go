package cyborgdb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"
)

// trainingStatusBody includes the worker and queue internals the service
// reports alongside the two documented fields.
const trainingStatusBody = `{
	"training_indexes": ["a", "b"],
	"retrain_threshold": 10000,
	"currently_training": "a",
	"queued_indexes": ["b"],
	"worker_running": true,
	"worker_pid": 1234,
	"global_training": {"x": 1}
}`

func TestGetTrainingStatusMapsFields(t *testing.T) {
	srv, rec := recordServer(t, 200, trainingStatusBody)
	client, err := NewClient(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	status, err := client.GetTrainingStatus(context.Background())
	if err != nil {
		t.Fatalf("GetTrainingStatus: %v", err)
	}
	if !reflect.DeepEqual(status.TrainingIndexes, []string{"a", "b"}) || status.RetrainThreshold != 10000 {
		t.Errorf("status = %+v, want [a b] and 10000", status)
	}
	if rec.Method != http.MethodGet || rec.Path != "/v1/indexes/training-status" {
		t.Errorf("request = %s %s, want GET /v1/indexes/training-status", rec.Method, rec.Path)
	}
}

// TestGetTrainingStatusLeavesOutServiceInternals: only the two documented
// fields reach the caller.
func TestGetTrainingStatusLeavesOutServiceInternals(t *testing.T) {
	srv, _ := recordServer(t, 200, trainingStatusBody)
	client, err := NewClient(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	status, err := client.GetTrainingStatus(context.Background())
	if err != nil {
		t.Fatalf("GetTrainingStatus: %v", err)
	}
	raw, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var keys map[string]interface{}
	_ = json.Unmarshal(raw, &keys)
	if got, want := sortedKeys(keys), []string{"retrain_threshold", "training_indexes"}; !reflect.DeepEqual(got, want) {
		t.Errorf("serialized keys = %v, want %v", got, want)
	}
}

func TestGetTrainingStatusEmptyIsNotNil(t *testing.T) {
	srv, _ := recordServer(t, 200, `{"training_indexes": [], "retrain_threshold": 10000}`)
	client, err := NewClient(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	status, err := client.GetTrainingStatus(context.Background())
	if err != nil {
		t.Fatalf("GetTrainingStatus: %v", err)
	}
	if status.TrainingIndexes == nil || len(status.TrainingIndexes) != 0 {
		t.Errorf("TrainingIndexes = %v, want empty non-nil slice", status.TrainingIndexes)
	}
}

func TestIsTrainingUsesTrainingStatus(t *testing.T) {
	srv, _ := recordServer(t, 200, trainingStatusBody)
	client, err := NewClient(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	for name, want := range map[string]bool{"a": true, "c": false} {
		index := &EncryptedIndex{indexName: name, client: client.internal}
		got, err := index.IsTraining(context.Background())
		if err != nil {
			t.Fatalf("IsTraining(%s): %v", name, err)
		}
		if got != want {
			t.Errorf("IsTraining(%s) = %v, want %v", name, got, want)
		}
	}
}

func TestGetTrainingStatusReturnsTypedError(t *testing.T) {
	srv, _ := recordServer(t, 500, `{"detail":"boom"}`)
	client, err := NewClient(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.GetTrainingStatus(context.Background())
	var service *ServiceError
	if !errors.As(err, &service) {
		t.Fatalf("GetTrainingStatus on 500 returned %T, want *ServiceError", err)
	}
}
