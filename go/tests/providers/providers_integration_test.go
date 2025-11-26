package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rexleimo/agno-go/internal/model"
	runtimeconfig "github.com/rexleimo/agno-go/internal/runtime/config"
)

// Providers integration smoke: logs availability/skip reasons and writes summary for coverage aggregation.
func TestProvidersIntegrationReport(t *testing.T) {
	base := providersRepoRoot(t)
	cfgPath := filepath.Join(base, "config", "default.yaml")
	envPath := filepath.Join(base, ".env")
	cfg, err := runtimeconfig.LoadWithEnv(cfgPath, envPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	statuses := cfg.ProviderStatuses()
	logPath := filepath.Join(base, "specs", "001-agno-agents-refactor", "artifacts", "coverage", "providers.log")
	summaryPath := filepath.Join(base, "specs", "001-agno-agents-refactor", "artifacts", "coverage.txt")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatalf("mkdir providers log: %v", err)
	}

	var available, skipped, errorsCount int
	var buf strings.Builder

	for _, st := range statuses {
		if st.Status != model.ProviderAvailable {
			skipped++
			buf.WriteString(fmt.Sprintf("provider=%s status=skipped reason=%s missing=%v\n", st.Provider, st.Status, st.MissingEnv))
			continue
		}
		available++
		buf.WriteString(fmt.Sprintf("provider=%s status=available (connectivity not executed in smoke)\n", st.Provider))
	}

	if buf.Len() == 0 {
		buf.WriteString("no providers configured\n")
	}

	if err := os.WriteFile(logPath, []byte(buf.String()), 0o644); err != nil {
		t.Fatalf("write providers log: %v", err)
	}

	summary := fmt.Sprintf("providers: available=%d skipped=%d errors=%d\n", available, skipped, errorsCount)
	_ = appendFile(summaryPath, []byte(summary))
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
