// 持久后端 sqlmock 面（契约 allowedTestSeam；对齐仓库既有持久层 sqlmock 测试的做法：
// mock 期望 + 0 skip + -race 可链接）。TestP8G8_PG_* 判据锁行为与查询形状，不锁 SQL 拼写。
package postgres_test

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"

	"github.com/rexleimo/agno-go/pkg/hno/store"
	"github.com/rexleimo/agno-go/pkg/hno/store/postgres"
	"github.com/rexleimo/agno-go/pkg/hno/vectordb"
)

var errPGEmbed = errors.New("pg fake embedder: boom")

// constEmbedder 返回固定向量（带调用计数与失败注入），让排序结果可手工预测。
type constEmbedder struct {
	dim    int
	vec    []float32
	calls  int
	failOn string
}

var _ vectordb.EmbeddingFunction = (*constEmbedder)(nil)

func (c *constEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	c.calls++
	out := make([][]float32, len(texts))
	for i, t := range texts {
		v, err := c.one(t)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

func (c *constEmbedder) EmbedSingle(_ context.Context, text string) ([]float32, error) {
	c.calls++
	return c.one(text)
}

func (c *constEmbedder) one(text string) ([]float32, error) {
	if c.failOn != "" && strings.Contains(text, c.failOn) {
		return nil, errPGEmbed
	}
	dim := c.dim
	if dim == 0 {
		dim = 8
	}
	v := make([]float32, dim)
	copy(v, c.vec)
	return v, nil
}

func encVec(v []float32) []byte {
	out := make([]byte, 4*len(v))
	for i, x := range v {
		binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(x))
	}
	return out
}

func newPG(t *testing.T, emb vectordb.EmbeddingFunction, opts ...postgres.Option) (*sql.DB, sqlmock.Sqlmock, *postgres.Storage) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	st, err := postgres.NewStorage(db, emb, opts...)
	if err != nil {
		t.Fatalf("NewStorage: %v", err)
	}
	return db, mock, st
}

// D8：五方法在 mock 驱动下往返；未命中翻译 ErrNotFound（Get 双路：sql.ErrNoRows 与 RowsAffected==0）。
func TestP8G8_PG_RoundTripAndNotFound(t *testing.T) {
	ctx := context.Background()
	db, mock, st := newPG(t, &constEmbedder{})
	defer db.Close()
	ns := []string{"users", "alice"}

	mock.ExpectExec(`INSERT INTO .+ ON CONFLICT`).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := st.Put(ctx, &store.Item{Namespace: ns, Key: "likes", Value: []byte("coffee")}); err != nil {
		t.Fatalf("Put: %v", err)
	}

	now := time.Now()
	mock.ExpectQuery(`SELECT .+ FROM`).WillReturnRows(
		sqlmock.NewRows([]string{"value", "created_at", "updated_at"}).AddRow([]byte("coffee"), now, now))
	got, err := st.Get(ctx, ns, "likes")
	if err != nil || string(got.Value) != "coffee" {
		t.Fatalf("Get = %v, %v", got, err)
	}

	mock.ExpectQuery(`SELECT .+ FROM`).WillReturnError(sql.ErrNoRows)
	if _, err := st.Get(ctx, ns, "missing"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Get miss = %v, want ErrNotFound", err)
	}

	mock.ExpectExec(`DELETE FROM`).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := st.Delete(ctx, ns, "likes"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	mock.ExpectExec(`DELETE FROM`).WillReturnResult(sqlmock.NewResult(0, 0))
	if err := st.Delete(ctx, ns, "likes"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Delete miss = %v, want ErrNotFound", err)
	}

	mock.ExpectClose()
	if err := st.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// D3：后端自身故障不得伪装成未命中——原样上交且与 ErrNotFound 可区分。
func TestP8G8_PG_BackendErrorNotMasked(t *testing.T) {
	ctx := context.Background()
	db, mock, st := newPG(t, &constEmbedder{})
	defer db.Close()
	boom := errors.New("pq: connection reset")
	mock.ExpectQuery(`SELECT .+ FROM`).WillReturnError(boom)
	if _, err := st.Get(ctx, []string{"u"}, "k"); !errors.Is(err, boom) || errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Get backend err = %v, want boom and not ErrNotFound", err)
	}
	execBoom := errors.New("pq: deadlock")
	mock.ExpectExec(`INSERT INTO .+ ON CONFLICT`).WillReturnError(execBoom)
	if err := st.Put(ctx, &store.Item{Namespace: []string{"u"}, Key: "k", Value: []byte("v")}); !errors.Is(err, execBoom) || errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Put backend err = %v, want execBoom and not ErrNotFound", err)
	}
}

