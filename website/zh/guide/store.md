# Store - 带命名空间的长期记忆

一个刻意收小的键值记忆核：层级命名空间 + 可选的向量相似度检索——
「跨会话记住用户」的原语，不贪大。

---

## Store 是什么？

`pkg/hno/store` 是 G8 的最小核：一个接口、层级命名空间、内存后端与
Postgres 后端。它在 `[]string` 命名空间路径下存 [`Item`](#item)，注入
`vectordb.EmbeddingFunction` 后可按嵌入相似度排序召回。

```go
st := store.NewMemoryStore(embedder) // embedder: vectordb.EmbeddingFunction

_ = st.Put(ctx, &store.Item{
    Namespace: []string{"users", "u-42", "prefs"},
    Key:       "tone",
    Content:   "偏好简洁回答",
})

hits, _ := st.SearchScored(ctx, []string{"users", "u-42"}, "回复该是什么口吻？", 3)
for _, h := range hits {
    fmt.Println(h.Item.Namespace, h.Item.Key, h.Score)
}
```

## 命名空间是层级路径

命名空间是有序的 `[]string` 路径（如 `["users", "u-42", "prefs"]`）。键在
**同一命名空间内**唯一——不同命名空间下的同名键是不同条目。
`store.Composite(ns, key)` 给出规范的展平形态；`ValidateNamespace` /
`ValidateKey` 强制形状规则（空段与空键都会被拒）。

## 后端

| 后端 | 导入 | 说明 |
|---|---|---|
| `MemoryStore` | `pkg/hno/store` | 内存实现，注入 embedder，刻意不设快照面。 |
| Postgres | `pkg/hno/store/postgres` | 同一个 `Store` 接口，单表实现（迁移 `003_agentos_store.sql`）。 |

两者满足同一 [`Store` 接口](#store-interface)。换后端靠构造，不靠配置。

## 检索

`Search` 按查询文本的嵌入相似度返回 top-k；`SearchScored` 额外给出每条
得分。相似度需要 embedder——把 `vectordb.EmbeddingFunction` 传给
`NewMemoryStore`（Postgres 后端用同一注入点组合）。没有 embedder 时用
`Get`/`List`/`PutMany`——键值面可独立使用。

## 刻意还没有的

- **v3.0 不做 vectordb/chromadb/redis 适配。** 组合可行性已实测（四个表示
  损失点，外加 `Document.ID` 上的键冲突规则），经裁决延到 v3.1。在此之前
  embedder 注入点是唯一耦合。
- **不带内置 embedder。** 自备 `vectordb.EmbeddingFunction`。
- **不做 RAG 文档摄取。** 那是 `pkg/hno/knowledge` 的职责（`Load`/`Chunk`）
  ——实测与本包零重叠。

## Store 接口

```go
type Store interface {
    Get(ctx context.Context, ns []string, key string) (*Item, error)
    Put(ctx context.Context, item *Item) error
    Delete(ctx context.Context, ns []string, key string) error
    List(ctx context.Context, ns []string) ([]*Item, error)
    Search(ctx context.Context, ns []string, query string, k int) ([]*Item, error)
    SearchScored(ctx context.Context, ns []string, query string, k int) ([]SearchHit, error)
    PutMany(ctx context.Context, items []*Item) error
}
```

`Get` 对缺失键返回 `ErrNotFound`。命名空间与键的校验错误在
`Put`/`PutMany` 入口即返回。

## Item

```go
type Item struct {
    Namespace []string `json:"namespace"`
    Key       string   `json:"key"`
    Content   string   `json:"content"`
    // ... metadata、embedding、时间戳等见结构体定义
}
```

完整字段见 `pkg/hno/store/item.go`。

## 相关页面

- [Graph Engine](/zh/guide/graph-engine) —— 执行内核。
- [Human-in-the-Loop](/zh/guide/human-in-the-loop) —— 挂起与按用户记忆天然成对。
