package providers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rexleimo/agno-go/internal/agent"
	runtimeconfig "github.com/rexleimo/agno-go/internal/runtime/config"
	runtimeproviders "github.com/rexleimo/agno-go/internal/runtime/providers"
)

// Providers integration smoke: exercises chat/stream/embed for configured providers and writes coverage logs.
func TestProvidersIntegrationReport(t *testing.T) {
	base := providersRepoRoot(t)
	cfgPath := filepath.Join(base, "config", "default.yaml")
	envPath := filepath.Join(base, ".env")
	cfg, err := runtimeconfig.LoadWithEnv(cfgPath, envPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	statuses := cfg.ProviderStatuses()
	configs := cfg.ProviderConfigs()
	logPath := filepath.Join(base, "specs", "001-agno-agents-refactor", "artifacts", "coverage", "providers.log")
	summaryPath := filepath.Join(base, "specs", "001-agno-agents-refactor", "artifacts", "coverage.txt")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatalf("mkdir providers log: %v", err)
	}

	timeout := runtimeproviders.DemoTimeout(cfg)
	results := make([]runtimeproviders.DemoResult, 0, len(statuses))
	statusMap := make(map[agent.Provider]runtimeconfig.ProviderConfig, len(configs))
	for provider, entry := range configs {
		statusMap[provider] = entry
	}
	for _, st := range statuses {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		result := runtimeproviders.Exercise(ctx, cfg, st.Provider, st, statusMap[st.Provider])
		cancel()
		results = append(results, result)
	}

	if err := runtimeproviders.WriteDemoLog(logPath, results); err != nil {
		t.Fatalf("write providers log: %v", err)
	}

	var success, skipped, failures int
	for _, res := range results {
		if res.Chat.Status == "success" || res.Stream.Status == "success" || res.Embed.Status == "success" {
			success++
		}
		if res.Chat.Status == "skipped" && res.Stream.Status == "skipped" && res.Embed.Status == "skipped" {
			skipped++
		}
		if hasError(res) {
			failures++
		}
	}

	summary := fmt.Sprintf("providers-demo: success=%d skipped=%d errors=%d\n", success, skipped, failures)
	_ = appendFile(summaryPath, []byte(summary))
}

func hasError(res runtimeproviders.DemoResult) bool {
	return res.Chat.Status == "error" || res.Stream.Status == "error" || res.Embed.Status == "error"
}

func providersRepoRoot(tb testing.TB) string {
	tb.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		tb.Fatalf("cannot resolve caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func appendFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = f.Write(data)
	return err
}
