package calculator

import (
	"context"
	"testing"
)

func TestCalculatorToolkit_Add(t *testing.T) {
	calc := New()
	ctx := context.Background()

	result, err := calc.Execute(ctx, "add", map[string]interface{}{
		"a": 5.0,
		"b": 3.0,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 8.0
	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestCalculatorToolkit_Subtract(t *testing.T) {
	calc := New()
	ctx := context.Background()

	result, err := calc.Execute(ctx, "subtract", map[string]interface{}{
		"a": 10.0,
		"b": 4.0,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 6.0
	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestCalculatorToolkit_Multiply(t *testing.T) {
	calc := New()
	ctx := context.Background()

	result, err := calc.Execute(ctx, "multiply", map[string]interface{}{
		"a": 5.0,
		"b": 3.0,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 15.0
	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestCalculatorToolkit_Divide(t *testing.T) {
	calc := New()
	ctx := context.Background()

	tests := []struct {
		name    string
		a       float64
		b       float64
		want    float64
		wantErr bool
	}{
		{
			name: "normal division",
			a:    10.0,
			b:    2.0,
			want: 5.0,
		},
		{
			name:    "division by zero",
			a:       10.0,
			b:       0.0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calc.Execute(ctx, "divide", map[string]interface{}{
				"a": tt.a,
				"b": tt.b,
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
			}

			if !tt.wantErr && result != tt.want {
				t.Errorf("expected %v, got %v", tt.want, result)
			}
		})
	}
}

func TestCalculatorToolkit_IntConversion(t *testing.T) {
	calc := New()
	ctx := context.Background()

	// Test with int values
	result, err := calc.Execute(ctx, "add", map[string]interface{}{
		"a": 5,
		"b": 3,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 8.0
	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}
