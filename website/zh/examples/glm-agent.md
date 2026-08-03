# GLM Agent 示例

本示例演示如何在 HNO 中使用 GLM (智谱AI)，中国领先的国产大语言模型平台。

## 概述

GLM (智谱AI) 是由清华大学知识工程实验室开发的先进语言模型。它提供：

- **中文优化**: 在中文语言任务上表现出色
- **GLM-4**: 主要对话模型，支持 128K 上下文
- **GLM-4V**: 支持视觉的多模态能力
- **GLM-3-Turbo**: 快速且经济的变体

## 前置要求

1. 安装 **Go 1.21+**
2. 从 https://open.bigmodel.cn/ 获取 **GLM API 密钥**

## 获取 API 密钥

1. 访问 https://open.bigmodel.cn/
2. 注册或登录
3. 进入 API Keys 部分
4. 创建新的 API 密钥

API 密钥格式为: `{key_id}.{key_secret}`

## 安装

```bash
go get github.com/rexleimo/agno-go
```

## 环境设置

创建 `.env` 文件或导出环境变量:

```bash
export ZHIPUAI_API_KEY=your-key-id.your-key-secret
```

## 基础示例

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/glm"
)

func main() {
    // 创建 GLM 模型
    model, err := glm.New("glm-4", glm.Config{
        APIKey:      os.Getenv("ZHIPUAI_API_KEY"),
        Temperature: 0.7,
        MaxTokens:   1024,
    })
    if err != nil {
        log.Fatalf("创建 GLM 模型失败: %v", err)
    }

    // 创建 Agent
    agent, err := agent.New(agent.Config{
        Name:         "GLM 助手",
        Model:        model,
        Instructions: "你是一个有用的 AI 助手。",
    })
    if err != nil {
        log.Fatalf("创建 Agent 失败: %v", err)
    }

    // 运行 Agent
    output, err := agent.Run(context.Background(), "你好！请介绍一下你自己。")
    if err != nil {
        log.Fatalf("Agent 运行失败: %v", err)
    }

    fmt.Println(output.Content)
}
```

## 带工具的示例

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
    // 创建 GLM 模型
    model, err := glm.New("glm-4", glm.Config{
        APIKey:      os.Getenv("ZHIPUAI_API_KEY"),
        Temperature: 0.7,
        MaxTokens:   1024,
    })
    if err != nil {
        log.Fatal(err)
    }

    // 创建带计算器工具的 Agent
    agent, err := agent.New(agent.Config{
        Name:         "GLM 计算器助手",
        Model:        model,
        Toolkits:     []toolkit.Toolkit{calculator.New()},
        Instructions: "你是一个有用的 AI 助手，可以执行计算任务。",
    })
    if err != nil {
        log.Fatal(err)
    }

    // 测试计算
    output, err := agent.Run(context.Background(), "123 乘以 456 是多少？")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("结果: %s\n", output.Content)
}
```

## 中文语言示例

GLM 在中文语言任务上表现出色:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/glm"
)

func main() {
    model, err := glm.New("glm-4", glm.Config{
        APIKey:      os.Getenv("ZHIPUAI_API_KEY"),
        Temperature: 0.7,
    })
    if err != nil {
        log.Fatal(err)
    }

    agent, err := agent.New(agent.Config{
        Name:         "中文助手",
        Model:        model,
        Instructions: "你是一个有用的中文AI助手。",
    })
    if err != nil {
        log.Fatal(err)
    }

    // 用中文提问
    output, err := agent.Run(context.Background(), "请用中文介绍一下人工智能的发展历史。")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(output.Content)
}
```

## 运行示例

1. 克隆仓库:
```bash
git clone https://github.com/rexleimo/agno-go.git
cd HNO
```

2. 设置 API 密钥:
```bash
export ZHIPUAI_API_KEY=your-key-id.your-key-secret
```

3. 运行示例:
```bash
go run cmd/examples/glm_agent/main.go
```

## 配置选项

```go
glm.Config{
    APIKey:      string  // 必需: {key_id}.{key_secret} 格式
    BaseURL:     string  // 可选: 自定义 API 端点
    Temperature: float64 // 可选: 0.0-1.0 (默认: 0.7)
    MaxTokens:   int     // 可选: 最大响应 Token 数
    TopP:        float64 // 可选: Top-p 采样参数
    DoSample:    bool    // 可选: 启用采样
}
```

## 认证

GLM 使用 JWT (JSON Web Token) 认证:

- API 密钥被拆分为 `key_id` 和 `key_secret`
- 使用 HMAC-SHA256 签名生成 JWT token
- Token 有效期为 7 天
- SDK 自动处理

## 支持的模型

| 模型 | 上下文 | 最适合 |
|-------|---------|----------|
| `glm-4` | 128K | 通用对话、中文语言 |
| `glm-4v` | 128K | 视觉任务、多模态 |
| `glm-3-turbo` | 128K | 快速响应、成本优化 |

## 常见问题

### API 密钥格式无效

**问题**: `API key must be in format {key_id}.{key_secret}`

**解决方案**: 确保您的 API 密钥在 key_id 和 key_secret 之间包含点号(.)分隔符。

### 认证失败

**问题**: `GLM API error: Invalid API key`

**解决方案**:
- 验证您的 API 密钥是否正确
- 在 https://open.bigmodel.cn/ 检查 API 密钥是否有效
- 确保环境变量中没有多余的空格

### 速率限制

**问题**: `GLM API error: Rate limit exceeded`

**解决方案**:
- 实现指数退避的重试逻辑
- 降低请求频率
- 如需要，升级您的 API 套餐

## 下一步

- 了解更多 [Models](/zh/guide/models) LLM 选项
- 添加更多 [Tools](/zh/guide/tools) 增强能力
- 构建 [Teams](/zh/guide/team) 多 Agent 协作
- 探索 [Workflows](/zh/guide/workflow) 复杂流程

## 相关示例

- [Simple Agent](/zh/examples/simple-agent) - OpenAI 示例
- [Claude Agent](/zh/examples/claude-agent) - Anthropic 示例
- [Team Demo](/zh/examples/team-demo) - 多 Agent 协作

## 资源

- [GLM 官方网站](https://www.bigmodel.cn/)
- [GLM API 文档](https://open.bigmodel.cn/dev/api)
- [HNO 仓库](https://github.com/rexleimo/agno-go)
