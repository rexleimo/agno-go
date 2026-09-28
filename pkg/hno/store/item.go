// Package store 提供跨会话/跨线程的长期记忆：结构化事实的存、取、列、语义检索。
// 形状逐字取自母约 §8 草图（D1 形状锚）；嵌入件只注入仓库现成的 vectordb.EmbeddingFunction，向量是后端写入期派生的物理列，不是 Item 的字段（D2/D7）。
package store

import (
	"context"
	"time"
)

// Item 是存储的键值对。
type Item struct {
	Key       string
	Namespace []string // 层级命名空间，如 ("users","profiles")
	Value     []byte
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SearchHit 是语义检索的带分命中：条目与其相似度得分（分数语义取决于注入的嵌入件）。
type SearchHit struct {
	Item  *Item
	Score float64
}

// Store 提供跨会话/跨线程的长期记忆。检索语义（ADJ-2/3/4 裁定）：namespace 精确匹配、
// List 全量不截断、Search 只对合法 UTF-8 条目打分。
type Store interface {
	Get(ctx context.Context, ns []string, key string) (*Item, error)
	Put(ctx context.Context, item *Item) error
	Delete(ctx context.Context, ns []string, key string) error
	List(ctx context.Context, ns []string) ([]*Item, error)
	Search(ctx context.Context, ns []string, query string, k int) ([]*Item, error)
	SearchScored(ctx context.Context, ns []string, query string, k int) ([]SearchHit, error)
	PutMany(ctx context.Context, items []*Item) error
}

var _ Store = (*MemoryStore)(nil)
