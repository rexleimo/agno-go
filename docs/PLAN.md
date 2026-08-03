# HNO 敏捷开发大纲（分阶段验收）

> 状态：**v2（已确认决策）**。本大纲取代旧版一次性规划，改为敏捷模式：每个阶段有明确交付物与验收标准，**验收通过才进入下一阶段**。
> 已确认决策：
> - 命名：**HNO**（独立品牌，不蹭 agno 流量；模块路径发布前再定 org）
> - 内容策略：博客/官网用**真实案例数据做技术对比**（性能参数等），蹭"技术对比"流量而非品牌流量
> - API 风格：**泛型** `Agent[D, O]`（公认更优雅，利于外部开发者）
> - 交付模式：**敏捷，阶段化验收**，AI 不一次性跑完全部
> - 技术设计：docs/design/go-agent-framework-design.md（终稿）
> - 代码规范：docs/design/code-organization.md（8 条硬性规则）

---

## 阶段总览（每个阶段 = 一次验收单元）

```
S0 立项基线    已就绪：竞品源码 + 研究 + 设计 + 规范         ✅ 完成
S1 内核重构    runner 引擎 + agent 拆分 + 流式工具循环        ← 第一个验收点
S2 工具层升级  类型安全工具 + 23 toolkit 转换
S3 Skills 机制 技能包 + 渐进式披露
S4 运维层      OTel + 成本 + 运维原语 + eval + AgentOS
S5 编排重构    Team + Workflow 共享内核 + 平台修复
S6 官网与文档  首页 + 架构页 + 图示 + 多语言
S7 博客与发布  对比博客 + README + v2.0 发布 + 冷启动
S8 社区运营    生态扩展 + 持续增长
```

---

## S1 内核重构（第一个验收点）

**目标**：跑通"引擎独立 + 流式支持工具调用"的新内核，行为与现状一致。

**交付物**：
- [ ] `pkg/agno/runner`：状态机循环（awaitModel/toolCallsPending/done），停止条件枚举 + max_turns/tool_call_limit
- [ ] `agent/agent.go`（1087 行）拆分为 config / message_builder / tool_executor / stream_aggregator
- [ ] RunStream 支持工具调用循环（现状缺陷修复）
- [ ] `models.Provider` 接口升级 + 可选能力接口（StructuredOutput/Multimodal/Reasoning）
- [ ] 首批 provider 适配（openai / anthropic / ollama 3 个先做）

**验收标准**：
- [ ] `go build ./...` 通过（chromadb 平台问题除外，列入 S5）
- [ ] `make test` 通过：现有 1093 个测试保持绿 + 新增 runner 状态机测试
- [ ] 新增测试：流式工具循环（httptest 模拟）、停止条件枚举全覆盖
- [ ] `make lint` / `go vet` 通过
- [ ] 代码规范自查：单文件 ≤500 行、函数 ≤5 参数、无 map 透传（code-organization.md 8 条）

**预计工作量**：约 800-1200 行新代码 + 对应测试

---

## S2 工具层升级

**目标**：工具定义类型安全化，旧代码零破坏。

**交付物**：
- [ ] `tools.Register[In, Out]` 泛型注册 + struct tag → JSON Schema
- [ ] 23 个 toolkit 自动转换（旧 map 参数 → 新 schema，兼容层）
- [ ] 校验失败自动回注模型重试（ValidationError → ModelRetry 闭环）
- [ ] goroutine 并发工具调用 + 工具级 hook + 权限校验

**验收标准**：
- [ ] 现有工具测试全绿（59 处 RegisterFunction 不重写）
- [ ] 新工具 API 示例（calculator 用泛型重写）
- [ ] 并发工具调用测试（验证结果正确性）

---

## S3 Skills 机制

**目标**：框架级技能包，对齐 Agent Skills 开放标准。

**交付物**：
- [ ] `pkg/agno/skills`：Metadata/Skill + SKILL.md 解析（frontmatter 校验）
- [ ] Loader（FS/embed.FS）+ Registry（Catalog/Activate + LRU）
- [ ] 渐进式披露：目录注入 + use_skill 工具
- [ ] 3 个示例技能（web-research / pdf-summary / code-review）

