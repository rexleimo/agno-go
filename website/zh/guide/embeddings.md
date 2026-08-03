---
title: Embeddings 嵌入
description: 配置 OpenAI/VLLM 嵌入以支持向量检索。
---

# Embeddings 嵌入

嵌入将文本转换为向量，用于相似度检索。HNO 提供统一接口，供 Chroma 等向量数据库调用。

## OpenAI

```go
embed, _ := openai.New(openai.Config{APIKey: os.Getenv("OPENAI_API_KEY"), Model: "text-embedding-3-small"})
db, _ := chromadb.New(chromadb.Config{BaseURL: "http://localhost:8000", CollectionName: "docs", EmbeddingFunction: embed})
```

## VLLM（OpenAI 兼容）

```go
emb, _ := vllm.New(vllm.Config{BaseURL: "http://localhost:8000/v1", Model: "bge-base"})
db, _ := chromadb.New(chromadb.Config{BaseURL: "http://localhost:8000", CollectionName: "docs", EmbeddingFunction: emb})
```

## 说明

- Provider 需实现 `EmbeddingFunction` 接口，彼此可替换。
- 批量与超时由具体 Provider 实现负责。

