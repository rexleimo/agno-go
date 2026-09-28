// Package postgres 是 Store 的 Postgres 持久后端：只复用仓库既有的 database/sql 持久化模式
// （调用方注入 *sql.DB、标识符校验、ON CONFLICT upsert、驱动错误翻译为包内哨兵），不复用存量实现。
// v1 检索 = 一次取行 + 应用层线性打分（D9），原生向量索引留 v3.1。
package postgres

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"time"
	"unicode/utf8"

	"github.com/rexleimo/agno-go/pkg/hno/store"
	"github.com/rexleimo/agno-go/pkg/hno/vectordb"
)

var identifierPattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

const (
	defaultSchema = "public"
	defaultTable  = "agno_store_items"
)

// Config 控制 Postgres 后端的落表位置。
type Config struct {
	Schema string
	Table  string
}

// Option 自定义配置。
type Option func(*Config)

// WithSchema 指定 schema 名称。
func WithSchema(schema string) Option {
	return func(cfg *Config) {
		cfg.Schema = schema
	}
}

// WithTable 指定表名称。
func WithTable(table string) Option {
	return func(cfg *Config) {
		cfg.Table = table
	}
}

// Storage 实现 store.Store，持久化 (ns,key) 条目与写入期向量。一个实例绑一个嵌入件。
type Storage struct {
	db        *sql.DB
	emb       vectordb.EmbeddingFunction
	tableName string
}

// NewStorage 创建 Postgres 后端：*sql.DB 与嵌入件均由调用方注入。
func NewStorage(db *sql.DB, emb vectordb.EmbeddingFunction, opts ...Option) (*Storage, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}
	cfg := Config{Schema: defaultSchema, Table: defaultTable}
	for _, opt := range opts {
		opt(&cfg)
	}
	tableName, err := buildQualifiedName(cfg.Schema, cfg.Table)
	if err != nil {
		return nil, err
	}
	return &Storage{db: db, emb: emb, tableName: tableName}, nil
}

// Get 按 (ns,key) 精确取条目；驱动层的无行结果翻译为 store.ErrNotFound。
func (s *Storage) Get(ctx context.Context, ns []string, key string) (*store.Item, error) {
	if err := checkIdentity(ns, key); err != nil {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT value, created_at, updated_at FROM %s WHERE namespace = $1 AND key = $2`, s.tableName),
		namespaceID(ns), key)
	var value []byte
	var createdAt, updatedAt time.Time
	if err := row.Scan(&value, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, err
	}
	return &store.Item{Key: key, Namespace: cloneNS(ns), Value: value, CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

// Put 以 upsert 语义写入：嵌入发生在写入期，created_at 经 COALESCE 写一次、updated_at 取新值。
func (s *Storage) Put(ctx context.Context, item *store.Item) error {
	if err := validateItem(item); err != nil {
		return err
	}
	var vec []float32
	if s.emb != nil {
		var err error
		if vec, err = s.emb.EmbedSingle(ctx, string(item.Value)); err != nil {
			return err
		}
	}
	in := *item
	stamp(&in)
	_, err := s.db.ExecContext(ctx, s.upsertSQL(), upsertArgs(&in, vec)...)
	return err
}

// Delete 按 (ns,key) 精确删除；零行受影响翻译为 store.ErrNotFound。
func (s *Storage) Delete(ctx context.Context, ns []string, key string) error {
	if err := checkIdentity(ns, key); err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx,
		fmt.Sprintf(`DELETE FROM %s WHERE namespace = $1 AND key = $2`, s.tableName),
		namespaceID(ns), key)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return store.ErrNotFound
	}
	return nil
}

// List 返回 namespace 精确匹配下的全量条目（不截断），按 Key 升序确定输出。
func (s *Storage) List(ctx context.Context, ns []string) ([]*store.Item, error) {
	if err := store.ValidateNamespace(ns); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT key, value, created_at, updated_at FROM %s WHERE namespace = $1 ORDER BY key ASC`, s.tableName),
		namespaceID(ns))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*store.Item, 0)
	for rows.Next() {
		var key string
		var value []byte
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&key, &value, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		out = append(out, &store.Item{Key: key, Namespace: cloneNS(ns), Value: value, CreatedAt: createdAt, UpdatedAt: updatedAt})
	}
	return out, rows.Err()
}

// Search 语义检索：按相似度降序返回前 k 条，不带分数（SearchScored 的去分形态）。
func (s *Storage) Search(ctx context.Context, ns []string, query string, k int) ([]*store.Item, error) {
	hits, err := s.SearchScored(ctx, ns, query, k)
	if err != nil {
		return nil, err
	}
	out := make([]*store.Item, len(hits))
	for i, h := range hits {
		out[i] = h.Item
	}
	return out, nil
}