**验收标准**：
- [ ] SKILL.md 解析校验测试（name 正则/保留字/路径安全）
- [ ] 渐进式披露测试（目录 100 tokens / 激活加载 body / 按需资源）
- [ ] 示例技能端到端演示（真实跑通一个）

---

## S4 运维层

**目标**：可观测性开箱即用。

**交付物**：
- [ ] `pkg/agno/observability`：OTel tracer（默认 no-op）+ invoke_agent/execute_tool/chat span + gen_ai.* 常量
- [ ] 成本估算（provider 定价表）
- [ ] Retry / RateLimiter / CircuitBreaker / PromptCache
- [ ] eval 升级：Assertion（Contains/Regex/JSONSchema/ToolTrace/LLMJudge）+ Dataset + Suite
- [ ] AgentOS 端点：/skills /observability /eval-runs

**验收标准**：
- [ ] OTel span 测试（模拟 exporter 验证属性）
- [ ] Logfire/Langfuse 接入示例跑通
- [ ] eval suite 跑通黄金数据集

---

## S5 编排重构 + 平台修复

**目标**：Team/Workflow 共享内核，平台问题清零。

**交付物**：
- [ ] Team 重构：组合 + Scheduler（轮转/LLM 选择/supervisor）+ AgentID 委派
- [ ] Workflow 重构：builder + executor + 泛型 Step + RunOptions
- [ ] chromadb Windows 构建修复（build tags / 纯 Go tokenizer）
- [ ] security 路径验证 Windows 分流
- [ ] workflow metrics duration==0 修复

**验收标准**：
- [ ] 平台失败清零（1093 通过 + 新增全绿，0 失败）
- [ ] Team/Workflow 现有测试适配后全绿
- [ ] `make test` 全量通过

---

## S6 官网与文档

**目标**：官网可评审、文档配图齐全。

**交付物**：
- [ ] VitePress 主题定制（website-ui-design.md 规范落地：白底/近黑/Go 青蓝/Inter）
- [ ] 首页落地页（Hero/特性/代码展示/对比表/性能区/CTA）
- [ ] 架构页重写 + 图示（SVG 线条图）
- [ ] guide 各章节配图（Mermaid/SVG）
- [ ] 中英双语

**验收标准**：
- [ ] `npm run docs:build` 通过
- [ ] 浏览器实测：无 AI 味（对照 website-ui-design.md 禁止清单）
- [ ] 你亲自过目首页 + 架构页

---

## S7 博客与发布

**目标**：对外发布 v2.0，技术对比内容上线。

**交付物**：
- [ ] README 重写（品牌名 + 定位 + 对比表 + 快速开始）
- [ ] 博客 1：《为什么不用 agno：技术对比与实测数据》（真实 benchmark）
- [ ] 博客 2：《Go Agent 框架的空白》
- [ ] 示例更新 + 贡献指南 + Release v2.0.0
- [ ] 冷启动：Show HN + r/golang + 掘金/CSDN

**验收标准**：
- [ ] GitHub Release 发布
- [ ] 对比博客含真实可复现的 benchmark 数据
- [ ] 官网与 README 品牌统一

---

## S8 社区运营（长期）

**目标**：生态增长。

- 每周 1 篇技术博客（中英双发）
- Issues <24h 响应、PR 欢迎
- 第三方 provider/toolkit PR 目标 10+
- 对比内容持续更新（性能数据随版本刷新）

---

## 每阶段开始前的流程

1. 我提出该阶段的任务清单与验收标准（本文档对应章节）
2. 你确认或调整
3. 我实施
4. **你验收**（按验收标准逐条打勾）
5. 通过 → 进入下一阶段；不通过 → 我修复后重新验收

## 当前状态

- 等待确认：品牌命名（Agent / HNO / 其他）
- 确认后立即开始 S1 内核重构
