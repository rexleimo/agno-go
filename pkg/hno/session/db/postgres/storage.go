package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/session"
	"github.com/rexleimo/agno-go/pkg/hno/session/db/batch"
)

var identifierPattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// Config 控制 Postgres 存储行为
// Config controls Postgres storage behaviour.
type Config struct {
	Schema      string
	Table       string
	BatchConfig *batch.Config
	Writer      batch.BatchWriter
}

// Option 是用于自定义配置的函数。
// Option customises the storage configuration.
type Option func(*Config)

// WithSchema 指定数据库 schema 名称。
func WithSchema(schema string) Option {
	return func(cfg *Config) {
		cfg.Schema = schema
	}
}

// WithTable 指定存储表名称。
func WithTable(table string) Option {
	return func(cfg *Config) {
		cfg.Table = table
	}
}

// WithBatchConfig 指定批量写入配置。
func WithBatchConfig(batchCfg *batch.Config) Option {
	return func(cfg *Config) {
		cfg.BatchConfig = batchCfg
	}
}

// WithBatchWriter 指定自定义批量写入器（测试用）。
func WithBatchWriter(writer batch.BatchWriter) Option {
	return func(cfg *Config) {
		cfg.Writer = writer
	}
}

// PostgresStorage 实现 session.Storage，支持 JSON 字段与批量导入。
type PostgresStorage struct {
	db          *sql.DB
	cfg         Config
	tableName   string
	columnNames string
	writer      batch.BatchWriter
}

const (
	defaultSchema = "public"
	defaultTable  = "sessions"
)

// NewStorage 创建 Postgres 存储实例。
func NewStorage(db *sql.DB, opts ...Option) (*PostgresStorage, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}

	cfg := Config{
		Schema: defaultSchema,
		Table:  defaultTable,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	tableName, err := buildQualifiedName(cfg.Schema, cfg.Table)
	if err != nil {
		return nil, err
	}

	columnList := strings.Join([]string{
		"session_id",
		"agent_id",
		"team_id",
		"workflow_id",
		"user_id",
		"name",
		"metadata",
		"state",
		"agent_data",
		"runs",
		"summary",
		"created_at",
		"updated_at",
	}, ", ")

	var writer batch.BatchWriter
	if cfg.Writer != nil {
		writer = cfg.Writer
	} else {
		w, err := batch.NewPostgresBatchWriter(db, cfg.BatchConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create batch writer: %w", err)
		}
		writer = w
	}

	return &PostgresStorage{
		db:          db,
		cfg:         cfg,
		tableName:   tableName,
		columnNames: columnList,
		writer:      writer,
	}, nil
}

// Create 插入新的 Session 记录（若已存在则更新）。
func (p *PostgresStorage) Create(ctx context.Context, sess *session.Session) error {
	if sess == nil {
		return fmt.Errorf("session cannot be nil")
	}
	return p.upsertSession(ctx, sess, false)
}

// Get 根据 SessionID 查询记录。
func (p *PostgresStorage) Get(ctx context.Context, sessionID string) (*session.Session, error) {
	if sessionID == "" {
		return nil, session.ErrInvalidSessionID
	}

	query := fmt.Sprintf(`SELECT %s FROM %s WHERE session_id = $1`, p.columnNames, p.tableName)

	row := p.db.QueryRowContext(ctx, query, sessionID)
	sess, err := scanSession(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, session.ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	return sess, nil
}

// Update 更新 Session（不存在则报错）。
func (p *PostgresStorage) Update(ctx context.Context, sess *session.Session) error {
	if sess == nil || sess.SessionID == "" {
		return session.ErrInvalidSessionID
	}

	// 确认目标存在
	if _, err := p.Get(ctx, sess.SessionID); err != nil {
		return err
	}

	return p.upsertSession(ctx, sess, false)
}

// Delete 删除 Session。
func (p *PostgresStorage) Delete(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return session.ErrInvalidSessionID
	}

	query := fmt.Sprintf(`DELETE FROM %s WHERE session_id = $1`, p.tableName)
	result, err := p.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return session.ErrSessionNotFound
	}
	return nil
}

