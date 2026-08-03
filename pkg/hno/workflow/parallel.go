package workflow

import (
	"context"
	"fmt"
	"sync"
)

// Parallel represents a node that executes multiple nodes concurrently
type Parallel struct {
	ID    string
	Name  string
	Nodes []Node
}

// ParallelConfig contains parallel configuration
type ParallelConfig struct {
	ID    string
	Name  string
	Nodes []Node
}

// NewParallel creates a new parallel node
func NewParallel(config ParallelConfig) (*Parallel, error) {
	if len(config.Nodes) == 0 {
		return nil, fmt.Errorf("parallel node requires at least one child node")
	}

	if config.ID == "" {
		config.ID = fmt.Sprintf("parallel-%s", config.Name)
	}

	if config.Name == "" {
		config.Name = config.ID
	}

	return &Parallel{
		ID:    config.ID,
		Name:  config.Name,
		Nodes: config.Nodes,
	}, nil
}

// Execute runs all child nodes in parallel
// Execute 并行运行所有子节点
func (p *Parallel) Execute(ctx context.Context, execCtx *ExecutionContext) (*ExecutionContext, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)
	results := make([]*ExecutionContext, len(p.Nodes))

	// Clone session state for each parallel branch to avoid race conditions
	// 为每个并行分支克隆会话状态以避免竞态条件
	sessionStateCopies := make([]*SessionState, len(p.Nodes))
	for i := range p.Nodes {
		if execCtx.SessionState != nil {
			sessionStateCopies[i] = execCtx.SessionState.Clone()
		} else {
			sessionStateCopies[i] = NewSessionState()
		}
	}

	for i, node := range p.Nodes {
		wg.Add(1)
		go func(idx int, n Node) {
			defer wg.Done()

			// Create a copy of the execution context for each parallel branch
			// 为每个并行分支创建执行上下文的副本
			branchCtx := &ExecutionContext{
				Input:        execCtx.Input,
				Output:       execCtx.Output,
				Data:         make(map[string]interface{}),
				Metadata:     make(map[string]interface{}),
				SessionState: sessionStateCopies[idx], // Use cloned session state
				SessionID:    execCtx.SessionID,
				UserID:       execCtx.UserID,
			}

			// Copy data
			// 复制数据
			for k, v := range execCtx.Data {
				branchCtx.Data[k] = v
			}

			// Copy metadata
			// 复制元数据
			for k, v := range execCtx.Metadata {
				branchCtx.Metadata[k] = v
			}

			// Execute node
			// 执行节点
			result, err := n.Execute(ctx, branchCtx)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
				return
			}

			mu.Lock()
			results[idx] = result
			mu.Unlock()
		}(i, node)
	}

	wg.Wait()

	// Check for errors
	// 检查错误
	if len(errors) > 0 {
		return nil, fmt.Errorf("parallel execution failed: %w", errors[0])
	}

	// Collect session states from all branches
	// 从所有分支收集会话状态
	modifiedSessionStates := make([]*SessionState, 0, len(results))
	for _, result := range results {
		if result != nil && result.SessionState != nil {
			modifiedSessionStates = append(modifiedSessionStates, result.SessionState)
		}
	}

	// Merge session states from parallel branches
	// 合并并行分支的会话状态
	if len(modifiedSessionStates) > 0 {
		originalSessionState := execCtx.SessionState
		if originalSessionState == nil {
			originalSessionState = NewSessionState()
		}
		execCtx.SessionState = MergeParallelSessionStates(originalSessionState, modifiedSessionStates)
	}

	// Merge results back into main execution context
	// Store each branch result in the context
	// 将结果合并回主执行上下文
	for i, result := range results {
		if result != nil {
			execCtx.Set(fmt.Sprintf("parallel_%s_branch_%d_output", p.ID, i), result.Output)
			// Merge data with prefix to avoid conflicts
			// 使用前缀合并数据以避免冲突
			for k, v := range result.Data {
				execCtx.Set(fmt.Sprintf("parallel_%s_branch_%d_%s", p.ID, i, k), v)
			}
		}
	}

	// Use the last result's output as the main output
	// 使用最后一个结果的输出作为主输出
	if len(results) > 0 && results[len(results)-1] != nil {
		execCtx.Output = results[len(results)-1].Output
	}

	return execCtx, nil
}

// GetID returns the parallel node ID
func (p *Parallel) GetID() string {
	return p.ID
}

// GetType returns the node type
func (p *Parallel) GetType() NodeType {
	return NodeTypeParallel
}
