# Agno-Go Development Guide | Agno-Go 开发指南

This guide provides essential information for developers contributing to Agno-Go.

本指南为 Agno-Go 贡献者提供开发相关的关键信息。

---

## Development Environment | 开发环境设置

### Requirements | 前置要求

- **Go**: 1.21 or later | 1.21 或更高版本
- **Git**: For version control | 用于版本控制
- **Node.js**: 20+ for documentation | 20+ 用于文档
- **golangci-lint**: (Optional) For code linting | (可选) 代码检查工具
- **goimports**: (Optional) For import formatting | (可选) 导入格式化工具

### Setup | 初始化

```bash
# Clone repository | 克隆仓库
cd agno-Go

# Download dependencies | 下载依赖
go mod download

# Set API keys for testing | 设置 API 密钥用于测试
export OPENAI_API_KEY=sk-...
export ANTHROPIC_API_KEY=sk-ant-...

# (Optional) Install development tools | (可选) 安装开发工具
make install-tools
```

---

## GitHub Pages Documentation | GitHub Pages 文档

### Prerequisites | 前置条件

The repository already has | 仓库已具备:
- ✅ VitePress documentation in `website/` directory | `website/` 目录中的 VitePress 文档
- ✅ GitHub Actions workflow in `.github/workflows/deploy-docs.yml` | `.github/workflows/deploy-docs.yml` 中的工作流
- ✅ Proper permissions configuration | 正确的权限配置
- ✅ `.nojekyll` file to prevent Jekyll processing | `.nojekyll` 文件防止 Jekyll 处理

### Enable GitHub Pages | 启用 GitHub Pages

**IMPORTANT: The following steps must be performed in the GitHub web interface**

**重要:以下步骤必须在 GitHub 网页界面中执行**

1. **Navigate to Repository Settings | 前往仓库设置**
   - Go to your repository on GitHub | 前往 GitHub 上的仓库
   - Click **Settings** tab | 点击 **Settings** 标签

2. **Enable GitHub Pages | 启用 GitHub Pages**
   - In the left sidebar, click **Pages** | 在左侧边栏中,点击 **Pages**
   - Under **Build and deployment**, find **Source** | 在 **构建和部署** 下,找到 **来源**
   - Select **GitHub Actions** from the dropdown | 从下拉菜单中选择 **GitHub Actions**
   - **NOT** "Deploy from a branch" | **不是** "Deploy from a branch"

3. **Save and Verify | 保存并验证**
   - The setting saves automatically | 设置会自动保存
   - You should see: "Your site is ready to be published" | 你应该看到:"Your site is ready to be published"

### Trigger Deployment | 触发部署

**Option 1: Push to main branch | 选项 1: 推送到 main 分支**
```bash
# Make any change to website files | 对 website 文件做任何更改
git add .
git commit -m "docs: update documentation"
git push origin main
```

**Option 2: Manual trigger | 选项 2: 手动触发**
- Go to **Actions** tab on GitHub | 前往 GitHub 上的 **Actions** 标签
- Select **Deploy VitePress Docs to GitHub Pages** workflow | 选择 **Deploy VitePress Docs to GitHub Pages** 工作流
- Click **Run workflow** button | 点击 **Run workflow** 按钮

### Access Deployed Site | 访问部署的网站

After successful deployment, site will be available at:

成功部署后,网站将在以下地址可用:

```
https://<username>.github.io/agno-Go/
```

### Troubleshooting Pages Deployment | Pages 部署故障排查

**Problem: "Get Pages site failed" error | 问题: "Get Pages site failed" 错误**

**Solution | 解决方案:**
- This error occurs when Pages is not enabled in repository settings
- 当仓库设置中未启用 Pages 时会出现此错误
- Follow the "Enable GitHub Pages" steps above
- 按照上述"启用 GitHub Pages"步骤操作

**Problem: 404 errors on deployed site | 问题: 部署网站上出现 404 错误**

