package agentos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rexleimo/agno-go/pkg/hno/vectordb"
)

// KnowledgeService 封装知识库操作
// KnowledgeService encapsulates knowledge operations
type KnowledgeService struct {
	vectorDB      vectordb.VectorDB
	embeddingFunc vectordb.EmbeddingFunction
	config        KnowledgeServiceConfig
	collections   map[string]struct{}
	schemes       map[string]struct{}
}

// KnowledgeServiceConfig 知识库服务配置
// KnowledgeServiceConfig is the knowledge service configuration
type KnowledgeServiceConfig struct {
	// DefaultLimit 默认返回结果数量
	// DefaultLimit is the default number of results to return
	DefaultLimit int

	// MaxLimit 最大返回结果数量
	// MaxLimit is the maximum number of results to return
	MaxLimit int

	// DefaultChunkerType 默认分块器类型
	// DefaultChunkerType is the default chunker type
	DefaultChunkerType string

	// DefaultChunkSize 默认块大小
	// DefaultChunkSize is the default chunk size
	DefaultChunkSize int

	// DefaultOverlap 默认重叠大小
	// DefaultOverlap is the default overlap size
	DefaultOverlap int

	// EmbeddingProvider 嵌入模型提供商
	// EmbeddingProvider is the embedding model provider
	EmbeddingProvider string

	// EmbeddingModel 嵌入模型名称
	// EmbeddingModel is the embedding model name
	EmbeddingModel string

	// EmbeddingDimensions 嵌入维度
	// EmbeddingDimensions is the embedding dimensions
	EmbeddingDimensions int

	// EnableSearch 是否启用搜索
	EnableSearch bool

	// EnableIngestion 是否启用知识入库
	EnableIngestion bool

	// EnableHealth 是否启用健康检查
	EnableHealth bool

	// SearchTimeout 搜索超时时间
	SearchTimeout time.Duration

	// IngestionTimeout 入库超时时间
	IngestionTimeout time.Duration

	// HealthTimeout 健康检查超时时间
	HealthTimeout time.Duration

	// DefaultCollection 默认集合名称
	DefaultCollection string

	// AllowedCollections 允许访问的集合列表
	AllowedCollections []string

	// AllowAllCollections 是否允许任意集合
	AllowAllCollections bool

	// AllowedSourceSchemes 允许的来源 URL scheme
	AllowedSourceSchemes []string
}

// NewKnowledgeService 创建知识库服务
// NewKnowledgeService creates a new knowledge service
func NewKnowledgeService(vdb vectordb.VectorDB, embFunc vectordb.EmbeddingFunction, config KnowledgeServiceConfig) *KnowledgeService {
	// 设置默认值
	// Set default values
	if config.DefaultLimit <= 0 {
		config.DefaultLimit = 10
	}
	if config.MaxLimit <= 0 {
		config.MaxLimit = 100
	}
	if config.DefaultChunkerType == "" {
		config.DefaultChunkerType = "character"
	}
	if config.DefaultChunkSize <= 0 {
		config.DefaultChunkSize = 1000
	}
	if config.DefaultOverlap < 0 {
		config.DefaultOverlap = 100
	}
	if config.EmbeddingProvider == "" {
		config.EmbeddingProvider = "openai"
	}
	if config.EmbeddingModel == "" {
		config.EmbeddingModel = "text-embedding-3-small"
	}
	if config.EmbeddingDimensions <= 0 {
		config.EmbeddingDimensions = 1536
	}
	if config.SearchTimeout <= 0 {
		config.SearchTimeout = 30 * time.Second
	}
	if config.IngestionTimeout <= 0 {
		config.IngestionTimeout = 60 * time.Second
	}
	if config.HealthTimeout <= 0 {
		config.HealthTimeout = 5 * time.Second
	}
	if !config.EnableSearch && !config.EnableIngestion && !config.EnableHealth {
		config.EnableSearch = true
		config.EnableIngestion = true
	}
	if len(config.AllowedSourceSchemes) == 0 {
		config.AllowedSourceSchemes = []string{"http", "https", "mcp"}
	}

	collectionSet := make(map[string]struct{})
	if config.DefaultCollection != "" {
		collectionSet[strings.ToLower(config.DefaultCollection)] = struct{}{}
	}
	for _, name := range config.AllowedCollections {
		if name == "" {
			continue
		}
		collectionSet[strings.ToLower(name)] = struct{}{}
	}

	schemeSet := make(map[string]struct{})
	for _, scheme := range config.AllowedSourceSchemes {
		if scheme == "" {
			continue
		}
		schemeSet[strings.ToLower(scheme)] = struct{}{}
	}

	return &KnowledgeService{
		vectorDB:      vdb,
		embeddingFunc: embFunc,
		config:        config,
		collections:   collectionSet,
		schemes:       schemeSet,
	}
}