// D8：标识符走 ^[a-zA-Z0-9_]+$ 校验（注入型非法表名/schema 被拒）。
func TestP8G8_PG_IdentifierValidation(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := postgres.NewStorage(db, nil, postgres.WithSchema("bad name")); err == nil {
		t.Fatal("invalid schema accepted")
	}
	if _, err := postgres.NewStorage(db, nil, postgres.WithTable(`items"; DROP SCHEMA`)); err == nil {
		t.Fatal("invalid table accepted")
	}
	if _, err := postgres.NewStorage(nil, nil); err == nil {
		t.Fatal("nil db accepted")
	}
}

// D9：一次 Search 恰一次 SELECT 取 (key,value,embedding) 行 + 应用层线性打分；
// D12：Search/SearchScored 同序；D7：维度不一致上交。
func TestP8G8_PG_SearchSingleQueryLinearScoring(t *testing.T) {
	ctx := context.Background()
	q := []float32{1, 0, 0, 0, 0, 0, 0, 0}
	emb := &constEmbedder{dim: 8, vec: q}
	db, mock, st := newPG(t, emb)
	defer db.Close()
	ns := []string{"users"}

	rows := sqlmock.NewRows([]string{"key", "value", "created_at", "updated_at", "embedding"}).
		AddRow("b_orthogonal", []byte("b"), time.Now(), time.Now(), encVec([]float32{0, 1, 0, 0, 0, 0, 0, 0})).
		AddRow("a_direct", []byte("a"), time.Now(), time.Now(), encVec(q)).
		AddRow("c_diagonal", []byte("c"), time.Now(), time.Now(), encVec([]float32{0.70711, 0.70711, 0, 0, 0, 0, 0, 0})).
		AddRow("z_binary", []byte{0xff, 0x00}, time.Now(), time.Now(), encVec(q))
	mock.ExpectQuery(`SELECT .+ FROM`).WillReturnRows(rows)

	hits, err := st.SearchScored(ctx, ns, "query", 999)
	if err != nil {
		t.Fatalf("SearchScored: %v", err)
	}
	want := []string{"a_direct", "c_diagonal", "b_orthogonal"}
	if len(hits) != len(want) {
		t.Fatalf("hits = %d, want %d (binary row excluded)", len(hits), len(want))
	}
	for i, w := range want {
		if hits[i].Item.Key != w {
			t.Fatalf("rank %d = %s, want %s", i, hits[i].Item.Key, w)
		}
	}
	for i := 1; i < len(hits); i++ {
		if hits[i].Score > hits[i-1].Score {
			t.Fatalf("scores not monotone at %d", i)
		}
	}

	rows = sqlmock.NewRows([]string{"key", "value", "created_at", "updated_at", "embedding"}).
		AddRow("b_orthogonal", []byte("b"), time.Now(), time.Now(), encVec([]float32{0, 1, 0, 0, 0, 0, 0, 0})).
		AddRow("a_direct", []byte("a"), time.Now(), time.Now(), encVec(q)).
		AddRow("c_diagonal", []byte("c"), time.Now(), time.Now(), encVec([]float32{0.70711, 0.70711, 0, 0, 0, 0, 0, 0})).
		AddRow("z_binary", []byte{0xff, 0x00}, time.Now(), time.Now(), encVec(q))
	mock.ExpectQuery(`SELECT .+ FROM`).WillReturnRows(rows)
	plain, err := st.Search(ctx, ns, "query", 2)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(plain) != 2 || plain[0].Key != "a_direct" || plain[1].Key != "c_diagonal" {
		t.Fatalf("top-2 = %+v", plain)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("search must be exactly one query per call: %v", err)
	}

	rows = sqlmock.NewRows([]string{"key", "value", "created_at", "updated_at", "embedding"}).
		AddRow("wide", []byte("w"), time.Now(), time.Now(), encVec(make([]float32, 8)))
	mock.ExpectQuery(`SELECT .+ FROM`).WillReturnRows(rows)
	narrow := &constEmbedder{dim: 4}
	narrowSt, err := postgres.NewStorage(db, narrow)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := narrowSt.Search(ctx, ns, "query", 1); !errors.Is(err, store.ErrDimensionMismatch) {
		t.Fatalf("Search = %v, want ErrDimensionMismatch", err)
	}
}

