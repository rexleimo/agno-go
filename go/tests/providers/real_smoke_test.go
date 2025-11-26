package providers

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/internal/agent"
	"github.com/rexleimo/agno-go/internal/model"
	runtimeconfig "github.com/rexleimo/agno-go/internal/runtime/config"
	runtimeproviders "github.com/rexleimo/agno-go/internal/runtime/providers"
)

// User-provided preferred model IDs.
var providerModels = map[agent.Provider]string{
	agent.ProviderOpenAI:      "gpt-4.1-mini",
	agent.ProviderOpenRouter:  "tngtech/deepseek-r1t2-chimera:free",
	agent.ProviderGroq:        "openai/gpt-oss-120b",
	agent.ProviderGemini:      "gemini-2.5-flash",
	agent.ProviderGLM4:        "GLM-4-Flash-250414",
	agent.ProviderSiliconFlow: "qwen2-7b-instruct",
	agent.ProviderModelScope:  "qwen2-7b-instruct",
	agent.ProviderOllama:      "huihui_ai/qwen3-abliterated:8b",
	agent.ProviderCerebras:    "llama-3.3-70b",
}

func TestProviderSmokes(t *testing.T) {
	baseDir := repoRoot(t)
	cfgPath := filepath.Join(baseDir, "config", "default.yaml")
	envPath := filepath.Join(baseDir, ".env")
	cfg, err := runtimeconfig.LoadWithEnv(cfgPath, envPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	configs := cfg.ProviderConfigs()
	statuses := cfg.ProviderStatuses()

	tests := []agent.Provider{
		agent.ProviderOpenRouter,
		agent.ProviderGroq,
		agent.ProviderGemini,
		agent.ProviderGLM4,
		agent.ProviderSiliconFlow,
		agent.ProviderModelScope,
		agent.ProviderOllama,
		agent.ProviderCerebras,
	}

	for _, prov := range tests {
		st := findStatus(statuses, prov)
		if st.Status != model.ProviderAvailable {
			t.Logf("skip %s: %s (missing: %v)", prov, st.Status, st.MissingEnv)
			continue
		}
		modelID := providerModels[prov]
		if modelID == "" {
			t.Fatalf("missing model ID for provider %s", prov)
		}
		endpoint := configs[prov].Endpoint
		apiKey := configs[prov].APIKey

		t.Run(string(prov), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			client, err := newProviderClient(prov, st, endpoint, apiKey)
			if err != nil {
				t.Fatalf("client: %v", err)
			}
			req := model.ChatRequest{
				Model: agent.ModelConfig{
					Provider: prov,
					ModelID:  modelID,
					Stream:   false,
				},
				Messages: []agent.Message{
					{Role: agent.RoleUser, Content: "Hello! respond with a short greeting."},
				},
			}
			resp, err := client.Chat(ctx, req)
			if err != nil {
				t.Logf("chat error for %s: %v", prov, err)
				return
			}
			if resp == nil || resp.Message.Content == "" {
				t.Fatalf("empty response from %s", prov)
			}
			t.Logf("%s usage prompt=%d completion=%d finish=%s", prov, resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.FinishReason)
		})
	}
}

func findStatus(all []model.ProviderStatus, p agent.Provider) model.ProviderStatus {
	for _, st := range all {
		if st.Provider == p {
			return st
		}
	}
	st := model.DefaultProviderStatus(p)
	st.Status = model.ProviderNotConfigured
	return st
}

func newProviderClient(prov agent.Provider, st model.ProviderStatus, endpoint, apiKey string) (model.ChatProvider, error) {
	cfg := runtimeconfig.ProviderConfig{
		Endpoint:   endpoint,
		APIKey:     apiKey,
		Status:     st.Status,
		MissingEnv: append([]string(nil), st.MissingEnv...),
		Reason:     st.Reason,
	}
	clients, err := runtimeproviders.Build(prov, st, cfg)
	if err != nil {
		return nil, err
	}
	if clients.Chat == nil {
		return nil, fmt.Errorf("provider %s does not expose chat client", prov)
	}
	return clients.Chat, nil
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("cannot resolve caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
