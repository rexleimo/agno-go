package vectordb

import (
	"context"
)

// Document represents a document to be stored in the vector database
type Document struct {
	ID        string                 `json:"id"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Embedding []float32              `json:"embedding,omitempty"`
}

// SearchResult represents a search result from the vector database
type SearchResult struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Score    float32                `json:"score"`    // Similarity score (higher is better)
	Distance float32                `json:"distance"` // Distance metric (lower is better)
}

// VectorDB defines the interface for vector database operations
type VectorDB interface {
	// CreateCollection creates a new collection (or connects to existing one)
	CreateCollection(ctx context.Context, name string, metadata map[string]interface{}) error

	// DeleteCollection deletes a collection
	DeleteCollection(ctx context.Context, name string) error

	// Add adds documents to the collection
	Add(ctx context.Context, documents []Document) error

	// Update updates existing documents in the collection
	Update(ctx context.Context, documents []Document) error

	// Delete deletes documents from the collection by IDs
	Delete(ctx context.Context, ids []string) error

	// Query searches for similar documents using text query
	// The query text will be embedded automatically
	Query(ctx context.Context, query string, limit int, filter map[string]interface{}) ([]SearchResult, error)

	// QueryWithEmbedding searches for similar documents using pre-computed embedding
	QueryWithEmbedding(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]SearchResult, error)

	// Get retrieves documents by IDs
	Get(ctx context.Context, ids []string) ([]Document, error)

	// Count returns the number of documents in the collection
	Count(ctx context.Context) (int, error)

	// Close closes the connection to the vector database
	Close() error
}

// EmbeddingFunction defines the interface for generating embeddings
type EmbeddingFunction interface {
	// Embed generates embeddings for the given texts
	Embed(ctx context.Context, texts []string) ([][]float32, error)

	// EmbedSingle generates embedding for a single text
	EmbedSingle(ctx context.Context, text string) ([]float32, error)
}

// DistanceFunction represents the distance metric used for similarity search
type DistanceFunction string

const (
	// L2 is the Euclidean distance (default for most use cases)
	L2 DistanceFunction = "l2"

	// Cosine similarity (1 - cosine distance)
	Cosine DistanceFunction = "cosine"

	// InnerProduct (dot product)
	InnerProduct DistanceFunction = "ip"
)

// CollectionMetadata represents metadata for a collection
type CollectionMetadata struct {
	Name             string                 `json:"name"`
	Size             int                    `json:"size"`
	DistanceFunction DistanceFunction       `json:"distance_function,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}
