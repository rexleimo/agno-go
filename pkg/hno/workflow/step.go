package workflow

import (
	"context"
	"fmt"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
)

// Step represents a basic workflow step that executes an agent
// Step 代表执行 agent 的基本工作流步骤
type Step struct {
	ID          string
	Name        string
	Agent       *agent.Agent
	Description string

	// History configuration (private fields)
	// 历史配置 (私有字段)

	// addHistoryToStep enables/disables history for this specific step
	// nil means inherit from workflow, true/false overrides
	// addHistoryToStep 为此特定步骤启用/禁用历史
	// nil 表示从 workflow 继承,true/false 覆盖
	addHistoryToStep *bool

	// numHistoryRuns specifies how many history runs to include
	// nil means use workflow default
	// numHistoryRuns 指定包含多少历史运行
	// nil 表示使用 workflow 默认值
	numHistoryRuns *int
}

// StepConfig contains step configuration
// StepConfig 包含步骤配置
type StepConfig struct {
	ID          string
	Name        string
	Agent       *agent.Agent
	Description string

	// History configuration
	// 历史配置

	// AddHistoryToStep enables/disables history for this step
	// nil means inherit from workflow
	// AddHistoryToStep 为此步骤启用/禁用历史
	// nil 表示从 workflow 继承
	AddHistoryToStep *bool `json:"add_history_to_step,omitempty"`

	// NumHistoryRuns specifies history count for this step
	// nil means use workflow default
	// NumHistoryRuns 指定此步骤的历史数量
	// nil 表示使用 workflow 默认值
	NumHistoryRuns *int `json:"num_history_runs,omitempty"`
}

// NewStep creates a new step
func NewStep(config StepConfig) (*Step, error) {
	if config.Agent == nil {
		return nil, fmt.Errorf("agent is required for step")
	}

	if config.ID == "" {
		config.ID = fmt.Sprintf("step-%s", config.Name)
	}

	if config.Name == "" {
		config.Name = config.ID
	}

	return &Step{
		ID:               config.ID,
		Name:             config.Name,
		Agent:            config.Agent,
		Description:      config.Description,
		addHistoryToStep: config.AddHistoryToStep,
		numHistoryRuns:   config.NumHistoryRuns,
	}, nil
}

// Execute runs the step
// Execute 执行步骤
func (s *Step) Execute(ctx context.Context, execCtx *ExecutionContext) (*ExecutionContext, error) {
	// 1. Get workflow history configuration from session state
	// 1. 从会话状态获取工作流历史配置
	var workflowConfig *WorkflowHistoryConfig
	if cfg, ok := execCtx.GetSessionState("workflow_history_config"); ok {
		if typedCfg, ok := cfg.(*WorkflowHistoryConfig); ok {
			workflowConfig = typedCfg
		}
	}

	// 2. Prepare input (use current output as input, or initial input if no output yet)
	// 2. 准备输入（使用当前输出作为输入,如果没有输出则使用初始输入）
	input := execCtx.Output
	if input == "" {
		input = execCtx.Input
	}

	// 3. Inject history into agent's system message if needed
	// 3. 如果需要,将历史注入到 agent 的系统消息中
	if s.shouldAddHistory(workflowConfig) && s.Agent != nil {
		// Get formatted history context from ExecutionContext
		// 从 ExecutionContext 获取格式化的历史上下文
		historyContext := execCtx.GetHistoryContext()

		if historyContext != "" {
			// Inject history into agent's temporary instructions
			// 将历史注入到 agent 的临时指令中
			// Note: Agent.Run() will automatically clear temp instructions after execution
			// 注意：Agent.Run() 会在执行后自动清除临时指令
			InjectHistoryToAgent(s.Agent, historyContext)
		}
	}

	// 4. Run the agent with history-enhanced system message
	// 4. 使用历史增强的系统消息运行 agent
	output, err := s.Agent.Run(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("step %s execution failed: %w", s.ID, err)
	}

	// 5. Update execution context
	// 5. 更新执行上下文
	execCtx.Output = output.Content
	execCtx.Set(fmt.Sprintf("step_%s_output", s.ID), output.Content)

	if len(output.Events) > 0 {
		execCtx.Set(stepEventsKey(s.ID), output.Events)
		if aggregated := aggregateEventContent(output.Events); aggregated != "" {
			execCtx.Set(fmt.Sprintf("step_%s_event_output", s.ID), aggregated)
			if execCtx.Output == "" {
				execCtx.Output = aggregated
			}
		}
	}

	// 6. Save messages to session state for history recording
	// 6. 保存消息到会话状态用于历史记录
	execCtx.AddMessages(output.Messages)

	return execCtx, nil
}

// GetID returns the step ID
func (s *Step) GetID() string {
	return s.ID
}

// GetType returns the node type
func (s *Step) GetType() NodeType {
	return NodeTypeStep
}

// shouldAddHistory determines whether to add history for this step
// shouldAddHistory 决定是否为此步骤添加历史
func (s *Step) shouldAddHistory(workflowConfig *WorkflowHistoryConfig) bool {
	// Step-level configuration takes precedence
	// Step 级别配置优先
	if s.addHistoryToStep != nil {
		return *s.addHistoryToStep
	}

	// Otherwise use workflow-level configuration
	// 否则使用 workflow 级别配置
	if workflowConfig != nil {
		return workflowConfig.AddHistoryToSteps
	}

	return false
}

// getHistoryRunCount gets the number of history runs to include
// getHistoryRunCount 获取包含的历史运行数量
func (s *Step) getHistoryRunCount(workflowConfig *WorkflowHistoryConfig) int {
	// Step-level configuration takes precedence
	// Step 级别配置优先
	if s.numHistoryRuns != nil {
		return *s.numHistoryRuns
	}

	// Otherwise use workflow-level configuration
	// 否则使用 workflow 级别配置
	if workflowConfig != nil {
		return workflowConfig.NumHistoryRuns
	}

	// Default fallback
	// 默认值
	return 3
}
