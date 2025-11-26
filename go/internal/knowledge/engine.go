package knowledge

import (
	"context"
	"fmt"
	"strings"

	"github.com/rexleimo/agno-go/internal/agent"
	"github.com/rexleimo/agno-go/internal/model"
)

// Retriever describes the interface for fetching relevant knowledge chunks.
type Retriever interface {
	Retrieve(ctx context.Context, query string, filters map[string]string) ([]Chunk, error)
}

// Chunk represents a knowledge item returned from a Retriever.
type Chunk struct {
	ID       string            `json:"id"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Score    float64           `json:"score,omitempty"`
}

// Engine orchestrates retrieval + embedding (if needed) and applies fallback when unavailable.
type Engine struct {
	Retriever Retriever
	Fallback  Strategy
}

// Fetch retrieves knowledge for the given query; on retriever error, returns fallback string or error per Strategy.
func (e *Engine) Fetch(ctx context.Context, query string, filters map[string]string) ([]Chunk, string, error) {
	if e == nil || e.Retriever == nil {
		msg, err := HandleUnavailable(e.strategyOrDefault(), "retriever not configured")
		return nil, msg, err
	}
	chunks, err := e.Retriever.Retrieve(ctx, query, filters)
	if err != nil {
		msg, fbErr := HandleUnavailable(e.strategyOrDefault(), err.Error())
		return nil, msg, fbErr
	}
	return chunks, "", nil
}

// BuildRAGRequest constructs a model.ChatRequest combining retrieved context with user query.
func BuildRAGRequest(modelCfg agent.ModelConfig, messages []agent.Message, chunks []Chunk) model.ChatRequest {
	system := agent.Message{Role: agent.RoleSystem, Content: renderContext(chunks)}
	return model.ChatRequest{
		Model:    modelCfg,
		Messages: append([]agent.Message{system}, messages...),
	}
}

func (e *Engine) strategyOrDefault() Strategy {
	if e == nil {
		return Strategy{Mode: ModeHint}
	}
	if e.Fallback.Mode == "" {
		return Strategy{Mode: ModeHint}
	}
	return e.Fallback
}

func renderContext(chunks []Chunk) string {
	if len(chunks) == 0 {
		return "No retrieved context."
	}
	builder := strings.Builder{}
	builder.WriteString("Relevant context:\n")
	for i, c := range chunks {
		builder.WriteString(fmt.Sprintf("[%d] %s\n", i+1, c.Content))
	}
	return builder.String()
}
