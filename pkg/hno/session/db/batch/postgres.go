package batch

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/rexleimo/agno-go/pkg/hno/session"
)

// PostgresBatchWriter PostgreSQL 批量写入器
// PostgresBatchWriter is the PostgreSQL batch writer implementation
type PostgresBatchWriter struct {
	db     *sql.DB
	config *Config
}

// NewPostgresBatchWriter 创建 PostgreSQL 批量写入器
// NewPostgresBatchWriter creates a new PostgreSQL batch writer
func NewPostgresBatchWriter(db *sql.DB, config *Config) (*PostgresBatchWriter, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}
	if config == nil {
		config = DefaultConfig()
	} else {
		config = normalizeConfig(config)
	}

	return &PostgresBatchWriter{
		db:     db,
		config: config,
	}, nil
}

// UpsertSessions 批量插入或更新 sessions
// UpsertSessions batch inserts or updates sessions
func (w *PostgresBatchWriter) UpsertSessions(
	ctx context.Context,
	sessions []*session.Session,
	preserveUpdatedAt bool,
) error {
	if len(sessions) == 0 {
		return nil
	}

	batchSize := w.config.BatchSize
	if batchSize <= 0 {
		batchSize = len(sessions)
	}

	minBatch := w.config.MinBatchSize
	if minBatch <= 0 {
		minBatch = 1
	}
	if minBatch > batchSize {
		minBatch = batchSize
	}

	maxRetries := w.config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1
	}

	throttle := w.config.ThrottleInterval
	if throttle < 0 {
		throttle = 0
	}

	for start := 0; start < len(sessions); {
		remaining := len(sessions) - start
		currentSize := batchSize
		if currentSize > remaining {
			currentSize = remaining
		}
		if currentSize < minBatch {
			currentSize = remaining
		}
		if currentSize < minBatch {
			currentSize = minBatch
		}
		if currentSize > remaining {
			currentSize = remaining
		}

		attemptSize := currentSize
		for attempt := 0; attempt < maxRetries; attempt++ {
			if attemptSize <= 0 {
				attemptSize = 1
			}
			if attemptSize > remaining {
				attemptSize = remaining
			}

			batchCtx := ctx
			var cancel context.CancelFunc
			if w.config.TimeoutSeconds > 0 {
				batchCtx, cancel = context.WithTimeout(ctx, time.Duration(w.config.TimeoutSeconds)*time.Second)
			}

			err := w.writeBatch(batchCtx, sessions[start:start+attemptSize], preserveUpdatedAt)
			if cancel != nil {
				cancel()
			}
			if err == nil {
				start += attemptSize
				if throttle > 0 && start < len(sessions) {
					time.Sleep(throttle)
				}
				break
			}

			if attempt == maxRetries-1 || attemptSize <= minBatch {
				return err
			}

			newSize := attemptSize / 2
			if newSize < minBatch {
				newSize = minBatch
			}
			if newSize >= attemptSize {
				return err
			}
			attemptSize = newSize
		}
	}

	return nil
}