func (s *KnowledgeService) searchTimeout() time.Duration {
	return s.config.SearchTimeout
}

func (s *KnowledgeService) ingestionTimeout() time.Duration {
	return s.config.IngestionTimeout
}

func (s *KnowledgeService) healthTimeout() time.Duration {
	return s.config.HealthTimeout
}

func (s *KnowledgeService) isCollectionAllowed(name string) bool {
	if name == "" || s.config.AllowAllCollections {
		return true
	}
	_, ok := s.collections[strings.ToLower(name)]
	return ok
}

func (s *KnowledgeService) searchFilters(collectionName string, filters map[string]interface{}) (map[string]interface{}, error) {
	if collectionName == "" {
		return filters, nil
	}

	merged := make(map[string]interface{}, len(filters)+1)
	for key, value := range filters {
		merged[key] = value
	}
	if existing, ok := merged["collection_name"]; ok && !strings.EqualFold(fmt.Sprint(existing), collectionName) {
		return nil, fmt.Errorf("filters.collection_name conflicts with collection_name")
	}
	merged["collection_name"] = collectionName
	return merged, nil
}

func (s *KnowledgeService) validateSourceMetadata(metadata map[string]interface{}) error {
	if metadata == nil || len(s.schemes) == 0 {
		return nil
	}
	keys := []string{"source_url", "sourceUrl", "source_uri", "sourceUri", "source"}
	for _, key := range keys {
		raw, ok := metadata[key]
		if !ok {
			continue
		}
		str, ok := raw.(string)
		if !ok {
			continue
		}
		str = strings.TrimSpace(str)
		if str == "" {
			continue
		}
		if key == "source" && !strings.Contains(str, "://") {
			continue
		}
		if err := s.validateSourceURL(str); err != nil {
			return err
		}
	}
	return nil
}

func (s *KnowledgeService) validateSourceURL(raw string) error {
	if len(s.schemes) == 0 {
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid source_url: %w", err)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme == "" {
		return fmt.Errorf("invalid source_url: missing scheme")
	}
	if _, ok := s.schemes[scheme]; !ok {
		return fmt.Errorf("source_url scheme %q is not allowed", scheme)
	}
	return nil
}

