package agent

import (
	"context"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/hooks"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// ctxModel is a tiny model that first asks to call a tool, then completes.
type ctxModel struct {
	models.BaseModel
	step int
}

func (m *ctxModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	if m.step == 0 {
		m.step++
		return &types.ModelResponse{ToolCalls: []types.ToolCall{
			{ID: "tc1", Function: types.ToolCallFunction{Name: "ctx_probe", Arguments: "{}"}},
		}}, nil
	}
	return &types.ModelResponse{Content: "done"}, nil
}

func (m *ctxModel) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	ch := make(chan types.ResponseChunk)
	close(ch)
	return ch, nil
}

func TestRun_PropagatesRunContextToHooksAndTools(t *testing.T) {
	var seenPre, seenTool, seenPost string

	// Toolkit with a function that returns RunContextID
	tk := toolkit.NewBaseToolkit("ctx")
	tk.RegisterFunction(&toolkit.Function{
		Name:        "ctx_probe",
		Description: "echo run context id",
		Parameters:  map[string]toolkit.Parameter{},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			if id, ok := RunContextID(ctx); ok {
				seenTool = id
				return map[string]string{"run_context_id": id}, nil
			}
			return map[string]string{"run_context_id": ""}, nil
		},
	})

	pre := func(ctx context.Context, in *hooks.HookInput) error {
		if id, ok := RunContextID(ctx); ok {
			seenPre = id
		}
		return nil
	}
	post := func(ctx context.Context, in *hooks.HookInput) error {
		if id, ok := RunContextID(ctx); ok {
			seenPost = id
		}
		return nil
	}

	ag, err := New(Config{
		Model:     &ctxModel{BaseModel: models.BaseModel{ID: "m", Provider: "mock"}},
		Toolkits:  []toolkit.Toolkit{tk},
		PreHooks:  []hooks.Hook{hooks.HookFunc(pre)},
		PostHooks: []hooks.Hook{hooks.HookFunc(post)},
	})
	if err != nil {
		t.Fatalf("new agent: %v", err)
	}

	ctx := WithRunContext(context.Background(), "rc-test-1")
	out, err := ag.Run(ctx, "hello")
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if out == nil || out.Content != "done" {
		t.Fatalf("unexpected output: %#v", out)
	}

	if seenPre != "rc-test-1" || seenTool != "rc-test-1" || seenPost != "rc-test-1" {
		t.Fatalf("run context not propagated, pre=%q tool=%q post=%q", seenPre, seenTool, seenPost)
	}
}

// extraModel captures the run_context metadata attached to InvokeRequest.Extra.
type extraModel struct {
	models.BaseModel
	lastExtra map[string]interface{}
}

func (m *extraModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	if req != nil {
		m.lastExtra = req.Extra
	}
	return &types.ModelResponse{Content: "ok"}, nil
}

func (m *extraModel) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	ch := make(chan types.ResponseChunk)
	close(ch)
	return ch, nil
}

func TestRun_AttachesRunContextToInvokeRequestExtra(t *testing.T) {
	m := &extraModel{BaseModel: models.BaseModel{ID: "m-extra", Provider: "mock"}}
	ag, err := New(Config{
		Model:  m,
		UserID: "user-ctx",
	})
	if err != nil {
		t.Fatalf("new agent: %v", err)
	}

	ctx := WithRunContext(context.Background(), "rc-extra-1")
	out, err := ag.Run(ctx, "hello")
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if out == nil || out.Content != "ok" {
		t.Fatalf("unexpected output: %#v", out)
	}

	if m.lastExtra == nil {
		t.Fatalf("expected InvokeRequest.Extra to be populated")
	}
	raw, ok := m.lastExtra["run_context"]
	if !ok {
		t.Fatalf("expected run_context key in InvokeRequest.Extra, got: %#v", m.lastExtra)
	}
	rcMap, ok := raw.(map[string]interface{})
	if !ok {
		t.Fatalf("run_context has wrong type: %#v", raw)
	}
	if rcMap["run_id"] != "rc-extra-1" {
		t.Fatalf("expected run_id rc-extra-1 in run_context, got %#v", rcMap["run_id"])
	}
	if rcMap["user_id"] != "user-ctx" {
		t.Fatalf("expected user_id user-ctx in run_context, got %#v", rcMap["user_id"])
	}
}
