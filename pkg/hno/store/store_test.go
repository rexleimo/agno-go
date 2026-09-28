// 契约 D1–D13 的外部包测试：只用导出 API，TestP8G8_* 命名（契约 allowedTestSeam）。
package store_test

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"hash/fnv"
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/store"
	"github.com/rexleimo/agno-go/pkg/hno/vectordb"
)

var errEmbedBoom = errors.New("fake embedder: boom")

// bagEmbedder 是确定性词袋嵌入件：调用/文本计数器是 D7/D13 的判据面；
// failOn 注入失败；wideDim 让含 "WIDE" 的文本用另一维度（D7 维度不一致的合法来源）。
type bagEmbedder struct {
	mu      sync.Mutex
	dim     int
	wideDim int
	calls   int
	texts   int
	failOn  string
}

var _ vectordb.EmbeddingFunction = (*bagEmbedder)(nil)

func (b *bagEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	b.mu.Lock()
	b.calls++
	b.texts += len(texts)
	failOn := b.failOn
	b.mu.Unlock()
	out := make([][]float32, len(texts))
	for i, t := range texts {
		if failOn != "" && strings.Contains(t, failOn) {
			return nil, errEmbedBoom
		}
		out[i] = b.embed(t)
	}
	return out, nil
}

func (b *bagEmbedder) EmbedSingle(_ context.Context, text string) ([]float32, error) {
	b.mu.Lock()
	b.calls++
	b.texts++
	failOn := b.failOn
	b.mu.Unlock()
	if failOn != "" && strings.Contains(text, failOn) {
		return nil, errEmbedBoom
	}
	return b.embed(text), nil
}

func (b *bagEmbedder) Calls() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.calls
}

func (b *bagEmbedder) embed(text string) []float32 {
	dim := b.dim
	if dim == 0 {
		dim = 32
	}
	low := strings.ToLower(text)
	if b.wideDim > 0 && strings.Contains(low, "wide") {
		dim = b.wideDim
		low = strings.ReplaceAll(low, "wide", "")
	}
	v := make([]float32, dim)
	for _, tok := range strings.Fields(low) {
		tok = strings.Trim(tok, ".,;:!?\"'()")
		if tok == "" {
			continue
		}
		h := fnv.New32a()
		_, _ = h.Write([]byte(tok))
		v[int(h.Sum32())%dim]++
	}
	var n float64
	for _, x := range v {
		n += float64(x) * float64(x)
	}
	if n == 0 {
		return v
	}
	for i := range v {
		v[i] = float32(float64(v[i]) / math.Sqrt(n))
	}
	return v
}

// halfDim 让查询向量维度减半（D7 换维度件路径，走 postgres 之外的内存形态用 wideDim）。
type halfDim struct{ inner *bagEmbedder }

var _ vectordb.EmbeddingFunction = (*halfDim)(nil)

func (h *halfDim) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	vs, err := h.inner.Embed(ctx, texts)
	for i := range vs {
		vs[i] = vs[i][:len(vs[i])/2]
	}
	return vs, err
}

func (h *halfDim) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	v, err := h.inner.EmbedSingle(ctx, text)
	if err != nil {
		return nil, err
	}
	return v[:len(v)/2], nil
}

func newStore(emb vectordb.EmbeddingFunction) *store.MemoryStore {
	return store.NewMemoryStore(emb)
}

func mustPut(t *testing.T, st store.Store, ns []string, key, value string) {
	t.Helper()
	if err := st.Put(context.Background(), &store.Item{Namespace: ns, Key: key, Value: []byte(value)}); err != nil {
		t.Fatalf("Put(%s): %v", key, err)
	}
}

