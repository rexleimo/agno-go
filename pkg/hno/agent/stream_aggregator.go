package agent

import (
	"context"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// AggregateResponseStream consumes a stream of ResponseChunk values and
// reconstructs a single ModelResponse. It concatenates content in arrival
// order and aggregates any tool calls. If a chunk carries a non-nil Error,
// aggregation stops and the error is returned.
//
// This helper is intended for future streaming Agent implementations so that
// the final assistant message is always bound to a concrete ModelResponse
// rather than remaining "unbound" to any step or history entry.
func AggregateResponseStream(ctx context.Context, ch <-chan types.ResponseChunk) (*types.ModelResponse, error) {
	if ch == nil {
		return &types.ModelResponse{}, nil
	}

	resp := &types.ModelResponse{}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case chunk, ok := <-ch:
			if !ok {
				return resp, nil
			}
			if chunk.Error != nil {
				return nil, chunk.Error
			}

			if chunk.Content != "" {
				resp.Content += chunk.Content
			}
			if len(chunk.ToolCalls) > 0 {
				resp.ToolCalls = append(resp.ToolCalls, chunk.ToolCalls...)
			}
		}
	}
}