func (w *PostgresBatchWriter) writeBatch(
	ctx context.Context,
	sessions []*session.Session,
	preserveUpdatedAt bool,
) error {
	if len(sessions) == 0 {
		return nil
	}

	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := w.upsertViaTemp(ctx, tx, sessions, preserveUpdatedAt); err != nil {
		return fmt.Errorf("failed to upsert sessions: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// upsertViaTemp 使用临时表+COPY策略批量写入
// upsertViaTemp uses temp table + COPY strategy for batch insert
func (w *PostgresBatchWriter) upsertViaTemp(
	ctx context.Context,
	tx *sql.Tx,
	sessions []*session.Session,
	preserveUpdatedAt bool,
) error {
	// 1. 创建临时表
	// 1. Create temporary table
	createTempSQL := `
		CREATE TEMPORARY TABLE temp_sessions (
			session_id VARCHAR(255),
			agent_id VARCHAR(255),
			team_id VARCHAR(255),
			workflow_id VARCHAR(255),
			user_id VARCHAR(255),
			name VARCHAR(255),
			metadata JSONB,
			state JSONB,
			agent_data JSONB,
			runs JSONB,
			summary JSONB,
			created_at TIMESTAMP WITH TIME ZONE,
			updated_at TIMESTAMP WITH TIME ZONE
		) ON COMMIT DROP
	`
	if _, err := tx.ExecContext(ctx, createTempSQL); err != nil {
		return fmt.Errorf("failed to create temp table: %w", err)
	}

	// 2. 使用 COPY 批量导入临时表
	// 2. Use COPY to bulk import into temp table
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn(
		"temp_sessions",
		"session_id", "agent_id", "team_id", "workflow_id", "user_id",
		"name", "metadata", "state", "agent_data", "runs", "summary",
		"created_at", "updated_at",
	))
	if err != nil {
		return fmt.Errorf("failed to prepare COPY statement: %w", err)
	}
	defer stmt.Close()

	// 批量添加行 / Batch add rows
	for _, s := range sessions {
		// 序列化 JSON 字段 / Serialize JSON fields
		metadata, _ := json.Marshal(s.Metadata)
		state, _ := json.Marshal(s.State)
		agentData, _ := json.Marshal(s.AgentData)
		runs, _ := json.Marshal(s.Runs)
		summary, _ := json.Marshal(s.Summary)

		updatedAt := s.UpdatedAt
		if !preserveUpdatedAt || updatedAt.IsZero() {
			updatedAt = time.Now()
		}

		if _, err := stmt.ExecContext(ctx,
			s.SessionID, s.AgentID, s.TeamID, s.WorkflowID, s.UserID,
			s.Name, metadata, state, agentData, runs, summary,
			s.CreatedAt, updatedAt,
		); err != nil {
			return fmt.Errorf("failed to add row to COPY: %w", err)
		}
	}

	// 执行 COPY / Execute COPY
	if _, err := stmt.ExecContext(ctx); err != nil {
		return fmt.Errorf("failed to execute COPY: %w", err)
	}

	// 3. 从临时表 UPSERT 到主表
	// 3. UPSERT from temp table to main table
	upsertSQL := w.buildUpsertSQL(preserveUpdatedAt)
	if _, err := tx.ExecContext(ctx, upsertSQL); err != nil {
		return fmt.Errorf("failed to upsert from temp table: %w", err)
	}

	return nil
}

// buildUpsertSQL 构建 UPSERT SQL 语句
// buildUpsertSQL builds the UPSERT SQL statement
func (w *PostgresBatchWriter) buildUpsertSQL(preserveUpdatedAt bool) string {
	updateClause := `
		agent_id = EXCLUDED.agent_id,
		team_id = EXCLUDED.team_id,
		workflow_id = EXCLUDED.workflow_id,
		user_id = EXCLUDED.user_id,
		name = EXCLUDED.name,
		metadata = EXCLUDED.metadata,
		state = EXCLUDED.state,
		agent_data = EXCLUDED.agent_data,
		runs = EXCLUDED.runs,
		summary = EXCLUDED.summary
	`

	if preserveUpdatedAt {
		// 保留原 updated_at / Preserve original updated_at
		updateClause += `,
		updated_at = EXCLUDED.updated_at`
	} else {
		// 使用当前时间 / Use current time
		updateClause += `,
		updated_at = CURRENT_TIMESTAMP`
	}

	return fmt.Sprintf(`
		INSERT INTO sessions (
			session_id, agent_id, team_id, workflow_id, user_id,
			name, metadata, state, agent_data, runs, summary,
			created_at, updated_at
		)
		SELECT
			session_id, agent_id, team_id, workflow_id, user_id,
			name, metadata, state, agent_data, runs, summary,
			created_at, updated_at
		FROM temp_sessions
		ON CONFLICT (session_id) DO UPDATE SET
			%s
	`, updateClause)
}

// Close 关闭批量写入器
// Close closes the batch writer
func (w *PostgresBatchWriter) Close() error {
	// PostgreSQL 连接由外部管理,这里不关闭
	// PostgreSQL connection is managed externally, don't close here
	return nil
}

func normalizeConfig(cfg *Config) *Config {
	defaults := DefaultConfig()
	normalized := *cfg

	if normalized.BatchSize <= 0 {
		normalized.BatchSize = defaults.BatchSize
	}
	if normalized.MinBatchSize <= 0 {
		normalized.MinBatchSize = defaults.MinBatchSize
	}
	if normalized.MinBatchSize > normalized.BatchSize {
		normalized.MinBatchSize = normalized.BatchSize
	}
	if normalized.MaxRetries <= 0 {
		normalized.MaxRetries = defaults.MaxRetries
	}
	if normalized.TimeoutSeconds < 0 {
		normalized.TimeoutSeconds = defaults.TimeoutSeconds
	}
	if normalized.ThrottleInterval < 0 {
		normalized.ThrottleInterval = 0
	}
	return &normalized
}
