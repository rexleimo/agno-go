//go:build integration
// +build integration

package batch

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/rexleimo/agno-go/pkg/hno/session"
)

func TestIntegration_PostgresBatchWriter_UpsertSessions(t *testing.T) {
	// 连接测试数据库
	// Connect to test database
	connStr := "postgres://test:test@localhost:5432/agno_test?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Cannot connect to test database: %v", err)
		return
	}
	defer db.Close()

	// 测试连接 / Test connection
	if err := db.Ping(); err != nil {
		t.Skipf("Cannot ping test database: %v", err)
		return
	}

	// 初始化表结构
	// Initialize schema
	initSchema(t, db)

	writer, err := NewPostgresBatchWriter(db, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()

	ctx := context.Background()

	t.Run("basic upsert", func(t *testing.T) {
		sessions := []*session.Session{
			{
				SessionID: "test-session-1",
				AgentID:   "agent-001",
				UserID:    "user-001",
				Name:      "Test Session 1",
				Metadata:  map[string]interface{}{"key": "value"},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}

		err := writer.UpsertSessions(ctx, sessions, false)
		if err != nil {
			t.Fatalf("UpsertSessions failed: %v", err)
		}

		// 验证数据
		// Verify data
		var count int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions WHERE session_id = $1", "test-session-1").Scan(&count)
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("Expected 1 session, got %d", count)
		}
	})

	t.Run("preserve updated_at", func(t *testing.T) {
		originalTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		sessions := []*session.Session{
			{
				SessionID: "test-session-2",
				AgentID:   "agent-001",
				UserID:    "user-001",
				CreatedAt: originalTime,
				UpdatedAt: originalTime,
			},
		}

		err := writer.UpsertSessions(ctx, sessions, true)
		if err != nil {
			t.Fatalf("UpsertSessions failed: %v", err)
		}

		// 验证 updated_at 保留
		// Verify updated_at preserved
		var updatedAt time.Time
		err = db.QueryRowContext(ctx, "SELECT updated_at FROM sessions WHERE session_id = $1", "test-session-2").Scan(&updatedAt)
		if err != nil {
			t.Fatal(err)
		}

		// 允许微小的时间差异 (由于数据库精度)
		// Allow small time differences (due to database precision)
		diff := updatedAt.Sub(originalTime)
		if diff > time.Second || diff < -time.Second {
			t.Errorf("updated_at should be preserved: got %v, want %v (diff: %v)", updatedAt, originalTime, diff)
		}
	})

	t.Run("batch upsert multiple records", func(t *testing.T) {
		sessions := []*session.Session{
			{
				SessionID: "batch-session-1",
				AgentID:   "agent-001",
				UserID:    "user-001",
				Name:      "Batch Session 1",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				SessionID: "batch-session-2",
				AgentID:   "agent-002",
				TeamID:    "team-001",
				Name:      "Batch Session 2",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				SessionID:  "batch-session-3",
				AgentID:    "agent-003",
				WorkflowID: "workflow-001",
				Name:       "Batch Session 3",
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
		}

		err := writer.UpsertSessions(ctx, sessions, false)
		if err != nil {
			t.Fatalf("UpsertSessions failed: %v", err)
		}

		// 验证所有记录都被插入
		// Verify all records inserted
		var count int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions WHERE session_id LIKE 'batch-session-%'").Scan(&count)
		if err != nil {
			t.Fatal(err)
		}
		if count != 3 {
			t.Errorf("Expected 3 sessions, got %d", count)
		}
	})

	t.Run("update existing record", func(t *testing.T) {
		// 首次插入 / First insert
		sessions := []*session.Session{
			{
				SessionID: "update-test-session",
				AgentID:   "agent-001",
				Name:      "Original Name",
				Metadata:  map[string]interface{}{"version": 1},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}

		err := writer.UpsertSessions(ctx, sessions, false)
		if err != nil {
			t.Fatalf("First UpsertSessions failed: %v", err)
		}

		// 更新记录 / Update record
		time.Sleep(100 * time.Millisecond) // 确保 updated_at 不同 / Ensure different updated_at
		sessions[0].Name = "Updated Name"
		sessions[0].Metadata = map[string]interface{}{"version": 2}

		err = writer.UpsertSessions(ctx, sessions, false)
		if err != nil {
			t.Fatalf("Second UpsertSessions failed: %v", err)
		}

		// 验证更新 / Verify update
		var name string
		err = db.QueryRowContext(ctx, "SELECT name FROM sessions WHERE session_id = $1", "update-test-session").Scan(&name)
		if err != nil {
			t.Fatal(err)
		}
		if name != "Updated Name" {
			t.Errorf("Expected 'Updated Name', got '%s'", name)
		}

		// 验证只有一条记录 / Verify only one record
		var count int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions WHERE session_id = $1", "update-test-session").Scan(&count)
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("Expected 1 session after update, got %d", count)
		}
	})
}

func initSchema(t *testing.T, db *sql.DB) {
	schema := `
	DROP TABLE IF EXISTS sessions;
	CREATE TABLE sessions (
		session_id VARCHAR(255) PRIMARY KEY,
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
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}
}
