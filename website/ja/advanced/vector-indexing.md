---
title: ベクターインデックス
description: プラガブルな VectorDB、マイグレーション CLI、埋め込みの利用。
---

# ベクターインデックス（Vector Indexing）

HNO はプラガブルな `VectorDB` インターフェースを提供します。デフォルトは Chroma、オプションで Redis（`-tags redis`）。簡易マイグレーション CLI 付き。

## Provider

- ChromaDB（デフォルト）：HTTP クライアント（`chroma-go`）
- Redis（オプション）：`-tags redis` でビルド、RediSearch 不要

## マイグレーション CLI

```bash
# Chroma
go run ./cmd/vectordb_migrate --action up --provider chroma --collection mycol \
  --chroma-url http://localhost:8000 --distance cosine

# Redis（オプション）
go run -tags redis ./cmd/vectordb_migrate --action up --provider redis \
  --collection mycol --chroma-url localhost:6379
```

`--distance`: `l2|cosine|ip`（Chroma は `l2` が既定、Redis は `cosine`）。

## 埋め込み（Embeddings）

```go
embed, _ := openai.New(openai.Config{APIKey: os.Getenv("OPENAI_API_KEY"), Model: "text-embedding-3-small"})
db, _ := chromadb.New(chromadb.Config{BaseURL: "http://localhost:8000", CollectionName: "docs", EmbeddingFunction: embed})
```

VLLM（OpenAI 互換）：

```go
emb, _ := vllm.New(vllm.Config{BaseURL: "http://localhost:8000/v1", Model: "bge-base"})
db, _ := chromadb.New(chromadb.Config{BaseURL: "http://localhost:8000", CollectionName: "docs", EmbeddingFunction: emb})
```

## メモ

- Redis Provider とテストは `-tags redis` でゲートされます。
- オプション依存を有効化しなければ既存ユーザに影響しません。

