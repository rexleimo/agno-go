package workflow

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrSessionNotFound is returned when a session is not found
	// ErrSessionNotFound 当会话未找到时返回
	ErrSessionNotFound = errors.New("session not found")

	// ErrSessionExists is returned when trying to create a session that already exists
	// ErrSessionExists 当尝试创建已存在的会话时返回
	ErrSessionExists = errors.New("session already exists")

	// ErrInvalidSessionID is returned when the session ID is invalid
	// ErrInvalidSessionID 当会话 ID 无效时返回
	ErrInvalidSessionID = errors.New("invalid session ID")

	// ErrInvalidWorkflowID is returned when the workflow ID is invalid
	// ErrInvalidWorkflowID 当工作流 ID 无效时返回
	ErrInvalidWorkflowID = errors.New("invalid workflow ID")
)

// WorkflowStorage defines the interface for storing and retrieving workflow sessions
// WorkflowStorage 定义用于存储和检索工作流会话的接口
type WorkflowStorage interface {
	// CreateSession creates a new workflow session
	// CreateSession 创建新的工作流会话
	// Returns ErrSessionExists if a session with the same ID already exists
	// 如果具有相同 ID 的会话已存在则返回 ErrSessionExists
	CreateSession(ctx context.Context, sessionID, workflowID, userID string) (*WorkflowSession, error)

	// GetSession retrieves a workflow session by ID
	// GetSession 通过 ID 检索工作流会话
	// Returns ErrSessionNotFound if the session does not exist
	// 如果会话不存在则返回 ErrSessionNotFound
	GetSession(ctx context.Context, sessionID string) (*WorkflowSession, error)

	// UpdateSession updates an existing workflow session
	// UpdateSession 更新现有的工作流会话
	// Returns ErrSessionNotFound if the session does not exist
	// 如果会话不存在则返回 ErrSessionNotFound
	UpdateSession(ctx context.Context, session *WorkflowSession) error

	// DeleteSession deletes a workflow session by ID
	// DeleteSession 通过 ID 删除工作流会话
	// Returns ErrSessionNotFound if the session does not exist
	// 如果会话不存在则返回 ErrSessionNotFound
	DeleteSession(ctx context.Context, sessionID string) error

	// ListSessions lists all sessions for a given workflow
	// ListSessions 列出给定工作流的所有会话
	// Returns empty slice if no sessions found
	// 如果未找到会话则返回空切片
	ListSessions(ctx context.Context, workflowID string, limit, offset int) ([]*WorkflowSession, error)

	// ListUserSessions lists all sessions for a given user
	// ListUserSessions 列出给定用户的所有会话
	// Returns empty slice if no sessions found
	// 如果未找到会话则返回空切片
	ListUserSessions(ctx context.Context, userID string, limit, offset int) ([]*WorkflowSession, error)

	// Clear removes all sessions older than the specified duration
	// Clear 删除早于指定持续时间的所有会话
	Clear(ctx context.Context, olderThan time.Duration) (int, error)

	// Close closes the storage and releases any resources
	// Close 关闭存储并释放任何资源
	Close() error
}

// SessionFilter defines filter criteria for querying sessions
// SessionFilter 定义查询会话的过滤条件
type SessionFilter struct {
	// WorkflowID filters sessions by workflow ID
	// WorkflowID 按工作流 ID 过滤会话
	WorkflowID string

	// UserID filters sessions by user ID
	// UserID 按用户 ID 过滤会话
	UserID string

	// CreatedAfter filters sessions created after this time
	// CreatedAfter 过滤在此时间之后创建的会话
	CreatedAfter time.Time

	// CreatedBefore filters sessions created before this time
	// CreatedBefore 过滤在此时间之前创建的会话
	CreatedBefore time.Time

	// Limit is the maximum number of sessions to return (0 = no limit)
	// Limit 是要返回的最大会话数（0 = 无限制）
	Limit int

	// Offset is the number of sessions to skip
	// Offset 是要跳过的会话数
	Offset int
}

// SessionStats provides statistics about workflow sessions
// SessionStats 提供有关工作流会话的统计信息
type SessionStats struct {
	// TotalSessions is the total number of sessions
	// TotalSessions 是会话的总数
	TotalSessions int

	// TotalRuns is the total number of workflow runs across all sessions
	// TotalRuns 是所有会话中工作流运行的总数
	TotalRuns int

	// CompletedRuns is the number of completed runs
	// CompletedRuns 是已完成运行的数量
	CompletedRuns int

	// SuccessfulRuns is the number of successful runs
	// SuccessfulRuns 是成功运行的数量
	SuccessfulRuns int

	// FailedRuns is the number of failed runs
	// FailedRuns 是失败运行的数量
	FailedRuns int

	// AverageDuration is the average duration of completed runs
	// AverageDuration 是已完成运行的平均持续时间
	AverageDuration time.Duration
}

// StorageStats is an optional interface for storage implementations that support statistics
// StorageStats 是支持统计信息的存储实现的可选接口
type StorageStats interface {
	// GetStats returns statistics about the stored sessions
	// GetStats 返回有关存储会话的统计信息
	GetStats(ctx context.Context) (*SessionStats, error)

	// GetWorkflowStats returns statistics for a specific workflow
	// GetWorkflowStats 返回特定工作流的统计信息
	GetWorkflowStats(ctx context.Context, workflowID string) (*SessionStats, error)
}
