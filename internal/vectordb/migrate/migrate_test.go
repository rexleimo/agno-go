package migrate

import (
	"context"
	"os"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/vectordb"
)

type fakeDB struct {
	created string
	deleted string
}

func (f *fakeDB) CreateCollection(ctx context.Context, name string, metadata map[string]interface{}) error {
	f.created = name
	return nil
}
func (f *fakeDB) DeleteCollection(ctx context.Context, name string) error {
	f.deleted = name
	return nil
}
func (f *fakeDB) Add(ctx context.Context, documents []vectordb.Document) error    { return nil }
func (f *fakeDB) Update(ctx context.Context, documents []vectordb.Document) error { return nil }
func (f *fakeDB) Delete(ctx context.Context, ids []string) error                  { return nil }
func (f *fakeDB) Query(ctx context.Context, query string, limit int, filter map[string]interface{}) ([]vectordb.SearchResult, error) {
	return nil, nil
}
func (f *fakeDB) QueryWithEmbedding(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]vectordb.SearchResult, error) {
	return nil, nil
}
func (f *fakeDB) Get(ctx context.Context, ids []string) ([]vectordb.Document, error) { return nil, nil }
func (f *fakeDB) Count(ctx context.Context) (int, error)                             { return 0, nil }
func (f *fakeDB) Close() error                                                       { return nil }

func TestUpDown_UsesFactoryAndCallsProvider(t *testing.T) {
	old := ProviderFactory
	defer func() { ProviderFactory = old }()

	f := &fakeDB{}
	ProviderFactory = func(opts Options) (vectordb.VectorDB, error) { return f, nil }

	opts := Options{Provider: "chroma", Collection: "docs", Distance: "cosine"}

	if err := Up(context.Background(), opts); err != nil {
		t.Fatalf("Up error: %v", err)
	}
	if f.created != "docs" {
		t.Fatalf("expected created 'docs', got %q", f.created)
	}

	if err := Down(context.Background(), opts); err != nil {
		t.Fatalf("Down error: %v", err)
	}
	if f.deleted != "docs" {
		t.Fatalf("expected deleted 'docs', got %q", f.deleted)
	}
}

func TestRedisProvider_Gated(t *testing.T) {
	if v := os.Getenv("TEST_REDIS_VECTORDB"); v != "1" {
		t.Skip("redis provider is optional; skipping unless TEST_REDIS_VECTORDB=1")
	}
	old := ProviderFactory
	defer func() { ProviderFactory = old }()
	f := &fakeDB{}
	ProviderFactory = func(opts Options) (vectordb.VectorDB, error) { return f, nil }
	opts := Options{Provider: "redis", Collection: "docs"}
	if err := Up(context.Background(), opts); err != nil {
		t.Fatalf("Up error: %v", err)
	}
	if f.created != "docs" {
		t.Fatalf("expected created 'docs'")
	}
	if err := Down(context.Background(), opts); err != nil {
		t.Fatalf("Down error: %v", err)
	}
}
