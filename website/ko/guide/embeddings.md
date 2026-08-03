---
title: 임베딩
description: 벡터 검색을 위한 OpenAI/VLLM 임베딩 설정.
---

# 임베딩

임베딩은 텍스트를 벡터로 변환하여 유사도 검색에 사용합니다. HNO 는 통합 인터페이스를 제공하며 Chroma 등 VectorDB 에서 사용할 수 있습니다.

## OpenAI

```go
embed, _ := openai.New(openai.Config{APIKey: os.Getenv("OPENAI_API_KEY"), Model: "text-embedding-3-small"})
db, _ := chromadb.New(chromadb.Config{BaseURL: "http://localhost:8000", CollectionName: "docs", EmbeddingFunction: embed})
```

## VLLM (OpenAI 호환)

```go
emb, _ := vllm.New(vllm.Config{BaseURL: "http://localhost:8000/v1", Model: "bge-base"})
db, _ := chromadb.New(chromadb.Config{BaseURL: "http://localhost:8000", CollectionName: "docs", EmbeddingFunction: emb})
```

## 참고

- Provider 는 `EmbeddingFunction` 인터페이스만 만족하면 상호 교체 가능합니다.
- 배치/타임아웃은 Provider 구현에 따릅니다.

