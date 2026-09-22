package cyborgdb

import (
	"net/http"
	"testing"
)

// TLS certificate verification on NewClient.
//
// Ported from cyborgdb-py tests/test_ssl_verification.py. Go resolves the host
// with url.Parse and compares Hostname() exactly, where the Python and TS SDKs
// use a substring match over the whole URL — see cyborgdb-core#2399. The
// lookalike cases below pass here and fail there; they are a regression guard
// against Go adopting the same shortcut.
//
// In-package so the resolved setting can be read off the transport rather than
// inferred from behaviour. No service required.

// resolvedSkipVerify reports the InsecureSkipVerify the client ended up with.
func resolvedSkipVerify(t *testing.T, baseURL string, verifySSL ...bool) bool {
	t.Helper()
	c, err := NewClient(baseURL, "test-key", verifySSL...)
	if err != nil {
		t.Fatalf("NewClient(%q) failed: %v", baseURL, err)
	}
	transport, ok := c.internal.APIClient.GetConfig().HTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", c.internal.APIClient.GetConfig().HTTPClient.Transport)
	}
	return transport.TLSClientConfig.InsecureSkipVerify
}

func TestSSLAutoDetectRemoteHostVerifies(t *testing.T) {
	if resolvedSkipVerify(t, "https://api.example.com") {
		t.Error("a remote https host must verify certificates")
	}
}

func TestSSLAutoDetectLocalhostSkipsVerification(t *testing.T) {
	for _, url := range []string{"https://localhost:8000", "https://127.0.0.1:8000"} {
		t.Run(url, func(t *testing.T) {
			if !resolvedSkipVerify(t, url) {
				t.Errorf("%s is local; verification should be skipped", url)
			}
		})
	}
}

func TestSSLHttpNeverVerifies(t *testing.T) {
	// No TLS on a plaintext URL.
	for _, url := range []string{"http://api.example.com", "http://localhost:8000"} {
		t.Run(url, func(t *testing.T) {
			if !resolvedSkipVerify(t, url) {
				t.Errorf("%s is plaintext; there is nothing to verify", url)
			}
		})
	}
}

func TestSSLLookalikeHostsStillVerify(t *testing.T) {
	// Each of these contains "localhost" or "127.0.0.1" as a substring but is
	// not a local host. Every one is a domain an attacker could register, or a
	// legitimate production URL. cyborgdb-core#2399 is the same case failing in
	// the Python and TS SDKs.
	for _, url := range []string{
		"https://localhost.evil.com",
		"https://127.0.0.1.evil.com",
		"https://notlocalhost.example.com",
		"https://my-localhost-proxy.example.com",
		"https://api.example.com/?region=localhost",
	} {
		t.Run(url, func(t *testing.T) {
			if resolvedSkipVerify(t, url) {
				t.Errorf("%s is not a local host; TLS must still be verified", url)
			}
		})
	}
}

func TestSSLExplicitSettingWins(t *testing.T) {
	cases := []struct {
		url            string
		verifySSL      bool
		wantSkipVerify bool
	}{
		{"https://api.example.com", true, false},
		{"https://api.example.com", false, true},
		// Auto-detection is a convenience, not a ceiling: asking for
		// verification against a local host must be honoured.
		{"https://localhost:8000", true, false},
		{"https://127.0.0.1:8000", true, false},
		// Go honours an explicit value even for a plaintext URL. Python and TS
		// force it off first and silently discard the request — noted on
		// cyborgdb-core#2399 as a divergence to reconcile.
		{"http://api.example.com", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.url, func(t *testing.T) {
			if got := resolvedSkipVerify(t, tc.url, tc.verifySSL); got != tc.wantSkipVerify {
				t.Errorf("verifySSL=%v on %s: InsecureSkipVerify=%v, want %v",
					tc.verifySSL, tc.url, got, tc.wantSkipVerify)
			}
		})
	}
}

func TestSSLInvalidURLIsRejected(t *testing.T) {
	if _, err := NewClient("://not-a-url", "test-key"); err == nil {
		t.Error("a malformed base URL should be rejected")
	}
}

func TestSSLDoesNotDisturbClientSetup(t *testing.T) {
	// The SSL branch runs before the rest of client construction; none of the
	// branches may leave the client unusable.
	for _, url := range []string{
		"https://api.example.com",
		"https://localhost:8000",
		"http://api.example.com",
	} {
		t.Run(url, func(t *testing.T) {
			c, err := NewClient(url, "secret-key")
			if err != nil {
				t.Fatalf("NewClient(%q) failed: %v", url, err)
			}
			if c.internal == nil {
				t.Fatal("client has no internal client")
			}
			if got := c.internal.APIClient.GetConfig().HTTPClient; got == nil {
				t.Error("client has no HTTP client")
			}
		})
	}
}
