package toolkit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rexleimo/agno-go/pkg/hno/models"
)

// Function represents a callable tool function
type Function struct {
	Name        string
	Description string
	Parameters  map[string]Parameter
	Handler     HandlerFunc
}

// Parameter defines a function parameter
type Parameter struct {
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
	Default     interface{} `json:"default,omitempty"`
}

// HandlerFunc is the function signature for tool handlers
type HandlerFunc func(ctx context.Context, args map[string]interface{}) (interface{}, error)

// Toolkit defines the interface for a collection of tools
type Toolkit interface {
	// Name returns the toolkit name
	Name() string

	// Functions returns all available functions in this toolkit
	Functions() map[string]*Function
}

// BaseToolkit provides common functionality for toolkit implementations
type BaseToolkit struct {
	name      string
	functions map[string]*Function
}

// NewBaseToolkit creates a new base toolkit
func NewBaseToolkit(name string) *BaseToolkit {
	return &BaseToolkit{
		name:      name,
		functions: make(map[string]*Function),
	}
}

// Name returns the toolkit name
func (t *BaseToolkit) Name() string {
	return t.name
}

// Functions returns all functions
func (t *BaseToolkit) Functions() map[string]*Function {
	return t.functions
}

// RegisterFunction registers a new function to the toolkit
func (t *BaseToolkit) RegisterFunction(fn *Function) {
	t.functions[fn.Name] = fn
}

// Execute calls a function by name with the given arguments
func (t *BaseToolkit) Execute(ctx context.Context, fnName string, args map[string]interface{}) (interface{}, error) {
	fn, ok := t.functions[fnName]
	if !ok {
		return nil, fmt.Errorf("function %s not found in toolkit %s", fnName, t.name)
	}

	if err := ValidateArgs(fn, args); err != nil {
		return nil, err
	}

	return fn.Handler(ctx, args)
}

// ValidateArgs checks that all required parameters are present.
// ValidateArgs 检查所有必填参数是否存在。
func ValidateArgs(fn *Function, args map[string]interface{}) error {
	for paramName, param := range fn.Parameters {
		if param.Required {
			if _, exists := args[paramName]; !exists {
				return fmt.Errorf("required parameter %s missing", paramName)
			}
		}
	}
	return nil
}

// ToModelToolDefinitions converts toolkit functions to model tool definitions
func ToModelToolDefinitions(toolkits []Toolkit) []models.ToolDefinition {
	var definitions []models.ToolDefinition

	for _, toolkit := range toolkits {
		for _, fn := range toolkit.Functions() {
			// Build parameters schema
			properties := make(map[string]interface{})
			var required []string

			for paramName, param := range fn.Parameters {
				properties[paramName] = map[string]interface{}{
					"type":        param.Type,
					"description": param.Description,
				}
				if len(param.Enum) > 0 {
					properties[paramName].(map[string]interface{})["enum"] = param.Enum
				}
				if param.Required {
					required = append(required, paramName)
				}
			}

			parameters := map[string]interface{}{
				"type":       "object",
				"properties": properties,
			}
			if len(required) > 0 {
				parameters["required"] = required
			}

			definitions = append(definitions, models.ToolDefinition{
				Type: "function",
				Function: models.FunctionSchema{
					Name:        fn.Name,
					Description: fn.Description,
					Parameters:  parameters,
				},
			})
		}
	}

	return definitions
}

// ParseArguments parses JSON arguments string into a map
func ParseArguments(argsJSON string) (map[string]interface{}, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}
	return args, nil
}

// FormatResult formats a tool execution result as JSON string
func FormatResult(result interface{}) (string, error) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to format result: %w", err)
	}
	return string(resultJSON), nil
}
