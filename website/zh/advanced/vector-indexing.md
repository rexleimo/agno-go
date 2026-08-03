---
title: 向量索引
description: 可插拔 VectorDB、迁移 CLI 与 Embeddings。
---

# 向量索引（Vector Indexing）

HNO 提供可插拔 `VectorDB` 接口：默认 Chroma，可选 Redis（`-tags redis`）。并包含简单迁移 CLI。

## Provider

- ChromaDB（默认）：HTTP 客户端（`chroma-go`）
- Redis（可选）：`-tags redis` 构建，无需 RediSearch

## 迁移 CLI

```bash
# Chroma
go run ./cmd/vectordb_migrate --action up --provider chroma --collection mycol \
  --chroma-url http://localhost:8000 --distance cosine

# Redis（可选）
go run -tags redis ./cmd/vectordb_migrate --action up --provider redis \
  --collection mycol --chroma-url localhost:6379
```

`--distance`: `l2|cosine|ip`（Chroma 默认 `l2`；Redis 默认 `cosine`）。

## Embeddings

```go
embed, _ := openai.New(openai.Config{APIKey: os.Getenv("OPENAI_API_KEY"), Model: "text-embedding-3-small"})
db, _ := chromadb.New(chromadb.Config{BaseURL: "http://localhost:8000", CollectionName: "docs", EmbeddingFunction: embed})
```

或使用 VLLM：

```go
emb, _ := vllm.New(vllm.Config{BaseURL: "http://localhost:8000/v1", Model: "bge-base"})
db, _ := chromadb.New(chromadb.Config{BaseURL: "http://localhost:8000", CollectionName: "docs", EmbeddingFunction: emb})
```

## 说明

- Redis Provider 与测试均通过 `-tags redis` 门控。
- 未启用可选依赖时不影响默认用户。

