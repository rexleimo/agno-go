package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rexleimo/agno-go/internal/agent"
	"github.com/rexleimo/agno-go/internal/model"
	"github.com/rexleimo/agno-go/internal/tool"
)

var (
	// ErrAgentNotFound indicates an agent lookup failed.
	ErrAgentNotFound = errors.New("agent not found")
	// ErrDuplicateAgentName prevents duplicate human-readable names.
	ErrDuplicateAgentName = errors.New("agent name already exists")
	// ErrInvalidSessionState blocks message handling when the session is not ready.
	ErrInvalidSessionState = errors.New("invalid session state for operation")
	// ErrSessionNotFound indicates a missing session.
	ErrSessionNotFound = agent.ErrSessionNotFound
	// ErrGuardrailViolation indicates guardrail checks blocked the request.
	ErrGuardrailViolation = errors.New("guardrail violation")
)

// Service orchestrates agents, sessions, and message flows.
type Service struct {
	store  agent.Store
	router *model.Router
	engine *Engine

	mu         sync.RWMutex
	agents     map[uuid.UUID]agent.Agent
	agentNames map[string]uuid.UUID
	sessions   map[uuid.UUID]map[uuid.UUID]*agent.Session

	guardrails GuardrailConfig
}

// NewService constructs a Service with the provided store and router.
func NewService(store agent.Store, router *model.Router) *Service {
	return &Service{
		store:      store,
		router:     router,
		engine:     &Engine{Router: router, Store: store},
		agents:     make(map[uuid.UUID]agent.Agent),
		agentNames: make(map[string]uuid.UUID),
		sessions:   make(map[uuid.UUID]map[uuid.UUID]*agent.Session),
		guardrails: GuardrailConfig{EnablePII: true, EnableInjection: true},
	}
}

// CreateAgent registers a new agent configuration.
func (s *Service) CreateAgent(ctx context.Context, cfg agent.Agent) (uuid.UUID, error) {
	if err := ctx.Err(); err != nil {
		return uuid.Nil, err
	}
	if cfg.Name == "" {
		return uuid.Nil, errors.New("agent name required")
	}
	if cfg.Model.Provider == "" {
		return uuid.Nil, errors.New("model provider required")
	}
	if cfg.Model.ModelID == "" {
		return uuid.Nil, errors.New("modelId required")
	}

	cfg.ID = uuid.New()
	now := time.Now().UTC()
	cfg.CreatedAt = now
	cfg.UpdatedAt = now

	s.mu.Lock()
	if existing, ok := s.agentNames[strings.ToLower(cfg.Name)]; ok {
		s.mu.Unlock()
		return uuid.Nil, fmt.Errorf("%w: %s (%s)", ErrDuplicateAgentName, cfg.Name, existing)
	}
	s.agents[cfg.ID] = cfg
	s.agentNames[strings.ToLower(cfg.Name)] = cfg.ID
	s.mu.Unlock()
	return cfg.ID, nil
}

