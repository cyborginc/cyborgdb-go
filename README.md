# CyborgDB Go SDK

The **CyborgDB Go SDK** provides a comprehensive client library for interacting with [CyborgDB](https://www.cyborg.co), the first Confidential Vector Database. This SDK enables you to perform encrypted vector operations including ingestion, search, and retrieval while maintaining end-to-end encryption of your vector embeddings. Built with Go's strong typing system, it offers excellent performance and seamless integration into Go applications and microservices.

This SDK provides an interface to `cyborgdb-service` which you will need to separately install and run in order to use the SDK. For more info, please see our [docs](https://docs.cyborg.co)

**Why CyborgDB?**

Vector Search powers critical AI applications like RAG systems, recommendation engines, and semantic search. The CyborgDB Go SDK brings confidential computing to your Go applications and services, ensuring vector embeddings remain encrypted throughout their entire lifecycle while providing fast, accurate search capabilities.

**Key Features**

* **End-to-End Encryption**: All vector operations maintain encryption with client-side keys
* **Full Go Type Safety**: Complete type definitions and compile-time safety
* **Batch Operations**: Efficient batch queries and upserts for high-throughput applications
* **Flexible Indexing**: Support for multiple index types (IVF, IVFPQ, IVFFlat) with customizable parameters
* **Context Support**: Built-in support for Go's context package for cancellation and timeouts
* **Performance Optimized**: Designed for high-performance Go applications

**Installation**

1. Install `cyborgdb-service`

2. Install the CyborgDB Go SDK:

```bash
go get github.com/cyborginc/cyborgdb-go
```

**Usage**

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
    
    // Create an encrypted index with IVFFlat configuration
    indexModel := &cyborgdb.IndexIVFFlat{
        Dimension: 128,
        Metric:    "euclidean",
        NLists:    1024,
    }
    
    index, err := client.CreateIndex(context.Background(), "my-index", indexKey, indexModel, nil)
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
    response, err := index.Query(context.Background(), queryVector, 10)
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

**Advanced Usage**

**Batch Queries**

```go
// Search with multiple query vectors simultaneously
queryVectors := [][]float32{
    {0.1, 0.2, 0.3}, // ... first vector
    {0.4, 0.5, 0.6}, // ... second vector
}

batchResults, err := index.Query(context.Background(), queryVectors, 5)
if err != nil {
    log.Fatal(err)
}
```

**Using QueryRequest for Complex Queries**

```go
// Search with metadata filters using QueryRequest
queryReq := &cyborgdb.QueryRequest{
    QueryVector: []float32{0.1, 0.2, 0.3}, // ... query vector
    TopK:        10,
    NProbes:     1,
    Greedy:      &[]bool{false}[0],
    Filters: map[string]interface{}{
        "category": "greeting",
        "language": "en",
    },
    Include: []string{"distance", "metadata", "contents"},
}

results, err := index.Query(context.Background(), queryReq)
if err != nil {
    log.Fatal(err)
}
```

**Complex Metadata Filtering**

```go
// Advanced metadata filtering with operators
complexFilter := map[string]interface{}{
    "$and": []map[string]interface{}{
        {"category": "greeting"},
        {"metadata.score": map[string]interface{}{"$gt": 0.8}},
        {"language": map[string]interface{}{"$in": []string{"en", "fr"}}},
    },
}

results, err := index.Query(
    context.Background(),
    queryVector,
    10,    // topK
    1,     // nProbes  
    false, // greedy
    complexFilter,
    []string{"distance", "metadata", "contents"},
)
```

**Getting Vectors by ID**

```go
// Retrieve specific vectors by their IDs
ids := []string{"doc1", "doc2", "doc3"}
response, err := index.Get(context.Background(), ids, []string{"vector", "metadata", "contents"})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Retrieved %d vectors\n", response.GetResultCount())
for _, result := range response.GetResults() {
    fmt.Printf("ID: %s\n", result.GetId())
    
    if result.HasMetadata() {
        fmt.Printf("  Metadata: %+v\n", result.GetMetadata())
    }
    
    if result.HasContents() {
        fmt.Printf("  Contents: %s\n", result.GetContents())
    }
    
    fmt.Printf("  Vector dimension: %d\n", result.GetVectorDimension())
}
```

**Index Training**

```go
// Train the index for better query performance (recommended for IVF indexes)
err = index.Train(context.Background(), 2048, 100, 1e-6)
if err != nil {
    log.Fatal(err)
}
```

**Error Handling**

The Go SDK uses standard Go error handling patterns:

```go
// Always check for errors
response, err := index.Query(context.Background(), queryVector, 10)
if err != nil {
    // Handle the error appropriately
    log.Printf("Query failed: %v", err)
    return
}

// Use the response
fmt.Printf("Found %d result sets\n", len(response.Results))
```

**Context and Timeouts**

All operations support Go's context package for cancellation and timeouts:

```go
// Create a context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Use the context for operations
response, err := index.Query(ctx, queryVector, 10)
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        log.Println("Query timed out")
    } else {
        log.Printf("Query failed: %v", err)
    }
    return
}
```

**Health Checking**

```go
// Check the health of the CyborgDB service
health, err := client.GetHealth(context.Background())
if err != nil {
    log.Printf("Health check failed: %v", err)
    return
}

fmt.Printf("Service status: %s\n", *health.Status)
```

**Documentation**

For more detailed documentation, visit:
* [CyborgDB Documentation](https://docs.cyborg.co/)

**License**

The CyborgDB Go SDK is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details.

**About CyborgDB**

CyborgDB is dedicated to making AI safe and secure through confidential computing. We develop solutions that enable organizations to leverage AI while maintaining the confidentiality and privacy of their data.

[Visit our website](https://www.cyborg.co/) | [Contact Us](mailto:hello@cyborg.co)
