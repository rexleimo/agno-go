package openai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/rexleimo/agno-go/internal/agent"
	"github.com/rexleimo/agno-go/internal/model"
)

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestClientChatSuccess(t *testing.T) {
	client := New(availableStatus(agent.ProviderOpenAI), "http://noop", "key")
	client.http = &http.Client{
		Transport: rtFunc(func(req *http.Request) (*http.Response, error) {
			body := `{"choices":[{"message":{"content":"ok"}}],"usage":{"prompt_tokens":2,"completion_tokens":3}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		}),
	}
	resp, err := client.Chat(context.Background(), model.ChatRequest{
		Model:    agent.ModelConfig{ModelID: "gpt-test"},
		Messages: []agent.Message{{Role: agent.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if resp.Message.Content != "ok" {
		t.Fatalf("unexpected message: %+v", resp.Message)
	}
	if resp.Usage.PromptTokens != 2 || resp.Usage.CompletionTokens != 3 {
		t.Fatalf("usage mismatch: %+v", resp.Usage)
	}
}

func TestClientStreamParsesTokens(t *testing.T) {
	client := New(availableStatus(agent.ProviderOpenAI), "http://noop", "key")
	client.http = &http.Client{
		Transport: rtFunc(func(req *http.Request) (*http.Response, error) {
			body := "" +
				"data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\n" +
				"data: {\"choices\":[{\"delta\":{\"content\":\" there\"}}]}\n\n" +
				"data: [DONE]\n\n"
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			}, nil
		}),
	}
	var tokens []string
	err := client.Stream(context.Background(), model.ChatRequest{
		Model:    agent.ModelConfig{ModelID: "gpt-test", Stream: true},
		Messages: []agent.Message{{Role: agent.RoleUser, Content: "hi"}},
	}, func(ev model.ChatStreamEvent) error {
		if ev.Type == "token" {
			tokens = append(tokens, strings.TrimSpace(ev.Delta))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if len(tokens) != 2 || tokens[0] != "hi" || tokens[1] != "there" {
		t.Fatalf("unexpected tokens: %v", tokens)
	}
}

func TestClientEmbedSuccess(t *testing.T) {
	client := New(availableStatus(agent.ProviderOpenAI), "http://noop", "key")
	client.http = &http.Client{
		Transport: rtFunc(func(req *http.Request) (*http.Response, error) {
			body := `{"data":[{"embedding":[0.1,0.2]}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		}),
	}
	resp, err := client.Embed(context.Background(), model.EmbeddingRequest{
		Model: agent.ModelConfig{ModelID: "text-embed"},
		Input: []string{"hello"},
	})
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(resp.Vectors) != 1 || len(resp.Vectors[0]) != 2 {
		t.Fatalf("unexpected vectors: %+v", resp.Vectors)
	}
}

func TestClientErrorMappingUnauthorized(t *testing.T) {
	client := New(availableStatus(agent.ProviderOpenAI), "http://noop", "key")
	client.http = &http.Client{
		Transport: rtFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"bad key"}}`)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		}),
	}
	_, err := client.Chat(context.Background(), model.ChatRequest{
		Model:    agent.ModelConfig{ModelID: "gpt-test"},
		Messages: []agent.Message{{Role: agent.RoleUser, Content: "hi"}},
	})
	if err == nil || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("expected unauthorized error, got %v", err)
	}
}

func availableStatus(provider agent.Provider) model.ProviderStatus {
	st := model.DefaultProviderStatus(provider)
	st.Status = model.ProviderAvailable
	return st
}
