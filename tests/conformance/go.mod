// Its own module, so the SDK's go.mod and `go test ./...` never see the adapter.
module github.com/cyborginc/cyborgdb-go/tests/conformance

go 1.21

require github.com/cyborginc/cyborgdb-go v0.0.0

replace github.com/cyborginc/cyborgdb-go => ../..
