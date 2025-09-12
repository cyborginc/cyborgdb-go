# CyborgDB Go SDK

The **CyborgDB Go SDK** provides a comprehensive client library for interacting with [CyborgDB](https://www.cyborg.co), the first Confidential Vector Database. This SDK enables you to perform encrypted vector operations including ingestion, search, and retrieval while maintaining end-to-end encryption of your vector embeddings. Built with Go's strong typing system, it offers excellent performance and seamless integration into Go applications and microservices.

This SDK provides an interface to [`cyborgdb-service`](https://pypi.org/project/cyborgdb-service/) which you will need to separately install and run in order to use the SDK. For more info, please see our [docs](https://docs.cyborg.co)
## Key Features

* **End-to-End Encryption**: All vector operations maintain encryption with client-side keys
* **Full Go Type Safety**: Complete type definitions and compile-time safety
* **Batch Operations**: Efficient batch queries and upserts for high-throughput applications
* **Flexible Indexing**: Support for multiple index types (IVF, IVFPQ, IVFFlat) with customizable parameters
* **Context Support**: Built-in support for Go's context package for cancellation and timeouts
* **Performance Optimized**: Designed for high-performance Go applications

## Getting Started

To get started in minutes, check out our [Quickstart Guide](https://docs.cyborg.co/quickstart).


### Installation

1. Install `cyborgdb-service`

```bash
# Install the CyborgDB Service
pip install cyborgdb-service

# Or via Docker
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
    "crypto/rand"
    "fmt"
    "log"
    
    cyborgdb "github.com/cyborginc/cyborgdb-go"
)

func main() {
    // Initialize the client
    client, err := cyborgdb.NewClient("http://localhost:8000", "your-api-key", false)
    if err != nil {
        log.Fatal(err)
    }
    
    // Generate a 32-byte encryption key
    indexKey := make([]byte, 32)
    if _, err := rand.Read(indexKey); err != nil {
        log.Fatal(err)
    }
    
    // Create an encrypted index
    createParams := &cyborgdb.CreateIndexParams{
        IndexName: "my-index",
        IndexKey:  fmt.Sprintf("%x", indexKey),
    }
    
    index, err := client.CreateIndex(context.Background(), createParams)
    if err != nil {
        log.Fatal(err)
    }
    
    // Add encrypted vector items
    items := []cyborgdb.VectorItem{
        {
            Id:     "doc1",
            Vector: []float32{0.1, 0.2, 0.3}, // ... 128 dimensions
            Contents: stringPtr("Hello world!"),
            Metadata: map[string]interface{}{
                "category": "greeting",
                "language": "en",
            },
        },
        {
            Id:     "doc2",
            Vector: []float32{0.4, 0.5, 0.6}, // ... 128 dimensions
            Contents: stringPtr("Bonjour le monde!"),
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
    
    // Print the results
    for _, resultSet := range response.Results {
        for _, result := range resultSet {
            fmt.Printf("ID: %s, Distance: %f\n", result.Id, *result.Distance)
        }
    }
}

// Helper function for string pointers
func stringPtr(s string) *string {
    return &s
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

## Documentation

For more information on CyborgDB, see the [Cyborg Docs](https://docs.cyborg.co).

## License

The CyborgDB Go SDK is licensed under the MIT License.
