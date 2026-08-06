// Package chromadb implements the VectorDB interface using ChromaDB via the
// v2 API client (chroma-go v0.4.x). The v2 client is pure Go (no cgo), so
// this package builds on Windows as well.
//
// chromadb 包通过 v2 API 客户端（chroma-go v0.4.x）实现 VectorDB 接口。
// v2 客户端为纯 Go（无 cgo），因此本包在 Windows 上也能构建。
package chromadb

import (
	"context"
	"fmt"
	"reflect"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"

	"github.com/rexleimo/agno-go/pkg/hno/vectordb"
)

// ChromaDB implements the VectorDB interface using ChromaDB
type ChromaDB struct {
	client         chroma.Client
	collection     chroma.Collection
	collectionName string
	embeddingFunc  vectordb.EmbeddingFunction
	distanceFunc   embeddings.DistanceMetric
}

// Config holds ChromaDB configuration
type Config struct {
	// BaseURL is the ChromaDB server URL (default: http://localhost:8000)
	BaseURL string

	// CollectionName is the name of the collection to use
	CollectionName string

	// Database name (for multi-tenant setups)
	Database string

	// Tenant name (for multi-tenant setups)
	Tenant string

	// CloudAPIKey for ChromaDB Cloud (optional)
	CloudAPIKey string

	// EmbeddingFunction to use for generating embeddings
	// If nil, documents must already have embeddings
	EmbeddingFunction vectordb.EmbeddingFunction

	// DistanceFunction to use for similarity search
	DistanceFunction vectordb.DistanceFunction

	// Metadata for the collection
	Metadata map[string]interface{}
}

// New creates a new ChromaDB vector database client
func New(config Config) (*ChromaDB, error) {
	if config.CollectionName == "" {
		return nil, fmt.Errorf("collection name is required")
	}

	// Set defaults
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:8000"
	}
	if config.Database == "" {
		config.Database = "default_database"
	}
	if config.Tenant == "" {
		config.Tenant = "default_tenant"
	}
	distanceMetric := embeddings.L2
	if config.DistanceFunction == "" {
		config.DistanceFunction = vectordb.L2
	}
	switch config.DistanceFunction {
	case vectordb.Cosine:
		distanceMetric = embeddings.COSINE
	case vectordb.InnerProduct:
		distanceMetric = embeddings.IP
	default:
		distanceMetric = embeddings.L2
	}

	// Keep every collection operation in the configured tenant/database rather
	// than silently falling back to Chroma's defaults.
	clientOpts := []chroma.ClientOption{
		chroma.WithBaseURL(config.BaseURL),
		chroma.WithDatabaseAndTenant(config.Database, config.Tenant),
	}
	if config.CloudAPIKey != "" {
		clientOpts = append(clientOpts, chroma.WithAuth(
			chroma.NewTokenAuthCredentialsProvider(config.CloudAPIKey, chroma.XChromaTokenHeader),
		))
	}

	// Create ChromaDB client
	client, err := chroma.NewHTTPClient(clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChromaDB client: %w", err)
	}

	db := &ChromaDB{
		client:         client,
		collectionName: config.CollectionName,
		embeddingFunc:  config.EmbeddingFunction,
		distanceFunc:   distanceMetric,
	}

	return db, nil
}

// CreateCollection creates a new collection or connects to existing one
func (c *ChromaDB) CreateCollection(ctx context.Context, name string, metadata map[string]interface{}) error {
	if name != "" {
		c.collectionName = name
	}

	// Convert metadata to ChromaDB format
	var chromaMetadata chroma.CollectionMetadata
	if metadata != nil {
		clean := make(map[string]interface{})
		for k, v := range metadata {
			if k != "distance_function" {
				clean[k] = v
			}
		}
		if len(clean) > 0 {
			chromaMetadata = chroma.NewMetadataFromMap(clean)
		}
	}

	// Get or create collection. Pass the configured embedding function so the
	// client never falls back to its default local ONNX embedding function
	// (which bootstraps an onnxruntime download and can fail with GitHub API
	// rate limits).
	createOpts := []chroma.CreateCollectionOption{
		chroma.WithHNSWSpaceCreate(c.distanceFunc),
		chroma.WithCollectionMetadataCreate(chromaMetadata),
	}
	if c.embeddingFunc != nil {
		createOpts = append(createOpts, chroma.WithEmbeddingFunctionCreate(&chromaEmbeddingFunc{
			embed: c.embeddingFunc,
			space: c.distanceFunc,
		}))
	}
	collection, err := c.client.GetOrCreateCollection(
		ctx,
		c.collectionName,
		createOpts...,
	)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	c.collection = collection
	return nil
}

