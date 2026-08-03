package session

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

const (
	defaultSummaryTimeout = 30 * time.Second
	defaultSummaryPrompt  = "你现在扮演对话记录员，请基于以下对话总结要点，并突出可后续跟进的信息。"
)

// SummaryManager 负责对 Session 生成摘要。
type SummaryManager struct {
	model         models.Model
	logger        *slog.Logger
	operationTime time.Duration
}

// SummaryOption 自定义 SummaryManager 行为。
type SummaryOption func(*SummaryManager)

// WithSummaryModel 使用指定模型生成摘要。
func WithSummaryModel(model models.Model) SummaryOption {
	return func(m *SummaryManager) {
		m.model = model
	}
}

// WithSummaryLogger 自定义日志记录器。
func WithSummaryLogger(logger *slog.Logger) SummaryOption {
	return func(m *SummaryManager) {
		m.logger = logger
	}
}

// WithSummaryTimeout 设置摘要生成超时时间。
func WithSummaryTimeout(timeout time.Duration) SummaryOption {
	return func(m *SummaryManager) {
		if timeout > 0 {
			m.operationTime = timeout
		}
	}
}

// NewSummaryManager 构造 SummaryManager。
func NewSummaryManager(opts ...SummaryOption) *SummaryManager {
	m := &SummaryManager{
		operationTime: defaultSummaryTimeout,
		logger:        slog.Default(),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(m)
		}
	}

	return m
}

// OperationTimeout 返回当前设置的超时时间。
func (m *SummaryManager) OperationTimeout() time.Duration {
	if m == nil || m.operationTime <= 0 {
		return defaultSummaryTimeout
	}
	return m.operationTime
}

// Generate 基于会话记录生成摘要。
func (m *SummaryManager) Generate(ctx context.Context, sess *Session) (*SessionSummary, error) {
	if sess == nil {
		return nil, errors.New("session cannot be nil")
	}
	if err := ensureContext(ctx); err != nil {
		return nil, err
	}

	content, tokenCount, err := m.buildSummary(ctx, sess)
	if err != nil {
		return nil, err
	}

	summary := &SessionSummary{
		Content:     content,
		RunCount:    len(sess.Runs),
		TotalTokens: tokenCount,
		CreatedAt:   time.Now().UTC(),
	}

	return summary, nil
}

func (m *SummaryManager) buildSummary(ctx context.Context, sess *Session) (string, int, error) {
	if m == nil {
		return fallbackSummaryText(sess), 0, nil
	}

	if m.model == nil {
		return fallbackSummaryText(sess), 0, nil
	}

	messages := m.flattenRuns(sess)
	if len(messages) == 0 {
		return fallbackSummaryText(sess), 0, nil
	}

	if ctx == nil {
		ctx = context.Background()
	}

	timeout := m.OperationTimeout()
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()

	req := &models.InvokeRequest{
		Messages: append([]*types.Message{types.NewSystemMessage(defaultSummaryPrompt)}, messages...),
	}

	resp, err := m.model.Invoke(ctx, req)
	if err != nil {
		if m.logger != nil {
			m.logger.Warn("summary model invoke failed", "error", err)
		}
		return fallbackSummaryText(sess), 0, nil
	}

	summaryText := strings.TrimSpace(resp.Content)
	if summaryText == "" {
		summaryText = fallbackSummaryText(sess)
	}

	return summaryText, resp.Usage.TotalTokens, nil
}

func (m *SummaryManager) flattenRuns(sess *Session) []*types.Message {
	var messages []*types.Message

	for _, run := range sess.Runs {
		if run == nil {
			continue
		}
		if len(run.Messages) > 0 {
			messages = append(messages, run.Messages...)
			continue
		}
		if run.Content != "" {
			messages = append(messages, types.NewAssistantMessage(run.Content))
		}
	}

	return messages
}

func fallbackSummaryText(sess *Session) string {
	if sess == nil || len(sess.Runs) == 0 {
		return "当前会话尚无足够内容用于生成摘要。"
	}

	var builder strings.Builder
	maxRuns := len(sess.Runs)
	if maxRuns > 3 {
		maxRuns = 3
	}

	for i := len(sess.Runs) - maxRuns; i < len(sess.Runs); i++ {
		run := sess.Runs[i]
		if run == nil {
			continue
		}
		if run.Content != "" {
			builder.WriteString(run.Content)
			builder.WriteString("\n")
		}
	}

	text := strings.TrimSpace(builder.String())
	if text == "" {
		return "会话包含运行记录，但无法提取文本内容。"
	}

	return text
}
