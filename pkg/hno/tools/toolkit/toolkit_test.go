package toolkit

import (
	"context"
	"testing"
)

// MockToolkit for testing
type MockToolkit struct {
	*BaseToolkit
}

func NewMockToolkit() *MockToolkit {
	t := &MockToolkit{
		BaseToolkit: NewBaseToolkit("mock"),
	}

	t.RegisterFunction(&Function{
		Name:        "test_function",
		Description: "A test function",
		Parameters: map[string]Parameter{
			"input": {
				Type:        "string",
				Description: "Test input",
				Required:    true,
			},
		},
		Handler: t.testHandler,
	})

	return t
}

func (m *MockToolkit) testHandler(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	input, _ := args["input"].(string)
	return "processed: " + input, nil
}

func TestNewBaseToolkit(t *testing.T) {
	toolkit := NewBaseToolkit("test")

	if toolkit.Name() != "test" {
		t.Errorf("Name() = %v, want test", toolkit.Name())
	}

	if toolkit.Functions() == nil {
		t.Error("Functions() should not be nil")
	}

	if len(toolkit.Functions()) != 0 {
		t.Errorf("Functions() count = %v, want 0", len(toolkit.Functions()))
	}
}

func TestBaseToolkit_RegisterFunction(t *testing.T) {
	toolkit := NewBaseToolkit("test")

	fn := &Function{
		Name:        "add",
		Description: "Add two numbers",
		Parameters: map[string]Parameter{
			"a": {Type: "number", Required: true},
			"b": {Type: "number", Required: true},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			a, _ := args["a"].(float64)
			b, _ := args["b"].(float64)
			return a + b, nil
		},
	}

	toolkit.RegisterFunction(fn)

	functions := toolkit.Functions()
	if len(functions) != 1 {
		t.Errorf("Functions() count = %v, want 1", len(functions))
	}

	if _, exists := functions["add"]; !exists {
		t.Error("add function not found")
	}
}

func TestBaseToolkit_Execute(t *testing.T) {
	toolkit := NewMockToolkit()

	result, err := toolkit.Execute(context.Background(), "test_function", map[string]interface{}{
		"input": "hello",
	})

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	if result != "processed: hello" {
		t.Errorf("Execute() result = %v, want 'processed: hello'", result)
	}
}

func TestBaseToolkit_Execute_FunctionNotFound(t *testing.T) {
	toolkit := NewMockToolkit()

	_, err := toolkit.Execute(context.Background(), "non_existent", map[string]interface{}{})

	if err == nil {
		t.Error("Execute() should return error for non-existent function")
	}
}

func TestParseArguments(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid JSON",
			input:   `{"a": 1, "b": 2}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   `{invalid}`,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true, // Empty string is invalid JSON
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseArguments(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseArguments() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("ParseArguments() should return non-nil map")
			}
		})
	}
}

func TestFormatResult(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    string
		wantErr bool
	}{
		{
			name:    "string result",
			input:   "hello",
			want:    `"hello"`, // JSON encoding adds quotes
			wantErr: false,
		},
		{
			name:    "number result",
			input:   42,
			want:    "42",
			wantErr: false,
		},
		{
			name:    "map result",
			input:   map[string]interface{}{"key": "value"},
			want:    `{"key":"value"}`,
			wantErr: false,
		},
		{
			name:    "array result",
			input:   []int{1, 2, 3},
			want:    "[1,2,3]",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatResult(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.want {
				t.Errorf("FormatResult() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestToModelToolDefinitions(t *testing.T) {
	toolkit1 := NewMockToolkit()
	toolkit2 := NewBaseToolkit("empty")

	toolkits := []Toolkit{toolkit1, toolkit2}

	definitions := ToModelToolDefinitions(toolkits)

	if len(definitions) != 1 { // Only toolkit1 has functions
		t.Errorf("ToModelToolDefinitions() count = %v, want 1", len(definitions))
	}

	if definitions[0].Type != "function" {
		t.Errorf("ToModelToolDefinitions() type = %v, want function", definitions[0].Type)
	}

	if definitions[0].Function.Name != "test_function" {
		t.Errorf("ToModelToolDefinitions() name = %v, want test_function", definitions[0].Function.Name)
	}
}

func TestToModelToolDefinitions_Empty(t *testing.T) {
	var toolkits []Toolkit

	definitions := ToModelToolDefinitions(toolkits)

	if len(definitions) != 0 {
		t.Errorf("ToModelToolDefinitions() count = %v, want 0", len(definitions))
	}
}

func TestToModelToolDefinitions_DetailedCheck(t *testing.T) {
	toolkit := NewMockToolkit()

	defs := ToModelToolDefinitions([]Toolkit{toolkit})

	if len(defs) != 1 {
		t.Fatalf("Expected 1 definition, got %d", len(defs))
	}

	def := defs[0]

	if def.Type != "function" {
		t.Errorf("Type = %v, want function", def.Type)
	}

	if def.Function.Name != "test_function" {
		t.Errorf("Name = %v, want test_function", def.Function.Name)
	}

	params := def.Function.Parameters
	if params == nil {
		t.Fatal("Parameters should not be nil")
	}

	if params["type"] != "object" {
		t.Errorf("Parameters type = %v, want object", params["type"])
	}

	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Properties should be a map")
	}

	if len(props) != 1 {
		t.Errorf("Properties count = %v, want 1", len(props))
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("Required should be a string array")
	}

	if len(required) != 1 || required[0] != "input" {
		t.Errorf("Required = %v, want [input]", required)
	}
}

func TestBaseToolkit_MultipleRegistrations(t *testing.T) {
	toolkit := NewBaseToolkit("test")

	fn1 := &Function{
		Name:    "func1",
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) { return "1", nil },
	}

	fn2 := &Function{
		Name:    "func2",
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) { return "2", nil },
	}

	toolkit.RegisterFunction(fn1)
	toolkit.RegisterFunction(fn2)

	functions := toolkit.Functions()
	if len(functions) != 2 {
		t.Errorf("Functions() count = %v, want 2", len(functions))
	}

	result1, _ := toolkit.Execute(context.Background(), "func1", map[string]interface{}{})
	if result1 != "1" {
		t.Errorf("func1 result = %v, want 1", result1)
	}

	result2, _ := toolkit.Execute(context.Background(), "func2", map[string]interface{}{})
	if result2 != "2" {
		t.Errorf("func2 result = %v, want 2", result2)
	}
}
