package bench

import (
	"context"
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
	report := fmt.Sprintf("bench basics samples=%d p95=%s peak_alloc=%d bytes\n", len(durations), p95, atomic.LoadUint64(&peakAlloc))
	outPath := filepath.Join(base, "specs", "001-agno-agents-refactor", "artifacts", "bench.txt")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir bench artifacts: %v", err)
	}
	if err := os.WriteFile(outPath, []byte(report), 0o644); err != nil {
		t.Fatalf("write bench report: %v", err)
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
