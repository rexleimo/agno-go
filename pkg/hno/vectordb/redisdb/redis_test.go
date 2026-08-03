//go:build redis

package redisdb

import (
	"context"
	"os"
	"testing"
)

func TestRedisDB_Smoke(t *testing.T) {
	if os.Getenv("TEST_REDIS_VECTORDB") != "1" {
		t.Skip("set TEST_REDIS_VECTORDB=1 to run redis vectordb test")
	}
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	db, err := New(Config{Addr: addr, CollectionName: "test-smoke"})
	if err != nil {
		t.Fatalf("new redis db: %v", err)
	}
	ctx := context.Background()
	if err := db.CreateCollection(ctx, "", nil); err != nil {
		t.Fatalf("create: %v", err)
	}
	defer db.DeleteCollection(ctx, "")
	err = db.Add(ctx, []Document{{ID: "1", Content: "hello", Embedding: []float32{0.1, 0.2, 0.3}}})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	n, err := db.Count(ctx)
	if err != nil || n < 1 {
		t.Fatalf("count: %v n=%d", err, n)
	}
}
