package agent

import (
	"context"
	"fmt"

	"github.com/rexleimo/agno-go/pkg/agno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/agno/types"
)

func (a *Agent) executeToolCalls(ctx context.Context, toolCalls []types.ToolCall) error {
	for _, tc := range toolCalls {
		// Find the toolkit that has this function
		var targetToolkit toolkit.Toolkit
		for _, tk := range a.Toolkits {
			if _, exists := tk.Functions()[tc.Function.Name]; exists {
				targetToolkit = tk
				break
			}
		}

		if targetToolkit == nil {
			errMsg := fmt.Sprintf("function %s not found in any toolkit", tc.Function.Name)
			a.logger.Warn("tool not found", "function", tc.Function.Name)
			a.Memory.Add(types.NewToolMessage(tc.ID, errMsg), a.UserID)
			continue
		}

		// Parse arguments
		args, err := toolkit.ParseArguments(tc.Function.Arguments)
		if err != nil {
			errMsg := fmt.Sprintf("failed to parse arguments: %v", err)
			a.logger.Error("argument parsing failed", "error", err)
			a.Memory.Add(types.NewToolMessage(tc.ID, errMsg), a.UserID)
			continue
		}

		// Execute tool
		a.logger.Info("executing tool", "function", tc.Function.Name, "args", args)

		// Get the function and execute it directly
		fn := targetToolkit.Functions()[tc.Function.Name]
		if fn == nil {
			errMsg := fmt.Sprintf("function %s not found", tc.Function.Name)
			a.logger.Error("function not found", "function", tc.Function.Name)
			a.Memory.Add(types.NewToolMessage(tc.ID, errMsg), a.UserID)
			continue
		}

		result, err := fn.Handler(ctx, args)
		if err != nil {
			errMsg := fmt.Sprintf("tool execution error: %v", err)
			a.logger.Error("tool execution failed", "function", tc.Function.Name, "error", err)
			a.Memory.Add(types.NewToolMessage(tc.ID, errMsg), a.UserID)
			continue
		}

		// Format and store result
		resultStr, err := toolkit.FormatResult(result)
		if err != nil {
			resultStr = fmt.Sprintf("%v", result)
		}

		a.logger.Info("tool executed successfully", "function", tc.Function.Name)
		a.Memory.Add(types.NewToolMessage(tc.ID, resultStr), a.UserID)
	}

	return nil
}

// ClearMemory clears the agent's conversation history for this user
