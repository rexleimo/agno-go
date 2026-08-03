---
title: Vector Indexing
description: Pluggable VectorDB providers, migrations CLI, and embeddings.
---

# Vector Indexing

HNO ships a pluggable `VectorDB` interface with a Chroma provider by default and an optional Redis provider (build tag `redis`). A simple migrations CLI is included.

## Providers

- ChromaDB (default): HTTP client via `chroma-go`
- Redis (optional): build with `-tags redis` for a minimal provider without RediSearch

## Migrations CLI

Create or drop collections:

```bash
# Chroma
go run ./cmd/vectordb_migrate --action up --provider chroma --collection mycol \
  --chroma-url http://localhost:8000 --distance cosine

# Redis (optional)
go run -tags redis ./cmd/vectordb_migrate --action up --provider redis \
  --collection mycol --chroma-url localhost:6379
```

`--distance`: `l2|cosine|ip` (defaults to `l2` for Chroma; `cosine` in Redis provider).

## Embeddings

Use OpenAI or VLLM embeddings with Chroma provider:

```go
embed, _ := openai.New(openai.Config{APIKey: os.Getenv("OPENAI_API_KEY"), Model: "text-embedding-3-small"})
db, _ := chromadb.New(chromadb.Config{BaseURL: "http://localhost:8000", CollectionName: "docs", EmbeddingFunction: embed})
// db.Add / db.Query
```

VLLM (OpenAI-compatible /v1/embeddings):

```go
emb, _ := vllm.New(vllm.Config{BaseURL: "http://localhost:8000/v1", Model: "bge-base"})
db, _ := chromadb.New(chromadb.Config{BaseURL: "http://localhost:8000", CollectionName: "docs", EmbeddingFunction: emb})
```

## Notes

- Redis provider and tests are tag-gated: `-tags redis`
- Optional dependencies do not affect users who don't enable them