// D1：Store 恰七方法、草图五方法签名逐字不变、Item 恰五字段。
func TestP8G8_ShapeAnchor(t *testing.T) {
	typ := reflect.TypeOf((*store.Store)(nil)).Elem()
	if typ.NumMethod() != 7 {
		t.Fatalf("Store method count = %d, want 7", typ.NumMethod())
	}
	want := map[string]string{
		"Get":          "func(context.Context, []string, string) (*store.Item, error)",
		"Put":          "func(context.Context, *store.Item) error",
		"Delete":       "func(context.Context, []string, string) error",
		"List":         "func(context.Context, []string) ([]*store.Item, error)",
		"Search":       "func(context.Context, []string, string, int) ([]*store.Item, error)",
		"SearchScored": "func(context.Context, []string, string, int) ([]store.SearchHit, error)",
		"PutMany":      "func(context.Context, []*store.Item) error",
	}
	if typ.NumMethod() != len(want) {
		t.Fatalf("method set = %d, want %d", typ.NumMethod(), len(want))
	}
	for name, sig := range want {
		m, ok := reflect.TypeOf((*store.Store)(nil)).Elem().MethodByName(name)
		if !ok {
			t.Fatalf("missing method %s", name)
		}
		if m.Type.String() != sig {
			t.Errorf("%s signature = %s, want %s", name, m.Type.String(), sig)
		}
	}
	it := reflect.TypeOf(store.Item{})
	if it.NumField() != 5 {
		t.Fatalf("Item fields = %d, want 5", it.NumField())
	}
	for i, f := range []string{"Key", "Namespace", "Value", "CreatedAt", "UpdatedAt"} {
		if it.Field(i).Name != f {
			t.Errorf("Item field %d = %s, want %s", i, it.Field(i).Name, f)
		}
	}
	v, _ := it.FieldByName("Value")
	if v.Type.Kind() != reflect.Slice || v.Type.Elem().Kind() != reflect.Uint8 {
		t.Errorf("Item.Value kind = %v/%v, want []byte", v.Type.Kind(), v.Type.Elem().Kind())
	}
	h := reflect.TypeOf(store.SearchHit{})
	if h.NumField() != 2 {
		t.Errorf("SearchHit fields = %d, want 2", h.NumField())
	}
	var _ store.Store = (*store.MemoryStore)(nil)
}

