package workflow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/run"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// P6（G10 OPT-β）等价特征化测试：断言全部落在 Workflow.Run 的公共行为上，
// 在旧 run.Loop 内核与图内核上必须逐字节同断言（契约 D1–D14）。夹具自带，
// 不依赖既有测试文件的符号。

// p6Stub 是可控桩节点：记录执行次序、可注入行为。
type p6Stub struct {
	id       string
	nodeType NodeType
	execute  func(context.Context, *ExecutionContext) (*ExecutionContext, error)
}

func (n *p6Stub) Execute(ctx context.Context, execCtx *ExecutionContext) (*ExecutionContext, error) {
	if n.execute != nil {
		return n.execute(ctx, execCtx)
	}
	return execCtx, nil
}

func (n *p6Stub) GetID() string { return n.id }

func (n *p6Stub) GetType() NodeType {
	if n.nodeType != "" {
		return n.nodeType
	}
	return NodeTypeStep
}

func p6Append(executed *[]string, mu *sync.Mutex, id string) func(context.Context, *ExecutionContext) (*ExecutionContext, error) {
	return func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
		mu.Lock()
		*executed = append(*executed, id)
		mu.Unlock()
		return ec, nil
	}
}

func p6MustWorkflow(t *testing.T, steps ...Node) *Workflow {
	t.Helper()
	wf, err := New(Config{Name: "p6g10", Steps: steps})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return wf
}

// TestP6G10_NoLoopKernelInstantiation 是本片的结构性判据：workflow 非测试源不得再
// 实例化 run.Loop（G10 = 控制流全换图内核；旧内核在 executor.go 实例化它，本测试
// 在旧内核上判红、迁移后转绿）。
func TestP6G10_NoLoopKernelInstantiation(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir error: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(".", name))
		if err != nil {
			t.Fatalf("ReadFile %s error: %v", name, err)
		}
		if strings.Contains(string(data), "run.Loop{") {
			t.Fatalf("non-test source %s still instantiates run.Loop: control flow must run on the graph kernel", name)
		}
	}
}

// D1：线性链穿线与键集。
func TestP6G10_LinearChainParity(t *testing.T) {
	var mu sync.Mutex
	var executed []string
	wf := p6MustWorkflow(t,
		&p6Stub{id: "s1", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
			ec.Output = "one"
			ec.Set("step_s1_output", "one")
			return ec, nil
		}},
		&p6Stub{id: "s2", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
			if ec.Output != "one" {
				t.Errorf("step s2 want prev output 'one', got %q", ec.Output)
			}
			ec.Output = "two"
			ec.Set("step_s2_output", "two")
			return ec, nil
		}},
		&p6Stub{id: "s3", execute: p6Append(&executed, &mu, "s3")},
	)
	ec, err := wf.Run(context.Background(), "start", "sess-linear")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if ec.Output != "two" {
		t.Fatalf("final output = %q, want %q", ec.Output, "two")
	}
	if _, ok := ec.Get("step_s1_output"); !ok {
		t.Fatalf("step_s1_output key missing")
	}
	if _, ok := ec.Get("step_s2_output"); !ok {
		t.Fatalf("step_s2_output key missing")
	}
	if len(executed) != 1 || executed[0] != "s3" {
		t.Fatalf("s3 execution tracking = %v, want [s3]", executed)
	}
	if ec.Input != "start" {
		t.Fatalf("input = %q, want %q", ec.Input, "start")
	}

	// 零步 workflow：原样返回，无错误（旧内核空 history 形状）。
	empty := p6MustWorkflow(t)
	ecEmpty, err := empty.Run(context.Background(), "go", "sess-empty")
	if err != nil {
		t.Fatalf("empty Run() error = %v", err)
	}
	if ecEmpty.Input != "go" || ecEmpty.Output != "" {
		t.Fatalf("empty workflow ec = %+v", ecEmpty)
	}
}

