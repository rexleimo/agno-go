# GLM (智谱AI) Model Integration

GLM (智谱AI/Zhipu AI) model integration for Agno-Go framework.

GLM (智谱AI/Zhipu AI) 模型在 Agno-Go 框架中的集成实现。

## Features / 功能特性

- ✅ **Full API Support** / 完整 API 支持
  - Synchronous calls (`Invoke`) / 同步调用
  - Streaming responses (`InvokeStream`) / 流式响应
  - Tool/Function calling / 工具/函数调用

- ✅ **JWT Authentication** / JWT 认证
  - Secure HMAC-SHA256 signing / 安全的 HMAC-SHA256 签名
  - Automatic token generation / 自动令牌生成
  - 7-day token expiration / 7 天令牌有效期

- ✅ **Well-Tested** / 测试完善
  - 57.2% test coverage / 57.2% 测试覆盖率
  - Unit tests for all core functions / 所有核心功能的单元测试
  - Mock server testing / 模拟服务器测试

## Supported Models / 支持的模型

- **glm-4** - Main chat model / 主要对话模型
- **glm-4v** - Vision model (multimodal) / 视觉模型（多模态）
- **glm-3-turbo** - Faster, lower-cost model / 更快、成本更低的模型
- **charglm-3** - Character role-playing model / 角色扮演模型

## Installation / 安装

```bash
go get github.com/rexleimo/agno-go
```

## Quick Start / 快速开始

### Basic Usage / 基础用法

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/agno-go/pkg/hno/models/glm"
)

