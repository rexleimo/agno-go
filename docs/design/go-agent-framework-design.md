# Agno-Go v2 框架设计（Go 原生优雅版 · 开源导向）

> 状态：**待对齐终稿**。本文整合 5 份研究报告（docs/research/）与现状诊断，回答"我们具体怎么做"。
> 依据（全部基于真实源码，非猜测）：
> - `competitor-framework-analysis.md` — agno/LangGraph/OpenAI SDK/Mastra
> - `agno-defects-pydanticai-design.md` — **agno 缺陷批判（115 参数 Agent、4 份重复循环、10828 行 workflow）+ PydanticAI 优雅设计**
> - `enterprise-multi-agent-frameworks.md` — CrewAI/AutoGen/Semantic Kernel
> - `ts-go-agent-framework-analysis.md` — Vercel AI SDK/Genkit Go/langchaingo/openai-go
> - `agent-skills-ops-research.md` — Agent Skills 开放标准 + OTel GenAI 语义约定 + 运维 checklist

---

## 0. 核心判断（结论先行）

1. **agno 的问题不是"功能多"，而是"形状失控"**：115 参数构造器、循环藏在 Model 层且重复 4 份、workflow 单文件 10828 行、245 处 `**kwargs`、Team 复制 Agent 的 run 管线（8963 行）。用户的直觉是对的——它不是好蓝本。
2. **PydanticAI 的优雅在于"形状"**：泛型贯穿（`Agent[DepsT, OutputDataT]`）、RunContext 依赖注入、显式图循环、类型安全从函数签名直达 JSON Schema。但它也有问题（v2 变重、版本激进、无记忆/知识内置）。
3. **Go 生态是空白**：langchaingo 停滞（Python 概念直译路线失败）、无官方 Semantic Kernel Go 版、Genkit 有 Go 但偏函数流程——**"Go 第一个优雅的多 Agent 框架"有真实市场空间**。
4. **我们的 Go 版要走第三条路**：薄 Provider（Vercel 式）+ 显式状态机循环（PydanticAI 式）+ 类型安全工具（PydanticAI/Go 泛型式）+ 渐进式披露 Skills（开放标准式）+ OTel 内置（Genkit 式）——但用 Go 的习惯表达（接口+组合+context+泛型），绝不照搬任何一家的类层次。

---

## 1. 设计哲学（八个原则 + 红线清单）

### 1.1 原则

| # | 原则 | 对应用例 | 反面教材（源码实证） |
|---|------|---------|---------------------|
| 1 | **Go 原生，不照搬 Python 概念** | 接口 + 组合 + 函数式选项 + context | agno Agent 115 参数构造器（agent.py L387-506） |
| 2 | **循环归 Runner，Provider 纯适配** | 工具循环/停止条件/上限在 Runner，唯一一份 | agno 循环藏在 Model 层且重复 4 份（models/base.py L703/925/1414/1693） |
| 3 | **类型安全工具** | struct + tag → JSON Schema，强类型 Handler | 现有 map[string]interface{}；agno `**kwargs` 透传（workflow.py 245 处） |
| 4 | **薄核心，可组合** | Agent 是配置壳（Vercel v5 实证），不含循环 | CrewAI crew.py 92KB 巨类 |
| 5 | **渐进式披露 Skills** | SKILL.md 目录包，三级披露（agentskills.io） | 全量注入 instructions（线性吃上下文） |
| 6 | **流式一等公民** | RunStream 与 Run 同等能力（含工具循环） | 现有 RunStream 聚合工具但不执行 |
| 7 | **可观测性内置** | OTel GenAI 语义 span，默认 no-op 零开销 | CrewAI 无结构化 trace 契约；现有仅 2 种 run event |
| 8 | **单二进制友好** | 技能/提示词/配置 embed.FS | 依赖外部技能文件路径 |

### 1.2 红线清单（必须避免，均来自 agno 实证）

1. 构造器参数 ≤ ~10 个（agno: 115）；跨切面能力走接口组合 + Options
2. 循环只有一份，sync/stream 共享内核（agno: 4 份副本）
3. 单文件 ≤ ~500 行（agno: workflow.py 10828 行、team/_run.py 8963 行）
4. 禁止 `map[string]any` 作为公共 API 参数（等价于 `**kwargs`）
5. 函数签名 ≤ 5 参数，运行期输入收敛为 `RunOptions` 结构体（agno run_dispatch 30 参数）
6. 编排器共享内核，禁止复制式复用（agno Team 复制 Agent run 管线）
7. 无隐式 LLM 调用（agno agentic memory 增删改查靠 LLM），可选项默认关闭且可观测
8. 配置（构造时）与运行期输入（RunOptions）严格分离（agno 两者混叠）

