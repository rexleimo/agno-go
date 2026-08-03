package calculator

import (
	"context"
	"fmt"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// CalculatorToolkit provides basic mathematical operations
type CalculatorToolkit struct {
	*toolkit.BaseToolkit
}

// New creates a new calculator toolkit
func New() *CalculatorToolkit {
	t := &CalculatorToolkit{
		BaseToolkit: toolkit.NewBaseToolkit("calculator"),
	}

	// Register add function
	t.RegisterFunction(&toolkit.Function{
		Name:        "add",
		Description: "Add two numbers together",
		Parameters: map[string]toolkit.Parameter{
			"a": {
				Type:        "number",
				Description: "First number",
				Required:    true,
			},
			"b": {
				Type:        "number",
				Description: "Second number",
				Required:    true,
			},
		},
		Handler: t.add,
	})

	// Register subtract function
	t.RegisterFunction(&toolkit.Function{
		Name:        "subtract",
		Description: "Subtract second number from first number",
		Parameters: map[string]toolkit.Parameter{
			"a": {
				Type:        "number",
				Description: "First number",
				Required:    true,
			},
			"b": {
				Type:        "number",
				Description: "Second number",
				Required:    true,
			},
		},
		Handler: t.subtract,
	})

	// Register multiply function
	t.RegisterFunction(&toolkit.Function{
		Name:        "multiply",
		Description: "Multiply two numbers",
		Parameters: map[string]toolkit.Parameter{
			"a": {
				Type:        "number",
				Description: "First number",
				Required:    true,
			},
			"b": {
				Type:        "number",
				Description: "Second number",
				Required:    true,
			},
		},
		Handler: t.multiply,
	})

	// Register divide function
	t.RegisterFunction(&toolkit.Function{
		Name:        "divide",
		Description: "Divide first number by second number",
		Parameters: map[string]toolkit.Parameter{
			"a": {
				Type:        "number",
				Description: "Numerator",
				Required:    true,
			},
			"b": {
				Type:        "number",
				Description: "Denominator",
				Required:    true,
			},
		},
		Handler: t.divide,
	})

	return t
}

// add adds two numbers
func (c *CalculatorToolkit) add(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	a, err := toFloat64(args["a"])
	if err != nil {
		return nil, err
	}
	b, err := toFloat64(args["b"])
	if err != nil {
		return nil, err
	}
	return a + b, nil
}

// subtract subtracts two numbers
func (c *CalculatorToolkit) subtract(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	a, err := toFloat64(args["a"])
	if err != nil {
		return nil, err
	}
	b, err := toFloat64(args["b"])
	if err != nil {
		return nil, err
	}
	return a - b, nil
}

// multiply multiplies two numbers
func (c *CalculatorToolkit) multiply(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	a, err := toFloat64(args["a"])
	if err != nil {
		return nil, err
	}
	b, err := toFloat64(args["b"])
	if err != nil {
		return nil, err
	}
	return a * b, nil
}

// divide divides two numbers
func (c *CalculatorToolkit) divide(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	a, err := toFloat64(args["a"])
	if err != nil {
		return nil, err
	}
	b, err := toFloat64(args["b"])
	if err != nil {
		return nil, err
	}
	if b == 0 {
		return nil, fmt.Errorf("division by zero")
	}
	return a / b, nil
}

// toFloat64 converts an interface{} to float64
func toFloat64(val interface{}) (float64, error) {
	switch v := val.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", val)
	}
}