// SearchScored 是 Search 的带分形态（ADJ-5=B）：同集合同序，只多分数。恰一次 SELECT +
// 应用层线性打分（D9）；非 UTF-8 条目不进候选集（ADJ-4=A）；维度不一致上交（D7）。
func (s *Storage) SearchScored(ctx context.Context, ns []string, query string, k int) ([]store.SearchHit, error) {
	if err := store.ValidateNamespace(ns); err != nil {
		return nil, err
	}
	if k <= 0 {
		return nil, store.ErrInvalidSearch
	}
	if s.emb == nil {
		return nil, store.ErrNoEmbedder
	}
	qvec, err := s.emb.EmbedSingle(ctx, query)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT key, value, created_at, updated_at, embedding FROM %s WHERE namespace = $1`, s.tableName),
		namespaceID(ns))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	hits := make([]store.SearchHit, 0)
	for rows.Next() {
		var key string
		var value, embedding []byte
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&key, &value, &createdAt, &updatedAt, &embedding); err != nil {
			return nil, err
		}
		if !utf8.Valid(value) || len(embedding) == 0 {
			continue
		}
		vec := decodeVec(embedding)
		if len(vec) != len(qvec) {
			return nil, store.ErrDimensionMismatch
		}
		hits = append(hits, store.SearchHit{Item: &store.Item{Key: key, Namespace: cloneNS(ns), Value: value, CreatedAt: createdAt, UpdatedAt: updatedAt}, Score: cosine(qvec, vec)})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
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

// PutMany 批量写入（ADJ-6=B）：整批恰一次嵌入调用；任一条目非法或嵌入失败即整批拒绝且不触库，错误点名该条 Key；写入走单事务，失败即回滚（整批原子）。
func (s *Storage) PutMany(ctx context.Context, items []*store.Item) error {
	if len(items) == 0 {
		return nil
	}
	for i, item := range items {
		if err := validateItem(item); err != nil {
			return fmt.Errorf("store: putmany item %d (key %q): %w", i, item.Key, err)
		}
	}
	vecs := make([][]float32, len(items))
	if s.emb != nil {
		texts := make([]string, len(items))
		for i, item := range items {
			texts[i] = string(item.Value)
		}
		var err error
		vecs, err = s.emb.Embed(ctx, texts)
		if err != nil {
			return err
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	for i, item := range items {
		in := *item
		stamp(&in)
		if _, err := tx.ExecContext(ctx, s.upsertSQL(), upsertArgs(&in, vecs[i])...); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// Close 关闭注入的连接池。
func (s *Storage) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Storage) upsertSQL() string {
	return fmt.Sprintf(`INSERT INTO %s (namespace, key, value, embedding, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (namespace, key) DO UPDATE SET value = EXCLUDED.value, embedding = EXCLUDED.embedding, created_at = COALESCE(%s.created_at, EXCLUDED.created_at), updated_at = EXCLUDED.updated_at`, s.tableName, s.tableName)
}

func upsertArgs(item *store.Item, vec []float32) []interface{} {
	var embedding []byte
	if len(vec) > 0 {
		embedding = encodeVec(vec)
	}
	return []interface{}{namespaceID(item.Namespace), item.Key, item.Value, embedding, item.CreatedAt, item.UpdatedAt}
}

func stamp(in *store.Item) {
	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now()
	}
	if in.UpdatedAt.IsZero() {
		in.UpdatedAt = in.CreatedAt
	}
}

// namespaceID 把层级 ns 编码为无碰撞的物理键前缀（复合主键的 namespace 列）。
func namespaceID(ns []string) string { return store.Composite(ns, "") }

func cloneNS(ns []string) []string { return append([]string(nil), ns...) }

func validateItem(item *store.Item) error {
	if item == nil || item.Value == nil {
		return store.ErrInvalidItem
	}
	if err := store.ValidateNamespace(item.Namespace); err != nil {
		return err
	}
	return store.ValidateKey(item.Key)
}

func checkIdentity(ns []string, key string) error {
	if err := store.ValidateNamespace(ns); err != nil {
		return err
	}
	return store.ValidateKey(key)
}

func encodeVec(v []float32) []byte {
	out := make([]byte, 4*len(v))
	for i, x := range v {
		binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(x))
	}
	return out
}

func decodeVec(b []byte) []float32 {
	if len(b)%4 != 0 {
		return nil
	}
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
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

func buildQualifiedName(schema, table string) (string, error) {
	if table == "" {
		return "", fmt.Errorf("table name cannot be empty")
	}
	if schema == "" {
		schema = defaultSchema
	}
	if !identifierPattern.MatchString(schema) {
		return "", fmt.Errorf("invalid schema name: %s", schema)
	}
	if !identifierPattern.MatchString(table) {
		return "", fmt.Errorf("invalid table name: %s", table)
	}
	return fmt.Sprintf(`"%s"."%s"`, schema, table), nil
}