---

## 2. 目标包结构

```
pkg/hno/
├── agent/            # Agent 编排层（薄：~400 LOC，配置壳）
│   ├── agent.go      #   Agent[D, O] 泛型结构体：组合 Runner + tools + memory + skills + hooks
│   ├── config.go     #   Config + AgentOption 函数式选项（≤10 核心参数 + Options）
│   ├── message_builder.go  # Memory + Instructions + Skills → InvokeRequest
│   ├── tool_executor.go    # 工具查找/解析/执行/回写（goroutine 并发工具调用）
│   └── stream_aggregator.go # 流式聚合（保留现有）
│
├── runner/           # 【新增】运行时循环（框架核心，~500 LOC）
│   ├── runner.go     #   Runner：显式状态机 Step(state, event) (state, action)
│   ├── stop.go       #   停止条件枚举 + max_turns/tool_call_limit 双上限
│   ├── state.go      #   状态机：awaitModel | toolCallsPending | done
│   └── stream.go     #   流式循环（与同步共享内核，仅替换输出消费者）
│
├── models/           # 模型层（纯适配器）
│   ├── base.go       #   Provider 接口（Invoke/Stream + 元数据）+ 可选能力接口
│   └── <17 providers> #   逐个适配；openai 基于官方 openai-go SDK
│
├── tools/            # 工具层（类型安全化）
│   ├── tool.go       #   Tool[D] 泛型 + 强类型注册 Register[In,Out]
│   ├── schema.go     #   struct tag → JSON Schema（含嵌套对象/数组/枚举）
│   ├── hook.go       #   工具级 hook（执行前/后）+ 权限校验
│   ├── prepare.go    #   每步动态工具装配（对齐 PydanticAI Tool.prepare）
│   └── <23 toolkits> #   旧 map 参数自动转换兼容
│
├── skills/           # 【新增】技能机制（对齐 Agent Skills 开放标准）
│   ├── skill.go      #   Metadata + Skill 类型
│   ├── parser.go     #   SKILL.md frontmatter 解析 + 规范校验
│   ├── loader.go     #   FS / embed.FS loader（可选 Git）
│   ├── registry.go   #   注册表：Catalog / Activate（LRU）
│   └── disclosure.go #   渐进式披露：目录块注入 + use_skill 工具
│
├── observability/    # 【新增】运维层
│   ├── tracer.go     #   OTel 注入（默认 no-op）+ span 层级（invoke_agent/execute_tool/chat）
│   ├── semconv.go    #   gen_ai.* 属性常量（集中管理）
│   ├── cost.go       #   provider 定价表 + usage → cost
│   ├── retry.go      #   指数退避 + jitter（幂等操作）
│   ├── ratelimit.go  #   令牌桶（按 key/租户/模型）
│   ├── breaker.go    #   熔断 + 半开探测
│   └── promptcache.go #  system prompt 前缀缓存（Anthropic cache_control）
│
├── eval/             # 评估升级
│   ├── assertion.go  #   Contains/Regex/JSONSchema/SemanticSimilarity/ToolTrace/LLMJudge
│   ├── dataset.go    #   黄金数据集（testdata/evals/*.yaml）
│   ├── suite.go      #   Suite + CI 门禁 + 结果持久化
│   └── judge.go      #   LLM-as-judge（criteria/threshold/num_iterations）
│
├── team/             # 多 Agent（重构：组合而非新类）
│   ├── team.go       #   Team = 协调器（[]*Agent + Scheduler），复用 runner 内核
│   ├── scheduler.go  #   Scheduler 接口：轮转/LLM 选择/supervisor 委派
│   └── delegate.go   #   委派工具（收 AgentID 而非角色名模糊匹配）
│
├── workflow/         # 工作流（重构：共享内核）
│   ├── workflow.go   #   builder + executor 两阶段（≤500 行/文件）
│   ├── step.go       #   Step 泛型化（对齐 Genkit Flow[In,Out]）
│   ├── condition.go / loop.go / parallel.go / router.go
│   └── run_options.go #  RunOptions 显式结构体（禁 20 参数签名）
│
├── session/          # 保留（补 checkpoint 语义接口 State()/LoadState()）
├── run/              # 保留（事件类型扩展：tool_call/reasoning/lifecycle）
├── types/            # 保留（Message 增加多模态内容块、RunContext 类型）
├── memory/           # 保留（拆 Storage/Vector/Embedder 三接口，可选）
├── mcp/              # 保留（MCP 客户端一等支持）
└── ...               # cache/guardrails/hooks/reasoning/vectordb/knowledge 保留
```

