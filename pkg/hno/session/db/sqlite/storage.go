package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/rexleimo/agno-go/pkg/hno/session"
)

const (
	defaultTableName      = "sessions"
	defaultOperationLimit = 200 * time.Millisecond
)

// Config configures the SQLite storage behaviour.
type Config struct {
	Table            string
	OperationTimeout time.Duration
}

// Storage persists sessions in SQLite.
type Storage struct {
	db      *sql.DB
	table   string
	timeout time.Duration
}

// NewStorage constructs a new SQLite-backed session storage.
func NewStorage(db *sql.DB, cfg Config) (*Storage, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}

	table := cfg.Table
	if table == "" {
		table = defaultTableName
	}

	if err := ensureSchema(db, table); err != nil {
		return nil, err
	}

	timeout := cfg.OperationTimeout
	if timeout <= 0 {
		timeout = defaultOperationLimit
	}

	return &Storage{
		db:      db,
		table:   table,
		timeout: timeout,
	}, nil
}

// Create inserts or upserts a session record.
func (s *Storage) Create(ctx context.Context, sess *session.Session) error {
	if err := s.validateSession(sess); err != nil {
		return err
	}
	return s.upsert(ctx, sess)
}

// Get retrieves a session by ID.
func (s *Storage) Get(ctx context.Context, sessionID string) (*session.Session, error) {
	if err := ensureContext(ctx); err != nil {
		return nil, err
	}
	if sessionID == "" {
		return nil, session.ErrInvalidSessionID
	}

	ctx, cancel := s.applyTimeout(ctx)
	defer cancel()

	query := fmt.Sprintf(`SELECT session_id, agent_id, team_id, workflow_id, user_id,
		name, metadata, state, agent_data, runs, summary, created_at, updated_at
		FROM %s WHERE session_id = ?`, s.table)

	row := s.db.QueryRowContext(ctx, query, sessionID)
	record, err := scanRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, session.ErrSessionNotFound
		}
		return nil, err
	}

	return record.toSession()
}

// Update updates an existing session.
func (s *Storage) Update(ctx context.Context, sess *session.Session) error {
	if err := s.validateSession(sess); err != nil {
		return err
	}

	if err := ensureContext(ctx); err != nil {
		return err
	}

	ctx, cancel := s.applyTimeout(ctx)
	defer cancel()

	record, err := toRecord(sess)
	if err != nil {
		return err
	}
	record.UpdatedAt = time.Now().UTC()

	query := fmt.Sprintf(`UPDATE %s SET agent_id = ?, team_id = ?, workflow_id = ?, user_id = ?, name = ?,
		metadata = ?, state = ?, agent_data = ?, runs = ?, summary = ?, updated_at = ? WHERE session_id = ?`, s.table)

	result, err := s.db.ExecContext(ctx, query,
		record.AgentID,
		record.TeamID,
		record.WorkflowID,
		record.UserID,
		record.Name,
		record.Metadata,
		record.State,
		record.AgentData,
		record.Runs,
		record.Summary,
		record.UpdatedAt,
		record.SessionID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return session.ErrSessionNotFound
	}

	return nil
}

// Delete removes a session by ID.
func (s *Storage) Delete(ctx context.Context, sessionID string) error {
	if err := ensureContext(ctx); err != nil {
		return err
	}
	if sessionID == "" {
		return session.ErrInvalidSessionID
	}

	ctx, cancel := s.applyTimeout(ctx)
	defer cancel()

	query := fmt.Sprintf(`DELETE FROM %s WHERE session_id = ?`, s.table)
	result, err := s.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return session.ErrSessionNotFound
	}

	return nil
}

