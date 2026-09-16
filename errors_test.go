package cyborgdb

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// Expected shape of SDK errors.
var taxonomy = []struct {
	status    int // 0 means "no HTTP response reached the client"
	matches   func(error) bool
	concept   string
	retryable bool
}{
	{400, func(e error) bool { var t *ValidationError; return errors.As(e, &t) }, "ValidationError", false},
	{422, func(e error) bool { var t *ValidationError; return errors.As(e, &t) }, "ValidationError", false},
	{401, func(e error) bool { var t *AuthenticationError; return errors.As(e, &t) }, "AuthenticationError", false},
	{403, func(e error) bool { var t *AuthenticationError; return errors.As(e, &t) }, "AuthenticationError", false},
	{404, func(e error) bool { var t *NotFoundError; return errors.As(e, &t) }, "NotFoundError", false},
	{409, func(e error) bool { var t *ConflictError; return errors.As(e, &t) }, "ConflictError", false},
	{429, func(e error) bool { var t *RateLimitError; return errors.As(e, &t) }, "RateLimitError", true},
	{500, func(e error) bool { var t *ServiceError; return errors.As(e, &t) }, "ServiceError", true},
	{502, func(e error) bool { var t *ServiceError; return errors.As(e, &t) }, "ServiceError", true},
	{503, func(e error) bool { var t *ServiceError; return errors.As(e, &t) }, "ServiceError", true},
	{504, func(e error) bool { var t *ServiceError; return errors.As(e, &t) }, "ServiceError", true},
}

// TestStatusMapping: every status the taxonomy names produces its type, with
// the retryability that type promises.
func TestStatusMapping(t *testing.T) {
	for _, tc := range taxonomy {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			err := errFromStatus(t, tc.status)
			if !tc.matches(err) {
				t.Fatalf("HTTP %d produced %T, want %s", tc.status, err, tc.concept)
			}
			var cerr Error
			if !errors.As(err, &cerr) {
				t.Fatalf("HTTP %d: errors.As to cyborgdb.Error failed for %T", tc.status, err)
			}
			if cerr.Retryable() != tc.retryable {
				t.Errorf("HTTP %d: Retryable() = %v, want %v", tc.status, cerr.Retryable(), tc.retryable)
			}
			if cerr.StatusCode() != tc.status {
				t.Errorf("StatusCode() = %d, want %d", cerr.StatusCode(), tc.status)
			}
		})
	}
}

// errFromStatus drives a real request through the generated client against a
// test server returning the given status, and returns the resulting error.
func errFromStatus(t *testing.T, code int) error {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "req-abc123")
		if code == 429 {
			w.Header().Set("Retry-After", "2.5")
		}
		w.WriteHeader(code)
		_, _ = w.Write([]byte(`{"detail":"synthetic failure"}`))
	}))
	t.Cleanup(srv.Close)

	client, err := NewClient(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.ListIndexes(context.Background())
	if err == nil {
		t.Fatalf("HTTP %d: expected an error, got nil", code)
	}
	return err
}

func TestStatusFieldsArePopulated(t *testing.T) {
	err := errFromStatus(t, 503)
	var cerr Error
	if !errors.As(err, &cerr) {
		t.Fatalf("errors.As to cyborgdb.Error failed for %T", err)
	}
	if got := cerr.StatusCode(); got != 503 {
		t.Errorf("StatusCode() = %d, want 503", got)
	}
	if got := cerr.RequestID(); got != "req-abc123" {
		t.Errorf("RequestID() = %q, want req-abc123", got)
	}
	if got := cerr.Detail(); got != "synthetic failure" {
		t.Errorf("Detail() = %q, want \"synthetic failure\"", got)
	}
	if !cerr.Retryable() {
		t.Error("Retryable() = false for a 503")
	}
}

func TestRetryAfterIsParsed(t *testing.T) {
	err := errFromStatus(t, 429)
	var cerr Error
	if !errors.As(err, &cerr) {
		t.Fatalf("errors.As to cyborgdb.Error failed for %T", err)
	}
	if got := cerr.RetryAfter(); got != 2.5 {
		t.Errorf("RetryAfter() = %v, want 2.5", got)
	}
}

// TestUnnamedStatusPassesThrough: a status the taxonomy does not name must not
// be dressed up as a typed error.
func TestUnnamedStatusPassesThrough(t *testing.T) {
	err := errFromStatus(t, 418)
	var cerr Error
	if errors.As(err, &cerr) {
		t.Errorf("HTTP 418 produced typed error %T; it should pass through untyped", cerr)
	}
}

func TestTransportError(t *testing.T) {
	// A server that is closed before the call: nothing answers, so no HTTP
	// response reaches the client.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	client, err := NewClient(url, "test-key")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.ListIndexes(context.Background())
	if err == nil {
		t.Fatal("expected a transport error, got nil")
	}
	var transport *TransportError
	if !errors.As(err, &transport) {
		t.Fatalf("got %T, want *TransportError", err)
	}
	if transport.StatusCode() != 0 {
		t.Errorf("StatusCode() = %d, want 0 for a transport failure", transport.StatusCode())
	}
	if !transport.Retryable() {
		t.Error("Retryable() = false for a transport failure")
	}
}

func TestPreflightValidation(t *testing.T) {
	for _, tc := range []struct{ name, url string }{
		{"empty", ""},
		{"schemeless", "localhost:8080"},
		{"not a url", "not-a-url"},
		{"wrong scheme", "ftp://example.com"},
		{"no host", "http://"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewClient(tc.url, "key")
			if err == nil {
				t.Fatalf("NewClient(%q) returned no error", tc.url)
			}
			// Both matchers must work: errors.Is on the established sentinel,
			// errors.As on the taxonomy type.
			if !errors.Is(err, ErrInvalidURL) {
				t.Errorf("errors.Is(err, ErrInvalidURL) = false for %q", tc.url)
			}
			var validation *ValidationError
			if !errors.As(err, &validation) {
				t.Errorf("errors.As to *ValidationError = false for %q (%T)", tc.url, err)
			}
			if validation != nil && validation.StatusCode() != 0 {
				t.Errorf("StatusCode() = %d, want 0 for pre-flight validation", validation.StatusCode())
			}
		})
	}
}

func TestValidBaseURLsAreAccepted(t *testing.T) {
	for _, url := range []string{"http://localhost:8080", "https://api.example.com", "http://127.0.0.1:7000"} {
		if _, err := NewClient(url, "key"); err != nil {
			t.Errorf("NewClient(%q) = %v, want no error", url, err)
		}
	}
}

// TestCauseChainIsIntact: wrapping must not sever errors.Unwrap, or callers
// lose the generated client's own error.
func TestCauseChainIsIntact(t *testing.T) {
	err := errFromStatus(t, 500)
	var cerr Error
	if !errors.As(err, &cerr) {
		t.Fatalf("errors.As to cyborgdb.Error failed for %T", err)
	}
	if cerr.Unwrap() == nil {
		t.Error("Unwrap() = nil; the originating error was dropped")
	}
}

// TestEveryTypeSatisfiesError guards against a new type being added without
// the Retryable method, which would silently drop it out of the interface.
func TestEveryTypeSatisfiesError(t *testing.T) {
	for _, e := range []any{
		&ValidationError{}, &AuthenticationError{}, &NotFoundError{},
		&ConflictError{}, &RateLimitError{}, &ServiceError{}, &TransportError{},
	} {
		if _, ok := e.(Error); !ok {
			t.Errorf("%s does not satisfy cyborgdb.Error", reflect.TypeOf(e))
		}
	}
}
