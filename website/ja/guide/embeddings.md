---
title: Embeddings 埋め込み
description: ベクター検索のための OpenAI / VLLM 埋め込み設定。
---

# Embeddings 埋め込み

テキストを数値ベクトルに変換し、類似度検索に使用します。HNO は統一インターフェースを提供し、Chroma などの VectorDB から利用できます。

## OpenAI

```go
embed, _ := openai.New(openai.Config{APIKey: os.Getenv("OPENAI_API_KEY"), Model: "text-embedding-3-small"})
db, _ := chromadb.New(chromadb.Config{BaseURL: "http://localhost:8000", CollectionName: "docs", EmbeddingFunction: embed})
```

## VLLM（OpenAI 互換）

```go
emb, _ := vllm.New(vllm.Config{BaseURL: "http://localhost:8000/v1", Model: "bge-base"})
db, _ := chromadb.New(chromadb.Config{BaseURL: "http://localhost:8000", CollectionName: "docs", EmbeddingFunction: emb})
```

## メモ

- Provider は `EmbeddingFunction` を実装すれば相互に置換可能です。
- バッチ/タイムアウトは各 Provider に依存します。

