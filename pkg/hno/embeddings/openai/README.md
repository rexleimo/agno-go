# OpenAI Embeddings

OpenAI embeddings implementation for Agno-Go.

## Features

- ✅ Support for all OpenAI embedding models
- ✅ Batch processing with automatic splitting
- ✅ Single and multi-text embedding
- ✅ Comprehensive error handling
- ✅ Configurable HTTP client

## Supported Models

| Model | Dimensions | Use Case | Cost |
|-------|------------|----------|------|
| `text-embedding-3-small` | 1536 | General purpose, cost-effective | $ |
| `text-embedding-3-large` | 3072 | Higher quality, better performance | $$ |
| `text-embedding-ada-002` | 1536 | Legacy model (OpenAI v1) | $ |

## Installation

```bash
go get github.com/rexleimo/agno-go
```

## Usage

### Basic Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/agno-go/pkg/hno/embeddings/openai"
)

func main() {
    // Create embedding function
    embedFunc, err := openai.New(openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "text-embedding-3-small",
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Embed single text
    embedding, err := embedFunc.EmbedSingle(ctx, "Hello, world!")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Embedding dimension: %d\n", len(embedding))
    fmt.Printf("First 5 values: %v\n", embedding[:5])
}
```

### Batch Embedding

```go
texts := []string{
    "The quick brown fox jumps over the lazy dog",
    "Machine learning is a subset of artificial intelligence",
    "Vector databases enable semantic search",
}

embeddings, err := embedFunc.Embed(ctx, texts)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Generated %d embeddings\n", len(embeddings))
for i, emb := range embeddings {
    fmt.Printf("Text %d: %d dimensions\n", i, len(emb))
}
```

### With Vector Database

```go
import (
    "github.com/rexleimo/agno-go/pkg/hno/vectordb/chromadb"
    "github.com/rexleimo/agno-go/pkg/hno/embeddings/openai"
)

// Create embedding function
embedFunc, err := openai.New(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "text-embedding-3-small",
})

// Create vector database with embedding function
db, err := chromadb.New(chromadb.Config{
    CollectionName:    "my_documents",
    EmbeddingFunction: embedFunc,
})

// Add documents (embeddings generated automatically)
documents := []vectordb.Document{
    {
        ID:      "doc1",
        Content: "ChromaDB is a vector database for AI applications",
    },
    {
        ID:      "doc2",
        Content: "OpenAI provides state-of-the-art language models",
    },
}

err = db.Add(ctx, documents)

// Query (query embedding generated automatically)
results, err := db.Query(ctx, "Tell me about vector databases", 5, nil)
```

### Custom Configuration

```go
import (
    "net/http"
    "time"
)

embedFunc, err := openai.New(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "text-embedding-3-large", // Higher quality
    BaseURL: "https://api.openai.com/v1", // Custom endpoint
    HTTPClient: &http.Client{
        Timeout: 60 * time.Second,
    },
})
```

### With Azure OpenAI

```go
embedFunc, err := openai.New(openai.Config{
    APIKey:  os.Getenv("AZURE_OPENAI_API_KEY"),
    Model:   "text-embedding-ada-002",
    BaseURL: "https://your-resource.openai.azure.com/openai/deployments/your-deployment",
})
```

## Configuration

### Config Options

```go
type Config struct {
    // APIKey for OpenAI API (required)
    APIKey string

    // Model to use (default: text-embedding-3-small)
    Model string

    // BaseURL for OpenAI API (default: https://api.openai.com/v1)
    BaseURL string

    // HTTPClient for custom HTTP configuration (optional)
    HTTPClient *http.Client
}
```

## Model Selection Guide

### text-embedding-3-small
- **Dimensions**: 1536
- **Best for**: Most use cases, cost-effective
- **Performance**: Fast, good quality
- **Cost**: ~$0.02 / 1M tokens

### text-embedding-3-large
- **Dimensions**: 3072
- **Best for**: High-quality semantic search, complex queries
- **Performance**: Higher quality, slower
- **Cost**: ~$0.13 / 1M tokens

### text-embedding-ada-002
- **Dimensions**: 1536
- **Best for**: Legacy projects, OpenAI v1 compatibility
- **Performance**: Similar to 3-small
- **Cost**: ~$0.10 / 1M tokens

## Performance Tips

### 1. Batch Processing

Process multiple texts at once for better throughput:

```go
// Good: Process in batch
embeddings, err := embedFunc.Embed(ctx, texts) // 100 texts

