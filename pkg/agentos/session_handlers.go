package agentos

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/session"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// CreateSessionRequest represents the request to create a new session
type CreateSessionRequest struct {
	AgentID    string                 `json:"agent_id" binding:"required"`
	UserID     string                 `json:"user_id,omitempty"`
	TeamID     string                 `json:"team_id,omitempty"`
	WorkflowID string                 `json:"workflow_id,omitempty"`
	Name       string                 `json:"name,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateSessionRequest represents the request to update a session
type UpdateSessionRequest struct {
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	State    map[string]interface{} `json:"state,omitempty"`
}

// SessionResponse represents the session API response
type SessionResponse struct {
	SessionID  string                  `json:"session_id"`
	AgentID    string                  `json:"agent_id,omitempty"`
	UserID     string                  `json:"user_id,omitempty"`
	TeamID     string                  `json:"team_id,omitempty"`
	WorkflowID string                  `json:"workflow_id,omitempty"`
	Name       string                  `json:"name,omitempty"`
	Metadata   map[string]interface{}  `json:"metadata,omitempty"`
	State      map[string]interface{}  `json:"state,omitempty"`
	RunCount   int                     `json:"run_count"`
	CreatedAt  int64                   `json:"created_at"`
	UpdatedAt  int64                   `json:"updated_at"`
	Summary    *session.SessionSummary `json:"summary,omitempty"`
	Runs       []SessionRunMetadata    `json:"runs,omitempty"`
}

// SessionRunMetadata captures key details of a run persisted in a session.
type SessionRunMetadata struct {
	RunID              string          `json:"run_id"`
	Status             agent.RunStatus `json:"status"`
	StartedAt          int64           `json:"started_at,omitempty"`
	CompletedAt        int64           `json:"completed_at,omitempty"`
	CancellationReason string          `json:"cancellation_reason,omitempty"`
	CacheHit           bool            `json:"cache_hit,omitempty"`
}

// SessionSummaryResult represents the payload returned by summary endpoints.
type SessionSummaryResult struct {
	Summary *session.SessionSummary `json:"summary"`
}

// ReuseSessionRequest allows attaching an existing session to other entities.
type ReuseSessionRequest struct {
	AgentID    string `json:"agent_id,omitempty"`
	TeamID     string `json:"team_id,omitempty"`
	WorkflowID string `json:"workflow_id,omitempty"`
	UserID     string `json:"user_id,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Status  string `json:"status,omitempty"` // "error"
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// handleCreateSession creates a new session
// POST /api/v1/sessions
func (s *Server) handleCreateSession(c *gin.Context) {
	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Status:  "error",
			Error:   "invalid request",
			Message: err.Error(),
			Code:    "INVALID_REQUEST",
		})
		return
	}

	// Generate session ID
	sessionID := uuid.New().String()

	// Create session
	sess := session.NewSession(sessionID, req.AgentID)
	sess.UserID = req.UserID
	sess.TeamID = req.TeamID
	sess.WorkflowID = req.WorkflowID
	sess.Name = req.Name

	if req.Metadata != nil {
		sess.Metadata = req.Metadata
	}

	// Store session
	if err := s.sessionStorage.Create(c.Request.Context(), sess); err != nil {
		s.logger.Error("failed to create session", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Status:  "error",
			Error:   "failed to create session",
			Message: err.Error(),
			Code:    "STORAGE_ERROR",
		})
		return
	}

	s.logger.Info("session created", "session_id", sessionID, "agent_id", req.AgentID)

	c.JSON(http.StatusCreated, sessionToResponse(sess))
}

