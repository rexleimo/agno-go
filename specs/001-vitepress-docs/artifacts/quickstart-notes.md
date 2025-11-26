# Quickstart Usability Test Notes (T044)

**Feature**: `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/spec.md`  
**Branch**: `001-vitepress-docs`

This file is used to:

- Record end-to-end Quickstart validation attempts for **T016** (maintainer-driven).  
- Record the results of Quickstart usability tests described in task **T044**
  (multi-participant, first-time developers).

The Quickstart flow under test comes from:

- `docs/guide/quickstart.md` (en)  
- `docs/zh/guide/quickstart.md` (zh)  
- `docs/ja/guide/quickstart.md` (ja)  
- `docs/ko/guide/quickstart.md` (ko)

> Note: This file currently provides the structure and guidance for recording results.
> Maintainers should use the T016 section for single-run validation, and the tables
> below for multi-participant usability sessions.

---

## Test protocol

- Participants: at least 10 developers who have not previously used Agno-Go.  
- Constraints: participants may only use the official docs site (no direct help from
  maintainers beyond clarifying ambiguous instructions).  
- Target: each participant should attempt to complete the end-to-end Quickstart flow
  within **10 minutes**, from starting the docs page to receiving a model response.  
- Environment: record OS, Go version and any relevant provider configuration used.  

---

## T016: Maintainer end-to-end run (current environment)

- **Date**: 2025-11-22  
- **Environment**:
  - OS: macOS (darwin/arm64, Apple M3)  
  - Go: 1.25.1  
  - Providers: not configured (`OPENAI_API_KEY` 等均为空)，使用 stub provider 以便在无真实密钥的情况下完成流程。  
- **Commands executed**:

  ```bash
  # 1) 在 Go 模块根启动服务
  cd <your-project-root>/go
  go run ./cmd/agno --config ../config/default.yaml
  ```

  在服务启动并监听 `0.0.0.0:8080` 后，另开终端执行 Quickstart 中的 HTTP 调用：

  ```bash
  # 2) 健康检查
  curl http://localhost:8080/health

  # 3) 创建 Agent
  curl -X POST http://localhost:8080/agents \
    -H "Content-Type: application/json" \
    -d '{
      "name": "quickstart-agent",
      "description": "A minimal agent created from the docs quickstart.",
      "model": {
        "provider": "openai",
        "modelId": "stub-bench",
        "stream": false
      },
      "tools": {},
      "memory": {}
    }'

  # 4) 创建会话
  curl -X POST http://localhost:8080/agents/<agent-id>/sessions \
    -H "Content-Type: application/json" \
    -d '{
      "userId": "quickstart-user",
      "metadata": {
        "source": "docs-quickstart"
      }
    }'

  # 5) 发送消息
  curl -X POST "http://localhost:8080/agents/<agent-id>/sessions/<session-id>/messages" \
    -H "Content-Type: application/json" \
    -d '{
      "messages": [
        {
          "role": "user",
          "content": "Introduce Agno-Go briefly."
        }
      ]
    }'
  ```

- **实际响应（节选）**：
  - `/health`：

    ```json
    {
      "status": "ok",
      "version": "dev",
      "providers": [
        {
          "provider": "openai",
          "status": "not-configured",
          "missingEnv": ["OPENAI_API_KEY"],
          "capabilities": ["chat", "embedding", "streaming"]
        },
        ...
      ]
    }
    ```

  - `POST /agents`：

    ```json
    {
      "agentId": "57fed239-87e2-4895-9f08-6820db3acff6"
    }
    ```

  - `POST /agents/<agent-id>/sessions`：

    ```json
    {
      "sessionId": "436711f6-b484-4d7f-bf92-9a9291a92d6c",
      "agentId": "...",
      "state": "idle",
      "userId": "quickstart-user",
      "metadata": {
        "source": "docs-quickstart",
        "tokenWindow": 256
      }
    }
    ```

  - `POST /agents/<agent-id>/sessions/<session-id>/messages`：

    ```json
    {
      "messageId": "a37e87dc-ef5e-4e47-b6bf-654a84507fb4",
      "content": "echo: Introduce Agno-Go briefly.",
      "toolCalls": null,
      "usage": {
        "promptTokens": 6,
        "completionTokens": 8,
        "latencyMs": 1
      },
      "state": "completed"
    }
    ```

- **耗时与观察**：
  - 从“服务启动并健康检查通过”到“收到第一条模型响应”的时间远小于 10 分钟（在本机上从创建 Agent 到完成消息调用约数秒）。  
  - Quickstart 文档中的路径、端点和请求体与实际行为一致；  
  - 由于使用 stub provider，返回内容为简单的 `echo:` 文本，建议在文档中注明这是示例环境的占位实现，真实部署时会由实际 provider 返回内容。  

- **T016 结论**：
  - 在实现 `go/cmd/agno` 入口后，Quickstart 端到端流程已在本环境中验证通过，并满足“10 分钟内完成”的目标。  
  - 后续的 T044 可用性测试可以沿用同一流程，观察第一次接触项目的开发者是否也能在 10 分钟内完成。

---

## Summary of latest usability session (T044, to be filled by maintainers)

- **Date**: _(YYYY-MM-DD)_  
- **Participants**: _(e.g. 10)_  
- **Locales used**: _(e.g. en: 5, zh: 3, ja: 1, ko: 1)_  
- **Completed within 10 minutes**: _(e.g. 8/10)_  
- **Top issues**:
  - _(e.g. “Config/default.yaml location unclear”)_  
  - _(e.g. “Provider env var names hard to find”)_  
- **Planned doc changes**:
  - _(link to specific docs pages to adjust)_  

---

## Per-participant log (T044)

Fill one row per participant after each session.

| ID  | Locale | OS / Go version      | Start time | End time | Duration | Completed ≤10m? | Blocking step or issue(s)                                   | Notes / Follow-ups                          |
|-----|--------|----------------------|------------|----------|----------|-----------------|-------------------------------------------------------------|---------------------------------------------|
| P01 |        |                      |            |          |          |                 |                                                             |                                             |
| P02 |        |                      |            |          |          |                 |                                                             |                                             |
| P03 |        |                      |            |          |          |                 |                                                             |                                             |
| P04 |        |                      |            |          |          |                 |                                                             |                                             |
| P05 |        |                      |            |          |          |                 |                                                             |                                             |
| P06 |        |                      |            |          |          |                 |                                                             |                                             |
| P07 |        |                      |            |          |          |                 |                                                             |                                             |
| P08 |        |                      |            |          |          |                 |                                                             |                                             |
| P09 |        |                      |            |          |          |                 |                                                             |                                             |
| P10 |        |                      |            |          |          |                 |                                                             |                                             |

Add more rows as needed for additional participants or future releases.

---

## How this ties back to docs

After each usability session:

- Update this file with participant-level results and summary findings.  
- Add a short summary to `docs-build-report.md` under the pre-release checks section,
  including:
  - Number of participants and languages used.  
  - How many completed within 10 minutes.  
  - The main documentation issues discovered and which pages they map to.  
- Create follow-up tasks to adjust the affected pages (Quickstart, Core Features & API,
  Provider Matrix, Advanced Guides, etc.) based on these findings.