---

## 3. 核心接口（Go 代码草案）

### 3.1 Provider —— 纯适配器，薄到极致（Vercel 实证）

```go
// Provider 只做"单次推理"，不含循环/缓存/重试。循环在 runner。
type Provider interface {
    Invoke(ctx context.Context, req *InvokeRequest) (*InvokeResponse, error)
    Stream(ctx context.Context, req *InvokeRequest) (<-chan StreamEvent, error)
    // 元数据
    GetProvider() string
    GetID() string
    GetName() string
}

// 可选能力接口：provider 实现部分或全部，Agent 用类型断言探测，未实现则优雅降级。
type StructuredOutputProvider interface {
    InvokeStructured(ctx context.Context, req *InvokeRequest, schema any) (*InvokeResponse, error)
}
type MultimodalProvider interface { /* req.Images/Audio/Videos */ }
type ReasoningProvider interface { /* 推理内容提取 */ }

// 显式结构体，禁 map 透传（红线 4）
type InvokeRequest struct {
    Messages       []*types.Message
    Tools          []ToolDefinition
    Temperature    float64
    MaxTokens      int
    ResponseFormat ResponseFormat // text | json_object | json_schema
    Images         []ImageInput
    // ProviderOptions: 提供商特有参数（显式字段或 option 函数）
}
```

### 3.2 Tool —— 类型安全（PydanticAI 可 Go 化 + Go 泛型）

```go
// 强类型工具注册：反射从 In 的 struct tag 生成 JSON Schema，执行时反序列化并校验。
func Register[In, Out any](
    name, description string,
    fn func(ctx context.Context, in In) (Out, error),
    opts ...ToolOption,
) *Tool

type Tool struct {
    Name        string
    Description string
    Schema      *Schema          // 由 reflect 从 In 生成（含嵌套对象/数组/枚举）
    Exec        func(ctx context.Context, args json.RawMessage) (any, error)
    StopAfterToolCall bool       // 循环短路标记
    Prepare     func(ctx RunContext[any]) ([]*Tool, error) // 可选：每步动态装配
}

// 用法示例（计算器）
calcAdd := tools.Register("add", "Add two numbers",
    func(ctx context.Context, in struct {
        A float64 `json:"a" jsonschema:"required,description=first number"`
        B float64 `json:"b" jsonschema:"required,description=second number"`
    }) (struct{ Result float64 `json:"result"` }, error) {
        return struct{ Result float64 }{in.A + in.B}, nil
    })
```

### 3.3 Runner —— 显式状态机循环（PydanticAI 实证 + 红线 2）

```go
// 停止条件枚举（对齐业界）
type StopReason string
const (
    StopNoToolCalls   StopReason = "no_tool_calls"
    StopLimitReached  StopReason = "limit_reached"
    StopAfterToolCall StopReason = "stop_after_tool_call"
    StopHITLBlocked   StopReason = "hitl_blocked"
    StopRequirements  StopReason = "requirements_pending"
    StopCancelled     StopReason = "cancelled"
)

// 状态机：每轮循环是纯函数，可单测、可 checkpoint、可恢复
type State int
const (
    StateAwaitModel      State = iota
    StateToolCallsPending
    StateDone
)

type Runner struct {
    Provider      Provider
    MaxTurns      int // 默认 10（OpenAI 式硬上限）
    ToolCallLimit int // agno 式工具调用上限
    // 停止回调、事件回调、缓存、重试由外部注入
}

func (r *Runner) Run(ctx context.Context, req *RunRequest) (*RunResult, error)
func (r *Runner) RunStream(ctx context.Context, req *RunRequest) (*StreamResult, error)
// 内核唯一：sync/stream 共享 Step()，仅输出消费者不同
func (r *Runner) Step(ctx context.Context, ev Event) (State, []Action, error)
```

