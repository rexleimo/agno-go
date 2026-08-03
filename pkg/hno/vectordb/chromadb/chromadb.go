package chromadb

import (
	"context"
	"fmt"

	chroma "github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/types"

	"github.com/rexleimo/agno-go/pkg/hno/vectordb"
)

// Helper functions for embedding conversion

// convertToChromaEmbeddings converts [][]float32 to []*types.Embedding
func convertToChromaEmbeddings(embeddings [][]float32) []*types.Embedding {
	return types.NewEmbeddingsFromFloat32(embeddings)
}

// convertFromChromaEmbeddings converts []*types.Embedding to [][]float32
func convertFromChromaEmbeddings(embeddings []*types.Embedding) [][]float32 {
	result := make([][]float32, len(embeddings))
	for i, emb := range embeddings {
		if emb != nil && emb.ArrayOfFloat32 != nil {
			result[i] = *emb.ArrayOfFloat32
		}
	}
	return result
}

// ChromaDB implements the VectorDB interface using ChromaDB
type ChromaDB struct {
	client         *chroma.Client
	collection     *chroma.Collection
	collectionName string
	embeddingFunc  vectordb.EmbeddingFunction
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
	if config.DistanceFunction == "" {
		config.DistanceFunction = vectordb.L2
	}

	// Create client options
	var clientOpts []chroma.ClientOption
	clientOpts = append(clientOpts, chroma.WithBasePath(config.BaseURL))
	clientOpts = append(clientOpts, chroma.WithTenant(config.Tenant))
	clientOpts = append(clientOpts, chroma.WithDatabase(config.Database))

	// Add cloud API key if provided
	if config.CloudAPIKey != "" {
		clientOpts = append(clientOpts, chroma.WithAuth(types.NewTokenAuthCredentialsProvider(config.CloudAPIKey, types.XChromaTokenHeader)))
	}

	// Create ChromaDB client
	client, err := chroma.NewClient(clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChromaDB client: %w", err)
	}

	db := &ChromaDB{
		client:         client,
		collectionName: config.CollectionName,
		embeddingFunc:  config.EmbeddingFunction,
	}

	return db, nil
}

