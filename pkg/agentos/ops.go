package agentos

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rexleimo/agno-go/pkg/hno/skills"
)

// opsHandler serves the S4 observability endpoints: /skills, /observability
// and /eval-runs. It is self-contained (no runtime dependencies) so the
// endpoints always respond.
//
// opsHandler 提供 S4 运维端点：/skills、/observability 和 /eval-runs。
// 它自包含（无运行时依赖），保证端点始终可响应。
type opsHandler struct {
	skillsReg    *skills.Registry
	agentCount   func() int
	sessionCount func() int
	startedAt    time.Time

	mu       sync.Mutex
	evalRuns []evalRun
}

// evalRun is a recorded evaluation run.
// evalRun 是记录的评估运行。
type evalRun struct {
	ID        string    `json:"id"`
	Model     string    `json:"model"`
	Timestamp time.Time `json:"timestamp"`
	Runs      int       `json:"runs"`
	Successes int       `json:"successes"`
	Duration  string    `json:"duration"`
}

// newOpsHandler builds the ops handler with optional counters.
// newOpsHandler 构建带可选计数器的 ops handler。
func newOpsHandler(reg *skills.Registry) *opsHandler {
	return &opsHandler{
		skillsReg: reg,
		startedAt: time.Now(),
	}
}

// register mounts the ops endpoints on the given router group.
// register 将运维端点挂载到给定路由组。
func (o *opsHandler) register(v1 *gin.RouterGroup) {
	ops := v1.Group("/ops")
	{
		ops.GET("/skills", o.handleSkills)
		ops.GET("/observability", o.handleObservability)
		ops.GET("/eval-runs", o.handleEvalRuns)
		ops.POST("/eval-runs", o.handleRecordEvalRun)
	}
}

// handleSkills lists registered skills (catalog only, progressive
// disclosure: names + descriptions, no full bodies).
//
// handleSkills 列出已注册技能（仅目录：名称+描述，不含正文，
// 遵循渐进式披露）。
func (o *opsHandler) handleSkills(c *gin.Context) {
	if o.skillsReg == nil {
		c.JSON(http.StatusOK, gin.H{"skills": []interface{}{}})
		return
	}
	catalog := o.skillsReg.Catalog()
	items := make([]gin.H, 0, len(catalog))
	for _, info := range catalog {
		items = append(items, gin.H{"name": info.Name, "description": info.Description})
	}
	c.JSON(http.StatusOK, gin.H{"skills": items})
}

// handleObservability reports server runtime status.
// handleObservability 报告服务器运行状态。
func (o *opsHandler) handleObservability(c *gin.Context) {
	uptime := time.Since(o.startedAt).Round(time.Second).String()
	body := gin.H{
		"uptime":     uptime,
		"started_at": o.startedAt.UTC().Format(time.RFC3339),
		"ops": gin.H{
			"skills":    o.skillsReg != nil,
			"eval_runs": true,
		},
	}
	if o.agentCount != nil {
		body["agents"] = o.agentCount()
	}
	if o.sessionCount != nil {
		body["sessions"] = o.sessionCount()
	}
	c.JSON(http.StatusOK, body)
}

// handleEvalRuns returns recorded evaluation runs.
// handleEvalRuns 返回记录的评估运行。
func (o *opsHandler) handleEvalRuns(c *gin.Context) {
	o.mu.Lock()
	defer o.mu.Unlock()
	runs := make([]evalRun, len(o.evalRuns))
	copy(runs, o.evalRuns)
	c.JSON(http.StatusOK, gin.H{"eval_runs": runs})
}

// handleRecordEvalRun accepts an eval run report (used by CI / eval tooling).
// handleRecordEvalRun 接收评估运行报告（供 CI / 评估工具使用）。
func (o *opsHandler) handleRecordEvalRun(c *gin.Context) {
	var run evalRun
	if err := c.ShouldBindJSON(&run); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid eval run payload: " + err.Error()})
		return
	}
	if run.ID == "" {
		run.ID = "eval-" + uuid.NewString()
	}
	if run.Timestamp.IsZero() {
		run.Timestamp = time.Now()
	}
	o.mu.Lock()
	o.evalRuns = append(o.evalRuns, run)
	o.mu.Unlock()
	c.JSON(http.StatusCreated, gin.H{"id": run.ID})
}

// newRunID returns a short unique run identifier.
// newRunID 返回简短的唯一运行标识。
func newRunID() string {
	return "eval-" + uuid.NewString()
}
