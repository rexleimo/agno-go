package team

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/hooks"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/run"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// Team represents a group of agents working together
type Team struct {
	ID          string
	Name        string
	Agents      []*agent.Agent
	Leader      *agent.Agent // Optional team leader
	Mode        TeamMode
	MaxRounds   int          // Maximum coordination rounds
	PreHooks    []hooks.Hook // Hooks executed before processing input
	PostHooks   []hooks.Hook // Hooks executed after generating output
	logger      *slog.Logger
	mu          sync.RWMutex
	taskResults map[string]*TaskResult

	sharedModel      models.Model
	inheritModel     bool
	modelOverrides   map[string]models.Model
	skipInheritance  map[string]struct{}
	inheritanceMu    sync.Mutex
	inheritanceTrace map[string]inheritanceRecord

	// Storage control flags
	// 注意: 这些标志由各个 Agent 在其 Run() 方法中处理
	storeToolMessages    bool // 是否存储工具消息
	storeHistoryMessages bool // 是否存储历史消息
}

// TeamMode defines how agents collaborate
type TeamMode string

const (
	// ModeSequential - agents work one after another
	ModeSequential TeamMode = "sequential"
	// ModeParallel - all agents work simultaneously
	ModeParallel TeamMode = "parallel"
	// ModeLeaderFollower - leader delegates tasks to followers
	ModeLeaderFollower TeamMode = "leader_follower"
	// ModeConsensus - agents discuss until reaching consensus
	ModeConsensus TeamMode = "consensus"
)

// Config contains team configuration
type Config struct {
	ID        string
	Name      string
	Agents    []*agent.Agent
	Leader    *agent.Agent
	Mode      TeamMode
	MaxRounds int
	PreHooks  []hooks.Hook // Hooks to execute before processing input
	PostHooks []hooks.Hook // Hooks to execute after generating output
	Logger    *slog.Logger

	// SharedModel 指定团队默认模型，未显式覆盖的成员将继承该模型。
	SharedModel models.Model

	// InheritModel 控制是否启用模型继承；nil 表示使用默认行为（当 SharedModel 存在时启用）。
	InheritModel *bool

	// ModelOverrides 为特定成员指定模型覆盖，优先级高于 SharedModel。
	ModelOverrides map[string]models.Model

	// DisableInheritanceFor 指定不参与模型继承的成员 ID。
	DisableInheritanceFor []string

	// Storage control flags (nil means use default: true)
	// 注意: Team 通过调用 Agent.Run() 工作，各个 Agent 已经实现了存储控制
	// 这些字段主要用于保持 API 一致性和未来扩展
	StoreToolMessages    *bool // 是否存储工具消息（由各个 Agent 处理）
	StoreHistoryMessages *bool // 是否存储历史消息（由各个 Agent 处理）
}

// TaskResult holds the result of an agent's task execution
type TaskResult struct {
	AgentID string
	Content string
	Error   error
}

// New creates a new team
func New(config Config) (*Team, error) {
	if len(config.Agents) == 0 {
		return nil, types.NewInvalidConfigError("team must have at least one agent", nil)
	}

	if config.ID == "" {
		config.ID = fmt.Sprintf("team-%s", config.Name)
	}

	if config.Name == "" {
		config.Name = config.ID
	}

	if config.Mode == "" {
		config.Mode = ModeSequential
	}

	if config.MaxRounds <= 0 {
		config.MaxRounds = 3
	}

	if config.Logger == nil {
		config.Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}

	// Validate leader-follower mode
	if config.Mode == ModeLeaderFollower && config.Leader == nil {
		return nil, types.NewInvalidConfigError("leader_follower mode requires a leader agent", nil)
	}

	// Helper function to handle nil bool pointers with default value
	boolOrDefault := func(ptr *bool, defaultVal bool) bool {
		if ptr == nil {
			return defaultVal
		}
		return *ptr
	}

	inheritModel := false
	if config.SharedModel != nil {
		inheritModel = true
	}
	if config.InheritModel != nil {
		inheritModel = *config.InheritModel && config.SharedModel != nil
	}

	var modelOverrides map[string]models.Model
	if len(config.ModelOverrides) > 0 {
		modelOverrides = make(map[string]models.Model, len(config.ModelOverrides))
		for id, mdl := range config.ModelOverrides {
			if mdl == nil {
				continue
			}
			modelOverrides[id] = mdl
		}
	}

	skipInheritance := make(map[string]struct{}, len(config.DisableInheritanceFor))
	for _, id := range config.DisableInheritanceFor {
		if id == "" {
			continue
		}
		skipInheritance[id] = struct{}{}
	}

	return &Team{
		ID:                   config.ID,
		Name:                 config.Name,
		Agents:               config.Agents,
		Leader:               config.Leader,
		Mode:                 config.Mode,
		MaxRounds:            config.MaxRounds,
		PreHooks:             config.PreHooks,
		PostHooks:            config.PostHooks,
		logger:               config.Logger,
		taskResults:          make(map[string]*TaskResult),
		storeToolMessages:    boolOrDefault(config.StoreToolMessages, true),
		storeHistoryMessages: boolOrDefault(config.StoreHistoryMessages, true),
		sharedModel:          config.SharedModel,
		inheritModel:         inheritModel,
		modelOverrides:       modelOverrides,
		skipInheritance:      skipInheritance,
		inheritanceTrace:     make(map[string]inheritanceRecord),
	}, nil
}

