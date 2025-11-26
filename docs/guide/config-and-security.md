## Configuration & Security Practices

This page explains how to configure Agno-Go and handle provider credentials safely.
It assumes you are running the server from the repository root with Go 1.25.1 and
using the default configuration file.

### 1. Configuration files and environment

- `config/default.yaml` – default server, provider, memory and runtime settings.  
- `.env` – environment variables for provider API keys and custom endpoints.  
- `.env.example` – a safe template you can copy and fill locally.  

Typical workflow:

1. Copy the example file:

   ```bash
   cp .env.example .env
   ```

2. Fill in the keys for the providers you want to enable (for example OpenAI or Groq).
3. Start the server with the default configuration:

   ```bash
   cd go
   go run ./cmd/agno --config ../config/default.yaml
   ```

### 2. Core environment variables

The `.env.example` file lists all supported providers. The most important variables are:

- **OpenAI**
  - `OPENAI_API_KEY` – required to enable OpenAI.
  - `OPENAI_ENDPOINT` – optional; override when using a proxy or Azure-style endpoint.
  - `OPENAI_ORG`, `OPENAI_API_VERSION` – optional; used for org scoping or preview APIs.

- **Gemini / Vertex**
  - `GEMINI_API_KEY` – required for direct Gemini API usage.
  - `GEMINI_ENDPOINT` – optional; defaults to the public Generative Language API.
  - `VERTEX_PROJECT`, `VERTEX_LOCATION` – optional; set when using Vertex AI.

- **GLM4**
  - `GLM4_API_KEY` – required to enable GLM4.
  - `GLM4_ENDPOINT` – default public endpoint; override for proxies if needed.

- **OpenRouter**
  - `OPENROUTER_API_KEY` – required to enable OpenRouter.
  - `OPENROUTER_ENDPOINT` – optional; override for custom routing.

- **SiliconFlow, Cerebras, ModelScope, Groq**
  - `SILICONFLOW_API_KEY`, `CEREBRAS_API_KEY`,
    `MODELSCOPE_API_KEY`, `GROQ_API_KEY` – required keys.
  - `SILICONFLOW_ENDPOINT`, `CEREBRAS_ENDPOINT`,
    `MODELSCOPE_ENDPOINT`, `GROQ_ENDPOINT` – optional endpoint overrides.

- **Ollama / local models**
  - `OLLAMA_ENDPOINT` – defaults to a local HTTP endpoint; set to the base URL of
    your local model server or leave empty to disable.

Leaving required keys empty will mark the provider as not-configured. Health checks and
provider tests are expected to skip such providers with a clear reason.

### 3. `config/default.yaml` overview

The default configuration file controls how the server runs:

- **Server**
  - `server.host` – listen address (default `0.0.0.0`).
  - `server.port` – port for the HTTP API (default `8080`).

- **Providers**
  - `providers.<name>.endpoint` – reads from the corresponding env var, for example
    `${OPENAI_ENDPOINT}` or `${GROQ_ENDPOINT}`.
  - Provider enablement is based on env keys; a missing key should disable that provider.

- **Memory**
  - `memory.storeType` – one of `memory`, `bolt`, `badger`.
  - `memory.tokenWindow` – how many tokens to keep in the rolling context window.

- **Runtime and benchmarking**
  - `runtime.maxConcurrentRequests`, `runtime.requestTimeout` and router-related fields
    control concurrency and timeouts.
  - `bench` contains defaults for internal benchmark runs (not required for basic use).

You can keep `config/default.yaml` under version control and only override values via
`.env` or environment variables in your deployment environment.

### 4. Security best practices

- **Never commit real secrets**
  - Do not commit `.env` or any file containing real API keys.
  - This repository’s `.gitignore` already excludes `.env` and typical local config files.

- **Use placeholders in documentation and examples**
  - When adapting examples, use placeholders such as `OPENAI_API_KEY=...` rather than
    pasting real keys into shell history or shared snippets.
  - In code samples, keep paths relative (for example `./config/default.yaml`) instead of
    using machine-specific absolute paths.

- **Separate environments**
  - Use different keys or projects for development, staging and production where possible.
  - Consider using your platform’s secret manager or CI secret storage instead of raw
    environment variables for long-lived credentials.

- **Auditing and tests**
  - Before contributing changes, run:

    ```bash
    make test providers-test coverage bench constitution-check
    ```

  - These commands help ensure that provider integrations still behave as expected and
    that no unsafe configuration changes were introduced.