// D2：注入仓库现成 vectordb.EmbeddingFunction 后五方法全链路可用。
func TestP8G8_EmbedderInjectionRoundTrip(t *testing.T) {
	ctx := context.Background()
	emb := &bagEmbedder{dim: 32}
	st := newStore(emb)
	ns := []string{"users", "profiles"}
	mustPut(t, st, ns, "likes", "coffee")
	got, err := st.Get(ctx, ns, "likes")
	if err != nil || string(got.Value) != "coffee" {
		t.Fatalf("Get = %v, %v", got, err)
	}
	items, err := st.List(ctx, ns)
	if err != nil || len(items) != 1 {
		t.Fatalf("List = %d, %v", len(items), err)
	}
	hits, err := st.Search(ctx, ns, "espresso coffee", 1)
	if err != nil || len(hits) != 1 || hits[0].Key != "likes" {
		t.Fatalf("Search = %+v, %v", hits, err)
	}
	if err := st.Delete(ctx, ns, "likes"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := st.Get(ctx, ns, "likes"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Get after Delete = %v, want ErrNotFound", err)
	}
}

// D2：nil 注入时四个非检索方法照常，Search/SearchScored fail-closed 报 ErrNoEmbedder。
func TestP8G8_NilEmbedderFailClosed(t *testing.T) {
	ctx := context.Background()
	st := newStore(nil)
	ns := []string{"u"}
	mustPut(t, st, ns, "k", "v")
	if _, err := st.Get(ctx, ns, "k"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if _, err := st.List(ctx, ns); err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, err := st.Search(ctx, ns, "v", 1); !errors.Is(err, store.ErrNoEmbedder) {
		t.Fatalf("Search = %v, want ErrNoEmbedder", err)
	}
	if _, err := st.SearchScored(ctx, ns, "v", 1); !errors.Is(err, store.ErrNoEmbedder) {
		t.Fatalf("SearchScored = %v, want ErrNoEmbedder", err)
	}
	if err := st.Delete(ctx, ns, "k"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := st.PutMany(ctx, []*store.Item{{Namespace: ns, Key: "m", Value: []byte("v")}}); err != nil {
		t.Fatalf("PutMany with nil embedder: %v", err)
	}
}

// D3：三类错误互斥可区分；未命中一律 ErrNotFound（Get 与 Delete 双夹具）。
func TestP8G8_ErrorTaxonomy(t *testing.T) {
	ctx := context.Background()
	st := newStore(&bagEmbedder{dim: 32})
	if _, err := st.Get(ctx, []string{"a"}, "nope"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("Get miss = %v, want ErrNotFound", err)
	}
	if err := st.Delete(ctx, []string{"a"}, "nope"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("Delete miss = %v, want ErrNotFound", err)
	}
	if err := st.Put(ctx, &store.Item{Namespace: []string{"a", ""}, Key: "k", Value: []byte("v")}); !errors.Is(err, store.ErrInvalidNamespace) {
		t.Errorf("empty segment = %v, want ErrInvalidNamespace", err)
	} else if errors.Is(err, store.ErrNotFound) {
		t.Errorf("invalid namespace masked as ErrNotFound")
	}
	if err := st.Put(ctx, &store.Item{Namespace: []string{"a"}, Key: "", Value: []byte("v")}); !errors.Is(err, store.ErrInvalidKey) {
		t.Errorf("empty key = %v, want ErrInvalidKey", err)
	}
	if err := st.Put(ctx, nil); !errors.Is(err, store.ErrInvalidItem) {
		t.Errorf("nil item = %v, want ErrInvalidItem", err)
	}
	if _, err := st.Search(ctx, []string{"a"}, "q", 0); !errors.Is(err, store.ErrInvalidSearch) {
		t.Errorf("k=0 = %v, want ErrInvalidSearch", err)
	} else if errors.Is(err, store.ErrNotFound) {
		t.Errorf("invalid search masked as ErrNotFound")
	}
}

// D4：(ns,key) 复合身份——同名 Key 跨 namespace 各自独立存在；规范编码无碰撞；
// naive '.' 压平必撞（反向对照）；根层合法；空段拒绝。
func TestP8G8_CompositeIdentity(t *testing.T) {
	ctx := context.Background()
	st := newStore(&bagEmbedder{dim: 32})
	mustPut(t, st, []string{"users", "alice"}, "likes", "alice coffee")
	mustPut(t, st, []string{"users", "bob"}, "likes", "bob coffee")
	a, err := st.List(ctx, []string{"users", "alice"})
	if err != nil || len(a) != 1 || string(a[0].Value) != "alice coffee" {
		t.Fatalf("alice list = %+v, %v", a, err)
	}
	b, _ := st.List(ctx, []string{"users", "bob"})
	if len(b) != 1 || string(b[0].Value) != "bob coffee" {
		t.Fatalf("bob list = %+v", b)
	}
	if store.Composite([]string{"a.b", "c"}, "k") == store.Composite([]string{"a", "b.c"}, "k") {
		t.Fatal("canonical encoding collided")
	}
	if strings.Join([]string{"a.b", "c"}, ".") == strings.Join([]string{"a", "b.c"}, ".") {
		t.Log("naive join collides exactly as the adjudication incident predicted")
	}
	root := []string{}
	mustPut(t, st, root, "k", "root value")
	if got, err := st.Get(ctx, root, "k"); err != nil || string(got.Value) != "root value" {
		t.Fatalf("root ns Get = %v, %v", got, err)
	}
	if err := st.Put(ctx, &store.Item{Namespace: []string{"a", ""}, Key: "k", Value: []byte("v")}); !errors.Is(err, store.ErrInvalidNamespace) {
		t.Fatalf("empty segment = %v, want ErrInvalidNamespace", err)
	}
}

// D5/ADJ-4：Value 字节恒等往返（含非法 UTF-8）；非文本条目不进检索候选集。
func TestP8G8_BinaryValueRoundTripNotSearchable(t *testing.T) {
	ctx := context.Background()
	st := newStore(&bagEmbedder{dim: 32})
	ns := []string{"u"}
	raw := []byte{0xff, 0xfe, 0x00}
	if err := st.Put(ctx, &store.Item{Namespace: ns, Key: "blob", Value: raw}); err != nil {
		t.Fatalf("Put binary: %v", err)
	}
	mustPut(t, st, ns, "likes", "coffee espresso")
	got, err := st.Get(ctx, ns, "blob")
	if err != nil || string(got.Value) != string(raw) {
		t.Fatalf("binary round trip = %v, %v", got, err)
	}
	hits, err := st.SearchScored(ctx, ns, "coffee", 10)
	if err != nil {
		t.Fatalf("SearchScored: %v", err)
	}
	if len(hits) != 1 || hits[0].Item.Key != "likes" {
		t.Fatalf("binary entry leaked into candidates: %+v", hits)
	}
}

// D6：按相似度降序、k 语义（k<=0 报错、k>命中数返全部）、确定性序、Search 只读。
// 键序与语义序刻意错开（Key 升序 a_runs < m_jazz < z_coffee），关键词/零分冒充打分的实现活不过本行。
func TestP8G8_SearchRankingAndK(t *testing.T) {
	ctx := context.Background()
	st := newStore(&bagEmbedder{dim: 32})
	ns := []string{"users", "profiles"}
	mustPut(t, st, ns, "a_runs", "the user runs marathons")
	mustPut(t, st, ns, "z_coffee", "the user prefers espresso")
	mustPut(t, st, ns, "m_jazz", "the user listens to jazz")
	hits, err := st.Search(ctx, ns, "espresso coffee", 1)
	if err != nil || len(hits) != 1 || hits[0].Key != "z_coffee" {
		t.Fatalf("top-1 = %+v, %v, want z_coffee", hits, err)
	}
	if _, err := st.Search(ctx, ns, "coffee", 0); !errors.Is(err, store.ErrInvalidSearch) {
		t.Fatalf("k=0 = %v, want ErrInvalidSearch", err)
	}
	all, err := st.Search(ctx, ns, "coffee", 999)
	if err != nil || len(all) != 3 {
		t.Fatalf("k>n = %d, %v, want 3", len(all), err)
	}
	again, err := st.Search(ctx, ns, "coffee", 999)
	if err != nil {
		t.Fatalf("second search: %v", err)
	}
	for i := range all {
		if all[i].Key != again[i].Key {
			t.Fatalf("order unstable at %d: %s vs %s", i, all[i].Key, again[i].Key)
		}
	}
	got, _ := st.Get(ctx, ns, "z_coffee")
	if got.UpdatedAt.IsZero() {
		t.Fatal("item vanished after search")
	}
}

// D7：嵌入调用数——写入期派生物理列，查询代价与存量行数无关（行数二倍，Search 侧增量恒 0）。
func TestP8G8_EmbedCallBudgetOnWrite(t *testing.T) {
	ctx := context.Background()
	emb := &bagEmbedder{dim: 32}
	st := newStore(emb)
	ns := []string{"u"}
	for i := 0; i < 200; i++ {
		mustPut(t, st, ns, fmtKey(i), "fact")
	}
	if got := emb.Calls(); got != 200 {
		t.Fatalf("after 200 puts calls = %d, want 200", got)
	}
	for i := 0; i < 10; i++ {
		if _, err := st.Search(ctx, ns, "fact", 3); err != nil {
			t.Fatal(err)
		}
	}
	if got := emb.Calls(); got != 210 {
		t.Fatalf("after 10 searches calls = %d, want 210", got)
	}
	for i := 200; i < 400; i++ {
		mustPut(t, st, ns, fmtKey(i), "fact")
	}
	for i := 0; i < 10; i++ {
		if _, err := st.Search(ctx, ns, "fact", 3); err != nil {
			t.Fatal(err)
		}
	}
	if got := emb.Calls(); got != 420 {
		t.Fatalf("after doubling rows calls = %d, want 420", got)
	}
}

func fmtKey(i int) string { return fmt.Sprintf("k%03d", i) }

// D7：维度不一致必须上交 ErrDimensionMismatch，不静默乱序。
func TestP8G8_DimensionMismatchSurfaces(t *testing.T) {
	ctx := context.Background()
	st := newStore(&bagEmbedder{dim: 32, wideDim: 16})
	ns := []string{"u"}
	mustPut(t, st, ns, "narrow", "tiny WIDE fact")
	mustPut(t, st, ns, "wide", "big fact")
	if _, err := st.Search(ctx, ns, "fact", 5); !errors.Is(err, store.ErrDimensionMismatch) {
		t.Fatalf("Search = %v, want ErrDimensionMismatch", err)
	}
}

// D12：Search 与 SearchScored 同集合同序，只差分数；分数单调不增。
func TestP8G8_SearchScoredEquivalence(t *testing.T) {
	ctx := context.Background()
	st := newStore(&bagEmbedder{dim: 32})
	ns := []string{"users", "profiles"}
	mustPut(t, st, ns, "coffee", "the user prefers espresso")
	mustPut(t, st, ns, "runs", "the user runs marathons")
	mustPut(t, st, ns, "jazz", "the user listens to jazz")
	mustPut(t, st, ns, "blob", "\xff\xfe\x00")
	plain, err := st.Search(ctx, ns, "espresso coffee", 2)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	scored, err := st.SearchScored(ctx, ns, "espresso coffee", 2)
	if err != nil {
		t.Fatalf("SearchScored: %v", err)
	}
	if len(plain) != len(scored) {
		t.Fatalf("lengths diverge: %d vs %d", len(plain), len(scored))
	}
	for i := range plain {
		if plain[i].Key != scored[i].Item.Key {
			t.Fatalf("order diverges at %d: %s vs %s", i, plain[i].Key, scored[i].Item.Key)
		}
	}
	for i := 1; i < len(scored); i++ {
		if scored[i].Score > scored[i-1].Score {
			t.Fatalf("scores not monotone: %v > %v", scored[i-1].Score, scored[i].Score)
		}
	}
	if _, err := st.SearchScored(ctx, ns, "q", 0); !errors.Is(err, store.ErrInvalidSearch) {
		t.Fatalf("k=0 = %v, want ErrInvalidSearch", err)
	}
	if _, err := st.SearchScored(ctx, []string{"a", ""}, "q", 1); !errors.Is(err, store.ErrInvalidNamespace) {
		t.Fatalf("bad ns = %v, want ErrInvalidNamespace", err)
	}
}

// D13：PutMany 整批一次嵌入（调用数与条数解耦）、整批原子、错误点名条目。
func TestP8G8_PutManyBatchSemantics(t *testing.T) {
	ctx := context.Background()
	emb := &bagEmbedder{dim: 32}
	st := newStore(emb)
	ns := []string{"bulk"}
	batch := make([]*store.Item, 0, 200)
	for i := 0; i < 200; i++ {
		batch = append(batch, &store.Item{Namespace: ns, Key: fmtKey(i), Value: []byte("fact")})
	}
	if err := st.PutMany(ctx, batch); err != nil {
		t.Fatalf("PutMany: %v", err)
	}
	if got := emb.Calls(); got != 1 {
		t.Fatalf("PutMany(200) embed calls = %d, want 1", got)
	}
	small := batch[:100]
	if err := st.PutMany(ctx, small); err != nil {
		t.Fatalf("PutMany(100): %v", err)
	}
	if got := emb.Calls(); got != 2 {
		t.Fatalf("after PutMany(100) calls = %d, want 2", got)
	}
	oneByOne := newStore(&bagEmbedder{dim: 32})
	for _, it := range batch {
		if err := oneByOne.Put(ctx, it); err != nil {
			t.Fatal(err)
		}
	}
	a, err := st.SearchScored(ctx, ns, "fact", 5)
	if err != nil {
		t.Fatal(err)
	}
	b, err := oneByOne.SearchScored(ctx, ns, "fact", 5)
	if err != nil {
		t.Fatal(err)
	}
	for i := range a {
		if a[i].Item.Key != b[i].Item.Key {
			t.Fatalf("PutMany result diverges at %d: %s vs %s", i, a[i].Item.Key, b[i].Item.Key)
		}
	}
	bad := append([]*store.Item{}, batch[:3]...)
	bad[2] = &store.Item{Namespace: []string{"n", ""}, Key: "bad3", Value: []byte("v")}
	fresh := newStore(&bagEmbedder{dim: 32})
	err = fresh.PutMany(ctx, bad)
	if !errors.Is(err, store.ErrInvalidNamespace) || !strings.Contains(err.Error(), "bad3") {
		t.Fatalf("invalid item err = %v, want ErrInvalidNamespace naming bad3", err)
	}
	if rows, _ := fresh.List(ctx, []string{"n"}); len(rows) != 0 {
		t.Fatalf("partial batch landed: %d rows", len(rows))
	}
	boom := &bagEmbedder{dim: 32, failOn: "BOOM"}
	boomStore := newStore(boom)
	err = boomStore.PutMany(ctx, []*store.Item{
		{Namespace: ns, Key: "ok1", Value: []byte("fine")},
		{Namespace: ns, Key: "bad", Value: []byte("BOOM")},
		{Namespace: ns, Key: "ok2", Value: []byte("fine too")},
	})
	if !errors.Is(err, errEmbedBoom) {
		t.Fatalf("embed failure err = %v, want errEmbedBoom", err)
	}
	if rows, _ := boomStore.List(ctx, ns); len(rows) != 0 {
		t.Fatalf("embed failure left partial rows: %d", len(rows))
	}
}

// D6/ADJ-3：List 全量返回、不静默截断、两次调用序一致。
func TestP8G8_ListFullAndDeterministic(t *testing.T) {
	ctx := context.Background()
	st := newStore(&bagEmbedder{dim: 32})
	ns := []string{"u"}
	for i := 0; i < 130; i++ {
		mustPut(t, st, ns, fmtKey(i), "v")
	}
	first, err := st.List(ctx, ns)
	if err != nil || len(first) != 130 {
		t.Fatalf("List = %d, %v, want 130 (no truncation)", len(first), err)
	}
	again, _ := st.List(ctx, ns)
	for i := range first {
		if first[i].Key != again[i].Key {
			t.Fatalf("List order unstable at %d", i)
		}
	}
}

// D11：导出符号集合 == 契约白名单，且不含任何 Loader/Chunk/Collection/VectorDB/Message/Session 命名。
func TestP8G8_ExportedSurface(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, 0)
	if err != nil {
		t.Fatalf("parse package dir: %v", err)
	}
	got := map[string]bool{}
	for pname, pkg := range pkgs {
		if strings.HasSuffix(pname, "_test") {
			continue
		}
		for fname, f := range pkg.Files {
			if strings.HasSuffix(fname, "_test.go") {
				continue
			}
			for _, d := range f.Decls {
				collectExported(d, got)
			}
		}
	}
	want := []string{
		"Composite", "ErrDimensionMismatch", "ErrInvalidItem", "ErrInvalidKey",
		"ErrInvalidNamespace", "ErrInvalidSearch", "ErrNoEmbedder", "ErrNotFound",
		"Item", "MaxNamespaceDepth", "MaxSegmentLen", "MemoryStore", "NewMemoryStore",
		"SearchHit", "Store", "UnitSeparator", "ValidateKey", "ValidateNamespace",
	}
	if len(got) != len(want) {
		t.Fatalf("exported count = %d, want %d: %v", len(got), len(want), keys(got))
	}
	for _, w := range want {
		if !got[w] {
			t.Errorf("missing export %s", w)
		}
	}
	for name := range got {
		for _, banned := range []string{"Loader", "Chunk", "Collection", "VectorDB", "Message", "Session"} {
			if strings.Contains(name, banned) {
				t.Errorf("exported %s crosses a responsibility boundary", name)
			}
		}
	}
}

func collectExported(d ast.Decl, into map[string]bool) {
	switch d := d.(type) {
	case *ast.FuncDecl:
		if d.Recv == nil && d.Name.IsExported() {
			into[d.Name.Name] = true
		}
	case *ast.GenDecl:
		for _, spec := range d.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				if s.Name.IsExported() {
					into[s.Name.Name] = true
				}
			case *ast.ValueSpec:
				for _, n := range s.Names {
					if n.IsExported() {
						into[n.Name] = true
					}
				}
			}
		}
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// D5 的失败注入半边：嵌入失败不留半截行（时间戳半边在同包夹具文件，用步进时钟）。
func TestP8G8_FailedPutLeavesNoPartialRow(t *testing.T) {
	ctx := context.Background()
	st := newStore(&bagEmbedder{dim: 32, failOn: "BOOM"})
	ns := []string{"u"}
	if err := st.Put(ctx, &store.Item{Namespace: ns, Key: "bad", Value: []byte("BOOM")}); !errors.Is(err, errEmbedBoom) {
		t.Fatalf("Put err = %v, want errEmbedBoom", err)
	}
	if _, err := st.Get(ctx, ns, "bad"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("partial row leaked: %v", err)
	}
}