// handleGetSession retrieves a session by ID
// GET /api/v1/sessions/:id
func (s *Server) handleGetSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Status: "error",
			Error:  "session ID is required",
			Code:   "INVALID_REQUEST",
		})
		return
	}

	sess, err := s.sessionStorage.Get(c.Request.Context(), sessionID)
	if err == session.ErrSessionNotFound {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Status: "error",
			Error:  "session not found",
			Code:   "SESSION_NOT_FOUND",
		})
		return
	}
	if err != nil {
		s.logger.Error("failed to get session", "error", err, "session_id", sessionID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Status:  "error",
			Error:   "failed to get session",
			Message: err.Error(),
			Code:    "STORAGE_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, sessionToResponse(sess))
}

// handleUpdateSession updates an existing session
// PUT /api/v1/sessions/:id
func (s *Server) handleUpdateSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Status: "error",
			Error:  "session ID is required",
			Code:   "INVALID_REQUEST",
		})
		return
	}

	var req UpdateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Status:  "error",
			Error:   "invalid request",
			Message: err.Error(),
			Code:    "INVALID_REQUEST",
		})
		return
	}

	// Get existing session
	sess, err := s.sessionStorage.Get(c.Request.Context(), sessionID)
	if err == session.ErrSessionNotFound {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Status: "error",
			Error:  "session not found",
			Code:   "SESSION_NOT_FOUND",
		})
		return
	}
	if err != nil {
		s.logger.Error("failed to get session", "error", err, "session_id", sessionID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Status:  "error",
			Error:   "failed to get session",
			Message: err.Error(),
			Code:    "STORAGE_ERROR",
		})
		return
	}

	// Update fields
	if req.Name != "" {
		sess.Name = req.Name
	}
	if req.Metadata != nil {
		sess.Metadata = req.Metadata
	}
	if req.State != nil {
		sess.State = req.State
	}

	// Save updated session
	if err := s.sessionStorage.Update(c.Request.Context(), sess); err != nil {
		s.logger.Error("failed to update session", "error", err, "session_id", sessionID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Status:  "error",
			Error:   "failed to update session",
			Message: err.Error(),
			Code:    "STORAGE_ERROR",
		})
		return
	}

	s.logger.Info("session updated", "session_id", sessionID)

	c.JSON(http.StatusOK, sessionToResponse(sess))
}

// handleDeleteSession deletes a session
// DELETE /api/v1/sessions/:id
func (s *Server) handleDeleteSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Status: "error",
			Error:  "session ID is required",
			Code:   "INVALID_REQUEST",
		})
		return
	}

	err := s.sessionStorage.Delete(c.Request.Context(), sessionID)
	if err == session.ErrSessionNotFound {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Status: "error",
			Error:  "session not found",
			Code:   "SESSION_NOT_FOUND",
		})
		return
	}
	if err != nil {
		s.logger.Error("failed to delete session", "error", err, "session_id", sessionID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Status:  "error",
			Error:   "failed to delete session",
			Message: err.Error(),
			Code:    "STORAGE_ERROR",
		})
		return
	}

	s.logger.Info("session deleted", "session_id", sessionID)

	c.JSON(http.StatusOK, gin.H{
		"message": "session deleted successfully",
	})
}

// handleListSessions lists sessions with optional filters
// GET /api/v1/sessions?agent_id=xxx&user_id=yyy
func (s *Server) handleListSessions(c *gin.Context) {
	// Build filters from query parameters
	filters := make(map[string]interface{})

	if agentID := c.Query("agent_id"); agentID != "" {
		filters["agent_id"] = agentID
	}
	if userID := c.Query("user_id"); userID != "" {
		filters["user_id"] = userID
	}
	if teamID := c.Query("team_id"); teamID != "" {
		filters["team_id"] = teamID
	}
	if workflowID := c.Query("workflow_id"); workflowID != "" {
		filters["workflow_id"] = workflowID
	}

	sessions, err := s.sessionStorage.List(c.Request.Context(), filters)
	if err != nil {
		s.logger.Error("failed to list sessions", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "failed to list sessions",
			Message: err.Error(),
			Code:    "STORAGE_ERROR",
		})
		return
	}

	// Convert to response format
	responses := make([]SessionResponse, len(sessions))
	for i, sess := range sessions {
		responses[i] = sessionToResponse(sess)
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": responses,
		"count":    len(responses),
	})
}

