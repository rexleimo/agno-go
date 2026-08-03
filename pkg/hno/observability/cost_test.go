package observability

import (
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

func TestEstimate_KnownModel(t *testing.T) {
	// gpt-4o-mini: $0.15/M input, $0.60/M output.
	// 1000 input tokens + 500 output tokens.
	// gpt-4o-mini：输入 $0.15/M，输出 $0.60/M。
	// 1000 输入 token + 500 输出 token。
	cost, ok := Estimate("openai", "gpt-4o-mini", types.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
	})
	if !ok {
		t.Fatal("expected price lookup to succeed")
	}
	// (0.15 * 1000 + 0.60 * 500) / 1e6 = (150 + 300) / 1e6 = 0.00045
	if cost < 0.00044 || cost > 0.00046 {
		t.Errorf("cost = %f, want ~0.00045", cost)
	}
}

func TestEstimate_PrefixFallback(t *testing.T) {
	// "gpt-4o-2024-11-20" should fall back to the "gpt-4o" entry.
	// "gpt-4o-2024-11-20" 应回退到 "gpt-4o" 条目。
	cost, ok := Estimate("openai", "gpt-4o-2024-11-20", types.Usage{
		PromptTokens:     1000,
		CompletionTokens: 0,
	})
	if !ok {
		t.Fatal("expected prefix fallback to succeed")
	}
	if cost != 0.0025 {
		t.Errorf("cost = %f, want 0.0025 (2.50/M * 1000 / 1e6)", cost)
	}
}

func TestEstimate_UnknownModel(t *testing.T) {
	_, ok := Estimate("openai", "totally-unknown-model", types.Usage{})
	if ok {
		t.Error("expected unknown model to report ok=false")
	}
}

func TestEstimate_UnknownProvider(t *testing.T) {
	_, ok := Estimate("not-a-provider", "gpt-4o", types.Usage{})
	if ok {
		t.Error("expected unknown provider to report ok=false")
	}
}

func TestEstimate_Anthropic(t *testing.T) {
	// claude-3-5-sonnet: $3/M in, $15/M out.
	cost, ok := Estimate("anthropic", "claude-3-5-sonnet", types.Usage{
		PromptTokens:     1_000_000,
		CompletionTokens: 1_000_000,
	})
	if !ok {
		t.Fatal("expected lookup to succeed")
	}
	if cost != 18.0 {
		t.Errorf("cost = %f, want 18.0", cost)
	}
}

func TestEstimate_OllamaFree(t *testing.T) {
	// Local inference costs nothing.
	// 本地推理无成本。
	cost, ok := Estimate("ollama", "qwen3-4b", types.Usage{
		PromptTokens:     1000,
		CompletionTokens: 1000,
	})
	if !ok {
		t.Fatal("expected ollama lookup to succeed")
	}
	if cost != 0 {
		t.Errorf("cost = %f, want 0", cost)
	}
}
