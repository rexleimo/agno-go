package batch

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rexleimo/agno-go/pkg/hno/session"
)

func BenchmarkPostgresBatchWriter_New(b *testing.B) {
	db, _, err := sqlmock.New()
	if err != nil {
		b.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := NewPostgresBatchWriter(db, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPostgresBatchWriter_BuildUpsertSQL(b *testing.B) {
	db, _, err := sqlmock.New()
	if err != nil {
		b.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	writer, err := NewPostgresBatchWriter(db, nil)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = writer.buildUpsertSQL(false)
	}
}

func BenchmarkPostgresBatchWriter_UpsertSessions_Empty(b *testing.B) {
	db, _, err := sqlmock.New()
	if err != nil {
		b.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	writer, err := NewPostgresBatchWriter(db, nil)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	sessions := []*session.Session{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := writer.UpsertSessions(ctx, sessions, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDefaultConfig(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = DefaultConfig()
	}
}

// 辅助函数:生成测试会话 / Helper function: generate test sessions
func generateBenchmarkSessions(count int) []*session.Session {
	sessions := make([]*session.Session, count)
	now := time.Now()

	for i := 0; i < count; i++ {
		sessions[i] = &session.Session{
			SessionID: "bench-session-id",
			AgentID:   "bench-agent-id",
			UserID:    "bench-user-id",
			Name:      "Benchmark Session",
			Metadata: map[string]interface{}{
				"key": "value",
			},
			State: map[string]interface{}{
				"status": "active",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	return sessions
}
