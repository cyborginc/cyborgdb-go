package cyborgdb

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestContentsBytesMarshalAsText(t *testing.T) {
	b := []byte("café")
	got, err := json.Marshal(Contents{Bytes: &b})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(got) != `"café"` {
		t.Errorf("contents = %s, want \"café\" (not base64)", got)
	}
}

func TestContentsInvalidBytesDoNotMarshal(t *testing.T) {
	b := []byte{0xff, 0xfe}
	if got, err := json.Marshal(Contents{Bytes: &b}); err == nil {
		t.Errorf("Marshal of non-UTF-8 bytes succeeded with %s", got)
	}
}

func TestUpsertSendsBytesContentsAsText(t *testing.T) {
	srv, rec := recordServer(t, 200, `{"status":"success","message":"ok"}`)
	index, _ := keyedIndex(t, srv.URL)

	b := []byte("alpha")
	item := VectorItem{Id: "a", Vector: []float32{1, 0}}
	item.SetContents(Contents{Bytes: &b})
	if err := index.Upsert(context.Background(), VectorItems{item}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	items, _ := rec.Body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("sent items = %v, want one", rec.Body["items"])
	}
	if got := items[0].(map[string]any)["contents"]; got != "alpha" {
		t.Errorf("sent contents = %v, want \"alpha\"", got)
	}
}

func TestUpsertRejectsNonUTF8Contents(t *testing.T) {
	invalid := []byte{'a', 0xff, 'b'}
	invalidString := string(invalid)

	bytesItem := VectorItem{Id: "a", Vector: []float32{1, 0}}
	bytesItem.SetContents(Contents{Bytes: &invalid})
	stringItem := VectorItem{Id: "a", Vector: []float32{1, 0}}
	stringItem.SetContentsString(invalidString)

	for _, tc := range []struct {
		name  string
		input UpsertInput
	}{
		{"items with bytes", VectorItems{bytesItem}},
		{"items with a string", VectorItems{stringItem}},
		{"binary", BinaryUpsertParams{IDs: []string{"a"}, Vectors: [][]float32{{1, 0}}, Contents: []string{invalidString}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, rec := recordServer(t, 200, `{"status":"success","message":"ok"}`)
			index, _ := keyedIndex(t, srv.URL)

			err := index.Upsert(context.Background(), tc.input)
			var verr *ValidationError
			if !errors.As(err, &verr) || !errors.Is(err, ErrContentsNotUTF8) {
				t.Fatalf("Upsert returned %v (%T), want a *ValidationError wrapping ErrContentsNotUTF8", err, err)
			}
			if rec.Path != "" {
				t.Errorf("a request was sent to %s", rec.Path)
			}
		})
	}
}
