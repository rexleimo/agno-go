package agentos

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/embeddings/openai"
	"github.com/rexleimo/agno-go/pkg/hno/session"
	"github.com/rexleimo/agno-go/pkg/hno/team"
	"github.com/rexleimo/agno-go/pkg/hno/vectordb"
	"github.com/rexleimo/agno-go/pkg/hno/vectordb/chromadb"
)

// Server represents the AgentOS HTTP server
type Server struct {
	router           *gin.Engine
	config           *Config
	sessionStorage   session.Storage
	agentRegistry    *AgentRegistry
	teamRegistry     *TeamRegistry
	logger           *slog.Logger
	httpServer       *http.Server
	knowledgeService *KnowledgeService // 知识库服务
	summaryManager   *session.SummaryManager
	instantiatedAt   time.Time
	docsMounted      bool
}

// Config holds server configuration
// Config 持有服务器配置
type Config struct {
	// Server address (default: :8080)
	// 服务器地址 (默认: :8080)
	Address string

	// API route prefix (e.g., "/api/v1", "/chat", empty for no prefix)
	// API 路由前缀 (例如: "/api/v1", "/chat", 空字符串表示无前缀)
	Prefix string

	// Session storage
	// Session 存储
	SessionStorage session.Storage

	// Logger
	// 日志记录器
	Logger *slog.Logger

	// Enable debug mode
	// 启用调试模式
	Debug bool

	// CORS settings
	// CORS 设置
	AllowOrigins []string
	AllowMethods []string
	AllowHeaders []string

	// Request timeout
	// 请求超时时间
	RequestTimeout time.Duration

	// Max request size (in bytes)
	// 最大请求大小 (字节)
	MaxRequestSize int64

	// VectorDBConfig 向量数据库配置（新增）
	// VectorDBConfig is the vector database configuration (new)
	VectorDBConfig *VectorDBConfig

	// EmbeddingConfig 嵌入模型配置（新增）
	// EmbeddingConfig is the embedding model configuration (new)
	EmbeddingConfig *EmbeddingConfig

	// KnowledgeAPI 知识 API 行为控制
	// KnowledgeAPI configures knowledge endpoints behaviour
	KnowledgeAPI *KnowledgeAPIOptions

	// SummaryManager 会话摘要管理器
	SummaryManager *session.SummaryManager

	// HealthPath allows overriding the health check endpoint path.
	HealthPath string
}

// VectorDBConfig 向量数据库配置
// VectorDBConfig is the vector database configuration
type VectorDBConfig struct {
	// Type 向量数据库类型（chromadb）
	// Type is the vector database type
	Type string

	// BaseURL 向量数据库 URL
	// BaseURL is the vector database URL
	BaseURL string

	// CollectionName 集合名称
	// CollectionName is the collection name
	CollectionName string

	// Database 数据库名称
	// Database is the database name
	Database string

	// Tenant 租户名称
	// Tenant is the tenant name
	Tenant string
}

// EmbeddingConfig 嵌入模型配置
// EmbeddingConfig is the embedding model configuration
type EmbeddingConfig struct {
	// Provider 提供商（openai）
	// Provider is the provider
	Provider string

	// APIKey API 密钥
	// APIKey is the API key
	APIKey string

	// Model 模型名称
	// Model is the model name
	Model string

	// BaseURL API 基础 URL
	// BaseURL is the API base URL
	BaseURL string
}

// KnowledgeAPIOptions controls knowledge endpoint behaviour
type KnowledgeAPIOptions struct {
	// DisableSearch 禁用搜索端点
	DisableSearch bool

	// DisableIngestion 禁用入库端点
	DisableIngestion bool

	// EnableHealth 暴露健康检查端点
	EnableHealth bool

	// DefaultLimit 搜索默认条数
	DefaultLimit int

	// MaxLimit 搜索最大条数
	MaxLimit int

	// SearchTimeout 搜索超时时间
	SearchTimeout time.Duration

	// IngestionTimeout 入库超时时间
	IngestionTimeout time.Duration

	// HealthTimeout 健康检查超时时间
	HealthTimeout time.Duration

	// AllowedCollections 允许的集合列表 (为空时仅默认集合)
	AllowedCollections []string

	// AllowAllCollections 是否允许任意集合
	AllowAllCollections bool

	// AllowedSourceSchemes 允许的来源 URL scheme（为空表示不限制）
	AllowedSourceSchemes []string
}

