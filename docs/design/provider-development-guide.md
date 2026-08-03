# Provider 开发指南（新增模型接入 ≤200 行）

本文档说明如何为 HNO 框架新增一个 LLM provider。目标：**新增 provider 只需写协议差异，公共逻辑（HTTP 骨架、SSE 解析、类型转换）全部复用抽象层**，核心文件 ≤200 行。

## 一、抽象层总览（你复用什么）

所有公共组件位于 `pkg/agno/models/`：

| 组件 | 位置 | 作用 |
|------|------|------|
| `PostJSON` / `PostJSONRaw` | http.go | 同步请求（自动 marshal/headers/状态码/decode）与流式请求（保留原始响应体） |
| `SSEDecoder` | sse.go | SSE 流式解析（data: 提取、event: 记录、[DONE] 跳过、任意长度行、多行 data 合并） |
| `NewToolCall` | converters.go | 统一构造框架 ToolCall（id/name/arguments → types.ToolCall） |
| `MarshalMap` | converters.go | map → JSON 字符串（nil → "{}"） |
| `ToInt` / `ToBool` | converters.go | JSON 值 → int/bool（处理所有数值类型 + string + json.Number） |
| `NewToolCallID` | converters.go | 生成唯一工具调用 ID（原子计数 + 纳秒时间戳） |
| `BaseModel` | base.go | 模型公共字段（ID/Provider/GetID/GetName/GetProvider） |
| `Model` 接口 | base.go | Invoke + InvokeStream + 元数据（实现它即可被框架使用） |
| 可选能力接口 | base.go | StructuredOutputModel / MultimodalModel / ReasoningModel（能力探测） |

## 二、文件结构（新 provider 目录约定）

```
pkg/agno/models/<provider>/
├── <provider>.go    # 核心：struct + Config + New + Invoke + InvokeStream + ValidateConfig
├── request.go       # 请求构建：buildXxxRequest（InvokeRequest → 线上格式）
├── response.go      # 响应解析：convertResponse / convertToChunk（线上格式 → ModelResponse）
└── capabilities.go  # 可选能力接口实现（结构化输出/多模态等）
```

- 每个文件 ≤500 行；总代码量 ≤200 行核心逻辑（不含类型定义）
- 参考实现：`openai/`（SDK 客户端模式）、`anthropic/`（裸 HTTP + SSE 模式）、`gemini/`（裸 HTTP + SSE + query key 模式）

## 三、五种接入模式（按协议形态选择）

### 模式 1：OpenAI 兼容（最常见，如 deepseek/groq/lmstudio）
用 go-openai SDK，只需改 BaseURL + 模型名：

```go
// <provider>.go
package foo

import (
	"context"
	"net/http"
	"time"

	"github.com/rexleimo/agno-go/pkg/agno/models"
	"github.com/rexleimo/agno-go/pkg/agno/types"
	"github.com/sashabaranov/go-openai"
)

type Foo struct {
	models.BaseModel
	client *openai.Client
	config Config
}

type Config struct {
	APIKey      string
	BaseURL     string
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration
}

func New(modelID string, config Config) (*Foo, error) {
	if config.APIKey == "" {
		return nil, types.NewInvalidConfigError("Foo API key is required", nil)
	}
	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	clientConfig.HTTPClient = &http.Client{Timeout: timeout}
	return &Foo{
		BaseModel: models.BaseModel{ID: modelID, Provider: "foo"},
		client:    openai.NewClientWithConfig(clientConfig),
		config:    config,
	}, nil
}

func (f *Foo) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	chatReq := f.buildChatRequest(req)
	resp, err := f.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return nil, types.NewAPIError("failed to call Foo API", err)
	}
	if len(resp.Choices) == 0 {
		return nil, types.NewAPIError("no response from Foo", nil)
	}
	choice := resp.Choices[0]
	return &types.ModelResponse{
		ID:      resp.ID,
		Content: choice.Message.Content,
		Model:   resp.Model,
		Usage: types.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		Metadata: types.Metadata{FinishReason: string(choice.FinishReason)},
		ToolCalls: toToolCalls(choice.Message.ToolCalls),
	}, nil
}
```

