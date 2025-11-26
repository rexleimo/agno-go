## 配置与安全实践

本页介绍如何配置 Agno-Go，以及在使用各模型供应商密钥时需要遵循的安全实践。示例假设你在仓库根目录使用 Go 1.25.1 运行服务，并使用默认的配置文件。

### 1. 配置文件与运行环境

- `config/default.yaml`：服务器、供应商、内存与运行时的默认配置。  
- `.env`：存放各模型供应商的 API Key 与自定义端点。  
- `.env.example`：安全的示例模板，可复制后在本地填写。  

推荐工作流程：

1. 复制示例环境文件：

   ```bash
   cp .env.example .env
   ```

2. 为需要启用的供应商填写对应的 Key（例如 OpenAI 或 Groq）。  
3. 使用默认配置启动服务：

   ```bash
   cd go
   go run ./cmd/agno --config ../config/default.yaml
   ```

### 2. 核心环境变量

`.env.example` 中列出了所有受支持的供应商。常用变量包括：

- **OpenAI**
  - `OPENAI_API_KEY`：启用 OpenAI 所必需。  
  - `OPENAI_ENDPOINT`：可选，用于代理或 Azure 风格端点。  
  - `OPENAI_ORG`、`OPENAI_API_VERSION`：可选，用于组织范围或预览 API。  

- **Gemini / Vertex**
  - `GEMINI_API_KEY`：直接使用 Gemini API 时需要。  
  - `GEMINI_ENDPOINT`：可选，默认为公开的 Generative Language API。  
  - `VERTEX_PROJECT`、`VERTEX_LOCATION`：可选，使用 Vertex AI 时设置。  

- **GLM4**
  - `GLM4_API_KEY`：启用 GLM4 所必需。  
  - `GLM4_ENDPOINT`：默认公开端点，可按需替换为代理。  

- **OpenRouter**
  - `OPENROUTER_API_KEY`：启用 OpenRouter 所必需。  
  - `OPENROUTER_ENDPOINT`：可选，用于自定义路由。  

- **SiliconFlow / Cerebras / ModelScope / Groq**
  - `SILICONFLOW_API_KEY`、`CEREBRAS_API_KEY`、`MODELSCOPE_API_KEY`、`GROQ_API_KEY`：对应供应商的必需密钥。  
  - `SILICONFLOW_ENDPOINT`、`CEREBRAS_ENDPOINT`、`MODELSCOPE_ENDPOINT`、`GROQ_ENDPOINT`：可选端点覆盖。  

- **Ollama / 本地模型**
  - `OLLAMA_ENDPOINT`：指向本地模型服务的 HTTP 端点；留空则视为禁用。  

约定行为：

- 必需密钥留空时，对应供应商会被标记为 “未配置”；  
- 健康检查与供应商测试会跳过这些供应商，并给出明确的跳过原因。  

### 3. `config/default.yaml` 概览

默认配置文件控制服务运行方式：

- **server**
  - `server.host`：监听地址（默认 `0.0.0.0`）。  
  - `server.port`：HTTP API 端口（默认 `8080`）。  

- **providers**
  - `providers.<name>.endpoint`：从对应的环境变量读取，例如 `${OPENAI_ENDPOINT}`、`${GROQ_ENDPOINT}`。  
  - 是否启用某个供应商由 env 中的 Key 决定；缺失 Key 的供应商应被视为禁用。  

- **memory**
  - `memory.storeType`：`memory` / `bolt` / `badger`。  
  - `memory.tokenWindow`：对话上下文窗口中保留的 token 数。  

- **runtime / bench**
  - `runtime.maxConcurrentRequests`、`runtime.requestTimeout`、`runtime.router.*` 等字段控制并发与超时策略。  
  - `bench` 段用于内部基准测试的默认参数（普通使用场景一般不需要修改）。  

推荐做法是将 `config/default.yaml` 保持在版本控制中，通过 `.env` 或部署环境的环境变量覆盖敏感值或不同环境的差异。

### 4. 安全实践

- **不要提交真实密钥**
  - 不要将 `.env` 或包含真实 API Key 的文件提交到版本库。  
  - 仓库的 `.gitignore` 已经忽略 `.env` 和常见的本地配置文件。  

- **在文档与示例中使用占位符**
  - 在 shell 示例中使用 `OPENAI_API_KEY=...` 这类占位写法，而不是粘贴真实密钥。  
  - 在代码示例中使用相对路径（例如 `./config/default.yaml`），避免出现本机绝对路径。  

- **区分环境**
  - 尽量为开发、预发布、生产环境使用不同的凭据或项目。  
  - 倾向使用云平台的 Secrets 管理或 CI 的密钥存储，而不是长期直接暴露在环境变量中。  

- **审计与测试**
  - 在提交改动前，建议运行：

    ```bash
    make test providers-test coverage bench constitution-check
    ```

  - 这些命令有助于确保供应商集成仍按预期工作，且没有引入不安全的配置变更。  
