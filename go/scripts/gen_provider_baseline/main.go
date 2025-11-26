package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/internal/agent"
	"github.com/rexleimo/agno-go/internal/model"
	runtimeconfig "github.com/rexleimo/agno-go/internal/runtime/config"
	"github.com/rexleimo/agno-go/internal/runtime/providers"
)

// providerFixture mirrors the contract fixture shape we consume in tests.
type providerFixture struct {
	FixtureID string `json:"fixtureId" yaml:"fixtureId"`
	Provider  string `json:"provider" yaml:"provider"`
	Type      string `json:"type" yaml:"type"` // chat|embedding
	Input     struct {
		Messages []struct {
			Role    string `json:"role" yaml:"role"`
			Content string `json:"content" yaml:"content"`
		} `json:"messages,omitempty" yaml:"messages,omitempty"`
		Text    []string `json:"text,omitempty" yaml:"text,omitempty"`
		ModelID string   `json:"modelId" yaml:"modelId"`
	} `json:"input" yaml:"input"`
	Expected struct {
		Contains  string      `json:"contains,omitempty" yaml:"contains,omitempty"`
		MinTokens int         `json:"minTokens,omitempty" yaml:"minTokens,omitempty"`
		Vectors   [][]float64 `json:"vectors,omitempty" yaml:"vectors,omitempty"`
	} `json:"expected" yaml:"expected"`
	Tolerance struct {
		TokenTolerance  int     `json:"tokenTolerance" yaml:"tokenTolerance"`
		EmbeddingCosine float64 `json:"embeddingCosine" yaml:"embeddingCosine"`
	} `json:"tolerance" yaml:"tolerance"`
	SourceCommit string `json:"sourceCommit,omitempty" yaml:"sourceCommit,omitempty"`
	Notes        string `json:"notes,omitempty" yaml:"notes,omitempty"`
}

var (
	chatModels = map[agent.Provider]string{
		agent.ProviderOpenAI:      "gpt-4o-mini",
		agent.ProviderGemini:      "gemini-1.5-flash",
		agent.ProviderGLM4:        "glm-4",
		agent.ProviderOpenRouter:  "openrouter/auto",
		agent.ProviderSiliconFlow: "Qwen/Qwen2.5-7B-Instruct",
		agent.ProviderCerebras:    "llama3.1-8b",
		agent.ProviderModelScope:  "qwen2-7b-instruct",
		agent.ProviderGroq:        "llama-3.3-70b-versatile",
		agent.ProviderOllama:      "llama3",
	}
	embedModels = map[agent.Provider]string{
		agent.ProviderOpenAI:      "text-embedding-3-small",
		agent.ProviderGemini:      "textembedding-gecko",
		agent.ProviderOpenRouter:  "openai/text-embedding-3-small",
		agent.ProviderGroq:        "", // groq embeddings not available
		agent.ProviderGLM4:        "glm-4-embedding",
		agent.ProviderSiliconFlow: "bge-large-en-v1.5",
		agent.ProviderCerebras:    "mistral-embed",
		agent.ProviderModelScope:  "bge-base-en-v1.5",
		agent.ProviderOllama:      "all-minilm",
	}
)

func main() {
	cfgPath := flag.String("config", filepath.FromSlash("../config/default.yaml"), "path to config YAML")
	envPath := flag.String("env", filepath.FromSlash("../.env"), "path to .env file")
	destDir := flag.String("dest", filepath.FromSlash("../specs/001-agno-agents-refactor/contracts/fixtures"), "destination fixtures directory")
	deviations := flag.String("deviations", filepath.FromSlash("../specs/001-agno-agents-refactor/contracts/deviations.md"), "path to deviations log")
	flag.Parse()

	cfg, err := runtimeconfig.LoadWithEnv(*cfgPath, *envPath)
	if err != nil {
		failWithDeviation(*deviations, "load config: %v", err)
	}

	if err := os.MkdirAll(*destDir, 0o755); err != nil {
		failWithDeviation(*deviations, "mkdir dest: %v", err)
	}

	statuses := cfg.ProviderStatuses()
	configs := cfg.ProviderConfigs()

	now := time.Now().UTC().Format(time.RFC3339)
	var generated int
	var hadErr bool

	for _, st := range statuses {
		if st.Status != model.ProviderAvailable {
			log.Printf("skip %s: %s (missing=%v)", st.Provider, st.Status, st.MissingEnv)
			continue
		}
		prov := st.Provider
		cfgEntry := configs[prov]
		chatClient, embedClient := buildClients(prov, st, cfgEntry)
		if chatClient != nil {
			if err := writeChatFixture(*destDir, prov, chatClient, now); err != nil {
				appendDeviation(*deviations, fmt.Sprintf("chat %s: %v", prov, err))
				hadErr = true
			} else {
				generated++
			}
		}
		if embedClient != nil {
			if err := writeEmbedFixture(*destDir, prov, embedClient, now); err != nil {
				appendDeviation(*deviations, fmt.Sprintf("embed %s: %v", prov, err))
				hadErr = true
			} else {
				generated++
			}
		}
	}

	log.Printf("fixtures generated/updated: %d -> %s", generated, *destDir)
	if hadErr {
		failWithDeviation(*deviations, "provider baseline generation completed with errors; see deviations at %s", time.Now().UTC().Format(time.RFC3339))
	}
}

