package migrate

import (
	"context"
	"fmt"

	"github.com/rexleimo/agno-go/pkg/hno/vectordb"
)

// Options controls migration behavior for a VectorDB provider
type Options struct {
	Provider   string
	Collection string
	// Chroma-specific
	ChromaBaseURL  string
	ChromaTenant   string
	ChromaDatabase string
	Distance       string // l2|cosine|ip
}

// Factory creates a VectorDB instance from options
type Factory func(opts Options) (vectordb.VectorDB, error)

// ProviderFactory can be overridden in tests
var ProviderFactory Factory = defaultFactory

// Up ensures the collection exists with optional metadata (distance)
func Up(ctx context.Context, opts Options) error {
	if opts.Collection == "" {
		return fmt.Errorf("collection is required")
	}
	db, err := ProviderFactory(opts)
	if err != nil {
		return err
	}
	defer db.Close()

	meta := map[string]interface{}{}
	switch opts.Distance {
	case "cosine":
		meta["distance_function"] = vectordb.Cosine
	case "ip", "inner", "inner_product":
		meta["distance_function"] = vectordb.InnerProduct
	case "", "l2":
		meta["distance_function"] = vectordb.L2
	default:
		return fmt.Errorf("invalid distance: %s", opts.Distance)
	}
	return db.CreateCollection(ctx, opts.Collection, meta)
}

// Down drops the collection
func Down(ctx context.Context, opts Options) error {
	if opts.Collection == "" {
		return fmt.Errorf("collection is required")
	}
	db, err := ProviderFactory(opts)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.DeleteCollection(ctx, opts.Collection)
}