// handleKnowledgeSearch 处理知识库搜索请求
// handleKnowledgeSearch handles knowledge search requests
func (s *Server) handleKnowledgeSearch(c *gin.Context) {
	knowledgeSvc, ok := s.requireKnowledgeService(c)
	if !ok {
		return
	}
	if !knowledgeSvc.config.EnableSearch {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "search_disabled",
			"message": "knowledge search endpoint is disabled",
		})
		return
	}

	var req VectorSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	if req.CollectionName != "" && !knowledgeSvc.isCollectionAllowed(req.CollectionName) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_collection",
			"message": fmt.Sprintf("collection %q is not allowed", req.CollectionName),
		})
		return
	}

	filters, err := knowledgeSvc.searchFilters(req.CollectionName, req.Filters)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	if req.Limit <= 0 {
		req.Limit = knowledgeSvc.config.DefaultLimit
	}
	if req.Limit > knowledgeSvc.config.MaxLimit {
		req.Limit = knowledgeSvc.config.MaxLimit
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), knowledgeSvc.searchTimeout())
	defer cancel()

	results, err := knowledgeSvc.vectorDB.Query(ctx, req.Query, req.Limit+req.Offset, filters)
	if err != nil {
		s.logger.Error("knowledge search failed", "error", err, "query", req.Query)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "search_failed",
			"message": "knowledge search failed",
		})
		return
	}

	totalCount := len(results)
	paginatedResults := results
	if req.Offset < len(results) {
		paginatedResults = results[req.Offset:]
		if len(paginatedResults) > req.Limit {
			paginatedResults = paginatedResults[:req.Limit]
		}
	} else {
		paginatedResults = []vectordb.SearchResult{}
	}

	searchResults := make([]VectorSearchResult, len(paginatedResults))
	for i, r := range paginatedResults {
		searchResults[i] = VectorSearchResult{
			ID:       r.ID,
			Content:  r.Content,
			Metadata: r.Metadata,
			Score:    r.Score,
			Distance: r.Distance,
		}
	}

	pageSize := req.Limit
	page := 1
	if pageSize > 0 {
		page = (req.Offset / pageSize) + 1
	}
	totalPages := 0
	if pageSize > 0 {
		totalPages = (totalCount + pageSize - 1) / pageSize
		if totalPages == 0 && totalCount > 0 {
			totalPages = 1
		}
	}
	hasMore := req.Offset+len(paginatedResults) < totalCount
	nextOffset := req.Offset
	if hasMore {
		nextOffset = req.Offset + len(paginatedResults)
	}

	response := VectorSearchResponse{
		Results: searchResults,
		Meta: PaginationMeta{
			TotalCount: totalCount,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
			Offset:     req.Offset,
			HasMore:    hasMore,
		},
	}
	if hasMore {
		response.Meta.NextOffset = nextOffset
	}

	c.JSON(http.StatusOK, response)
}

// handleKnowledgeConfig 处理知识库配置查询
// handleKnowledgeConfig handles knowledge configuration query
func (s *Server) handleKnowledgeConfig(c *gin.Context) {
	knowledgeSvc, ok := s.requireKnowledgeService(c)
	if !ok {
		return
	}

	// 构建配置响应
	// Build configuration response
	response := KnowledgeConfigResponse{
		AvailableChunkers: []ChunkerInfo{
			{
				Name:             "character",
				Description:      "按字符数量分块，适合通用文本",
				DefaultChunkSize: 1000,
				DefaultOverlap:   100,
				Metadata: map[string]interface{}{
					"chunk_size":    1000,
					"chunk_overlap": 100,
				},
			},
			{
				Name:             "sentence",
				Description:      "按句子分块，适合对话和文档",
				DefaultChunkSize: 1000,
				DefaultOverlap:   0,
				Metadata: map[string]interface{}{
					"chunk_size":    1000,
					"chunk_overlap": 0,
				},
			},
			{
				Name:             "paragraph",
				Description:      "按段落分块，适合长文档",
				DefaultChunkSize: 2000,
				DefaultOverlap:   0,
				Metadata: map[string]interface{}{
					"chunk_size":    2000,
					"chunk_overlap": 0,
				},
			},
		},
		AvailableVectorDBs: []string{"chromadb"},
		DefaultChunker:     knowledgeSvc.config.DefaultChunkerType,
		DefaultVectorDB:    "chromadb",
		EmbeddingModel: EmbeddingModelInfo{
			Provider:   knowledgeSvc.config.EmbeddingProvider,
			Model:      knowledgeSvc.config.EmbeddingModel,
			Dimensions: knowledgeSvc.config.EmbeddingDimensions,
		},
		DefaultCollection: knowledgeSvc.config.DefaultCollection,
		Features: KnowledgeFeatures{
			SearchEnabled:    knowledgeSvc.config.EnableSearch,
			IngestionEnabled: knowledgeSvc.config.EnableIngestion,
			HealthEnabled:    knowledgeSvc.config.EnableHealth,
		},
		Limits: KnowledgeLimits{
			DefaultLimit: knowledgeSvc.config.DefaultLimit,
			MaxLimit:     knowledgeSvc.config.MaxLimit,
		},
		AllowedSourceSchemes: knowledgeSvc.config.AllowedSourceSchemes,
	}

	if knowledgeSvc.config.AllowAllCollections {
		response.AllowedCollections = []string{"*"}
	} else {
		unique := make(map[string]struct{})
		list := []string{}
		if knowledgeSvc.config.DefaultCollection != "" {
			unique[strings.ToLower(knowledgeSvc.config.DefaultCollection)] = struct{}{}
			list = append(list, knowledgeSvc.config.DefaultCollection)
		}
		for _, name := range knowledgeSvc.config.AllowedCollections {
			if name == "" {
				continue
			}
			key := strings.ToLower(name)
			if _, exists := unique[key]; exists {
				continue
			}
			unique[key] = struct{}{}
			list = append(list, name)
		}
		if len(list) > 0 {
			response.AllowedCollections = list
		}
	}

	c.JSON(http.StatusOK, response)
}

