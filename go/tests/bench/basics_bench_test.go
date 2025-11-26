package bench

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/internal/agent"
	"github.com/rexleimo/agno-go/internal/model"
	"github.com/rexleimo/agno-go/internal/runtime"
	runtimeconfig "github.com/rexleimo/agno-go/internal/runtime/config"
	"github.com/rexleimo/agno-go/pkg/memory"
	"github.com/rexleimo/agno-go/pkg/providers/stub"
)

// TestBasicsBenchMetrics simulates a small load using stub providers and records simple p95 + peak memory metrics.
func TestBasicsBenchMetrics(t *testing.T) {
	base := repoRoot(t)
	cfg, err := runtimeconfig.LoadWithEnv(filepath.Join(base, "config", "default.yaml"), "")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	benchCfg := cfg.Bench
	if benchCfg.Concurrency <= 0 {
		benchCfg.Concurrency = 4
	}
	if benchCfg.Duration <= 0 {
		benchCfg.Duration = 2 * time.Second
	}
	if benchCfg.InputTokens <= 0 {
		benchCfg.InputTokens = 32
	}
	// Keep CI runtime sane by capping duration.
	if benchCfg.Duration > 5*time.Second {
		benchCfg.Duration = 5 * time.Second
	}

	router := model.NewRouter()
	router.RegisterChatProvider(stub.New(agent.ProviderOpenAI, model.ProviderAvailable, nil))
	store := memory.NewInMemoryStore()
	svc := runtime.NewService(store, router)

	ctx, cancel := context.WithTimeout(context.Background(), benchCfg.Duration)
	defer cancel()

	agentID, err := svc.CreateAgent(ctx, agent.Agent{
		Name: "bench-stub",
		Model: agent.ModelConfig{
			Provider: agent.ProviderOpenAI,
			ModelID:  "stub-bench",
			Stream:   false,
		},
		Memory: agent.MemoryConfig{TokenWindow: benchCfg.InputTokens * 2},
	})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	session, err := svc.CreateSession(ctx, agentID, "bench-user", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	var durations []time.Duration
	var mu sync.Mutex
	var peakAlloc uint64

	runOnce := func(worker int) {
		defer func() {
			var m goruntime.MemStats
			goruntime.ReadMemStats(&m)
			atomic.StoreUint64(&peakAlloc, maxUint64(atomic.LoadUint64(&peakAlloc), m.Alloc))
		}()
		req := runtime.MessageRequest{
			Messages: []agent.Message{{Role: agent.RoleUser, Content: strings.Repeat("x", benchCfg.InputTokens)}},
		}
		start := time.Now()
		_, err := svc.PostMessage(ctx, agentID, session.ID, req)
		dur := time.Since(start)
		mu.Lock()
		durations = append(durations, dur)
		mu.Unlock()
		if err != nil {
			t.Logf("worker=%d error: %v", worker, err)
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < benchCfg.Concurrency; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for time.Now().Before(time.Now().Add(benchCfg.Duration / 2)) {
				select {
				case <-ctx.Done():
					return
				default:
					runOnce(worker)
				}
			}
		}(i)
	}
	wg.Wait()

	if len(durations) == 0 {
		t.Fatalf("no benchmark samples recorded")
	}

	p95 := percentile(durations, 0.95)
	peakBytes := atomic.LoadUint64(&peakAlloc)
	measured := benchMetrics{
		Samples:   len(durations),
		P95:       p95,
		P95Ms:     float64(p95) / float64(time.Millisecond),
		PeakAlloc: peakBytes,
	}
	report := fmt.Sprintf("bench basics samples=%d p95_ms=%.2f peak_alloc=%d bytes\n", measured.Samples, measured.P95Ms, measured.PeakAlloc)
	outPath := filepath.Join(base, "specs", "001-agno-agents-refactor", "artifacts", "bench.txt")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir bench artifacts: %v", err)
	}
	if err := os.WriteFile(outPath, []byte(report), 0o644); err != nil {
		t.Fatalf("write bench report: %v", err)
	}
	if err := evaluateBaseline(base, measured); err != nil {
		t.Fatalf("baseline comparison failed: %v", err)
	}
}

func percentile(durations []time.Duration, p float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	idx := int(float64(len(durations)-1) * p)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(durations) {
		idx = len(durations) - 1
	}
	return durations[idx]
}

func maxUint64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

type benchMetrics struct {
	Samples   int
	P95       time.Duration
	P95Ms     float64
	PeakAlloc uint64
}

type benchBaseline struct {
	P95Ms          float64 `json:"p95_ms"`
	PeakAllocBytes uint64  `json:"peak_alloc_bytes"`
}

func evaluateBaseline(base string, measured benchMetrics) error {
	baselinePath := filepath.Join(base, "specs", "001-agno-agents-refactor", "artifacts", "baseline", "python-bench.json")
	deviations := filepath.Join(base, "specs", "001-agno-agents-refactor", "contracts", "deviations.md")
	baseline, err := loadBaseline(baselinePath)
	if err != nil {
		appendDeviation(deviations, fmt.Sprintf("bench baseline missing or unreadable: %v", err))
		return nil
	}
	if baseline.P95Ms <= 0 || baseline.PeakAllocBytes == 0 {
		appendDeviation(deviations, "bench baseline missing required metrics; capture python baseline and rerun (owner=tbd)")
		return nil
	}
	p95Reduction := (baseline.P95Ms - measured.P95Ms) / baseline.P95Ms
	if p95Reduction < 0 {
		p95Reduction = 0
	}
	peakReduction := float64(baseline.PeakAllocBytes-measured.PeakAlloc) / float64(baseline.PeakAllocBytes)
	if peakReduction < 0 {
		peakReduction = 0
	}

	var regressions []string
	if p95Reduction < 0.20 {
		regressions = append(regressions, fmt.Sprintf("p95 improvement %.2f%% < target 20%%", p95Reduction*100))
	}
	if peakReduction < 0.25 {
		regressions = append(regressions, fmt.Sprintf("peak alloc improvement %.2f%% < target 25%%", peakReduction*100))
	}
	if len(regressions) > 0 {
		msg := fmt.Sprintf("bench regression: %s (owner=runtime, next=optimize router/memory)", strings.Join(regressions, "; "))
		appendDeviation(deviations, msg)
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func loadBaseline(path string) (benchBaseline, error) {
	var baseline benchBaseline
	data, err := os.ReadFile(path)
	if err != nil {
		return baseline, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return baseline, fmt.Errorf("baseline file empty")
	}
	if err := json.Unmarshal(data, &baseline); err != nil {
		return benchBaseline{}, err
	}
	return baseline, nil
}

func appendDeviation(path, msg string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	ts := time.Now().UTC().Format(time.RFC3339)
	_, _ = fmt.Fprintf(f, "- [bench] %s: %s\n", ts, msg)
}
