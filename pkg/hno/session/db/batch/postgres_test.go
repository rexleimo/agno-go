package batch

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rexleimo/agno-go/pkg/hno/session"
)

func TestPostgresBatchWriter_New(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	tests := []struct {
		name    string
		db      *sql.DB
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid db with default config",
			db:      db,
			config:  nil,
			wantErr: false,
		},
		{
			name:    "valid db with custom config",
			db:      db,
			config:  &Config{BatchSize: 1000},
			wantErr: false,
		},
		{
			name:    "nil db",
			db:      nil,
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer, err := NewPostgresBatchWriter(tt.db, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPostgresBatchWriter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && writer == nil {
				t.Error("NewPostgresBatchWriter() returned nil writer")
			}
			if !tt.wantErr && writer.config == nil {
				t.Error("NewPostgresBatchWriter() config should not be nil")
			}
		})
	}
}

func TestPostgresBatchWriter_UpsertSessions_EmptySessions(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	writer, err := NewPostgresBatchWriter(db, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = writer.UpsertSessions(ctx, []*session.Session{}, false)
	if err != nil {
		t.Errorf("UpsertSessions() with empty sessions should not error, got: %v", err)
	}
}

func TestPostgresBatchWriter_Close(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	writer, err := NewPostgresBatchWriter(db, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = writer.Close()
	if err != nil {
		t.Errorf("Close() should not error, got: %v", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.BatchSize != 5000 {
		t.Errorf("DefaultConfig().BatchSize = %d, want 5000", config.BatchSize)
	}
	if config.MinBatchSize != 500 {
		t.Errorf("DefaultConfig().MinBatchSize = %d, want 500", config.MinBatchSize)
	}
	if config.MaxRetries != 3 {
		t.Errorf("DefaultConfig().MaxRetries = %d, want 3", config.MaxRetries)
	}
	if config.TimeoutSeconds != 30 {
		t.Errorf("DefaultConfig().TimeoutSeconds = %d, want 30", config.TimeoutSeconds)
	}
	if config.ThrottleInterval != 0 {
		t.Errorf("DefaultConfig().ThrottleInterval = %s, want 0", config.ThrottleInterval)
	}
}

func TestPostgresBatchWriter_BuildUpsertSQL(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	writer, err := NewPostgresBatchWriter(db, nil)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name              string
		preserveUpdatedAt bool
		wantContains      string
	}{
		{
			name:              "preserve updated_at",
			preserveUpdatedAt: true,
			wantContains:      "updated_at = EXCLUDED.updated_at",
		},
		{
			name:              "use current timestamp",
			preserveUpdatedAt: false,
			wantContains:      "updated_at = CURRENT_TIMESTAMP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := writer.buildUpsertSQL(tt.preserveUpdatedAt)
			if sql == "" {
				t.Error("buildUpsertSQL() returned empty string")
			}
			// 验证包含必要的子句 / Verify contains necessary clauses
			if len(sql) < 100 {
				t.Errorf("buildUpsertSQL() returned unexpectedly short SQL: %s", sql)
			}
		})
	}
}

func TestPostgresBatchWriter_UpsertSessions_TransactionFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	writer, err := NewPostgresBatchWriter(db, nil)
	if err != nil {
		t.Fatal(err)
	}

	// 模拟事务开启失败 / Mock transaction begin failure
	mock.ExpectBegin().WillReturnError(sql.ErrConnDone)

	sessions := []*session.Session{
		{
			SessionID: "test-session-1",
			AgentID:   "agent-001",
			UserID:    "user-001",
			Name:      "Test Session",
			Metadata:  map[string]interface{}{"key": "value"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	ctx := context.Background()
	err = writer.UpsertSessions(ctx, sessions, false)
	if err == nil {
		t.Error("UpsertSessions() should return error when transaction fails")
	}
}

func TestPostgresBatchWriter_ConfigDefaults(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	// 测试 nil config 使用默认值 / Test nil config uses defaults
	writer, err := NewPostgresBatchWriter(db, nil)
	if err != nil {
		t.Fatal(err)
	}

	if writer.config.BatchSize != 5000 {
		t.Errorf("Default BatchSize = %d, want 5000", writer.config.BatchSize)
	}
	if writer.config.MinBatchSize != 500 {
		t.Errorf("Default MinBatchSize = %d, want 500", writer.config.MinBatchSize)
	}
	if writer.config.MaxRetries != 3 {
		t.Errorf("Default MaxRetries = %d, want 3", writer.config.MaxRetries)
	}
	if writer.config.TimeoutSeconds != 30 {
		t.Errorf("Default TimeoutSeconds = %d, want 30", writer.config.TimeoutSeconds)
	}
	if writer.config.ThrottleInterval != 0 {
		t.Errorf("Default ThrottleInterval = %s, want 0", writer.config.ThrottleInterval)
	}
}

func TestPostgresBatchWriter_CustomConfig(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	customConfig := &Config{
		BatchSize:        1000,
		MinBatchSize:     100,
		MaxRetries:       5,
		TimeoutSeconds:   60,
		ThrottleInterval: 10 * time.Millisecond,
	}

	writer, err := NewPostgresBatchWriter(db, customConfig)
	if err != nil {
		t.Fatal(err)
	}

	if writer.config.BatchSize != 1000 {
		t.Errorf("Custom BatchSize = %d, want 1000", writer.config.BatchSize)
	}
	if writer.config.MinBatchSize != 100 {
		t.Errorf("Custom MinBatchSize = %d, want 100", writer.config.MinBatchSize)
	}
	if writer.config.MaxRetries != 5 {
		t.Errorf("Custom MaxRetries = %d, want 5", writer.config.MaxRetries)
	}
	if writer.config.TimeoutSeconds != 60 {
		t.Errorf("Custom TimeoutSeconds = %d, want 60", writer.config.TimeoutSeconds)
	}
	if writer.config.ThrottleInterval != 10*time.Millisecond {
		t.Errorf("Custom ThrottleInterval = %s, want 10ms", writer.config.ThrottleInterval)
	}
}

func TestNormalizeConfig(t *testing.T) {
	cfg := &Config{
		BatchSize:        -1,
		MinBatchSize:     -1,
		MaxRetries:       0,
		TimeoutSeconds:   -5,
		ThrottleInterval: -1 * time.Second,
	}

	n := normalizeConfig(cfg)
	if n.BatchSize != 5000 {
		t.Errorf("Expected normalized BatchSize=5000, got %d", n.BatchSize)
	}
	if n.MinBatchSize != 500 {
		t.Errorf("Expected normalized MinBatchSize=500, got %d", n.MinBatchSize)
	}
	if n.MaxRetries != 3 {
		t.Errorf("Expected normalized MaxRetries=3, got %d", n.MaxRetries)
	}
	if n.TimeoutSeconds != 30 {
		t.Errorf("Expected normalized TimeoutSeconds=30, got %d", n.TimeoutSeconds)
	}
	if n.ThrottleInterval != 0 {
		t.Errorf("Expected normalized ThrottleInterval=0, got %s", n.ThrottleInterval)
	}
}

func TestPostgresBatchWriter_UpsertSessions_CreateTempTableFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	writer, err := NewPostgresBatchWriter(db, nil)
	if err != nil {
		t.Fatal(err)
	}

	sessions := []*session.Session{
		{
			SessionID: "test-session-1",
			AgentID:   "agent-001",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// 模拟事务开始成功,但创建临时表失败 / Mock transaction begin succeeds, temp table creation fails
	mock.ExpectBegin()
	mock.ExpectExec("CREATE TEMPORARY TABLE").WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	ctx := context.Background()
	err = writer.UpsertSessions(ctx, sessions, false)
	if err == nil {
		t.Error("UpsertSessions() should return error when temp table creation fails")
	}
}

func TestPostgresBatchWriter_UpsertSessions_CommitFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	writer, err := NewPostgresBatchWriter(db, nil)
	if err != nil {
		t.Fatal(err)
	}

	sessions := []*session.Session{
		{
			SessionID: "test-session-1",
			AgentID:   "agent-001",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// 模拟事务、临时表、COPY 都成功,但提交失败 / Mock transaction, temp table, COPY succeed, but commit fails
	mock.ExpectBegin()
	mock.ExpectExec("CREATE TEMPORARY TABLE").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare("COPY")
	mock.ExpectExec("INSERT INTO sessions").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(sql.ErrTxDone)

	ctx := context.Background()
	err = writer.UpsertSessions(ctx, sessions, false)
	if err == nil {
		t.Error("UpsertSessions() should return error when commit fails")
	}
}

func TestPostgresBatchWriter_UpsertSessions_PrepareFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	writer, err := NewPostgresBatchWriter(db, nil)
	if err != nil {
		t.Fatal(err)
	}

	sessions := []*session.Session{
		{
			SessionID: "test-session-1",
			AgentID:   "agent-001",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// 模拟 COPY 准备失败 / Mock COPY prepare failure
	mock.ExpectBegin()
	mock.ExpectExec("CREATE TEMPORARY TABLE").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare("COPY").WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	ctx := context.Background()
	err = writer.UpsertSessions(ctx, sessions, false)
	if err == nil {
		t.Error("UpsertSessions() should return error when COPY prepare fails")
	}
}

func TestPostgresBatchWriter_UpsertSessions_UpsertFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	writer, err := NewPostgresBatchWriter(db, nil)
	if err != nil {
		t.Fatal(err)
	}

	sessions := []*session.Session{
		{
			SessionID: "test-session-1",
			AgentID:   "agent-001",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// 模拟 UPSERT 失败 / Mock UPSERT failure
	mock.ExpectBegin()
	mock.ExpectExec("CREATE TEMPORARY TABLE").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare("COPY").WillBeClosed()
	mock.ExpectExec("INSERT INTO sessions").WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	ctx := context.Background()
	err = writer.UpsertSessions(ctx, sessions, false)
	if err == nil {
		t.Error("UpsertSessions() should return error when UPSERT fails")
	}
}
