// Package cost estimates LLM call costs from provider price tables.
// Prices are per 1M tokens in USD, keyed by provider and model name.
//
// cost 包根据 provider 价格表估算 LLM 调用成本。
// 价格为每 100 万 token 的美元数，按 provider 和模型名索引。
package observability

import (
	"strings"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// Price holds per-1M-token prices in USD.
// Price 保存每 100 万 token 的美元价格。
type Price struct {
	Input  float64
	Output float64
}

// Estimate returns the USD cost of a call for the given provider/model.
// Unknown models return 0 (no data) with ok=false.
//
// Estimate 返回给定 provider/模型的调用美元成本。
// 未知模型返回 0（无数据）且 ok=false。
func Estimate(provider, model string, usage types.Usage) (float64, bool) {
	p, ok := lookup(provider, model)
	if !ok {
		return 0, false
	}
	return (p.Input*float64(usage.PromptTokens) + p.Output*float64(usage.CompletionTokens)) / 1_000_000, true
}

// lookup resolves a price, with model-prefix fallback (e.g. "gpt-4o-mini"
// matches a "gpt-4o" entry). Local providers (ollama) always cost zero.
//
// lookup 解析价格，支持模型前缀回退（如 "gpt-4o-mini" 匹配 "gpt-4o" 条目）。
// 本地 provider（ollama）始终为零成本。
func lookup(provider, model string) (Price, bool) {
	p := strings.ToLower(provider)
	// Local inference: no observability.
	// 本地推理：无成本。
	if p == "ollama" || p == "lmstudio" {
		return Price{}, true
	}
	models := tables[p]
	if models == nil {
		return Price{}, false
	}
	m := strings.ToLower(model)
	if p, ok := models[m]; ok {
		return p, true
	}
	// Prefix fallback: longest matching key.
	// 前缀回退：匹配最长键。
	best := ""
	var bestP Price
	for key, p := range models {
		if strings.HasPrefix(m, key) && len(key) > len(best) {
			best = key
			bestP = p
		}
	}
	if best != "" {
		return bestP, true
	}
	return Price{}, false
}

// tables holds per-provider price maps (per 1M tokens, USD).
// tables 保存各 provider 的价格表（每 100 万 token，美元）。
var tables = map[string]map[string]Price{
	"openai": {
		"gpt-4o":        {Input: 2.50, Output: 10.00},
		"gpt-4o-mini":   {Input: 0.15, Output: 0.60},
		"gpt-4":         {Input: 30.00, Output: 60.00},
		"gpt-3.5-turbo": {Input: 0.50, Output: 1.50},
		"o1":            {Input: 15.00, Output: 60.00},
		"o3":            {Input: 2.00, Output: 8.00},
	},
	"anthropic": {
		"claude-3-5-sonnet": {Input: 3.00, Output: 15.00},
		"claude-3-5-haiku":  {Input: 0.80, Output: 4.00},
		"claude-3-opus":     {Input: 15.00, Output: 75.00},
		"claude-sonnet":     {Input: 3.00, Output: 15.00},
	},
	"gemini": {
		"gemini-1.5-pro":   {Input: 1.25, Output: 5.00},
		"gemini-1.5-flash": {Input: 0.075, Output: 0.30},
		"gemini-2.0-flash": {Input: 0.10, Output: 0.40},
	},
	"deepseek": {
		"deepseek-chat":     {Input: 0.27, Output: 1.10},
		"deepseek-reasoner": {Input: 0.55, Output: 2.19},
	},
	"groq": {
		"llama-3":   {Input: 0.59, Output: 0.79},
		"mixtral":   {Input: 0.24, Output: 0.24},
		"llama-3.1": {Input: 0.59, Output: 0.79},
	},
	"together": {
		"llama-3": {Input: 0.18, Output: 0.18},
		"qwen2":   {Input: 0.18, Output: 0.18},
	},
	"ollama": {
		// Local inference: no observability.
		// 本地推理：无成本。
		"": {Input: 0, Output: 0},
	},
}
