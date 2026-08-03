package eval

import (
	"context"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

func TestContainsAssertion(t *testing.T) {
	a := ContainsAssertion{Expected: "hello"}
	if err := a.Check(context.Background(), "Hello world", nil); err != nil {
		t.Errorf("should pass case-insensitively: %v", err)
	}
	if err := a.Check(context.Background(), "goodbye", nil); err == nil {
		t.Error("should fail when absent")
	}

	// Empty expected always passes (no constraint).
	// 空 expected 始终通过（无约束）。
	empty := ContainsAssertion{}
	if err := empty.Check(context.Background(), "anything", nil); err != nil {
		t.Errorf("empty expected should always pass: %v", err)
	}
}

func TestRegexAssertion(t *testing.T) {
	a, err := NewRegexAssertion(`\d+ errors`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if err := a.Check(context.Background(), "found 3 errors", nil); err != nil {
		t.Errorf("should pass: %v", err)
	}
	if err := a.Check(context.Background(), "no issues", nil); err == nil {
		t.Error("should fail when no match")
	}
	if _, err := NewRegexAssertion(`(`); err == nil {
		t.Error("invalid pattern should error")
	}
}

func TestJSONSchemaAssertion(t *testing.T) {
	a := JSONSchemaAssertion{Required: map[string]string{
		"result":     "number",
		"data.items": "array",
	}}

	if err := a.Check(context.Background(), `{"result": 42, "data": {"items": [1,2]}}`, nil); err != nil {
		t.Errorf("should pass: %v", err)
	}
	if err := a.Check(context.Background(), `{"result": "42"}`, nil); err == nil {
		t.Error("should fail on type mismatch")
	}
	if err := a.Check(context.Background(), `not json`, nil); err == nil {
		t.Error("should fail on invalid JSON")
	}
	if err := a.Check(context.Background(), `{"result": 42}`, nil); err == nil {
		t.Error("should fail on missing path")
	}
}

func TestToolTraceAssertion(t *testing.T) {
	a := ToolTraceAssertion{ToolName: "get_weather"}
	if err := a.Check(context.Background(), "", []types.ToolCall{
		{Function: types.ToolCallFunction{Name: "get_weather"}},
	}); err != nil {
		t.Errorf("should pass when tool called: %v", err)
	}
	if err := a.Check(context.Background(), "", []types.ToolCall{
		{Function: types.ToolCallFunction{Name: "other"}},
	}); err == nil {
		t.Error("should fail when tool not called")
	}
}

// mockJudgeModel returns a fixed response for the judge.
// mockJudgeModel 为评委返回固定响应。
type mockJudgeModel struct {
	content string
}

func (m *mockJudgeModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	return &types.ModelResponse{Content: m.content, Model: "mock"}, nil
}
func (m *mockJudgeModel) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	ch := make(chan types.ResponseChunk)
	close(ch)
	return ch, nil
}
func (m *mockJudgeModel) GetID() string       { return "mock" }
func (m *mockJudgeModel) GetName() string     { return "mock" }
func (m *mockJudgeModel) GetProvider() string { return "mock" }

func TestLLMJudgeAssertion(t *testing.T) {
	yes := &mockJudgeModel{content: "YES, it meets the criterion"}
	no := &mockJudgeModel{content: "NO, it does not"}

	if err := (LLMJudgeAssertion{Model: yes}).Check(context.Background(), "response", nil); err != nil {
		t.Errorf("should pass when judge says YES: %v", err)
	}
	if err := (LLMJudgeAssertion{Model: no}).Check(context.Background(), "response", nil); err == nil {
		t.Error("should fail when judge says NO")
	}
}

func TestRunOne(t *testing.T) {
	// Use a model that echoes the input; assert on content.
	// 使用回显输入的模型；对内容做断言。
	m := &mockJudgeModel{content: "The result is 42"}
	ok, err := RunOne(context.Background(), m, "what is the result", ContainsAssertion{Expected: "42"})
	if err != nil || !ok {
		t.Errorf("expected pass, got ok=%v err=%v", ok, err)
	}
	ok, err = RunOne(context.Background(), m, "x", ContainsAssertion{Expected: "missing"})
	if err == nil || ok {
		t.Errorf("expected fail, got ok=%v err=%v", ok, err)
	}
}
