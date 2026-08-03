package chromadb

import (
	"context"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/vectordb"
)

// MockEmbeddingFunction is a mock embedding function for testing
type MockEmbeddingFunction struct{}

func (m *MockEmbeddingFunction) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i := range texts {
		// Simple mock: create a 384-dimensional embedding with some variation
		embeddings[i] = make([]float32, 384)
		for j := range embeddings[i] {
			embeddings[i][j] = float32(i*10 + j)
		}
	}
	return embeddings, nil
}

func (m *MockEmbeddingFunction) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := m.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return embeddings[0], nil
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				CollectionName: "test_collection",
				BaseURL:        "http://localhost:8000",
			},
			wantErr: false,
		},
		{
			name: "missing collection name",
			config: Config{
				BaseURL: "http://localhost:8000",
			},
			wantErr: true,
		},
		{
			name: "default base URL",
			config: Config{
				CollectionName: "test_collection",
			},
			wantErr: false,
		},
		{
			name: "with embedding function",
			config: Config{
				CollectionName:    "test_collection",
				BaseURL:           "http://localhost:8000",
				EmbeddingFunction: &MockEmbeddingFunction{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if db == nil {
					t.Error("New() returned nil database")
				}
				if db.collectionName != tt.config.CollectionName {
					t.Errorf("collectionName = %v, want %v", db.collectionName, tt.config.CollectionName)
				}
			}
		})
	}
}

