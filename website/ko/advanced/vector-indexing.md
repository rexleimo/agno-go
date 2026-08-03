---
title: 벡터 인덱싱
description: 플러그형 VectorDB, 마이그레이션 CLI, 임베딩 활용.
---

# 벡터 인덱싱 (Vector Indexing)

HNO 는 플러그형 `VectorDB` 인터페이스를 제공합니다. 기본은 Chroma, 선택적으로 Redis(`-tags redis`)를 지원하며 간단한 마이그레이션 CLI를 포함합니다.

## Provider

- ChromaDB(기본): HTTP 클라이언트(`chroma-go`)
- Redis(선택): `-tags redis` 빌드, RediSearch 불필요

## 마이그레이션 CLI

```bash
# Chroma
go run ./cmd/vectordb_migrate --action up --provider chroma --collection mycol \
  --chroma-url http://localhost:8000 --distance cosine

# Redis(선택)
go run -tags redis ./cmd/vectordb_migrate --action up --provider redis \
  --collection mycol --chroma-url localhost:6379
```

`--distance`: `l2|cosine|ip` (Chroma 기본 `l2`, Redis 기본 `cosine`)

## 임베딩

```go
embed, _ := openai.New(openai.Config{APIKey: os.Getenv("OPENAI_API_KEY"), Model: "text-embedding-3-small"})
db, _ := chromadb.New(chromadb.Config{BaseURL: "http://localhost:8000", CollectionName: "docs", EmbeddingFunction: embed})
```

또는 VLLM:

```go
emb, _ := vllm.New(vllm.Config{BaseURL: "http://localhost:8000/v1", Model: "bge-base"})
db, _ := chromadb.New(chromadb.Config{BaseURL: "http://localhost:8000", CollectionName: "docs", EmbeddingFunction: emb})
```

## 참고

- Redis Provider 및 테스트는 `-tags redis` 로 게이트됩니다.
- 선택 의존성을 활성화하지 않으면 기존 사용자에 영향이 없습니다.