**Solution | 解决方案:**
- Check that `base: '/agno-Go/'` is correctly set in `website/.vitepress/config.ts`
- 检查 `website/.vitepress/config.ts` 中 `base: '/agno-Go/'` 是否设置正确
- Ensure `.nojekyll` file exists in `website/public/`
- 确保 `website/public/` 中存在 `.nojekyll` 文件

**Problem: Assets not loading | 问题: 资源加载失败**

**Solution | 解决方案:**
- All asset paths must include the base path `/agno-Go/`
- 所有资源路径必须包含基础路径 `/agno-Go/`
- Use VitePress `withBase()` helper for dynamic paths
- 使用 VitePress 的 `withBase()` 辅助函数处理动态路径

### Local Documentation Preview | 本地文档预览

Before pushing, test the documentation locally:

推送前,在本地测试文档:

```bash
# Install dependencies | 安装依赖
npm install

# Start dev server | 启动开发服务器
npm run docs:dev

# Build and preview | 构建并预览
npm run docs:build
npm run docs:preview
```

The dev server runs at `http://localhost:5173/agno-Go/`

开发服务器运行在 `http://localhost:5173/agno-Go/`

---

## Code Standards | 代码规范

### Code Style | 代码风格

**Function Documentation | 函数文档**
```go
// New creates a new Agent with the given configuration.
// Returns an error if model is not provided or configuration is invalid.
//
// New 创建一个新的 Agent,使用给定的配置。
// 如果未提供 Model 或配置无效,返回错误。
func New(config *Config) (*Agent, error) {
    // ...
}
```

**Error Handling | 错误处理**
```go
if err != nil {
    return nil, fmt.Errorf("failed to create agent: %w", err)
}
```

**Context Usage | Context 使用**
```go
func (a *Agent) Run(ctx context.Context, input string) (*RunOutput, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // implementation
    }
}
```

### Before Committing | 提交前检查

1. **Format code | 格式化代码**: `make fmt`
2. **Run tests | 运行测试**: `make test`
3. **Run linter | 运行检查**: `make lint` (if golangci-lint installed)
4. **Verify coverage | 验证覆盖率**: `make coverage`

---

## Testing Standards | 测试标准

### Coverage Requirements | 覆盖率要求

- **Core packages | 核心包**: >70% test coverage
- **New features | 新特性**: Must include tests
- **Bug fixes | Bug 修复**: Must include regression tests

### Test Structure | 测试结构

**Unit Test Example | 单元测试示例**
```go
func TestAgentRun(t *testing.T) {
    model := &MockModel{
        InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
            return &types.ModelResponse{
                Content: "test response",
            }, nil
        },
    }

    agent, err := New(Config{
        Name:  "test-agent",
        Model: model,
    })
    if err != nil {
        t.Fatalf("Failed to create agent: %v", err)
    }

    output, err := agent.Run(context.Background(), "test input")
    if err != nil {
        t.Fatalf("Run failed: %v", err)
    }

    if output.Content != "test response" {
        t.Errorf("Expected 'test response', got '%s'", output.Content)
    }
}
```

**Benchmark Example | 性能基准测试示例**
```go
func BenchmarkAgentCreation(b *testing.B) {
    model := &MockModel{}

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        _, err := New(Config{
            Name:  "benchmark-agent",
            Model: model,
        })
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

---

## Adding New Components | 添加新组件

### Adding a Model Provider | 添加模型提供商

1. Create directory | 创建目录: `pkg/hno/models/<your_model>/`
2. Implement `models.Model` interface | 实现 `models.Model` 接口:
   ```go
   type Model interface {
       Invoke(ctx context.Context, req *InvokeRequest) (*types.ModelResponse, error)
       InvokeStream(ctx context.Context, req *InvokeRequest) (<-chan types.ResponseChunk, error)
       GetProvider() string
       GetID() string
   }
   ```
3. Add unit tests | 添加单元测试: `<your_model>_test.go`
4. Update documentation | 更新文档

**Example Structure | 示例结构**
```go
type YourModel struct {
    models.BaseModel
    config     Config
    httpClient *http.Client
}

