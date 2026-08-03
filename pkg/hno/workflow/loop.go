package workflow

import (
	"context"
	"fmt"
)

// LoopConditionFunc determines whether to continue looping
type LoopConditionFunc func(ctx *ExecutionContext, iteration int) bool

// Loop represents a loop node that repeats execution
type Loop struct {
	ID           string
	Name         string
	Body         Node
	Condition    LoopConditionFunc
	MaxIteration int
}

// LoopConfig contains loop configuration
type LoopConfig struct {
	ID           string
	Name         string
	Body         Node
	Condition    LoopConditionFunc
	MaxIteration int
}

// NewLoop creates a new loop node
func NewLoop(config LoopConfig) (*Loop, error) {
	if config.Body == nil {
		return nil, fmt.Errorf("loop body is required")
	}

	if config.Condition == nil {
		return nil, fmt.Errorf("loop condition is required")
	}

	if config.ID == "" {
		config.ID = fmt.Sprintf("loop-%s", config.Name)
	}

	if config.Name == "" {
		config.Name = config.ID
	}

	if config.MaxIteration <= 0 {
		config.MaxIteration = 10 // Default max iterations
	}

	return &Loop{
		ID:           config.ID,
		Name:         config.Name,
		Body:         config.Body,
		Condition:    config.Condition,
		MaxIteration: config.MaxIteration,
	}, nil
}

// Execute runs the loop
func (l *Loop) Execute(ctx context.Context, execCtx *ExecutionContext) (*ExecutionContext, error) {
	iteration := 0

	for iteration < l.MaxIteration {
		// Check if we should continue
		if !l.Condition(execCtx, iteration) {
			break
		}

		// Execute loop body
		result, err := l.Body.Execute(ctx, execCtx)
		if err != nil {
			return nil, fmt.Errorf("loop %s iteration %d failed: %w", l.ID, iteration, err)
		}

		execCtx = result
		iteration++
	}

	// Store loop metadata
	execCtx.Set(fmt.Sprintf("loop_%s_iterations", l.ID), iteration)

	return execCtx, nil
}

// GetID returns the loop ID
func (l *Loop) GetID() string {
	return l.ID
}

// GetType returns the node type
func (l *Loop) GetType() NodeType {
	return NodeTypeLoop
}