// D2：resume 从中步起；不存在的 resume 步 → InvalidInput。
func TestP6G10_ResumeParity(t *testing.T) {
	var mu sync.Mutex
	var executed []string
	wf := p6MustWorkflow(t,
		&p6Stub{id: "a", execute: p6Append(&executed, &mu, "a")},
		&p6Stub{id: "b", execute: p6Append(&executed, &mu, "b")},
		&p6Stub{id: "c", execute: p6Append(&executed, &mu, "c")},
	)
	ec, err := wf.Run(context.Background(), "start", "sess-resume", WithResumeFrom("b"))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(executed) != 2 || executed[0] != "b" || executed[1] != "c" {
		t.Fatalf("executed = %v, want [b c]", executed)
	}
	_ = ec

	_, err = wf.Run(context.Background(), "start", "sess-resume-bad", WithResumeFrom("zz"))
	if err == nil {
		t.Fatalf("expected error for unknown resume step")
	}
	if !strings.Contains(err.Error(), "resume step not found") || !strings.Contains(err.Error(), "step zz not in workflow") {
		t.Fatalf("resume error text = %q", err.Error())
	}
	var hno *types.HnoError
	if !errors.As(err, &hno) || hno.Code != types.ErrCodeInvalidInput {
		t.Fatalf("resume error = %v, want INVALID_INPUT HnoError", err)
	}
}

// D3：错误聚合逐字节复刻——报最后一个成功步，[UNKNOWN] 前缀，Unwrap 可达根因。
func TestP6G10_ErrorAttribution(t *testing.T) {
	boom := errors.New("boom-in")
	wf := p6MustWorkflow(t,
		&p6Stub{id: "ok1", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
			ec.Output = "done1"
			return ec, nil
		}},
		&p6Stub{id: "bad", execute: func(_ context.Context, _ *ExecutionContext) (*ExecutionContext, error) {
			return nil, boom
		}},
		&p6Stub{id: "never", execute: func(_ context.Context, _ *ExecutionContext) (*ExecutionContext, error) {
			t.Error("step after failure must not run")
			return nil, nil
		}},
	)
	_, err := wf.Run(context.Background(), "start", "sess-fail")
	if err == nil {
		t.Fatalf("expected error")
	}
	want := "[UNKNOWN] step ok1 failed: boom-in"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
	var hno *types.HnoError
	if !errors.As(err, &hno) || hno.Code != types.ErrCodeUnknown {
		t.Fatalf("error = %v, want UNKNOWN HnoError", err)
	}
	if !errors.Is(err, boom) {
		t.Fatalf("errors.Is(root cause) failed: %v", err)
	}

	// 第一步即失败：最后一个成功步为空串（旧内核空 history 的文案形状）。
	first := p6MustWorkflow(t, &p6Stub{id: "head", execute: func(_ context.Context, _ *ExecutionContext) (*ExecutionContext, error) {
		return nil, boom
	}})
	_, err = first.Run(context.Background(), "start", "sess-fail-first")
	if err == nil {
		t.Fatalf("expected error")
	}
	if want := "[UNKNOWN] step  failed: boom-in"; err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

