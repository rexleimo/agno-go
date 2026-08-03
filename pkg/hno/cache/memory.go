package cache

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

const (
	defaultCapacity  = 1024
	defaultMemoryTTL = 5 * time.Minute
)

// MemoryProvider 基于 expirable LRU 的内存缓存实现。
type MemoryProvider struct {
	cache *expirable.LRU[string, *types.ModelResponse]
	mu    sync.RWMutex
	ttl   time.Duration
}

// NewMemoryProvider 创建内存缓存。
func NewMemoryProvider(capacity int, ttl time.Duration) (*MemoryProvider, error) {
	if capacity <= 0 {
		capacity = defaultCapacity
	}
	if ttl <= 0 {
		ttl = defaultMemoryTTL
	}

	lruCache := expirable.NewLRU[string, *types.ModelResponse](capacity, nil, ttl)
	return &MemoryProvider{cache: lruCache, ttl: ttl}, nil
}

// Get 实现 Provider 接口。
func (p *MemoryProvider) Get(ctx context.Context, key string) (*types.ModelResponse, bool, error) {
	if err := ensureContext(ctx); err != nil {
		return nil, false, err
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	value, ok := p.cache.Get(key)
	if !ok {
		return nil, false, nil
	}

	return cloneResponse(value), true, nil
}

// Set 实现 Provider 接口。
func (p *MemoryProvider) Set(ctx context.Context, key string, value *types.ModelResponse, ttl time.Duration) error {
	if err := ensureContext(ctx); err != nil {
		return err
	}
	if value == nil {
		return errors.New("cache value cannot be nil")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// expirable.LRU 在构造时已设置 TTL，此处忽略自定义 ttl 参数。
	p.cache.Add(key, cloneResponse(value))
	return nil
}

func ensureContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func cloneResponse(resp *types.ModelResponse) *types.ModelResponse {
	if resp == nil {
		return nil
	}

	clone := *resp

	if len(resp.ToolCalls) > 0 {
		clone.ToolCalls = make([]types.ToolCall, len(resp.ToolCalls))
		for i, tc := range resp.ToolCalls {
			clone.ToolCalls[i] = cloneToolCall(tc)
		}
	}

	if resp.Metadata.Extra != nil {
		clone.Metadata.Extra = cloneMap(resp.Metadata.Extra)
	}

	if resp.ReasoningContent != nil {
		rc := *resp.ReasoningContent
		clone.ReasoningContent = &rc
		if resp.ReasoningContent.RedactedContent != nil {
			redacted := *resp.ReasoningContent.RedactedContent
			clone.ReasoningContent.RedactedContent = &redacted
		}
		if resp.ReasoningContent.TokenCount != nil {
			count := *resp.ReasoningContent.TokenCount
			clone.ReasoningContent.TokenCount = &count
		}
	}

	return &clone
}

func cloneToolCall(tc types.ToolCall) types.ToolCall {
	clone := tc
	if tc.Metadata != nil {
		clone.Metadata = cloneMap(tc.Metadata)
	}
	return clone
}

func cloneMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	clone := make(map[string]interface{}, len(src))
	for k, v := range src {
		clone[k] = v
	}
	return clone
}
