# T032 Basics 文档更新草稿

本草稿用于同步到外部文档仓库（目标路径 `../rex-agno/agno-Go/docs` 的 Basics 章节）。内容覆盖运行步骤、环境变量、差异说明与回退/安全路径。未找到目标仓库时，可用 `DOCS_DIR=/path/to/docs make docs` 定位路径并复用 Make 任务。

## 如何运行 Basics（Go 侧）
- 本地 CLI/SDK：`go run ./go/cmd/agno --config config/default.yaml --scenario {basic|memory|rag|tool|workflow} --fixtures specs/001-agno-agents-refactor/contracts/fixtures`；默认为 SSE/流式，可切换 stub provider 回放契约。
- 契约验证：`go test ./go/tests/contract -run Basics`（五个场景 >=95% 通过视为达标）。
- 供应商探测：`go run ./go/cmd/agno --demo --providers openai,gemini --parallel --providers-log specs/001-agno-agents-refactor/artifacts/coverage/providers.log`；缺密钥/不可达会跳过并写明原因。
- 回归套件：`make fmt lint test providers-test coverage bench constitution-check`（含 `make docs`，输出集中于 `specs/001-agno-agents-refactor/artifacts/`）。

## 环境与配置
- 认证：仅接受 `X-API-Key` 头，值来自 `.env` 的 `AGNO_API_KEY`；Basic/OAuth/JWT/自定义头会被拒绝并返回 401（见 `go/internal/runtime/middleware` 与契约测试）。
- 供应商：`.env.example` 已列出九家所需变量与默认模型（`*_CHAT_MODEL`/`*_EMBED_MODEL`），缺 key 时标记为 not-configured 并在 providers.log 记录跳过；Ollama 需本地可达 `OLLAMA_ENDPOINT`。
- 配置加载：`config/default.yaml` 读取 env 并设定 router 超时/重试/并发与 memory store（memory/bolt/badger）；推荐 `GOCACHE=$PWD/.cache/go-build` 复用构建缓存。

## 与 Python 版的差异/已知缺口
- Basics 契约治具目前以占位/形状为主，尚未覆盖真实多模态/流式响应；替换治具后需同步 `contracts/deviations.md`。
- providers 基线多因缺 key/模型不可用而跳过或返回 4xx（详见 `specs/001-agno-agents-refactor/contracts/deviations.md` 与 `artifacts/coverage/providers.log`）；Groq/Gemini/GLM4 默认模型需按最新可用型号调整。
- 覆盖率/性能：`artifacts/coverage.txt` 仍显示 aggregate 未达 95%/90%/85% 阈值，`artifacts/bench.txt` 仅含当前 Go 侧样本，Python 基线待补；文档需提示这些达标线与阻断策略。

## 回退与安全说明
- 知识库不可用：`knowledge.Engine` 支持 Hint 模式（返回提示信息，不污染历史）与 Error 模式（返回 `ErrUnavailable`）；均不会写入会话记录。
- 守护/防护：PII 与 Prompt 注入会抛出 `ErrGuardrailViolation` 并拒绝写入历史；安全消息示例与允许的正常输入可参考 `go/tests/contract/security_contract_test.go`。
- 认证路径：仅允许 `X-API-Key`；其它 Authorization 方案会被拒绝（参考 `auth_middleware_negative_test.go`），确保 docs 明确这一 FR-004 限制。

## 文档构建
- 在文档仓库运行：`npm install`（如未安装依赖）后执行 `npm run docs:build`；从本仓库可通过 `DOCS_DIR=/absolute/path/to/docs make docs` 触发，日志落在 `specs/001-agno-agents-refactor/artifacts/logs/docs.log`。
- 更新 Basics 章节时需同步上述命令、环境变量表、已知差异与回退/安全行为，并在提交前完成一次构建以避免 CI 阻断。