// D4：Condition 三态 + 判定键 + 两分支同对象只执行一次。
func TestP6G10_ConditionParity(t *testing.T) {
	var mu sync.Mutex
	taken := map[string]int{}
	count := func(id string) Node {
		return &p6Stub{id: id, execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
			mu.Lock()
			taken[id]++
			mu.Unlock()
			ec.Output = id
			return ec, nil
		}}
	}

	wf := p6MustWorkflow(t,
		&p6Stub{id: "prev"},
		&Condition{ID: "cond1", Condition: func(_ *ExecutionContext) bool { return true }, TrueNode: count("t"), FalseNode: count("f")},
		&p6Stub{id: "after"},
	)
	ec, err := wf.Run(context.Background(), "start", "sess-cond-true")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if taken["t"] != 1 || taken["f"] != 0 {
		t.Fatalf("branch counts = %v, want t=1 f=0", taken)
	}
	if v, _ := ec.Get("condition_cond1_result"); v != true {
		t.Fatalf("condition_cond1_result = %v, want true", v)
	}
	if ec.Output != "t" {
		t.Fatalf("output = %q, want t (branch output carried through the no-op tail)", ec.Output)
	}

	// 假分支 + nil TrueNode（no-op 穿出）。
	wf2 := p6MustWorkflow(t,
		&p6Stub{id: "head"},
		&Condition{ID: "cond2", Condition: func(_ *ExecutionContext) bool { return false }, TrueNode: nil, FalseNode: count("f2")},
	)
	ec2, err := wf2.Run(context.Background(), "start", "sess-cond-false")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if taken["f2"] != 1 {
		t.Fatalf("false-branch executions = %d, want 1", taken["f2"])
	}
	if ec2.Output != "f2" {
		t.Fatalf("output = %q, want f2", ec2.Output)
	}

	// 双 nil 分支：判定键仍在，上下文原样穿出。
	wf3 := p6MustWorkflow(t,
		&Condition{ID: "cond3", Condition: func(ec *ExecutionContext) bool { return ec.Input == "start" }},
	)
	ec3, err := wf3.Run(context.Background(), "start", "sess-cond-nil")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if v, _ := ec3.Get("condition_cond3_result"); v != true {
		t.Fatalf("condition_cond3_result = %v, want true", v)
	}

	// TrueNode == FalseNode（同一对象）：旧内核只执行一次，互补谓词下编译图也只激活一次。
	shared := count("shared")
	wf4 := p6MustWorkflow(t,
		&Condition{ID: "cond4", Condition: func(_ *ExecutionContext) bool { return true }, TrueNode: shared, FalseNode: shared},
	)
	if _, err := wf4.Run(context.Background(), "start", "sess-cond-shared"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if taken["shared"] != 1 {
		t.Fatalf("shared branch executions = %d, want 1", taken["shared"])
	}
}

// D5：Router 命中 / nil 路由 no-op / 未命中逐字报错（不许静默兜底）。
func TestP6G10_RouterParity(t *testing.T) {
	var mu sync.Mutex
	picked := ""
	wf := p6MustWorkflow(t,
		&p6Stub{id: "prev"},
		&Router{
			ID:     "rt1",
			Router: func(_ *ExecutionContext) string { return "gold" },
			Routes: map[string]Node{
				"gold": &p6Stub{id: "gold", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
					mu.Lock()
					picked = "gold"
					mu.Unlock()
					ec.Output = "gold"
					return ec, nil
				}},
				"void": nil,
			},
		},
	)
	ec, err := wf.Run(context.Background(), "start", "sess-rt-hit")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if picked != "gold" || ec.Output != "gold" {
		t.Fatalf("picked=%q output=%q, want gold", picked, ec.Output)
	}
	if v, _ := ec.Get("router_rt1_selected"); v != "gold" {
		t.Fatalf("router_rt1_selected = %v, want gold", v)
	}

	// 未命中：逐字复刻旧 router.go 的报错形状。
	wfMiss := p6MustWorkflow(t,
		&p6Stub{id: "prev"},
		&Router{ID: "rt2", Router: func(_ *ExecutionContext) string { return "nope" }, Routes: map[string]Node{"gold": nil}},
	)
	_, err = wfMiss.Run(context.Background(), "start", "sess-rt-miss")
	if err == nil {
		t.Fatalf("expected miss error")
	}
	want := "[UNKNOWN] step prev failed: router rt2: route 'nope' not found"
	if err.Error() != want {
		t.Fatalf("miss error = %q, want %q", err.Error(), want)
	}

	// 选中路由的节点为 nil：no-op 穿出。
	wfNil := p6MustWorkflow(t,
		&Router{ID: "rt3", Router: func(_ *ExecutionContext) string { return "gold" }, Routes: map[string]Node{"gold": nil}},
	)
	ecNil, err := wfNil.Run(context.Background(), "start", "sess-rt-nil")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if v, _ := ecNil.Get("router_rt3_selected"); v != "gold" {
		t.Fatalf("router_rt3_selected = %v, want gold", v)
	}
}

