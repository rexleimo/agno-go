package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/rexleimo/agno-go/pkg/hno/session/db/batch"
	"github.com/rexleimo/agno-go/pkg/hno/session"
)

func main() {
	// 连接数据库 / Connect to database
	// 默认连接字符串,可通过环境变量 DATABASE_URL 覆盖
	// Default connection string, can be overridden by DATABASE_URL env var
	connStr := "postgres://user:pass@localhost:5432/agno?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 测试连接 / Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	fmt.Println("✓ Successfully connected to database")

	// 创建批量写入器 / Create batch writer
	// 使用默认配置: BatchSize=5000, MaxRetries=3, TimeoutSeconds=30
	// Using default config: BatchSize=5000, MaxRetries=3, TimeoutSeconds=30
	writer, err := batch.NewPostgresBatchWriter(db, nil)
	if err != nil {
		log.Fatalf("Failed to create batch writer: %v", err)
	}
	defer writer.Close()
	fmt.Println("✓ Batch writer created")

	// 示例 1: 批量插入新 sessions
	// Example 1: Batch insert new sessions
	fmt.Println("\n--- Example 1: Batch Insert ---")
	sessions := generateSampleSessions(100)
	ctx := context.Background()

	start := time.Now()
	if err := writer.UpsertSessions(ctx, sessions, false); err != nil {
		log.Fatalf("Failed to upsert sessions: %v", err)
	}
	elapsed := time.Since(start)

	fmt.Printf("✓ Inserted %d sessions in %v (%.0f records/sec)\n",
		len(sessions), elapsed, float64(len(sessions))/elapsed.Seconds())

	// 示例 2: 更新现有 sessions (preserveUpdatedAt=false)
	// Example 2: Update existing sessions (preserveUpdatedAt=false)
	fmt.Println("\n--- Example 2: Update Sessions (Auto UpdatedAt) ---")
	for i := range sessions {
		sessions[i].Name = fmt.Sprintf("Updated Session %d", i+1)
		sessions[i].Metadata["updated"] = true
	}

	start = time.Now()
	if err := writer.UpsertSessions(ctx, sessions, false); err != nil {
		log.Fatalf("Failed to update sessions: %v", err)
	}
	elapsed = time.Since(start)

	fmt.Printf("✓ Updated %d sessions in %v (%.0f records/sec)\n",
		len(sessions), elapsed, float64(len(sessions))/elapsed.Seconds())

	// 示例 3: 批量迁移 (preserveUpdatedAt=true)
	// Example 3: Batch migration (preserveUpdatedAt=true)
	fmt.Println("\n--- Example 3: Migration (Preserve UpdatedAt) ---")
	migrationSessions := generateMigrationSessions(50)

	start = time.Now()
	if err := writer.UpsertSessions(ctx, migrationSessions, true); err != nil {
		log.Fatalf("Failed to migrate sessions: %v", err)
	}
	elapsed = time.Since(start)

	fmt.Printf("✓ Migrated %d sessions in %v (%.0f records/sec)\n",
		len(migrationSessions), elapsed, float64(len(migrationSessions))/elapsed.Seconds())

	// 示例 4: 使用自定义配置
	// Example 4: Using custom configuration
	fmt.Println("\n--- Example 4: Custom Config ---")
	customConfig := &batch.Config{
		BatchSize:      1000, // 较小的批量大小 / Smaller batch size
		MaxRetries:     5,    // 更多重试次数 / More retries
		TimeoutSeconds: 60,   // 更长的超时时间 / Longer timeout
	}

	customWriter, err := batch.NewPostgresBatchWriter(db, customConfig)
	if err != nil {
		log.Fatalf("Failed to create custom batch writer: %v", err)
	}
	defer customWriter.Close()

	customSessions := generateSampleSessions(10)
	if err := customWriter.UpsertSessions(ctx, customSessions, false); err != nil {
		log.Fatalf("Failed to upsert with custom config: %v", err)
	}

	fmt.Printf("✓ Inserted %d sessions with custom config (BatchSize=%d)\n",
		len(customSessions), customConfig.BatchSize)

	fmt.Println("\n✓ All examples completed successfully!")
}

// generateSampleSessions 生成示例 sessions
// generateSampleSessions generates sample sessions
func generateSampleSessions(count int) []*session.Session {
	sessions := make([]*session.Session, count)
	now := time.Now()

	for i := 0; i < count; i++ {
		sessions[i] = &session.Session{
			SessionID: fmt.Sprintf("session-%d-%d", now.Unix(), i),
			AgentID:   fmt.Sprintf("agent-%03d", i%10),
			UserID:    fmt.Sprintf("user-%03d", i%20),
			Name:      fmt.Sprintf("Session %d", i+1),
			Metadata: map[string]interface{}{
				"source":  "example",
				"batch":   true,
				"index":   i,
				"created": now.Format(time.RFC3339),
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

// generateMigrationSessions 生成用于迁移的历史 sessions
// generateMigrationSessions generates historical sessions for migration
func generateMigrationSessions(count int) []*session.Session {
	sessions := make([]*session.Session, count)
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < count; i++ {
		createdAt := baseTime.Add(time.Duration(i) * 24 * time.Hour)
		updatedAt := createdAt.Add(2 * time.Hour)

		sessions[i] = &session.Session{
			SessionID: fmt.Sprintf("migration-session-%d", i),
			AgentID:   fmt.Sprintf("legacy-agent-%03d", i%5),
			UserID:    fmt.Sprintf("legacy-user-%03d", i%10),
			Name:      fmt.Sprintf("Historical Session %d", i+1),
			Metadata: map[string]interface{}{
				"source":    "migration",
				"legacy_id": fmt.Sprintf("old-%d", i),
			},
			CreatedAt: createdAt,
			UpdatedAt: updatedAt, // 保留原始时间戳 / Preserve original timestamp
		}
	}

	return sessions
}
