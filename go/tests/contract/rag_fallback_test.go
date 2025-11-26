package contract_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rexleimo/agno-go/internal/knowledge"
)

type failingRetriever struct {
	err error
}

func (f failingRetriever) Retrieve(context.Context, string, map[string]string) ([]knowledge.Chunk, error) {
	if f.err != nil {
		return nil, f.err
	}
	return nil, errors.New("retriever failure")
}

func TestKnowledgeFallbackHintDoesNotPolluteState(t *testing.T) {
	engine := knowledge.Engine{
		Retriever: nil,
		Fallback: knowledge.Strategy{
			Mode:        knowledge.ModeHint,
			HintMessage: "Fallback hint",
		},
	}
	chunks, msg, err := engine.Fetch(context.Background(), "query", nil)
	if err != nil {
		t.Fatalf("expected hint fallback without error, got %v", err)
	}
	if msg == "" || !strings.Contains(msg, "Fallback") {
		t.Fatalf("expected fallback hint message, got %q", msg)
	}
	if len(chunks) != 0 {
		t.Fatalf("expected no chunks when retriever missing")
	}
	logRAGCoverage(t, "hint_mode", "ok")
}

func TestKnowledgeFallbackErrorMode(t *testing.T) {
	engine := knowledge.Engine{
		Retriever: failingRetriever{err: errors.New("vector store down")},
		Fallback: knowledge.Strategy{
			Mode:        knowledge.ModeError,
			ErrorPrefix: "RAG failed",
		},
	}
	chunks, msg, err := engine.Fetch(context.Background(), "query", nil)
	if err == nil || !errors.Is(err, knowledge.ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
	if msg != "" {
		t.Fatalf("expected no hint message in error mode, got %q", msg)
	}
	if len(chunks) != 0 {
		t.Fatalf("expected no chunks on error fallback")
	}
	logRAGCoverage(t, "error_mode", "ok")
}

func logRAGCoverage(t *testing.T, scenario, status string) {
	t.Helper()
	base := repoRoot(t)
	coverageLog := filepath.Join(base, "specs", "001-agno-agents-refactor", "artifacts", "coverage.txt")
	line := fmt.Sprintf("rag_fallback_%s status=%s\n", scenario, status)
	_ = appendFile(coverageLog, []byte(line))
}