func normalizeKnowledgeOptions(opts *KnowledgeAPIOptions) {
	if opts == nil {
		return
	}

	if opts.DefaultLimit <= 0 {
		opts.DefaultLimit = 10
	}
	if opts.MaxLimit <= 0 {
		opts.MaxLimit = 100
	}
	if opts.MaxLimit < opts.DefaultLimit {
		opts.MaxLimit = opts.DefaultLimit
	}
	if opts.SearchTimeout <= 0 {
		opts.SearchTimeout = 30 * time.Second
	}
	if opts.IngestionTimeout <= 0 {
		opts.IngestionTimeout = 60 * time.Second
	}
	if opts.HealthTimeout <= 0 {
		opts.HealthTimeout = 5 * time.Second
	}
	if len(opts.AllowedSourceSchemes) == 0 {
		opts.AllowedSourceSchemes = []string{"http", "https", "mcp"}
	} else {
		seen := make(map[string]struct{}, len(opts.AllowedSourceSchemes))
		normalized := make([]string, 0, len(opts.AllowedSourceSchemes))
		for _, scheme := range opts.AllowedSourceSchemes {
			if scheme == "" {
				continue
			}
			lower := strings.ToLower(scheme)
			if _, exists := seen[lower]; exists {
				continue
			}
			seen[lower] = struct{}{}
			normalized = append(normalized, lower)
		}
		opts.AllowedSourceSchemes = normalized
	}
}

// NewServer creates a new AgentOS server
func NewServer(config *Config) (*Server, error) {
	if config == nil {
		config = &Config{}
	}

	// Set defaults
	if config.Address == "" {
		config.Address = ":8080"
	}

	if config.SessionStorage == nil {
		config.SessionStorage = session.NewMemoryStorage()
	}

	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	if config.SummaryManager == nil {
		config.SummaryManager = session.NewSummaryManager(session.WithSummaryLogger(config.Logger))
	}

	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}

	if config.MaxRequestSize == 0 {
		config.MaxRequestSize = 10 * 1024 * 1024 // 10MB
	}

	if config.KnowledgeAPI == nil {
		config.KnowledgeAPI = &KnowledgeAPIOptions{}
	}
	normalizeKnowledgeOptions(config.KnowledgeAPI)
	if strings.TrimSpace(config.HealthPath) == "" {
		config.HealthPath = "/health"
	}

	// Set Gin mode
	if config.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(loggerMiddleware(config.Logger))
	router.Use(corsMiddleware(config))
	router.Use(timeoutMiddleware(config.RequestTimeout))

	server := &Server{
		router:         router,
		config:         config,
		sessionStorage: config.SessionStorage,
		agentRegistry:  NewAgentRegistry(),
		teamRegistry:   NewTeamRegistry(),
		logger:         config.Logger,
		summaryManager: config.SummaryManager,
		instantiatedAt: time.Now().UTC(),
	}

	// 初始化知识库服务（如果配置了）
	// Initialize knowledge service (if configured)
	if config.VectorDBConfig != nil && config.EmbeddingConfig != nil {
		knowledgeSvc, err := initKnowledgeService(config, config.Logger)
		if err != nil {
			config.Logger.Warn("failed to initialize knowledge service", "error", err)
		} else {
			server.knowledgeService = knowledgeSvc
			config.Logger.Info("knowledge service initialized")
		}
	}

	// Register routes
	server.registerRoutes()

	return server, nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info("starting AgentOS server", "address", s.config.Address)

	s.httpServer = &http.Server{
		Addr:         s.config.Address,
		Handler:      s.router,
		ReadTimeout:  s.config.RequestTimeout,
		WriteTimeout: s.config.RequestTimeout,
	}

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down AgentOS server")

	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown server: %w", err)
		}
	}

	// Close storage
	if s.sessionStorage != nil {
		if err := s.sessionStorage.Close(); err != nil {
			s.logger.Warn("failed to close session storage", "error", err)
		}
	}

	return nil
}

// RegisterAgent registers an agent with the server
func (s *Server) RegisterAgent(agentID string, ag *agent.Agent) error {
	return s.agentRegistry.Register(agentID, ag)
}

