package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rexleimo/agno-go/internal/agent"
	"github.com/rexleimo/agno-go/internal/model"
	rtmiddleware "github.com/rexleimo/agno-go/internal/runtime/middleware"
)

// Server exposes HTTP handlers for the AgentOS runtime.
type Server struct {
	Router           chi.Router
	providerStatuses func() []model.ProviderStatus
	version          string
	svc              *Service
	logger           *log.Logger
}

type serverOptions struct {
	middlewares        []func(http.Handler) http.Handler
	messageMiddlewares []func(http.Handler) http.Handler
	logger             *log.Logger
	timeout            time.Duration
	apiKey             string
	apiKeyHeader       string
	enableRecovery     bool
}

// ServerOption configures runtime server behavior.
type ServerOption func(*serverOptions)

// WithMiddlewares applies router-wide middleware (e.g., request tracing).
func WithMiddlewares(mw ...func(http.Handler) http.Handler) ServerOption {
	return func(o *serverOptions) {
		o.middlewares = append(o.middlewares, mw...)
	}
}

// WithMessageMiddleware applies middleware specifically to message endpoints.
func WithMessageMiddleware(mw ...func(http.Handler) http.Handler) ServerOption {
	return func(o *serverOptions) {
		o.messageMiddlewares = append(o.messageMiddlewares, mw...)
	}
}

// WithLogger overrides the default server logger. If nil, a stdout logger is used.
func WithLogger(l *log.Logger) ServerOption {
	return func(o *serverOptions) {
		o.logger = l
	}
}

// WithTimeout configures a request-level timeout middleware.
func WithTimeout(d time.Duration) ServerOption {
	return func(o *serverOptions) {
		if d > 0 {
			o.timeout = d
		}
	}
}

// WithAPIKeyAuth enforces API Key header authentication when a non-empty key is provided.
func WithAPIKeyAuth(key, header string) ServerOption {
	return func(o *serverOptions) {
		o.apiKey = key
		if header != "" {
			o.apiKeyHeader = header
		}
	}
}

// WithConcurrencyLimiter is a convenience option to attach a limiter to message routes.
func WithConcurrencyLimiter(l *rtmiddleware.ConcurrencyLimiter) ServerOption {
	if l == nil {
		return func(o *serverOptions) {}
	}
	return WithMessageMiddleware(l.Middleware)
}

func defaultServerOptions() serverOptions {
	return serverOptions{
		middlewares:    nil,
		logger:         log.New(os.Stdout, "runtime: ", log.LstdFlags),
		timeout:        60 * time.Second,
		apiKeyHeader:   "X-API-Key",
		enableRecovery: true,
	}
}

