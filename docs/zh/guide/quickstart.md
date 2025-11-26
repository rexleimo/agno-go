# 快速开始：Basics 场景（对齐 docs.agno.com/basics）

范围：围绕 `./go` 模块的 Basics 五场景（basic / memory / rag / tool+HITL / workflow），Go 版本 1.25.1。运行时命令在 `go/` 内执行，Make 目标从仓库根目录执行。

## 1）前置与环境

仓库根目录：
```bash
cp .env.example .env
```
进入 Go 模块根：
```bash
cd go
export GOCACHE=$PWD/../.cache/go-build   # 可选
```
- `AGNO_API_KEY` 必填（仅 `X-API-Key`；FR-004）。
- 供应商变量（`*_API_KEY`、`*_CHAT_MODEL`、`*_EMBED_MODEL`）可选；缺失视为未配置，demo/测试会跳过并记录原因。
- 配置：`../config/default.yaml`（路由超时/重试/并发；memory store `memory|bolt|badger`；端点读取 env）。

## 2）运行五个 Basics 场景（CLI + 治具）

选择一个场景：
```bash
cd go
go run ./cmd/agno --config ../config/default.yaml \
  --scenario <basic|memory|rag|tool|workflow> \
  --fixtures ../specs/001-agno-agents-refactor/contracts/fixtures
```
- `basic`：单轮 agent，stub 回放。
- `memory`：多轮会话，历史写入 MemoryStore。
- `rag`：检索不可用 → 提示/错误回退。
- `tool`：工具 + HITL + 守护（含流式事件）。
- `workflow`：分支/工作流占位，预留多模态钩子。

## 3）测试与演示

- 契约测试（目标 ≥95%）：
  ```bash
  cd go
  go test ./tests/contract -run Basics
  ```
- 供应商探测（按 env 兜底跳过）：
  ```bash
  cd go
  go run ./cmd/agno --demo --providers openai,gemini \
    --parallel --providers-log ../specs/001-agno-agents-refactor/artifacts/coverage/providers.log
  ```
- 全量回归 + 文档构建（仓库根）：
  ```bash
  make constitution-check
  ```
  工件：`specs/001-agno-agents-refactor/artifacts/coverage.txt`、`bench.txt`、`coverage/providers.log`。

## 4）Provider 客户端示例（Go）

客户端位于 `go/pkg/providers/<provider>`。示例（OpenAI）：
```go
package main

import (
  "context"
  "log"
  "os"
  "time"

  "github.com/rexleimo/agno-go/internal/agent"
  "github.com/rexleimo/agno-go/internal/model"
  "github.com/rexleimo/agno-go/pkg/providers/openai"
)

func main() {
  ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
  defer cancel()
  apiKey := os.Getenv("OPENAI_API_KEY")
  if apiKey == "" {
    log.Fatal("OPENAI_API_KEY not set")
  }
  client := openai.New("", apiKey, nil)
  resp, err := client.Chat(ctx, model.ChatRequest{
    Model:    agent.ModelConfig{Provider: agent.ProviderOpenAI, ModelID: "gpt-4o-mini", Stream: false},
    Messages: []agent.Message{{Role: agent.RoleUser, Content: "用一句话介绍 Agno-Go。"}},
  })
  if err != nil {
    log.Fatalf("chat error: %v", err)
  }
  log.Println("assistant:", resp.Message.Content)
}
```
其他供应商按同样形状调用 `pkg/providers/<provider>`，遵循各自必需的 env。

## 5）回退、守护与认证

- RAG 回退：`knowledge.Engine` 提示模式返回指导但不写历史；错误模式抛出 `ErrUnavailable` 且不污染状态。
- 守护：PII / Prompt 注入 → `ErrGuardrailViolation`，不写入历史（见 `go/tests/contract/security_contract_test.go`）。
- 认证（FR-004）：仅 `X-API-Key`，Basic/Bearer/OAuth/自定义 Authorization 拒绝（见 `go/tests/contract/auth_middleware_negative_test.go`）。

## 6）供应商跳过与偏差

- 缺密钥/不可达会记录为 skipped（`specs/001-agno-agents-refactor/artifacts/coverage/providers.log`）。
- Python 差异（多模态/流式治具占位等）在 `specs/001-agno-agents-refactor/contracts/deviations.md`。
- 基准对比 `specs/001-agno-agents-refactor/artifacts/baseline/python-bench.json`（如有）；当前 Go 样本在 `specs/001-agno-agents-refactor/artifacts/bench.txt`，目标 p95 -20%、峰值内存 -25%。

## 7）构建文档（VitePress）

仓库根目录：
```bash
cd docs
npm run docs:build    # 首次需要先 npm install
# 或：
DOCS_DIR=$(pwd)/docs make docs
```
输出：`docs/.vitepress/dist`；Make 日志：`specs/001-agno-agents-refactor/artifacts/logs/docs.log`。
