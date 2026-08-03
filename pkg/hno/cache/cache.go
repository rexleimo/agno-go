package cache

import (
	"context"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// Provider 定义了响应缓存的最小接口。
type Provider interface {
	// Get 根据 key 获取缓存的模型响应。
	Get(ctx context.Context, key string) (*types.ModelResponse, bool, error)

	// Set 写入缓存，ttl<=0 时使用实现的默认 TTL。
	Set(ctx context.Context, key string, value *types.ModelResponse, ttl time.Duration) error
}