// D6：Loop 迭代号 0 起、MaxIteration 守卫、iterations 键（含 0）、体错误文案。
func TestP6G10_LoopParity(t *testing.T) {
	var mu sync.Mutex
	seen := []int{}
	calls := 0
	wf := p6MustWorkflow(t,
		&p6Stub{id: "prev"},
		&Loop{
			ID:           "lp1",
			MaxIteration: 3,
			Condition:    func(_ *ExecutionContext, iteration int) bool { return iteration < 2 },
			Body: &p6Stub{id: "body", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
				mu.Lock()
				seen = append(seen, len(seen))
				calls++
				mu.Unlock()
				return ec, nil
			}},
		},
	)
	ec, err := wf.Run(context.Background(), "start", "sess-loop")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("body calls = %d, want 2", calls)
	}
	if v, _ := ec.Get("loop_lp1_iterations"); v != 2 {
		t.Fatalf("loop_lp1_iterations = %v, want 2", v)
	}

	// 条件恒真：MaxIteration 封顶。
	capCalls := 0
	wfCap := p6MustWorkflow(t,
		&Loop{
			ID:           "lp2",
			MaxIteration: 4,
			Condition:    func(_ *ExecutionContext, _ int) bool { return true },
			Body: &p6Stub{id: "body2", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
				mu.Lock()
				capCalls++
				mu.Unlock()
				return ec, nil
			}},
		},
	)
	ecCap, err := wfCap.Run(context.Background(), "start", "sess-loop-cap")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if capCalls != 4 {
		t.Fatalf("capped body calls = %d, want 4", capCalls)
	}
	if v, _ := ecCap.Get("loop_lp2_iterations"); v != 4 {
		t.Fatalf("loop_lp2_iterations = %v, want 4", v)
	}

	// 首轮即假：0 次迭代，键仍写。
	wfZero := p6MustWorkflow(t,
		&Loop{ID: "lp3", Condition: func(_ *ExecutionContext, _ int) bool { return false }, Body: &p6Stub{id: "b3"}},
	)
	ecZero, err := wfZero.Run(context.Background(), "start", "sess-loop-zero")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if v, _ := ecZero.Get("loop_lp3_iterations"); v != 0 {
		t.Fatalf("loop_lp3_iterations = %v, want 0", v)
	}

	// 体错误：loop %s iteration %d failed，%d 为在飞迭代号（0 起）。
	errCalls := 0
	wfErr := p6MustWorkflow(t,
		&p6Stub{id: "prev"},
		&Loop{
			ID:           "lp4",
			Condition:    func(_ *ExecutionContext, it int) bool { return it < 5 },
			MaxIteration: 5,
			Body: &p6Stub{id: "b4", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
				mu.Lock()
				n := errCalls
				errCalls++
				mu.Unlock()
				if n == 1 {
					return nil, errors.New("boom-iter")
				}
				return ec, nil
			}},
		},
	)
	_, err = wfErr.Run(context.Background(), "start", "sess-loop-err")
	if err == nil {
		t.Fatalf("expected loop error")
	}
	want := "[UNKNOWN] step prev failed: loop lp4 iteration 1 failed: boom-iter"
	if err.Error() != want {
		t.Fatalf("loop error = %q, want %q", err.Error(), want)
	}
}

