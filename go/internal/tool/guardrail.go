package tool

import "strings"

// GuardrailConfig holds toggles for PII/Prompt injection detection.
type GuardrailConfig struct {
	EnablePII       bool
	EnableInjection bool
}

// ShouldBlock returns true if content trips configured guardrails.
func ShouldBlock(cfg GuardrailConfig, content string) bool {
	if cfg.EnablePII && GuardrailHook(content) != nil {
		return true
	}
	if cfg.EnableInjection && injectionLike(content) {
		return true
	}
	return false
}

func injectionLike(content string) bool {
	lower := strings.ToLower(content)
	return strings.Contains(lower, "ignore previous") || strings.Contains(lower, "system:") || strings.Contains(lower, "jailbreak")
}
