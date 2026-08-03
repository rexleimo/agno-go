package migrate

import (
    "fmt"

    "github.com/rexleimo/agno-go/pkg/hno/vectordb"
    "github.com/rexleimo/agno-go/pkg/hno/vectordb/chromadb"
)

// defaultFactory provides Chroma provider by default; Redis is optional and disabled here.
func defaultFactory(opts Options) (vectordb.VectorDB, error) {
    switch opts.Provider {
    case "chroma", "chromadb", "chromad b":
        cfg := chromadb.Config{BaseURL: opts.ChromaBaseURL, CollectionName: opts.Collection, Database: opts.ChromaDatabase, Tenant: opts.ChromaTenant}
        return chromadb.New(cfg)
    case "redis":
        return nil, fmt.Errorf("redis vectordb provider not enabled (build with -tags redis)")
    default:
        return nil, fmt.Errorf("unsupported provider: %s", opts.Provider)
    }
}

