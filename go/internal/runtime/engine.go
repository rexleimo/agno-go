package runtime

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/rexleimo/agno-go/internal/agent"
	"github.com/rexleimo/agno-go/internal/model"
)

// Engine is a thin executor for a single turn; it wires router + store and can be reused by higher-level flows.
type Engine struct {
	Router *model.Router
	Store  agent.Store
}

// Execute runs a non-streaming turn, persists history, and returns the assistant message.
func (e *Engine) Execute(ctx context.Context, agentID uuid.UUID, sessionID uuid.UUID, modelCfg agent.ModelConfig, messages []agent.Message) (*agent.Message, agent.Usage, error) {
	if e == nil || e.Router == nil {
		return nil, agent.Usage{}, errors.New("engine missing router")
	}
	req := model.ChatRequest{
		Model:    modelCfg,
		Messages: messages,
	}
	resp, err := e.Router.Chat(ctx, req)
	if err != nil {
		return nil, agent.Usage{}, err
	}
	assistant := agent.Message{
		ID:        uuid.New(),
		AgentID:   agentID,
		SessionID: sessionID,
		Role:      agent.RoleAssistant,
		Content:   resp.Message.Content,
		Usage:     resp.Usage,
		CreatedAt: time.Now().UTC(),
	}
	if e.Store != nil {
		for _, m := range messages {
			_ = e.Store.AppendMessage(ctx, agentID, sessionID, m)
		}
		_ = e.Store.AppendMessage(ctx, agentID, sessionID, assistant)
	}
	return &assistant, resp.Usage, nil
}
