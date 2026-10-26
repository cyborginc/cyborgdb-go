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
	return errFromResponse(t, code, `{"detail":"synthetic failure"}`)
}

// errFromResponse is errFromStatus with the response body under test control.
func errFromResponse(t *testing.T, code int, body string) error {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "req-abc123")
		if code == 429 {
			w.Header().Set("Retry-After", "2.5")
		}
		w.WriteHeader(code)
		_, _ = w.Write([]byte(body))
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

// TestUnnamedStatusIsAPIError: a status the taxonomy does not name still comes
// back as an exported, non-retryable type carrying the status — the Go
// equivalent of the base CyborgDBError in the Python and JS SDKs.
func TestUnnamedStatusIsAPIError(t *testing.T) {
	for _, code := range []int{405, 413, 418} {
		err := errFromStatus(t, code)
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("HTTP %d produced %T, want *APIError", code, err)
		}
		if apiErr.StatusCode() != code {
			t.Errorf("StatusCode() = %d, want %d", apiErr.StatusCode(), code)
		}
		if apiErr.Retryable() {
			t.Errorf("HTTP %d: Retryable() = true, want false", code)
		}
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

// TestServiceKindIsCarried: a kind named in the response body is returned
// verbatim, including kinds this SDK does not name yet.
func TestServiceKindIsCarried(t *testing.T) {
	for _, tc := range []struct {
		status int
		kind   string
	}{
		{401, KindWrongKey},
		{401, KindUnauthenticated},
		{403, KindQuotaExceeded},
		{403, KindPermissionDenied},
		{400, KindFailedPrecondition},
		{500, KindDataLoss},
		{400, "some_new_kind"},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			err := errFromResponse(t, tc.status, `{"detail":"synthetic failure","kind":"`+tc.kind+`"}`)
			var cerr Error
			if !errors.As(err, &cerr) {
				t.Fatalf("errors.As to cyborgdb.Error failed for %T", err)
			}
			if got := cerr.Kind(); got != tc.kind {
				t.Errorf("Kind() = %q, want %q", got, tc.kind)
			}
			if got := cerr.Detail(); got != "synthetic failure" {
				t.Errorf("Detail() = %q, want \"synthetic failure\"", got)
			}
		})
	}
}

func TestKindFallsBackToStatus(t *testing.T) {
	for _, tc := range []struct {
		status int
		kind   string
	}{
		{400, KindInvalidArgument},
		{401, KindUnauthenticated},
		{403, KindPermissionDenied},
		{404, KindNotFound},
		{405, KindInvalidArgument},
		{409, KindAlreadyExists},
		{422, KindInvalidArgument},
		{429, KindRateLimited},
		{500, KindInternal},
		{503, KindUnavailable},
	} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			err := errFromStatus(t, tc.status)
			var cerr Error
			if !errors.As(err, &cerr) {
				t.Fatalf("errors.As to cyborgdb.Error failed for %T", err)
			}
			if got := cerr.Kind(); got != tc.kind {
				t.Errorf("HTTP %d: Kind() = %q, want %q", tc.status, got, tc.kind)
			}
		})
	}
}

func TestKindWithoutJSONBody(t *testing.T) {
	err := errFromResponse(t, 503, "Service Unavailable")
	var cerr Error
	if !errors.As(err, &cerr) {
		t.Fatalf("errors.As to cyborgdb.Error failed for %T", err)
	}
	if got := cerr.Kind(); got != KindUnavailable {
		t.Errorf("Kind() = %q, want %q", got, KindUnavailable)
	}
	if got := cerr.Detail(); got != "Service Unavailable" {
		t.Errorf("Detail() = %q, want \"Service Unavailable\"", got)
	}
}

// TestJSONBodyWithoutDetail: a JSON error body with no detail leaves Detail
// empty rather than echoing the raw body, and still carries the kind.
func TestJSONBodyWithoutDetail(t *testing.T) {
	err := errFromResponse(t, 401, `{"kind":"wrong_key"}`)
	var cerr Error
	if !errors.As(err, &cerr) {
		t.Fatalf("errors.As to cyborgdb.Error failed for %T", err)
	}
	if got := cerr.Detail(); got != "" {
		t.Errorf("Detail() = %q, want \"\"", got)
	}
	if got := cerr.Kind(); got != KindWrongKey {
		t.Errorf("Kind() = %q, want %q", got, KindWrongKey)
	}
}