// CreateCollection creates a new collection or connects to existing one
func (c *ChromaDB) CreateCollection(ctx context.Context, name string, metadata map[string]interface{}) error {
	if name != "" {
		c.collectionName = name
	}

	// Convert distance function
	distanceFunc := types.L2
	if metadata != nil {
		if df, ok := metadata["distance_function"].(vectordb.DistanceFunction); ok {
			switch df {
			case vectordb.Cosine:
				distanceFunc = types.COSINE
			case vectordb.InnerProduct:
				distanceFunc = types.IP
			default:
				distanceFunc = types.L2
			}
		}
	}

	// Convert metadata to ChromaDB format
	chromaMetadata := make(map[string]interface{})
	if metadata != nil {
		for k, v := range metadata {
			if k != "distance_function" {
				chromaMetadata[k] = v
			}
		}
	}

	// Create or get collection (createOrGet = true means get if exists)
	collection, err := c.client.CreateCollection(
		ctx,
		c.collectionName,
		chromaMetadata,
		true, // createOrGet = true
		nil,  // embedding function (we handle it ourselves)
		distanceFunc,
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

	_, err := c.client.DeleteCollection(ctx, name)
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
	ids := make([]string, len(documents))
	contents := make([]string, len(documents))
	metadatas := make([]map[string]interface{}, len(documents))
	embeddings := make([][]float32, len(documents))

	for i, doc := range documents {
		ids[i] = doc.ID
		contents[i] = doc.Content
		metadatas[i] = doc.Metadata
		embeddings[i] = doc.Embedding
	}

	// Generate embeddings if needed and embedding function is provided
	if c.embeddingFunc != nil {
		// Check if any documents need embeddings
		needsEmbedding := false
		for _, emb := range embeddings {
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

			// Use generated embeddings for documents that don't have them
			for i, emb := range embeddings {
				if len(emb) == 0 {
					embeddings[i] = generatedEmbeddings[i]
				}
			}
		}
	}

	// Convert embeddings to ChromaDB format
	chromaEmbeddings := convertToChromaEmbeddings(embeddings)

	// Add to ChromaDB
	_, err := c.collection.Add(ctx, chromaEmbeddings, metadatas, contents, ids)
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

	// Prepare data for ChromaDB
	ids := make([]string, len(documents))
	contents := make([]string, len(documents))
	metadatas := make([]map[string]interface{}, len(documents))
	embeddings := make([][]float32, len(documents))

	for i, doc := range documents {
		ids[i] = doc.ID
		contents[i] = doc.Content
		metadatas[i] = doc.Metadata
		embeddings[i] = doc.Embedding
	}

	// Generate embeddings if needed
	if c.embeddingFunc != nil {
		needsEmbedding := false
		for _, emb := range embeddings {
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

			for i, emb := range embeddings {
				if len(emb) == 0 {
					embeddings[i] = generatedEmbeddings[i]
				}
			}
		}
	}

	// Convert embeddings to ChromaDB format
	chromaEmbeddings := convertToChromaEmbeddings(embeddings)

	// Modify in ChromaDB (Update is for collection metadata, Modify is for documents)
	_, err := c.collection.Modify(ctx, chromaEmbeddings, metadatas, contents, ids)
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

	_, err := c.collection.Delete(ctx, ids, nil, nil)
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
	chromaEmb := types.NewEmbeddingFromFloat32(embedding)

	// Build query options
	queryOpts := []types.CollectionQueryOption{
		types.WithQueryEmbedding(chromaEmb),
		types.WithNResults(int32(limit)),
		types.WithInclude("documents", "metadatas", "distances"),
	}

	// Add filter if provided
	if filter != nil {
		queryOpts = append(queryOpts, types.WithWhereMap(filter))
	}

	// Query ChromaDB using QueryWithOptions
	queryResult, err := c.collection.QueryWithOptions(ctx, queryOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}

	// Convert to our SearchResult format
	results := make([]vectordb.SearchResult, 0)
	if queryResult != nil && len(queryResult.Ids) > 0 {
		for i := range queryResult.Ids[0] {
			result := vectordb.SearchResult{
				ID: queryResult.Ids[0][i],
			}

			if queryResult.Documents != nil && len(queryResult.Documents) > 0 {
				result.Content = queryResult.Documents[0][i]
			}

			if queryResult.Metadatas != nil && len(queryResult.Metadatas) > 0 {
				result.Metadata = queryResult.Metadatas[0][i]
			}

			if queryResult.Distances != nil && len(queryResult.Distances) > 0 {
				result.Distance = queryResult.Distances[0][i]
				// Calculate score (inverse of distance)
				result.Score = 1.0 / (1.0 + queryResult.Distances[0][i])
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

	// Get documents from ChromaDB
	result, err := c.collection.Get(
		ctx,
		nil, // where
		nil, // whereDocuments
		ids,
		[]types.QueryEnum{"documents", "metadatas", "embeddings"},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get documents: %w", err)
	}

	// Convert to our Document format
	documents := make([]vectordb.Document, 0)
	if result != nil && len(result.Ids) > 0 {
		for i := range result.Ids {
			doc := vectordb.Document{
				ID: result.Ids[i],
			}

			if result.Documents != nil && len(result.Documents) > i {
				doc.Content = result.Documents[i]
			}

			if result.Metadatas != nil && len(result.Metadatas) > i {
				doc.Metadata = result.Metadatas[i]
			}

			if result.Embeddings != nil && len(result.Embeddings) > i && result.Embeddings[i] != nil {
				if result.Embeddings[i].ArrayOfFloat32 != nil {
					doc.Embedding = *result.Embeddings[i].ArrayOfFloat32
				}
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

	return int(count), nil
}

// Close closes the connection to the vector database
func (c *ChromaDB) Close() error {
	// ChromaDB Go client doesn't require explicit closing
	c.collection = nil
	c.client = nil
	return nil
}
