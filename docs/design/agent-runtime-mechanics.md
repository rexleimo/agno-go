# Agent 运行时机制（Agent 如何跑通一个流程）

> 目的：讲清楚 Agent 运行的底层机制——上下文装配、ReAct 循环、工具/Skill 发现、编排分工。
> 依据：agno（models/base.py L703 while True）、OpenAI Agents SDK（Runner）、PydanticAI（_agent_graph.py 显式图）三份源码验证 + 我们的设计。
> 配套：架构通俗版 docs/design/architecture-explained.md、代码组织规范 docs/design/code-organization.md。

---

## 1. 一次完整运行的链路

```
用户输入 "查今天的天气并写个报告"
    │
    ▼
① 上下文装配（ContextBuilder）
   system prompt（角色指令 + 技能目录 + 工具清单）
   + 历史消息 + Memory 检索结果 + Knowledge 检索结果 + 用户输入
    │
    ▼
② 模型推理（Provider.Invoke —— 只做单次调用，无循环）
    │
    ▼
③ 模型返回两种可能：
   A. 直接文本回答 → 结束
   B. 工具调用请求 get_weather(city="北京") → 进入工具阶段
    │
    ▼
④ 工具执行（ToolExecutor）
   查注册表 → 校验参数（JSON Schema）→ 执行 → 格式化结果
   → 得到 "北京 25°C 晴"
    │
    ▼
⑤ 结果回填：工具结果作为消息加入历史
    │
    ▼
⑥ 再次调用模型（回到 ②）
   模型看到结果 → 可能再调工具（write_report）→ 再循环
    │
    ▼
⑦ 停止条件命中：
   - 模型不再要工具（最常见）
   - 达到 max_turns（默认 10）/ tool_call_limit
   - 用户取消（context cancelled）
    │
    ▼
⑧ 后处理：post-hooks → 结构化输出校验 → 评估（可选）→ RunOutput
```

**核心认知：这整个循环叫 ReAct（Reasoning + Acting）**——模型推理 → 决定行动 → 观察结果 → 再推理。这是所有主流 Agent 框架的同一内核（agno/OpenAI SDK/PydanticAI/LangGraph 验证一致）。

---

## 2. 组件发现机制（不是"发现"，是"被告知 + 自己选"）

### 2.1 工具（Tools）—— 声明式注册 + Schema 广播

1. 开发者注册工具：`Register("get_weather", schema, handler)`
2. Agent 把每个工具的 JSON Schema（名称/描述/参数）拼进请求发给模型
3. 模型读 Schema 自行决定调哪个 —— 这就是"发现"
4. **推论**：工具描述质量直接决定模型会不会用。框架职责：自动从 struct tag 生成高质量 Schema、校验参数、错误回注模型重试。

### 2.2 技能（Skills）—— 渐进式披露（Agent Skills 开放标准）

| 级别 | 内容 | 开销 | 时机 |
|------|------|------|------|
| L1 目录 | 所有技能 name + 一句话描述 | ~100 tokens/技能 | 常驻 system prompt |
| L2 正文 | 选中技能的 SKILL.md 全文 | <5000 tokens | 模型调用 use_skill 时 |
| L3 资源 | scripts/references 文件 | 按需 | 模型读取时 |

装 50 个技能只付目录费；用哪个加载哪个。这是业界标准（agentskills.io，Anthropic 2025-12 发布为开放标准）。

### 2.3 记忆 / 会话 / 知识 —— 装配时按需注入

- Session 历史：自动拼入（多轮对话）
- Memory（长期记忆）：按需检索注入
- Knowledge（RAG）：检索 top-k 注入
- 统一由 ContextBuilder 管线管理（显式、可排序、可审计），不做隐式 LLM 魔法

---

## 3. 编排分工（谁负责"智能"）

| 角色 | 职责 | 智能程度 |
|------|------|---------|
| 模型 | 决定调哪个工具、顺序、何时结束 | ★★★ 真正的智能在这里 |
| 框架（Runner）| 管循环上限、回填结果、错误恢复、可观测性 | ★★ 确定性保障 |
| Team（多智能体）| Scheduler：轮流/模型选人/领导者分派 | ★★ 协作策略 |
| Workflow（工作流）| 预定义 DAG：固定步骤、条件、并行 | ★ 开发者决定一切 |

**黄金法则：需要模型聪明的地方用 Agent；需要确定性的地方用 Workflow。**
- 写诗、开放问答、探索性任务 → Agent
- 订单流水线、数据处理管道、固定流程 → Workflow
- 多个角色协作、任务可分解 → Team

---

## 4. 流式（Streaming）与同步的关系

- 同步：整轮推理完再返回
- 流式：模型逐 token 返回，框架把内容实时推给前端（SSE 事件流），工具调用阶段与同步完全一致
- **设计承诺**：RunStream 与 Run 共享同一个循环内核，流式也支持工具调用（当前 agno-go 的 RunStream 只聚合工具不执行——这是要修的首要缺陷）

---

## 5. 可观测性（跑完怎么知道发生了什么）

每次运行产生：
- RunOutput：最终结果 + 元数据（轮数、token 用量、成本）
- 事件流：run_content / tool_call / reasoning / run_completed（SSE 消费）
- OTel spans：invoke_agent → execute_tool / chat（运维采集）
- 三者并行：应用层看事件流，运维看 trace，用户看结果

---

## 6. 30 秒讲清楚（考核用）

"Agent 的流程是：把指令、历史、工具清单拼成上下文发给模型；模型要么直接回答，要么请求调用某个工具；框架执行工具、把结果回填、再问模型；如此循环，直到模型不再需要工具。工具怎么被发现？框架把每个工具的说明书（Schema）发给模型，模型自己选。技能怎么被发现？只暴露名字和一句话描述，模型觉得相关才加载全文——这叫渐进式披露。编排的智能在模型，框架负责管好循环、保证安全、全程可观测。"