// List 查询全部 Session，支持常用过滤条件。
func (p *PostgresStorage) List(ctx context.Context, filters map[string]interface{}) ([]*session.Session, error) {
	whereClause, args := buildWhereClause(filters)
	query := fmt.Sprintf(`SELECT %s FROM %s %s ORDER BY updated_at DESC`, p.columnNames, p.tableName, whereClause)

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*session.Session
	for rows.Next() {
		sess, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sessions, nil
}

// ListByAgent 查询代理下的全部 Session。
func (p *PostgresStorage) ListByAgent(ctx context.Context, agentID string) ([]*session.Session, error) {
	return p.List(ctx, map[string]interface{}{
		"agent_id": agentID,
	})
}

// ListByUser 查询用户下的 Session。
func (p *PostgresStorage) ListByUser(ctx context.Context, userID string) ([]*session.Session, error) {
	return p.List(ctx, map[string]interface{}{
		"user_id": userID,
	})
}

// Close releases the batch writer and database connection pool.
func (p *PostgresStorage) Close() error {
	var errs []error
	if p.writer != nil {
		if err := p.writer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if p.db != nil {
		if err := p.db.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// BulkUpsertSessions 批量导入 Session（迁移场景）。
func (p *PostgresStorage) BulkUpsertSessions(
	ctx context.Context,
	sessions []*session.Session,
	preserveUpdatedAt bool,
) error {
	if p.writer == nil {
		return fmt.Errorf("batch writer is not initialised")
	}
	return p.writer.UpsertSessions(ctx, sessions, preserveUpdatedAt)
}

func (p *PostgresStorage) upsertSession(ctx context.Context, sess *session.Session, preserveUpdatedAt bool) error {
	if sess.SessionID == "" {
		return session.ErrInvalidSessionID
	}

	now := time.Now()
	if sess.CreatedAt.IsZero() {
		sess.CreatedAt = now
	}
	if sess.UpdatedAt.IsZero() || !preserveUpdatedAt {
		sess.UpdatedAt = now
	}

	args, err := buildArgs(sess)
	if err != nil {
		return err
	}

	updateClause := strings.Join([]string{
		"agent_id = EXCLUDED.agent_id",
		"team_id = EXCLUDED.team_id",
		"workflow_id = EXCLUDED.workflow_id",
		"user_id = EXCLUDED.user_id",
		"name = EXCLUDED.name",
		"metadata = EXCLUDED.metadata",
		"state = EXCLUDED.state",
		"agent_data = EXCLUDED.agent_data",
		"runs = EXCLUDED.runs",
		"summary = EXCLUDED.summary",
		"created_at = COALESCE(" + p.tableName + ".created_at, EXCLUDED.created_at)",
	}, ", ")

	if preserveUpdatedAt {
		updateClause += ", updated_at = " + p.tableName + ".updated_at"
	} else {
		updateClause += ", updated_at = EXCLUDED.updated_at"
	}

	query := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)
ON CONFLICT (session_id) DO UPDATE SET %s`, p.tableName, p.columnNames, placeholders(len(args)), updateClause)

	_, err = p.db.ExecContext(ctx, query, args...)
	return err
}

func buildArgs(sess *session.Session) ([]interface{}, error) {
	metadata, err := json.Marshal(sess.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	state, err := json.Marshal(sess.State)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}
	agentData, err := json.Marshal(sess.AgentData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal agent data: %w", err)
	}
	runs, err := json.Marshal(sess.Runs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal runs: %w", err)
	}
	summary, err := json.Marshal(sess.Summary)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal summary: %w", err)
	}

	return []interface{}{
		sess.SessionID,
		sess.AgentID,
		sess.TeamID,
		sess.WorkflowID,
		sess.UserID,
		sess.Name,
		bytesOrNull(metadata),
		bytesOrNull(state),
		bytesOrNull(agentData),
		bytesOrNull(runs),
		bytesOrNull(summary),
		sess.CreatedAt,
		sess.UpdatedAt,
	}, nil
}

func scanSession(scanner interface {
	Scan(dest ...interface{}) error
}) (*session.Session, error) {
	var (
		sess           session.Session
		metadataBytes  sql.NullString
		stateBytes     sql.NullString
		agentDataBytes sql.NullString
		runsBytes      sql.NullString
		summaryBytes   sql.NullString
	)

	err := scanner.Scan(
		&sess.SessionID,
		&sess.AgentID,
		&sess.TeamID,
		&sess.WorkflowID,
		&sess.UserID,
		&sess.Name,
		&metadataBytes,
		&stateBytes,
		&agentDataBytes,
		&runsBytes,
		&summaryBytes,
		&sess.CreatedAt,
		&sess.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := decodeJSON(metadataBytes, &sess.Metadata); err != nil {
		return nil, err
	}
	if err := decodeJSON(stateBytes, &sess.State); err != nil {
		return nil, err
	}
	if err := decodeJSON(agentDataBytes, &sess.AgentData); err != nil {
		return nil, err
	}

	if err := decodeJSON(runsBytes, &sess.Runs); err != nil {
		return nil, err
	}
	if err := decodeJSON(summaryBytes, &sess.Summary); err != nil {
		return nil, err
	}

	if sess.Metadata == nil {
		sess.Metadata = make(map[string]interface{})
	}
	if sess.State == nil {
		sess.State = make(map[string]interface{})
	}
	if sess.AgentData == nil {
		sess.AgentData = make(map[string]interface{})
	}
	if sess.Runs == nil {
		sess.Runs = make([]*agent.RunOutput, 0)
	}

	return &sess, nil
}

func decodeJSON(src sql.NullString, target interface{}) error {
	if !src.Valid || src.String == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(src.String), target); err != nil {
		return fmt.Errorf("failed to decode json: %w", err)
	}
	return nil
}

func bytesOrNull(b []byte) interface{} {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	return string(b)
}

func placeholders(n int) string {
	values := make([]string, n)
	for i := range values {
		values[i] = fmt.Sprintf("$%d", i+1)
	}
	return strings.Join(values, ", ")
}

func buildQualifiedName(schema, table string) (string, error) {
	if table == "" {
		return "", fmt.Errorf("table name cannot be empty")
	}
	if schema == "" {
		schema = defaultSchema
	}

	if !identifierPattern.MatchString(schema) {
		return "", fmt.Errorf("invalid schema name: %s", schema)
	}
	if !identifierPattern.MatchString(table) {
		return "", fmt.Errorf("invalid table name: %s", table)
	}

	return fmt.Sprintf(`"%s"."%s"`, schema, table), nil
}

func buildWhereClause(filters map[string]interface{}) (string, []interface{}) {
	if len(filters) == 0 {
		return "", nil
	}

	allowed := map[string]bool{
		"agent_id":    true,
		"user_id":     true,
		"team_id":     true,
		"workflow_id": true,
		"session_id":  true,
	}

	var (
		clauses []string
		args    []interface{}
	)

	argIndex := 1
	for key, value := range filters {
		if !allowed[key] || value == nil {
			continue
		}
		clauses = append(clauses, fmt.Sprintf(`%s = $%d`, key, argIndex))
		args = append(args, value)
		argIndex++
	}

	if len(clauses) == 0 {
		return "", nil
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}