// GetAgentRegistry returns the agent registry
func (s *Server) GetAgentRegistry() *AgentRegistry {
	return s.agentRegistry
}

// RegisterTeam registers a team with the server.
func (s *Server) RegisterTeam(teamID string, tm *team.Team) error {
	if s.teamRegistry == nil {
		s.teamRegistry = NewTeamRegistry()
	}
	return s.teamRegistry.Register(teamID, tm)
}

// GetTeamRegistry returns the team registry.
func (s *Server) GetTeamRegistry() *TeamRegistry {
	return s.teamRegistry
}

// registerRoutes registers all API routes
// registerRoutes 注册所有 API 路由
func (s *Server) registerRoutes() {
	// Create base group with prefix (if specified)
	// 使用前缀创建基础路由组 (如果指定了前缀)
	var baseGroup *gin.RouterGroup
	if s.config.Prefix != "" {
		baseGroup = s.router.Group(s.config.Prefix)
	} else {
		baseGroup = &s.router.RouterGroup
	}

	// Health check (always at root level, customizable via GetHealthRouter)
	if group := s.GetHealthRouter(s.config.HealthPath); group != nil {
		group.GET("", s.handleHealth)
	}
	s.RegisterDocs(nil)

	// API v1 under the prefix
	// 前缀下的 API v1
	v1 := baseGroup.Group("/api/v1")
	{
		// Session endpoints
		// Session 端点
		sessions := v1.Group("/sessions")
		{
			sessions.POST("", s.handleCreateSession)
			sessions.GET("/:id", s.handleGetSession)
			sessions.PUT("/:id", s.handleUpdateSession)
			sessions.DELETE("/:id", s.handleDeleteSession)
			sessions.GET("/:id/summary", s.handleGetSessionSummary)
			sessions.POST("/:id/summary", s.handlePostSessionSummary)
			sessions.POST("/:id/reuse", s.handleReuseSession)
			sessions.GET("/:id/history", s.handleSessionHistory)
			sessions.GET("", s.handleListSessions)
		}

		// Agent endpoints
		// Agent 端点
		agents := v1.Group("/agents")
		{
			agents.GET("", s.handleListAgents)
			agents.POST("/:id/run", s.handleAgentRun)
			agents.POST("/:id/run/stream", s.handleAgentRunStream) // P1: SSE 流式输出
		}

		// Team endpoints
		teams := v1.Group("/teams")
		{
			teams.GET("/:id/tools", s.handleTeamTools)
		}

		// Knowledge endpoints
		if s.knowledgeService != nil {
			knowledge := v1.Group("/knowledge")
			knowledge.GET("/config", s.handleKnowledgeConfig)
			if s.knowledgeService.config.EnableSearch {
				knowledge.POST("/search", s.handleKnowledgeSearch)
			}
			if s.knowledgeService.config.EnableIngestion {
				knowledge.POST("/content", s.handleAddContent)
				knowledge.POST("/upload", s.handleAddContent)
			}
			if s.knowledgeService.config.EnableHealth {
				knowledge.GET("/health", s.handleKnowledgeHealth)
			}
		}
	}
}

// GetHealthRouter returns a router group for the given health path so callers
// can attach additional handlers (e.g. custom health checks) while the server
// still maintains its default `/health` endpoint.
func (s *Server) GetHealthRouter(path string) *gin.RouterGroup {
	if s.router == nil {
		return nil
	}
	if path == "" {
		path = "/health"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return s.router.Group(path)
}

// RegisterDocs mounts the OpenAPI specification and Swagger UI handlers onto
// the provided router. Passing nil registers the routes on the server's primary
// router, allowing callers to re-register after custom wiring or resync steps.
func (s *Server) RegisterDocs(router *gin.Engine) {
	if router == nil {
		router = s.router
		if s.docsMounted {
			return
		}
		s.docsMounted = true
	}
	if router == nil {
		return
	}
	router.GET("/openapi.yaml", s.handleOpenAPISpec)
	router.GET("/docs", s.handleDocs)
}

// Health check handler
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":          "ok",
		"service":         "agentos",
		"instantiated_at": s.instantiatedAt.Format(time.RFC3339),
		"uptime_seconds":  time.Since(s.instantiatedAt).Seconds(),
	})
}

