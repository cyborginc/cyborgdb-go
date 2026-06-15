#!/bin/bash

set -e

echo "Generating Go client from openapi.json..."

# Check if OpenAPI spec exists
if [ ! -f "openapi.json" ]; then
    echo "Error: openapi.json not found!"
    exit 1
fi

# Check if openapi-generator-cli is installed
if ! command -v openapi-generator-cli &> /dev/null; then
    echo "Installing openapi-generator-cli..."
    npm install -g @openapitools/openapi-generator-cli
fi

# Backup custom wrapper file
echo "Backing up custom files..."
cp internal/client_wrapper.go /tmp/client_wrapper.go.bak

# Clean existing generated files
echo "Cleaning old generated files..."
rm -f internal/*.go

# Generate the Go client
echo "Generating Go client..."
openapi-generator-cli generate \
    -i openapi.json \
    -g go \
    -o ./internal \
    --package-name internal \
    --additional-properties=isGoSubmodule=true,withGoMod=false \
    --skip-validate-spec

# Restore custom wrapper file
echo "Restoring custom files..."
cp /tmp/client_wrapper.go.bak internal/client_wrapper.go

# Fix binary type issues (OpenAPI Generator bug with format:binary in Go)
echo "Fixing generated code issues..."

# Fix model_contents.go - completely replace the broken file
cat > internal/model_contents.go << 'EOF'
package internal

import (
	"encoding/json"
	"fmt"
)

type Contents struct {
	Bytes  *[]byte
	String *string
}

func (dst *Contents) UnmarshalJSON(data []byte) error {
	if string(data) == "" || string(data) == "{}" || string(data) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		dst.String = &s
		return nil
	}
	return fmt.Errorf("data failed to match schemas in anyOf(Contents)")
}

func (src Contents) MarshalJSON() ([]byte, error) {
	if src.Bytes != nil {
		return json.Marshal(src.Bytes)
	}
	if src.String != nil {
		return json.Marshal(src.String)
	}
	return []byte("null"), nil
}

type NullableContents struct {
	value *Contents
	isSet bool
}

func (v NullableContents) Get() *Contents { return v.value }
func (v *NullableContents) Set(val *Contents) { v.value = val; v.isSet = true }
func (v NullableContents) IsSet() bool { return v.isSet }
func (v *NullableContents) Unset() { v.value = nil; v.isSet = false }
func NewNullableContents(val *Contents) *NullableContents { return &NullableContents{value: val, isSet: true} }
func (v NullableContents) MarshalJSON() ([]byte, error) { return json.Marshal(v.value) }
func (v *NullableContents) UnmarshalJSON(src []byte) error { v.isSet = true; return json.Unmarshal(src, &v.value) }
EOF

# Fix model_binary_vector_batch_contents_inner.go
cat > internal/model_binary_vector_batch_contents_inner.go << 'EOF'
package internal

import (
	"encoding/json"
	"fmt"
)

type BinaryVectorBatchContentsInner struct {
	Bytes  *[]byte
	String *string
}

func (dst *BinaryVectorBatchContentsInner) UnmarshalJSON(data []byte) error {
	if string(data) == "" || string(data) == "{}" || string(data) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		dst.String = &s
		return nil
	}
	return fmt.Errorf("data failed to match schemas in anyOf(BinaryVectorBatchContentsInner)")
}

func (src BinaryVectorBatchContentsInner) MarshalJSON() ([]byte, error) {
	if src.Bytes != nil {
		return json.Marshal(src.Bytes)
	}
	if src.String != nil {
		return json.Marshal(src.String)
	}
	return []byte("null"), nil
}

type NullableBinaryVectorBatchContentsInner struct {
	value *BinaryVectorBatchContentsInner
	isSet bool
}

func (v NullableBinaryVectorBatchContentsInner) Get() *BinaryVectorBatchContentsInner { return v.value }
func (v *NullableBinaryVectorBatchContentsInner) Set(val *BinaryVectorBatchContentsInner) { v.value = val; v.isSet = true }
func (v NullableBinaryVectorBatchContentsInner) IsSet() bool { return v.isSet }
func (v *NullableBinaryVectorBatchContentsInner) Unset() { v.value = nil; v.isSet = false }
func NewNullableBinaryVectorBatchContentsInner(val *BinaryVectorBatchContentsInner) *NullableBinaryVectorBatchContentsInner { return &NullableBinaryVectorBatchContentsInner{value: val, isSet: true} }
func (v NullableBinaryVectorBatchContentsInner) MarshalJSON() ([]byte, error) { return json.Marshal(v.value) }
func (v *NullableBinaryVectorBatchContentsInner) UnmarshalJSON(src []byte) error { v.isSet = true; return json.Unmarshal(src, &v.value) }
EOF

# Fix model_get_result_item_model.go
sed -i '' 's/"os"//g' internal/model_get_result_item_model.go
sed -i '' 's/Nullable\*os\.File/NullableString/g' internal/model_get_result_item_model.go
sed -i '' 's/\*\*os\.File/\*string/g' internal/model_get_result_item_model.go
sed -i '' 's/\*os\.File/string/g' internal/model_get_result_item_model.go

# Fix DisallowUnknownFields - server may return extra fields not in OpenAPI spec
# This allows the client to accept responses with additional fields
echo "Removing strict field checking..."
for f in internal/model_*.go; do
    sed -i '' 's/decoder.DisallowUnknownFields()//g' "$f" 2>/dev/null || true
done

# Debug logging dumps the full request/response, leaking the X-API-Key header,
# request bodies (index keys), and response bodies (minted user keys). Log only
# method/path/status instead (CodeQL go/clear-text-logging).
echo "Stripping sensitive data from debug logging..."
sed -i '' 's|dump, err := httputil.DumpRequestOut(request, true)|dump, err := []byte(fmt.Sprintf("%s %s", request.Method, request.URL.Path)), error(nil)|' internal/client.go
sed -i '' 's|dump, err := httputil.DumpResponse(resp, true)|dump, err := []byte(resp.Status), error(nil)|' internal/client.go
sed -i '' '\|"net/http/httputil"|d' internal/client.go

# Clean up extra generated files
echo "Cleaning up extra files..."
rm -rf internal/docs internal/test internal/api internal/.openapi-generator
rm -f internal/.travis.yml internal/git_push.sh internal/.gitignore internal/README.md internal/go.mod internal/go.sum

# Test build
echo "Testing build..."
go mod tidy
if go build ./...; then
    echo "Build successful!"
else
    echo "Build failed - please check for errors"
    exit 1
fi

echo ""
echo "Go OpenAPI client updated successfully!"
