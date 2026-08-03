package team

import (
	"context"
	"fmt"
	"log/slog"
	"os"
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

	switch t.Mode {
	case ModeSequential:
		output, err = t.runSequential(ctx, input)
	case ModeParallel:
		output, err = t.runParallel(ctx, input)
	case ModeLeaderFollower:
		output, err = t.runLeaderFollower(ctx, input)
	case ModeConsensus:
		output, err = t.runConsensus(ctx, input)
	default:
		return nil, types.NewInvalidConfigError(fmt.Sprintf("unknown team mode: %s", t.Mode), nil)
	}

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

// runSequential executes agents one after another, passing output as input to next
func (t *Team) runSequential(ctx context.Context, input string) (*RunOutput, error) {
	currentInput := input
	agentOutputs := make([]*AgentOutput, 0, len(t.Agents))

	for i, ag := range t.Agents {
		t.logger.Info("running agent", "agent_id", ag.ID, "sequence", i+1)

		result, err := t.invokeAgent(ctx, ag, currentInput)
		if err != nil {
			return nil, types.NewError(types.ErrCodeUnknown, fmt.Sprintf("agent %s failed", ag.ID), err)
		}

		agentOutputs = append(agentOutputs, &AgentOutput{
			AgentID: ag.ID,
			Content: result.Content,
		})

		// Pass output to next agent
		currentInput = result.Content
	}

	// Last agent's output is the final result
	finalContent := ""
	if len(agentOutputs) > 0 {
		finalContent = agentOutputs[len(agentOutputs)-1].Content
	}

	metadata := map[string]interface{}{
		"mode":        string(ModeSequential),
		"agent_count": len(t.Agents),
	}
	t.appendInheritanceMetadata(metadata)

	return &RunOutput{
		Content:      finalContent,
		AgentOutputs: agentOutputs,
		Metadata:     metadata,
	}, nil
}

// runParallel executes all agents simultaneously
func (t *Team) runParallel(ctx context.Context, input string) (*RunOutput, error) {
	var wg sync.WaitGroup
	results := make(chan *AgentOutput, len(t.Agents))
	errors := make(chan error, len(t.Agents))

	for _, ag := range t.Agents {
		wg.Add(1)
		go func(a *agent.Agent) {
			defer wg.Done()

			t.logger.Info("running agent in parallel", "agent_id", a.ID)

			result, err := t.invokeAgent(ctx, a, input)
			if err != nil {
				errors <- types.NewError(types.ErrCodeUnknown, fmt.Sprintf("agent %s failed", a.ID), err)
				return
			}

			results <- &AgentOutput{
				AgentID: a.ID,
				Content: result.Content,
			}
		}(ag)
	}

	wg.Wait()
	close(results)
	close(errors)

	// Check for errors
	if len(errors) > 0 {
		return nil, <-errors
	}

	// Collect all results
	agentOutputs := make([]*AgentOutput, 0, len(t.Agents))
	for output := range results {
		agentOutputs = append(agentOutputs, output)
	}

	// Combine outputs (simple concatenation for now)
	combinedContent := ""
	for i, output := range agentOutputs {
		if i > 0 {
			combinedContent += "\n\n"
		}
		combinedContent += fmt.Sprintf("[%s]: %s", output.AgentID, output.Content)
	}

	metadata := map[string]interface{}{
		"mode":        string(ModeParallel),
		"agent_count": len(t.Agents),
	}
	t.appendInheritanceMetadata(metadata)

	return &RunOutput{
		Content:      combinedContent,
		AgentOutputs: agentOutputs,
		Metadata:     metadata,
	}, nil
}

// runLeaderFollower uses leader to delegate tasks and synthesize results
func (t *Team) runLeaderFollower(ctx context.Context, input string) (*RunOutput, error) {
	// Step 1: Leader plans and delegates
	t.logger.Info("leader planning", "leader_id", t.Leader.ID)

	planPrompt := fmt.Sprintf(`You are a team leader. Break down this task into subtasks for your team members:
Task: %s

Respond with a JSON array of subtasks, one for each team member.
Example: ["subtask1", "subtask2", "subtask3"]`, input)

	planResult, err := t.invokeAgent(ctx, t.Leader, planPrompt)
	if err != nil {
		return nil, types.NewError(types.ErrCodeUnknown, "leader planning failed", err)
	}

	// For simplicity, assign the same task to all followers
	// In a real implementation, parse planResult.Content to extract subtasks
	var wg sync.WaitGroup
	results := make(chan *AgentOutput, len(t.Agents))
	errors := make(chan error, len(t.Agents))

	// Step 2: Followers execute
	for _, ag := range t.Agents {
		wg.Add(1)
		go func(a *agent.Agent) {
			defer wg.Done()

			t.logger.Info("follower executing", "agent_id", a.ID)

			result, err := t.invokeAgent(ctx, a, input) // Use original input for now
			if err != nil {
				errors <- types.NewError(types.ErrCodeUnknown, fmt.Sprintf("agent %s failed", a.ID), err)
				return
			}

			results <- &AgentOutput{
				AgentID: a.ID,
				Content: result.Content,
			}
		}(ag)
	}

	wg.Wait()
	close(results)
	close(errors)

	if len(errors) > 0 {
		return nil, <-errors
	}

	// Collect follower outputs
	followerOutputs := make([]*AgentOutput, 0, len(t.Agents))
	combinedResults := ""
	for output := range results {
		followerOutputs = append(followerOutputs, output)
		combinedResults += fmt.Sprintf("\n[%s]: %s", output.AgentID, output.Content)
	}

	// Step 3: Leader synthesizes results
	synthesisPrompt := fmt.Sprintf(`You are a team leader. Synthesize these team member outputs into a final answer:

Original Task: %s

Team Outputs:%s

Provide a comprehensive final answer.`, input, combinedResults)

	finalResult, err := t.invokeAgent(ctx, t.Leader, synthesisPrompt)
	if err != nil {
		return nil, types.NewError(types.ErrCodeUnknown, "leader synthesis failed", err)
	}

	// Include leader outputs
	allOutputs := append([]*AgentOutput{{
		AgentID: t.Leader.ID + "_plan",
		Content: planResult.Content,
	}}, followerOutputs...)
	allOutputs = append(allOutputs, &AgentOutput{
		AgentID: t.Leader.ID + "_final",
		Content: finalResult.Content,
	})

	metadata := map[string]interface{}{
		"mode":        string(ModeLeaderFollower),
		"leader_id":   t.Leader.ID,
		"agent_count": len(t.Agents),
	}
	t.appendInheritanceMetadata(metadata)

	return &RunOutput{
		Content:      finalResult.Content,
		AgentOutputs: allOutputs,
		Metadata:     metadata,
	}, nil
}

// runConsensus runs multiple rounds until agents reach consensus
func (t *Team) runConsensus(ctx context.Context, input string) (*RunOutput, error) {
	allOutputs := make([]*AgentOutput, 0)
	previousOutputs := ""

	for round := 1; round <= t.MaxRounds; round++ {
		t.logger.Info("consensus round", "round", round, "max_rounds", t.MaxRounds)

		roundPrompt := input
		if round > 1 {
			roundPrompt = fmt.Sprintf(`Original task: %s

Previous round outputs:
%s

Consider the previous outputs and provide your refined answer. If you agree with a previous answer, state so clearly.`, input, previousOutputs)
		}

		// Run all agents in parallel
		var wg sync.WaitGroup
		results := make(chan *AgentOutput, len(t.Agents))
		errors := make(chan error, len(t.Agents))

		for _, ag := range t.Agents {
			wg.Add(1)
			go func(a *agent.Agent) {
				defer wg.Done()

				result, err := t.invokeAgent(ctx, a, roundPrompt)
				if err != nil {
					errors <- err
					return
				}

				results <- &AgentOutput{
					AgentID: fmt.Sprintf("%s_round%d", a.ID, round),
					Content: result.Content,
				}
			}(ag)
		}

		wg.Wait()
		close(results)
		close(errors)

		if len(errors) > 0 {
			return nil, <-errors
		}

		// Collect round outputs
		roundOutputs := make([]*AgentOutput, 0, len(t.Agents))
		previousOutputs = ""
		for output := range results {
			roundOutputs = append(roundOutputs, output)
			allOutputs = append(allOutputs, output)
			previousOutputs += fmt.Sprintf("\n[%s]: %s", output.AgentID, output.Content)
		}

		// Simple consensus check: if all outputs are similar length (placeholder logic)
		// In real implementation, use semantic similarity or voting
		if round >= 2 {
			// Consider consensus reached if we've done at least 2 rounds
			break
		}
	}

	// Use last round's first agent output as final
	finalContent := ""
	if len(allOutputs) > 0 {
		finalContent = allOutputs[len(allOutputs)-1].Content
	}

	metadata := map[string]interface{}{
		"mode":        string(ModeConsensus),
		"rounds":      t.MaxRounds,
		"agent_count": len(t.Agents),
	}
	t.appendInheritanceMetadata(metadata)

	return &RunOutput{
		Content:      finalContent,
		AgentOutputs: allOutputs,
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

type inheritanceRecord struct {
	ModelID string
	Source  string
}

type modelScope struct {
	restore func()
	record  *inheritanceRecord
}

func (t *Team) invokeAgent(ctx context.Context, ag *agent.Agent, input string) (*agent.RunOutput, error) {
	scope := t.prepareAgentModel(ag)
	output, err := ag.Run(ctx, input)
	if scope.restore != nil {
		scope.restore()
	}
	if scope.record != nil {
		t.recordInheritance(ag.ID, scope.record)
	}
	if output != nil && len(output.Events) > 0 {
		annotateEventsWithTeam(t.ID, output.Events)
	}
	return output, err
}

func annotateEventsWithTeam(teamID string, events run.Events) {
	if teamID == "" {
		return
	}
	for _, evt := range events {
		switch e := evt.(type) {
		case *run.RunContentEvent:
			if e.TeamID == "" {
				e.TeamID = teamID
			}
		case *run.RunCompletedEvent:
			if e.TeamID == "" {
				e.TeamID = teamID
			}
		}
	}
}

func (t *Team) prepareAgentModel(ag *agent.Agent) modelScope {
	scope := modelScope{
		restore: func() {},
	}
	if ag == nil {
		return scope
	}

	original := ag.Model
	scope.restore = func() {
		ag.Model = original
	}

	if override, ok := t.modelOverrides[ag.ID]; ok && override != nil {
		ag.Model = override
		scope.record = &inheritanceRecord{
			ModelID: override.GetID(),
			Source:  "override",
		}
		return scope
	}

	if !t.inheritModel || t.sharedModel == nil {
		return scope
	}

	if _, skip := t.skipInheritance[ag.ID]; skip {
		return scope
	}

	ag.Model = t.sharedModel
	scope.record = &inheritanceRecord{
		ModelID: t.sharedModel.GetID(),
		Source:  "team",
	}
	return scope
}

func (t *Team) recordInheritance(agentID string, record *inheritanceRecord) {
	if agentID == "" || record == nil {
		return
	}
	t.inheritanceMu.Lock()
	defer t.inheritanceMu.Unlock()

	if t.inheritanceTrace == nil {
		t.inheritanceTrace = make(map[string]inheritanceRecord)
	}
	t.inheritanceTrace[agentID] = *record
}

func (t *Team) snapshotInheritance() map[string]map[string]string {
	t.inheritanceMu.Lock()
	defer t.inheritanceMu.Unlock()

	if len(t.inheritanceTrace) == 0 {
		return nil
	}

	result := make(map[string]map[string]string, len(t.inheritanceTrace))
	for agentID, record := range t.inheritanceTrace {
		result[agentID] = map[string]string{
			"model_id": record.ModelID,
			"source":   record.Source,
		}
	}
	return result
}

func (t *Team) appendInheritanceMetadata(metadata map[string]interface{}) {
	if metadata == nil {
		return
	}
	if trace := t.snapshotInheritance(); len(trace) > 0 {
		metadata["model_inheritance"] = trace
	}
}

func (t *Team) resetInheritance() {
	t.inheritanceMu.Lock()
	defer t.inheritanceMu.Unlock()
	if t.inheritanceTrace == nil {
		t.inheritanceTrace = make(map[string]inheritanceRecord)
		return
	}
	for k := range t.inheritanceTrace {
		delete(t.inheritanceTrace, k)
	}
}
