package eval

import (
    "context"
    "encoding/json"
    "strings"
    "time"

    "github.com/rexleimo/agno-go/pkg/hno/models"
    "github.com/rexleimo/agno-go/pkg/hno/types"
)

// Scenario represents a simple evaluation case
type Scenario struct {
    Input            string
    // ExpectedContains checks if model output includes this substring (minimal oracle)
    ExpectedContains string
}

// RunMetrics captures metrics for a single evaluation run
type RunMetrics struct {
    Model            string        `json:"model"`
    Duration         time.Duration `json:"duration"`
    PromptTokens     int           `json:"prompt_tokens"`
    CompletionTokens int           `json:"completion_tokens"`
    TotalTokens      int           `json:"total_tokens"`
    ToolCalls        int           `json:"tool_calls"`
    Success          bool          `json:"success"`
    Error            string        `json:"error,omitempty"`
    Timestamp        time.Time     `json:"timestamp"`
}

// Summary aggregates metrics across runs
type Summary struct {
    Runs            int              `json:"runs"`
    Successes       int              `json:"successes"`
    Failures        int              `json:"failures"`
    SuccessRate     float64          `json:"success_rate"`
    AvgLatencyMS    float64          `json:"avg_latency_ms"`
    AvgPromptTokens float64          `json:"avg_prompt_tokens"`
    AvgCompletion   float64          `json:"avg_completion_tokens"`
    AvgTotalTokens  float64          `json:"avg_total_tokens"`
    TotalToolCalls  int              `json:"total_tool_calls"`
    ByModel         map[string]Summary `json:"by_model,omitempty"`
}

// Comparison holds model-to-summary mapping
type Comparison struct {
    Models map[string]Summary `json:"models"`
}

// Evaluator runs scenarios against models and aggregates metrics
type Evaluator struct{}

// EvaluateModel runs scenarios on a single model
func (e *Evaluator) EvaluateModel(ctx context.Context, m models.Model, scenarios []Scenario) ([]RunMetrics, Summary) {
    runs := make([]RunMetrics, 0, len(scenarios))
    var sum Summary
    sum.ByModel = nil

    for _, sc := range scenarios {
        start := time.Now()
        resp, err := m.Invoke(ctx, &models.InvokeRequest{Messages: []*types.Message{types.NewUserMessage(sc.Input)}})
        dur := time.Since(start)

        rm := RunMetrics{
            Model:     m.GetID(),
            Duration:  dur,
            Timestamp: time.Now(),
        }
        if err != nil {
            rm.Error = err.Error()
            rm.Success = false
        } else {
            rm.Success = (sc.ExpectedContains == "" || strings.Contains(strings.ToLower(resp.Content), strings.ToLower(sc.ExpectedContains)))
            rm.PromptTokens = resp.Usage.PromptTokens
            rm.CompletionTokens = resp.Usage.CompletionTokens
            rm.TotalTokens = resp.Usage.TotalTokens
            rm.ToolCalls = len(resp.ToolCalls)
        }

        runs = append(runs, rm)
        // accumulate
        sum.Runs++
        if rm.Success {
            sum.Successes++
        } else {
            sum.Failures++
        }
        sum.AvgLatencyMS += float64(dur.Milliseconds())
        sum.AvgPromptTokens += float64(rm.PromptTokens)
        sum.AvgCompletion += float64(rm.CompletionTokens)
        sum.AvgTotalTokens += float64(rm.TotalTokens)
        sum.TotalToolCalls += rm.ToolCalls
    }

    if sum.Runs > 0 {
        sum.SuccessRate = float64(sum.Successes) / float64(sum.Runs)
        sum.AvgLatencyMS /= float64(sum.Runs)
        sum.AvgPromptTokens /= float64(sum.Runs)
        sum.AvgCompletion /= float64(sum.Runs)
        sum.AvgTotalTokens /= float64(sum.Runs)
    }

    return runs, sum
}

// CompareModels runs the same scenarios on multiple models and returns a comparison
func (e *Evaluator) CompareModels(ctx context.Context, modelsMap map[string]models.Model, scenarios []Scenario) (map[string][]RunMetrics, Comparison) {
    results := make(map[string][]RunMetrics, len(modelsMap))
    cmp := Comparison{Models: make(map[string]Summary, len(modelsMap))}

    for name, m := range modelsMap {
        runs, sum := e.EvaluateModel(ctx, m, scenarios)
        results[name] = runs
        cmp.Models[name] = sum
    }
    return results, cmp
}

// JSON encoders for reports
func (s Summary) JSON() []byte {
    b, _ := json.Marshal(s)
    return b
}

func (c Comparison) JSON() []byte {
    b, _ := json.Marshal(c)
    return b
}

