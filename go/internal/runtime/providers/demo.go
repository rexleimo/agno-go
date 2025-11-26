package providers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/internal/agent"
	"github.com/rexleimo/agno-go/internal/model"
	runtimeconfig "github.com/rexleimo/agno-go/internal/runtime/config"
)

// CapabilityResult captures the outcome of a provider capability probe.
type CapabilityResult struct {
	Status string
	Error  string
}

// DemoResult aggregates the outcomes for chat/stream/embed capabilities.
type DemoResult struct {
	Provider agent.Provider
	Chat     CapabilityResult
	Stream   CapabilityResult
	Embed    CapabilityResult
	Reason   string
}

// Exercise probes the configured provider using lightweight chat/stream/embed calls.
func Exercise(ctx context.Context, cfg *runtimeconfig.Config, provider agent.Provider, status model.ProviderStatus, entry runtimeconfig.ProviderConfig) DemoResult {
	if status.Provider == "" {
		status = model.DefaultProviderStatus(provider)
		status.Status = model.ProviderNotConfigured
		status.Reason = "provider status unavailable"
	}
	res := DemoResult{Provider: provider}
	if status.Status != model.ProviderAvailable {
		res.Chat = CapabilityResult{Status: "skipped"}
		res.Stream = CapabilityResult{Status: "skipped"}
		res.Embed = CapabilityResult{Status: "skipped"}
		res.Reason = status.Reason
		return res
	}

	clients, err := Build(provider, status, entry)
	if err != nil {
		res.Chat = CapabilityResult{Status: "error", Error: err.Error()}
		res.Stream = CapabilityResult{Status: "error", Error: err.Error()}
		res.Embed = CapabilityResult{Status: "error", Error: err.Error()}
		res.Reason = err.Error()
		return res
	}

	if clients.Chat == nil {
		res.Chat = CapabilityResult{Status: "skipped", Error: "chat unavailable"}
		res.Stream = CapabilityResult{Status: "skipped", Error: "chat unavailable"}
	} else {
		modelID := ResolveModel(provider, false)
		if modelID == "" {
			res.Chat = CapabilityResult{Status: "skipped", Error: "chat model not configured"}
			res.Stream = CapabilityResult{Status: "skipped", Error: "chat model not configured"}
		} else {
			msgs := []agent.Message{{Role: agent.RoleUser, Content: "Reply with a short identifier for Go demo smoke test."}}
			_, err := clients.Chat.Chat(ctx, model.ChatRequest{
				Model:    agent.ModelConfig{Provider: provider, ModelID: modelID},
				Messages: msgs,
			})
			if err != nil {
				res.Chat = CapabilityResult{Status: "error", Error: err.Error()}
			} else {
				res.Chat = CapabilityResult{Status: "success"}
			}
			streamErr := clients.Chat.Stream(ctx, model.ChatRequest{
				Model:    agent.ModelConfig{Provider: provider, ModelID: modelID, Stream: true},
				Messages: msgs,
			}, func(ev model.ChatStreamEvent) error { return nil })
			if streamErr != nil {
				res.Stream = CapabilityResult{Status: "error", Error: streamErr.Error()}
			} else {
				res.Stream = CapabilityResult{Status: "success"}
			}
		}
	}

	if clients.Embed == nil {
		res.Embed = CapabilityResult{Status: "skipped", Error: "embedding unavailable"}
	} else {
		modelID := ResolveModel(provider, true)
		if modelID == "" {
			res.Embed = CapabilityResult{Status: "skipped", Error: "embedding model not configured"}
		} else {
			_, err := clients.Embed.Embed(ctx, model.EmbeddingRequest{
				Model: agent.ModelConfig{Provider: provider, ModelID: modelID},
				Input: []string{"Go provider demo embedding sample."},
			})
			if err != nil {
				res.Embed = CapabilityResult{Status: "error", Error: err.Error()}
			} else {
				res.Embed = CapabilityResult{Status: "success"}
			}
		}
	}
	return res
}

// ResolveModel returns the configured or default model ID for the provider.
func ResolveModel(provider agent.Provider, embedding bool) string {
	info := model.ProviderMetadata(provider)
	envSuffix := "CHAT_MODEL"
	defaultModel := info.DefaultChatModel
	if embedding {
		envSuffix = "EMBED_MODEL"
		defaultModel = info.DefaultEmbedModel
	}
	envKey := fmt.Sprintf("%s_%s", strings.ToUpper(string(provider)), envSuffix)
	if val := strings.TrimSpace(os.Getenv(envKey)); val != "" {
		return val
	}
	return defaultModel
}

// DemoTimeout returns a reasonable per-provider timeout derived from config.
func DemoTimeout(cfg *runtimeconfig.Config) time.Duration {
	if cfg.Runtime.Router.ProviderTimeout > 0 {
		return cfg.Runtime.Router.ProviderTimeout
	}
	return 30 * time.Second
}

// WriteDemoLog persists demo results to the given path.
func WriteDemoLog(path string, results []DemoResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	ts := time.Now().UTC().Format(time.RFC3339)
	for _, res := range results {
		line := fmt.Sprintf("%s provider=%s chat=%s stream=%s embed=%s reason=%s\n",
			ts, res.Provider, res.Chat.Status, res.Stream.Status, res.Embed.Status, sanitize(res.Reason))
		if _, err := f.WriteString(line); err != nil {
			return err
		}
	}
	return nil
}

func sanitize(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}
