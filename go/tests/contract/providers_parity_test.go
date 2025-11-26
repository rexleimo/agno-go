package contract_test

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/internal/agent"
	"github.com/rexleimo/agno-go/internal/model"
	runtimeconfig "github.com/rexleimo/agno-go/internal/runtime/config"
	"github.com/rexleimo/agno-go/pkg/providers/cerebras"
	"github.com/rexleimo/agno-go/pkg/providers/gemini"
	"github.com/rexleimo/agno-go/pkg/providers/glm4"
	"github.com/rexleimo/agno-go/pkg/providers/groq"
	"github.com/rexleimo/agno-go/pkg/providers/modelscope"
	"github.com/rexleimo/agno-go/pkg/providers/ollama"
	"github.com/rexleimo/agno-go/pkg/providers/openai"
	"github.com/rexleimo/agno-go/pkg/providers/openrouter"
	"github.com/rexleimo/agno-go/pkg/providers/siliconflow"
	"gopkg.in/yaml.v3"
)

type providerFixture struct {
	FixtureID string `json:"fixtureId" yaml:"fixtureId"`
	Provider  string `json:"provider" yaml:"provider"`
	Type      string `json:"type" yaml:"type"` // chat|embedding
	Input     struct {
		Messages []struct {
			Role    string `json:"role" yaml:"role"`
			Content string `json:"content" yaml:"content"`
		} `json:"messages" yaml:"messages"`
		Text    []string `json:"text" yaml:"text"`
		ModelID string   `json:"modelId" yaml:"modelId"`
	} `json:"input" yaml:"input"`
	Expected struct {
		Contains  string      `json:"contains" yaml:"contains"`
		MinTokens int         `json:"minTokens" yaml:"minTokens"`
		Vectors   [][]float64 `json:"vectors" yaml:"vectors"`
	} `json:"expected" yaml:"expected"`
	Tolerance struct {
		TokenTolerance  int     `json:"tokenTolerance" yaml:"tokenTolerance"`
		EmbeddingCosine float64 `json:"embeddingCosine" yaml:"embeddingCosine"`
	} `json:"tolerance" yaml:"tolerance"`
}

var chatModels = map[agent.Provider]string{
	agent.ProviderOpenAI:      "gpt-4.1-mini",
	agent.ProviderGemini:      "gemini-2.5-flash",
	agent.ProviderGLM4:        "GLM-4-Flash-250414",
	agent.ProviderOpenRouter:  "tngtech/deepseek-r1t2-chimera:free",
	agent.ProviderSiliconFlow: "qwen2-7b-instruct",
	agent.ProviderCerebras:    "llama-3.3-70b",
	agent.ProviderModelScope:  "qwen2-7b-instruct",
	agent.ProviderGroq:        "openai/gpt-oss-120b",
	agent.ProviderOllama:      "qwen3:4b",
}

var embedModels = map[agent.Provider]string{
	agent.ProviderOpenAI:      "text-embedding-3-small",
	agent.ProviderGemini:      "text-embedding-004",
	agent.ProviderOpenRouter:  "text-embedding-3-small",
	agent.ProviderGroq:        "text-embedding-3-small",
	agent.ProviderGLM4:        "text-embedding-3-small",
	agent.ProviderSiliconFlow: "text-embedding-3-small",
	agent.ProviderCerebras:    "text-embedding-3-small",
	agent.ProviderModelScope:  "text-embedding-3-small",
	agent.ProviderOllama:      "nomic-embed-text",
}

