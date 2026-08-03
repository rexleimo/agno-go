package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/rexleimo/agno-go/pkg/hno/session/db/surreal"
	"github.com/rexleimo/agno-go/pkg/hno/session"
)

func main() {
	ctx := context.Background()

	clientCfg := surreal.ClientConfig{
		BaseURL:   getEnv("SURREAL_URL", "http://localhost:8000"),
		Namespace: getEnv("SURREAL_NAMESPACE", "demo"),
		Database:  getEnv("SURREAL_DATABASE", "demo"),
		Username:  os.Getenv("SURREAL_USERNAME"),
		Password:  os.Getenv("SURREAL_PASSWORD"),
	}

	client, err := surreal.NewClient(clientCfg)
	if err != nil {
		log.Fatalf("failed to create SurrealDB client: %v", err)
	}

	storage, err := surreal.NewStorage(client, &surreal.StorageConfig{Table: "sessions"})
	if err != nil {
		log.Fatalf("failed to create SurrealDB storage: %v", err)
	}

	sessionID := uuid.NewString()
	sess := session.NewSession(sessionID, "demo-agent")
	sess.UserID = "demo-user"
	sess.Metadata = map[string]interface{}{
		"source": "surreal_demo",
	}
	sess.State = map[string]interface{}{
		"step": 1,
	}

	log.Printf("🌱 creating session %s …", sessionID)
	if err := storage.Create(ctx, sess); err != nil {
		log.Fatalf("failed to create session: %v", err)
	}
	log.Printf("✅ session stored in SurrealDB")

	log.Printf("🔄 updating session state …")
	sess.State["last_seen"] = time.Now().UTC().Format(time.RFC3339)
	if err := storage.Update(ctx, sess); err != nil {
		log.Fatalf("failed to update session: %v", err)
	}

	log.Printf("📚 listing sessions for agent demo-agent …")
	sessions, err := storage.List(ctx, map[string]interface{}{
		"agent_id": "demo-agent",
	})
	if err != nil {
		log.Fatalf("failed to list sessions: %v", err)
	}
	for _, s := range sessions {
		log.Printf("• %s (user=%s, updated=%s)", s.SessionID, s.UserID, s.UpdatedAt.Format(time.RFC3339))
	}

	log.Printf("📊 fetching SurrealDB metrics …")
	metrics, err := storage.Metrics(ctx)
	if err != nil {
		log.Fatalf("failed to get metrics: %v", err)
	}
	log.Printf("total sessions: %d, active in 24h: %d, active in 1h: %d",
		metrics.TotalSessions, metrics.ActiveLast24h, metrics.UpdatedLastHour)

	log.Printf("🧹 deleting demo session …")
	if err := storage.Delete(ctx, sessionID); err != nil {
		log.Fatalf("failed to delete session: %v", err)
	}
	log.Printf("✨ cleanup complete")
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