func TestFieldErrorsAreCarried(t *testing.T) {
	err := errFromResponse(t, 422, `{"detail":"synthetic failure","kind":"invalid_argument",`+
		`"errors":[{"field":"body.items.0.vector","message":"Field required","type":"missing"}]}`)
	var cerr Error
	if !errors.As(err, &cerr) {
		t.Fatalf("errors.As to cyborgdb.Error failed for %T", err)
	}
	want := []FieldError{{Field: "body.items.0.vector", Message: "Field required", Type: "missing"}}
	if got := cerr.FieldErrors(); !reflect.DeepEqual(got, want) {
		t.Errorf("FieldErrors() = %+v, want %+v", got, want)
	}
}

// TestMalformedFieldDoesNotDiscardOthers: one field in an unexpected shape is
// dropped alone — the rest of the body still decodes, instead of the whole
// payload collapsing into a status-derived kind.
func TestMalformedFieldDoesNotDiscardOthers(t *testing.T) {
	for _, tc := range []struct {
		name       string
		body       string
		wantDetail string
		wantKind   string
		wantErrors []FieldError
	}{
		{
			name:       "kind is a number",
			body:       `{"detail":"synthetic failure","kind":7}`,
			wantDetail: "synthetic failure",
			wantKind:   KindNotFound, // falls back to the 404 status
		},
		{
			name:       "errors is not a list",
			body:       `{"detail":"synthetic failure","kind":"not_found","errors":"oops"}`,
			wantDetail: "synthetic failure",
			wantKind:   KindNotFound,
		},
		{
			name:       "detail is an object",
			body:       `{"detail":{"msg":"boom"},"kind":"not_found"}`,
			wantDetail: `{"msg":"boom"}`,
			wantKind:   KindNotFound,
		},
		{
			name:     "kind malformed but errors intact",
			body:     `{"kind":[],"errors":[{"field":"body.k","message":"bad","type":"invalid"}]}`,
			wantKind: KindNotFound,
			wantErrors: []FieldError{
				{Field: "body.k", Message: "bad", Type: "invalid"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := errFromResponse(t, 404, tc.body)
			var cerr Error
			if !errors.As(err, &cerr) {
				t.Fatalf("errors.As to cyborgdb.Error failed for %T", err)
			}
			if got := cerr.Detail(); got != tc.wantDetail {
				t.Errorf("Detail() = %q, want %q", got, tc.wantDetail)
			}
			if got := cerr.Kind(); got != tc.wantKind {
				t.Errorf("Kind() = %q, want %q", got, tc.wantKind)
			}
			if got := cerr.FieldErrors(); !reflect.DeepEqual(got, tc.wantErrors) {
				t.Errorf("FieldErrors() = %+v, want %+v", got, tc.wantErrors)
			}
		})
	}
}

func TestFieldErrorsAbsentByDefault(t *testing.T) {
	err := errFromResponse(t, 404, `{"detail":"synthetic failure","kind":"not_found"}`)
	var cerr Error
	if !errors.As(err, &cerr) {
		t.Fatalf("errors.As to cyborgdb.Error failed for %T", err)
	}
	if got := cerr.FieldErrors(); got != nil {
		t.Errorf("FieldErrors() = %+v, want nil", got)
	}
}

// TestClientSideKinds: errors raised without any response still carry a kind.
func TestClientSideKinds(t *testing.T) {
	_, err := NewClient("not-a-url", "key")
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("NewClient: got %T, want *ValidationError", err)
	}
	if got := validation.Kind(); got != KindInvalidArgument {
		t.Errorf("pre-flight Kind() = %q, want %q", got, KindInvalidArgument)
	}
	if got := validation.FieldErrors(); got != nil {
		t.Errorf("pre-flight FieldErrors() = %+v, want nil", got)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()
	client, err := NewClient(url, "test-key")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.ListIndexes(context.Background())
	var transport *TransportError
	if !errors.As(err, &transport) {
		t.Fatalf("got %T, want *TransportError", err)
	}
	if got := transport.Kind(); got != KindTransport {
		t.Errorf("transport Kind() = %q, want %q", got, KindTransport)
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
		&APIError{},
	} {
		if _, ok := e.(Error); !ok {
			t.Errorf("%s does not satisfy cyborgdb.Error", reflect.TypeOf(e))
		}
	}
}

// TestDecodeFailureOn2xxIsNotAPIError: a 200 whose body fails to decode is not
// an HTTP failure and must not be reported as one.
func TestDecodeFailureOn2xxIsNotAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"indexes": not-json`))
	}))
	t.Cleanup(srv.Close)

	client, err := NewClient(srv.URL, "test-key")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.ListIndexes(context.Background())
	if err == nil {
		t.Fatal("expected a decode error, got nil")
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		t.Errorf("decode failure on HTTP 200 produced %v; want the decode error unchanged", apiErr)
	}
}