// D7：Parallel 克隆/汇聚/错误包装。
func TestP6G10_ParallelParity(t *testing.T) {
	wf := p6MustWorkflow(t,
		&p6Stub{id: "prev", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
			ec.Set("shared", "from-prev")
			ec.Output = "prev-out"
			return ec, nil
		}},
		&Parallel{
			ID: "pa1",
			Nodes: []Node{
				&p6Stub{id: "b0", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
					if v, _ := ec.Get("shared"); v != "from-prev" {
						t.Errorf("branch0 must see the cloned Data, got %v", v)
					}
					ec.Set("branch_key", "zero")
					ec.Output = "zero-out"
					return ec, nil
				}},
				&p6Stub{id: "b1", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
					ec.Set("branch_key", "one")
					ec.Output = "one-out"
					return ec, nil
				}},
			},
		},
	)
	ec, err := wf.Run(context.Background(), "start", "sess-par")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	// 输出取最后一个分支下标。
	if ec.Output != "one-out" {
		t.Fatalf("output = %q, want one-out (last branch index)", ec.Output)
	}
	// 每支 output 键。
	if v, _ := ec.Get("parallel_pa1_branch_0_output"); v != "zero-out" {
		t.Fatalf("branch0 output key = %v", v)
	}
	if v, _ := ec.Get("parallel_pa1_branch_1_output"); v != "one-out" {
		t.Fatalf("branch1 output key = %v", v)
	}
	// 分支 Data 带前缀合并，且不泄漏无前缀键。
	if v, _ := ec.Get("parallel_pa1_branch_0_branch_key"); v != "zero" {
		t.Fatalf("branch0 prefixed key = %v", v)
	}
	if v, _ := ec.Get("parallel_pa1_branch_1_branch_key"); v != "one" {
		t.Fatalf("branch1 prefixed key = %v", v)
	}
	if _, ok := ec.Get("branch_key"); ok {
		t.Fatalf("branch_key leaked without prefix into the main context")
	}
	// 分支内的克隆不覆盖主干既有键。
	if v, _ := ec.Get("shared"); v != "from-prev" {
		t.Fatalf("shared key = %v, want from-prev", v)
	}

	// 分支错误 → parallel execution failed 包装。
	wfErr := p6MustWorkflow(t,
		&p6Stub{id: "prev"},
		&Parallel{
			ID: "pa2",
			Nodes: []Node{
				&p6Stub{id: "pb0"},
				&p6Stub{id: "pb1", execute: func(_ context.Context, _ *ExecutionContext) (*ExecutionContext, error) {
					return nil, errors.New("branch-boom")
				}},
			},
		},
	)
	_, err = wfErr.Run(context.Background(), "start", "sess-par-err")
	if err == nil {
		t.Fatalf("expected parallel error")
	}
	want := "[UNKNOWN] step prev failed: parallel execution failed: branch-boom"
	if err.Error() != want {
		t.Fatalf("parallel error = %q, want %q", err.Error(), want)
	}
}

// D7/D8：并行分支写各自的克隆上下文与克隆会话状态；两支用屏障强制重叠，
// 合并后互不覆盖（[P6] 形状经编译器不可表达；-race 背书）。
func TestP6G10_ParallelConcurrentWriteNoRace(t *testing.T) {
	const branches = 4
	entered := make(chan int, branches)
	release := make(chan struct{})
	wf := p6MustWorkflow(t,
		&p6Stub{id: "prev", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
			ec.SetSessionState("origin", "yes")
			return ec, nil
		}},
		&Parallel{
			ID: "pa1",
			Nodes: []Node{
				&p6Stub{id: "w0", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
					entered <- 0
					<-release
					ec.Set("mine", "w0")
					ec.SetSessionState("w0", "done")
					ec.Output = "out0"
					return ec, nil
				}},
				&p6Stub{id: "w1", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
					entered <- 1
					<-release
					ec.Set("mine", "w1")
					ec.SetSessionState("w1", "done")
					ec.Output = "out1"
					return ec, nil
				}},
				&p6Stub{id: "w2", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
					entered <- 2
					<-release
					ec.Set("mine", "w2")
					ec.SetSessionState("w2", "done")
					ec.Output = "out2"
					return ec, nil
				}},
				&p6Stub{id: "w3", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
					entered <- 3
					<-release
					ec.Set("mine", "w3")
					ec.SetSessionState("w3", "done")
					ec.Output = "out3"
					return ec, nil
				}},
			},
		},
	)
	go func() {
		for i := 0; i < branches; i++ {
			<-entered
		}
		close(release)
	}()
	ec, err := wf.Run(context.Background(), "start", "sess-par-race")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for i := 0; i < branches; i++ {
		key := fmt.Sprintf("parallel_pa1_branch_%d_mine", i)
		if v, _ := ec.Get(key); v == nil {
			t.Fatalf("prefixed key %s missing", key)
		}
	}
	if _, ok := ec.Get("mine"); ok {
		t.Fatalf("branch writes leaked into the shared context")
	}
	if v, _ := ec.GetSessionState("origin"); v != "yes" {
		t.Fatalf("origin session key = %v, want yes", v)
	}
	for i := 0; i < branches; i++ {
		if v, _ := ec.GetSessionState(fmt.Sprintf("w%d", i)); v != "done" {
			t.Fatalf("branch %d session key missing after merge", i)
		}
	}
	if ec.Output != "out3" {
		t.Fatalf("output = %q, want out3 (last branch index)", ec.Output)
	}
}

