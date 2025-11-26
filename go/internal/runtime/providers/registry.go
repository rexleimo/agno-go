package providers

import (
	"fmt"
	"os"
	"strings"

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
)

// Clients holds chat/embedding providers for a single vendor.
type Clients struct {
	Chat  model.ChatProvider
	Embed model.EmbeddingProvider
}

// Build constructs chat/embedding providers for the given vendor using runtime config.
func Build(p agent.Provider, st model.ProviderStatus, cfg runtimeconfig.ProviderConfig) (Clients, error) {
	switch p {
	case agent.ProviderOpenAI:
		client := openai.New(st, cfg.Endpoint, cfg.APIKey)
		return Clients{Chat: client, Embed: client}, nil
	case agent.ProviderGemini:
		client := gemini.New(st, cfg.Endpoint, cfg.APIKey)
		return Clients{Chat: client, Embed: client}, nil
	case agent.ProviderGLM4:
		return Clients{
			Chat:  glm4.New(st, cfg.Endpoint, cfg.APIKey),
			Embed: glm4.NewEmbed(st, cfg.Endpoint, cfg.APIKey),
		}, nil
	case agent.ProviderOpenRouter:
		headers := openRouterHeaders()
		return Clients{
			Chat:  openrouter.New(st, cfg.Endpoint, cfg.APIKey, headers),
			Embed: openrouter.NewEmbed(st, cfg.Endpoint, cfg.APIKey, headers),
		}, nil
	case agent.ProviderSiliconFlow:
		return Clients{
			Chat:  siliconflow.New(st, cfg.Endpoint, cfg.APIKey),
			Embed: siliconflow.NewEmbed(st, cfg.Endpoint, cfg.APIKey),
		}, nil
	case agent.ProviderCerebras:
		return Clients{
			Chat:  cerebras.New(st, cfg.Endpoint, cfg.APIKey),
			Embed: cerebras.NewEmbed(st, cfg.Endpoint, cfg.APIKey),
		}, nil
	case agent.ProviderModelScope:
		return Clients{
			Chat:  modelscope.New(st, cfg.Endpoint, cfg.APIKey),
			Embed: modelscope.NewEmbed(st, cfg.Endpoint, cfg.APIKey),
		}, nil
	case agent.ProviderGroq:
		return Clients{
			Chat: groq.New(st, cfg.Endpoint, cfg.APIKey),
		}, nil
	case agent.ProviderOllama:
		return Clients{
			Chat:  ollama.New(st, cfg.Endpoint, cfg.APIKey),
			Embed: ollama.NewEmbed(st, cfg.Endpoint, cfg.APIKey),
		}, nil
	default:
		return Clients{}, fmt.Errorf("unsupported provider: %s", p)
	}
}

func openRouterHeaders() map[string]string {
	headers := map[string]string{
		"HTTP-Referer": "https://local.agno",
		"X-Title":      "Go-Agno",
	}
	if ref := strings.TrimSpace(os.Getenv("OPENROUTER_HTTP_REFERER")); ref != "" {
		headers["HTTP-Referer"] = ref
	}
	if title := strings.TrimSpace(os.Getenv("OPENROUTER_TITLE")); title != "" {
		headers["X-Title"] = title
	}
	return headers
}
