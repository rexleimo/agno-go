package postgres

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/session"
)

type mockBatchWriter struct {
	calls            int
	lastPreserveFlag bool
}

func (m *mockBatchWriter) UpsertSessions(ctx context.Context, sessions []*session.Session, preserveUpdatedAt bool) error {
	m.calls++
	m.lastPreserveFlag = preserveUpdatedAt
	return nil
}

func (m *mockBatchWriter) Close() error {
	return nil
}

func TestNewStorage_Defaults(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	storage, err := NewStorage(db)
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	if storage.cfg.Schema != defaultSchema {
		t.Fatalf("expected default schema %s, got %s", defaultSchema, storage.cfg.Schema)
	}
	if storage.cfg.Table != defaultTable {
		t.Fatalf("expected default table %s, got %s", defaultTable, storage.cfg.Table)
	}
}

func TestCreate_InsertsSession(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	writer := &mockBatchWriter{}
	storage, err := NewStorage(db, WithBatchWriter(writer))
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	sess := session.NewSession("session-1", "agent-1")
	sess.TeamID = "team-1"
	sess.UserID = "user-1"
	sess.Runs = []*agent.RunOutput{}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "public"."sessions"`)).
		WithArgs(
			sqlmock.AnyArg(), // session_id
			sqlmock.AnyArg(), // agent_id
			sqlmock.AnyArg(), // team_id
			sqlmock.AnyArg(), // workflow_id
			sqlmock.AnyArg(), // user_id
			sqlmock.AnyArg(), // name
			sqlmock.AnyArg(), // metadata
			sqlmock.AnyArg(), // state
			sqlmock.AnyArg(), // agent_data
			sqlmock.AnyArg(), // runs
			sqlmock.AnyArg(), // summary
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
		).WillReturnResult(sqlmock.NewResult(1, 1))

	if err := storage.Create(context.Background(), sess); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("not all expectations met: %v", err)
	}
}

func TestGet_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	storage, err := NewStorage(db, WithBatchWriter(&mockBatchWriter{}))
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT session_id, agent_id, team_id, workflow_id, user_id, name, metadata, state, agent_data, runs, summary, created_at, updated_at FROM "public"."sessions" WHERE session_id = $1`)).
		WithArgs("missing-session").
		WillReturnError(sql.ErrNoRows)

	_, err = storage.Get(context.Background(), "missing-session")
	if err != session.ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestList_WithFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	storage, err := NewStorage(db, WithBatchWriter(&mockBatchWriter{}))
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	rows := sqlmock.NewRows([]string{
		"session_id", "agent_id", "team_id", "workflow_id", "user_id", "name",
		"metadata", "state", "agent_data", "runs", "summary", "created_at", "updated_at",
	}).AddRow(
		"session-1", "agent-1", "team-1", "workflow-1", "user-1", "Demo",
		"{}", "{}", "{}", "[]", "null", time.Now(), time.Now(),
	)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT session_id, agent_id, team_id, workflow_id, user_id, name, metadata, state, agent_data, runs, summary, created_at, updated_at FROM "public"."sessions" WHERE agent_id = $1 ORDER BY updated_at DESC`)).
		WithArgs("agent-1").
		WillReturnRows(rows)

	result, err := storage.List(context.Background(), map[string]interface{}{"agent_id": "agent-1"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 session, got %d", len(result))
	}
}

func TestDelete_Session(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	storage, err := NewStorage(db, WithBatchWriter(&mockBatchWriter{}))
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "public"."sessions" WHERE session_id = $1`)).
		WithArgs("session-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := storage.Delete(context.Background(), "session-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestBulkUpsertSessions_DelegatesToWriter(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	writer := &mockBatchWriter{}
	storage, err := NewStorage(db, WithBatchWriter(writer))
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	sess := session.NewSession("session-1", "agent-1")
	if err := storage.BulkUpsertSessions(context.Background(), []*session.Session{sess}, true); err != nil {
		t.Fatalf("BulkUpsertSessions() error = %v", err)
	}
	if writer.calls != 1 {
		t.Fatalf("expected writer to be called once, got %d", writer.calls)
	}
	if !writer.lastPreserveFlag {
		t.Fatalf("expected preserve flag to be true")
	}
}

func TestBuildWhereClause_Empty(t *testing.T) {
	clause, args := buildWhereClause(map[string]interface{}{})
	if clause != "" || len(args) != 0 {
		t.Fatalf("expected empty clause, got %s args=%v", clause, args)
	}
}