// D9：按步事件入账（键集 = 已执行主干步，内容 = step_<id>_events）。
func TestP6G10_EventsParity(t *testing.T) {
	store := NewMemoryStorage(10)
	evt := run.NewRunContentEvent("run-1", "a1", "assistant", "chunk", 0)
	wf, err := New(Config{
		Name: "p6-events",
		Steps: []Node{
			&p6Stub{id: "e1", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
				ec.Set(stepEventsKey("e1"), run.Events{evt})
				return ec, nil
			}},
			&p6Stub{id: "e2"},
		},
		EnableHistory: true,
		HistoryStore:  store,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, err := wf.Run(context.Background(), "start", "sess-events"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	session, err := store.GetSession(context.Background(), "sess-events")
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}
	runs := session.GetRuns()
	if len(runs) != 1 {
		t.Fatalf("runs = %d, want 1", len(runs))
	}
	if len(runs[0].Events) != 1 {
		t.Fatalf("stored events = %d, want 1", len(runs[0].Events))
	}
	if c, ok := runs[0].Events[0].(*run.RunContentEvent); !ok || c.Content != "chunk" {
		t.Fatalf("stored event = %#v, want chunk content", runs[0].Events[0])
	}
}

// D10：取消边界语义——刚完成的步计入 lastStepID，raw ctx.Err() 为主错，取消记录落地。
func TestP6G10_CancellationParity(t *testing.T) {
	store := NewMemoryStorage(10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wf, err := New(Config{
		Name: "p6-cancel",
		Steps: []Node{
			&p6Stub{id: "cancel-step", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
				cancel()
				return ec, nil
			}},
			&p6Stub{id: "final-step", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
				ec.Output = "finished"
				return ec, nil
			}},
		},
		EnableHistory: true,
		HistoryStore:  store,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_, runErr := wf.Run(ctx, "start", "sess-cancel")
	if runErr == nil {
		t.Fatalf("expected cancellation error")
	}
	want := "[UNKNOWN] step cancel-step failed: context canceled"
	if runErr.Error() != want {
		t.Fatalf("cancel error = %q, want %q", runErr.Error(), want)
	}
	if !errors.Is(runErr, context.Canceled) {
		t.Fatalf("errors.Is(context.Canceled) failed: %v", runErr)
	}

	session, err := store.GetSession(context.Background(), "sess-cancel")
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}
	if len(session.Cancellations) != 1 {
		t.Fatalf("cancellations = %d, want 1", len(session.Cancellations))
	}
	rec := session.Cancellations[0]
	if rec.StepID != "cancel-step" {
		t.Fatalf("cancellation step_id = %q, want cancel-step", rec.StepID)
	}
	if rec.Reason != context.Canceled.Error() {
		t.Fatalf("cancellation reason = %q", rec.Reason)
	}
	runs := session.GetRuns()
	if len(runs) == 0 || runs[len(runs)-1].Status != RunStatusCancelled {
		t.Fatalf("last run status = %+v, want cancelled", runs)
	}

	// 预取消 ctx：一个步都不跑，文案为空步 ID（旧内核入口判取消的形状）。
	preCtx, preCancel := context.WithCancel(context.Background())
	preCancel()
	wf2 := p6MustWorkflow(t, &p6Stub{id: "noop"})
	_, err = wf2.Run(preCtx, "start", "sess-precancel")
	if err == nil {
		t.Fatalf("expected pre-cancelled error")
	}
	if want := "[UNKNOWN] step  failed: context canceled"; err.Error() != want {
		t.Fatalf("pre-cancelled error = %q, want %q", err.Error(), want)
	}
}

// D11：结构预算（高迭代 loop 不撞默认 1000）与同名步 ID。
func TestP6G10_BudgetAndDuplicateIDs(t *testing.T) {
	const iterations = 600
	calls := 0
	var mu sync.Mutex
	wf := p6MustWorkflow(t,
		&Loop{
			ID:           "big",
			MaxIteration: iterations,
			Condition:    func(_ *ExecutionContext, _ int) bool { return true },
			Body: &p6Stub{id: "unit", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
				mu.Lock()
				calls++
				mu.Unlock()
				return ec, nil
			}},
		},
	)
	ec, err := wf.Run(context.Background(), "start", "sess-budget")
	if err != nil {
		t.Fatalf("Run() error = %v (step budget must be computed from the structure, not the default)", err)
	}
	if calls != iterations {
		t.Fatalf("body calls = %d, want %d", calls, iterations)
	}
	if v, _ := ec.Get("loop_big_iterations"); v != iterations {
		t.Fatalf("loop_big_iterations = %v, want %d", v, iterations)
	}

	// 同名步 ID：两步都执行；错误归属用原 ID（合成名不外泄）。
	dupRuns := 0
	wfDup := p6MustWorkflow(t,
		&p6Stub{id: "dup", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
			mu.Lock()
			dupRuns++
			mu.Unlock()
			ec.Output = "first"
			return ec, nil
		}},
		&p6Stub{id: "dup", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
			mu.Lock()
			dupRuns++
			mu.Unlock()
			return nil, errors.New("dup-boom")
		}},
	)
	_, err = wfDup.Run(context.Background(), "start", "sess-dup")
	if err == nil {
		t.Fatalf("expected dup error")
	}
	if dupRuns != 2 {
		t.Fatalf("dup step executions = %d, want 2", dupRuns)
	}
	if want := "[UNKNOWN] step dup failed: dup-boom"; err.Error() != want {
		t.Fatalf("dup error = %q, want %q", err.Error(), want)
	}
}

