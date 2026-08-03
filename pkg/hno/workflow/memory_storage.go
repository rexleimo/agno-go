package workflow

import (
	"context"
	"sync"
	"time"
)

// MemoryStorage implements WorkflowStorage using in-memory storage
// MemoryStorage 使用内存存储实现 WorkflowStorage
type MemoryStorage struct {
	mu       sync.RWMutex
	sessions map[string]*WorkflowSession // sessionID -> session
	maxSize  int                         // Maximum number of sessions to store (0 = unlimited)
}

// NewMemoryStorage creates a new memory-based workflow storage
// NewMemoryStorage 创建新的基于内存的工作流存储
// maxSize specifies the maximum number of sessions to store (0 = unlimited)
// maxSize 指定要存储的最大会话数（0 = 无限制）
func NewMemoryStorage(maxSize int) *MemoryStorage {
	return &MemoryStorage{
		sessions: make(map[string]*WorkflowSession),
		maxSize:  maxSize,
	}
}

// CreateSession creates a new workflow session
// CreateSession 创建新的工作流会话
func (m *MemoryStorage) CreateSession(ctx context.Context, sessionID, workflowID, userID string) (*WorkflowSession, error) {
	// Validate inputs
	// 验证输入
	if sessionID == "" {
		return nil, ErrInvalidSessionID
	}
	if workflowID == "" {
		return nil, ErrInvalidWorkflowID
	}

	// Check context cancellation
	// 检查上下文取消
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if session already exists
	// 检查会话是否已存在
	if _, exists := m.sessions[sessionID]; exists {
		return nil, ErrSessionExists
	}

	// Check size limit
	// 检查大小限制
	if m.maxSize > 0 && len(m.sessions) >= m.maxSize {
		// Remove oldest session (simple FIFO eviction)
		// 删除最旧的会话（简单的 FIFO 驱逐）
		var oldestID string
		var oldestTime time.Time
		for id, session := range m.sessions {
			if oldestTime.IsZero() || session.CreatedAt.Before(oldestTime) {
				oldestID = id
				oldestTime = session.CreatedAt
			}
		}
		if oldestID != "" {
			delete(m.sessions, oldestID)
		}
	}

	// Create new session
	// 创建新会话
	session := NewWorkflowSession(sessionID, workflowID, userID)
	m.sessions[sessionID] = session

	return session, nil
}

// GetSession retrieves a workflow session by ID
// GetSession 通过 ID 检索工作流会话
func (m *MemoryStorage) GetSession(ctx context.Context, sessionID string) (*WorkflowSession, error) {
	// Validate input
	// 验证输入
	if sessionID == "" {
		return nil, ErrInvalidSessionID
	}

	// Check context cancellation
	// 检查上下文取消
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, ErrSessionNotFound
	}

	return session, nil
}

// UpdateSession updates an existing workflow session
// UpdateSession 更新现有的工作流会话
func (m *MemoryStorage) UpdateSession(ctx context.Context, session *WorkflowSession) error {
	// Validate input
	// 验证输入
	if session == nil || session.SessionID == "" {
		return ErrInvalidSessionID
	}

	// Check context cancellation
	// 检查上下文取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if session exists
	// 检查会话是否存在
	if _, exists := m.sessions[session.SessionID]; !exists {
		return ErrSessionNotFound
	}

	// Update session
	// 更新会话
	session.UpdatedAt = time.Now()
	m.sessions[session.SessionID] = session

	return nil
}

// DeleteSession deletes a workflow session by ID
// DeleteSession 通过 ID 删除工作流会话
func (m *MemoryStorage) DeleteSession(ctx context.Context, sessionID string) error {
	// Validate input
	// 验证输入
	if sessionID == "" {
		return ErrInvalidSessionID
	}

	// Check context cancellation
	// 检查上下文取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if session exists
	// 检查会话是否存在
	if _, exists := m.sessions[sessionID]; !exists {
		return ErrSessionNotFound
	}

	delete(m.sessions, sessionID)

	return nil
}

// ListSessions lists all sessions for a given workflow
// ListSessions 列出给定工作流的所有会话
func (m *MemoryStorage) ListSessions(ctx context.Context, workflowID string, limit, offset int) ([]*WorkflowSession, error) {
	// Validate input
	// 验证输入
	if workflowID == "" {
		return nil, ErrInvalidWorkflowID
	}

	// Check context cancellation
	// 检查上下文取消
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Filter sessions by workflow ID
	// 按工作流 ID 过滤会话
	filtered := make([]*WorkflowSession, 0)
	for _, session := range m.sessions {
		if session.WorkflowID == workflowID {
			filtered = append(filtered, session)
		}
	}

	// Apply offset and limit
	// 应用偏移量和限制
	return applyPagination(filtered, limit, offset), nil
}