// D8：注入型 nil 嵌入件时非检索方法照常、Search fail-closed（ErrNoEmbedder）。
func TestP8G8_PG_NilEmbedderFailClosed(t *testing.T) {
	ctx := context.Background()
	db, mock, st := newPG(t, nil)
	defer db.Close()
	mock.ExpectExec(`INSERT INTO .+ ON CONFLICT`).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := st.Put(ctx, &store.Item{Namespace: []string{"u"}, Key: "k", Value: []byte("v")}); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if _, err := st.Search(ctx, []string{"u"}, "v", 1); !errors.Is(err, store.ErrNoEmbedder) {
		t.Fatalf("Search = %v, want ErrNoEmbedder", err)
	}
}

// D13：PutMany 校验失败/嵌入失败不触库（整批原子），成功路径单事务提交。
func TestP8G8_PG_PutManyAtomic(t *testing.T) {
	ctx := context.Background()
	items := []*store.Item{
		{Namespace: []string{"u"}, Key: "k1", Value: []byte("v1")},
		{Namespace: []string{"u"}, Key: "k2", Value: []byte("v2")},
		{Namespace: []string{"u"}, Key: "k3", Value: []byte("v3")},
	}

	db, mock, st := newPG(t, &constEmbedder{})
	defer db.Close()
	mock.ExpectBegin()
	for range items {
		mock.ExpectExec(`INSERT INTO .+ ON CONFLICT`).WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()
	if err := st.PutMany(ctx, items); err != nil {
		t.Fatalf("PutMany: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}

	boom := &constEmbedder{failOn: "BOOM"}
	_, mock2, st2 := newPG(t, boom)
	bad := []*store.Item{{Namespace: []string{"u"}, Key: "bad", Value: []byte("BOOM")}, items[0]}
	if err := st2.PutMany(ctx, bad); !errors.Is(err, errPGEmbed) {
		t.Fatalf("embed failure err = %v, want errPGEmbed", err)
	}
	if err := mock2.ExpectationsWereMet(); err != nil {
		t.Fatalf("embed failure must not touch the database: %v", err)
	}

	_, mock3, st3 := newPG(t, &constEmbedder{})
	invalid := []*store.Item{items[0], {Namespace: []string{"u", ""}, Key: "bad3", Value: []byte("v")}}
	err := st3.PutMany(ctx, invalid)
	if !errors.Is(err, store.ErrInvalidNamespace) || !strings.Contains(err.Error(), "bad3") {
		t.Fatalf("invalid item err = %v, want ErrInvalidNamespace naming bad3", err)
	}
	if err := mock3.ExpectationsWereMet(); err != nil {
		t.Fatalf("validation failure must not touch the database: %v", err)
	}
}

// ADJ-3：List 全量返回、顺序确定（mock 行序透传）。
func TestP8G8_PG_ListReturnsAllRows(t *testing.T) {
	ctx := context.Background()
	db, mock, st := newPG(t, &constEmbedder{})
	defer db.Close()
	now := time.Now()
	mock.ExpectQuery(`SELECT .+ FROM`).WillReturnRows(
		sqlmock.NewRows([]string{"key", "value", "created_at", "updated_at"}).
			AddRow("a", []byte("va"), now, now).
			AddRow("b", []byte("vb"), now, now))
	got, err := st.List(ctx, []string{"u"})
	if err != nil || len(got) != 2 {
		t.Fatalf("List = %d, %v, want 2", len(got), err)
	}
	if got[0].Key != "a" || got[1].Key != "b" {
		t.Fatalf("List order = %s,%s", got[0].Key, got[1].Key)
	}
	if len(got[0].Namespace) != 1 || got[0].Namespace[0] != "u" {
		t.Fatalf("List item namespace = %v", got[0].Namespace)
	}
}