func (s *Server) handleOpenAPISpec(c *gin.Context) {
	c.Data(http.StatusOK, "application/yaml", openAPISpec)
}

func (s *Server) handleDocs(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", docsHTML)
}

// Middleware: Logger
func loggerMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		logger.Info("request",
			"method", method,
			"path", path,
			"status", status,
			"duration", duration.String(),
			"ip", c.ClientIP(),
		)
	}
}

// Middleware: CORS
func corsMiddleware(config *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(config.AllowOrigins) > 0 {
			c.Writer.Header().Set("Access-Control-Allow-Origin", config.AllowOrigins[0])
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		if len(config.AllowMethods) > 0 {
			methods := ""
			for i, m := range config.AllowMethods {
				if i > 0 {
					methods += ", "
				}
				methods += m
			}
			c.Writer.Header().Set("Access-Control-Allow-Methods", methods)
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		}

		if len(config.AllowHeaders) > 0 {
			headers := ""
			for i, h := range config.AllowHeaders {
				if i > 0 {
					headers += ", "
				}
				headers += h
			}
			c.Writer.Header().Set("Access-Control-Allow-Headers", headers)
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Middleware: Timeout
func timeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// initKnowledgeService 初始化知识库服务
// initKnowledgeService initializes the knowledge service
func initKnowledgeService(config *Config, logger *slog.Logger) (*KnowledgeService, error) {
	// 初始化嵌入函数
	// Initialize embedding function
	embConfig := openai.Config{
		APIKey:  config.EmbeddingConfig.APIKey,
		Model:   config.EmbeddingConfig.Model,
		BaseURL: config.EmbeddingConfig.BaseURL,
	}
	embFunc, err := openai.New(embConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding function: %w", err)
	}

	// 初始化向量数据库
	// Initialize vector database
	var vdb vectordb.VectorDB
	switch config.VectorDBConfig.Type {
	case "chromadb":
		chromaConfig := chromadb.Config{
			BaseURL:           config.VectorDBConfig.BaseURL,
			CollectionName:    config.VectorDBConfig.CollectionName,
			Database:          config.VectorDBConfig.Database,
			Tenant:            config.VectorDBConfig.Tenant,
			EmbeddingFunction: embFunc,
		}
		chromaDB, err := chromadb.New(chromaConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create chromadb: %w", err)
		}

		// 创建或连接到集合
		// Create or connect to collection
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := chromaDB.CreateCollection(ctx, chromaConfig.CollectionName, nil); err != nil {
			// 如果集合已存在，忽略错误
			// Ignore error if collection already exists
			logger.Debug("collection may already exist", "collection", chromaConfig.CollectionName, "error", err)
		}

		vdb = chromaDB
	default:
		return nil, fmt.Errorf("unsupported vector db type: %s", config.VectorDBConfig.Type)
	}

	opts := config.KnowledgeAPI

	svcConfig := KnowledgeServiceConfig{
		DefaultLimit:         opts.DefaultLimit,
		MaxLimit:             opts.MaxLimit,
		DefaultChunkerType:   "character",
		DefaultChunkSize:     1000,
		DefaultOverlap:       100,
		EmbeddingProvider:    config.EmbeddingConfig.Provider,
		EmbeddingModel:       config.EmbeddingConfig.Model,
		EmbeddingDimensions:  1536, // OpenAI text-embedding-3-small 默认维度
		EnableSearch:         !opts.DisableSearch,
		EnableIngestion:      !opts.DisableIngestion,
		EnableHealth:         opts.EnableHealth,
		SearchTimeout:        opts.SearchTimeout,
		IngestionTimeout:     opts.IngestionTimeout,
		HealthTimeout:        opts.HealthTimeout,
		DefaultCollection:    config.VectorDBConfig.CollectionName,
		AllowedCollections:   opts.AllowedCollections,
		AllowAllCollections:  opts.AllowAllCollections,
		AllowedSourceSchemes: opts.AllowedSourceSchemes,
	}

	return NewKnowledgeService(vdb, embFunc, svcConfig), nil
}

// Resync re-registers documentation routes ensuring `/docs` remains mounted
// after custom wiring or hot reload flows.
func (s *Server) Resync() {
	s.RegisterDocs(nil)
}
