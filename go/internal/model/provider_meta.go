package model

import (
	"slices"

	"github.com/rexleimo/agno-go/internal/agent"
)

// ProviderInfo describes built-in metadata for a provider.
type ProviderInfo struct {
	Provider          agent.Provider
	Priority          int
	Capabilities      []Capability
	Fallbacks         []agent.Provider
	DefaultChatModel  string
	DefaultEmbedModel string
}

// providerCatalog holds static metadata for the supported providers.
var providerCatalog = map[agent.Provider]ProviderInfo{
	agent.ProviderOpenAI: {
		Provider:          agent.ProviderOpenAI,
		Priority:          100,
		Capabilities:      []Capability{CapabilityChat, CapabilityEmbedding, CapabilityStreaming},
		DefaultChatModel:  "gpt-4o-mini",
		DefaultEmbedModel: "text-embedding-3-small",
	},
	agent.ProviderOpenRouter: {
		Provider:          agent.ProviderOpenRouter,
		Priority:          95,
		Capabilities:      []Capability{CapabilityChat, CapabilityEmbedding, CapabilityStreaming},
		DefaultChatModel:  "openrouter/auto",
		DefaultEmbedModel: "openrouter/embedding",
	},
	agent.ProviderGemini: {
		Provider:          agent.ProviderGemini,
		Priority:          90,
		Capabilities:      []Capability{CapabilityChat, CapabilityEmbedding, CapabilityStreaming},
		DefaultChatModel:  "gemini-1.5-pro",
		DefaultEmbedModel: "textembedding-gecko",
	},
	agent.ProviderGroq: {
		Provider:          agent.ProviderGroq,
		Priority:          85,
		Capabilities:      []Capability{CapabilityChat, CapabilityEmbedding, CapabilityStreaming},
		DefaultChatModel:  "llama3-8b-8192",
		DefaultEmbedModel: "text-embedding-3-large",
	},
	agent.ProviderGLM4: {
		Provider:          agent.ProviderGLM4,
		Priority:          80,
		Capabilities:      []Capability{CapabilityChat, CapabilityEmbedding, CapabilityStreaming},
		DefaultChatModel:  "glm-4",
		DefaultEmbedModel: "glm-4-embedding",
	},
	agent.ProviderSiliconFlow: {
		Provider:          agent.ProviderSiliconFlow,
		Priority:          75,
		Capabilities:      []Capability{CapabilityChat, CapabilityEmbedding, CapabilityStreaming},
		DefaultChatModel:  "qwen2.5-72b-instruct",
		DefaultEmbedModel: "bge-large-en-v1.5",
	},
	agent.ProviderCerebras: {
		Provider:          agent.ProviderCerebras,
		Priority:          70,
		Capabilities:      []Capability{CapabilityChat, CapabilityEmbedding, CapabilityStreaming},
		DefaultChatModel:  "llama3.1-8b",
		DefaultEmbedModel: "mistral-embed",
	},
	agent.ProviderModelScope: {
		Provider:          agent.ProviderModelScope,
		Priority:          65,
		Capabilities:      []Capability{CapabilityChat, CapabilityEmbedding, CapabilityStreaming},
		DefaultChatModel:  "qwen2-7b-instruct",
		DefaultEmbedModel: "bge-base-en-v1.5",
	},
	agent.ProviderOllama: {
		Provider:          agent.ProviderOllama,
		Priority:          60,
		Capabilities:      []Capability{CapabilityChat, CapabilityEmbedding, CapabilityStreaming},
		DefaultChatModel:  "huihui_ai/qwen3-abliterated:8b",
		DefaultEmbedModel: "huihui_ai/deepseek-r1-abliterated:8b",
	},
}

func init() {
	order := make([]agent.Provider, 0, len(providerCatalog))
	for provider := range providerCatalog {
		order = append(order, provider)
	}
	slices.SortFunc(order, func(a, b agent.Provider) int {
		return providerCatalog[b].Priority - providerCatalog[a].Priority
	})
	for _, provider := range order {
		fallbacks := make([]agent.Provider, 0, len(order)-1)
		for _, candidate := range order {
			if candidate == provider {
				continue
			}
			fallbacks = append(fallbacks, candidate)
		}
		info := providerCatalog[provider]
		info.Fallbacks = fallbacks
		providerCatalog[provider] = info
	}
}

// ProviderMetadata returns read-only metadata for a provider.
func ProviderMetadata(p agent.Provider) ProviderInfo {
	if info, ok := providerCatalog[p]; ok {
		return info
	}
	return ProviderInfo{
		Provider:     p,
		Priority:     0,
		Capabilities: []Capability{CapabilityChat, CapabilityEmbedding, CapabilityStreaming},
		Fallbacks:    nil,
	}
}

// DefaultProviderStatus builds a ProviderStatus seeded with metadata.
func DefaultProviderStatus(p agent.Provider) ProviderStatus {
	info := ProviderMetadata(p)
	return ProviderStatus{
		Provider:     p,
		Status:       ProviderDisabled,
		Capabilities: append([]Capability(nil), info.Capabilities...),
		Priority:     info.Priority,
		Fallbacks:    append([]agent.Provider(nil), info.Fallbacks...),
	}
}