func New(modelID string, config Config) (*YourModel, error) {
    // Initialize
    return &YourModel{
        config:     config,
        httpClient: &http.Client{Timeout: 30 * time.Second},
    }, nil
}

func (m *YourModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
    // Implementation
}
```

### Adding a Tool | 添加工具

1. Create directory | 创建目录: `pkg/hno/tools/<your_tool>/`
2. Embed `toolkit.BaseToolkit` | 嵌入 `toolkit.BaseToolkit`
3. Register functions | 注册函数
4. Add unit tests | 添加单元测试

**Example Structure | 示例结构**
```go
type MyToolkit struct {
    *toolkit.BaseToolkit
}

func New() *MyToolkit {
    t := &MyToolkit{
        BaseToolkit: toolkit.NewBaseToolkit("my_tools"),
    }

    t.RegisterFunction(&toolkit.Function{
        Name:        "my_function",
        Description: "Performs a useful operation | 执行有用的操作",
        Parameters: map[string]toolkit.Parameter{
            "input": {
                Type:        "string",
                Description: "Input parameter | 输入参数",
                Required:    true,
            },
        },
        Handler: t.myHandler,
    })

    return t
}

func (t *MyToolkit) myHandler(args map[string]interface{}) (interface{}, error) {
    input := args["input"].(string)
    // Implementation
    return result, nil
}
```

### Built-in Tool Integrations | 内置工具集成

Agno-Go ships with several production-ready toolkits. They can be imported directly from `pkg/hno/tools/...` or registered via configuration builders.

| Toolkit | Package | Configuration Notes |
| --- | --- | --- |
| Claude Agent Skills | `pkg/hno/tools/claude` | Requires an Anthropic API key (`ANTHROPIC_API_KEY`); optional custom base URL for self-hosted gateways. |
| Tavily Search + Reader | `pkg/hno/tools/tavily` | Requires `TAVILY_API_KEY`; supports quick answers and reader mode with `extract=true`. |
| PPTX Reader | `pkg/hno/tools/file` (`read_pptx`) | No external credentials; parses slide text for ingestion pipelines. |
| Jira Worklogs | `pkg/hno/tools/jira` | Provide Jira base URL and a PAT/Bearer token; adds worklogs via REST v3. |
| Gmail Mark-as-Read | `pkg/hno/tools/gmail` | Requires OAuth access token; removes the `UNREAD` label for a message. |
| ElevenLabs Speech | `pkg/hno/tools/elevenlabs` | Requires `ELEVENLABS_API_KEY`; exposes `generate_speech` with stability/similarity controls. |

Each toolkit includes contract tests (`*_test.go`) demonstrating expected payloads. When wiring them into an agent, pass the configuration struct (e.g. `claude.Config`, `jira.Config`) with the required keys and the shared `toolkit.ToModelToolDefinitions` helper will expose them to LLMs.

---

## Runtime Parity & Configuration | 运行时功能对等配置

### Session Runtime | 会话运行时

- **Shared sessions** – `POST /api/v1/sessions/{id}/reuse` attaches existing sessions to new agents, teams, workflows, or users, matching Python semantics. | **会话复用** – 通过 `POST /api/v1/sessions/{id}/reuse` 将现有会话绑定到新的代理、团队、工作流或用户，语义与 Python 版本一致。
- **Summaries** – `GET`/`POST /api/v1/sessions/{id}/summary` call `session.SummaryManager`; `POST` queues async generation and persists the latest snapshot. | **会话摘要** – 使用 `GET`/`POST /api/v1/sessions/{id}/summary` 调用 `session.SummaryManager`；`POST` 会异步生成并持久化最新摘要。
- **History filters** – `GET /api/v1/sessions/{id}/history?num_messages=N&stream_events=true` limits messages and mirrors the `stream_events` toggle used by Python SSE streams. | **历史过滤** – `GET /api/v1/sessions/{id}/history?num_messages=N&stream_events=true` 限制消息数量，并与 Python SSE 的 `stream_events` 开关保持一致。
- **Run metadata** – API payloads expose `runs[*].cache_hit`, `runs[*].status`, timestamps, and `cancellation_reason` for audit/resume flows. | **运行元数据** – API 返回中包含 `runs[*].cache_hit`、`runs[*].status`、时间戳以及 `cancellation_reason`，用于审计与恢复。

```bash
curl -X POST \
  http://localhost:8080/api/v1/sessions/SESSION_ID/reuse \
  -H 'Content-Type: application/json' \
  -d '{"agent_id":"agent-writer","team_id":"team-research"}'
