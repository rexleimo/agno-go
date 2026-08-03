package workflow

import (
	"context"
	"fmt"
)

// RouterFunc determines which node to execute
type RouterFunc func(ctx *ExecutionContext) string

// Router represents a dynamic routing node
type Router struct {
	ID     string
	Name   string
	Router RouterFunc
	Routes map[string]Node
}

// RouterConfig contains router configuration
type RouterConfig struct {
	ID     string
	Name   string
	Router RouterFunc
	Routes map[string]Node
}

// NewRouter creates a new router node
func NewRouter(config RouterConfig) (*Router, error) {
	if config.Router == nil {
		return nil, fmt.Errorf("router function is required")
	}

	if len(config.Routes) == 0 {
		return nil, fmt.Errorf("router requires at least one route")
	}

	if config.ID == "" {
		config.ID = fmt.Sprintf("router-%s", config.Name)
	}

	if config.Name == "" {
		config.Name = config.ID
	}

	return &Router{
		ID:     config.ID,
		Name:   config.Name,
		Router: config.Router,
		Routes: config.Routes,
	}, nil
}

// Execute evaluates the router and executes the selected route
func (r *Router) Execute(ctx context.Context, execCtx *ExecutionContext) (*ExecutionContext, error) {
	// Determine route
	routeKey := r.Router(execCtx)

	execCtx.Set(fmt.Sprintf("router_%s_selected", r.ID), routeKey)

	// Execute selected route
	node, exists := r.Routes[routeKey]
	if !exists {
		return nil, fmt.Errorf("router %s: route '%s' not found", r.ID, routeKey)
	}

	if node == nil {
		// No-op route, just return context
		return execCtx, nil
	}

	return node.Execute(ctx, execCtx)
}

// GetID returns the router ID
func (r *Router) GetID() string {
	return r.ID
}

// GetType returns the node type
func (r *Router) GetType() NodeType {
	return NodeTypeRouter
}
