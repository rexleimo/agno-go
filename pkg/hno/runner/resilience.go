package runner

import (
	"context"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/observability"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// G9 seam (slice 25): resilience + model-stage span wiring around the single
// TurnInvoker boundary in Run. Declared in the consumer package (runner) per
// the CacheStore precedent; consumes observability parts as-is.

// clampAttempts applies the guard-clamp precedent (WithMaxConcurrency, graph
// slice 22): MaxAttempts < 1 means exactly one attempt — never the silent
// success that observability.Retry would produce for a zero-iteration loop.
func clampAttempts(cfg *observability.RetryConfig) int {
	if cfg == nil || cfg.MaxAttempts < 1 {
		return 1
	}
	return cfg.MaxAttempts
}

// invokeTurn executes one logical model turn: breaker gate, per-attempt chat
// span with usage attribution, retry loop. Returns the raw stage error; Run
// keeps sole ownership of the "runner: model invoke: %w" wrapping.
func (r *Runner) invokeTurn(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	if r.breaker != nil && !r.breaker.Allow() {
		// Fail closed, never retried: an open breaker is a rejection, not a
		// model failure to burn attempts on. The bare sentinel keeps Run's
		// single wrap point ("runner: model invoke: %w") as the only prefix.
		return nil, observability.ErrOpen
	}

	attempts := clampAttempts(r.retry)
	var resp *types.ModelResponse
	var attemptErr error
	fn := func(ctx context.Context) error {
		spanCtx, span := observability.StartChatSpan(ctx, r.provider, r.modelName)
		resp, attemptErr = r.invoker.InvokeTurn(spanCtx, req)
		if attemptErr != nil {
			span.RecordError(attemptErr)
		} else if resp != nil {
			observability.SetUsage(span, resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
		}
		span.End()
		return attemptErr
	}

	if attempts == 1 {
		attemptErr = fn(ctx)
	} else {
		cfg := *r.retry
		cfg.MaxAttempts = attempts
		attemptErr = observability.Retry(ctx, cfg, fn)
	}

	// Breaker records ONE logical outcome per turn (post-retry), so retry
	// hides transients from the breaker instead of burning credits per attempt.
	if r.breaker != nil {
		if attemptErr != nil {
			r.breaker.Failure()
		} else {
			r.breaker.Success()
		}
	}
	return resp, attemptErr
}
