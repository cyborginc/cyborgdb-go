<p align="center">
  <a href="https://www.cyborg.co">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/cyborginc/cyborgdb-go/main/assets/cyborgdb-logo-dark.svg">
      <img src="https://raw.githubusercontent.com/cyborginc/cyborgdb-go/main/assets/cyborgdb-logo-light.svg" alt="CyborgDB" width="320">
    </picture>
  </a>
</p>

# CyborgDB Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/cyborginc/cyborgdb-go.svg)](https://pkg.go.dev/github.com/cyborginc/cyborgdb-go)
![License](https://img.shields.io/github/license/cyborginc/cyborgdb-go)
![Go Version](https://img.shields.io/github/go-mod/go-version/cyborginc/cyborgdb-go)

The **CyborgDB Go SDK** is the Go client for [CyborgDB](https://www.cyborg.co) — the vector database that stays encrypted even while it's searching. Run similarity search directly on encrypted data with client-side keys; only the result of a query is ever decrypted, never the index. Built with Go's type system and `context` support for services and microservices.

This SDK talks to [`cyborgdb-service`](https://hub.docker.com/r/cyborginc/cyborgdb-service), which you self-host in your own VPC or on-prem and run alongside your app. Install and start it separately. See our [docs](https://docs.cyborg.co) for more info.

## Key Features

- **Encryption-in-use**: Search runs directly on ciphertext — only the query result is decrypted, never the index or stored vectors
- **Encrypted ANN**: Disk-backed encrypted DiskIVF index with recall within 2% of a plaintext baseline ([read the benchmarks](https://www.cyborg.co/performance))
- **Filters on encrypted metadata**: Combine vector similarity with equality and range predicates in a single request
- **BYOK / HYOK**: Wrap per-index keys with AWS KMS or AWS Secrets Manager, or hold the key client-side — you control the key material
- **Per-tenant key isolation**: Per-index, per-user keys with cryptographic RBAC; revoke a user and their keys are erased
- **Idiomatic Go API**: Strong typing with `context` support for cancellation and timeouts

## Getting Started

To get started in minutes, check out our [Quickstart Guide](https://docs.cyborg.co/quickstart).


### Installation

1. Install `cyborgdb-service`

```bash
# Pull the CyborgDB Service image
docker pull cyborginc/cyborgdb-service
```

2. Install `cyborgdb` SDK:

```bash
go get github.com/cyborginc/cyborgdb-go
```

### Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    cyborgdb "github.com/cyborginc/cyborgdb-go"
)

func main() {
    // Initialize the client
    client, err := cyborgdb.NewClient("https://localhost:8000", "your-api-key", false)
    if err != nil {
        log.Fatal(err)
    }
    
    // Generate a 32-byte encryption key
    indexKey, err := cyborgdb.GenerateKey()
    if err != nil {
        log.Fatal(err)
    }
    
    // Create an encrypted index
    dimension := int32(128)
    createParams := &cyborgdb.CreateIndexParams{
        IndexName: "my-index",
        IndexKey:  indexKey,
        Dimension: &dimension,
    }
    
    index, err := client.CreateIndex(context.Background(), createParams)
    if err != nil {
        log.Fatal(err)
    }
    
    // Add encrypted vector items
    items := cyborgdb.VectorItems{
        {
            Id:     "doc1",
            Vector: []float32{0.1, 0.2, 0.3}, // ... 128 dimensions
            Metadata: map[string]interface{}{
                "category": "greeting",
                "language": "en",
            },
        },
        {
            Id:     "doc2",
            Vector: []float32{0.4, 0.5, 0.6}, // ... 128 dimensions
            Metadata: map[string]interface{}{
                "category": "greeting",
                "language": "fr",
            },
        },
    }
    
    err = index.Upsert(context.Background(), items)
    if err != nil {
        log.Fatal(err)
    }
    
    // Query the encrypted index
    queryVector := []float32{0.1, 0.2, 0.3} // ... 128 dimensions
    queryParams := cyborgdb.QueryParams{
        QueryVector: queryVector,
        TopK:        10,
        Include:     []string{"metadata"},
    }
    response, err := index.Query(context.Background(), queryParams)
    if err != nil {
        log.Fatal(err)
    }
    
    // Print the results. For a single-vector query the matches are in
    // Results.ArrayOfQueryResultItem (batch queries populate
    // ArrayOfArrayOfQueryResultItem instead).
    if response.Results.ArrayOfQueryResultItem != nil {
        for _, result := range *response.Results.ArrayOfQueryResultItem {
            fmt.Printf("ID: %s, Distance: %f\n", result.Id, result.GetDistance())
        }
    }
}
```

### Advanced Usage

#### Batch Queries

```go
// Search with multiple query vectors simultaneously
queryVectors := [][]float32{
    {0.1, 0.2, 0.3}, // ... first vector
    {0.4, 0.5, 0.6}, // ... second vector
}

queryParams := cyborgdb.QueryParams{
    BatchQueryVectors: queryVectors,
    TopK:              5,
    Include:           []string{"metadata"},
}
batchResults, err := index.Query(context.Background(), queryParams)
if err != nil {
    log.Fatal(err)
}
```

#### Complex Metadata Filtering

```go
// Advanced metadata filtering with operators
complexFilter := map[string]interface{}{
    "$and": []map[string]interface{}{
        {"category": "greeting"},
        {"metadata.score": map[string]interface{}{"$gt": 0.8}},
        {"language": map[string]interface{}{"$in": []string{"en", "fr"}}},
    },
}

queryVector := []float32{0.1, 0.2, 0.3} // ... your query vector
nProbes := int32(1)
greedy := false

queryParams := cyborgdb.QueryParams{
    QueryVector: queryVector,
    TopK:        10,
    NProbes:     &nProbes,
    Greedy:      &greedy,
    Filters:     complexFilter,
    Include:     []string{"distance", "metadata", "contents"},
}
results, err := index.Query(context.Background(), queryParams)
```

#### Bring Your Own Key (BYOK) via KMS

When the service is configured with a `kms.registry` entry, the SDK can
delegate key management entirely to the server-side KMS. The service
generates the data encryption key, wraps it under the named KMS slot, and
persists the envelope — the SDK never sees or holds the key.

```go
// Create a KMS-backed index — no IndexKey from the SDK side.
// "vendor-kms-slot" must match an entry in the service's cyborgdb.yaml.
kmsName := "vendor-kms-slot"
dimension := int32(128)
metric := "euclidean"
index, err := client.CreateIndex(ctx, &cyborgdb.CreateIndexParams{
    IndexName: "kms-backed-index",
    KmsName:   &kmsName,
    Dimension: &dimension,
    Metric:    &metric,
})

// Reopening the index later doesn't require a key either; the service
// resolves the data key from the index's stored KMS envelope.
loaded, err := client.LoadIndex(ctx, "kms-backed-index", nil)
```

For `provider: none` registry entries, the SDK still supplies the KEK on
every call — pass both `IndexKey` and `KmsName`:

```go
kmsName := "plain"   // registry slot with provider: none
index, err := client.CreateIndex(ctx, &cyborgdb.CreateIndexParams{
    IndexName: "sdk-keyed-index",
    IndexKey:  indexKey,
    KmsName:   &kmsName,
    Dimension: &dimension,
})
```

> **How slots are configured.** A `kms.registry` slot is added to the
> service's `cyborgdb.yaml` by your **cyborgdb-service operator** — not
> from the SDK. Each slot declares one provider (`aws-kms`, `aws`,
> or `none`) plus the AWS identifiers needed to wrap/unwrap data keys.
> For real-KMS slots (`aws-kms` / `aws`), set-up also requires IAM
> work on the customer's AWS account; see `BYOK.md` in the
> cyborgdb-service repo for the full operator + customer walkthrough.
> From the SDK side, you only need the slot name your operator
> provisioned.

## Documentation

For more information on CyborgDB, see the [Cyborg Docs](https://docs.cyborg.co).

## License

The CyborgDB Go SDK is licensed under the MIT License.