// D13：嵌套组合——loop{condition}、parallel{leaf, loop}、loop{parallel{fail}} 的
// 错误归属链顺序（由内向外：parallel 前缀在里、loop 前缀在外、主干包最外）。
func TestP6G10_NestedComposites(t *testing.T) {
	// loop{condition}：跨迭代走两个分支。
	side := map[string]int{}
	var mu sync.Mutex
	wf := p6MustWorkflow(t,
		&Loop{
			ID:           "outer",
			MaxIteration: 4,
			Condition:    func(_ *ExecutionContext, it int) bool { return it < 2 },
			Body: &Condition{
				ID: "inner",
				Condition: func(_ *ExecutionContext) bool {
					mu.Lock()
					n := side["cond"] + side["t"] + side["f"]
					mu.Unlock()
					return n%2 == 0
				},
				TrueNode: &p6Stub{id: "t", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
					mu.Lock()
					side["t"]++
					mu.Unlock()
					return ec, nil
				}},
				FalseNode: &p6Stub{id: "f", execute: func(_ context.Context, ec *ExecutionContext) (*ExecutionContext, error) {
					mu.Lock()
					side["f"]++
					mu.Unlock()
					return ec, nil
				}},
			},
		},
	)
	ec, err := wf.Run(context.Background(), "start", "sess-nested-1")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if side["t"]+side["f"] != 2 {
		t.Fatalf("body executions = %v, want 2 total", side)
	}
	if v, _ := ec.Get("loop_outer_iterations"); v != 2 {
		t.Fatalf("loop_outer_iterations = %v, want 2", v)
	}

	// parallel{leaf, loop}：分支体内的 loop iterations 键经前缀合并进主干。
	wf2 := p6MustWorkflow(t,
		&Parallel{
			ID: "pa",
			Nodes: []Node{
				&p6Stub{id: "leaf"},
				&Loop{
					ID:           "inpar",
					MaxIteration: 2,
					Condition:    func(_ *ExecutionContext, _ int) bool { return true },
					Body:         &p6Stub{id: "unit"},
				},
			},
		},
	)
	ec2, err := wf2.Run(context.Background(), "start", "sess-nested-2")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if v, _ := ec2.Get("parallel_pa_branch_1_loop_inpar_iterations"); v != 2 {
		t.Fatalf("prefixed loop key = %v, want 2", v)
	}

	// loop{parallel{fail}}：错误归属链 = loop（外）包 parallel（内），主干包最外。
	wf3 := p6MustWorkflow(t,
		&p6Stub{id: "prev"},
		&Loop{
			ID:           "lp",
			MaxIteration: 3,
			Condition:    func(_ *ExecutionContext, _ int) bool { return true },
			Body: &Parallel{
				ID: "inner-pa",
				Nodes: []Node{
					&p6Stub{id: "pb"},
					&p6Stub{id: "pf", execute: func(_ context.Context, _ *ExecutionContext) (*ExecutionContext, error) {
						return nil, errors.New("deep-boom")
					}},
				},
			},
		},
	)
	_, err = wf3.Run(context.Background(), "start", "sess-nested-3")
	if err == nil {
		t.Fatalf("expected nested error")
	}
	want := "[UNKNOWN] step prev failed: loop lp iteration 0 failed: parallel execution failed: deep-boom"
	if err.Error() != want {
		t.Fatalf("nested error = %q, want %q", err.Error(), want)
	}
}

