package workflow

import (
	"context"
	"errors"
)

// Step represents a unit of work; it returns a map of outputs for chaining.
type Step func(ctx context.Context, input map[string]any) (map[string]any, error)

// Engine executes steps sequentially with optional branch selection.
type Engine struct {
	steps []Step
}

// New builds an Engine with the provided steps in order.
func New(steps ...Step) *Engine {
	return &Engine{steps: steps}
}

// Run executes steps; if any step errors, execution stops and the error bubbles up.
func (e *Engine) Run(ctx context.Context, initial map[string]any) (map[string]any, error) {
	if e == nil || len(e.steps) == 0 {
		return nil, errors.New("workflow has no steps")
	}
	payload := initial
	for _, step := range e.steps {
		out, err := step(ctx, payload)
		if err != nil {
			return nil, err
		}
		if out != nil {
			payload = out
		}
	}
	return payload, nil
}