func main() {
    // Create GLM model
    // 创建 GLM 模型
    model, err := glm.New("glm-4", glm.Config{
        APIKey:      os.Getenv("ZHIPUAI_API_KEY"), // Format: {key_id}.{key_secret}
        Temperature: 0.7,
        MaxTokens:   1024,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Make a request
    // 发起请求
    resp, err := model.Invoke(context.Background(), &models.InvokeRequest{
        Messages: []*types.Message{
            types.NewUserMessage("你好！请介绍一下你自己。"),
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Content)
}
```

### With Agent / 与 Agent 结合使用

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/glm"
    "github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
    "github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
    // Create GLM model
    // 创建 GLM 模型
    model, err := glm.New("glm-4", glm.Config{
        APIKey:      os.Getenv("ZHIPUAI_API_KEY"),
        Temperature: 0.7,
        MaxTokens:   1024,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create agent with GLM model and tools
    // 使用 GLM 模型和工具创建 agent
    ag, err := agent.New(agent.Config{
        Name:         "GLM Assistant",
        Model:        model,
        Toolkits:     []toolkit.Toolkit{calculator.New()},
        Instructions: "你是一个有用的 AI 助手，可以使用计算器工具帮助用户。",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Run agent
    // 运行 agent
    output, err := ag.Run(context.Background(), "请计算 123 * 456 的结果")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(output.Content)
}
```

### Streaming Response / 流式响应

```go
// Create streaming request
// 创建流式请求
chunks, err := model.InvokeStream(context.Background(), &models.InvokeRequest{
    Messages: []*types.Message{
        types.NewUserMessage("写一首关于人工智能的诗"),
    },
})
if err != nil {
    log.Fatal(err)
}

// Process chunks
// 处理响应块
for chunk := range chunks {
    if chunk.Error != nil {
        log.Fatal(chunk.Error)
    }
    if chunk.Content != "" {
        fmt.Print(chunk.Content)
    }
    if chunk.Done {
        break
    }
}
fmt.Println()
```

## Configuration / 配置

### Environment Variables / 环境变量

```bash
# GLM API Key (required) - Format: {key_id}.{key_secret}
# GLM API 密钥（必需）- 格式：{key_id}.{key_secret}
export ZHIPUAI_API_KEY=your-key-id.your-key-secret

# Custom Base URL (optional)
# 自定义 Base URL（可选）
export ZHIPUAI_BASE_URL=https://open.bigmodel.cn/api/paas/v4
```

### Config Options / 配置选项

```go
type Config struct {
    APIKey      string  // Required: API key in format {key_id}.{key_secret}
                        // 必需：API 密钥，格式为 {key_id}.{key_secret}

    BaseURL     string  // Optional: Custom API endpoint
                        // 可选：自定义 API 端点
                        // Default: https://open.bigmodel.cn/api/paas/v4

    Temperature float64 // Optional: Temperature parameter (0.0-1.0)
                        // 可选：温度参数 (0.0-1.0)

    MaxTokens   int     // Optional: Maximum tokens to generate
                        // 可选：生成的最大 token 数

    TopP        float64 // Optional: Top-p sampling parameter
                        // 可选：Top-p 采样参数

    DoSample    bool    // Optional: Whether to use sampling
                        // 可选：是否使用采样
}
```

## API Key Format / API 密钥格式

GLM API keys consist of two parts separated by a dot:
GLM API 密钥由两部分组成，用点号分隔：

```
{key_id}.{key_secret}
```

Example / 示例:
```
a1b2c3d4e5f6.g7h8i9j0k1l2m3n4
```

You can get your API key from: https://open.bigmodel.cn/
你可以从以下地址获取 API 密钥：https://open.bigmodel.cn/

## Authentication / 认证

GLM uses JWT (JSON Web Token) authentication with HMAC-SHA256 signing:
GLM 使用 JWT（JSON Web Token）认证，采用 HMAC-SHA256 签名：

1. API key is parsed into `key_id` and `key_secret`
   API 密钥被解析为 `key_id` 和 `key_secret`

2. JWT token is generated with claims: `api_key`, `timestamp`, `exp`
   生成包含以下声明的 JWT 令牌：`api_key`、`timestamp`、`exp`

3. Token is signed using `key_secret` with HS256 algorithm
   使用 `key_secret` 和 HS256 算法签名令牌

4. Token is sent in `Authorization: Bearer {token}` header
   令牌通过 `Authorization: Bearer {token}` 头发送

Tokens are valid for 7 days and are automatically regenerated for each request.
令牌有效期为 7 天，每次请求时自动重新生成。

## Examples / 示例

See the complete example at: [cmd/examples/glm_agent/](../../../../cmd/examples/glm_agent/)
查看完整示例：[cmd/examples/glm_agent/](../../../../cmd/examples/glm_agent/)

Run the example / 运行示例:
```bash
export ZHIPUAI_API_KEY=your-key-id.your-key-secret
go run cmd/examples/glm_agent/main.go
```

## Testing / 测试

Run tests / 运行测试:
```bash
go test -v ./pkg/hno/models/glm/...
```

Run tests with coverage / 运行测试并生成覆盖率:
```bash
go test -v -cover ./pkg/hno/models/glm/...
```

## Error Handling / 错误处理

The GLM client returns typed errors for better error handling:
GLM 客户端返回类型化的错误以便更好地处理：

```go
resp, err := model.Invoke(ctx, req)
if err != nil {
    switch e := err.(type) {
    case *types.InvalidConfigError:
        // Configuration error (e.g., missing API key)
        // 配置错误（例如：缺少 API 密钥）
        log.Printf("Config error: %v", e)

    case *types.APIError:
        // API call error (e.g., network error, API error)
        // API 调用错误（例如：网络错误、API 错误）
        log.Printf("API error: %v", e)

    default:
        // Other errors
        // 其他错误
        log.Printf("Error: %v", err)
    }
}
```

## Limitations / 限制

- Streaming mode (`InvokeStream`) returns content chunks but does not currently support partial tool calls
  流式模式（`InvokeStream`）返回内容块，但目前不支持部分工具调用

- Some GLM-specific features like `web_search` tool are not yet exposed in the API
  某些 GLM 特定功能（如 `web_search` 工具）尚未在 API 中公开

## Contributing / 贡献

Contributions are welcome! Please:
欢迎贡献！请：

1. Fork the repository / Fork 仓库
2. Create a feature branch / 创建功能分支
3. Make your changes / 进行更改
4. Add tests / 添加测试
5. Submit a pull request / 提交 pull request

## License / 许可证

MIT License - see [LICENSE](../../../../LICENSE) for details
MIT 许可证 - 详见 [LICENSE](../../../../LICENSE)

## Links / 链接

- **GLM Official Website / 官方网站**: https://www.bigmodel.cn/
- **API Documentation / API 文档**: https://open.bigmodel.cn/dev/api
- **Agno-Go Repository / 仓库**: https://github.com/rexleimo/agno-go
