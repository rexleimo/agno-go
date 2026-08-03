# Release Notes - Agno-Go v1.0.2

**Release Date:** 2025-10-05
**Type:** Feature Release
**Status:** ✅ Production Ready

---

## 🎯 Overview

Agno-Go v1.0.2 adds support for **GLM (智谱AI)**, China's leading domestic LLM platform, bringing the total number of supported LLM providers to **7**. This release includes comprehensive JWT authentication, streaming support, and full tool calling capabilities.

---

## ✨ What's New

### New LLM Provider: GLM (智谱AI)

We're excited to announce full integration with Zhipu AI's GLM models, providing first-class support for Chinese language AI applications.

#### Supported Models
- **GLM-4** - Main conversational model
- **GLM-4V** - Vision-enabled multimodal model
- **GLM-3-Turbo** - Faster, cost-effective model

#### Key Features
- ✅ **Custom JWT Authentication** - Secure HMAC-SHA256 token signing
- ✅ **Synchronous API** - `Invoke()` method for standard calls
- ✅ **Streaming Support** - `InvokeStream()` for real-time responses
- ✅ **Tool Calling** - Full function calling integration
- ✅ **Type Safety** - Strongly-typed API with Go's type system
- ✅ **Error Handling** - Custom error types for better debugging
- ✅ **Bilingual** - All code comments in English/中文

---

## 📦 Installation

### Go Get
```bash
go get github.com/rexleimo/agno-go@v1.0.2
```

### Update Existing Installation
```bash
go get -u github.com/rexleimo/agno-go
```

---

## 🚀 Quick Start

### 1. Get Your API Key

