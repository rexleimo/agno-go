package contract_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rexleimo/agno-go/internal/agent"
	"github.com/rexleimo/agno-go/internal/knowledge"
	"github.com/rexleimo/agno-go/internal/model"
	runtimepkg "github.com/rexleimo/agno-go/internal/runtime"
	"github.com/rexleimo/agno-go/pkg/memory"
	"github.com/rexleimo/agno-go/pkg/providers/stub"
	"gopkg.in/yaml.v3"
)

// basicsContract is a minimal shape used to sanity-check fixtures.
type basicsContract struct {
	FixtureID string `json:"fixtureId" yaml:"fixtureId"`
	Provider  string `json:"provider" yaml:"provider"`
	Type      string `json:"type" yaml:"type"`
}

// TestBasicsContracts loads fixtures (if present) and records a summary for coverage aggregation.
func TestBasicsContracts(t *testing.T) {
	base := repoRoot(t)
	fixturesDir := filepath.Join(base, "specs", "001-agno-agents-refactor", "contracts", "fixtures")
	coverageLog := filepath.Join(base, "specs", "001-agno-agents-refactor", "artifacts", "coverage.txt")
	deviations := filepath.Join(base, "specs", "001-agno-agents-refactor", "contracts", "deviations.md")

	files, err := collectFixtureFiles(fixturesDir)
	if errors.Is(err, os.ErrNotExist) {
		t.Skipf("fixtures dir missing: %s", fixturesDir)
	}
	if err != nil {
		t.Fatalf("collect fixtures: %v", err)
	}
	if len(files) == 0 {
		t.Skipf("no fixtures found in %s", fixturesDir)
	}

	var ok, failed int
	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			fx, err := loadBasicsFixture(path)
			if err != nil {
				failed++
				appendDeviation(deviations, "fixture %s parse error: %v", path, err)
				t.Fatalf("parse fixture: %v", err)
			}
			if fx.FixtureID == "" || fx.Type == "" {
				failed++
				appendDeviation(deviations, "fixture %s missing required fields", path)
				t.Fatalf("fixture missing required fields: %+v", fx)
			}
			ok++
		})
	}

	summary := []byte(
		fmt.Sprintf("contracts: total=%d ok=%d failed=%d\n", len(files), ok, failed),
	)
	if err := os.MkdirAll(filepath.Dir(coverageLog), 0o755); err == nil {
		_ = appendFile(coverageLog, summary)
	}
}

// Lightweight smoke subtests to exercise Basics scenarios with stub provider.
func TestBasicsScenariosWithStub(t *testing.T) {
	base := repoRoot(t)
	coverageLog := filepath.Join(base, "specs", "001-agno-agents-refactor", "artifacts", "coverage.txt")

	scenarios := map[string]func(t *testing.T){
		"agent_basic":        testBasicAgent,
		"memory_session":     testMemorySession,
		"rag_fallback":       testRAGFallback,
		"tool_hitl_guarded":  testGuardrailHook,
		"workflow_branching": func(t *testing.T) { t.Skip("workflow branching placeholder; awaiting workflow engine wiring") },
	}

	for name, fn := range scenarios {
		name := name
		fn := fn
		t.Run(name, func(t *testing.T) {
			status := "ok"
			t.Cleanup(func() {
				switch {
				case t.Skipped():
					status = "skipped"
				case t.Failed():
					status = "failed"
				}
				_ = appendFile(coverageLog, []byte(fmt.Sprintf("basics_scenario=%s status=%s\n", name, status)))
			})
			fn(t)
		})
	}
}

func testBasicAgent(t *testing.T) {
	svc, _, agentID, sessionID := buildStubService(t)
	resp, err := svc.PostMessage(context.Background(), agentID, sessionID, runtimepkg.MessageRequest{
		Messages: []agent.Message{{Role: agent.RoleUser, Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("post message: %v", err)
	}
	if !strings.HasPrefix(resp.Message.Content, "echo:") {
		t.Fatalf("unexpected response: %s", resp.Message.Content)
	}
}

func testMemorySession(t *testing.T) {
	svc, store, agentID, sessionID := buildStubService(t)
	_, _ = svc.PostMessage(context.Background(), agentID, sessionID, runtimepkg.MessageRequest{
		Messages: []agent.Message{{Role: agent.RoleUser, Content: "remember this"}},
	})
	_, _ = svc.PostMessage(context.Background(), agentID, sessionID, runtimepkg.MessageRequest{
		Messages: []agent.Message{{Role: agent.RoleUser, Content: "second turn"}},
	})
	history, err := store.LoadHistory(context.Background(), agentID, sessionID, agent.HistoryOptions{TokenWindow: 0})
	if err != nil {
		t.Fatalf("load history: %v", err)
	}
	if len(history) < 2 {
		t.Fatalf("expected history >=2, got %d", len(history))
	}
}

func testRAGFallback(t *testing.T) {
	msg, err := knowledge.HandleUnavailable(knowledge.Strategy{
		Mode:        knowledge.ModeHint,
		HintMessage: "Fallback: knowledge unavailable",
	}, "vector store offline")
	if err != nil {
		t.Fatalf("fallback hint returned error: %v", err)
	}
	if !strings.Contains(msg, "Fallback") {
		t.Fatalf("unexpected fallback message: %s", msg)
	}
}

func testGuardrailHook(t *testing.T) {
	// Placeholder to assert that guardrail hook can be invoked; using stub router to avoid real detection.
	svc, _, agentID, sessionID := buildStubService(t)
	_, err := svc.PostMessage(context.Background(), agentID, sessionID, runtimepkg.MessageRequest{
		Messages: []agent.Message{{Role: agent.RoleUser, Content: "detect pii 123-45-6789"}},
	})
	if err != nil {
		t.Fatalf("guardrail path failed: %v", err)
	}
}

func buildStubService(t *testing.T) (*runtimepkg.Service, agent.Store, uuid.UUID, uuid.UUID) {
	t.Helper()
	router := model.NewRouter()
	router.RegisterChatProvider(stub.New(agent.ProviderOpenAI, model.ProviderAvailable, nil))
	store := memory.NewInMemoryStore()
	svc := runtimepkg.NewService(store, router)
	agentID, err := svc.CreateAgent(context.Background(), agent.Agent{
		Name: "basics-stub",
		Model: agent.ModelConfig{
			Provider: agent.ProviderOpenAI,
			ModelID:  "stub-basic",
			Stream:   false,
		},
	})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	session, err := svc.CreateSession(context.Background(), agentID, "user-1", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	return svc, store, agentID, session.ID
}

func loadBasicsFixture(path string) (*basicsContract, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var fx basicsContract
	switch ext := filepath.Ext(path); ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(raw, &fx)
	default:
		err = json.Unmarshal(raw, &fx)
	}
	if err != nil {
		return nil, err
	}
	return &fx, nil
}

func appendFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = f.Write(data)
	return err
}

func appendDeviation(path, format string, args ...interface{}) {
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	msg := fmt.Sprintf("- [contract] %s: %s\n", time.Now().UTC().Format(time.RFC3339), fmt.Sprintf(format, args...))
	_ = appendFile(path, []byte(msg))
}

func repoRoot(tb testing.TB) string {
	tb.Helper()
	_, file, _, ok := goruntime.Caller(0)
	if !ok {
		tb.Fatalf("cannot resolve caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