```

```go
db, _ := sql.Open("postgres", os.Getenv("DATABASE_URL"))
store, _ := postgres.NewStorage(db, postgres.WithSchema("agentos"))

summaryModel, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: os.Getenv("OPENAI_API_KEY")})
summary := session.NewSummaryManager(
    session.WithSummaryModel(summaryModel),
    session.WithSummaryTimeout(45*time.Second),
)

server, _ := agentos.NewServer(&agentos.Config{
    Address:        ":8080",
    SessionStorage: store,
    SummaryManager: summary,
})
```

### Storage Adapters | 存储适配器

| Backend | Package | Notes |
| --- | --- | --- |
| Postgres | `pkg/hno/session/db/postgres` | Batch writer + `jsonb` columns keep sessions, runs, summaries, and cancellation snapshots consistent.<br>批量写入器与 `jsonb` 列确保会话、运行、摘要与取消快照保持一致。 |
| MongoDB | `pkg/hno/session/db/mongo` | `ReplaceOne` with `upsert=true`, 200 ms default timeout, cancellations stored under `cancellations[run_id]`.<br>`ReplaceOne` 使用 `upsert=true`，默认超时 200 ms，取消信息存储在 `cancellations[run_id]` 中。 |
| SQLite | `pkg/hno/session/db/sqlite` | `modernc.org/sqlite` driver, `busy_timeout=200ms`, JSON persisted as text with helpers for marshal/unmarshal.<br>基于 `modernc.org/sqlite`，设置 `busy_timeout=200ms`，JSON 以文本形式存储并辅以编解码工具。 |

All adapters honour context cancellation via the shared `ensureContext` helper, so always propagate `context.Context` from HTTP handlers or background jobs. | 所有适配器通过 `ensureContext` 共享辅助函数响应取消，因此请始终从 HTTP 处理器或后台任务传递 `context.Context`。

### Response Cache & Cancellation | 响应缓存与取消

- Enable agent caching via `agent.Config{EnableCache: true, CacheTTL: 5 * time.Minute}` or provide a custom `cache.Provider` implementation. | 通过 `agent.Config{EnableCache: true, CacheTTL: 5 * time.Minute}` 启用代理缓存，或自定义 `cache.Provider` 实现。
- When a context is cancelled, `agent.Run` persists `RunStatusCancelled` with `cancellation_reason`; downstream stores capture the snapshot for recovery. | 当上下文被取消时，`agent.Run` 会持久化 `RunStatusCancelled` 及 `cancellation_reason`，存储层将捕获快照以便恢复。

### Teams & Workflows | 团队与工作流

- `team.Config.SharedModel` and `InheritModel` let teams default to a shared provider, while `DisableInheritanceFor` and `ModelOverrides` mirror Python’s inheritance matrix. | `team.Config.SharedModel` 与 `InheritModel` 允许团队默认共享模型，`DisableInheritanceFor` 与 `ModelOverrides` 复刻 Python 继承矩阵。
- Workflows accept `workflow.WithResumeFrom(stepID)` and `workflow.WithSessionState(snapshot)` to resume partial runs; `WithMediaPayload` validates image/audio/video inputs before execution. | 工作流支持 `workflow.WithResumeFrom(stepID)` 与 `workflow.WithSessionState(snapshot)` 以恢复部分运行；`WithMediaPayload` 会在执行前验证图像/音频/视频输入。

### Media & Guardrails | 媒体与防护

- Media attachments flow through `media.Normalize` and surface in workflow/session history while preventing empty payloads. | 媒体附件由 `media.Normalize` 处理后写入工作流/会话历史，同时禁止空载荷。
- Stream hooks respect the `stream_events` flag so SSE topics match the Python runtime (`run_start`, `reasoning`, `token`, `tool_call`, `complete`, `error`). | 流式钩子遵循 `stream_events` 标志，SSE 主题与 Python 运行时一致（`run_start`、`reasoning`、`token`、`tool_call`、`complete`、`error`）。

---

## Git Workflow | Git 工作流

### Branch Strategy | 分支策略

```
main (protected | 受保护)
  ↓
