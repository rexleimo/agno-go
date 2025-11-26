# Research Notes - 001-agno-agents-refactor

## Decisions

- **Decision**: 范围仅覆盖 docs.agno.com/basics（Agents/Teams、Memory/Sessions/State、Knowledge/RAG、Tools/MCP/HITL/Guardrails、Workflows、多模态、Reasoning/Evals/Telemetry），暂不实现 Agent OS 接口层。  
  **Rationale**: 用户要求先完成基础能力，减少返工与依赖；Agent OS 接口需要额外交互与部署链，当前不在目标内。  
  **Alternatives considered**: 扩展到 Agent OS（增加 A2A/AG UI/Slack/WhatsApp 演示）——被拒绝因范围收敛；仅做少量接口 stub —— 与“暂不覆盖”不符且会稀释测试目标。

- **Decision**: VitePress 文档仅更新与本次 Go 改动相关的 Basics 章节，并保证 VitePress 构建通过。  
  **Rationale**: 与实现范围一致，专注基础能力；全站更新会增加发布阻力且超出本迭代。  
  **Alternatives considered**: 全量站点同步（超范围）；仅更新 changelog 不改正文（不足以指导使用）。
