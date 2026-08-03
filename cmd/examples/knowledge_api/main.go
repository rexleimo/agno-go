package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rexleimo/agno-go/pkg/agentos"
)

func main() {
	// 配置日志
	// Configure logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// 从环境变量获取配置
	// Get configuration from environment variables
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	chromaURL := os.Getenv("CHROMADB_URL")
	if chromaURL == "" {
		chromaURL = "http://localhost:8000" // 默认值 / Default value
	}

	// 创建服务器配置
	// Create server configuration
	config := &agentos.Config{
		Address: ":8080",
		Prefix:  "",
		Debug:   true,
		Logger:  logger,
		VectorDBConfig: &agentos.VectorDBConfig{
			Type:           "chromadb",
			BaseURL:        chromaURL,
			CollectionName: "hno_knowledge",
			Database:       "default_database",
			Tenant:         "default_tenant",
		},
		EmbeddingConfig: &agentos.EmbeddingConfig{
			Provider: "openai",
			APIKey:   openaiKey,
			Model:    "text-embedding-3-small",
			BaseURL:  "https://api.openai.com/v1",
		},
		AllowOrigins: []string{"*"},
	}

	// 创建服务器
	// Create server
	server, err := agentos.NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// 启动服务器
	// Start server
	go func() {
		logger.Info("Starting Knowledge API server", "address", config.Address)
		logger.Info("Knowledge API endpoints:")
		logger.Info("  POST /api/v1/knowledge/search - Search knowledge base")
		logger.Info("  GET  /api/v1/knowledge/config - Get configuration")
		logger.Info("")
		logger.Info("Example usage:")
		logger.Info(`  curl -X POST http://localhost:8080/api/v1/knowledge/search \
    -H "Content-Type: application/json" \
    -d '{"query":"如何创建 Agent?","limit":5}'`)
		logger.Info("")
		logger.Info(`  curl http://localhost:8080/api/v1/knowledge/config`)
		logger.Info("")

		if err := server.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// 等待中断信号
	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// 优雅关闭
	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	logger.Info("Server stopped")
}
