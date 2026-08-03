package workflow

// WorkflowHistoryConfig contains workflow-level history configuration
// WorkflowHistoryConfig 包含工作流级别的历史配置
type WorkflowHistoryConfig struct {
	// AddHistoryToSteps enables automatic history injection to all steps
	// AddHistoryToSteps 启用自动向所有步骤注入历史
	AddHistoryToSteps bool `json:"add_history_to_steps"`

	// NumHistoryRuns is the default number of history runs to include
	// NumHistoryRuns 是默认包含的历史运行数量
	NumHistoryRuns int `json:"num_history_runs"`
}