// RunOutput contains the team execution result
type RunOutput struct {
	Content      string                 `json:"content"`
	AgentOutputs []*AgentOutput         `json:"agent_outputs"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// AgentOutput contains output from a single agent
type AgentOutput struct {
	AgentID string `json:"agent_id"`
	Content string `json:"content"`
}

// Run executes the team with the given input
func (t *Team) Run(ctx context.Context, input string) (*RunOutput, error) {
	if input == "" {
		return nil, types.NewInvalidInputError("input cannot be empty", nil)
	}

	// Ensure a shared RunContext for the team execution so that downstream
	// agents and tools can correlate telemetry with a stable run identifier
	// and team identifier, mirroring Python's run context behaviour.
	if ctx == nil {
		ctx = context.Background()
	}
	if rc, ok := run.FromContext(ctx); ok && rc != nil {
		rc = rc.Clone()
		if rc.TeamID == "" && t.ID != "" {
			rc.TeamID = t.ID
		}
		rc.EnsureRunID()
		ctx = run.WithContext(ctx, rc)
	} else {
		rc := run.NewContext()
		if t.ID != "" {
			rc.TeamID = t.ID
		}
		rc.EnsureRunID()
		ctx = run.WithContext(ctx, rc)
	}

	t.resetInheritance()

	t.logger.Info("team run started",
		"team_id", t.ID,
		"mode", t.Mode,
		"agents", len(t.Agents),
		"store_tool_messages", t.storeToolMessages,
		"store_history_messages", t.storeHistoryMessages,
	)

	// Execute pre-hooks
	if len(t.PreHooks) > 0 {
		t.logger.Debug("executing team pre-hooks", "count", len(t.PreHooks))
		hookInput := hooks.NewHookInput(input).
			WithAgentID(t.ID).
			WithMessages([]interface{}{})

		if err := hooks.ExecuteHooks(ctx, t.PreHooks, hookInput); err != nil {
			t.logger.Error("team pre-hook failed", "error", err)
			return nil, types.NewInputCheckError("team pre-hook validation failed", err)
		}
	}

	var output *RunOutput
	var err error

	// Delegate the collaboration strategy to the mode's scheduler. The team
	// owns the run loop: it asks the scheduler for the next agent, executes
	// it via the shared kernel, and stops when the scheduler returns nil.
	// 将协作策略委托给模式对应的 scheduler。team 拥有运行循环：
	// 向 scheduler 请求下一个 agent，用共享内核执行，scheduler 返回
	// nil 时结束。
	scheduler := schedulerFor(t.Mode)
	if scheduler == nil {
		return nil, types.NewInvalidConfigError(fmt.Sprintf("unknown team mode: %s", t.Mode), nil)
	}
	output, err = t.runWithScheduler(ctx, scheduler, input)

	if err != nil {
		t.logger.Error("team run failed", "error", err)
		return nil, err
	}

	// Execute post-hooks
	if len(t.PostHooks) > 0 {
		t.logger.Debug("executing team post-hooks", "count", len(t.PostHooks))
		hookInput := hooks.NewHookInput(input).
			WithOutput(output.Content).
			WithAgentID(t.ID).
			WithMessages([]interface{}{})

		if err := hooks.ExecuteHooks(ctx, t.PostHooks, hookInput); err != nil {
			t.logger.Error("team post-hook failed", "error", err)
			return nil, types.NewOutputCheckError("team post-hook validation failed", err)
		}
	}

	t.logger.Info("team run completed", "team_id", t.ID)
	return output, nil
}

// runWithScheduler is the shared execution kernel: it drives the run loop by
// asking the scheduler for the next agent until completion, then builds the
// final RunOutput. Mode-specific behaviour (ordering, input chaining,
// convergence) lives entirely in the scheduler.
//
// runWithScheduler 是共享执行内核：通过向 scheduler 请求下一个 agent
// 驱动运行循环直至完成，然后构建最终 RunOutput。模式专属行为
// （顺序、输入串联、收敛）完全由 scheduler 承担。
func (t *Team) runWithScheduler(ctx context.Context, s Scheduler, input string) (*RunOutput, error) {
	// Adapt the team to the shared run.Loop kernel: the scheduler's Next
	// decides the next agent, InputFor derives its input, and each agent
	// invocation is a Unit.
	// 将团队适配到共享 run.Loop 内核：scheduler 的 Next 决定下一个
	// agent，InputFor 推导输入，每次 agent 调用是一个 Unit。
	loop := &run.Loop{
		Next: func(history []run.UnitOutput) (run.Unit, string, error) {
			// Translate run.UnitOutput history into AgentOutput history.
			// 将 run.UnitOutput 历史转换为 AgentOutput 历史。
			agentHistory := make([]AgentOutput, len(history))
			for i, h := range history {
				agentHistory[i] = AgentOutput{AgentID: h.UnitID, Content: h.Output}
			}

			ag, err := s.Next(ctx, t, agentHistory)
			if err != nil {
				return nil, "", err
			}
			if ag == nil {
				return nil, "", nil
			}

			agentInput := s.InputFor(input, agentHistory)

			// Leader planning stage needs its own prompt for leader-follower.
			// leader-follower 的规划阶段需要专属提示词。
			if _, ok := s.(*LeaderFollowerScheduler); ok && len(agentHistory) == 0 {
				agentInput = fmt.Sprintf(
					`You are a team leader. Break down this task into subtasks for your team members:
Task: %s

Respond with a JSON array of subtasks, one for each team member.
Example: ["subtask1", "subtask2", "subtask3"]`, input)
			}

			unit := run.NewUnitFunc(ag.ID, func(ctx context.Context, in string) (string, run.Events, error) {
				result, err := t.invokeAgent(ctx, ag, in)
				if err != nil {
					return "", nil, err
				}
				return result.Content, nil, nil
			})
			return unit, agentInput, nil
		},
	}

	history, err := loop.Run(ctx)
	if err != nil {
		return nil, types.NewError(types.ErrCodeUnknown, "team run failed", err)
	}

	// Translate back to AgentOutput for output assembly.
	// 转换回 AgentOutput 用于输出组装。
	agentOutputs := make([]AgentOutput, len(history))
	for i, h := range history {
		agentOutputs[i] = AgentOutput{AgentID: h.UnitID, Content: h.Output}
	}
	return t.buildRunOutput(input, agentOutputs)
}

// buildRunOutput assembles the final RunOutput from the execution history.
// The last agent's output is the team's answer (or the leader's synthesis
// for leader-follower mode).
//
// buildRunOutput 从执行历史组装最终 RunOutput。最后一个 agent 的
// 输出即团队答案（leader-follower 模式为 leader 的合成）。
func (t *Team) buildRunOutput(input string, history []AgentOutput) (*RunOutput, error) {
	if len(history) == 0 {
		return nil, types.NewError(types.ErrCodeUnknown, "no agents executed", nil)
	}

	finalContent := history[len(history)-1].Content
	// Parallel and consensus modes combine all outputs into the final answer.
	// 并行与共识模式将所有输出合并为最终答案。
	if t.Mode == ModeParallel || t.Mode == ModeConsensus {
		var sb strings.Builder
		for _, out := range history {
			sb.WriteString(fmt.Sprintf("\n[%s]: %s", out.AgentID, out.Content))
		}
		finalContent = strings.TrimSpace(sb.String())
	}
	metadata := map[string]interface{}{
		"mode":        string(t.Mode),
		"agent_count": len(t.Agents),
	}
	if t.Mode == ModeLeaderFollower && t.Leader != nil {
		metadata["leader_id"] = t.Leader.ID
	}
	t.appendInheritanceMetadata(metadata)

	outputs := make([]*AgentOutput, len(history))
	for i := range history {
		outputs[i] = &history[i]
	}
	return &RunOutput{
		Content:      finalContent,
		AgentOutputs: outputs,
		Metadata:     metadata,
	}, nil
}

// AddAgent adds an agent to the team
func (t *Team) AddAgent(ag *agent.Agent) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Agents = append(t.Agents, ag)
}

// RemoveAgent removes an agent from the team
func (t *Team) RemoveAgent(agentID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i, ag := range t.Agents {
		if ag.ID == agentID {
			t.Agents = append(t.Agents[:i], t.Agents[i+1:]...)
			return
		}
	}
}

// GetAgents returns all agents in the team
func (t *Team) GetAgents() []*agent.Agent {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Return a copy to prevent external modification
	agents := make([]*agent.Agent, len(t.Agents))
	copy(agents, t.Agents)
	return agents
}
