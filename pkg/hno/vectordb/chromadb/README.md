# ChromaDB Vector Database Integration

ChromaDB implementation for Agno-Go vector database interface.

## Features

- ✅ Full `VectorDB` interface implementation
- ✅ Support for local and cloud ChromaDB instances
- ✅ Automatic embedding generation
- ✅ Multiple distance functions (L2, Cosine, Inner Product)
- ✅ Metadata filtering
- ✅ Batch operations

## Installation

```bash
go get github.com/amikos-tech/chroma-go
```

## Prerequisites

### Local ChromaDB Server

Start ChromaDB using Docker:

```bash
docker pull chromadb/chroma
docker run -p 8000:8000 chromadb/chroma
```

Or install and run locally:

```bash
pip install chromadb
chroma run --path /db_path
```

## Usage

### Basic Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/rexleimo/agno-go/pkg/hno/vectordb/chromadb"
    "github.com/rexleimo/agno-go/pkg/hno/vectordb"
)

func main() {
    // Create ChromaDB client
    db, err := chromadb.New(chromadb.Config{
        BaseURL:        "http://localhost:8000",
        CollectionName: "my_documents",
        Database:       "default_database",
        Tenant:         "default_tenant",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    ctx := context.Background()

    // Create collection
    err = db.CreateCollection(ctx, "", map[string]interface{}{
        "description": "My document collection",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Add documents (with pre-computed embeddings)
    documents := []vectordb.Document{
        {
            ID:        "doc1",
            Content:   "ChromaDB is a vector database",
            Embedding: []float32{0.1, 0.2, 0.3, ...}, // 384-dim embedding
            Metadata: map[string]interface{}{
                "source": "documentation",
                "date":   "2025-01-01",
            },
        },
    }

    err = db.Add(ctx, documents)
    if err != nil {
        log.Fatal(err)
    }

    // Query similar documents (requires embedding function)
    results, err := db.Query(ctx, "What is ChromaDB?", 5, nil)
    if err != nil {
        log.Fatal(err)
    }

    for _, result := range results {
        fmt.Printf("ID: %s, Score: %.4f\n", result.ID, result.Score)
        fmt.Printf("Content: %s\n\n", result.Content)
    }
}
```

### With Automatic Embeddings

```go
import (
    "github.com/rexleimo/agno-go/pkg/hno/embeddings/openai"
)

// Create embedding function
embedFunc, err := openai.New(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "text-embedding-3-small",
})

// Create ChromaDB with embedding function
db, err := chromadb.New(chromadb.Config{
    BaseURL:           "http://localhost:8000",
    CollectionName:    "my_documents",
    EmbeddingFunction: embedFunc,
})

// Now you can add documents without pre-computed embeddings
documents := []vectordb.Document{
    {
        ID:      "doc1",
        Content: "ChromaDB is a vector database",
        // Embedding will be generated automatically
    },
}

err = db.Add(ctx, documents)

// Query with text (embedding generated automatically)
results, err := db.Query(ctx, "What is ChromaDB?", 5, nil)
```

### Query with Metadata Filtering

```go
// Filter by metadata
filter := map[string]interface{}{
    "source": "documentation",
    "date": map[string]interface{}{
        "$gte": "2025-01-01",
    },
}

results, err := db.Query(ctx, "vector database", 10, filter)
```

### ChromaDB Cloud

```go
db, err := chromadb.New(chromadb.Config{
    CloudAPIKey:    os.Getenv("CHROMA_API_KEY"),
    CollectionName: "my_documents",
    Database:       "my-database",
    Tenant:         "team-uuid",
})
```

### Update Documents

```go
// Update existing documents
updated := []vectordb.Document{
    {
        ID:      "doc1",
        Content: "Updated content",
        Metadata: map[string]interface{}{
            "version": 2,
            "updated": "2025-01-15",
        },
    },
}

err = db.Update(ctx, updated)
```

### Delete Documents

```go
// Delete by IDs
err = db.Delete(ctx, []string{"doc1", "doc2"})
```

### Get Documents by ID

```go
docs, err := db.Get(ctx, []string{"doc1", "doc2", "doc3"})

for _, doc := range docs {
    fmt.Printf("ID: %s\n", doc.ID)
    fmt.Printf("Content: %s\n", doc.Content)
    fmt.Printf("Metadata: %v\n\n", doc.Metadata)
}
```

### Count Documents

```go
count, err := db.Count(ctx)
fmt.Printf("Total documents: %d\n", count)
```

## Configuration

### Config Options

```go
type Config struct {
    // BaseURL is the ChromaDB server URL (default: http://localhost:8000)
    BaseURL string

    // CollectionName is the name of the collection to use (required)
    CollectionName string

    // Database name for multi-tenant setups (default: default_database)
    Database string

    // Tenant name for multi-tenant setups (default: default_tenant)
    Tenant string

    // CloudAPIKey for ChromaDB Cloud (optional)
    CloudAPIKey string

    // EmbeddingFunction for automatic embedding generation (optional)
    EmbeddingFunction vectordb.EmbeddingFunction

    // DistanceFunction for similarity search (default: L2)
    // Options: vectordb.L2, vectordb.Cosine, vectordb.InnerProduct
    DistanceFunction vectordb.DistanceFunction

    // Metadata for the collection (optional)
    Metadata map[string]interface{}
}
```

## Distance Functions

ChromaDB supports multiple distance metrics:

- **L2** (Euclidean distance): Default, good for most use cases
- **Cosine**: Measures angular similarity, good for text embeddings
- **InnerProduct**: Dot product similarity

```go
db, err := chromadb.New(chromadb.Config{
    CollectionName: "my_docs",
    Metadata: map[string]interface{}{
        "distance_function": vectordb.Cosine,
    },
})
```

## Testing

### Unit Tests

```bash
# Run unit tests (without requiring ChromaDB server)
go test -v ./pkg/hno/vectordb/chromadb

# Run integration tests (requires ChromaDB server)
go test -v ./pkg/hno/vectordb/chromadb -run TestAddAndQuery
```

### Integration Tests

To run integration tests, start ChromaDB server first:

```bash
docker run -p 8000:8000 chromadb/chroma
```

Then remove the `t.Skip()` lines in test file and run:

```bash
go test -v ./pkg/hno/vectordb/chromadb
```

## Performance Tips

1. **Batch Operations**: Add/update multiple documents at once
2. **Embedding Caching**: Cache embeddings to avoid re-computation
3. **Connection Pooling**: Reuse ChromaDB client instances
4. **Metadata Indexing**: Use metadata filters to reduce search space

## Troubleshooting

### Connection Error

```
Error: failed to create ChromaDB client: connection refused
```

**Solution**: Ensure ChromaDB server is running on the specified URL.

### Embedding Error

```
Error: embedding function required for text query
```

**Solution**: Provide an `EmbeddingFunction` in config or use `QueryWithEmbedding()`.

### Collection Not Found

```
Error: collection not initialized
```

**Solution**: Call `CreateCollection()` before performing operations.

## Resources

- [ChromaDB Official Docs](https://docs.trychroma.com/)
- [ChromaDB Go Client Docs](https://go-client.chromadb.dev/)
- [ChromaDB Go Client GitHub](https://github.com/amikos-tech/chroma-go)

## License

Same as Agno-Go (MIT License)