// handleKnowledgeHealth 返回知识库健康状态
func (s *Server) handleKnowledgeHealth(c *gin.Context) {
	knowledgeSvc, ok := s.requireKnowledgeService(c)
	if !ok {
		return
	}
	if !knowledgeSvc.config.EnableHealth {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "health_disabled",
			"message": "knowledge health endpoint is disabled",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), knowledgeSvc.healthTimeout())
	defer cancel()

	count, err := knowledgeSvc.vectorDB.Count(ctx)
	status := "ok"
	response := gin.H{
		"status":        status,
		"collection":    knowledgeSvc.config.DefaultCollection,
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
		"documents":     count,
		"search":        knowledgeSvc.config.EnableSearch,
		"ingestion":     knowledgeSvc.config.EnableIngestion,
		"health":        knowledgeSvc.config.EnableHealth,
		"max_limit":     knowledgeSvc.config.MaxLimit,
		"default_limit": knowledgeSvc.config.DefaultLimit,
	}

	statusCode := http.StatusOK
	if err != nil {
		response["status"] = "degraded"
		response["error"] = "knowledge store unavailable"
		delete(response, "documents")
		s.logger.Warn("knowledge health check failed", "error", err)
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// handleAddContent 处理添加内容到知识库的请求
// handleAddContent handles adding content to the knowledge base
func (s *Server) handleAddContent(c *gin.Context) {
	knowledgeSvc, ok := s.requireKnowledgeService(c)
	if !ok {
		return
	}
	if !knowledgeSvc.config.EnableIngestion {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "ingestion_disabled",
			"message": "knowledge ingestion endpoint is disabled",
		})
		return
	}

	var req AddContentRequest

	// 检查 Content-Type
	// Check Content-Type
	contentType := c.ContentType()

	switch {
	case strings.HasPrefix(contentType, "multipart/form-data"):
		if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
			s.logger.Error("failed to parse multipart request", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_request",
				"message": "failed to parse multipart form",
			})
			return
		}
		fileHeader, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_request",
				"message": "file field is required",
			})
			return
		}
		file, err := fileHeader.Open()
		if err != nil {
			s.logger.Error("failed to open uploaded file", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_request",
				"message": "failed to read uploaded file",
			})
			return
		}
		defer file.Close()
		buf := bytes.NewBuffer(nil)
		if _, err := io.Copy(buf, file); err != nil {
			s.logger.Error("failed to copy uploaded file", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_request",
				"message": "failed to read uploaded file",
			})
			return
		}
		req.Content = buf.String()
		req.CollectionName = c.PostForm("collection_name")
		req.ChunkerType = c.DefaultPostForm("chunker_type", knowledgeSvc.config.DefaultChunkerType)
		req.ChunkSize = parsePositiveInt(c.PostForm("chunk_size"), knowledgeSvc.config.DefaultChunkSize)
		req.ChunkOverlap = parsePositiveInt(c.PostForm("chunk_overlap"), knowledgeSvc.config.DefaultOverlap)
		if metadata := c.PostForm("metadata"); metadata != "" {
			if err := json.Unmarshal([]byte(metadata), &req.Metadata); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "invalid_metadata",
					"message": "metadata must be JSON",
				})
				return
			}
		}

	case contentType == "text/plain":
		// 读取纯文本内容
		// Read plain text content
		body, err := c.GetRawData()
		if err != nil {
			s.logger.Error("failed to read request body", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_request",
				"message": "failed to read request body",
			})
			return
		}

		req.Content = string(body)

		// 从查询参数获取可选配置
		// Get optional config from query params
		req.ChunkerType = c.DefaultQuery("chunker_type", "character")
		req.CollectionName = c.Query("collection_name")
		if sizeStr := c.Query("chunk_size"); sizeStr != "" {
			if parsed, err := strconv.Atoi(sizeStr); err == nil {
				req.ChunkSize = parsed
			}
		}
		if overlapStr := c.Query("chunk_overlap"); overlapStr != "" {
			if parsed, err := strconv.Atoi(overlapStr); err == nil {
				req.ChunkOverlap = parsed
			}
		}

	case contentType == "application/json":
		// 解析 JSON 请求
		// Parse JSON request
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_request",
				"message": err.Error(),
			})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_content_type",
			"message": "content type must be text/plain or application/json",
		})
		return
	}

	// 验证内容
	// Validate content
	if req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "content is required",
		})
		return
	}

	if req.CollectionName != "" && !knowledgeSvc.isCollectionAllowed(req.CollectionName) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_collection",
			"message": fmt.Sprintf("collection %q is not allowed", req.CollectionName),
		})
		return
	}
	if err := knowledgeSvc.validateSourceMetadata(req.Metadata); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_metadata",
			"message": err.Error(),
		})
		return
	}

	// 设置默认值
	// Set defaults
	if req.ChunkerType == "" {
		req.ChunkerType = knowledgeSvc.config.DefaultChunkerType
	}
	if req.ChunkSize <= 0 {
		req.ChunkSize = knowledgeSvc.config.DefaultChunkSize
	}
	if req.ChunkOverlap < 0 {
		req.ChunkOverlap = knowledgeSvc.config.DefaultOverlap
	}

	// 创建超时上下文
	// Create timeout context
	ctx, cancel := context.WithTimeout(c.Request.Context(), knowledgeSvc.ingestionTimeout())
	defer cancel()

	// 执行知识入库
	// Perform knowledge ingestion
	result, err := s.ingestContent(ctx, knowledgeSvc, &req)
	if err != nil {
		s.logger.Error("failed to ingest content",
			"error", err,
			"chunker_type", req.ChunkerType,
			"content_length", len(req.Content),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "ingestion_failed",
			"message": "knowledge ingestion failed",
		})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// AddContentResponse 表示添加内容的响应