// GetAgent retrieves an agent by ID.
func (s *Service) GetAgent(ctx context.Context, id uuid.UUID) (agent.Agent, error) {
	if err := ctx.Err(); err != nil {
		return agent.Agent{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	agentCfg, ok := s.agents[id]
	if !ok {
		return agent.Agent{}, fmt.Errorf("%w: %s", ErrAgentNotFound, id)
	}
	return agentCfg, nil
}

// CreateSession initializes a session for an agent.
func (s *Service) CreateSession(ctx context.Context, agentID uuid.UUID, userID string, metadata map[string]any) (agent.Session, error) {
	if err := ctx.Err(); err != nil {
		return agent.Session{}, err
	}
	s.mu.RLock()
	a, ok := s.agents[agentID]
	s.mu.RUnlock()
	if !ok {
		return agent.Session{}, fmt.Errorf("%w: %s", ErrAgentNotFound, agentID)
	}

	sessionID := uuid.New()
	now := time.Now().UTC()
	session := agent.Session{
		ID:        sessionID,
		AgentID:   agentID,
		State:     agent.SessionIdle,
		UserID:    userID,
		Metadata:  metadata,
		CreatedAt: now,
	}

	if err := s.store.UpsertSession(ctx, agentID, sessionID); err != nil {
		return agent.Session{}, err
	}

	s.mu.Lock()
	if _, ok := s.sessions[agentID]; !ok {
		s.sessions[agentID] = make(map[uuid.UUID]*agent.Session)
	}
	s.sessions[agentID][sessionID] = &session
	s.mu.Unlock()

	// Persist any default memory constraints to session metadata for convenience.
	if session.Metadata == nil {
		session.Metadata = map[string]any{}
	}
	if a.Memory.TokenWindow > 0 {
		session.Metadata["tokenWindow"] = a.Memory.TokenWindow
	}
	return session, nil
}

// MessageRequest captures a post message request.
type MessageRequest struct {
	Messages []agent.Message  `json:"messages"`
	Tools    []agent.ToolCall `json:"tools,omitempty"`
	Metadata map[string]any   `json:"metadata,omitempty"`
	Stream   bool             `json:"-"`
}

// MessageResponse contains a final assistant response.
type MessageResponse struct {
	Message agent.Message `json:"message"`
	Usage   agent.Usage   `json:"usage"`
}

// PostMessage handles non-streaming message flows.
func (s *Service) PostMessage(ctx context.Context, agentID, sessionID uuid.UUID, req MessageRequest) (*MessageResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	session, err := s.getSession(agentID, sessionID)
	if err != nil {
		return nil, err
	}
	if !canSend(session.State) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidSessionState, session.State)
	}
	history, err := s.store.LoadHistory(ctx, agentID, sessionID, agent.HistoryOptions{TokenWindow: s.tokenWindow(agentID)})
	if err != nil {
		return nil, err
	}
	if err := s.guardrailCheck(req.Messages); err != nil {
		return nil, err
	}
	session.State = agent.SessionStreaming
	session.LastActivityAt = time.Now().UTC()

	for _, msg := range req.Messages {
		msg.AgentID = agentID
		msg.SessionID = sessionID
		if msg.ID == uuid.Nil {
			msg.ID = uuid.New()
		}
		msg.CreatedAt = time.Now().UTC()
		if err := s.store.AppendMessage(ctx, agentID, sessionID, msg); err != nil {
			return nil, err
		}
	}

	if s.router == nil {
		return s.fallbackMessageResponse(ctx, session, agentID, sessionID, req)
	}

	chatMessages := append(history, req.Messages...)
	respMsg, usage, err := s.engine.Execute(ctx, agentID, sessionID, s.agentModel(agentID), chatMessages)
	if err != nil {
		session.State = agent.SessionErrored
		session.LastActivityAt = time.Now().UTC()
		return nil, err
	}

	session.State = agent.SessionCompleted
	session.LastActivityAt = time.Now().UTC()
	session.History = append(chatMessages, *respMsg)
	return &MessageResponse{
		Message: *respMsg,
		Usage:   usage,
	}, nil
}