Sign up at [https://open.bigmodel.cn/](https://open.bigmodel.cn/) to get your GLM API key.

The key format is: `{key_id}.{key_secret}`

### 2. Set Environment Variable

```bash
export ZHIPUAI_API_KEY=your-key-id.your-key-secret
```

### 3. Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/agno-go/pkg/hno/models/glm"
    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
    "github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
    // Create GLM model
    model, err := glm.New("glm-4", glm.Config{
        APIKey:      os.Getenv("ZHIPUAI_API_KEY"),
        Temperature: 0.7,
        MaxTokens:   1024,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create agent with GLM
    agent, err := agent.New(agent.Config{
        Name:         "GLM Assistant",
        Model:        model,
        Toolkits:     []toolkit.Toolkit{calculator.New()},
        Instructions: "你是一个有用的AI助手。",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Run agent
    output, err := agent.Run(context.Background(), "你好！请计算 123 * 456")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(output.Content)
}
```

### 4. Run the Example

```bash
go run cmd/examples/glm_agent/main.go
```

---

## 🔧 Technical Details

### Architecture

The GLM integration follows Agno-Go's clean architecture principles:

```
pkg/hno/models/glm/
├── glm.go          # Main model implementation (410 lines)
├── auth.go         # JWT authentication logic (59 lines)
├── types.go        # API type definitions (105 lines)
├── glm_test.go     # Comprehensive unit tests (320 lines)
└── README.md       # Usage documentation
```

### Authentication Flow

GLM uses a custom JWT authentication mechanism:

1. **API Key Parsing** - Split `{key_id}.{key_secret}` format
2. **JWT Generation** - Create token with claims: `api_key`, `timestamp`, `exp`
3. **HMAC Signing** - Sign with HS256 algorithm using `key_secret`
4. **Header Injection** - Send via `Authorization: Bearer {token}`

Tokens are valid for **7 days** and automatically regenerated for each request.

### API Compatibility

The GLM API follows an OpenAI-compatible structure, making it easy to integrate:

- Similar request/response format
- Tool calling with same schema
- Streaming via Server-Sent Events (SSE)
- Compatible error handling

---

## 📊 Test Coverage

### GLM Package Tests

```
✅ TestParseAPIKey           - API key format validation
✅ TestGenerateJWT            - JWT token generation
✅ TestNew                    - Model constructor
✅ TestBuildGLMRequest        - Request building
✅ TestInvoke                 - Synchronous API calls
✅ TestInvokeError            - Error handling
✅ TestConvertToModelResponse - Response conversion
```

**Results:**
- All 7 tests passing ✅
- 57.2% code coverage
- Race detector: PASS
- Build verification: SUCCESS

---

## 📝 Configuration Options

### GLM Config

```go
type Config struct {
    APIKey      string  // Required: {key_id}.{key_secret}
    BaseURL     string  // Optional: Custom endpoint
    Temperature float64 // Optional: 0.0-1.0
    MaxTokens   int     // Optional: Max tokens to generate
    TopP        float64 // Optional: Top-p sampling
    DoSample    bool    // Optional: Enable sampling
}
```

### Environment Variables

```bash
# Required
export ZHIPUAI_API_KEY=your-key-id.your-key-secret

# Optional (uses default if not set)
export ZHIPUAI_BASE_URL=https://open.bigmodel.cn/api/paas/v4
```

---

## 🌍 Supported LLM Providers

Agno-Go now supports **7 major LLM providers**:

| Provider | Models | Coverage | Status |
|----------|--------|----------|--------|
| OpenAI | GPT-4, GPT-3.5, GPT-4 Turbo | 44.6% | ✅ |
| Anthropic | Claude 3.5 Sonnet, Opus, Haiku | 50.9% | ✅ |
| **GLM** | **GLM-4, GLM-4V, GLM-3-Turbo** | **57.2%** | ✅ **NEW** |
| Ollama | Llama, Mistral, CodeLlama | 43.8% | ✅ |
| DeepSeek | DeepSeek-V2, DeepSeek-Coder | - | ✅ |
| Google | Gemini Pro, Flash | - | ✅ |
| ModelScope | Qwen, Yi models | - | ✅ |

---

## 📚 Documentation

### New Documentation

- **pkg/hno/models/glm/README.md** - Comprehensive GLM usage guide
  - Quick start examples
  - Configuration reference
  - Authentication details
  - Error handling
  - API compatibility notes

### Updated Documentation

- **README.md** - Added GLM to supported models with code examples
- **CLAUDE.md** - Added GLM environment variables and configuration
- **CHANGELOG.md** - Complete v1.0.2 changelog

### Examples

- **cmd/examples/glm_agent/main.go** - Full GLM integration example
  - Simple conversation
  - Calculator tool usage
  - Multi-step calculations
  - Chinese language support

---

## 🔄 Migration Guide

### From v1.0.0/v1.0.1 to v1.0.2

No breaking changes! This is a **feature-only release**.

#### New Features Available

1. **GLM Model Support** - Add GLM to your existing agents
2. **JWT Dependency** - Automatically added via `go get`

#### To Start Using GLM

```go
// Before (v1.0.0)
import "github.com/rexleimo/agno-go/pkg/hno/models/openai"
model, _ := openai.New("gpt-4", config)

// After (v1.0.2) - GLM option now available
import "github.com/rexleimo/agno-go/pkg/hno/models/glm"
model, _ := glm.New("glm-4", glm.Config{
    APIKey: os.Getenv("ZHIPUAI_API_KEY"),
})
```

---

## 📦 Dependencies

### New Dependencies

- **github.com/golang-jwt/jwt/v5** v5.3.0 - For GLM JWT authentication

### No Breaking Changes

All existing dependencies remain unchanged.

---

## 🐛 Known Issues

None at this time. All tests passing.

---

## 🎉 What's Next

### Planned for v1.0.3

- Additional GLM-specific features (web search tool)
- Improved streaming support for tool calls
- More comprehensive integration tests with real API

### Long-term Roadmap

- More LLM providers
- Advanced RAG features
- Performance optimizations
- Enhanced observability

---

## 🙏 Credits

Special thanks to:
- **Zhipu AI** for providing excellent API documentation
- **Community contributors** for testing and feedback
- **Agno Python team** for the original framework design

---

## 📞 Support

- **Documentation**: [https://github.com/rexleimo/agno-go](https://github.com/rexleimo/agno-go)
- **Issues**: [https://github.com/rexleimo/agno-go/issues](https://github.com/rexleimo/agno-go/issues)
- **Discussions**: [https://github.com/rexleimo/agno-go/discussions](https://github.com/rexleimo/agno-go/discussions)

---

## 📜 License

MIT License - see [LICENSE](../LICENSE) for details

---

**Enjoy building AI agents with GLM! 🚀**
