package session

import (
	"context"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

type mockSummaryModel struct {
	models.BaseModel
	invokeCount int
}

func (m *mockSummaryModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	m.invokeCount++
	return &types.ModelResponse{Content: "model summary", Usage: types.Usage{TotalTokens: 7}}, nil
}

func (m *mockSummaryModel) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	ch := make(chan types.ResponseChunk)
	close(ch)
	return ch, nil
}

func TestSummaryManagerGenerate(t *testing.T) {
	model := &mockSummaryModel{BaseModel: models.BaseModel{ID: "summary", Provider: "mock"}}
	manager := NewSummaryManager(WithSummaryModel(model), WithSummaryTimeout(time.Second))

	sess := NewSession("summary", "agent-1")
	sess.AddRun(&agent.RunOutput{
		Content: "assistant reply",
		Messages: []*types.Message{
			types.NewUserMessage("hello"),
			types.NewAssistantMessage("assistant reply"),
		},
	})

	summary, err := manager.Generate(context.Background(), sess)
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	if summary == nil || summary.Content != "model summary" {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if summary.TotalTokens != 7 {
		t.Fatalf("expected token count 7, got %d", summary.TotalTokens)
	}
	if model.invokeCount != 1 {
		t.Fatalf("expected model invoke, got %d", model.invokeCount)
	}
}

func TestSummaryManagerFallback(t *testing.T) {
	manager := NewSummaryManager()
	sess := NewSession("summary", "agent-1")
	sess.AddRun(&agent.RunOutput{Content: "final text"})

	summary, err := manager.Generate(context.Background(), sess)
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	if summary == nil || summary.Content == "" {
		t.Fatalf("expected fallback summary")
	}
	if summary.Content == "model summary" {
		t.Fatalf("fallback should not use model output")
	}
}

func TestSummaryManagerContextCancelled(t *testing.T) {
	model := &mockSummaryModel{BaseModel: models.BaseModel{ID: "summary", Provider: "mock"}}
	manager := NewSummaryManager(WithSummaryModel(model))
	sess := NewSession("summary", "agent-1")
	sess.AddRun(&agent.RunOutput{Content: "text"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := manager.Generate(ctx, sess); err == nil {
		t.Fatalf("expected context cancellation error")
	}
}
