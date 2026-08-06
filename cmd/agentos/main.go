package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/rexleimo/agno-go/pkg/agentos"
	postgresstore "github.com/rexleimo/agno-go/pkg/hno/session/db/postgres"
)

func main() {
	config, err := configFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	server, err := agentos.NewServer(config)
	if err != nil {
		log.Fatalf("create AgentOS server: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil {
			log.Fatalf("AgentOS server: %v", err)
		}
	case signal := <-sigCh:
		log.Printf("received %s, shutting down AgentOS", signal)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("AgentOS shutdown: %v", err)
	}
}

func configFromEnv() (*agentos.Config, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	config := &agentos.Config{
		Address:      envOrDefault("AGENTOS_ADDRESS", ":8080"),
		Prefix:       strings.TrimSpace(os.Getenv("AGENTOS_PREFIX")),
		Debug:        parseBool(os.Getenv("AGENTOS_DEBUG")),
		Logger:       logger,
		AllowOrigins: splitCSV(os.Getenv("AGENTOS_ALLOW_ORIGINS")),
	}

	if err := configureSessionStorage(config); err != nil {
		return nil, err
	}

	chromaURL := firstEnv("CHROMA_URL", "CHROMADB_URL")
	collection := strings.TrimSpace(os.Getenv("KNOWLEDGE_COLLECTION"))
	knowledgeEnabled := parseBool(os.Getenv("KNOWLEDGE_ENABLED")) || parseBool(os.Getenv("AGENTOS_KNOWLEDGE_ENABLED"))
	if !knowledgeEnabled {
		return config, nil
	}

	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if chromaURL == "" || collection == "" || apiKey == "" {
		return nil, fmt.Errorf("knowledge API requires CHROMA_URL (or CHROMADB_URL), KNOWLEDGE_COLLECTION, and OPENAI_API_KEY")
	}

	config.KnowledgeAPI = &agentos.KnowledgeAPIOptions{
		EnableHealth: !strings.EqualFold(strings.TrimSpace(os.Getenv("KNOWLEDGE_HEALTH_ENABLED")), "false"),
	}

	config.VectorDBConfig = &agentos.VectorDBConfig{
		Type:           "chromadb",
		BaseURL:        chromaURL,
		CollectionName: collection,
		Database:       envOrDefault("CHROMA_DATABASE", "default_database"),
		Tenant:         envOrDefault("CHROMA_TENANT", "default_tenant"),
	}
	config.EmbeddingConfig = &agentos.EmbeddingConfig{
		Provider: "openai",
		APIKey:   apiKey,
		Model:    envOrDefault("EMBEDDING_MODEL", "text-embedding-3-small"),
		BaseURL:  strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")),
	}
	return config, nil
}

func configureSessionStorage(config *agentos.Config) error {
	dsn := firstEnv("AGENTOS_SESSION_DSN", "AGNO_PG_DSN", "DATABASE_URL")
	if dsn == "" {
		return nil
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open AgentOS session database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("connect AgentOS session database: %w", err)
	}

	opts := make([]postgresstore.Option, 0, 2)
	if schema := strings.TrimSpace(os.Getenv("AGENTOS_SESSION_SCHEMA")); schema != "" {
		opts = append(opts, postgresstore.WithSchema(schema))
	}
	if table := strings.TrimSpace(os.Getenv("AGENTOS_SESSION_TABLE")); table != "" {
		opts = append(opts, postgresstore.WithTable(table))
	}
	storage, err := postgresstore.NewStorage(db, opts...)
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("create AgentOS session storage: %w", err)
	}

	config.SessionStorage = storage
	return nil
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	var values []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, item)
		}
	}
	return values
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
