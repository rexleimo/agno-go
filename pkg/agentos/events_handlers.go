package agentos

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// handleAgentRunStream 处理流式 Agent 运行请求（SSE）
// handleAgentRunStream handles streaming agent run requests (SSE)
func (s *Server) handleAgentRunStream(c *gin.Context) {
	agentID := c.Param("id")

	// 解析请求
	// Parse request
	var req AgentRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	ag, err := s.agentRegistry.Get(agentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "agent_not_found",
			"message": fmt.Sprintf("agent with ID '%s' not found", agentID),
		})
		return
	}

	attachments, normErr := normalizeRunRequest(&req)
	if normErr != nil {
		switch {
		case errors.Is(normErr, errInvalidMediaPayload):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_media", "message": normErr.Error()})
		case errors.Is(normErr, errMissingRunInput):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": "input or media payload is required"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": normErr.Error()})
		}
		return
	}

	ctxWithRunContext, runCtx := deriveRunContext(c.Request.Context(), req.RunContext, req.SessionID)
	if req.SessionID == "" && runCtx != nil && runCtx.SessionID != "" {
		req.SessionID = runCtx.SessionID
	}
	req.Stream = true
	s.streamAgentRun(c, agentID, ag, req, attachments, ctxWithRunContext, runCtx)
}

// sendSSE 发送单个 SSE 事件
// sendSSE sends a single SSE event
func (s *Server) sendSSE(w io.Writer, event *Event) error {
	sseData := event.ToSSE()
	_, err := fmt.Fprint(w, sseData)
	return err
}

func buildReasoningEvents(messages []*types.Message, provider, modelID string) ([]*Event, *ReasoningSummary) {
	if len(messages) == 0 {
		return nil, nil
	}

	events := make([]*Event, 0)
	var latestSummary *ReasoningSummary

	for idx, msg := range messages {
		if msg == nil || msg.ReasoningContent == nil {
			continue
		}

		rc := msg.ReasoningContent
		if strings.TrimSpace(rc.Content) == "" && rc.RedactedContent == nil {
			continue
		}

		data := ReasoningData{
			Content:         rc.Content,
			TokenCount:      rc.TokenCount,
			RedactedContent: rc.RedactedContent,
			MessageIndex:    idx,
			Model:           modelID,
			Provider:        provider,
		}

		event := NewEvent(EventReasoning, data)
		events = append(events, event)

		latestSummary = &ReasoningSummary{
			Content:         rc.Content,
			TokenCount:      rc.TokenCount,
			RedactedContent: rc.RedactedContent,
			Model:           modelID,
			Provider:        provider,
		}
	}

	return events, latestSummary
}

func buildUsageSummary(metadata map[string]interface{}) *UsageMetrics {
	if metadata == nil {
		return nil
	}

	raw, ok := metadata["usage"]
	if !ok {
		return nil
	}

	var usage types.Usage
	switch v := raw.(type) {
	case types.Usage:
		usage = v
	case *types.Usage:
		if v != nil {
			usage = *v
		}
	default:
		return nil
	}

	return &UsageMetrics{
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
	}
}

func (s *Server) emitRunEvents(w io.Writer, flusher http.Flusher, filter *EventFilter, agentID, sessionID string, runContextID string, output *agent.RunOutput, ag *agent.Agent) {
	if output == nil {
		return
	}

	var modelProvider, modelID string
	if ag != nil && ag.Model != nil {
		modelProvider = ag.Model.GetProvider()
		modelID = ag.Model.GetID()
	}

	reasoningEvents, reasoningSummary := buildReasoningEvents(output.Messages, modelProvider, modelID)
	for _, evt := range reasoningEvents {
		if evt == nil {
			continue
		}
		evt.AgentID = agentID
		evt.SessionID = sessionID
		evt.RunContextID = runContextID
		if filter.ShouldSend(evt) {
			s.sendSSE(w, evt)
			flusher.Flush()
		}
	}

	usageSummary := buildUsageSummary(output.Metadata)
	if usageSummary != nil && reasoningSummary != nil && reasoningSummary.TokenCount != nil {
		usageSummary.ReasoningTokens = *reasoningSummary.TokenCount
	}

	cacheHit := false
	if output.Metadata != nil {
		if raw, ok := output.Metadata["cache_hit"]; ok {
			if val, ok := raw.(bool); ok {
				cacheHit = val
			}
		}
	}

	duration := output.CompletedAt.Sub(output.StartedAt).Seconds()
	if duration < 0 {
		duration = 0
	}

	tokenCount := 0
	if usageSummary != nil {
		tokenCount = usageSummary.TotalTokens
	}

	complete := NewEvent(EventComplete, CompleteData{
		Output:             output.Content,
		Duration:           duration,
		TokenCount:         tokenCount,
		Reasoning:          reasoningSummary,
		Usage:              usageSummary,
		Status:             string(output.Status),
		CacheHit:           cacheHit,
		RunID:              output.RunID,
		CancellationReason: output.CancellationReason,
	})
	complete.AgentID = agentID
	complete.SessionID = sessionID
	complete.RunContextID = runContextID
	if filter.ShouldSend(complete) {
		s.sendSSE(w, complete)
		flusher.Flush()
	}
}

func tokenizeContent(content string) []string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}

	return strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\t'
	})
}
