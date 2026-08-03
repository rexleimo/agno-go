# Groq Model Integration

Groq 超快速 LLM 推理集成 - 为 Agno-Go 提供业界领先的推理速度。

## 特性

- ⚡ **超快推理**: 利用 Groq 的 LPU (Language Processing Unit) 实现 10x 更快的推理速度
- 🔧 **OpenAI 兼容**: 使用 OpenAI API 格式,易于集成
- 🛠️ **工具支持**: 完整支持函数调用 (Function Calling)
- 📡 **流式响应**: 支持流式和非流式推理模式
- 🎯 **多模型**: 支持 LLaMA 3.1, Mixtral, Gemma 等模型

## 支持的模型

### LLaMA 模型 (Meta)
- `llama-3.1-8b-instant` - 最快的推理速度 (推荐)
- `llama-3.1-70b-versatile` - 最强大的性能
- `llama-3.3-70b-versatile` - 最新版本

### Mixtral 模型 (Mistral AI)
- `mixtral-8x7b-32768` - Mixture of Experts 架构

### Gemma 模型 (Google)
- `gemma2-9b-it` - 紧凑但强大

### 特殊模型
- `whisper-large-v3` - 语音识别
- `llama-guard-3-8b` - 内容审核

## 快速开始

### 1. 获取 API 密钥

访问 [Groq Console](https://console.groq.com/keys) 获取免费 API 密钥。

### 2. 设置环境变量

```bash
export GROQ_API_KEY=gsk-...
```

### 3. 使用示例

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/groq"
    "github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
    "github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
    // 创建 Groq 模型
    model, err := groq.New(groq.ModelLlama38B, groq.Config{
        APIKey:      "gsk-...",
        Temperature: 0.7,
        MaxTokens:   1024,
    })
    if err != nil {
        log.Fatal(err)
    }

    // 创建 Agent
    agent, err := agent.New(agent.Config{
        Name:         "Groq Agent",
        Model:        model,
        Toolkits:     []toolkit.Toolkit{calculator.New()},
        Instructions: "You are a helpful assistant powered by Groq.",
    })
    if err != nil {
        log.Fatal(err)
    }

    // 运行 Agent
    output, err := agent.Run(context.Background(), "Calculate 123 + 456")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(output.Content)
}
```

## 配置选项

```go
type Config struct {
    APIKey      string        // Groq API 密钥 (必需)
    BaseURL     string        // API 基础 URL (默认: https://api.groq.com/openai/v1)
    Temperature float64       // 温度参数 (0.0-2.0)
    MaxTokens   int           // 最大生成 token 数
    Timeout     time.Duration // 请求超时时间 (默认: 60s)
}
```

## 性能

Groq 的 LPU 架构提供:

- **推理速度**: 高达 10x 于传统云 LLM 提供商
- **延迟**: 极低的首 token 延迟
- **吞吐量**: 高并发请求支持

### 基准测试示例

```
Model: llama-3.1-8b-instant
Input tokens: 50
Output tokens: 100
Time: ~0.5s (vs ~5s for traditional providers)
```

## 运行示例

```bash
# 设置 API 密钥
export GROQ_API_KEY=gsk-your-api-key

# 运行示例程序
go run cmd/examples/groq_agent/main.go
```

## 测试

```bash
# 运行单元测试
go test ./pkg/hno/models/groq/

# 运行测试并查看覆盖率
go test -v -coverprofile=coverage.out ./pkg/hno/models/groq/
go tool cover -html=coverage.out
```

**当前测试覆盖率**: 52.4%

## API 文档

### 创建模型

```go
model, err := groq.New(modelID string, config Config) (*Groq, error)
```

### 模型信息查询

```go
info, found := groq.GetModelInfo(groq.ModelLlama38B)
if found {
    fmt.Printf("Model: %s\n", info.Name)
    fmt.Printf("Context: %d tokens\n", info.ContextWindow)
    fmt.Printf("Supports Tools: %v\n", info.SupportsTools)
}
```

### 调用模型

```go
// 同步调用
response, err := model.Invoke(ctx, &models.InvokeRequest{
    Messages: messages,
    Tools:    tools,
})

// 流式调用
chunks, err := model.InvokeStream(ctx, &models.InvokeRequest{
    Messages: messages,
})
for chunk := range chunks {
    fmt.Print(chunk.Content)
}
```

## 优势

### vs OpenAI
- ✅ 10x 更快的推理速度
- ✅ 免费额度更高
- ✅ 开源模型选择

### vs Anthropic
- ✅ 更低的延迟
- ✅ 更高的吞吐量
- ✅ 相似的质量 (LLaMA 3.1 70B)

### vs 本地部署
- ✅ 无需硬件投资
- ✅ 自动扩展
- ✅ 更好的性能

## 限制

- 需要互联网连接
- 免费层有速率限制
- 模型选择相比 OpenAI 较少

## 参考资源

- [Groq 官网](https://groq.com/)
- [API 文档](https://console.groq.com/docs)
- [获取 API 密钥](https://console.groq.com/keys)
- [模型列表](https://console.groq.com/docs/models)

## 许可

本集成遵循 Agno-Go 项目许可。Groq API 使用需遵循 Groq 的服务条款。
