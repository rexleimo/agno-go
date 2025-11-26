package stub

import (
	"context"
	"strings"

	"github.com/rexleimo/agno-go/internal/agent"
	"github.com/rexleimo/agno-go/internal/model"
)

// Provider is a lightweight provider implementation used until real clients are wired.
type Provider struct {
	name         agent.Provider
	status       model.Availability
	missingEnv   []string
	capabilities []model.Capability
}

// New constructs a stub provider with the given status and missing env reasons.
func New(name agent.Provider, status model.Availability, missing []string) *Provider {
	return &Provider{
		name:         name,
		status:       status,
		missingEnv:   missing,
		capabilities: []model.Capability{model.CapabilityChat, model.CapabilityEmbedding, model.CapabilityStreaming},
	}
}

func (p *Provider) Name() agent.Provider { return p.name }

func (p *Provider) Status() model.ProviderStatus {
	meta := model.ProviderMetadata(p.name)
	return model.ProviderStatus{
		Provider:     p.name,
		Status:       p.status,
		Capabilities: p.capabilities,
		MissingEnv:   p.missingEnv,
		Priority:     meta.Priority,
		Fallbacks:    append([]agent.Provider(nil), meta.Fallbacks...),
	}
}

func (p *Provider) Chat(ctx context.Context, req model.ChatRequest) (*model.ChatResponse, error) {
	if p.status != model.ProviderAvailable {
		return nil, model.ErrProviderUnavailable
	}
	content := replyText(req.Messages)
	msg := agent.Message{
		Role:    agent.RoleAssistant,
		Content: content,
	}
	return &model.ChatResponse{
		Message: msg,
		Usage: agent.Usage{
			PromptTokens:     estimateTokens(req.Messages),
			CompletionTokens: estimateTokens([]agent.Message{msg}),
			LatencyMs:        1,
		},
	}, nil
}

func (p *Provider) Stream(ctx context.Context, req model.ChatRequest, fn model.StreamHandler) error {
	if p.status != model.ProviderAvailable {
		return model.ErrProviderUnavailable
	}
	text := replyText(req.Messages)
	parts := strings.Fields(text)
	if len(parts) == 0 {
		parts = []string{text}
	}
	for _, token := range parts {
		if err := fn(model.ChatStreamEvent{Type: "token", Delta: token + " "}); err != nil {
			return err
		}
	}
	return fn(model.ChatStreamEvent{Type: "end", Done: true})
}

func (p *Provider) Embed(ctx context.Context, req model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	if p.status != model.ProviderAvailable {
		return nil, model.ErrProviderUnavailable
	}
	vectors := make([][]float64, len(req.Input))
	for i := range req.Input {
		vectors[i] = []float64{0.1, 0.2, 0.3}
	}
	return &model.EmbeddingResponse{Vectors: vectors}, nil
}

func replyText(msgs []agent.Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == agent.RoleUser {
			return "echo: " + msgs[i].Content
		}
	}
	return "echo: ok"
}

func estimateTokens(msgs []agent.Message) int {
	total := 0
	for _, m := range msgs {
		if m.Usage.PromptTokens > 0 || m.Usage.CompletionTokens > 0 {
			total += m.Usage.PromptTokens + m.Usage.CompletionTokens
			continue
		}
		total += len([]rune(m.Content)) / 4
	}
	if total == 0 {
		total = 1
	}
	return total
}
