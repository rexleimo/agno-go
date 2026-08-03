# 代码组织规范（分层 · 分类 · 阅读友好）

> 目的：保证 Agno-Go 代码库"分层清晰、分类明确、阅读容易"——这是软件工程底线。
> 反面教材：agno 的 workflow.py 10828 行、Agent 115 参数、team/_run.py 8963 行（无分层无分类的灾难现场）。
> 适用范围：本次重构全部新代码 + 重构的存量代码。

---

## 1. 包结构（一个包只做一件事）

```
pkg/agno/
├── agent/         编排层（配置壳）：agent.go / config.go / message_builder.go / tool_executor.go / stream_aggregator.go
├── runner/        引擎层（唯一循环）：runner.go / stop.go / state.go / stream.go
├── models/        适配层（对接 LLM）：base.go（接口）+ <provider>/ 每 provider 一个文件
├── tools/         工具层：tool.go / schema.go / hook.go + <toolkit>/ 23 个子目录
├── skills/        技能层：skill.go / parser.go / loader.go / registry.go / disclosure.go
├── observability/ 运维层：tracer.go / semconv.go / cost.go / retry.go / ratelimit.go / breaker.go
├── eval/          评估层：assertion.go / dataset.go / suite.go / judge.go
├── team/          编排层：team.go / scheduler.go / delegate.go
├── workflow/      流程层：workflow.go / step.go / condition.go / loop.go / parallel.go / router.go / run_options.go
├── session/       会话层（保留，补 checkpoint 语义）
├── memory/        记忆层（保留，拆 Storage/Vector/Embedder 接口）
├── types/         公共类型（保留）：message.go / response.go / errors.go / runcontext.go
├── run/           事件流（保留）：events.go / events_json.go
└── mcp/           保留（MCP 客户端一等支持）
```

## 2. 依赖方向（只许上层依赖下层，禁止反向）

```
agent / team / workflow     （编排层）
      ↓ 依赖
runner                       （引擎层）
      ↓ 依赖
models / tools / skills      （能力层）
      ↓ 依赖
types / session / memory     （基础层）
```

检查方法：`go list -deps` 验证每个包的依赖集合；出现"基础层依赖编排层"即视为违规。

## 3. 硬性规则（每条对应一个 agno 事故）

| # | 规则 | 上限 | agno 反面 |
|---|------|------|----------|
| 1 | 单文件行数 | ≤500 行 | workflow.py 10828 行 |
| 2 | 构造函数参数 | ≤10 个 | Agent __init__ 115 参数 |
| 3 | 函数签名参数 | ≤5 个，超了用结构体 | run_dispatch ~30 参数 |
| 4 | 公共 API 参数类型 | 显式结构体，禁 map[string]any | **kwargs 245 处 |
| 5 | 循环实现 | 唯一一份（runner 包） | Model 层 4 份重复循环 |
| 6 | 编排器复用 | Team/Workflow 共享 runner，禁复制 | team/_run.py 8963 行复制 |
| 7 | 隐式 LLM 调用 | 禁止；可选项显式开启且可观测 | agentic memory 自动调 LLM |
| 8 | 序列化 | 用 encoding/json，禁手写映射 | to_dict/from_dict 手写 |

## 4. 文件内组织（每个文件的可读性标准）

```
1. package 声明 + 文件职责注释（1-2 行，说明"这个文件做什么"）
2. imports（标准库 → 第三方 → 内部，分组）
3. 常量/类型定义
4. 构造函数
5. 方法（按调用顺序排列：对外入口在前，内部辅助在后）
6. 每个导出符号必须有 GoDoc 注释（以符号名开头）
```

## 5. 命名规范

- 包名：小写单词，不缩写（`observability` 不叫 `obs`，`runner` 不叫 `r`）
- 接口：以能力命名（`Provider`、`Loader`、`Scheduler`），后缀 `er` 仅在动词性接口
- 构造器：`New(...)` 返回 `(*Type, error)`；配置类型 `TypeConfig` 或函数式选项 `WithXxx`
- 错误：`fmt.Errorf("context: %w", err)`，消息小写，不跨层裸传

## 6. 阅读路径（新人上手顺序）

```
README.md → docs/design/architecture-explained.md（人话架构）
  → pkg/agno/runner/runner.go（引擎核心，~200 行）
  → pkg/agno/agent/agent.go（编排，~150 行）
  → pkg/agno/models/base.go（接口，~80 行）
  → 任一 provider（如 models/openai/openai.go）作为适配示例
  → cmd/examples/simple_agent/main.go（端到端示例）
```

目标：**每层可独立读懂，不需要通读全部代码**。读 runner 只需懂 types；读 agent 只需懂 runner 的接口签名。

## 7. 测试规范

- 表驱动测试（`cases := []struct{...}`），与源码同目录
- 外部依赖用 httptest 模拟（真实地址，不用假 URL）
- 核心包覆盖率 ≥75%
- 每个 bug 修复附回归测试

## 8. 文档与代码的对应

| 代码位置 | 对应文档 |
|---------|---------|
| pkg/agno/runner | docs/design/agent-runtime-mechanics.md |
| pkg/agno/agent | docs/design/go-agent-framework-design.md |
| 整体架构 | docs/design/architecture-explained.md |
| 分层规则 | 本文档 |

*评审检查项：新代码合入前按第 3 节 8 条硬性规则自查。*