// DeleteCollection deletes a collection
func (c *ChromaDB) DeleteCollection(ctx context.Context, name string) error {
	if name == "" {
		name = c.collectionName
	}

	err := c.client.DeleteCollection(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	if name == c.collectionName {
		c.collection = nil
	}

	return nil
}

// Add adds documents to the collection
func (c *ChromaDB) Add(ctx context.Context, documents []vectordb.Document) error {
	if c.collection == nil {
		if err := c.CreateCollection(ctx, c.collectionName, nil); err != nil {
			return err
		}
	}

	if len(documents) == 0 {
		return nil
	}

	// Prepare data for ChromaDB
	ids := make([]chroma.DocumentID, len(documents))
	contents := make([]string, len(documents))
	metadatas := make([]chroma.DocumentMetadata, len(documents))
	embeddingsList := make([][]float32, len(documents))

	for i, doc := range documents {
		ids[i] = chroma.DocumentID(doc.ID)
		contents[i] = doc.Content
		if doc.Metadata != nil {
			metadatas[i] = chroma.NewMetadataFromMap(doc.Metadata)
		}
		embeddingsList[i] = doc.Embedding
	}

	// Generate embeddings if needed
	if c.embeddingFunc != nil {
		needsEmbedding := false
		for _, emb := range embeddingsList {
			if len(emb) == 0 {
				needsEmbedding = true
				break
			}
		}

		if needsEmbedding {
			generatedEmbeddings, err := c.embeddingFunc.Embed(ctx, contents)
			if err != nil {
				return fmt.Errorf("failed to generate embeddings: %w", err)
			}

			for i, emb := range embeddingsList {
				if len(emb) == 0 && i < len(generatedEmbeddings) {
					embeddingsList[i] = generatedEmbeddings[i]
				}
			}
		}
	}

	// Convert embeddings to ChromaDB format
	chromaEmbeddings, err := convertToChromaEmbeddings(embeddingsList)
	if err != nil {
		return fmt.Errorf("failed to convert embeddings: %w", err)
	}

	// Add to ChromaDB
	err = c.collection.Add(ctx,
		chroma.WithIDs(ids...),
		chroma.WithTexts(contents...),
		chroma.WithEmbeddings(chromaEmbeddings...),
		chroma.WithMetadatas(metadatas...),
	)
	if err != nil {
		return fmt.Errorf("failed to add documents: %w", err)
	}

	return nil
}

// Update updates existing documents in the collection
func (c *ChromaDB) Update(ctx context.Context, documents []vectordb.Document) error {
	if c.collection == nil {
		return fmt.Errorf("collection not initialized")
	}

	if len(documents) == 0 {
		return nil
	}

	ids := make([]chroma.DocumentID, len(documents))
	contents := make([]string, len(documents))
	metadatas := make([]chroma.DocumentMetadata, len(documents))
	embeddingsList := make([][]float32, len(documents))

	for i, doc := range documents {
		ids[i] = chroma.DocumentID(doc.ID)
		contents[i] = doc.Content
		if doc.Metadata != nil {
			metadatas[i] = chroma.NewMetadataFromMap(doc.Metadata)
		}
		embeddingsList[i] = doc.Embedding
	}

	if c.embeddingFunc != nil {
		needsEmbedding := false
		for _, emb := range embeddingsList {
			if len(emb) == 0 {
				needsEmbedding = true
				break
			}
		}

		if needsEmbedding {
			generatedEmbeddings, err := c.embeddingFunc.Embed(ctx, contents)
			if err != nil {
				return fmt.Errorf("failed to generate embeddings: %w", err)
			}

			for i, emb := range embeddingsList {
				if len(emb) == 0 && i < len(generatedEmbeddings) {
					embeddingsList[i] = generatedEmbeddings[i]
				}
			}
		}
	}

	chromaEmbeddings, err := convertToChromaEmbeddings(embeddingsList)
	if err != nil {
		return fmt.Errorf("failed to convert embeddings: %w", err)
	}

	// Upsert semantics: v2 Update requires existing docs; Upsert covers both.
	// v2 的 Update 要求文档已存在；Upsert 同时覆盖新建与更新。
	err = c.collection.Upsert(ctx,
		chroma.WithIDs(ids...),
		chroma.WithTexts(contents...),
		chroma.WithEmbeddings(chromaEmbeddings...),
		chroma.WithMetadatas(metadatas...),
	)
	if err != nil {
		return fmt.Errorf("failed to update documents: %w", err)
	}

	return nil
}

// Delete deletes documents from the collection by IDs
func (c *ChromaDB) Delete(ctx context.Context, ids []string) error {
	if c.collection == nil {
		return fmt.Errorf("collection not initialized")
	}

	if len(ids) == 0 {
		return nil
	}

	docIDs := make([]chroma.DocumentID, len(ids))
	for i, id := range ids {
		docIDs[i] = chroma.DocumentID(id)
	}

	err := c.collection.Delete(ctx, chroma.WithIDs(docIDs...))
	if err != nil {
		return fmt.Errorf("failed to delete documents: %w", err)
	}

	return nil
}

// Query searches for similar documents using text query
func (c *ChromaDB) Query(ctx context.Context, query string, limit int, filter map[string]interface{}) ([]vectordb.SearchResult, error) {
	if c.collection == nil {
		return nil, fmt.Errorf("collection not initialized")
	}

	if c.embeddingFunc == nil {
		return nil, fmt.Errorf("embedding function required for text query")
	}

	// Generate embedding for query
	embedding, err := c.embeddingFunc.EmbedSingle(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	return c.QueryWithEmbedding(ctx, embedding, limit, filter)
}

// QueryWithEmbedding searches for similar documents using pre-computed embedding
func (c *ChromaDB) QueryWithEmbedding(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]vectordb.SearchResult, error) {
	if c.collection == nil {
		return nil, fmt.Errorf("collection not initialized")
	}

	if limit <= 0 {
		limit = 10
	}

	// Convert embedding to ChromaDB format
	chromaEmb := embeddings.NewEmbeddingFromFloat32(embedding)

	// Build query options
	queryOpts := []chroma.CollectionQueryOption{
		chroma.WithQueryEmbeddings(chromaEmb),
		chroma.WithNResults(limit),
		chroma.WithInclude(chroma.IncludeDocuments, chroma.IncludeMetadatas, chroma.IncludeDistances),
	}

	// Add filter if provided
	if filter != nil {
		wf := whereFromMap(filter)
		if wf != nil {
			queryOpts = append(queryOpts, chroma.WithWhere(wf))
		}
	}

	// Query ChromaDB
	queryResult, err := c.collection.Query(ctx, queryOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}

	// Convert to our SearchResult format
	results := make([]vectordb.SearchResult, 0)
	idGroups := queryResult.GetIDGroups()
	if len(idGroups) > 0 {
		ids := idGroups[0]
		docGroups := queryResult.GetDocumentsGroups()
		metaGroups := queryResult.GetMetadatasGroups()
		distGroups := queryResult.GetDistancesGroups()

		for i := range ids {
			result := vectordb.SearchResult{
				ID: string(ids[i]),
			}

			if len(docGroups) > 0 && i < len(docGroups[0]) {
				result.Content = docGroups[0][i].ContentString()
			}

			if len(metaGroups) > 0 && i < len(metaGroups[0]) {
				result.Metadata = metadataToMap(metaGroups[0][i])
			}

			if len(distGroups) > 0 && i < len(distGroups[0]) {
				d := float64(distGroups[0][i])
				result.Distance = float32(d)
				result.Score = float32(1.0 / (1.0 + d))
			}

			results = append(results, result)
		}
	}

	return results, nil
}

// Get retrieves documents by IDs
func (c *ChromaDB) Get(ctx context.Context, ids []string) ([]vectordb.Document, error) {
	if c.collection == nil {
		return nil, fmt.Errorf("collection not initialized")
	}

	if len(ids) == 0 {
		return []vectordb.Document{}, nil
	}

	docIDs := make([]chroma.DocumentID, len(ids))
	for i, id := range ids {
		docIDs[i] = chroma.DocumentID(id)
	}

	// Get documents from ChromaDB
	result, err := c.collection.Get(ctx,
		chroma.WithIDs(docIDs...),
		chroma.WithInclude(chroma.IncludeDocuments, chroma.IncludeMetadatas, chroma.IncludeEmbeddings),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get documents: %w", err)
	}

	// Convert to our Document format
	documents := make([]vectordb.Document, 0)
	resIDs := result.GetIDs()
	if len(resIDs) > 0 {
		docList := result.GetDocuments()
		metaList := result.GetMetadatas()
		embList := result.GetEmbeddings()

		for i := range resIDs {
			doc := vectordb.Document{
				ID: string(resIDs[i]),
			}

			if i < len(docList) {
				doc.Content = docList[i].ContentString()
			}

			if i < len(metaList) {
				doc.Metadata = metadataToMap(metaList[i])
			}

			if i < len(embList) && embList[i] != nil {
				doc.Embedding = embList[i].ContentAsFloat32()
			}

			documents = append(documents, doc)
		}
	}

	return documents, nil
}

// Count returns the number of documents in the collection
func (c *ChromaDB) Count(ctx context.Context) (int, error) {
	if c.collection == nil {
		return 0, fmt.Errorf("collection not initialized")
	}

	count, err := c.collection.Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	return count, nil
}

// Close closes the connection to the vector database
func (c *ChromaDB) Close() error {
	if c.client != nil {
		_ = c.client.Close()
	}
	c.collection = nil
	c.client = nil
	return nil
}

// chromaEmbeddingFunc adapts a vectordb.EmbeddingFunction to the chroma-go
// embeddings.EmbeddingFunction interface. Without it, collection creation
// falls back to chroma-go's default local ONNX embedding function, which
// bootstraps an onnxruntime download and can fail on GitHub API rate limits.
type chromaEmbeddingFunc struct {
	embed vectordb.EmbeddingFunction
	space embeddings.DistanceMetric
}

func (f *chromaEmbeddingFunc) EmbedDocuments(ctx context.Context, texts []string) ([]embeddings.Embedding, error) {
	vectors, err := f.embed.Embed(ctx, texts)
	if err != nil {
		return nil, err
	}
	result := make([]embeddings.Embedding, len(vectors))
	for i, vector := range vectors {
		result[i] = embeddings.NewEmbeddingFromFloat32(vector)
	}
	return result, nil
}

func (f *chromaEmbeddingFunc) EmbedQuery(ctx context.Context, text string) (embeddings.Embedding, error) {
	vectors, err := f.embed.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, fmt.Errorf("no embedding generated for query")
	}
	return embeddings.NewEmbeddingFromFloat32(vectors[0]), nil
}