func TestCreateCollection(t *testing.T) {
	// This test requires a running ChromaDB instance
	// Skip if ChromaDB is not available
	t.Skip("Requires running ChromaDB instance")

	db, err := New(Config{
		CollectionName: "test_create_collection",
		BaseURL:        "http://localhost:8000",
	})
	if err != nil {
		t.Fatalf("Failed to create ChromaDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Test creating collection
	err = db.CreateCollection(ctx, "", map[string]interface{}{
		"description": "Test collection",
	})
	if err != nil {
		t.Fatalf("CreateCollection() error = %v", err)
	}

	if db.collection == nil {
		t.Error("Collection not initialized after CreateCollection()")
	}

	// Clean up
	err = db.DeleteCollection(ctx, "test_create_collection")
	if err != nil {
		t.Errorf("DeleteCollection() error = %v", err)
	}
}

func TestAddAndQuery(t *testing.T) {
	// This test requires a running ChromaDB instance
	t.Skip("Requires running ChromaDB instance")

	db, err := New(Config{
		CollectionName:    "test_add_query",
		BaseURL:           "http://localhost:8000",
		EmbeddingFunction: &MockEmbeddingFunction{},
	})
	if err != nil {
		t.Fatalf("Failed to create ChromaDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create collection
	err = db.CreateCollection(ctx, "", nil)
	if err != nil {
		t.Fatalf("CreateCollection() error = %v", err)
	}

	// Add documents
	documents := []vectordb.Document{
		{
			ID:      "doc1",
			Content: "The quick brown fox jumps over the lazy dog",
			Metadata: map[string]interface{}{
				"category": "animals",
			},
		},
		{
			ID:      "doc2",
			Content: "A journey of a thousand miles begins with a single step",
			Metadata: map[string]interface{}{
				"category": "wisdom",
			},
		},
		{
			ID:      "doc3",
			Content: "To be or not to be, that is the question",
			Metadata: map[string]interface{}{
				"category": "literature",
			},
		},
	}

	err = db.Add(ctx, documents)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Count documents
	count, err := db.Count(ctx)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != len(documents) {
		t.Errorf("Count() = %v, want %v", count, len(documents))
	}

	// Query documents
	results, err := db.Query(ctx, "quick fox", 2, nil)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if len(results) == 0 {
		t.Error("Query() returned no results")
	}

	// First result should be doc1
	if len(results) > 0 && results[0].ID != "doc1" {
		t.Errorf("Query() first result ID = %v, want doc1", results[0].ID)
	}

	// Clean up
	db.DeleteCollection(ctx, "test_add_query")
}

func TestGet(t *testing.T) {
	// This test requires a running ChromaDB instance
	t.Skip("Requires running ChromaDB instance")

	db, err := New(Config{
		CollectionName:    "test_get",
		BaseURL:           "http://localhost:8000",
		EmbeddingFunction: &MockEmbeddingFunction{},
	})
	if err != nil {
		t.Fatalf("Failed to create ChromaDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create collection and add documents
	err = db.CreateCollection(ctx, "", nil)
	if err != nil {
		t.Fatalf("CreateCollection() error = %v", err)
	}

	documents := []vectordb.Document{
		{
			ID:      "doc1",
			Content: "Document 1",
			Metadata: map[string]interface{}{
				"index": 1,
			},
		},
		{
			ID:      "doc2",
			Content: "Document 2",
			Metadata: map[string]interface{}{
				"index": 2,
			},
		},
	}

	err = db.Add(ctx, documents)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Get documents by IDs
	retrieved, err := db.Get(ctx, []string{"doc1", "doc2"})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(retrieved) != 2 {
		t.Errorf("Get() returned %v documents, want 2", len(retrieved))
	}

	// Verify content
	for i, doc := range retrieved {
		if doc.ID != documents[i].ID {
			t.Errorf("Get() doc %d ID = %v, want %v", i, doc.ID, documents[i].ID)
		}
		if doc.Content != documents[i].Content {
			t.Errorf("Get() doc %d Content = %v, want %v", i, doc.Content, documents[i].Content)
		}
	}

	// Clean up
	db.DeleteCollection(ctx, "test_get")
}

func TestUpdate(t *testing.T) {
	// This test requires a running ChromaDB instance
	t.Skip("Requires running ChromaDB instance")

	db, err := New(Config{
		CollectionName:    "test_update",
		BaseURL:           "http://localhost:8000",
		EmbeddingFunction: &MockEmbeddingFunction{},
	})
	if err != nil {
		t.Fatalf("Failed to create ChromaDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create collection and add document
	err = db.CreateCollection(ctx, "", nil)
	if err != nil {
		t.Fatalf("CreateCollection() error = %v", err)
	}

	original := []vectordb.Document{
		{
			ID:      "doc1",
			Content: "Original content",
			Metadata: map[string]interface{}{
				"version": 1,
			},
		},
	}

	err = db.Add(ctx, original)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Update document
	updated := []vectordb.Document{
		{
			ID:      "doc1",
			Content: "Updated content",
			Metadata: map[string]interface{}{
				"version": 2,
			},
		},
	}

	err = db.Update(ctx, updated)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	retrieved, err := db.Get(ctx, []string{"doc1"})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(retrieved) != 1 {
		t.Fatalf("Get() returned %v documents, want 1", len(retrieved))
	}

	if retrieved[0].Content != "Updated content" {
		t.Errorf("Updated content = %v, want 'Updated content'", retrieved[0].Content)
	}

	// Clean up
	db.DeleteCollection(ctx, "test_update")
}

func TestDelete(t *testing.T) {
	// This test requires a running ChromaDB instance
	t.Skip("Requires running ChromaDB instance")

	db, err := New(Config{
		CollectionName:    "test_delete",
		BaseURL:           "http://localhost:8000",
		EmbeddingFunction: &MockEmbeddingFunction{},
	})
	if err != nil {
		t.Fatalf("Failed to create ChromaDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create collection and add documents
	err = db.CreateCollection(ctx, "", nil)
	if err != nil {
		t.Fatalf("CreateCollection() error = %v", err)
	}

	documents := []vectordb.Document{
		{
			ID:      "doc1",
			Content: "Document 1",
		},
		{
			ID:      "doc2",
			Content: "Document 2",
		},
		{
			ID:      "doc3",
			Content: "Document 3",
		},
	}

	err = db.Add(ctx, documents)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Delete one document
	err = db.Delete(ctx, []string{"doc2"})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify count
	count, err := db.Count(ctx)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 2 {
		t.Errorf("Count() after delete = %v, want 2", count)
	}

	// Verify deletion
	retrieved, err := db.Get(ctx, []string{"doc1", "doc2", "doc3"})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(retrieved) != 2 {
		t.Errorf("Get() returned %v documents, want 2", len(retrieved))
	}

	// doc2 should not be in results
	for _, doc := range retrieved {
		if doc.ID == "doc2" {
			t.Error("Deleted document doc2 still exists")
		}
	}

	// Clean up
	db.DeleteCollection(ctx, "test_delete")
}

func TestQueryWithEmbedding(t *testing.T) {
	// This test requires a running ChromaDB instance
	t.Skip("Requires running ChromaDB instance")

	db, err := New(Config{
		CollectionName:    "test_query_embedding",
		BaseURL:           "http://localhost:8000",
		EmbeddingFunction: &MockEmbeddingFunction{},
	})
	if err != nil {
		t.Fatalf("Failed to create ChromaDB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create collection and add documents
	err = db.CreateCollection(ctx, "", nil)
	if err != nil {
		t.Fatalf("CreateCollection() error = %v", err)
	}

	documents := []vectordb.Document{
		{
			ID:      "doc1",
			Content: "Machine learning is a subset of AI",
		},
		{
			ID:      "doc2",
			Content: "Deep learning uses neural networks",
		},
	}

	err = db.Add(ctx, documents)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Generate query embedding
	mockEmbed := &MockEmbeddingFunction{}
	queryEmbedding, err := mockEmbed.EmbedSingle(ctx, "artificial intelligence")
	if err != nil {
		t.Fatalf("Failed to generate query embedding: %v", err)
	}

	// Query with embedding
	results, err := db.QueryWithEmbedding(ctx, queryEmbedding, 2, nil)
	if err != nil {
		t.Fatalf("QueryWithEmbedding() error = %v", err)
	}

	if len(results) == 0 {
		t.Error("QueryWithEmbedding() returned no results")
	}

	// Verify results have required fields
	for i, result := range results {
		if result.ID == "" {
			t.Errorf("Result %d has empty ID", i)
		}
		if result.Content == "" {
			t.Errorf("Result %d has empty Content", i)
		}
	}

	// Clean up
	db.DeleteCollection(ctx, "test_query_embedding")
}
