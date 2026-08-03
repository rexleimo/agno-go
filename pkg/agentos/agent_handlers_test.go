package agentos

import (
	"context"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/run"
)

func TestDeriveRunContextUsesPayload(t *testing.T) {
	ctx, rc := deriveRunContext(context.Background(), &RunContextRequest{
		RunID:     "run-123",
		SessionID: "sess-1",
		UserID:    "user-42",
	}, "")
	if rc.RunID != "run-123" {
		t.Fatalf("expected run id run-123, got %s", rc.RunID)
	}
	if rc.SessionID != "sess-1" {
		t.Fatalf("expected session id sess-1, got %s", rc.SessionID)
	}
	stored, ok := run.FromContext(ctx)
	if !ok || stored == nil {
		t.Fatal("expected run context in returned context")
	}
	if stored.RunID != rc.RunID {
		t.Fatalf("context run id mismatch: %s vs %s", stored.RunID, rc.RunID)
	}
}

func TestDeriveRunContextFallsBackToSession(t *testing.T) {
	ctx, rc := deriveRunContext(context.Background(), nil, "sess-fallback")
	if rc.SessionID != "sess-fallback" {
		t.Fatalf("expected fallback session, got %s", rc.SessionID)
	}
	stored, _ := run.FromContext(ctx)
	if stored == nil || stored.SessionID != "sess-fallback" {
		t.Fatal("expected stored context to include fallback session")
	}
	if stored.RunID == "" {
		t.Fatal("expected generated run id")
	}
}
