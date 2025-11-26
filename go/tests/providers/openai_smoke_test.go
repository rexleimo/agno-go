package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/internal/agent"
	"github.com/rexleimo/agno-go/internal/model"
	"github.com/rexleimo/agno-go/pkg/providers/openai"
)

func TestOpenAIStreamingAndErrorBranches(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set; skipping")
	}
	status := model.DefaultProviderStatus(agent.ProviderOpenAI)
	status.Status = model.ProviderAvailable

	streamSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		writeChunk := func(delta string) {
			chunk := map[string]any{
				"choices": []map[string]any{
					{"delta": map[string]string{"content": delta}},
				},
			}
			payload, _ := json.Marshal(chunk)
			if _, err := fmt.Fprintf(w, "data: %s\n\n", string(payload)); err != nil {
				panic(err)
			}
		}
		writeChunk("hi")
		writeChunk(" there")
		if _, err := fmt.Fprint(w, "data: [DONE]\n\n"); err != nil {
			panic(err)
		}
	}))
	defer streamSrv.Close()

	client := openai.New(status, streamSrv.URL, apiKey)
	req := model.ChatRequest{
		Model: agent.ModelConfig{
			Provider: agent.ProviderOpenAI,
			ModelID:  "gpt-test",
			Stream:   true,
		},
		Messages: []agent.Message{
			{Role: agent.RoleUser, Content: "ping stream"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var tokens []string
	var sawEnd bool
	if err := client.Stream(ctx, req, func(ev model.ChatStreamEvent) error {
		if ev.Type == "token" {
			tokens = append(tokens, strings.TrimSpace(ev.Delta))
		}
		if ev.Done {
			sawEnd = true
		}
		return nil
	}); err != nil {
		t.Fatalf("stream: %v", err)
	}
	if len(tokens) == 0 || tokens[0] != "hi" {
		t.Fatalf("expected streaming tokens, got %v", tokens)
	}
	if !sawEnd {
		t.Fatalf("expected end event to be signaled")
	}

	errorSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer errorSrv.Close()

	errorClient := openai.New(status, errorSrv.URL, apiKey)
	errorReq := req
	errorReq.Stream = false
	if _, err := errorClient.Chat(ctx, errorReq); err == nil {
		t.Fatalf("expected error response from chat endpoint")
	}
}