feature/your-feature (development | 开发分支)
```

### Commit Message Format | 提交信息格式

```
<type>(<scope>): <subject>

# Examples | 示例
feat(agent): add streaming support
fix(models): fix openai timeout issue
test(agent): add unit tests for memory
docs(readme): update installation guide

# Types | 类型
feat:     New feature | 新功能
fix:      Bug fix | Bug 修复
test:     Tests | 测试
docs:     Documentation | 文档
refactor: Refactoring | 重构
perf:     Performance | 性能优化
chore:    Maintenance | 维护
```

### Pull Request Process | PR 流程

1. **Create feature branch | 创建功能分支**
   ```bash
   git checkout -b feature/my-feature
   git commit -m "feat(scope): add feature"
   git push origin feature/my-feature
   ```

2. **PR Checklist | PR 检查项**
   - [ ] CI passes | CI 通过
   - [ ] Test coverage maintained | 测试覆盖率保持
   - [ ] Code documented | 代码有注释
   - [ ] Documentation updated | 文档已更新

3. **Merge | 合并**
   - Use Squash Merge | 使用 Squash 合并
   - Delete feature branch | 删除功能分支

---

## Common Commands | 常用命令

### Testing | 测试

```bash
# Run all tests | 运行所有测试
make test

# Run specific package | 运行特定包
go test -v ./pkg/hno/agent/...

# Generate coverage report | 生成覆盖率报告
make coverage

# Run benchmarks | 运行性能测试
go test -bench=. -benchmem ./pkg/hno/agent/
```

### Code Quality | 代码质量

```bash
# Format code | 格式化代码
make fmt

# Run linter | 运行检查
make lint

# Run go vet | 运行 go vet
make vet
```

### Building | 构建

```bash
# Build examples | 构建示例
make build

