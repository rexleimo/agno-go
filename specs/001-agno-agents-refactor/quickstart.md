# Quickstart - 001-agno-agents-refactor

## 前置

1) 安装 Go 1.25.1；确保 `make` 可用。  
2) 复制 `.env.example` 为 `.env`，设置 `AGNO_API_KEY`（用于 HTTP/CLI `X-API-Key` 认证）并按需填入供应商 key/endpoint/model；未配置的供应商会被跳过。  
3) 推荐：`GOCACHE=$PWD/.cache/go-build` 复用构建缓存。

## 开发与验证

1) 格式/静态检查：`make fmt lint`  
2) 单元+契约：`make test`（含 go/tests/contract）  
3) 供应商集成：`make providers-test`（跳过缺 key/不可达，并写入 artifacts/coverage/providers.log）  
4) 覆盖率：`make coverage`（综合 ≥85%）  
5) 基准：`make bench`（记录 p95 与峰值内存，存入 specs/001-agno-agents-refactor/artifacts/）  
6) 治具：`make gen-fixtures`（或 `./scripts/gen-fixtures.sh`，VERIFY_ONLY=true 可仅校验）  
7) 聚合阻断：`go run ./go/scripts/coverage/aggregate_basics --feature-dir specs/001-agno-agents-refactor`（汇总 Basics 场景、供应商与覆盖率，如未达标会向 `artifacts/coverage.txt` 写入差异并返回非零）

## 运行示例（Basics 聚焦）

- 基础 Agent/Tools/HITL：`go run ./go/cmd/agno --config config/default.yaml`，或使用内置 stub provider 直接回放 Basics 场景：`go run ./go/cmd/agno --config config/default.yaml --scenario basic --fixtures specs/001-agno-agents-refactor/contracts/fixtures`（支持 basic|memory|rag|tool|workflow）。契约/CLI 场景测试：`go test ./go/tests/contract -run Basics`。  
- 知识/RAG：运行知识示例脚本，确认嵌入与检索路径；若向量库不可用，回退策略需在日志中说明（已在 `knowledge/fallback.go` 提供提示/错误策略）。  
- 多模态：启用支持多模态的模型/工具，验证图像/音频输入输出大小限制与错误提示（工作流/engine 占位，按需扩展）。
- 供应商演示/日志：`go run ./go/cmd/agno --demo --providers openai,gemini --parallel --providers-log specs/001-agno-agents-refactor/artifacts/coverage/providers.log`，用于并行探测 chat/stream/embed 状态。命令会读取 `.env` 中的密钥与 `*_CHAT_MODEL`/`*_EMBED_MODEL`，跳过缺密钥/不可达的供应商并在 `providers.log` 记录原因，同时在 `artifacts/coverage.txt` 输出 summary。

## 文档（VitePress）

1) 按照实现改动更新 Basics 相关章节（示例、参数、跳过策略、差异说明）。  
2) 运行 VitePress 构建（命令根据文档仓库定义，如 `npm run docs:build`），确保无错误。  
3) 将文档更新与实现一起提交/发布，记录所需 env 与已知偏差。
