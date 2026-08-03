package eval

import (
    "context"
    "testing"
    "time"

    "github.com/rexleimo/agno-go/pkg/hno/models"
    "github.com/rexleimo/agno-go/pkg/hno/types"
)

// stubModel implements models.Model for testing
type stubModel struct{
    models.BaseModel
    reply string
    usage types.Usage
    delay time.Duration
    err   error
}

func (s *stubModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
    if s.delay > 0 { time.Sleep(s.delay) }
    if s.err != nil { return nil, s.err }
    return &types.ModelResponse{ Content: s.reply, Usage: s.usage, Model: s.ID }, nil
}
func (s *stubModel) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
    ch := make(chan types.ResponseChunk, 2)
    go func(){
        defer close(ch)
        ch <- types.ResponseChunk{Content: s.reply}
        ch <- types.ResponseChunk{Done: true}
    }()
    return ch, nil
}

func TestEvaluator_EvaluateModel(t *testing.T) {
    m := &stubModel{ BaseModel: models.BaseModel{ID:"m1", Provider:"stub"}, reply: "the quick brown fox", usage: types.Usage{PromptTokens: 5, CompletionTokens: 7, TotalTokens: 12}, delay: 10*time.Millisecond }
    ev := &Evaluator{}
    scenarios := []Scenario{{Input:"test", ExpectedContains:"brown"}, {Input:"test2", ExpectedContains:"fox"}}
    runs, sum := ev.EvaluateModel(context.Background(), m, scenarios)
    if len(runs) != 2 { t.Fatalf("runs=%d, want 2", len(runs)) }
    if sum.Runs != 2 || sum.Successes != 2 || sum.Failures != 0 { t.Fatalf("unexpected summary: %+v", sum) }
    if sum.AvgPromptTokens != 5 { t.Fatalf("avg prompt tokens = %v, want 5", sum.AvgPromptTokens) }
    if sum.AvgCompletion != 7 { t.Fatalf("avg completion tokens = %v, want 7", sum.AvgCompletion) }
}

func TestEvaluator_CompareModels(t *testing.T) {
    m1 := &stubModel{ BaseModel: models.BaseModel{ID:"m1", Provider:"stub"}, reply: "alpha", usage: types.Usage{} }
    m2 := &stubModel{ BaseModel: models.BaseModel{ID:"m2", Provider:"stub"}, reply: "beta", usage: types.Usage{} }
    ev := &Evaluator{}
    res, cmp := ev.CompareModels(context.Background(), map[string]models.Model{"a": m1, "b": m2}, []Scenario{{Input:"i", ExpectedContains:"a"}})
    if len(res) != 2 { t.Fatalf("results models=%d, want 2", len(res)) }
    if len(cmp.Models) != 2 { t.Fatalf("cmp models=%d, want 2", len(cmp.Models)) }
}