func appendDeviation(path, msg string) {
	if path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Printf("warn: unable to write deviation to %s: %v", path, err)
		return
	}
	defer func() { _ = f.Close() }()
	ts := time.Now().UTC().Format(time.RFC3339)
	if _, err := fmt.Fprintf(f, "- [fixtures] %s: %s\n", ts, msg); err != nil {
		log.Printf("warn: unable to append deviation: %v", err)
	}
}

func failWithDeviation(path, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	appendDeviation(path, msg)
	log.Fatalf("%s", msg)
}

func buildClients(p agent.Provider, st model.ProviderStatus, cfg runtimeconfig.ProviderConfig) (model.ChatProvider, model.EmbeddingProvider) {
	clients, err := providers.Build(p, st, cfg)
	if err != nil {
		return nil, nil
	}
	return clients.Chat, clients.Embed
}

func writeChatFixture(dest string, p agent.Provider, client model.ChatProvider, now string) error {
	modelID := chosenModel(p, false)
	if modelID == "" {
		return fmt.Errorf("no chat model for %s", p)
	}
	prompt := fmt.Sprintf("Respond ONLY with: BASELINE-%s", strings.ToUpper(string(p)))
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	_, err := client.Chat(ctx, model.ChatRequest{
		Model:    agent.ModelConfig{Provider: p, ModelID: modelID},
		Messages: []agent.Message{{Role: agent.RoleUser, Content: prompt}},
	})
	if err != nil {
		return err
	}
	fx := providerFixture{}
	fx.FixtureID = fmt.Sprintf("chat-%s-baseline", p)
	fx.Provider = string(p)
	fx.Type = "chat"
	fx.Input.ModelID = modelID
	fx.Input.Messages = []struct {
		Role    string `json:"role" yaml:"role"`
		Content string `json:"content" yaml:"content"`
	}{{Role: "user", Content: prompt}}
	fx.Expected.Contains = fmt.Sprintf("BASELINE-%s", strings.ToUpper(string(p)))
	fx.Expected.MinTokens = 0
	fx.Tolerance.TokenTolerance = 2
	fx.Tolerance.EmbeddingCosine = 0.98
	fx.SourceCommit = "go-generated"
	fx.Notes = fmt.Sprintf("Generated %s via live provider call", now)

	path := filepath.Join(dest, fmt.Sprintf("chat_%s.json", p))
	return writeFixtureFile(path, fx)
}

func writeEmbedFixture(dest string, p agent.Provider, client model.EmbeddingProvider, now string) error {
	modelID := chosenModel(p, true)
	if modelID == "" {
		return fmt.Errorf("no embed model for %s", p)
	}
	text := fmt.Sprintf("baseline embedding text for %s", p)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	resp, err := client.Embed(ctx, model.EmbeddingRequest{
		Model: agent.ModelConfig{Provider: p, ModelID: modelID},
		Input: []string{text},
	})
	if err != nil {
		return err
	}
	if len(resp.Vectors) == 0 {
		return fmt.Errorf("empty embedding from %s", p)
	}
	fx := providerFixture{}
	fx.FixtureID = fmt.Sprintf("embedding-%s-baseline", p)
	fx.Provider = string(p)
	fx.Type = "embedding"
	fx.Input.ModelID = modelID
	fx.Input.Text = []string{text}
	fx.Expected.Vectors = resp.Vectors
	fx.Tolerance.TokenTolerance = 2
	fx.Tolerance.EmbeddingCosine = 0.98
	fx.SourceCommit = "go-generated"
	fx.Notes = fmt.Sprintf("Generated %s via live provider call", now)

	path := filepath.Join(dest, fmt.Sprintf("embedding_%s.json", p))
	return writeFixtureFile(path, fx)
}

func writeFixtureFile(path string, fx providerFixture) error {
	data, err := json.MarshalIndent(fx, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	return nil
}

func chosenModel(p agent.Provider, embedding bool) string {
	prefix := strings.ToUpper(string(p))
	var candidates []string
	if embedding {
		candidates = []string{
			fmt.Sprintf("%s_EMBED_MODEL", prefix),
			fmt.Sprintf("%s_MODEL", prefix),
		}
	} else {
		candidates = []string{
			fmt.Sprintf("%s_CHAT_MODEL", prefix),
			fmt.Sprintf("%s_MODEL", prefix),
		}
	}
	for _, key := range candidates {
		if val := strings.TrimSpace(os.Getenv(key)); val != "" {
			return val
		}
	}
	if embedding {
		return embedModels[p]
	}
	return chatModels[p]
}
