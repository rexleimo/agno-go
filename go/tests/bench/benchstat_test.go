package bench

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/perf/benchstat" //nolint:staticcheck // benchstat deprecated upstream; retained per specs
)

func TestBenchstatReport(t *testing.T) {
	output := firstNonEmpty(os.Getenv("BENCH_OUTPUT"), filepath.Join("..", "..", "..", "specs", "001-agno-agents-refactor", "artifacts", "bench.txt"))
	baselineDefault := filepath.Join("..", "..", "..", "specs", "001-agno-agents-refactor", "artifacts", "baseline", "python-bench.json")
	if _, err := os.Stat(output); err != nil {
		t.Skipf("bench output not found, run make bench to generate: %v", err)
	}
	baseline := firstNonEmpty(os.Getenv("BENCH_BASELINE"), baselineDefault)
	paths := []string{output}
	if baseline != "" {
		if filepath.Ext(baseline) == ".json" {
			t.Logf("baseline is JSON; benchstat compare skipped: %s", baseline)
		} else if info, err := os.Stat(baseline); err == nil && info.Size() > 0 {
			paths = append([]string{baseline}, paths...)
		} else if err != nil && !errorsIsNotExist(err) {
			t.Fatalf("stat baseline: %v", err)
		}
	}

	var c benchstat.Collection
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("open %s: %v", path, err)
		}
		if err := c.AddFile(filepath.Base(path), f); err != nil {
			t.Fatalf("add bench file %s: %v", path, err)
		}
		defer func() { _ = f.Close() }()
	}

	tables := c.Tables()
	var buf bytes.Buffer
	benchstat.FormatText(&buf, tables)

	dest := filepath.Join("..", "..", "..", "specs", "001-agno-agents-refactor", "artifacts", "benchstat.txt")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatalf("mkdir bench dir: %v", err)
	}
	if err := os.WriteFile(dest, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write benchstat report: %v", err)
	}

	report := buildReport(output, baseline, buf.String())
	reportPath := filepath.Join("..", "..", "..", "specs", "001-agno-agents-refactor", "artifacts", "bench", "report.md")
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		t.Fatalf("mkdir bench report dir: %v", err)
	}
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		t.Fatalf("write bench report: %v", err)
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func buildReport(output, baseline, benchstatText string) string {
	var b strings.Builder
	b.WriteString("# Benchmark Summary\n\n")
	b.WriteString(fmt.Sprintf("- output: %s\n", output))
	if strings.TrimSpace(baseline) == "" {
		b.WriteString("- baseline: (not provided)\n")
	} else if _, err := os.Stat(baseline); err == nil {
		b.WriteString(fmt.Sprintf("- baseline: %s\n", baseline))
	} else {
		b.WriteString("- baseline: (not provided)\n")
	}
	b.WriteString(fmt.Sprintf("- generated: %s\n\n", time.Now().Format(time.RFC3339)))
	b.WriteString("```text\n")
	b.WriteString(benchstatText)
	b.WriteString("\n```\n")
	return b.String()
}

func errorsIsNotExist(err error) bool {
	return err != nil && os.IsNotExist(err)
}