# Run example | 运行示例
./bin/simple_agent
```

---

## Project Structure | 项目结构

```
agno-Go/
├── cmd/
│   └── examples/          # Example programs | 示例程序
├── pkg/
│   ├── agno/
│   │   ├── agent/        # Core Agent | 核心 Agent
│   │   ├── team/         # Multi-agent | 多智能体
│   │   ├── workflow/     # Workflow engine | 工作流引擎
│   │   ├── models/       # LLM providers | LLM 提供商
│   │   ├── tools/        # Toolkits | 工具包
│   │   ├── memory/       # Memory management | 记忆管理
│   │   ├── types/        # Core types | 核心类型
│   │   ├── vectordb/     # Vector databases | 向量数据库
│   │   ├── embeddings/   # Embedding providers | 嵌入提供商
│   │   ├── knowledge/    # Knowledge base | 知识库
│   │   └── session/      # Session management | 会话管理
│   └── agentos/          # HTTP API server | HTTP API 服务器
├── docs/                 # Documentation | 文档
├── scripts/              # Utility scripts | 工具脚本
├── Makefile             # Build automation | 构建自动化
├── go.mod               # Go dependencies | Go 依赖
└── README.md            # Project overview | 项目概览
```

---

## Design Principles | 设计原则

### KISS Principle | KISS 原则

Agno-Go follows **Keep It Simple, Stupid**:

Agno-Go 遵循 **保持简单,愚蠢** 原则:

1. **Quality over Quantity | 质量优于数量**
   - 3 core LLM providers (not 45+) | 3 个核心 LLM 提供商(不是 45+)
   - Essential tools only | 仅核心工具
   - 1 vector DB for validation | 1 个向量数据库用于验证

2. **Interface over Implementation | 接口优于实现**
   - Depend on abstractions | 依赖抽象
   - Easy to mock and test | 易于 mock 和测试

3. **Composition over Inheritance | 组合优于继承**
   - Use struct embedding | 使用结构体嵌入
   - Clear responsibilities | 清晰的职责

4. **Explicit over Implicit | 显式优于隐式**
   - No magic | 没有魔法
   - Clear error handling | 清晰的错误处理

---

## Performance Guidelines | 性能指南

### Optimization Strategies | 优化策略

1. **Minimize Allocations | 最小化内存分配**
   - Pre-allocate slices: `make([]T, 0, capacity)`
   - Use `strings.Builder` for string concatenation
   - Avoid unnecessary copies

2. **Concurrency | 并发**
   - Use goroutines wisely | 明智使用 goroutines
   - Implement worker pools for bounded concurrency | 实现 worker pool 控制并发
   - Use buffered channels | 使用带缓冲的 channel

3. **Memory Management | 内存管理**
   - Consider `sync.Pool` for frequently allocated objects | 考虑使用 `sync.Pool`
   - Profile with `pprof` | 使用 `pprof` 分析
   - Monitor with benchmarks | 使用基准测试监控

### Profiling | 性能分析

```bash
# CPU profiling | CPU 分析
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Memory profiling | 内存分析
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof

# Trace execution | 执行追踪
go test -trace=trace.out
go tool trace trace.out
```

---

## Troubleshooting | 故障排除

### Common Issues | 常见问题

**Issue: Tests fail with timeout | 测试超时失败**
```bash
# Increase timeout | 增加超时时间
go test -timeout 5m ./...
```

**Issue: Import cycles | 导入循环**
```bash
# Check dependencies | 检查依赖
go mod graph | grep your-package
```

**Issue: Race condition detected | 检测到竞态条件**
```bash
# Run with race detector | 使用竞态检测器运行
go test -race ./...
```

---

## Resources | 资源

### Documentation | 文档

- [Architecture Design | 架构设计](ARCHITECTURE.md)
- [API Reference | API 参考](API_REFERENCE.md)
- [Deployment Guide | 部署指南](DEPLOYMENT.md)
- [Performance Benchmarks | 性能基准](PERFORMANCE.md)

### External Resources | 外部资源

- [Go Style Guide](https://google.github.io/styleguide/go/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

---

## Getting Help | 获取帮助

### Communication Channels | 沟通渠道

- **GitHub Issues**: Bug reports and feature requests | Bug 报告和功能请求
- **GitHub Discussions**: Questions and discussions | 问题和讨论
- **Code Review**: Pull request comments | Pull request 评论

### Asking Questions | 提问

When asking for help | 寻求帮助时:

1. Provide context | 提供上下文
2. Include code samples | 包含代码示例
3. Share error messages | 分享错误信息
4. Describe expected vs actual behavior | 描述预期和实际行为

---

## Contributing | 贡献

We welcome contributions! | 我们欢迎贡献!

### How to Contribute | 如何贡献

1. **Fork the repository | Fork 仓库**
2. **Create a feature branch | 创建功能分支**
3. **Make your changes | 做出修改**
4. **Add tests | 添加测试**
5. **Submit a pull request | 提交 pull request**

### Code of Conduct | 行为准则

- Be respectful | 尊重他人
- Be constructive | 建设性反馈
- Be collaborative | 协作精神

---

**Keep it simple, keep it clean, keep it tested.**

**保持简单,保持清晰,保持测试。**
