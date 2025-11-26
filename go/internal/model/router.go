package model

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/rexleimo/agno-go/internal/agent"
	"github.com/rexleimo/agno-go/internal/runtime/metrics"
)

// Capability captures provider abilities.
type Capability string

const (
	CapabilityChat      Capability = "chat"
	CapabilityEmbedding Capability = "embedding"
	CapabilityStreaming Capability = "streaming"
)

// Availability describes provider status derived from configuration.
type Availability string

const (
	ProviderAvailable     Availability = "available"
	ProviderNotConfigured Availability = "not-configured"
	ProviderDisabled      Availability = "disabled"
)

// ProviderStatus reports configuration and capability state for a provider.
type ProviderStatus struct {
	Provider     agent.Provider   `json:"provider"`
	Status       Availability     `json:"status"`
	Capabilities []Capability     `json:"capabilities,omitempty"`
	MissingEnv   []string         `json:"missingEnv,omitempty"`
	Priority     int              `json:"priority,omitempty"`
	Fallbacks    []agent.Provider `json:"fallbacks,omitempty"`
	Reason       string           `json:"reason,omitempty"`
}

// ChatRequest models a provider-agnostic chat request.
type ChatRequest struct {
	Model    agent.ModelConfig `json:"model"`
	Messages []agent.Message   `json:"messages"`
	Tools    []agent.ToolCall  `json:"tools,omitempty"`
	Metadata map[string]any    `json:"metadata,omitempty"`
	Stream   bool              `json:"stream,omitempty"`
}

// ChatResponse wraps a single assistant turn and usage data.
type ChatResponse struct {
	Message      agent.Message `json:"message"`
	Usage        agent.Usage   `json:"usage,omitempty"`
	FinishReason string        `json:"finishReason,omitempty"`
}

// ChatStreamEvent represents a streaming delta (text or tool call).
type ChatStreamEvent struct {
	Type         string          `json:"type"` // token|tool_call|end
	Delta        string          `json:"delta,omitempty"`
	ToolCall     *agent.ToolCall `json:"toolCall,omitempty"`
	Usage        agent.Usage     `json:"usage,omitempty"`
	FinishReason string          `json:"finishReason,omitempty"`
	Done         bool            `json:"done,omitempty"`
}

// StreamHandler consumes streaming chat events.
type StreamHandler func(event ChatStreamEvent) error

// EmbeddingRequest models an embedding call.
type EmbeddingRequest struct {
	Model agent.ModelConfig `json:"model"`
	Input []string          `json:"input"`
}

// EmbeddingResponse contains embedding vectors.
type EmbeddingResponse struct {
	Vectors [][]float64 `json:"vectors"`
	Usage   agent.Usage `json:"usage,omitempty"`
}

// ChatProvider defines chat completion capabilities.
type ChatProvider interface {
	Name() agent.Provider
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	Stream(ctx context.Context, req ChatRequest, fn StreamHandler) error
	Status() ProviderStatus
}

// EmbeddingProvider defines embedding capabilities.
type EmbeddingProvider interface {
	Name() agent.Provider
	Embed(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error)
	Status() ProviderStatus
}

// Router dispatches chat/embedding requests to registered providers.
type Router struct {
	chatProviders      map[agent.Provider]ChatProvider
	embeddingProviders map[agent.Provider]EmbeddingProvider

	metrics metrics.Recorder
	plan    map[Capability][]ProviderInfo

	limiter chan struct{}
	timeout time.Duration
	retries int
	backoff time.Duration
}

// Errors returned by the router when routing fails.
var (
	ErrProviderNotRegistered = errors.New("provider not registered")
	ErrCapabilityUnsupported = errors.New("capability unsupported")
	ErrProviderUnavailable   = errors.New("provider not available")
	ErrMissingCredentials    = errors.New("provider missing credentials")
)

// NewRouter constructs an empty provider router.
func NewRouter(opts ...RouterOption) *Router {
	router := &Router{
		chatProviders:      make(map[agent.Provider]ChatProvider),
		embeddingProviders: make(map[agent.Provider]EmbeddingProvider),
		metrics:            metrics.NoopRecorder{},
		plan:               make(map[Capability][]ProviderInfo),
		timeout:            60 * time.Second,
		retries:            1,
		backoff:            50 * time.Millisecond,
	}
	for _, info := range providerCatalog {
		for _, cap := range info.Capabilities {
			router.plan[cap] = append(router.plan[cap], info)
		}
	}
	for cap, entries := range router.plan {
		sort.SliceStable(entries, func(i, j int) bool {
			return entries[i].Priority > entries[j].Priority
		})
		router.plan[cap] = entries
	}
	for _, opt := range opts {
		opt(router)
	}
	return router
}

// RouterOption customizes router behavior.
type RouterOption func(*Router)

// WithMaxConcurrency limits in-flight provider calls; zero disables limiting.
func WithMaxConcurrency(n int) RouterOption {
	return func(r *Router) {
		if n > 0 {
			r.limiter = make(chan struct{}, n)
		}
	}
}

// WithTimeout sets a per-request timeout applied to provider calls.
func WithTimeout(d time.Duration) RouterOption {
	return func(r *Router) {
		if d > 0 {
			r.timeout = d
		}
	}
}

// WithRetries enables retry attempts for transient failures.
func WithRetries(count int, backoff time.Duration) RouterOption {
	return func(r *Router) {
		if count >= 0 {
			r.retries = count
		}
		if backoff > 0 {
			r.backoff = backoff
		}
	}
}

