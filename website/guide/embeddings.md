---
title: Embeddings
description: Configure embeddings providers (OpenAI, VLLM) for vector search.
---

# Embeddings

Embeddings convert text into numeric vectors for similarity search. HNO provides a common interface used by vector databases (e.g., Chroma).

## OpenAI

```go
embed, _ := openai.New(openai.Config{APIKey: os.Getenv("OPENAI_API_KEY"), Model: "text-embedding-3-small"})
db, _ := chromadb.New(chromadb.Config{BaseURL: "http://localhost:8000", CollectionName: "docs", EmbeddingFunction: embed})
```

## VLLM (OpenAI-Compatible)

```go
emb, _ := vllm.New(vllm.Config{BaseURL: "http://localhost:8000/v1", Model: "bge-base"})
db, _ := chromadb.New(chromadb.Config{BaseURL: "http://localhost:8000", CollectionName: "docs", EmbeddingFunction: emb})
```

## Notes

- Batch sizes and timeouts are handled by provider implementations.
- Providers are interchangeable as long as they implement the `EmbeddingFunction` interface.