### 3.4 RunContext —— 类型化依赖注入（PydanticAI 可 Go 化）

```go
// RunContext：deps 从 Agent → Run → 工具函数一路静态类型绑定
type RunContext[D any] struct {
    Deps     D
    Messages []*types.Message
    Usage    types.Usage
    Retries  map[string]int
    RunID    string
    ToolName string // 当前工具上下文
    // ...
}

type Agent[D any, O any] struct {
    model  Provider
    tools  *ToolRegistry[D]
    memory memory.Store
    // ...仅核心字段
}
func New[D, O any](model Provider, opts ...Option[D, O]) (*Agent[D, O], error)
func (a *Agent[D, O]) Run(ctx context.Context, input string, opts ...RunOption[D]) (*RunResult[O], error)
```

### 3.5 Skills —— 对齐开放标准（agentskills.io）

```go
type Metadata struct {
    Name          string            // ^[a-z0-9]+(-[a-z0-9]+)*$，≤64，与目录名一致
    Description   string            // ≤1024，非空
    License       string
    Compatibility string
    Metadata      map[string]string
    AllowedTools  []string          // 激活期工具授权
}

type Skill struct {
    Metadata
    Dir   string
    Body  string   // SKILL.md frontmatter 之后的正文
    Files []string // scripts/references/assets 清单
    Size  int64
}

type Loader interface {
    List(ctx context.Context) ([]Metadata, error)
    Load(ctx context.Context, name string) (*Skill, error)
}
func NewFSLoader(fsys fs.FS) Loader // os.DirFS / embed.FS

type Registry struct{}
func NewRegistry(loaders ...Loader) (*Registry, error)
func (r *Registry) Catalog() []Metadata          // → 目录块注入 system prompt（~100 tokens/技能）
func (r *Registry) Activate(ctx, name) (*Skill, error) // → use_skill 工具 handler（LRU 缓存）
```

### 3.6 Team —— 组合而非新类（红线 6）

```go
// Team = 协调器：持有 []*Agent + Scheduler，复用 runner 内核，不复制消息/会话/工具管线
type Team struct {
    Members   []*Agent[any, any] // 成员（普通 Agent 组件）
    Scheduler Scheduler          // 轮转 / LLM 选择 / supervisor 委派
    Shared    *SharedState       // 会话缓冲（ConversationState）
}

type Scheduler interface {
    // 决定下一轮谁发言；返回成员索引或错误
    Next(ctx context.Context, state *SharedState) (int, error)
}

// 委派工具收 AgentID 而非角色名模糊匹配（CrewAI 教训）
type AgentID struct{ Type, Key string }
func (t *Team) delegate(ctx context.Context, to AgentID, task string) (string, error)
```

### 3.7 Workflow —— builder + executor 两阶段

```go
// 显式 builder：AddStep/AddEdge/AddParallel/AddBranch（对齐 Mastra commitStep 思路）
// 泛型 Step[In, Out]（对齐 Genkit Flow[In,Out]）
// RunOptions 显式结构体，禁 20 参数签名（红线 5）
type RunOptions struct {
    Session  *session.Session
    Media    MediaInput
    Stream   bool
    Deps     any
    Metadata map[string]string
}
```

---

## 4. 关键设计决策（为什么 + 备选）

