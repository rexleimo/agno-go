package cache

import (
	"context"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

func TestMemoryProvider_SetGet(t *testing.T) {
	provider, err := NewMemoryProvider(16, time.Minute)
	if err != nil {
		t.Fatalf("NewMemoryProvider error = %v", err)
	}

	original := &types.ModelResponse{Content: "cached", Usage: types.Usage{TotalTokens: 42}}

	if err := provider.Set(context.Background(), "key", original, 0); err != nil {
		t.Fatalf("Set error = %v", err)
	}

	resp, ok, err := provider.Get(context.Background(), "key")
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}
	if !ok {
		t.Fatalf("expected cache hit")
	}

	if resp.Content != "cached" {
		t.Fatalf("unexpected content: %v", resp.Content)
	}

	resp.Content = "mutated"
	resp.Metadata.Extra = map[string]interface{}{"foo": "bar"}

	resp2, ok, err := provider.Get(context.Background(), "key")
	if err != nil {
		t.Fatalf("Get second error = %v", err)
	}
	if !ok {
		t.Fatalf("expected cache hit")
	}
	if resp2.Content != "cached" {
		t.Fatalf("cache value mutated: %v", resp2.Content)
	}
}

func TestMemoryProvider_Expiration(t *testing.T) {
	provider, err := NewMemoryProvider(4, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("NewMemoryProvider error = %v", err)
	}

	if err := provider.Set(context.Background(), "expire", &types.ModelResponse{Content: "short"}, 0); err != nil {
		t.Fatalf("Set error = %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	_, ok, err := provider.Get(context.Background(), "expire")
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}
	if ok {
		t.Fatalf("expected entry to expire")
	}
}

func TestMemoryProvider_ContextCancelled(t *testing.T) {
	provider, err := NewMemoryProvider(4, time.Minute)
	if err != nil {
		t.Fatalf("NewMemoryProvider error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := provider.Set(ctx, "ctx", &types.ModelResponse{Content: "x"}, 0); err == nil {
		t.Fatalf("expected context error on set")
	}

	_, _, err = provider.Get(ctx, "ctx")
	if err == nil {
		t.Fatalf("expected context error on get")
	}
}
