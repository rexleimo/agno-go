package sentencetransformer

import (
	"context"
	"sync"
	"testing"
)

type stubEncoder struct {
	encodeFn func(ctx context.Context, texts []string) ([][]float32, error)
}

func (s *stubEncoder) Encode(ctx context.Context, texts []string) ([][]float32, error) {
	return s.encodeFn(ctx, texts)
}

func TestEmbedderInitialisesOnce(t *testing.T) {
	var loads int
	loader := func() (Encoder, error) {
		loads++
		return &stubEncoder{encodeFn: func(ctx context.Context, texts []string) ([][]float32, error) {
			result := make([][]float32, len(texts))
			for i := range texts {
				result[i] = []float32{1, 2, 3}
			}
			return result, nil
		}}, nil
	}

	embedder := New(loader)
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := embedder.Embed(context.Background(), []string{"test"}); err != nil {
				t.Errorf("Embed() error = %v", err)
			}
		}()
	}
	wg.Wait()
	if loads != 1 {
		t.Fatalf("expected loader to run once, got %d", loads)
	}
}
