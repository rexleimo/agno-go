package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

func (a *Agent) tryCacheGet(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, string, bool, error) {
	if !a.cacheEnabled || a.cache == nil {
		return nil, "", false, nil
	}

	key := a.buildCacheKey(req)
	resp, ok, err := a.cache.Get(ctx, key)
	return resp, key, ok, err
}

func (a *Agent) tryCacheSet(ctx context.Context, key string, resp *types.ModelResponse) {
	if !a.cacheEnabled || a.cache == nil || key == "" || resp == nil {
		return
	}

	if resp.HasToolCalls() {
		return
	}

	if err := a.cache.Set(ctx, key, resp, a.cacheTTL); err != nil && a.logger != nil {
		a.logger.Warn("failed to cache response", "error", err)
	}
}

func (a *Agent) buildCacheKey(req *models.InvokeRequest) string {
	if req == nil {
		return ""
	}

	var builder strings.Builder
	builder.Grow(256)

	builder.WriteString(a.Model.GetProvider())
	builder.WriteString(":")
	builder.WriteString(a.Model.GetID())

	for _, msg := range req.Messages {
		builder.WriteString("|")
		builder.WriteString(string(msg.Role))
		builder.WriteString(":")
		builder.WriteString(msg.Content)
		if len(msg.ToolCalls) > 0 {
			builder.WriteString("#toolcalls")
		}
	}

	if len(req.Tools) > 0 {
		builder.WriteString("|tools:")
		for _, tool := range req.Tools {
			builder.WriteString(tool.Function.Name)
			builder.WriteString(",")
		}
	}

	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:])
}
