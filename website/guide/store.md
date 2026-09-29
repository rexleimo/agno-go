# Store - Long-Term Memory with Namespaces

A minimal key/value memory core with hierarchical namespaces and optional
vector-similarity search — the "remember the user across sessions" primitive,
kept deliberately small.

---

## What is the Store?

`pkg/hno/store` is the G8 minimal core: one interface, hierarchical
namespaces, an in-memory backend, and a Postgres backend. It stores
[`Item`](#item) values under `[]string` namespace paths and can rank recall
by embedding similarity when you inject a
`vectordb.EmbeddingFunction`.

```go
st := store.NewMemoryStore(embedder) // embedder: vectordb.EmbeddingFunction

_ = st.Put(ctx, &store.Item{
    Namespace: []string{"users", "u-42", "prefs"},
    Key:       "tone",
    Content:   "prefers terse answers",
})

hits, _ := st.SearchScored(ctx, []string{"users", "u-42"}, "how should replies sound?", 3)
for _, h := range hits {
    fmt.Println(h.Item.Namespace, h.Item.Key, h.Score)
}
```

## Namespaces are hierarchical paths

A namespace is an ordered `[]string` path (for example
`["users", "u-42", "prefs"]`). Keys are unique **within** a namespace — the
same key under a different namespace is a different item. `store.Composite(ns,
key)` shows the canonical flattened form; `ValidateNamespace` /
`ValidateKey` enforce the shape rules (empty segments and empty keys are
rejected).

## Backends

| Backend | Import | Notes |
|---|---|---|
| `MemoryStore` | `pkg/hno/store` | In-memory, embedder-injected, `Snapshot`-free by design. |
| Postgres | `pkg/hno/store/postgres` | Same `Store` interface over a single table (migration `003_agentos_store.sql`). |

Both satisfy the same [`Store` interface](#store-interface). Swap them by
construction, not by configuration.

## Search

`Search` returns the top-k items by embedding similarity to the query text;
`SearchScored` additionally returns each hit's score. Similarity needs an
embedder — pass a `vectordb.EmbeddingFunction` to `NewMemoryStore` (the
Postgres backend composes with the same injection point). Without an
embedder, use `Get`/`List`/`PutMany` — the key/value surface works standalone.

## What Is Deliberately Not Here

- **No vectordb/chromadb/redis adapters in v3.0.** The composition was
  measured (four lossy points, plus a key-collision rule on
  `Document.ID`) and deferred to v3.1 by adjudication. Until then, the
  embedder injection point is the only coupling.
- **No built-in embedder.** Bring your own `vectordb.EmbeddingFunction`.
- **No RAG document loading.** That stays `pkg/hno/knowledge`'s job
  (`Load`/`Chunk`) — measured as zero overlap with this package.

## Store interface

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

`Get` returns `ErrNotFound` for missing keys. Namespace and key validation
errors are returned from `Put`/`PutMany` up front.

## Item

```go
type Item struct {
    Namespace []string `json:"namespace"`
    Key       string   `json:"key"`
    Content   string   `json:"content"`
    // ... metadata, embedding, timestamps per the struct definition
}
```

See `pkg/hno/store/item.go` for the full field list.

## Related Pages

- [Graph Engine](/guide/graph-engine) — the execution kernel.
- [Human-in-the-Loop](/guide/human-in-the-loop) — suspensions pair naturally
  with per-user memory.