// NewServer builds a chi router with placeholder handlers matching the OpenAPI surface.
// providerStatuses may be nil; in that case health will omit provider details.
func NewServer(providerStatuses func() []model.ProviderStatus, version string, svc *Service, opts ...ServerOption) *Server {
	config := defaultServerOptions()
	for _, opt := range opts {
		opt(&config)
	}
	if providerStatuses == nil {
		providerStatuses = func() []model.ProviderStatus { return nil }
	}
	if config.logger == nil {
		config.logger = log.New(os.Stdout, "runtime: ", log.LstdFlags)
	}
	r := chi.NewRouter()
	if config.enableRecovery {
		r.Use(rtmiddleware.Recoverer(config.logger))
	}
	r.Use(rtmiddleware.RequestID())
	if config.timeout > 0 {
		r.Use(rtmiddleware.Timeout(config.timeout))
	}
	if config.apiKey != "" {
		r.Use(rtmiddleware.APIKeyAuth(config.apiKey, config.apiKeyHeader, config.logger))
	}
	for _, mw := range config.middlewares {
		if mw != nil {
			r.Use(mw)
		}
	}
	s := &Server{
		Router:           r,
		providerStatuses: providerStatuses,
		version:          version,
		svc:              svc,
		logger:           config.logger,
	}

	r.Get("/health", s.healthHandler)
	r.Post("/agents", s.createAgent)
	r.Get("/agents/{agentId}", s.getAgent)
	r.Post("/agents/{agentId}/sessions", s.createSession)
	r.With(config.messageMiddlewares...).Post("/agents/{agentId}/sessions/{sessionId}/messages", s.postMessage)
	r.Patch("/agents/{agentId}/tools/{toolName}", s.toggleTool)
	r.Get("/contracts/fixtures/{fixtureId}", s.notImplemented)

	return s
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	value := map[string]any{
		"status":    "ok",
		"ts":        time.Now().UTC().Format(time.RFC3339),
		"version":   s.version,
		"providers": s.providerStatuses(),
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) notImplemented(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (s *Server) createAgent(w http.ResponseWriter, r *http.Request) {
	if s.svc == nil {
		s.notImplemented(w, r)
		return
	}
	var req agent.Agent
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	id, err := s.svc.CreateAgent(r.Context(), req)
	if err != nil {
		s.logf("createAgent error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"agentId": id.String()})
}

func (s *Server) getAgent(w http.ResponseWriter, r *http.Request) {
	if s.svc == nil {
		s.notImplemented(w, r)
		return
	}
	agentID, err := parseUUIDParam(r, "agentId")
	if err != nil {
		http.Error(w, "invalid agentId", http.StatusBadRequest)
		return
	}
	a, err := s.svc.GetAgent(r.Context(), agentID)
	if err != nil {
		s.logf("getAgent error: %v", err)
		http.Error(w, err.Error(), statusForError(err))
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	if s.svc == nil {
		s.notImplemented(w, r)
		return
	}
	agentID, err := parseUUIDParam(r, "agentId")
	if err != nil {
		http.Error(w, "invalid agentId", http.StatusBadRequest)
		return
	}
	var body struct {
		UserID   string         `json:"userId"`
		Metadata map[string]any `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err.Error() != "EOF" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	session, err := s.svc.CreateSession(r.Context(), agentID, body.UserID, body.Metadata)
	if err != nil {
		s.logf("createSession error: %v", err)
		http.Error(w, err.Error(), statusForError(err))
		return
	}
	writeJSON(w, http.StatusCreated, session)
}

func (s *Server) postMessage(w http.ResponseWriter, r *http.Request) {
	if s.svc == nil {
		s.notImplemented(w, r)
		return
	}
	agentID, err := parseUUIDParam(r, "agentId")
	if err != nil {
		http.Error(w, "invalid agentId", http.StatusBadRequest)
		return
	}
	sessionID, err := parseUUIDParam(r, "sessionId")
	if err != nil {
		http.Error(w, "invalid sessionId", http.StatusBadRequest)
		return
	}
	stream := r.URL.Query().Get("stream") == "true"

	var req MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	req.Stream = stream

	if stream {
		s.handleStream(r.Context(), w, agentID, sessionID, req)
		return
	}
	resp, err := s.svc.PostMessage(r.Context(), agentID, sessionID, req)
	if err != nil {
		s.logf("postMessage error: %v", err)
		http.Error(w, err.Error(), statusForError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"messageId": resp.Message.ID.String(),
		"content":   resp.Message.Content,
		"toolCalls": resp.Message.ToolCalls,
		"usage":     resp.Usage,
		"state":     agent.SessionCompleted,
	})
}

func (s *Server) toggleTool(w http.ResponseWriter, r *http.Request) {
	if s.svc == nil {
		s.notImplemented(w, r)
		return
	}
	agentID, err := parseUUIDParam(r, "agentId")
	if err != nil {
		http.Error(w, "invalid agentId", http.StatusBadRequest)
		return
	}
	toolName := chi.URLParam(r, "toolName")
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	tools, err := s.svc.ToggleTool(r.Context(), agentID, toolName, body.Enabled)
	if err != nil {
		http.Error(w, err.Error(), statusForError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"toolName": toolName,
		"enabled":  body.Enabled,
		"tools":    tools,
	})
}

func (s *Server) handleStream(ctx context.Context, w http.ResponseWriter, agentID, sessionID uuid.UUID, req MessageRequest) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusMultiStatus)

	encoder := func(ev model.ChatStreamEvent) error {
		payload, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte("event: message\n")); err != nil {
			return err
		}
		if _, err := w.Write([]byte("data: ")); err != nil {
			return err
		}
		if _, err := w.Write(payload); err != nil {
			return err
		}
		if _, err := w.Write([]byte("\n\n")); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	if err := s.svc.StreamMessage(ctx, agentID, sessionID, req, encoder); err != nil {
		s.logf("streamMessage error: %v", err)
		http.Error(w, err.Error(), statusForError(err))
		return
	}
}

func parseUUIDParam(r *http.Request, key string) (uuid.UUID, error) {
	raw := chi.URLParam(r, key)
	return uuid.Parse(raw)
}

func statusForError(err error) int {
	if errors.Is(err, context.DeadlineExceeded) {
		return http.StatusRequestTimeout
	}
	if errors.Is(err, context.Canceled) {
		return http.StatusRequestTimeout
	}
	if errors.Is(err, model.ErrMissingCredentials) {
		return http.StatusServiceUnavailable
	}
	if errors.Is(err, model.ErrProviderUnavailable) {
		return http.StatusServiceUnavailable
	}
	if errors.Is(err, model.ErrProviderNotRegistered) {
		return http.StatusBadRequest
	}
	if errors.Is(err, model.ErrCapabilityUnsupported) {
		return http.StatusBadRequest
	}
	if errors.Is(err, ErrAgentNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrInvalidSessionState) {
		return http.StatusConflict
	}
	if errors.Is(err, ErrSessionNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrDuplicateAgentName) {
		return http.StatusConflict
	}
	return http.StatusBadRequest
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) logf(format string, args ...any) {
	if s.logger == nil {
		return
	}
	s.logger.Printf(format, args...)
}
