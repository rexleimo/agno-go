package models

import (
	"encoding/json"
	"testing"
)

func TestToInt(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  int
		ok    bool
	}{
		{"int", 42, 42, true},
		{"int64", int64(42), 42, true},
		{"float64", 42.0, 42, true},
		{"float64 fractional", 42.7, 42, true},
		{"uint", uint(42), 42, true},
		{"string", "42", 42, true},
		{"json.Number", json.Number("42"), 42, true},
		{"empty string", "", 0, false},
		{"non-numeric string", "abc", 0, false},
		{"bool", true, 0, false},
		{"nil", nil, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ToInt(tt.value)
			if ok != tt.ok {
				t.Errorf("ToInt(%v) ok = %v, want %v", tt.value, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("ToInt(%v) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}

func TestToBool(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  bool
		ok    bool
	}{
		{"bool true", true, true, true},
		{"bool false", false, false, true},
		{"string true", "true", true, true},
		{"int nonzero", 1, true, true},
		{"int zero", 0, false, true},
		{"float nonzero", 2.5, true, true},
		{"empty string", "", false, false},
		{"bad string", "maybe", false, false},
		{"nil", nil, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ToBool(tt.value)
			if ok != tt.ok {
				t.Errorf("ToBool(%v) ok = %v, want %v", tt.value, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("ToBool(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestNewToolCallID(t *testing.T) {
	a := NewToolCallID()
	b := NewToolCallID()
	if a == "" || b == "" {
		t.Fatal("tool call IDs must not be empty")
	}
	if a == b {
		t.Errorf("expected unique IDs, got %q twice", a)
	}
}

func TestParseArguments(t *testing.T) {
	tests := []struct {
		name string
		args string
		want map[string]interface{}
		err  bool
	}{
		{"empty", "", map[string]interface{}{}, false},
		{"valid", `{"a":1}`, map[string]interface{}{"a": float64(1)}, false},
		{"invalid", `{not json`, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseArguments(tt.args)
			if tt.err {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Errorf("ParseArguments(%q) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestNewToolCall(t *testing.T) {
	tc := NewToolCall("call_1", "get_weather", `{"city":"Beijing"}`)
	if tc.ID != "call_1" {
		t.Errorf("ID = %q, want call_1", tc.ID)
	}
	if tc.Type != "function" {
		t.Errorf("Type = %q, want function", tc.Type)
	}
	if tc.Function.Name != "get_weather" {
		t.Errorf("Name = %q, want get_weather", tc.Function.Name)
	}
	if tc.Function.Arguments != `{"city":"Beijing"}` {
		t.Errorf("Arguments = %q", tc.Function.Arguments)
	}
}

func TestMarshalMap(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		want string
	}{
		{"nil", nil, "{}"},
		{"empty", map[string]interface{}{}, "{}"},
		{"simple", map[string]interface{}{"a": 1}, `{"a":1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MarshalMap(tt.m)
			if got != tt.want {
				t.Errorf("MarshalMap(%v) = %q, want %q", tt.m, got, tt.want)
			}
		})
	}
}
