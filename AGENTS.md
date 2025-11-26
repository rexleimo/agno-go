# agno-Go 开发指引

自动汇总于所有功能计划。最后更新： 2025-11-22

## 在用技术
- Go 1.25.1 作为唯一运行时；Python 3.11 仅用于离线治具生成
- 标准库 `net/http` + `github.com/go-chi/chi/v5` 路由/中间件
- 自研 REST 客户端覆盖九家模型供应商（Ollama、Gemini、OpenAI、GLM4、OpenRouter、SiliconFlow、Cerebras、ModelScope、Groq）
- 质量工具：golangci-lint、gofumpt、benchstat，统一入口为 Makefile
- 记忆存储接口 `MemoryStore`，默认内存实现，提供 Bolt/Badger 可选持久化

## 项目结构

```text
/Users/rex/cool.cnb/agno-Go/
├── agno/                         # Python 参考实现（只读）
├── specs/001-go-agno-rewrite/    # 当前迭代规格、计划、契约、研究
├── specs/001-vitepress-docs/     # 官方文档站规格、计划、契约与任务（VitePress 多语言）
├── go/                           # 计划新增的 Go 模块根（cmd/internal/pkg/tests）
├── scripts/                      # Go/标准工具脚本
├── docs/                         # VitePress 官方文档工程（en/zh/ja/ko 多语言）
├── Makefile                      # fmt/lint/test/providers-test/coverage/bench/gen-fixtures/release/constitution-check/docs-build/docs-check
├── .github/workflows/ci.yml      # 运行 make fmt/lint/test/providers-test/coverage/bench/constitution-check/docs-check 并上传工件
└── .env.example                  # 待补全的供应商占位
```

## 可用命令
- `make fmt` / `make lint` / `make test` / `make providers-test` / `make coverage` / `make bench` / `make gen-fixtures` / `make release` / `make constitution-check`
- `go run ./go/cmd/agno --config /Users/rex/cool.cnb/agno-Go/config/default.yaml`（启动 AgentOS，本地）
- `./scripts/gen-fixtures.sh`（从 `specs/001-go-agno-rewrite/contracts/fixtures-src` 复制/验证治具到 fixtures；`VERIFY_ONLY=true` 时仅校验）

## 代码风格
- gofumpt 格式化，golangci-lint 默认配置；所有包需有 `_test.go`
- API 与错误语义需与 Python 版保持兼容，禁止任何运行时 Python/cgo/子进程依赖

## 最新变更
- 001-go-agno-rewrite：规划纯 Go AgentOS，定义数据模型、OpenAPI 契约、记忆存储策略与性能目标（20% p95、25% 峰值内存），新增 CI（Go 1.25.1 + make fmt/lint/test/providers-test/coverage/bench/constitution-check）与 gofumpt/golangci-lint 配置

<!-- MANUAL ADDITIONS START -->
- 契约治具：`specs/001-go-agno-rewrite/contracts/fixtures/` 已生成部分基线（Gemini/GLM4/Groq/SiliconFlow），其余 provider 仍为占位（缺 key/鉴权/模型/接口兼容）。未覆盖的项 parity 仅做形状/非空检查。
- 供应商测试：缺 key 时 `make providers-test` 会跳过并记录 `artifacts/coverage/providers.log`；Ollama 需本地 `OLLAMA_ENDPOINT` 可达，否则标记 unreachable。
- 契约/供应商回归推荐命令（复用本地缓存）：`GOCACHE=$PWD/.cache/go-build go test ./tests/contract ./tests/providers`，输出/跳过原因会写入 `specs/001-go-agno-rewrite/artifacts/coverage/providers.log`。
- 替换治具后记得更新 `contracts/deviations.md` 记录与 Python 的差异。
- 基线治具生成：可运行 `go run ./go/scripts/gen_provider_baseline`（读取 `.env` + `config/default.yaml`）生成/更新 fixtures；已生成 Gemini/GLM4/Groq/SiliconFlow 的 chat 基线，以及 Gemini/SiliconFlow 的 embedding；Groq/GLM4 embedding、Cerebras/ModelScope/Ollama 仍因限额/鉴权/接口兼容性保留占位（详见 `contracts/deviations.md`）。Ollama 本地建议 `OLLAMA_ENDPOINT=http://localhost:11434/api`。
- 官方文档站：用户文档统一通过基于 VitePress 的站点 `https://rexai.top/agno-Go/` 提供，其源代码位于仓库根目录的 `docs/`（多语言 VitePress 工程），其内容应与 `specs/001-go-agno-rewrite` 和 `specs/001-vitepress-docs` 中的 quickstart/契约保持一致，并在英文、中文、日文、韩文四种语言之间保持结构与示例的对齐。
<!-- MANUAL ADDITIONS END -->
