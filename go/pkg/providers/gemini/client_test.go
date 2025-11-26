package gemini

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

func TestChatSuccess(t *testing.T) {
	client := New(availableStatus(agent.ProviderGemini), "", "key")
	client.http = &http.Client{Transport: rtFunc(func(req *http.Request) (*http.Response, error) {
		body := `{"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})}

	resp, err := client.Chat(context.Background(), model.ChatRequest{
		Model:    agent.ModelConfig{ModelID: "gemini-1"},
		Messages: []agent.Message{{Role: agent.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if resp.Message.Content != "ok" {
		t.Fatalf("unexpected: %+v", resp.Message)
	}
}

func TestStreamParsesTokens(t *testing.T) {
	client := New(availableStatus(agent.ProviderGemini), "", "key")
	client.http = &http.Client{Transport: rtFunc(func(req *http.Request) (*http.Response, error) {
		body := "" +
			"data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"hi\"}]}}]}\n\n" +
			"data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\" there\"}]}}]}\n\n" +
			"data: [DONE]\n\n"
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		}, nil
	})}

	var tokens []string
	err := client.Stream(context.Background(), model.ChatRequest{
		Model:    agent.ModelConfig{ModelID: "gemini-1", Stream: true},
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

func TestEmbedSuccess(t *testing.T) {
	client := New(availableStatus(agent.ProviderGemini), "", "key")
	client.http = &http.Client{Transport: rtFunc(func(req *http.Request) (*http.Response, error) {
		body := `{"embedding":{"values":[0.1,0.2]}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})}

	resp, err := client.Embed(context.Background(), model.EmbeddingRequest{
		Model: agent.ModelConfig{ModelID: "embed"},
		Input: []string{"text"},
	})
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(resp.Vectors) != 1 || len(resp.Vectors[0]) != 2 {
		t.Fatalf("unexpected vectors: %+v", resp.Vectors)
	}
}

func TestErrorMapping(t *testing.T) {
	client := New(availableStatus(agent.ProviderGemini), "", "key")
	client.http = &http.Client{Transport: rtFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader("fail")),
		}, nil
	})}
	if _, err := client.Chat(context.Background(), model.ChatRequest{
		Model:    agent.ModelConfig{ModelID: "gemini-1"},
		Messages: []agent.Message{{Role: agent.RoleUser, Content: "hi"}},
	}); err == nil {
		t.Fatalf("expected error")
	}
}

func availableStatus(provider agent.Provider) model.ProviderStatus {
	st := model.DefaultProviderStatus(provider)
	st.Status = model.ProviderAvailable
	return st
}