// ToggleTool enables or disables a tool on an agent configuration.
func (s *Service) ToggleTool(ctx context.Context, agentID uuid.UUID, toolName string, enabled bool) (agent.ToolConfig, error) {
	if err := ctx.Err(); err != nil {
		return agent.ToolConfig{}, err
	}
	if strings.TrimSpace(toolName) == "" {
		return agent.ToolConfig{}, errors.New("toolName required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, ok := s.agents[agentID]
	if !ok {
		return agent.ToolConfig{}, fmt.Errorf("%w: %s", ErrAgentNotFound, agentID)
	}
	tools := cfg.Tools
	if enabled && !contains(tools.Registered, toolName) {
		tools.Registered = append(tools.Registered, toolName)
	}
	tools.Enabled = setEnabled(tools.Enabled, toolName, enabled)
	cfg.Tools = tools
	cfg.UpdatedAt = time.Now().UTC()
	s.agents[agentID] = cfg
	return tools, nil
}

// StreamMessage produces streaming events by delegating to provider router when available.
func (s *Service) StreamMessage(ctx context.Context, agentID, sessionID uuid.UUID, req MessageRequest, fn model.StreamHandler) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	session, err := s.getSession(agentID, sessionID)
	if err != nil {
		return err
	}
	if !canSend(session.State) {
		return fmt.Errorf("%w: %s", ErrInvalidSessionState, session.State)
	}
	history, err := s.store.LoadHistory(ctx, agentID, sessionID, agent.HistoryOptions{TokenWindow: s.tokenWindow(agentID)})
	if err != nil {
		return err
	}
	if err := s.guardrailCheck(req.Messages); err != nil {
		return err
	}
	session.State = agent.SessionStreaming
	session.LastActivityAt = time.Now().UTC()

	for _, msg := range req.Messages {
		msg.AgentID = agentID
		msg.SessionID = sessionID
		if msg.ID == uuid.Nil {
			msg.ID = uuid.New()
		}
		msg.CreatedAt = time.Now().UTC()
		if err := s.store.AppendMessage(ctx, agentID, sessionID, msg); err != nil {
			return err
		}
	}

	if s.router == nil {
		return s.fallbackStream(ctx, session, agentID, sessionID, req, fn)
	}

	var buffer strings.Builder
	wrapped := func(ev model.ChatStreamEvent) error {
		if ev.Type == "token" {
			buffer.WriteString(ev.Delta)
		}
		return fn(ev)
	}

	chatMessages := append(history, req.Messages...)
	if err := s.router.Stream(ctx, model.ChatRequest{
		Model:    s.agentModel(agentID),
		Messages: chatMessages,
		Tools:    req.Tools,
		Metadata: req.Metadata,
		Stream:   true,
	}, wrapped); err != nil {
		session.State = agent.SessionErrored
		session.LastActivityAt = time.Now().UTC()
		return err
	}

	finalText := strings.TrimSpace(buffer.String())
	var assistant *agent.Message
	if finalText != "" {
		msg := agent.Message{
			ID:        uuid.New(),
			AgentID:   agentID,
			SessionID: sessionID,
			Role:      agent.RoleAssistant,
			Content:   finalText,
			CreatedAt: time.Now().UTC(),
		}
		if err := s.store.AppendMessage(ctx, agentID, sessionID, msg); err != nil {
			return err
		}
		assistant = &msg
	}

	session.History = history
	session.History = append(session.History, req.Messages...)
	if assistant != nil {
		session.History = append(session.History, *assistant)
	}

	session.State = agent.SessionCompleted
	session.LastActivityAt = time.Now().UTC()
	return nil
}

func (s *Service) fallbackMessageResponse(ctx context.Context, session *agent.Session, agentID, sessionID uuid.UUID, req MessageRequest) (*MessageResponse, error) {
	history, err := s.store.LoadHistory(ctx, agentID, sessionID, agent.HistoryOptions{TokenWindow: s.tokenWindow(agentID)})
	if err != nil {
		return nil, err
	}
	assistant := agent.Message{
		ID:        uuid.New(),
		AgentID:   agentID,
		SessionID: sessionID,
		Role:      agent.RoleAssistant,
		Content:   "ok",
		CreatedAt: time.Now().UTC(),
		Usage: agent.Usage{
			PromptTokens:     estimateTokens(req.Messages),
			CompletionTokens: 1,
			LatencyMs:        0,
		},
	}

	if err := s.store.AppendMessage(ctx, agentID, sessionID, assistant); err != nil {
		return nil, err
	}
	session.State = agent.SessionCompleted
	session.LastActivityAt = time.Now().UTC()
	session.History = append(history, req.Messages...)
	session.History = append(session.History, assistant)

	return &MessageResponse{
		Message: assistant,
		Usage:   assistant.Usage,
	}, nil
}

func (s *Service) fallbackStream(ctx context.Context, session *agent.Session, agentID, sessionID uuid.UUID, req MessageRequest, fn model.StreamHandler) error {
	history, err := s.store.LoadHistory(ctx, agentID, sessionID, agent.HistoryOptions{TokenWindow: s.tokenWindow(agentID)})
	if err != nil {
		return err
	}
	if err := fn(model.ChatStreamEvent{Type: "token", Delta: "ok"}); err != nil {
		return err
	}
	final := agent.Message{
		ID:        uuid.New(),
		AgentID:   agentID,
		SessionID: sessionID,
		Role:      agent.RoleAssistant,
		Content:   "ok",
		CreatedAt: time.Now().UTC(),
	}
	if err := s.store.AppendMessage(ctx, agentID, sessionID, final); err != nil {
		return err
	}
	session.State = agent.SessionCompleted
	session.LastActivityAt = time.Now().UTC()
	session.History = append(history, req.Messages...)
	session.History = append(session.History, final)
	return fn(model.ChatStreamEvent{Type: "end", Done: true})
}

func (s *Service) getSession(agentID, sessionID uuid.UUID) (*agent.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sessions, ok := s.sessions[agentID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrAgentNotFound, agentID)
	}
	session, ok := sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, sessionID)
	}
	return session, nil
}

