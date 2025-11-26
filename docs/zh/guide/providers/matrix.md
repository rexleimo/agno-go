
# 模型供应商能力矩阵

本页按高层粒度汇总 Agno-Go 支持的各模型供应商的能力和配置要点，帮助你快速判断：

- 哪些供应商可用于对话（chat）。  
- 哪些供应商可用于向量嵌入（embedding，视支持情况而定）。  
- 哪些供应商支持流式输出。  
- 每个供应商需要配置哪些环境变量。  

> 具体可用的模型与能力取决于你在各供应商处的账号、区域与配额。以下信息以当前 Go 适配器与 `.env.example` 为准，详细限制请以各供应商官方文档为准。

## 总览表

| 供应商      | 对话支持                  | 嵌入支持                       | 流式支持                         | 关键环境变量                                                                      |
|-------------|---------------------------|--------------------------------|----------------------------------|-----------------------------------------------------------------------------------|
| OpenAI      | 支持（chat）             | 支持（embeddings）             | 支持（chat 流式）                | `OPENAI_API_KEY`, `OPENAI_ENDPOINT`, `OPENAI_ORG`, `OPENAI_API_VERSION`           |
| Gemini      | 支持（chat）             | 支持（embeddings）             | 支持（chat 流式）                | `GEMINI_API_KEY`, `GEMINI_ENDPOINT`, `VERTEX_PROJECT`, `VERTEX_LOCATION`          |
| GLM4        | 支持（chat）             | 限制/规划中*                   | 视具体模型而定                   | `GLM4_API_KEY`, `GLM4_ENDPOINT`                                                   |
| OpenRouter  | 支持（chat/路由）        | 取决于底层模型是否支持         | 取决于底层模型是否支持           | `OPENROUTER_API_KEY`, `OPENROUTER_ENDPOINT`                                       |
| SiliconFlow | 支持（chat）             | 支持（embeddings）             | 支持（chat 流式）                | `SILICONFLOW_API_KEY`, `SILICONFLOW_ENDPOINT`                                     |
| Cerebras    | 支持（chat）             | 视官方支持情况而定             | 视官方支持情况而定               | `CEREBRAS_API_KEY`, `CEREBRAS_ENDPOINT`                                           |
| ModelScope  | 支持（chat）             | 视官方支持情况而定             | 视官方支持情况而定               | `MODELSCOPE_API_KEY`, `MODELSCOPE_ENDPOINT`                                       |
| Groq        | 支持（chat）             | 限制/规划中*                   | 支持（chat 流式）                | `GROQ_API_KEY`, `GROQ_ENDPOINT`                                                   |
| Ollama      | 支持（本地 chat）        | 取决于本地模型是否实现嵌入     | 支持（本地 chat 流式）           | `OLLAMA_ENDPOINT`                                                                 |

`*` 部分供应商的 embedding 支持仍在演进中。在尚未完全支持或仅对部分模型开放的情况下，Go 适配器会在测试中跳过不支持的调用，或在契约文档中记录偏差。

## 配置说明

- 所有供应商相关的环境变量均在 `.env.example` 中列出。复制为 `.env` 后，仅填入你确实需要启用的供应商。  
- 必需 key 留空时，健康检查与供应商测试会跳过该供应商并给出跳过原因；运行时不会主动调用未配置的供应商。  
- 类似 `OPENAI_ENDPOINT`、`GEMINI_ENDPOINT` 的变量默认指向官方托管 API，你可以根据需要改为私有网关或代理。  
- `OLLAMA_ENDPOINT` 通常指向本地运行的 Ollama/vLLM 实例（例如 `http://localhost:11434/v1`），仅在你显式启用本地模型时使用。  

关于路由逻辑与错误规约，请结合 **核心功能与 API 概览** 页面以及 specs 目录中的契约文档一起阅读。

## 下一步

- 查看[配置与安全实践](../config-and-security)，进一步了解本页中列出的环境变量含义以及推荐的密钥管理方式。  
- 若希望基于不同供应商扩展 Quickstart 示例，可以回到[快速开始](../quickstart) 并根据本页矩阵调整配置。  