// AddContentResponse represents the response for adding content
type AddContentResponse struct {
	// DocumentIDs 添加的文档 ID 列表
	// DocumentIDs is the list of added document IDs
	DocumentIDs []string `json:"document_ids"`

	// ChunkCount 生成的块数量
	// ChunkCount is the number of chunks generated
	ChunkCount int `json:"chunk_count"`

	// ChunkerType 使用的分块器类型
	// ChunkerType is the chunker type used
	ChunkerType string `json:"chunker_type"`

	// Collection 使用的集合名称
	// Collection is the collection that stored the documents
	Collection string `json:"collection,omitempty"`

	// Message 处理消息
	// Message is the processing message
	Message string `json:"message"`
}

// ingestContent 执行内容入库流程：分块 → 嵌入 → 存储
// ingestContent performs content ingestion: chunking → embedding → storage
func (s *Server) ingestContent(ctx context.Context, knowledgeSvc *KnowledgeService, req *AddContentRequest) (*AddContentResponse, error) {
	if req.CollectionName != "" && !knowledgeSvc.isCollectionAllowed(req.CollectionName) {
		return nil, fmt.Errorf("collection %q is not allowed", req.CollectionName)
	}

	collectionName := knowledgeSvc.config.DefaultCollection
	if req.CollectionName != "" {
		collectionName = req.CollectionName
	}
	// 注意：这里简化实现，实际应从 knowledge 包导入 Chunker
	// Note: Simplified implementation, should import Chunker from knowledge package
	content := req.Content
	chunks := make([]string, 0)
	if len(content) == 0 {
		chunks = []string{}
	} else {
		chunkSize := req.ChunkSize
		if chunkSize <= 0 || chunkSize > len(content) {
			chunkSize = len(content)
		}
		overlap := req.ChunkOverlap
		if overlap < 0 {
			overlap = 0
		}
		if overlap >= chunkSize {
			overlap = 0
		}
		for start := 0; start < len(content); {
			end := start + chunkSize
			if end > len(content) {
				end = len(content)
			}
			chunks = append(chunks, content[start:end])
			if end == len(content) {
				break
			}
			start = end - overlap
			if start < 0 {
				start = 0
			}
		}
	}

	// 准备文档
	// Prepare documents
	documents := make([]vectordb.Document, len(chunks))
	for i, chunk := range chunks {
		docID := fmt.Sprintf("doc_%d_%d", time.Now().Unix(), i)

		// 合并元数据
		// Merge metadata
		metadata := make(map[string]interface{})
		if req.Metadata != nil {
			for k, v := range req.Metadata {
				metadata[k] = v
			}
		}
		if collectionName != "" {
			metadata["collection_name"] = collectionName
		}
		metadata["chunk_index"] = i
		metadata["chunk_count"] = len(chunks)
		metadata["chunker_type"] = req.ChunkerType
		metadata["chunk_size"] = req.ChunkSize
		metadata["chunk_overlap"] = req.ChunkOverlap
		metadata["ingested_at"] = time.Now().Format(time.RFC3339)

		documents[i] = vectordb.Document{
			ID:       docID,
			Content:  chunk,
			Metadata: metadata,
		}
	}

	// 添加到向量数据库
	// Add to vector database
	if err := knowledgeSvc.vectorDB.Add(ctx, documents); err != nil {
		return nil, fmt.Errorf("failed to add documents to vector db: %w", err)
	}

	// 提取文档 ID
	// Extract document IDs
	docIDs := make([]string, len(documents))
	for i, doc := range documents {
		docIDs[i] = doc.ID
	}

	return &AddContentResponse{
		DocumentIDs: docIDs,
		ChunkCount:  len(chunks),
		ChunkerType: req.ChunkerType,
		Message:     fmt.Sprintf("successfully ingested %d chunks", len(chunks)),
		Collection:  collectionName,
	}, nil
}

func parsePositiveInt(raw string, defaultVal int) int {
	if raw == "" {
		return defaultVal
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return defaultVal
	}
	return value
}

func (s *Server) requireKnowledgeService(c *gin.Context) (*KnowledgeService, bool) {
	knowledgeSvc := s.getKnowledgeService()
	if knowledgeSvc != nil {
		return knowledgeSvc, true
	}

	if s.knowledgeConfigured() {
		_ = s.initializeKnowledgeService()
		if knowledgeSvc = s.getKnowledgeService(); knowledgeSvc != nil {
			return knowledgeSvc, true
		}
	}

	message := "knowledge service not configured"
	if s.getKnowledgeInitErr() != nil {
		message = "knowledge service unavailable"
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error":   "service_unavailable",
		"message": message,
	})
	return nil, false
}

// getKnowledgeService gets the current knowledge service instance.
func (s *Server) getKnowledgeService() *KnowledgeService {
	s.knowledgeMu.RLock()
	defer s.knowledgeMu.RUnlock()
	return s.knowledgeService
}