// handleGetSessionSummary returns the stored summary for a session.
func (s *Server) handleGetSessionSummary(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Status: "error",
			Error:  "session ID is required",
			Code:   "INVALID_REQUEST",
		})
		return
	}

	sess, err := s.sessionStorage.Get(c.Request.Context(), sessionID)
	if err == session.ErrSessionNotFound {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "session not found",
			Code:  "SESSION_NOT_FOUND",
		})
		return
	}
	if err != nil {
		s.logger.Error("failed to get session", "error", err, "session_id", sessionID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "failed to get session",
			Message: err.Error(),
			Code:    "STORAGE_ERROR",
		})
		return
	}

	if sess.Summary == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "summary not available",
			Code:  "SUMMARY_NOT_FOUND",
		})
		return
	}

	c.JSON(http.StatusOK, SessionSummaryResult{Summary: sess.Summary})
}

// handlePostSessionSummary triggers summary generation for a session.
func (s *Server) handlePostSessionSummary(c *gin.Context) {
	if s.summaryManager == nil {
		c.JSON(http.StatusNotImplemented, ErrorResponse{
			Error: "session summary not configured",
			Code:  "SUMMARY_DISABLED",
		})
		return
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "session ID is required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	async := parseBoolQuery(c, "async")

	sess, err := s.sessionStorage.Get(c.Request.Context(), sessionID)
	if err == session.ErrSessionNotFound {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "session not found",
			Code:  "SESSION_NOT_FOUND",
		})
		return
	}
	if err != nil {
		s.logger.Error("failed to get session", "error", err, "session_id", sessionID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "failed to get session",
			Message: err.Error(),
			Code:    "STORAGE_ERROR",
		})
		return
	}

	if async {
		s.scheduleSessionSummary(sessionID)
		c.JSON(http.StatusAccepted, gin.H{
			"status":     "scheduled",
			"session_id": sessionID,
		})
		return
	}

	summary, err := s.generateSessionSummary(c.Request.Context(), sess)
	if err != nil {
		s.logger.Error("failed to generate summary", "error", err, "session_id", sessionID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "failed to generate session summary",
			Message: err.Error(),
			Code:    "SUMMARY_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, SessionSummaryResult{Summary: summary})
}

// handleReuseSession attaches an existing session to provided entities.
func (s *Server) handleReuseSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "session ID is required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	var req ReuseSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid request",
			Message: err.Error(),
			Code:    "INVALID_REQUEST",
		})
		return
	}

	if req.AgentID == "" && req.TeamID == "" && req.WorkflowID == "" && req.UserID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "at least one target identifier is required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	sess, err := s.sessionStorage.Get(c.Request.Context(), sessionID)
	if err == session.ErrSessionNotFound {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "session not found",
			Code:  "SESSION_NOT_FOUND",
		})
		return
	}
	if err != nil {
		s.logger.Error("failed to get session", "error", err, "session_id", sessionID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "failed to get session",
			Message: err.Error(),
			Code:    "STORAGE_ERROR",
		})
		return
	}

	if req.AgentID != "" {
		sess.AgentID = req.AgentID
	}
	if req.TeamID != "" {
		sess.TeamID = req.TeamID
	}
	if req.WorkflowID != "" {
		sess.WorkflowID = req.WorkflowID
	}
	if req.UserID != "" {
		sess.UserID = req.UserID
	}
	sess.UpdatedAt = time.Now().UTC()

	if err := s.sessionStorage.Update(c.Request.Context(), sess); err != nil {
		s.logger.Error("failed to update session", "error", err, "session_id", sessionID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "failed to update session",
			Message: err.Error(),
			Code:    "STORAGE_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, sessionToResponse(sess))
}

// handleSessionHistory returns recent messages for a session.
func (s *Server) handleSessionHistory(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "session ID is required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	sess, err := s.sessionStorage.Get(c.Request.Context(), sessionID)
	if err == session.ErrSessionNotFound {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Status: "error",
			Error:  "session not found",
			Code:   "SESSION_NOT_FOUND",
		})
		return
	}
	if err != nil {
		s.logger.Error("failed to get session history", "error", err, "session_id", sessionID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Status:  "error",
			Error:   "failed to get session",
			Message: err.Error(),
			Code:    "STORAGE_ERROR",
		})
		return
	}

	messages := cloneMessages(sess)
	if limit := parseIntQuery(c, "num_messages"); limit > 0 && len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}

	response := gin.H{
		"session_id":    sessionID,
		"messages":      messages,
		"stream_events": parseBoolQuery(c, "stream_events"),
		"runs":          buildSessionRuns(sess.Runs),
	}
	if sess.Summary != nil {
		response["summary"] = sess.Summary
	}

	c.JSON(http.StatusOK, response)
}

