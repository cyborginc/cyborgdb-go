// errors.go — typed errors for the CyborgDB Go SDK.

package cyborgdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/cyborginc/cyborgdb-go/internal"
)

// Error is satisfied by every typed error this SDK returns. Callers who want
// to catch any CyborgDB failure — rather than one specific kind — use it with
// errors.As:
//
//	var cerr cyborgdb.Error
//	if errors.As(err, &cerr) && cerr.Retryable() {
//	    // back off and try again
//	}
type Error interface {
	error
	// StatusCode is the HTTP status, or 0 when no response reached the client
	// (transport failures) or the call never left it (pre-flight validation).
	StatusCode() int
	// RequestID is the service's correlation id, or "" when the service did
	// not supply one.
	RequestID() string
	// Detail is the service's own message, or "" when there was no response.
	Detail() string
	// RetryAfter is the Retry-After value in seconds, or 0 when absent.
	RetryAfter() float64
	// Retryable reports whether backing off and retrying can succeed. It is
	// fixed per error type — retry loops branch on this rather than
	// re-deriving a status table.
	Retryable() bool
	// Unwrap exposes the originating error, so errors.Is and errors.As reach
	// the generated client's types and the package's sentinel errors.
	Unwrap() error
}

// baseError carries the fields shared by every typed error. Concrete types
// embed it and add Retryable.
type baseError struct {
	kind       string
	statusCode int
	requestID  string
	detail     string
	retryAfter float64
	err        error
}

func (e *baseError) Error() string {
	msg := e.kind
	if e.statusCode != 0 {
		msg = fmt.Sprintf("%s (HTTP %d)", msg, e.statusCode)
	}
	switch {
	case e.detail != "":
		msg = fmt.Sprintf("%s: %s", msg, e.detail)
	case e.err != nil:
		msg = fmt.Sprintf("%s: %s", msg, e.err.Error())
	}
	if e.requestID != "" {
		msg = fmt.Sprintf("%s [request_id=%s]", msg, e.requestID)
	}
	return msg
}

func (e *baseError) Unwrap() error       { return e.err }
func (e *baseError) StatusCode() int     { return e.statusCode }
func (e *baseError) RequestID() string   { return e.requestID }
func (e *baseError) Detail() string      { return e.detail }
func (e *baseError) RetryAfter() float64 { return e.retryAfter }

// ValidationError is returned for HTTP 400 and 422, and for arguments this SDK
// rejects before sending the request. The package's client-side sentinels
// (ErrInvalidURL, ErrEmptyIDs, ErrInconsistentDimension, …) are wrapped inside
// it, so errors.Is keeps working alongside errors.As.
type ValidationError struct{ baseError }

// Retryable reports false: the request is malformed, so resending it fails again.
func (e *ValidationError) Retryable() bool { return false }

// AuthenticationError is returned for HTTP 401 and 403. Check the API key and
// its permissions; inspect StatusCode to tell the two apart.
type AuthenticationError struct{ baseError }

// Retryable reports false: credentials do not fix themselves.
func (e *AuthenticationError) Retryable() bool { return false }

// NotFoundError is returned for HTTP 404 — the collection or item does not exist.
type NotFoundError struct{ baseError }

// Retryable reports false.
func (e *NotFoundError) Retryable() bool { return false }

// ConflictError is returned for HTTP 409 — a state conflict, such as
// requesting training while training is already in progress.
type ConflictError struct{ baseError }

// Retryable reports false: poll for the terminal state instead. A backoff loop
// must not retry a 409.
func (e *ConflictError) Retryable() bool { return false }

// RateLimitError is returned for HTTP 429. Honor RetryAfter when it is set.
//
// The service does not rate-limit yet (cyborgdb-core#2386); this type is
// defined so callers can write the handler once.
type RateLimitError struct{ baseError }

// Retryable reports true.
func (e *RateLimitError) Retryable() bool { return true }

// ServiceError is returned for any 5xx. Not "ServiceUnavailable": a 500 is a
// server bug, not unavailability.
type ServiceError struct{ baseError }

// Retryable reports true.
func (e *ServiceError) Retryable() bool { return true }

// TransportError is returned when no HTTP response reached the client — DNS
// failure, connection refused, TLS failure, or timeout. StatusCode is 0.
//
// Timeouts land here too: a timeout and a refused connection have the same
// caller action, and the difference is diagnostic — read Detail.
type TransportError struct{ baseError }

// Retryable reports true, for idempotent calls.
func (e *TransportError) Retryable() bool { return true }

// compile-time check that every typed error satisfies Error.
var (
	_ Error = (*ValidationError)(nil)
	_ Error = (*AuthenticationError)(nil)
	_ Error = (*NotFoundError)(nil)
	_ Error = (*ConflictError)(nil)
	_ Error = (*RateLimitError)(nil)
	_ Error = (*ServiceError)(nil)
	_ Error = (*TransportError)(nil)
)

// newValidationError builds a pre-flight ValidationError around one of the
// package's sentinel errors, so callers can match with either errors.Is on the
// sentinel or errors.As on the type.
func newValidationError(err error) *ValidationError {
	return &ValidationError{baseError{kind: "validation error", err: err}}
}

// translateHTTPError converts a generated-client error into a typed error.
// It returns nil when err is nil, and returns err unchanged for statuses the
// taxonomy does not name (other 4xx, and 3xx).
//
// The generated client returns the raw *http.Response alongside the error;
// internal.APIResponse is not used on this path.
func translateHTTPError(resp *http.Response, err error) error {
	if err == nil {
		return nil
	}
	// No response reached us: DNS, connection refused, TLS, timeout.
	if resp == nil {
		return &TransportError{baseError{kind: "transport error", err: err}}
	}

	base := baseError{
		statusCode: resp.StatusCode,
		requestID:  resp.Header.Get("X-Request-Id"),
		detail:     detailFrom(err),
		retryAfter: retryAfterFrom(resp),
		err:        err,
	}

	switch code := resp.StatusCode; {
	case code == 401 || code == 403:
		base.kind = "authentication error"
		return &AuthenticationError{base}
	case code == 404:
		base.kind = "not found"
		return &NotFoundError{base}
	case code == 409:
		base.kind = "conflict"
		return &ConflictError{base}
	case code == 429:
		base.kind = "rate limited"
		return &RateLimitError{base}
	case code == 400 || code == 422:
		base.kind = "validation error"
		return &ValidationError{base}
	case code >= 500:
		base.kind = "service error"
		return &ServiceError{base}
	default:
		// Not named by the taxonomy — hand the original error back unchanged.
		return err
	}
}

// detailFrom pulls the service's own message out of the generated client's
// error body. The service currently returns FastAPI's {"detail": ...} shape.
func detailFrom(err error) string {
	var apiErr *internal.GenericOpenAPIError
	if !errors.As(err, &apiErr) {
		return ""
	}
	body := apiErr.Body()
	if len(body) == 0 {
		return ""
	}
	var payload struct {
		Detail json.RawMessage `json:"detail"`
	}
	if json.Unmarshal(body, &payload) != nil || len(payload.Detail) == 0 {
		return string(body)
	}
	var s string
	if json.Unmarshal(payload.Detail, &s) == nil {
		return s
	}
	return string(payload.Detail)
}

// retryAfterFrom reads the Retry-After header, seconds form only. The
// HTTP-date form is not used by the service and is reported as absent.
func retryAfterFrom(resp *http.Response) float64 {
	if resp == nil {
		return 0
	}
	v := resp.Header.Get("Retry-After")
	if v == "" {
		return 0
	}
	secs, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}
	return secs
}
