
# 高级指南：带记忆的对话（Go 优先）

本指南介绍如何在 **你自己的 Go 应用** 中，通过现有的 provider 客户端和自定义存储，
实现跨轮次、跨会话的“带记忆对话”。所有示例都基于 Go 代码，不依赖任何 HTTP 运行时。

## 1. 记忆的三种层次

可以把“记忆”粗略分为三层：

- **会话历史**：当前对话中的最近若干轮消息  
- **用户画像**：用户的长期偏好/配置（例如学习偏好、语言偏好）  
- **领域知识记录**：如历史工单、订单、重要事件等结构化事实  

Agno-Go 只对第一层提供原语（`ChatRequest` / `ChatResponse`），其余两层应由
你自己的系统负责。

## 2. 在 Go 中表示会话历史

在 Go 里，会话历史可以简单地用 `[]agent.Message` 来表示：

```go
var history []agent.Message

history = append(history,
  agent.Message{Role: agent.RoleUser, Content: "我喜欢短一点的学习时长。"},
)

// 后面模型的回复也追加进去
history = append(history,
  agent.Message{Role: agent.RoleAssistant, Content: "好的，我会尽量把每次学习控制在 30 分钟以内。"},
)
```

调用模型时，只需要把需要的那一部分历史放进 `ChatRequest.Messages` 即可：

```go
resp, err := client.Chat(ctx, model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
  },
  Messages: history,
})
```

由你的应用来决定：

- 要保留多少轮历史  
- 是否需要定期对旧消息做摘要  
- 历史记录存到哪里（数据库、缓存等）  

## 3. 引入长期记忆

长期记忆通常存放在外部存储中，需要时再注入到 prompt 中。例如：

```go
type UserProfile struct {
  ID          string
  Preferences string // 一段简短的自然语言总结
}

func buildPrompt(profile UserProfile, recent []agent.Message) string {
  var buf strings.Builder
  buf.WriteString("你是一名乐于助人的助手。\n\n")
  buf.WriteString("【用户画像】\n")
  buf.WriteString(profile.Preferences)
  buf.WriteString("\n\n【最近对话】\n")
  for _, m := range recent {
    buf.WriteString(string(m.Role))
    buf.WriteString(": ")
    buf.WriteString(m.Content)
    buf.WriteString("\n")
  }
  buf.WriteString("\n请基于上述信息回答用户的问题。\n")
  return buf.String()
}
```

然后把生成的 prompt 作为单条 `user` 消息发送即可：

```go
prompt := buildPrompt(profile, recentHistory)

resp, err := client.Chat(ctx, model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
  },
  Messages: []agent.Message{
    {Role: agent.RoleUser, Content: prompt},
  },
})
```

你的服务需要负责：

- 从数据库加载/更新 `UserProfile`  
- 决定何时做摘要、何时保留完整历史  
- 确保敏感数据符合安全与合规要求  

## 4. 结合流式输出

带记忆的对话同样可以使用流式输出：

```go
req := model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
    Stream:   true,
  },
  Messages: history,
}

err := client.Stream(ctx, req, func(ev model.ChatStreamEvent) error {
  if ev.Type == "token" {
    fmt.Print(ev.Delta)
  }
  if ev.Done {
    fmt.Println()
  }
  return nil
})
if err != nil {
  log.Fatalf("stream error: %v", err)
}
```

在处理完一次对话后，可以把最终的助手回复追加回 `history`，这样后续请求就能看到更新后的上下文。

## 5. 存储与配置

在记忆较重的场景下建议：

- 使用你自己的数据库/缓存（如 Postgres、Redis、键值存储）持久化用户画像和对话片段  
- 使用 [配置与安全实践](../config-and-security) 决定启用哪些 Provider 以及如何管理 API Key  
- 尽量避免把大量敏感或长期数据直接塞进 prompt，优先使用经过筛选和总结后的关键信息  

Agno-Go 不强制任何存储后端，只定义了消息和请求的结构。

## 6. 与其它文档的关系

- [快速开始](../quickstart) 展示了最简单的“无状态”调用路径  
- 本文在此基础上增加了应用层的记忆与存储逻辑  
- 规范中的 HTTP 运行时设计同样围绕会话和记忆，但在实现稳定之前，请将其视为内部
  设计而非已经上线的现成功能  