func (f *chromaEmbeddingFunc) Name() string {
	return "agno-custom"
}

func (f *chromaEmbeddingFunc) GetConfig() embeddings.EmbeddingFunctionConfig {
	return embeddings.EmbeddingFunctionConfig{}
}

func (f *chromaEmbeddingFunc) DefaultSpace() embeddings.DistanceMetric {
	return f.space
}

func (f *chromaEmbeddingFunc) SupportedSpaces() []embeddings.DistanceMetric {
	return []embeddings.DistanceMetric{f.space}
}

// convertToChromaEmbeddings converts [][]float32 to []embeddings.Embedding
func convertToChromaEmbeddings(embeddingsList [][]float32) ([]embeddings.Embedding, error) {
	out := make([]embeddings.Embedding, 0, len(embeddingsList))
	for _, e := range embeddingsList {
		if len(e) == 0 {
			// Empty embedding: caller must have provided text for server-side
			// embedding, but v2 requires embeddings when WithEmbeddings is used.
			// Return a zero-length embedding placeholder.
			// 空嵌入：调用方需提供文本供服务端嵌入。v2 使用 WithEmbeddings 时
			// 需要嵌入值，这里返回零长度占位。
			out = append(out, embeddings.NewEmbeddingFromFloat32([]float32{}))
			continue
		}
		out = append(out, embeddings.NewEmbeddingFromFloat32(e))
	}
	return out, nil
}