### 模式 2：裸 HTTP 同步（无 SDK，如 anthropic）
用共享 `PostJSON`，Invoke 只需 3 行：

```go
func (a *Anthropic) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	claudeReq := a.buildClaudeRequest(req)
	var claudeResp ClaudeResponse
	err := models.PostJSON(ctx, a.httpClient, a.config.BaseURL+"/messages", a.authHeaders(), claudeReq, &claudeResp)
	if err != nil {
		return nil, err
	}
	return a.convertResponse(&claudeResp), nil
}
```

### 模式 3：SSE 流式（anthropic/gemini 模式）
用共享 `SSEDecoder`，流式解析统一：

```go
decoder := models.NewSSEDecoder(resp.Body)
for {
	data, err := decoder.Next()
	if err != nil {
		if err != io.EOF {
			chunks <- types.ResponseChunk{Done: true, Error: err}
		}
		return
	}
	// 解析 data 为你的线上响应类型，转换后发送 chunk
}
```

### 模式 4：JSON lines 流式（ollama 模式）
每行一个完整 JSON 对象（无 data: 前缀），用 `json.NewDecoder` 逐行：

```go
decoder := json.NewDecoder(resp.Body)
for {
	var streamResp OllamaResponse
	if err := decoder.Decode(&streamResp); err != nil {
		if err != io.EOF { chunks <- types.ResponseChunk{Done: true, Error: err} }
		return
	}
	chunks <- types.ResponseChunk{Content: streamResp.Message.Content, Done: streamResp.Done}
}
```

### 模式 5：工具调用转换
响应里的工具调用统一用 `models.NewToolCall` 构造，参数用 `models.MarshalMap`：

```go
modelResp.ToolCalls = append(modelResp.ToolCalls, models.NewToolCall(
	models.NewToolCallID(),      // 或线上返回的真实 ID
	part.FunctionCall.Name,
	models.MarshalMap(part.FunctionCall.Args),
))
```

## 四、新增 provider 的步骤（Checklist）

1. **建目录**：`pkg/agno/models/<provider>/`，按上述文件结构拆分
2. **写 Config + New**：校验 APIKey、设置默认 BaseURL/Timeout；嵌入 `models.BaseModel`（Provider 字段填 provider 名）
3. **写 buildXxxRequest**：InvokeRequest.Messages/Tools/Temperature/MaxTokens → 线上格式
4. **写 Invoke**：选模式 1（SDK）或模式 2（PostJSON）
5. **写 InvokeStream**：选模式 3（SSE）或模式 4（JSON lines）；工具调用增量用 partialToolCall 累积（参考 openai）
6. **写 convertResponse / convertToChunk**：文本、工具调用（NewToolCall）、usage、reasoning 提取
7. **能力接口（可选）**：实现 StructuredOutputModel/MultimodalModel/ReasoningModel
8. **测试**：表驱动测试 + httptest mock；有真实 API 的写集成测试（无 key 自动跳过）
9. **验证**：`go build ./pkg/agno/models/<provider>/` + `go test ./pkg/agno/models/<provider>/` + `go vet`
10. **注册**：在 `models/README.md` 的 provider 列表中加入新条目

## 五、验收标准

- [ ] 核心文件 ≤500 行，单个 provider 新增逻辑 ≤200 行
- [ ] 复用公共组件：PostJSON / SSEDecoder / NewToolCall / MarshalMap / ToInt / ToBool（不复制实现）
- [ ] 表驱动测试覆盖：对话、流式、工具调用、错误处理
- [ ] GoDoc 注释完整（英文注释 + 中文说明，与现有 provider 一致）
- [ ] `go vet` 无警告，`go build` 通过
- [ ] 行为可验证：本地用 protocolsim + llama.cpp 真机跑通（见 `cmd/examples/simtest`）

## 六、本地验证工具

- `cmd/protocolsim`：本地协议模拟器（Anthropic/Gemini/Responses 端点 → llama.cpp）
- `cmd/examples/simtest` / `geminitest` / `responsetest`：各协议端到端验证示例
- 真机流程：llama.cpp 起 18080 → protocolsim 起 16000 → provider 指向模拟器跑 Agent
