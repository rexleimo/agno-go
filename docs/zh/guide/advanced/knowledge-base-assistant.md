
# 高级指南：基于知识库的助手（Go 优先）

本指南说明如何基于 **你自己的文档与检索系统** 构建问答助手，使用的只是
Agno-Go 的 Go provider 客户端和数据模型，不依赖尚未完成的 HTTP 运行时。

## 1. 场景概览

你希望助手能回答关于以下内容的问题：

- 产品文档  
- 内部规范或政策  
- 知识库文章  

常见模式是：

1. 离线对文档做 embedding，将向量和元信息存入向量库/数据库  
2. 查询时，根据用户问题检索若干相关片段  
3. 将这些片段作为上下文写入 `agent.Message.Content` 中，请求模型生成回答  

Agno-Go **不会** 自带向量库，你需要自备存储，只需用 provider 客户端完成
embedding 和 chat 调用即可。

## 2. 使用 provider 做文档 embedding

任何实现了 `model.EmbeddingProvider` 的客户端都可以用于 embedding。具体模型 ID
和支持情况请参考 Provider Matrix 和 `.env.example`。

```go
package main

import (
  "context"
  "fmt"
  "log"
  "os"
  "time"

  "github.com/rexleimo/agno-go/internal/agent"
  "github.com/rexleimo/agno-go/internal/model"
  "github.com/rexleimo/agno-go/pkg/providers/openai"
)

func embedDoc(ctx context.Context, text string) ([]float64, error) {
  apiKey := os.Getenv("OPENAI_API_KEY")
  if apiKey == "" {
    return nil, fmt.Errorf("OPENAI_API_KEY not set")
  }

  client := openai.New("", apiKey, nil)

  resp, err := client.Embed(ctx, model.EmbeddingRequest{
    Model: agent.ModelConfig{
      Provider: agent.ProviderOpenAI,
      ModelID:  "text-embedding-3-small", // 选择合适的 embedding 模型
    },
    Input: []string{text},
  })
  if err != nil {
    return nil, err
  }
  if len(resp.Vectors) == 0 {
    return nil, fmt.Errorf("empty embedding response")
  }
  return resp.Vectors[0], nil
}
```

向量如何存储（Postgres、ClickHouse、专用向量库等）完全由你自行决定，Agno-Go
不会做约束。

## 3. 带上下文回答问题

当你已经能根据问题检索出若干片段（例如 `[]string`），可以按如下方式构造提示词：

```go
func answerWithContext(
  ctx context.Context,
  client model.ChatProvider,
  provider agent.Provider,
  modelID string,
  question string,
  passages []string,
) (string, error) {
  var contextText string
  for _, p := range passages {
    contextText += "- " + p + "\n"
  }

  prompt := fmt.Sprintf(
    "你是一名乐于助人的助手。\n\n上下文：\n%s\n问题：%s\n\n请只根据上述上下文回答；如果上下文中没有答案，请明确说明“我不知道”。",
    contextText,
    question,
  )

  resp, err := client.Chat(ctx, model.ChatRequest{
    Model: agent.ModelConfig{
      Provider: provider,
      ModelID:  modelID,
    },
    Messages: []agent.Message{
      {Role: agent.RoleUser, Content: prompt},
    },
  })
  if err != nil {
    return "", err
  }
  return resp.Message.Content, nil
}
```

这里的 `client` 可以是任意实现了 `ChatProvider` 的客户端（OpenAI、Gemini、Groq 等），
只要你在 `.env` 中配置了对应的 Key。

## 4. 组合完整知识库助手

一个完整的知识库助手通常包含三部分：

- **索引器** – 读取文档，调用 `Embed` 得到向量，并把向量+元信息写入存储  
- **检索器** – 根据问题从存储中取回若干相关片段  
- **回答器** – 使用上述模式，将片段拼成上下文，调用 `Chat` 生成回答  

在这个故事里，Agno-Go 的职责很小：

- 提供统一的 `ChatRequest` / `EmbeddingRequest` 形状  
- 提供实现了统一接口的 provider 客户端  
- 统一部分错误处理与 provider 状态表示  

存储、索引、排序等都属于你的应用层逻辑。

## 5. 与其它文档的关系

- 使用 [模型供应商矩阵](../providers/matrix) 选择适合长上下文的 Provider 和模型  
- 使用 [配置与安全实践](../config-and-security) 配置必要的环境变量
  （例如 `OPENAI_API_KEY`、`GEMINI_API_KEY`）  
- 规范中的 HTTP 运行时设计与这里的思路一致（“检索 + 对话”），在实现稳定之前，
  请将其视为数据形状参考，而不是已经上线的现成接口  