// whereFromMap converts a plain map filter into a ChromaDB WhereFilter
// (AND of equality clauses). Unsupported value types are skipped.
//
// whereFromMap 将普通 map 过滤器转换为 ChromaDB WhereFilter
// （相等子句的 AND 组合）。不支持的值类型被跳过。
func whereFromMap(filter map[string]interface{}) chroma.WhereFilter {
	clauses := make([]chroma.WhereClause, 0, len(filter))
	for k, v := range filter {
		if k == "" {
			continue
		}
		switch val := v.(type) {
		case string:
			clauses = append(clauses, chroma.EqString(k, val))
		case int:
			clauses = append(clauses, chroma.EqInt(k, val))
		case int64:
			clauses = append(clauses, chroma.EqInt(k, int(val)))
		case float64:
			clauses = append(clauses, chroma.EqFloat(k, float32(val)))
		case float32:
			clauses = append(clauses, chroma.EqFloat(k, val))
		case bool:
			clauses = append(clauses, chroma.EqBool(k, val))
		}
	}
	if len(clauses) == 0 {
		return nil
	}
	if len(clauses) == 1 {
		return clauses[0]
	}
	return chroma.And(clauses...)
}

// metadataToMap converts a v2 DocumentMetadata into a plain map. The v2
// interface has no key enumeration, so we reflect over the concrete
// implementation's internal map (works for DocumentMetadataImpl from the
// official client). Returns nil on failure. Metadata is best-effort: it
// must never panic the query path.
//
// metadataToMap 将 v2 DocumentMetadata 转换为普通 map。v2 接口没有
// key 枚举方法，因此对官方客户端的 DocumentMetadataImpl 内部 map
// 做反射读取。失败时返回 nil。metadata 是尽力而为：绝不能 panic
// 查询路径。
func metadataToMap(m chroma.DocumentMetadata) (out map[string]interface{}) {
	defer func() {
		if r := recover(); r != nil {
			// Unexported fields are not reflect-readable; metadata is
			// best-effort only.
			// 未导出字段不可反射读取；metadata 仅为尽力而为。
			out = nil
		}
	}()

	if m == nil {
		return nil
	}
	v := reflect.ValueOf(m)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	f := v.FieldByName("metadata")
	if !f.IsValid() || f.Kind() != reflect.Map || !f.CanInterface() {
		return nil
	}
	out = make(map[string]interface{})
	iter := f.MapRange()
	for iter.Next() {
		key, _ := iter.Key().Interface().(string)
		if key == "" {
			continue
		}
		mv := iter.Value()
		if mv.Kind() == reflect.Ptr {
			mv = mv.Elem()
		}
		out[key] = metadataValueToInterface(mv)
	}
	return out
}

