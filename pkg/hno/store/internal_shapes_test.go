// 同包夹具（契约 allowedTestSeam）：计数/失败/变维假件在 store_test.go（外部包），
// 这里只放外部包无法触及的夹具——步进时钟（注入 MemoryStore.now）——以及
// 依赖它的 D5 时间戳确定性测试（禁 sleep、禁墙钟断言）。
package store

import (
	"context"
	"sync"
	"testing"
	"time"
)

// stepClock 是单调步进假时钟：每次读取前进 1s，保证两次 Put 的 UpdatedAt 严格递增。
type stepClock struct {
	mu  sync.Mutex
	cur time.Time
}

func (c *stepClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cur = c.cur.Add(time.Second)
	return c.cur
}

// TestP8G8_PutUpsertTimestamps：同 (ns,key) 二次 Put 不新增行、CreatedAt 写一次、
// UpdatedAt 每次刷新、Value 覆盖（D5；COALESCE/EXCLUDED 同族语义见持久后端）。
func TestP8G8_PutUpsertTimestamps(t *testing.T) {
	ctx := context.Background()
	clk := &stepClock{cur: time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)}
	m := NewMemoryStore(nil)
	m.now = clk.Now
	ns := []string{"users"}
	if err := m.Put(ctx, &Item{Namespace: ns, Key: "k", Value: []byte("v1")}); err != nil {
		t.Fatal(err)
	}
	g1, err := m.Get(ctx, ns, "k")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Put(ctx, &Item{Namespace: ns, Key: "k", Value: []byte("v2")}); err != nil {
		t.Fatal(err)
	}
	g2, err := m.Get(ctx, ns, "k")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := m.List(ctx, ns)
	if err != nil || len(rows) != 1 {
		t.Fatalf("upsert created %d rows, want 1 (err=%v)", len(rows), err)
	}
	if !g1.CreatedAt.Equal(g2.CreatedAt) {
		t.Fatalf("CreatedAt rewritten: %v -> %v", g1.CreatedAt, g2.CreatedAt)
	}
	if !g2.UpdatedAt.After(g1.UpdatedAt) {
		t.Fatalf("UpdatedAt not refreshed: %v -> %v", g1.UpdatedAt, g2.UpdatedAt)
	}
	if string(g2.Value) != "v2" {
		t.Fatalf("Value not overwritten: %q", g2.Value)
	}
	if g1.CreatedAt.IsZero() || g1.UpdatedAt.IsZero() {
		t.Fatal("zero timestamps stored")
	}
}