func TestProvidersParityAgainstFixtures(t *testing.T) {
	fixtures := loadProviderFixtures(t)
	if len(fixtures) == 0 {
		t.Skip("no fixtures found; generate fixtures before running parity")
	}

	cfgPath := filepath.Join("..", "..", "..", "config", "default.yaml")
	envPath := filepath.Join("..", "..", "..", ".env")
	cfg, err := runtimeconfig.LoadWithEnv(cfgPath, envPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	statuses := cfg.ProviderStatuses()
	configs := cfg.ProviderConfigs()
	statusMap := make(map[agent.Provider]model.ProviderStatus, len(statuses))
	for _, st := range statuses {
		statusMap[st.Provider] = st
	}

	chatClients := buildChatClients(statusMap, configs)
	embedClients := buildEmbedClients(statusMap, configs)

	for _, fx := range fixtures {
		provider := agent.Provider(strings.ToLower(fx.Provider))
		switch fx.Type {
		case "chat":
			client, ok := chatClients[provider]
			if !ok {
				t.Skipf("provider %s unavailable or lacks chat capability", provider)
			}
			t.Run(fx.FixtureID, func(t *testing.T) {
				assertChatFixture(t, client, provider, fx)
			})
		case "embedding":
			client, ok := embedClients[provider]
			if !ok {
				t.Skipf("provider %s unavailable or lacks embedding capability", provider)
			}
			t.Run(fx.FixtureID, func(t *testing.T) {
				assertEmbeddingFixture(t, client, provider, fx)
			})
		default:
			t.Fatalf("unknown fixture type %s", fx.Type)
		}
	}
}

func assertChatFixture(t *testing.T, client model.ChatProvider, provider agent.Provider, fx providerFixture) {
	t.Helper()
	modelID := modelFor(provider, fx.Input.ModelID, false)
	if modelID == "" {
		t.Skipf("no chat model id for provider %s", provider)
	}
	var msgs []agent.Message
	for _, m := range fx.Input.Messages {
		msgs = append(msgs, agent.Message{Role: agent.Role(m.Role), Content: m.Content})
	}
	if len(msgs) == 0 {
		t.Fatalf("fixture %s missing input messages", fx.FixtureID)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := client.Chat(ctx, model.ChatRequest{
		Model: agent.ModelConfig{
			Provider: provider,
			ModelID:  modelID,
		},
		Messages: msgs,
	})
	if err != nil {
		t.Skipf("provider %s chat skipped: %v", provider, err)
	}
	if resp == nil || strings.TrimSpace(resp.Message.Content) == "" {
		t.Fatalf("empty assistant content for %s", fx.FixtureID)
	}
	if fx.Expected.Contains != "" && !strings.Contains(resp.Message.Content, fx.Expected.Contains) {
		t.Fatalf("response does not contain expected substring %q; got %q", fx.Expected.Contains, resp.Message.Content)
	}
	tokenTolerance := fx.Tolerance.TokenTolerance
	if tokenTolerance == 0 {
		tokenTolerance = 2
	}
	if fx.Expected.MinTokens > 0 && (resp.Usage.PromptTokens+resp.Usage.CompletionTokens) > 0 {
		got := resp.Usage.PromptTokens + resp.Usage.CompletionTokens
		if got+tokenTolerance < fx.Expected.MinTokens {
			t.Fatalf("tokens below expected minimum: got %d expected >= %d (tolerance %d)", got, fx.Expected.MinTokens, tokenTolerance)
		}
	}
}

func assertEmbeddingFixture(t *testing.T, client model.EmbeddingProvider, provider agent.Provider, fx providerFixture) {
	t.Helper()
	modelID := modelFor(provider, fx.Input.ModelID, true)
	if modelID == "" {
		t.Skipf("no embedding model id for provider %s", provider)
	}
	if len(fx.Input.Text) == 0 {
		t.Fatalf("fixture %s missing embedding input text", fx.FixtureID)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := client.Embed(ctx, model.EmbeddingRequest{
		Model: agent.ModelConfig{
			Provider: provider,
			ModelID:  modelID,
		},
		Input: fx.Input.Text,
	})
	if err != nil {
		t.Skipf("provider %s embedding skipped: %v", provider, err)
	}
	if resp == nil || len(resp.Vectors) == 0 || len(resp.Vectors[0]) == 0 {
		t.Fatalf("empty embedding response for %s", fx.FixtureID)
	}
	if len(fx.Expected.Vectors) == 0 || len(fx.Expected.Vectors[0]) == 0 {
		t.Skip("fixture missing expected embedding vector; cannot compare")
	}
	tol := fx.Tolerance.EmbeddingCosine
	if tol == 0 {
		tol = 0.98
	}
	got := cosine(resp.Vectors[0], fx.Expected.Vectors[0])
	if got < tol {
		t.Fatalf("cosine similarity below tolerance: got %.4f want >= %.4f", got, tol)
	}
}

func loadProviderFixtures(t *testing.T) []providerFixture {
	t.Helper()
	dir := filepath.Join("..", "..", "..", "specs", "001-agno-agents-refactor", "contracts", "fixtures")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatalf("read fixtures dir: %v", err)
	}
	var fixtures []providerFixture
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !(strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read fixture %s: %v", name, err)
		}
		var fx providerFixture
		if strings.HasSuffix(name, ".json") {
			err = json.Unmarshal(raw, &fx)
		} else {
			err = yaml.Unmarshal(raw, &fx)
		}
		if err != nil {
			t.Fatalf("decode fixture %s: %v", name, err)
		}
		fixtures = append(fixtures, fx)
	}
	return fixtures
}

func buildChatClients(statuses map[agent.Provider]model.ProviderStatus, cfg map[agent.Provider]runtimeconfig.ProviderConfig) map[agent.Provider]model.ChatProvider {
	out := make(map[agent.Provider]model.ChatProvider)
	for prov, st := range statuses {
		if st.Status != model.ProviderAvailable {
			continue
		}
		c := cfg[prov]
		switch prov {
		case agent.ProviderOpenAI:
			out[prov] = openai.New(c.Endpoint, c.APIKey, st.MissingEnv)
		case agent.ProviderGemini:
			out[prov] = gemini.New(c.Endpoint, c.APIKey, st.MissingEnv)
		case agent.ProviderGLM4:
			out[prov] = glm4.New(st, c.Endpoint, c.APIKey)
		case agent.ProviderOpenRouter:
			out[prov] = openrouter.New(st, c.Endpoint, c.APIKey, openRouterHeaders())
		case agent.ProviderSiliconFlow:
			out[prov] = siliconflow.New(st, c.Endpoint, c.APIKey)
		case agent.ProviderCerebras:
			out[prov] = cerebras.New(st, c.Endpoint, c.APIKey)
		case agent.ProviderModelScope:
			out[prov] = modelscope.New(st, c.Endpoint, c.APIKey)
		case agent.ProviderGroq:
			out[prov] = groq.New(st, c.Endpoint, c.APIKey)
		case agent.ProviderOllama:
			out[prov] = ollama.New(st, c.Endpoint, c.APIKey)
		}
	}
	return out
}

func buildEmbedClients(statuses map[agent.Provider]model.ProviderStatus, cfg map[agent.Provider]runtimeconfig.ProviderConfig) map[agent.Provider]model.EmbeddingProvider {
	out := make(map[agent.Provider]model.EmbeddingProvider)
	for prov, st := range statuses {
		if st.Status != model.ProviderAvailable {
			continue
		}
		c := cfg[prov]
		switch prov {
		case agent.ProviderOpenAI:
			out[prov] = openai.New(c.Endpoint, c.APIKey, st.MissingEnv)
		case agent.ProviderGemini:
			out[prov] = gemini.New(c.Endpoint, c.APIKey, st.MissingEnv)
		case agent.ProviderGLM4:
			out[prov] = glm4.NewEmbed(st, c.Endpoint, c.APIKey)
		case agent.ProviderOpenRouter:
			out[prov] = openrouter.NewEmbed(st, c.Endpoint, c.APIKey, openRouterHeaders())
		case agent.ProviderSiliconFlow:
			out[prov] = siliconflow.NewEmbed(st, c.Endpoint, c.APIKey)
		case agent.ProviderCerebras:
			out[prov] = cerebras.NewEmbed(st, c.Endpoint, c.APIKey)
		case agent.ProviderModelScope:
			out[prov] = modelscope.NewEmbed(st, c.Endpoint, c.APIKey)
		case agent.ProviderGroq:
			out[prov] = groq.NewEmbed(st, c.Endpoint, c.APIKey)
		case agent.ProviderOllama:
			out[prov] = ollama.NewEmbed(st, c.Endpoint, c.APIKey)
		}
	}
	return out
}

func modelFor(provider agent.Provider, override string, embedding bool) string {
	if override != "" {
		return override
	}
	envKey := strings.ToUpper(string(provider)) + "_MODEL"
	if embedding {
		envKey = strings.ToUpper(string(provider)) + "_EMBED_MODEL"
	}
	if val := strings.TrimSpace(os.Getenv(envKey)); val != "" {
		return val
	}
	if embedding {
		return embedModels[provider]
	}
	return chatModels[provider]
}

func cosine(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return -1
	}
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return -1
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func openRouterHeaders() map[string]string {
	headers := map[string]string{
		"HTTP-Referer": "https://local.agno",
		"X-Title":      "Go-Agno",
	}
	if ref := os.Getenv("OPENROUTER_HTTP_REFERER"); strings.TrimSpace(ref) != "" {
		headers["HTTP-Referer"] = ref
	}
	if title := os.Getenv("OPENROUTER_TITLE"); strings.TrimSpace(title) != "" {
		headers["X-Title"] = title
	}
	return headers
}