// metadataValueToInterface extracts the underlying value of a MetadataValue.
// metadataValueToInterface 提取 MetadataValue 的底层值。
func metadataValueToInterface(mv reflect.Value) interface{} {
	if !mv.IsValid() {
		return nil
	}
	// Field order matches MetadataValue: Bool, Float64, Int, StringValue,
	// NilValue, StringArray, IntArray, FloatArray, BoolArray.
	// 字段顺序与 MetadataValue 一致。
	for i := 0; i < mv.NumField(); i++ {
		f := mv.Field(i)
		if f.Kind() == reflect.Ptr {
			if f.IsNil() {
				continue
			}
			f = f.Elem()
			if f.Kind() == reflect.String {
				return f.String()
			}
			if f.Kind() == reflect.Int64 {
				return f.Int()
			}
			if f.Kind() == reflect.Float64 {
				return f.Float()
			}
			if f.Kind() == reflect.Bool {
				return f.Bool()
			}
			continue
		}
		if f.Kind() == reflect.Bool {
			// NilValue marker: return nil.
			// NilValue 标记：返回 nil。
			if f.Bool() {
				return nil
			}
			continue
		}
		if f.Kind() == reflect.Slice && !f.IsNil() {
			out := make([]interface{}, 0, f.Len())
			for j := 0; j < f.Len(); j++ {
				out = append(out, f.Index(j).Interface())
			}
			return out
		}
	}
	return nil
}