// Less efficient: Process one by one
for _, text := range texts {
    emb, err := embedFunc.EmbedSingle(ctx, text)
}
```

### 2. Caching

Cache embeddings to avoid redundant API calls:

```go
type EmbeddingCache struct {
    cache map[string][]float32
    embedFunc *openai.OpenAIEmbedding
}

func (c *EmbeddingCache) Embed(ctx context.Context, text string) ([]float32, error) {
    if emb, ok := c.cache[text]; ok {
        return emb, nil
    }

    emb, err := c.embedFunc.EmbedSingle(ctx, text)
    if err != nil {
        return nil, err
    }

    c.cache[text] = emb
    return emb, nil
}
```

### 3. Automatic Batching

The library automatically splits large batches (>2048 texts):

```go
// Automatically split into smaller batches
largeTexts := make([]string, 5000)
embeddings, err := embedFunc.Embed(ctx, largeTexts) // Handled internally
```

## Error Handling

```go
embeddings, err := embedFunc.Embed(ctx, texts)
if err != nil {
    // Check error type
    if strings.Contains(err.Error(), "invalid_api_key") {
        log.Fatal("Invalid API key")
    } else if strings.Contains(err.Error(), "rate_limit") {
        // Implement retry with backoff
        time.Sleep(5 * time.Second)
        embeddings, err = embedFunc.Embed(ctx, texts)
    } else {
        log.Fatalf("Embedding error: %v", err)
    }
}
```

## Testing

### Unit Tests

```bash
# Run unit tests (with mock server)
go test -v ./pkg/hno/embeddings/openai
```

### Integration Tests

```bash
# Set API key
export OPENAI_API_KEY=your-api-key

# Run integration tests
go test -v ./pkg/hno/embeddings/openai -run TestEmbedIntegration
```

## Troubleshooting

### Invalid API Key

```
Error: API error: Invalid API key (type: invalid_request_error)
```

**Solution**: Check your API key is correct and active.

### Rate Limit

```
Error: API error: Rate limit exceeded
```

**Solution**: Implement exponential backoff or reduce request frequency.

### Timeout

```
Error: failed to send request: context deadline exceeded
```

**Solution**: Increase HTTP client timeout:

```go
embedFunc, err := openai.New(openai.Config{
    APIKey: apiKey,
    HTTPClient: &http.Client{
        Timeout: 120 * time.Second,
    },
})
```

## Examples

### Calculate Similarity

```go
// Embed two texts
emb1, _ := embedFunc.EmbedSingle(ctx, "I love machine learning")
emb2, _ := embedFunc.EmbedSingle(ctx, "AI is fascinating")

// Calculate cosine similarity
similarity := cosineSimilarity(emb1, emb2)
fmt.Printf("Similarity: %.4f\n", similarity)

func cosineSimilarity(a, b []float32) float32 {
    var dotProduct, normA, normB float32
    for i := range a {
        dotProduct += a[i] * b[i]
        normA += a[i] * a[i]
        normB += b[i] * b[i]
    }
    return dotProduct / (sqrt(normA) * sqrt(normB))
}
```

### Semantic Search

```go
// Embed documents
docs := []string{
    "Python is a programming language",
    "Dogs are loyal animals",
    "Machine learning requires data",
}

docEmbeddings, _ := embedFunc.Embed(ctx, docs)

// Embed query
query := "Tell me about programming"
queryEmb, _ := embedFunc.EmbedSingle(ctx, query)

// Find most similar document
bestIdx := 0
bestScore := float32(0)

for i, docEmb := range docEmbeddings {
    score := cosineSimilarity(queryEmb, docEmb)
    if score > bestScore {
        bestScore = score
        bestIdx = i
    }
}

fmt.Printf("Most relevant: %s (score: %.4f)\n", docs[bestIdx], bestScore)
```

## Resources

- [OpenAI Embeddings Guide](https://platform.openai.com/docs/guides/embeddings)
- [OpenAI API Reference](https://platform.openai.com/docs/api-reference/embeddings)
- [Best Practices](https://platform.openai.com/docs/guides/embeddings/use-cases)

## License

Same as Agno-Go (MIT License)
