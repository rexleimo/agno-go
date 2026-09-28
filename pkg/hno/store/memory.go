package store

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/rexleimo/agno-go/pkg/hno/vectordb"
)

// stored 是内存后端的内部行：Item 与写入期算出的向量（物理列形态，不进 Item）。
type stored struct {
	item Item
	vec  []float32
}

// MemoryStore 是内存后端：进程内持久性由 Postgres 后端（pkg/hno/store/postgres）以同一接口承担。
type MemoryStore struct {
	mu   sync.RWMutex
	rows map[string]*stored
	emb  vectordb.EmbeddingFunction
	now  func() time.Time
}

// NewMemoryStore 构造内存后端：嵌入件为构造期注入；emb 为 nil 时四个非检索方法照常工作，Search/SearchScored fail-closed 报 ErrNoEmbedder。
func NewMemoryStore(emb vectordb.EmbeddingFunction) *MemoryStore {
	return &MemoryStore{rows: map[string]*stored{}, emb: emb, now: time.Now}
}

// Get 按 (ns,key) 精确取条目；未命中报 ErrNotFound。
func (m *MemoryStore) Get(_ context.Context, ns []string, key string) (*Item, error) {
	if err := checkIdentity(ns, key); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.rows[Composite(ns, key)]
	if !ok {
		return nil, ErrNotFound
	}
	cp := s.item
	return &cp, nil
}

// Put 以 upsert 语义写入：CreatedAt 写一次、UpdatedAt 每次刷新、Value 覆盖；嵌入发生在写入期，失败不留半截行（D5/D7）。
func (m *MemoryStore) Put(ctx context.Context, item *Item) error {
	if err := validateItem(item); err != nil {
		return err
	}
	vec, err := m.embedOne(ctx, string(item.Value))
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.upsertLocked(*item, vec)
	return nil
}

// Delete 按 (ns,key) 精确删除；未命中报 ErrNotFound（对齐仓库先例，不做静默 no-op）。
func (m *MemoryStore) Delete(_ context.Context, ns []string, key string) error {
	if err := checkIdentity(ns, key); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	id := Composite(ns, key)
	if _, ok := m.rows[id]; !ok {
		return ErrNotFound
	}
	delete(m.rows, id)
	return nil
}

// List 返回 namespace 精确匹配下的全量条目（不截断，ADJ-3=A），按 Key 升序确定输出。
func (m *MemoryStore) List(_ context.Context, ns []string) ([]*Item, error) {
	if err := ValidateNamespace(ns); err != nil {
		return nil, err
	}
	m.mu.RLock()
	out := make([]*Item, 0)
	for _, s := range m.rows {
		if !namespaceEquals(s.item.Namespace, ns) {
			continue
		}
		cp := s.item
		out = append(out, &cp)
	}
	m.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

// Search 语义检索：按相似度降序返回前 k 条（top-k = min(k, 命中数)），不带分数。
func (m *MemoryStore) Search(ctx context.Context, ns []string, query string, k int) ([]*Item, error) {
	hits, err := m.SearchScored(ctx, ns, query, k)
	if err != nil {
		return nil, err
	}
	out := make([]*Item, len(hits))
	for i, h := range hits {
		out[i] = h.Item
	}
	return out, nil
}

// SearchScored 是 Search 的带分形态（ADJ-5=B）：同集合同序，只多分数；k<=0 报 ErrInvalidSearch；
// 查询期恰一次嵌入调用（D7/D9）；非法 UTF-8 条目不进候选集（ADJ-4=A）；Search 只读（D6）。
func (m *MemoryStore) SearchScored(ctx context.Context, ns []string, query string, k int) ([]SearchHit, error) {
	if err := ValidateNamespace(ns); err != nil {
		return nil, err
	}
	if k <= 0 {
		return nil, ErrInvalidSearch
	}
	if m.emb == nil {
		return nil, ErrNoEmbedder
	}
	qvec, err := m.emb.EmbedSingle(ctx, query)
	if err != nil {
		return nil, err
	}
	m.mu.RLock()
	hits := make([]SearchHit, 0)
	for _, s := range m.rows {
		if !namespaceEquals(s.item.Namespace, ns) || len(s.vec) == 0 || !utf8.Valid(s.item.Value) {
			continue
		}
		if len(s.vec) != len(qvec) {
			m.mu.RUnlock()
			return nil, ErrDimensionMismatch
		}
		cp := s.item
		hits = append(hits, SearchHit{Item: &cp, Score: cosine(qvec, s.vec)})
	}
	m.mu.RUnlock()
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].Item.Key < hits[j].Item.Key
		}
		return hits[i].Score > hits[j].Score
	})
	if len(hits) > k {
		hits = hits[:k]
	}
	return hits, nil
}

// PutMany 批量写入（ADJ-6=B）：整批恰一次嵌入调用（调用数与条数解耦，D13）；任一条目非法或嵌入失败即整批拒绝、不留半截行，错误点名该条 Key。
func (m *MemoryStore) PutMany(ctx context.Context, items []*Item) error {
	if len(items) == 0 {
		return nil
	}
	for i, item := range items {
		if err := validateItem(item); err != nil {
			return fmt.Errorf("store: putmany item %d (key %q): %w", i, item.Key, err)
		}
	}
	vecs := make([][]float32, len(items))
	if m.emb != nil {
		texts := make([]string, len(items))
		for i, item := range items {
			texts[i] = string(item.Value)
		}
		var err error
		vecs, err = m.emb.Embed(ctx, texts)
		if err != nil {
			return err
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, item := range items {
		m.upsertLocked(*item, vecs[i])
	}
	return nil
}

func (m *MemoryStore) embedOne(ctx context.Context, text string) ([]float32, error) {
	if m.emb == nil {
		return nil, nil
	}
	return m.emb.EmbedSingle(ctx, text)
}

func (m *MemoryStore) upsertLocked(in Item, vec []float32) {
	id := Composite(in.Namespace, in.Key)
	now := m.now()
	if in.CreatedAt.IsZero() {
		in.CreatedAt = now
	}
	if in.UpdatedAt.IsZero() {
		in.UpdatedAt = now
	}
	if prev, ok := m.rows[id]; ok {
		in.CreatedAt = prev.item.CreatedAt
		in.UpdatedAt = now
	}
	m.rows[id] = &stored{item: in, vec: vec}
}

func validateItem(item *Item) error {
	if item == nil || item.Value == nil {
		return ErrInvalidItem
	}
	if err := ValidateNamespace(item.Namespace); err != nil {
		return err
	}
	return ValidateKey(item.Key)
}

func checkIdentity(ns []string, key string) error {
	if err := ValidateNamespace(ns); err != nil {
		return err
	}
	return ValidateKey(key)
}

func cosine(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
