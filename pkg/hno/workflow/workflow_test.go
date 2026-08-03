package workflow

import (
	"context"
	"strings"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/media"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// MockModel for testing
type MockModel struct {
	models.BaseModel
	InvokeFunc func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error)
}

func (m *MockModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	if m.InvokeFunc != nil {
		return m.InvokeFunc(ctx, req)
	}
	return &types.ModelResponse{
		ID:      "test-response",
		Content: "mock response",
		Model:   "test",
	}, nil
}

func (m *MockModel) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	return nil, nil
}

func createMockAgent(id string, responseContent string) *agent.Agent {
	model := &MockModel{
		BaseModel: models.BaseModel{ID: id, Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			return &types.ModelResponse{
				ID:      "test-" + id,
				Content: responseContent,
				Model:   id,
			}, nil
		},
	}

	ag, _ := agent.New(agent.Config{
		ID:    id,
		Name:  "Agent " + id,
		Model: model,
	})

	return ag
}

type stubNode struct {
	id       string
	nodeType NodeType
	execute  func(context.Context, *ExecutionContext) (*ExecutionContext, error)
}

func (n *stubNode) Execute(ctx context.Context, execCtx *ExecutionContext) (*ExecutionContext, error) {
	if n.execute != nil {
		return n.execute(ctx, execCtx)
	}
	return execCtx, nil
}

func (n *stubNode) GetID() string {
	return n.id
}

func (n *stubNode) GetType() NodeType {
	if n.nodeType != "" {
		return n.nodeType
	}
	return NodeTypeStep
}

func TestNew(t *testing.T) {
	step1, _ := NewStep(StepConfig{
		ID:    "step1",
		Agent: createMockAgent("a1", "output1"),
	})

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid workflow",
			config: Config{
				Name:  "test-workflow",
				Steps: []Node{step1},
			},
			wantErr: false,
		},
		{
			name: "empty workflow",
			config: Config{
				Name:  "empty",
				Steps: []Node{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && wf == nil {
				t.Error("New() returned nil workflow")
			}
		})
	}
}

func TestWorkflow_Run(t *testing.T) {
	step1, _ := NewStep(StepConfig{
		ID:    "step1",
		Agent: createMockAgent("a1", "step1 complete"),
	})

	step2, _ := NewStep(StepConfig{
		ID:    "step2",
		Agent: createMockAgent("a2", "step2 complete"),
	})

	wf, _ := New(Config{
		Name:  "sequential-workflow",
		Steps: []Node{step1, step2},
	})

	result, err := wf.Run(context.Background(), "start", "")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result == nil {
		t.Fatal("Run() returned nil result")
	}

	if result.Output != "step2 complete" {
		t.Errorf("Final output = %v, want 'step2 complete'", result.Output)
	}

	// Check that step outputs are stored
	if _, exists := result.Get("step_step1_output"); !exists {
		t.Error("step1 output not stored in context")
	}

	if _, exists := result.Get("step_step2_output"); !exists {
		t.Error("step2 output not stored in context")
	}

	metrics, ok := result.Metadata["workflow_metrics"].(map[string]interface{})
	if !ok {
		t.Fatal("workflow metrics metadata missing")
	}
	if _, ok := metrics["duration_seconds"]; !ok {
		t.Fatal("workflow metrics missing duration_seconds")
	}
}

