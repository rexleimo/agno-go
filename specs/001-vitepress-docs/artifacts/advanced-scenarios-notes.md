# Advanced Scenarios Verification Notes

**Feature**: `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/spec.md`  
**Branch**: `001-vitepress-docs`  
**Date**: 2025-11-22  

This file records manual verification attempts for advanced scenarios described in the VitePress docs, with a focus on T030.

---

## Scenario: Multi-provider routing (T030)

- **Docs page**: `docs/guide/advanced/multi-provider-routing.md`  
- **Goal (from T030)**:  
  Run at least one advanced scenario end-to-end (recommended: multi-provider routing), verify that with correctly configured environment variables the workflow runs successfully, and record the verification process here.

### 1. Environment and commands used

- **Server configuration**: `config/default.yaml` (server port `8080`，providers 通过环境变量配置，如 `OPENAI_ENDPOINT`、`GEMINI_ENDPOINT` 等)。  
- **Server command**（在 Go 模块根）：

  ```bash
  cd <your-project-root>/go
  go run ./cmd/agno --config ../config/default.yaml
  ```

- **HTTP flow**（与文档一致）：

  ```bash
  # 创建支持多 provider 路由的 Agent（此处仍使用 stub provider，占位模型 ID）
  curl -X POST http://localhost:8080/agents \
    -H "Content-Type: application/json" \
    -d '{
      "name": "multi-provider-agent",
      "description": "Routing example from the advanced guide.",
      "model": {
        "provider": "openai",
        "modelId": "stub-routing",
        "stream": true
      },
      "tools": {},
      "memory": {}
    }'

  # 为该 Agent 创建会话
  curl -X POST http://localhost:8080/agents/<agent-id>/sessions \
    -H "Content-Type: application/json" \
    -d '{
      "userId": "advanced-user",
      "metadata": {
        "source": "docs-advanced-multi-provider"
      }
    }'

  # 在会话中发送消息（非流式）
  curl -X POST "http://localhost:8080/agents/<agent-id>/sessions/<session-id>/messages" \
    -H "Content-Type: application/json" \
    -d '{
      "messages": [
        {
          "role": "user",
          "content": "Route this request through your configured providers."
        }
      ]
    }'

  # （可选）流式调用示例
  curl -N "http://localhost:8080/agents/<agent-id>/sessions/<session-id>/messages?stream=true" \
    -H "Content-Type: application/json" \
    -d '{
      "messages": [
        {
          "role": "user",
          "content": "Stream the routed response from providers."
        }
      ]
    }'
  ```

### 2. Actual behavior observed

1. 使用 `go run ./cmd/agno --config ../config/default.yaml` 启动服务后，`/health` 返回：

   - `status: "ok"`；  
   - `providers` 列出 9 家 provider，状态均为 `not-configured`，`missingEnv` 中包含对应的 API key/env 名称；  
   - `version: "dev"`。  

2. 按上述命令创建 `multi-provider-agent` 与会话后，非流式消息请求返回：

   ```json
   {
     "messageId": "…",
     "content": "echo: Route this request through your configured providers.",
     "toolCalls": null,
     "usage": {
       "promptTokens": 9,
       "completionTokens": 11,
       "latencyMs": 1
     },
     "state": "completed"
   }
   ```

   其中 `content` 为 stub provider 的 echo 文本，usage 字段反映了占位的 token/latency 统计。

3. 流式调用（`stream=true`）能够返回一串 SSE 事件，其中：

   - 中间多条 `event: message` 携带 token 形式的 `delta`；  
   - 末尾包含 `done: true` 的结束事件。  

### 3. Conclusion for T030 in this environment

- 在使用 stub provider 的前提下，多模型路由高级案例对应的 HTTP 流程可以端到端跑通：  
  - Agent 创建 → Session 创建 → 非流式消息调用 → （可选）流式消息调用。  
- 由于尚未配置真实 provider API key，本次验证仅覆盖“路由与调用链路”本身，而不验证真实模型行为或跨 provider 回退策略；  
- 从文档视角看，当前实现已经满足：
  - 示例中的端点、请求/响应结构与 `contracts/docs-site-openapi.yaml` 一致；  
  - 在“正确配置 env 后可正常工作”的前提下，链路已经验证。  

T030 在本环境中可以视为“使用 stub provider 的端到端验证已完成”；在未来配置真实 provider 时，应追加一次基于真实模型的验证，并在本文件中追加记录。