// D14：编译期守卫——nil 步与自引用复合环返回清晰错误而非 panic/栈溢出
// （旧内核在这两条路径上分别 panic 与进程崩，本行为是披露的改善项）。
func TestP6G10_CompileGuards(t *testing.T) {
	wfNil := p6MustWorkflow(t, &p6Stub{id: "head"}, nil)
	_, err := wfNil.Run(context.Background(), "start", "sess-guard-1")
	if err == nil {
		t.Fatalf("expected nil-step error")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Fatalf("nil-step error = %q", err.Error())
	}

	selfLoop := &Loop{ID: "self", MaxIteration: 2, Condition: func(_ *ExecutionContext, _ int) bool { return true }}
	selfLoop.Body = selfLoop
	wfCycle := p6MustWorkflow(t, selfLoop)
	_, err = wfCycle.Run(context.Background(), "start", "sess-guard-2")
	if err == nil {
		t.Fatalf("expected cyclic reference error")
	}
	if !strings.Contains(err.Error(), "cyclic") {
		t.Fatalf("cycle error = %q", err.Error())
	}

	selfCond := &Condition{ID: "self-cond", Condition: func(_ *ExecutionContext) bool { return true }}
	selfCond.TrueNode = selfCond
	wfCondCycle := p6MustWorkflow(t, selfCond)
	_, err = wfCondCycle.Run(context.Background(), "start", "sess-guard-3")
	if err == nil {
		t.Fatalf("expected cyclic condition error")
	}
}