func (s *Service) agentModel(agentID uuid.UUID) agent.ModelConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	agentCfg, ok := s.agents[agentID]
	if !ok {
		return agent.ModelConfig{}
	}
	return agentCfg.Model
}

func estimateTokens(msgs []agent.Message) int {
	total := 0
	for _, m := range msgs {
		if m.Usage.PromptTokens > 0 || m.Usage.CompletionTokens > 0 {
			total += m.Usage.PromptTokens + m.Usage.CompletionTokens
			continue
		}
		total += len([]rune(m.Content)) / 4 // rough heuristic
	}
	if total == 0 {
		total = 1
	}
	return total
}

func canSend(state agent.SessionState) bool {
	switch state {
	case agent.SessionIdle, agent.SessionCompleted:
		return true
	default:
		return false
	}
}

func (s *Service) tokenWindow(agentID uuid.UUID) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if a, ok := s.agents[agentID]; ok && a.Memory.TokenWindow > 0 {
		return a.Memory.TokenWindow
	}
	return 0
}

// DebugSessions returns an internal snapshot of sessions for testing purposes.
func (s *Service) DebugSessions() map[uuid.UUID]map[uuid.UUID]*agent.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[uuid.UUID]map[uuid.UUID]*agent.Session, len(s.sessions))
	for agentID, m := range s.sessions {
		copyMap := make(map[uuid.UUID]*agent.Session, len(m))
		for sid, sess := range m {
			copySess := *sess
			copyMap[sid] = &copySess
		}
		out[agentID] = copyMap
	}
	return out
}

// SetSessionStateForTest mutates a session state; intended for testing flows.
func (s *Service) SetSessionStateForTest(agentID, sessionID uuid.UUID, state agent.SessionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sessMap, ok := s.sessions[agentID]; ok {
		if sess, ok := sessMap[sessionID]; ok {
			sess.State = state
		}
	}
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func setEnabled(list []string, v string, enabled bool) []string {
	filtered := make([]string, 0, len(list)+1)
	for _, item := range list {
		if item == v {
			continue
		}
		filtered = append(filtered, item)
	}
	if enabled {
		filtered = append(filtered, v)
	}
	return filtered
}

// GuardrailConfig toggles PII/Prompt injection checks at runtime.
type GuardrailConfig = tool.GuardrailConfig

func (s *Service) guardrailCheck(messages []agent.Message) error {
	if s == nil {
		return nil
	}
	for _, msg := range messages {
		if msg.Role != agent.RoleUser {
			continue
		}
		if tool.ShouldBlock(s.guardrails, msg.Content) {
			return fmt.Errorf("%w: content rejected", ErrGuardrailViolation)
		}
	}
	return nil
}
