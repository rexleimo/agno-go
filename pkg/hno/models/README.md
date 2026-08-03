# Models Package

This package provides the model abstraction layer and shared utilities for
implementing model providers. 本包提供模型抽象层与实现 provider 的共享工具。

## Architecture / 架构

```
agent / team / workflow  (只依赖 Model 接口)
        │
        ▼
   Model 接口 (base.go)  ← 所有 provider 实现此接口
   Invoke / InvokeStream / GetID / GetName / GetProvider
        │
        ▼
   抽象层组件 (公共复用)：
   PostJSON / PostJSONRaw (http.go)
   SSEDecoder (sse.go)
   NewToolCall / MarshalMap / ToInt / ToBool / NewToolCallID (converters.go)
   BaseModel / Model 接口 / 可选能力接口 (base.go)
        │
        ▼
   适配层：pkg/hno/models/<provider>/  (每个只写协议差异)
```

## Shared Utilities / 公共组件

### HTTP (http.go)

```go
// 同步请求：自动 marshal、设置 headers、校验状态码、解码响应
err := models.PostJSON(ctx, client, url, headers, payload, &out)

// 流式请求：返回原始响应体（调用方负责 Close 与状态码检查）
resp, err := models.PostJSONRaw(ctx, client, url, headers, payload)
```

### SSE (sse.go)

```go
decoder := models.NewSSEDecoder(resp.Body)
for {
    data, err := decoder.Next()   // io.EOF 结束；[DONE] 哨兵自动跳过
    if err != nil { ... }
    // data 是事件负载（多行 data: 已按规范拼接）
}
```

### 类型转换 (converters.go)

```go
tc := models.NewToolCall(id, name, argsJSON)          // 构造 ToolCall
s := models.MarshalMap(map[string]interface{}{...})   // map → JSON 字符串
i, ok := models.ToInt(value)                          // 任意数值/string/json.Number → int
b, ok := models.ToBool(value)                         // → bool
id := models.NewToolCallID()                          // 唯一工具调用 ID
```

### 能力接口 (base.go)

provider 可选择性实现，框架用类型断言探测：

```go
models.StructuredOutputModel  // InvokeStructured：结构化输出
models.MultimodalModel        // SupportsImages：图像输入
models.ReasoningModel         // ExtractReasoning：思维链提取
```

## Provider 目录结构

```
pkg/hno/models/<provider>/
├── <provider>.go    # struct + Config + New + Invoke + InvokeStream + ValidateConfig
├── request.go       # buildXxxRequest（InvokeRequest → 线上格式）
├── response.go      # convertResponse / convertToChunk（线上格式 → ModelResponse）
└── capabilities.go  # 可选能力接口实现
```

## 新增 Provider

完整指南见 [docs/design/provider-development-guide.md](../../docs/design/provider-development-guide.md)。

要点：
1. 新建目录 `pkg/hno/models/<provider>/`
2. 实现 `Model` 接口（嵌入 `BaseModel`，Provider 字段填 provider 名）
3. 复用公共组件（PostJSON / SSEDecoder / NewToolCall / MarshalMap），不复制实现
4. 表驱动测试 + httptest mock；有真实 API 的加集成测试（无 key 自动跳过）
5. 本地真机验证：`cmd/protocolsim`（协议模拟器）+ llama.cpp

## Providers / 已实现 provider

- **OpenAI** (`openai/`): 官方 SDK；GPT-4/GPT-3.5/函数调用/流式；结构化输出+多模态能力
- **Anthropic** (`anthropic/`): 裸 HTTP + SSE；Claude 3 系列/流式/思考模式
- **Gemini** (`gemini/`): 裸 HTTP + SSE；Gemini Pro/Ultra/函数调用
- **DeepSeek** (`deepseek/`): OpenAI 兼容 SDK；deepseek-chat/deepseek-reasoner
- **ModelScope** (`modelscope/`): OpenAI 兼容 SDK（DashScope）；Qwen 系列
- **Ollama** (`ollama/`): 裸 HTTP + JSON lines 流式；本地模型
- **GLM** (`glm/`): 智谱 GLM 系列
- **Cohere** (`cohere/`): Cohere API
- **Groq** (`groq/`): OpenAI 兼容；Groq 高速推理
- **InternLM** (`internlm/`): 书生浦语
- **LMStudio** (`lmstudio/`): OpenAI 兼容；本地 LM Studio
- **OpenRouter** (`openrouter/`): 多模型路由
- **Portkey** (`portkey/`): 网关/多提供商路由
- **SambaNova** (`sambanova/`): SambaNova 推理
- **Together** (`together/`): OpenAI 兼容
- **Vercel** (`vercel/`): Vercel AI Gateway
- **EvoLink** (`evolink/`): 多模态（image/text/video 子包）

## 本地验证工具

- `cmd/protocolsim`：协议模拟器（Anthropic Messages / Gemini generateContent / OpenAI Responses 端点 → 任意 OpenAI 兼容后端）
- `cmd/examples/simtest` / `geminitest` / `responsetest`：各协议端到端验证
- `cmd/examples/llama_test`：llama.cpp 真机验证（同步/流式/工具）
