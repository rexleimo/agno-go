package sentencetransformer

import (
	"context"
	"fmt"
	"sync"

	"github.com/rexleimo/agno-go/pkg/hno/vectordb"
)

// Encoder defines the minimal behaviour required from a sentence transformer
// backend. It mirrors the Encode method exposed by popular transformer
// libraries so the embedder can wrap local or remote implementations.
type Encoder interface {
	Encode(ctx context.Context, texts []string) ([][]float32, error)
}

// Loader is responsible for creating the underlying encoder. It is executed at
// most once even when multiple goroutines attempt to embed simultaneously.
type Loader func() (Encoder, error)

// Embedder provides a vectordb.EmbeddingFunction backed by a SentenceTransformer
// encoder. Construction is cheap; the heavy model initialization occurs lazily
// via sync.Once to guarantee thread safety.
type Embedder struct {
	loader Loader
	once   sync.Once
	model  Encoder
	err    error
}

// New creates an Embedder using the provided loader.
func New(loader Loader) *Embedder {
	return &Embedder{loader: loader}
}

func (e *Embedder) ensureModel() error {
	if e.loader == nil {
		return fmt.Errorf("sentence transformer loader is not configured")
	}
	e.once.Do(func() {
		e.model, e.err = e.loader()
	})
	return e.err
}

// Embed implements vectordb.EmbeddingFunction.Embed.
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if err := e.ensureModel(); err != nil {
		return nil, err
	}
	if len(texts) == 0 {
		return [][]float32{}, nil
	}
	return e.model.Encode(ctx, texts)
}

// EmbedSingle implements vectordb.EmbeddingFunction.EmbedSingle.
func (e *Embedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	result, err := e.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return []float32{}, nil
	}
	return result[0], nil
}

var _ vectordb.EmbeddingFunction = (*Embedder)(nil)
