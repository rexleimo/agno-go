# Quickstart: 使用官方文档站与 Go AgentOS

**Feature**: `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/spec.md`  
**Branch**: `001-vitepress-docs`  
**Audience**: 新接触 Agno-Go 的开发者

> 目标：在 10 分钟内完成一次从“启动服务”到“收到模型响应”的完整体验，并了解官方文档站如何帮助你继续深入。

---

## 1. 前置条件

- 已安装 Go 1.25.1（或兼容版本）。  
- 已克隆项目仓库到本地（例如 `<your-project-root>`，下文示例均以该目录为根目录）。  
- 可选：若希望调用真实模型供应商，需要准备对应的 API Key，并在 `.env` 中配置。

> 提示：示例中的路径均以项目根目录为基准，例如 `go/cmd/agno`、`config/default.yaml`；请根据你的本地目录结构进行替换，但不要直接复制任何维护者的绝对路径。

---

## 2. 启动 Go AgentOS 服务

在项目根目录下，执行：

```bash
cd <your-project-root>
go run ./go/cmd/agno --config ./config/default.yaml
```

默认情况下，这会在本地启动一个 HTTP 服务（例如 `http://localhost:8080`），暴露健康检查与 AgentOS API。  
你可以在另一个终端窗口中通过健康检查验证服务是否正常：

```bash
curl http://localhost:8080/health
```

预期返回一个包含 `status`, `version`, `providers` 等字段的 JSON。

---

## 3. 创建一个最小 Agent

接下来，通过 HTTP API 创建一个最小的 Agent 定义。  
在终端中运行：

```bash
curl -X POST http://localhost:8080/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "quickstart-agent",
    "description": "A minimal agent created from the docs quickstart.",
    "model": "openai:gpt-4o-mini",        // 示例模型名称，请根据实际可用模型替换
    "tools": [],
    "config": {}
  }'
```

如果请求成功，你将收到类似如下的响应：

```json
{
  "agentId": "b1a4c1c8-8f6a-4a7f-9e5f-12f3d6c90abc"
}
```

请记录下返回的 `agentId`，后续步骤会用到。

> 提示：文档站中的“快速开始”页面将以类似结构展示完整请求与响应示例，并说明常见错误及排查方式。

---

## 4. 创建会话并发送消息

1. 创建会话：

```bash
curl -X POST http://localhost:8080/agents/<agent-id>/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "quickstart-user",
    "metadata": {
      "source": "docs-quickstart"
    }
  }'
```

将 `<agent-id>` 替换为上一步返回的 `agentId`。  
预期响应中会包含会话 `id`（例如 `sessionId` 字段），后续将使用该值。

2. 在会话中发送一条消息：

```bash
curl -X POST "http://localhost:8080/agents/<agent-id>/sessions/<session-id>/messages" \
  -H "Content-Type: application/json" \
  -d '{
    "role": "user",
    "content": "你好，帮我简要介绍一下 Agno-Go。"
  }'
```

预期响应示例（字段可能根据模型响应略有变化）：

```json
{
  "messageId": "7a1b9f50-ef9e-4f3f-bb5a-1a2c3d4e5f60",
  "content": [
    {
      "type": "text",
      "value": "Agno-Go 是一个针对多模型、多供应商的 Go 版 AgentOS..."
    }
  ],
  "toolCalls": [],
  "usage": {
    "promptTokens": 42,
    "completionTokens": 128,
    "totalTokens": 170
  },
  "state": "completed"
}
```

---

## 5. 体验流式响应（可选）

若你的环境和供应商配置支持流式输出，可以在请求中追加 `stream=true` 查询参数：

```bash
curl -N "http://localhost:8080/agents/<agent-id>/sessions/<session-id>/messages?stream=true" \
  -H "Content-Type: application/json" \
  -d '{
    "role": "user",
    "content": "请流式回答：使用 Agno-Go 构建一个知识库助手需要考虑哪些点？"
  }'
```

服务会通过 Server-Sent Events 返回分片消息，终端将持续输出事件流。  
文档站中的“高级案例”将基于此示例进一步展开完整的工具调用与多步推理流程。

---

## 6. 在官方文档站中继续探索

完成以上步骤后，你已经：

- 启动了本地 Go AgentOS 服务；  
- 创建了一个最小 Agent；  
- 在会话中发送了消息并查看了响应；  
-（可选）体验了流式输出。

接下来，你可以在官方文档站中继续探索：

- “核心功能与 API”：系统性了解 Agent、会话、消息、工具和模型供应商的概念与行为。  
- “模型供应商矩阵”：查看九家供应商的能力支持与配置差异。  
- “高级案例”：学习如何构建多模型路由、知识库助手或带持久记忆的对话代理。

> 该 quickstart 文件将作为 VitePress 文档站多语言版本（en/zh/ja/ko）中“快速开始”页面的单一事实来源，后续更新需保持两侧同步。

