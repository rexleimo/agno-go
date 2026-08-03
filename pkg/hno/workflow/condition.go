package workflow

import (
	"context"
	"fmt"
)

// ConditionFunc is a function that evaluates a condition
type ConditionFunc func(ctx *ExecutionContext) bool

// Condition represents a conditional branching node
type Condition struct {
	ID        string
	Name      string
	Condition ConditionFunc
	TrueNode  Node
	FalseNode Node
}

// ConditionConfig contains condition configuration
type ConditionConfig struct {
	ID        string
	Name      string
	Condition ConditionFunc
	TrueNode  Node
	FalseNode Node
}

// NewCondition creates a new condition node
func NewCondition(config ConditionConfig) (*Condition, error) {
	if config.Condition == nil {
		return nil, fmt.Errorf("condition function is required")
	}

	if config.ID == "" {
		config.ID = fmt.Sprintf("condition-%s", config.Name)
	}

	if config.Name == "" {
		config.Name = config.ID
	}

	return &Condition{
		ID:        config.ID,
		Name:      config.Name,
		Condition: config.Condition,
		TrueNode:  config.TrueNode,
		FalseNode: config.FalseNode,
	}, nil
}

// Execute evaluates the condition and executes the appropriate branch
func (c *Condition) Execute(ctx context.Context, execCtx *ExecutionContext) (*ExecutionContext, error) {
	// Evaluate condition
	result := c.Condition(execCtx)

	execCtx.Set(fmt.Sprintf("condition_%s_result", c.ID), result)

	// Execute appropriate branch
	if result {
		if c.TrueNode != nil {
			return c.TrueNode.Execute(ctx, execCtx)
		}
	} else {
		if c.FalseNode != nil {
			return c.FalseNode.Execute(ctx, execCtx)
		}
	}

	return execCtx, nil
}

// GetID returns the condition ID
func (c *Condition) GetID() string {
	return c.ID
}

// GetType returns the node type
func (c *Condition) GetType() NodeType {
	return NodeTypeCondition
}