func TestWorkflow_RunStoresEvents(t *testing.T) {
	step, _ := NewStep(StepConfig{
		ID:    "event-step",
		Agent: createMockAgent("agent-evt", "streamed output"),
	})

	store := NewMemoryStorage(10)
	wf, err := New(Config{
		ID:            "workflow-events",
		Name:          "workflow-events",
		Steps:         []Node{step},
		EnableHistory: true,
		HistoryStore:  store,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()
	if _, err := wf.Run(ctx, "start", "session-events"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	session, err := store.GetSession(ctx, "session-events")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	runs := session.GetRuns()
	if len(runs) == 0 {
		t.Fatalf("expected workflow run to be persisted")
	}
	if len(runs[0].Events) == 0 {
		t.Fatalf("expected workflow run to store events")
	}
}

func TestWorkflow_RunEmptyInput(t *testing.T) {
	step1, _ := NewStep(StepConfig{
		ID:    "step1",
		Agent: createMockAgent("a1", "output"),
	})

	wf, _ := New(Config{
		Name:  "test",
		Steps: []Node{step1},
	})

	_, err := wf.Run(context.Background(), "", "")
	if err == nil {
		t.Error("Run() with empty input should return error")
	}
}

func TestStep_Execute(t *testing.T) {
	agent := createMockAgent("test", "processed")
	step, err := NewStep(StepConfig{
		ID:    "test-step",
		Agent: agent,
	})

	if err != nil {
		t.Fatalf("NewStep() error = %v", err)
	}

	execCtx := NewExecutionContext("input")
	result, err := step.Execute(context.Background(), execCtx)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.Output != "processed" {
		t.Errorf("Output = %v, want 'processed'", result.Output)
	}

	if step.GetType() != NodeTypeStep {
		t.Errorf("GetType() = %v, want step", step.GetType())
	}
}

func TestCondition_Execute(t *testing.T) {
	trueAgent := createMockAgent("true", "true branch")
	falseAgent := createMockAgent("false", "false branch")

	trueStep, _ := NewStep(StepConfig{ID: "true-step", Agent: trueAgent})
	falseStep, _ := NewStep(StepConfig{ID: "false-step", Agent: falseAgent})

	// Test true condition
	cond, _ := NewCondition(ConditionConfig{
		ID: "test-condition",
		Condition: func(ctx *ExecutionContext) bool {
			return ctx.Output == "go-true"
		},
		TrueNode:  trueStep,
		FalseNode: falseStep,
	})

	execCtx := NewExecutionContext("input")
	execCtx.Output = "go-true"

	result, err := cond.Execute(context.Background(), execCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.Output != "true branch" {
		t.Errorf("True condition output = %v, want 'true branch'", result.Output)
	}

	// Test false condition
	execCtx2 := NewExecutionContext("input")
	execCtx2.Output = "go-false"

	result2, err := cond.Execute(context.Background(), execCtx2)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result2.Output != "false branch" {
		t.Errorf("False condition output = %v, want 'false branch'", result2.Output)
	}

	if cond.GetType() != NodeTypeCondition {
		t.Errorf("GetType() = %v, want condition", cond.GetType())
	}
}

func TestLoop_Execute(t *testing.T) {
	loopAgent := createMockAgent("loop", "iteration")
	loopStep, _ := NewStep(StepConfig{ID: "loop-step", Agent: loopAgent})

	loop, err := NewLoop(LoopConfig{
		ID:   "test-loop",
		Body: loopStep,
		Condition: func(ctx *ExecutionContext, iteration int) bool {
			return iteration < 3 // Loop 3 times
		},
		MaxIteration: 5,
	})

	if err != nil {
		t.Fatalf("NewLoop() error = %v", err)
	}

	execCtx := NewExecutionContext("start")
	result, err := loop.Execute(context.Background(), execCtx)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	iterations, exists := result.Get("loop_test-loop_iterations")
	if !exists {
		t.Fatal("Loop iterations not stored in context")
	}

	if iterations != 3 {
		t.Errorf("Iterations = %v, want 3", iterations)
	}

	if loop.GetType() != NodeTypeLoop {
		t.Errorf("GetType() = %v, want loop", loop.GetType())
	}
}

func TestParallel_Execute(t *testing.T) {
	agent1 := createMockAgent("a1", "parallel1")
	agent2 := createMockAgent("a2", "parallel2")

	step1, _ := NewStep(StepConfig{ID: "step1", Agent: agent1})
	step2, _ := NewStep(StepConfig{ID: "step2", Agent: agent2})

	parallel, err := NewParallel(ParallelConfig{
		ID:    "test-parallel",
		Nodes: []Node{step1, step2},
	})

	if err != nil {
		t.Fatalf("NewParallel() error = %v", err)
	}

	execCtx := NewExecutionContext("input")
	result, err := parallel.Execute(context.Background(), execCtx)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Check that both branches executed
	branch0, exists0 := result.Get("parallel_test-parallel_branch_0_output")
	if !exists0 {
		t.Error("Branch 0 output not found")
	}

	branch1, exists1 := result.Get("parallel_test-parallel_branch_1_output")
	if !exists1 {
		t.Error("Branch 1 output not found")
	}

	// Both outputs should be present
	if branch0 != "parallel1" {
		t.Errorf("Branch 0 output = %v, want 'parallel1'", branch0)
	}

	if branch1 != "parallel2" {
		t.Errorf("Branch 1 output = %v, want 'parallel2'", branch1)
	}

	if parallel.GetType() != NodeTypeParallel {
		t.Errorf("GetType() = %v, want parallel", parallel.GetType())
	}
}

func TestRouter_Execute(t *testing.T) {
	route1Agent := createMockAgent("r1", "route1 executed")
	route2Agent := createMockAgent("r2", "route2 executed")

	route1, _ := NewStep(StepConfig{ID: "route1", Agent: route1Agent})
	route2, _ := NewStep(StepConfig{ID: "route2", Agent: route2Agent})

	router, err := NewRouter(RouterConfig{
		ID: "test-router",
		Router: func(ctx *ExecutionContext) string {
			if strings.Contains(ctx.Output, "path1") {
				return "route1"
			}
			return "route2"
		},
		Routes: map[string]Node{
			"route1": route1,
			"route2": route2,
		},
	})

	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	// Test route1
	execCtx1 := NewExecutionContext("input")
	execCtx1.Output = "path1"

	result1, err := router.Execute(context.Background(), execCtx1)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result1.Output != "route1 executed" {
		t.Errorf("Route1 output = %v, want 'route1 executed'", result1.Output)
	}

	// Test route2
	execCtx2 := NewExecutionContext("input")
	execCtx2.Output = "path2"

	result2, err := router.Execute(context.Background(), execCtx2)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result2.Output != "route2 executed" {
		t.Errorf("Route2 output = %v, want 'route2 executed'", result2.Output)
	}

	if router.GetType() != NodeTypeRouter {
		t.Errorf("GetType() = %v, want router", router.GetType())
	}
}

func TestExecutionContext(t *testing.T) {
	ctx := NewExecutionContext("test input")

	if ctx.Input != "test input" {
		t.Errorf("Input = %v, want 'test input'", ctx.Input)
	}

	// Test Set/Get
	ctx.Set("key1", "value1")
	val, exists := ctx.Get("key1")

	if !exists {
		t.Error("Get() returned false for existing key")
	}

	if val != "value1" {
		t.Errorf("Get() = %v, want 'value1'", val)
	}

	// Test non-existent key
	_, exists = ctx.Get("nonexistent")
	if exists {
		t.Error("Get() returned true for non-existent key")
	}
}

func TestWorkflow_RunWithSessionStateInjection(t *testing.T) {
	store := NewMemoryStorage(10)

	inspectNode := &stubNode{
		id: "inspect",
		execute: func(_ context.Context, execCtx *ExecutionContext) (*ExecutionContext, error) {
			val, ok := execCtx.GetSessionState("progress")
			if !ok || val.(int) != 42 {
				t.Fatalf("expected session state progress=42, got %v (ok=%v)", val, ok)
			}
			execCtx.Output = "done"
			return execCtx, nil
		},
	}

	wf, err := New(Config{
		Name:          "inject",
		Steps:         []Node{inspectNode},
		EnableHistory: true,
		HistoryStore:  store,
	})
	if err != nil {
		t.Fatalf("failed to create workflow: %v", err)
	}

	execCtx, err := wf.Run(context.Background(), "ignored", "sess-inject",
		WithSessionState(map[string]interface{}{"progress": 42}),
		WithUserID("user-1"),
	)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if execCtx.Output != "done" {
		t.Fatalf("expected output 'done', got %s", execCtx.Output)
	}
}

func TestWorkflow_RunResumeFromStep(t *testing.T) {
	var executed []string

	nodeA := &stubNode{
		id: "step-a",
		execute: func(_ context.Context, execCtx *ExecutionContext) (*ExecutionContext, error) {
			executed = append(executed, "a")
			execCtx.Output = "a"
			return execCtx, nil
		},
	}

	nodeB := &stubNode{
		id: "step-b",
		execute: func(_ context.Context, execCtx *ExecutionContext) (*ExecutionContext, error) {
			executed = append(executed, "b")
			execCtx.Output = "b"
			return execCtx, nil
		},
	}

	nodeC := &stubNode{
		id: "step-c",
		execute: func(_ context.Context, execCtx *ExecutionContext) (*ExecutionContext, error) {
			executed = append(executed, "c")
			execCtx.Output = "c"
			return execCtx, nil
		},
	}

	wf, err := New(Config{
		Name:  "resume",
		Steps: []Node{nodeA, nodeB, nodeC},
	})
	if err != nil {
		t.Fatalf("failed to create workflow: %v", err)
	}

	execCtx, err := wf.Run(context.Background(), "start", "sess-resume", WithResumeFrom("step-b"))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if execCtx.Output != "c" {
		t.Fatalf("expected output 'c', got %s", execCtx.Output)
	}
	if len(executed) != 2 || executed[0] != "b" || executed[1] != "c" {
		t.Fatalf("expected execution order [b c], got %v", executed)
	}
}

func TestWorkflow_CancellationPersistence(t *testing.T) {
	store := NewMemoryStorage(10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cancelNode := &stubNode{
		id: "cancel-step",
		execute: func(_ context.Context, execCtx *ExecutionContext) (*ExecutionContext, error) {
			cancel()
			return execCtx, nil
		},
	}

	finalNode := &stubNode{
		id: "final-step",
		execute: func(_ context.Context, execCtx *ExecutionContext) (*ExecutionContext, error) {
			execCtx.Output = "finished"
			return execCtx, nil
		},
	}

	wf, err := New(Config{
		Name:          "cancel",
		Steps:         []Node{cancelNode, finalNode},
		EnableHistory: true,
		HistoryStore:  store,
	})
	if err != nil {
		t.Fatalf("failed to create workflow: %v", err)
	}

	sessionID := "sess-cancel"
	_, err = wf.Run(ctx, "start", sessionID)
	if err == nil {
		t.Fatalf("expected cancellation error")
	}

	session, err := store.GetSession(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}

	if len(session.Cancellations) != 1 {
		t.Fatalf("expected 1 cancellation record, got %d", len(session.Cancellations))
	}

	record := session.Cancellations[0]
	if record.StepID != "cancel-step" {
		t.Fatalf("expected step_id cancel-step, got %s", record.StepID)
	}
	if record.Reason != context.Canceled.Error() {
		t.Fatalf("expected reason context canceled, got %s", record.Reason)
	}
	if record.RunID == "" {
		t.Fatalf("expected run_id recorded")
	}

	runs := session.GetRuns()
	if len(runs) == 0 || runs[len(runs)-1].Status != RunStatusCancelled {
		t.Fatalf("expected last run cancelled, got %+v", runs)
	}
	if runs[len(runs)-1].CancellationReason == "" {
		t.Fatalf("expected cancellation reason recorded on run")
	}
}

func TestWorkflow_RunWithMediaPayloadOnly(t *testing.T) {
	node := &stubNode{
		id: "media-step",
		execute: func(_ context.Context, execCtx *ExecutionContext) (*ExecutionContext, error) {
			execCtx.Output = "ok"
			if _, ok := execCtx.GetSessionState("media_payload"); !ok {
				t.Fatalf("expected media payload in session state")
			}
			return execCtx, nil
		},
	}

	wf, err := New(Config{
		Name:  "media",
		Steps: []Node{node},
	})
	if err != nil {
		t.Fatalf("failed to create workflow: %v", err)
	}

	execCtx, err := wf.Run(context.Background(), "", "sess-media", WithMediaPayload([]media.Attachment{
		{Type: "image", URL: "https://example.com/image.png"},
	}))
	if err != nil {
		t.Fatalf("Run() with media payload failed: %v", err)
	}
	if execCtx.Output != "ok" {
		t.Fatalf("expected output ok, got %s", execCtx.Output)
	}
}

func TestWorkflow_AddStep(t *testing.T) {
	wf, _ := New(Config{Name: "test"})

	initialCount := len(wf.Steps)

	step, _ := NewStep(StepConfig{
		ID:    "new-step",
		Agent: createMockAgent("a1", "output"),
	})

	wf.AddStep(step)

	if len(wf.Steps) != initialCount+1 {
		t.Errorf("AddStep() failed, expected %d steps, got %d", initialCount+1, len(wf.Steps))
	}
}

func TestComplexWorkflow(t *testing.T) {
	// Build a complex workflow: Step -> Condition -> (True: Loop, False: Parallel)

	agent1 := createMockAgent("a1", "processed")
	step1, _ := NewStep(StepConfig{ID: "initial", Agent: agent1})

	// True branch: Loop
	loopAgent := createMockAgent("loop", "looped")
	loopBody, _ := NewStep(StepConfig{ID: "loop-body", Agent: loopAgent})
	loopNode, _ := NewLoop(LoopConfig{
		ID:   "loop",
		Body: loopBody,
		Condition: func(ctx *ExecutionContext, iteration int) bool {
			return iteration < 2
		},
	})

	// False branch: Parallel
	parallelAgent1 := createMockAgent("p1", "parallel1")
	parallelAgent2 := createMockAgent("p2", "parallel2")
	parallelStep1, _ := NewStep(StepConfig{ID: "p1", Agent: parallelAgent1})
	parallelStep2, _ := NewStep(StepConfig{ID: "p2", Agent: parallelAgent2})
	parallelNode, _ := NewParallel(ParallelConfig{
		ID:    "parallel",
		Nodes: []Node{parallelStep1, parallelStep2},
	})

	// Condition
	condition, _ := NewCondition(ConditionConfig{
		ID: "condition",
		Condition: func(ctx *ExecutionContext) bool {
			return strings.Contains(ctx.Output, "loop")
		},
		TrueNode:  loopNode,
		FalseNode: parallelNode,
	})

	wf, _ := New(Config{
		Name:  "complex",
		Steps: []Node{step1, condition},
	})

	// Test false branch (parallel)
	result, err := wf.Run(context.Background(), "test", "")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should have executed parallel branch
	if _, exists := result.Get("parallel_parallel_branch_0_output"); !exists {
		t.Error("Parallel branch was not executed")
	}
}
