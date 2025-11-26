package contract_test

import (
	"context"
	"testing"

	"github.com/rexleimo/agno-go/internal/agent"
	"github.com/rexleimo/agno-go/internal/model"
	"github.com/rexleimo/agno-go/internal/runtime"
	"github.com/rexleimo/agno-go/pkg/memory"
	"github.com/rexleimo/agno-go/pkg/providers/stub"
)

// CLI/SDK style flow using in-memory server and stub provider; exercises stream + state replay.
func TestCLISDKStreamingAndReplay(t *testing.T) {
	router := model.NewRouter()
	router.RegisterChatProvider(stub.New(agent.ProviderOpenAI, model.ProviderAvailable, nil))
	store := memory.NewInMemoryStore()
	svc := runtime.NewService(store, router)
	server := runtime.NewServer(router.Statuses, "dev", svc)

	agentID := createAgent(t, server.Router)
	sessionID := createSession(t, server.Router, agentID)

	// Non-stream
	msgResp := postMessage(t, server.Router, agentID, sessionID, false)
	if msgResp.Content == "" {
		t.Fatalf("empty message content")
	}
	if msgResp.Usage.PromptTokens == 0 {
		t.Fatalf("usage not populated")
	}

	// Stream
	events := postMessageStream(t, server.Router, agentID, sessionID)
	if len(events) == 0 {
		t.Fatalf("expected stream events")
	}
	var sawEnd bool
	for _, ev := range events {
		if ev.Done {
			sawEnd = true
		}
	}
	if !sawEnd {
		t.Fatalf("missing stream end event")
	}

	// Replay/History check
	history, err := store.LoadHistory(context.Background(), agentID, sessionID, agent.HistoryOptions{TokenWindow: 0})
	if err != nil {
		t.Fatalf("history load: %v", err)
	}
	if len(history) < 1 {
		t.Fatalf("expected history entries")
	}
}