// WithMetricsRecorder wires a metrics recorder into the router.
func WithMetricsRecorder(rec metrics.Recorder) RouterOption {
	return func(r *Router) {
		if rec != nil {
			r.metrics = rec
		}
	}
}

// RegisterChatProvider adds or replaces a chat provider.
func (r *Router) RegisterChatProvider(p ChatProvider) {
	r.chatProviders[p.Name()] = p
}

// RegisterEmbeddingProvider adds or replaces an embedding provider.
func (r *Router) RegisterEmbeddingProvider(p EmbeddingProvider) {
	r.embeddingProviders[p.Name()] = p
}

// Chat routes a completion request to the configured provider.
func (r *Router) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	p, ok := r.chatProviders[req.Model.Provider]
	if !ok {
		return nil, ErrProviderNotRegistered
	}
	status := p.Status()
	if status.Status != ProviderAvailable {
		return nil, providerError(status)
	}
	if req.Stream {
		return nil, errors.New("stream requested without stream handler")
	}
	var resp *ChatResponse
	start := time.Now()
	err := r.execute(ctx, func(callCtx context.Context) error {
		var err error
		resp, err = p.Chat(callCtx, req)
		return err
	})
	if err != nil {
		r.metrics.ObserveProviderError(string(req.Model.Provider), req.Model.ModelID, err.Error())
		return nil, err
	}
	r.metrics.ObserveModelLatency(string(req.Model.Provider), req.Model.ModelID, time.Since(start))
	if resp != nil {
		r.metrics.ObserveTokens(string(req.Model.Provider), resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
	}
	return resp, nil
}

// Stream routes a streaming completion request.
func (r *Router) Stream(ctx context.Context, req ChatRequest, fn StreamHandler) error {
	p, ok := r.chatProviders[req.Model.Provider]
	if !ok {
		return ErrProviderNotRegistered
	}
	status := p.Status()
	if status.Status != ProviderAvailable {
		return providerError(status)
	}
	start := time.Now()
	err := r.execute(ctx, func(callCtx context.Context) error {
		return p.Stream(callCtx, req, fn)
	})
	if err != nil {
		r.metrics.ObserveProviderError(string(req.Model.Provider), req.Model.ModelID, err.Error())
		return err
	}
	r.metrics.ObserveModelLatency(string(req.Model.Provider), req.Model.ModelID, time.Since(start))
	return nil
}

// Embed routes an embedding request.
func (r *Router) Embed(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error) {
	p, ok := r.embeddingProviders[req.Model.Provider]
	if !ok {
		return nil, ErrProviderNotRegistered
	}
	status := p.Status()
	if status.Status != ProviderAvailable {
		return nil, providerError(status)
	}
	var resp *EmbeddingResponse
	start := time.Now()
	err := r.execute(ctx, func(callCtx context.Context) error {
		var err error
		resp, err = p.Embed(callCtx, req)
		return err
	})
	if err != nil {
		r.metrics.ObserveProviderError(string(req.Model.Provider), req.Model.ModelID, err.Error())
		return nil, err
	}
	r.metrics.ObserveModelLatency(string(req.Model.Provider), req.Model.ModelID, time.Since(start))
	if resp != nil {
		r.metrics.ObserveTokens(string(req.Model.Provider), resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
	}
	return resp, nil
}

// Statuses returns provider statuses for health checks.
func (r *Router) Statuses() []ProviderStatus {
	result := make([]ProviderStatus, 0, len(r.chatProviders)+len(r.embeddingProviders))
	seen := make(map[agent.Provider]bool)

	for _, p := range r.chatProviders {
		st := p.Status()
		result = append(result, st)
		seen[p.Name()] = true
	}
	for _, p := range r.embeddingProviders {
		if seen[p.Name()] {
			continue
		}
		result = append(result, p.Status())
	}
	return result
}

// ProviderPlan returns the configured provider order for the given capability.
func (r *Router) ProviderPlan(cap Capability) []ProviderInfo {
	list := r.plan[cap]
	if len(list) == 0 {
		return nil
	}
	out := make([]ProviderInfo, len(list))
	copy(out, list)
	return out
}

func (r *Router) execute(ctx context.Context, fn func(context.Context) error) error {
	release, err := r.acquire(ctx)
	if err != nil {
		return err
	}
	if release != nil {
		defer release()
	}

	attempts := r.retries + 1
	if attempts < 1 {
		attempts = 1
	}

	for i := 0; i < attempts; i++ {
		callCtx := ctx
		var cancel context.CancelFunc
		if r.timeout > 0 {
			callCtx, cancel = context.WithTimeout(ctx, r.timeout)
		}
		err = fn(callCtx)
		if cancel != nil {
			cancel()
		}
		if err == nil {
			return nil
		}
		if !r.retryable(err) || i == attempts-1 {
			return err
		}
		if r.backoff > 0 {
			select {
			case <-time.After(r.backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return err
}

func (r *Router) acquire(ctx context.Context) (func(), error) {
	if r.limiter == nil {
		return nil, nil
	}
	select {
	case r.limiter <- struct{}{}:
		return func() { <-r.limiter }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, fmt.Errorf("%w: max in-flight reached", ErrProviderUnavailable)
	}
}

func (r *Router) retryable(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}

func providerError(status ProviderStatus) error {
	if status.Status == ProviderNotConfigured {
		return fmt.Errorf("%w: not-configured missing=%v", ErrMissingCredentials, status.MissingEnv)
	}
	reason := status.Reason
	if reason == "" {
		reason = string(status.Status)
	}
	return fmt.Errorf("%w: %s", ErrProviderUnavailable, reason)
}