| 决策 | 选择 | 理由（实证） | 备选与放弃原因 |
|------|------|-------------|---------------|
| 循环归属 | 独立 `runner` 包 + 显式状态机 | agno 4 份重复循环是行为漂移温床；PydanticAI 显式图循环可测可恢复 | 塞 Provider（agno 教训）；图引擎（LangGraph Pregel 在 Go 维护成本过高） |
| Agent 形态 | 泛型 `Agent[D, O]` 配置壳，≤10 构造参数 + Options | PydanticAI 泛型贯穿；Vercel Agent 是配置壳；agno 115 参数是灾难 | Config 巨结构体（当前形状，继续膨胀） |
| 工具参数 | struct tag → JSON Schema + 泛型 Handler | PydanticAI 函数签名→schema；Go 反射天然适合 | map 方案（现状，不安全）；手写 schema（样板多） |
| 工具校验闭环 | 校验失败 → 错误回注模型重试 | PydanticAI ValidationError→ModelRetry | 直接失败（体验差） |
| Skills | 对齐 agentskills.io 开放标准 | 生态兼容；渐进式披露省 token | 自创格式；全量注入 |
| 可观测性 | OTel GenAI 语义约定，默认 no-op | Genkit Go 内置 tracing 实证；业界标准 | 自研事件体系（run.Events 保留给应用层，并行） |
| 多 Agent | Team = 组合 + Scheduler，共享内核 | agno Team 复制管线（8963 行）是教训；AutoGen Actor 的 AgentID 可 Go 化 | AutoGen 完整消息运行时（复杂度过高）；CrewAI 字符串路由（脆弱） |
| Workflow | builder + executor，泛型 Step | Mastra/Genkit 实证 | LangGraph 图引擎直译 |
| 兼容策略 | 可选接口 + 适配层，公共 API 源码兼容 | 24 示例/118 测试不重写；分阶段交付 | 彻底换 API（风险高） |

---

## 5. 实施路线（6 个里程碑，每阶段可交付、可验证）

```
M1 内核重构   runner 包（显式状态机）+ agent 拆分 + 流式工具循环 + Provider 接口升级
            → 行为不变，测试全绿（agno 红线 2/5 落地）
M2 工具层升级 struct tag schema + 泛型 Handler + 23 toolkit 自动转换 + Prepare 动态装配
M3 Skills    skills 包 + SKILL.md 解析 + 渐进式披露 + use_skill 工具（对齐开放标准）
M4 运维层     observability（OTel/成本/Retry/限流/熔断）+ eval 升级 + AgentOS 端点
M5 编排重构   team 重构为组合+Scheduler + workflow builder/executor + 平台问题修复
M6 开源发布   文档图示 + 示例更新 + README/贡献指南 + semver 纪律 + 发布 v2.0
```

每阶段验收：
- `go build ./...` 通过（chromadb 平台问题在 M5 解决）
- `make test` 通过（平台失败清零）
- `make lint` / `go vet` 通过
- 新增能力有单元测试（表驱动 + httptest 真实地址，不用假 URL）
- `openspec validate --strict` 通过

---

## 6. 需要你确认的决策点

1. **仓库路径**：保持 `github.com/rexleimo/agno-go` 还是新 org 新仓库（开源建议独立 org，如 `github.com/agno-go/agno`）？
2. **API 风格**：泛型 `Agent[D, O]` + 函数式选项（我倾向，PydanticAI/Genkit 实证）—— 但 Go 泛型对新手有门槛，是否接受？
3. **Team 首版范围**：M5 的 Team 重构为"组合 + Scheduler（轮转/LLM 选择/supervisor）+ 委派工具"最小实现（我倾向），还是保留现有 4 模式逐一适配？
4. **Skills 披露默认模式**：`use_skill` 工具机制（模型自主决策）为默认，显式 `WithSkills` 自动注入为可选项 —— 是否认可？
5. **评估门禁**：golden dataset suite 是否纳入 CI（每次 PR 跑 eval，成本换质量）？
6. **首版发布范围**：全部 17 provider / 23 toolkit 发布，还是先核心（openai/anthropic/ollama + 常用工具）发布 v2.0，其余随社区补充？

---

## 7. 附：研究结论与旧提案的关系

本设计取代 `openspec/changes/refactor-agent-framework/` 的初步方案（该方案以"对齐 Python agno"为隐含前提，现已证明方向需修正）。对齐后我将更新 OpenSpec 提案（proposal/design/tasks/spec deltas）并 `openspec validate --strict` 校验后进入实施。

**五份研究报告索引**（docs/research/）：
- agno-defects-pydanticai-design.md（44K，含附录 A 全部源码行号）
- enterprise-multi-agent-frameworks.md（22K，CrewAI/AutoGen/SK + 多 Agent 建议）
- ts-go-agent-framework-analysis.md（10K，Vercel/Genkit/Go 生态）
- competitor-framework-analysis.md（28K，第一轮四框架）
- agent-skills-ops-research.md（39K，Skill 标准 + OTel + 运维 checklist）
- 源码缓存：docs/research/_src/（agno/pai-v222/langgraph/crewAI/autogen/semantic-kernel/vercel/genkit/langchaingo/openai-go 等）