// sessionToResponse converts a session to API response format
func sessionToResponse(sess *session.Session) SessionResponse {
	return SessionResponse{
		SessionID:  sess.SessionID,
		AgentID:    sess.AgentID,
		UserID:     sess.UserID,
		TeamID:     sess.TeamID,
		WorkflowID: sess.WorkflowID,
		Name:       sess.Name,
		Metadata:   sess.Metadata,
		State:      sess.State,
		RunCount:   sess.GetRunCount(),
		CreatedAt:  sess.CreatedAt.Unix(),
		UpdatedAt:  sess.UpdatedAt.Unix(),
		Summary:    sess.Summary,
		Runs:       buildSessionRuns(sess.Runs),
	}
}

func buildSessionRuns(runs []*agent.RunOutput) []SessionRunMetadata {
	if len(runs) == 0 {
		return []SessionRunMetadata{}
	}

	metadata := make([]SessionRunMetadata, 0, len(runs))
	for _, run := range runs {
		if run == nil {
			continue
		}

		entry := SessionRunMetadata{
			RunID:              run.RunID,
			Status:             run.Status,
			CancellationReason: run.CancellationReason,
		}
		if !run.StartedAt.IsZero() {
			entry.StartedAt = run.StartedAt.Unix()
		}
		if !run.CompletedAt.IsZero() {
			entry.CompletedAt = run.CompletedAt.Unix()
		}
		if run.Metadata != nil {
			if hit, ok := run.Metadata["cache_hit"].(bool); ok {
				entry.CacheHit = hit
			}
		}

		metadata = append(metadata, entry)
	}

	return metadata
}

func (s *Server) generateSessionSummary(ctx context.Context, sess *session.Session) (*session.SessionSummary, error) {
	if s.summaryManager == nil {
		return nil, errors.New("summary manager is not configured")
	}

	summary, err := s.summaryManager.Generate(ctx, sess)
	if err != nil {
		return nil, err
	}

	sess.Summary = summary
	sess.UpdatedAt = time.Now().UTC()

	if err := s.sessionStorage.Update(ctx, sess); err != nil {
		return nil, err
	}

	return summary, nil
}

func (s *Server) scheduleSessionSummary(sessionID string) {
	if s.summaryManager == nil {
		return
	}

	go func(id string) {
		ctx, cancel := context.WithTimeout(context.Background(), s.summaryManager.OperationTimeout())
		defer cancel()

		sess, err := s.sessionStorage.Get(ctx, id)
		if err != nil {
			if err != session.ErrSessionNotFound {
				s.logger.Warn("async summary skipped", "error", err, "session_id", id)
			}
			return
		}

		if _, err := s.generateSessionSummary(ctx, sess); err != nil {
			s.logger.Warn("async summary generation failed", "error", err, "session_id", id)
		}
	}(sessionID)
}

func cloneMessages(sess *session.Session) []*types.Message {
	if sess == nil {
		return nil
	}

	var messages []*types.Message
	for _, run := range sess.Runs {
		if run == nil {
			continue
		}
		if len(run.Messages) == 0 {
			if run.Content != "" {
				messages = append(messages, types.NewAssistantMessage(run.Content))
			}
			continue
		}
		for _, msg := range run.Messages {
			if msg == nil {
				continue
			}
			messages = append(messages, cloneMessage(msg))
		}
	}
	return messages
}

func cloneMessage(msg *types.Message) *types.Message {
	if msg == nil {
		return nil
	}
	clone := *msg
	if len(msg.ToolCalls) > 0 {
		clone.ToolCalls = append([]types.ToolCall(nil), msg.ToolCalls...)
	}
	if msg.ReasoningContent != nil {
		rc := *msg.ReasoningContent
		clone.ReasoningContent = &rc
	}
	return &clone
}

func parseBoolQuery(c *gin.Context, key string) bool {
	value := c.Query(key)
	if value == "" {
		return false
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}
	return parsed
}

func parseIntQuery(c *gin.Context, key string) int {
	value := c.Query(key)
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}