// List returns sessions that match optional filters.
func (s *Storage) List(ctx context.Context, filters map[string]interface{}) ([]*session.Session, error) {
	if err := ensureContext(ctx); err != nil {
		return nil, err
	}

	ctx, cancel := s.applyTimeout(ctx)
	defer cancel()

	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(fmt.Sprintf(`SELECT session_id, agent_id, team_id, workflow_id, user_id,
		name, metadata, state, agent_data, runs, summary, created_at, updated_at FROM %s`, s.table))

	var args []interface{}
	if len(filters) > 0 {
		clauses := make([]string, 0, len(filters))
		for _, field := range []string{"agent_id", "user_id", "team_id", "workflow_id", "session_id"} {
			if val, ok := filters[field]; ok {
				if str, ok := val.(string); ok && str != "" {
					clauses = append(clauses, fmt.Sprintf("%s = ?", field))
					args = append(args, str)
				}
			}
		}
		if len(clauses) > 0 {
			queryBuilder.WriteString(" WHERE ")
			queryBuilder.WriteString(strings.Join(clauses, " AND "))
		}
	}

	queryBuilder.WriteString(" ORDER BY updated_at DESC")

	rows, err := s.db.QueryContext(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*session.Session
	for rows.Next() {
		record, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		sess, err := record.toSession()
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

// ListByAgent returns sessions for the given agent ID.
func (s *Storage) ListByAgent(ctx context.Context, agentID string) ([]*session.Session, error) {
	return s.List(ctx, map[string]interface{}{"agent_id": agentID})
}

// ListByUser returns sessions for the given user ID.
func (s *Storage) ListByUser(ctx context.Context, userID string) ([]*session.Session, error) {
	return s.List(ctx, map[string]interface{}{"user_id": userID})
}

// Close closes the underlying database connection.
func (s *Storage) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Storage) upsert(ctx context.Context, sess *session.Session) error {
	if err := ensureContext(ctx); err != nil {
		return err
	}

	ctx, cancel := s.applyTimeout(ctx)
	defer cancel()

	record, err := toRecord(sess)
	if err != nil {
		return err
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	record.UpdatedAt = time.Now().UTC()

	query := fmt.Sprintf(`INSERT INTO %s (session_id, agent_id, team_id, workflow_id, user_id, name,
		metadata, state, agent_data, runs, summary, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET
		agent_id = excluded.agent_id,
		team_id = excluded.team_id,
		workflow_id = excluded.workflow_id,
		user_id = excluded.user_id,
		name = excluded.name,
		metadata = excluded.metadata,
		state = excluded.state,
		agent_data = excluded.agent_data,
		runs = excluded.runs,
		summary = excluded.summary,
		created_at = MIN(%s.created_at, excluded.created_at),
		updated_at = excluded.updated_at`, s.table, s.table)

	_, err = s.db.ExecContext(ctx, query,
		record.SessionID,
		record.AgentID,
		record.TeamID,
		record.WorkflowID,
		record.UserID,
		record.Name,
		record.Metadata,
		record.State,
		record.AgentData,
		record.Runs,
		record.Summary,
		record.CreatedAt,
		record.UpdatedAt,
	)
	return err
}

func (s *Storage) validateSession(sess *session.Session) error {
	if sess == nil {
		return fmt.Errorf("session cannot be nil")
	}
	if sess.SessionID == "" {
		return session.ErrInvalidSessionID
	}
	return nil
}

func (s *Storage) applyTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.timeout <= 0 {
		return ctx, func() {}
	}
	if deadline, ok := ctx.Deadline(); ok {
		if time.Until(deadline) <= s.timeout {
			return ctx, func() {}
		}
	}
	return context.WithTimeout(ctx, s.timeout)
}

func ensureSchema(db *sql.DB, table string) error {
	stmt := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		session_id TEXT PRIMARY KEY,
		agent_id TEXT,
		team_id TEXT,
		workflow_id TEXT,
		user_id TEXT,
		name TEXT,
		metadata TEXT,
		state TEXT,
		agent_data TEXT,
		runs TEXT,
		summary TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`, table)

	_, err := db.Exec(stmt)
	return err
}

func ensureContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

type record struct {
	SessionID  string
	AgentID    sql.NullString
	TeamID     sql.NullString
	WorkflowID sql.NullString
	UserID     sql.NullString
	Name       sql.NullString
	Metadata   []byte
	State      []byte
	AgentData  []byte
	Runs       []byte
	Summary    []byte
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func toRecord(sess *session.Session) (*record, error) {
	metadata, err := marshalJSON(sess.Metadata)
	if err != nil {
		return nil, err
	}
	state, err := marshalJSON(sess.State)
	if err != nil {
		return nil, err
	}
	agentData, err := marshalJSON(sess.AgentData)
	if err != nil {
		return nil, err
	}
	runs, err := marshalJSON(sess.Runs)
	if err != nil {
		return nil, err
	}
	summary, err := marshalJSON(sess.Summary)
	if err != nil {
		return nil, err
	}

	return &record{
		SessionID:  sess.SessionID,
		AgentID:    toNullString(sess.AgentID),
		TeamID:     toNullString(sess.TeamID),
		WorkflowID: toNullString(sess.WorkflowID),
		UserID:     toNullString(sess.UserID),
		Name:       toNullString(sess.Name),
		Metadata:   metadata,
		State:      state,
		AgentData:  agentData,
		Runs:       runs,
		Summary:    summary,
		CreatedAt:  sess.CreatedAt,
		UpdatedAt:  sess.UpdatedAt,
	}, nil
}

func (r *record) toSession() (*session.Session, error) {
	sess := &session.Session{
		SessionID:  r.SessionID,
		AgentID:    r.AgentID.String,
		TeamID:     r.TeamID.String,
		WorkflowID: r.WorkflowID.String,
		UserID:     r.UserID.String,
		Name:       r.Name.String,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}

	if len(r.Metadata) > 0 {
		if err := json.Unmarshal(r.Metadata, &sess.Metadata); err != nil {
			return nil, err
		}
	}
	if len(r.State) > 0 {
		if err := json.Unmarshal(r.State, &sess.State); err != nil {
			return nil, err
		}
	}
	if len(r.AgentData) > 0 {
		if err := json.Unmarshal(r.AgentData, &sess.AgentData); err != nil {
			return nil, err
		}
	}
	if len(r.Runs) > 0 {
		if err := json.Unmarshal(r.Runs, &sess.Runs); err != nil {
			return nil, err
		}
	}
	if len(r.Summary) > 0 {
		if err := json.Unmarshal(r.Summary, &sess.Summary); err != nil {
			return nil, err
		}
	}

	return sess, nil
}

func scanRecord(scanner interface {
	Scan(dest ...interface{}) error
}) (*record, error) {
	var rec record
	if err := scanner.Scan(
		&rec.SessionID,
		&rec.AgentID,
		&rec.TeamID,
		&rec.WorkflowID,
		&rec.UserID,
		&rec.Name,
		&rec.Metadata,
		&rec.State,
		&rec.AgentData,
		&rec.Runs,
		&rec.Summary,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &rec, nil
}

func toNullString(val string) sql.NullString {
	if val == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: val, Valid: true}
}

func marshalJSON(value interface{}) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	return json.Marshal(value)
}