// ListUserSessions lists all sessions for a given user
// ListUserSessions 列出给定用户的所有会话
func (m *MemoryStorage) ListUserSessions(ctx context.Context, userID string, limit, offset int) ([]*WorkflowSession, error) {
	// Check context cancellation
	// 检查上下文取消
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Filter sessions by user ID
	// 按用户 ID 过滤会话
	filtered := make([]*WorkflowSession, 0)
	for _, session := range m.sessions {
		if session.UserID == userID {
			filtered = append(filtered, session)
		}
	}

	// Apply offset and limit
	// 应用偏移量和限制
	return applyPagination(filtered, limit, offset), nil
}

// Clear removes all sessions older than the specified duration
// Clear 删除早于指定持续时间的所有会话
func (m *MemoryStorage) Clear(ctx context.Context, olderThan time.Duration) (int, error) {
	// Check context cancellation
	// 检查上下文取消
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	cutoffTime := time.Now().Add(-olderThan)
	count := 0

	// Remove old sessions
	// 删除旧会话
	for id, session := range m.sessions {
		if session.CreatedAt.Before(cutoffTime) {
			delete(m.sessions, id)
			count++
		}
	}

	return count, nil
}

// Close closes the storage and releases any resources
// Close 关闭存储并释放任何资源
func (m *MemoryStorage) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear all sessions
	// 清除所有会话
	m.sessions = make(map[string]*WorkflowSession)

	return nil
}

// GetStats returns statistics about the stored sessions
// GetStats 返回有关存储会话的统计信息
func (m *MemoryStorage) GetStats(ctx context.Context) (*SessionStats, error) {
	// Check context cancellation
	// 检查上下文取消
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &SessionStats{
		TotalSessions: len(m.sessions),
	}

	var totalDuration time.Duration
	completedCount := 0

	for _, session := range m.sessions {
		stats.TotalRuns += session.CountRuns()
		stats.CompletedRuns += session.CountCompletedRuns()
		stats.SuccessfulRuns += session.CountSuccessfulRuns()
		stats.FailedRuns += (session.CountCompletedRuns() - session.CountSuccessfulRuns())

		// Calculate average duration
		// 计算平均持续时间
		for _, run := range session.Runs {
			if run.IsCompleted() {
				totalDuration += run.Duration()
				completedCount++
			}
		}
	}

	if completedCount > 0 {
		stats.AverageDuration = totalDuration / time.Duration(completedCount)
	}

	return stats, nil
}

// GetWorkflowStats returns statistics for a specific workflow
// GetWorkflowStats 返回特定工作流的统计信息
func (m *MemoryStorage) GetWorkflowStats(ctx context.Context, workflowID string) (*SessionStats, error) {
	// Validate input
	// 验证输入
	if workflowID == "" {
		return nil, ErrInvalidWorkflowID
	}

	// Check context cancellation
	// 检查上下文取消
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &SessionStats{}
	var totalDuration time.Duration
	completedCount := 0

	for _, session := range m.sessions {
		if session.WorkflowID == workflowID {
			stats.TotalSessions++
			stats.TotalRuns += session.CountRuns()
			stats.CompletedRuns += session.CountCompletedRuns()
			stats.SuccessfulRuns += session.CountSuccessfulRuns()
			stats.FailedRuns += (session.CountCompletedRuns() - session.CountSuccessfulRuns())

			// Calculate average duration
			// 计算平均持续时间
			for _, run := range session.Runs {
				if run.IsCompleted() {
					totalDuration += run.Duration()
					completedCount++
				}
			}
		}
	}

	if completedCount > 0 {
		stats.AverageDuration = totalDuration / time.Duration(completedCount)
	}

	return stats, nil
}

// applyPagination applies offset and limit to a session list
// applyPagination 对会话列表应用偏移量和限制
func applyPagination(sessions []*WorkflowSession, limit, offset int) []*WorkflowSession {
	// Apply offset
	// 应用偏移量
	if offset > 0 {
		if offset >= len(sessions) {
			return []*WorkflowSession{}
		}
		sessions = sessions[offset:]
	}

	// Apply limit
	// 应用限制
	if limit > 0 && len(sessions) > limit {
		sessions = sessions[:limit]
	}

	return sessions
}
