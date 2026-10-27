// Command adapter serves cyborgdb-tests' adapter protocol (fd 3/4) over this repo's Go SDK, which
// does its own HTTP to cyborgdb-service. See cyborgdb-tests/ADAPTERS.md.
//
// CYBORGDB_TESTS_VARIANT picks the SDK path: json (default) or binary. The Go client has no
// separate async form, so there is no async variant.
//
// Run under the driver, from a cyborgdb-tests checkout:
//
//	(cd ../cyborgdb-go/tests/conformance && go build -o /tmp/adapter-go-sdk .)
//	CYBORG_TEST_SDK_BASE_URL=http://localhost:8000 bin/cyborgdb-tests run \
//	  --adapter /tmp/adapter-go-sdk --env sdk --variant json
package main

import (
	"fmt"
	"os"
)

// sdkVersion is the SDK commit, set at build time (-ldflags "-X main.sdkVersion=...").
var sdkVersion = "dev"

func main() {
	variant := os.Getenv("CYBORGDB_TESTS_VARIANT")
	if variant == "" {
		variant = "json"
	}
	if variant != "json" && variant != "binary" {
		fmt.Fprintf(os.Stderr, "adapter: CYBORGDB_TESTS_VARIANT=%s, want json or binary\n", variant)
		os.Exit(1)
	}
	h := newHandler(sdk{}, variant, os.Getenv)
	if err := serveFDs(h); err != nil {
		fmt.Fprintln(os.Stderr, "adapter:", err)
		os.Exit(1)
	}
}
