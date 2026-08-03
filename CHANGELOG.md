# Changelog

All notable changes to Agno-Go will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.9] - 2025-11-14

### ✨ Added
- EvoLink provider for text, images, and video exposed under `pkg/hno/providers/evolink` and `pkg/hno/models/evolink/*`, enabling agents and workflows to call EvoLink via the standard model interfaces.
- New EvoLink Media Agents documentation (`website/examples/evolink-media-agents.md`, `website/zh/examples/evolink-media-agents.md`) with end-to-end examples for text → image → video pipelines.
- Knowledge upload chunking: `POST /api/v1/knowledge/content` now accepts `chunk_size` and `chunk_overlap` for JSON, `text/plain` (query params), and multipart form uploads, propagating these values into stored chunk metadata.
- AgentOS HTTP tips surfaced in docs, covering custom health endpoints, `/openapi.yaml` and `/docs` routes, and `server.Resync()` usage.

### 🛠️ Changed
- Knowledge API handlers persist `chunk_size`, `chunk_overlap`, and `chunker_type` for each stored chunk, mirroring the Python AgentOS responses and enabling downstream pipelines to inspect segmentation strategy.
- AgentOS documentation is now the canonical source for configuring health probes and documentation routes for the Go server, aligning README and VitePress content.

### 🧪 Tests
- Added focused tests for EvoLink image and video models to validate configuration boundaries and task polling behavior.
- Extended knowledge API tests to cover chunking parameters, metadata propagation, and compatibility with existing search/config endpoints.

### ✅ Compatibility
- Additive release; public HTTP and Go APIs remain backward compatible.
- Knowledge chunking parameters are optional and default to previous behavior when omitted.

## [1.2.8] - 2025-11-10

### ✨ Added
- Run Context 贯穿执行：向 hooks、tools、遥测传递上下文，并在流式事件中提供 `run_context_id` 便于关联。
- 会话状态扩展：持久化 `AGUI` 子态，`GET /sessions/{id}` 返回包含 UI 状态的 `session_state`。
- 向量索引能力：
  - 可插拔 VectorDB 提供方（保持 Chroma 为默认示例，Redis 为可选依赖）。
  - VectorDB 迁移 CLI（`migrate up/down`）支持集合与索引的幂等创建/回滚。
- Embeddings：新增 VLLM（本地/远端）提供方，遵循通用 `EmbeddingFunction` 接口。
- MCPTools：支持可选参数 `tool_name_prefix`，为注册工具名加前缀。

### 🛠️ Changed
- Redis 从向量数据库默认依赖中剥离，未配置时零影响；启用时按配置注册。
- 团队模型继承仅下沉主模型配置；辅助参数需在 Agent 端显式开启。

### 🐛 Fixed
- 修复模型响应未正确绑定到步骤导致的“未绑定/零值”问题。
- 修复团队场景下基于 OS Schema 的工具判定，避免成员工具丢失。
- 修复异步存储（知识库）复合过滤与超时上下文配合的边界问题（不泄露 goroutine）。
- 强化工具包导入时的错误信息，缺失模块返回结构化提示而非 panic。
- AgentOS 错误处理路径更一致，便于契约测试断言。

### 🧪 Tests
- 新增覆盖：Run Context 传播、AGUI 状态持久化、团队主模型继承、MCP 前缀、VLLM 嵌入。
- 可选依赖（Redis）用例按环境开关执行，默认跳过。

### ✅ Compatibility
- 增量更新；默认关闭的可选能力不影响现有用户；外部 API 形态保持不变。

## [1.2.7] - 2025-11-03

### ✨ Added
- Go session service matching the Python AgentOS `/sessions` API, including Chi router, Postgres-backed store, and health endpoints.
- Deployment assets for the session runtime: dedicated Dockerfile, Compose stack, Helm chart, and curl-based verification script.
- Documentation for quick start and production deployment of the Go session service.

### 🧪 Tests
- Contract suite and Postgres store coverage ensuring session parity with existing Python fixtures.

### ✅ Compatibility
- Additive release; existing APIs remain backward compatible with the Go session runtime as an opt-in component.

---

## [1.2.6] - 2025-10-31

### ✨ Added
- Session runtime parity: session reuse endpoint, history pagination filters (`num_messages`, `stream_events`), and summary manager supporting sync/async generation.
- Response caching for agents and teams via in-memory LRU store, exposing cache-hit metadata in session runs.
- Media attachments across agents, teams, and workflows, including validation helpers and `WithMediaPayload` run option.
- Pluggable session storage drivers: MongoDB and SQLite implementations alongside Postgres with shared JSON contracts.
- Tooling expansions: Tavily Reader/Search, Claude Agent Skills, Gmail mark-as-read, Jira worklogs, ElevenLabs speech synthesis, and enhanced file toolkit.

### 🛠️ Changed
- AgentOS session handlers now expose summary endpoints, reuse semantics, run metadata (status timestamps, cancellation reasons, cache hits), and history filters.
- Workflow engine supports resumable checkpoints, cancellation persistence, and media-only payload execution paths.
- MCP client caches capability manifests and forwards media attachments in tool calls.

### 🧪 Tests
- Added dedicated coverage for cache layer, summary manager, storage drivers, workflow media flows, and new toolkits.

### ✅ Compatibility
- Additive release; existing APIs remain backward compatible with new features opt-in.

---

## [1.2.5] - 2025-10-20

### ✨ Added
- Model providers: Cohere, Together, OpenRouter, LM Studio, Vercel, Portkey, InternLM, SambaNova (Invoke/Stream + function calling)
- Core modules:
  - Evaluation system (scenario runner, per-run metrics, aggregated summary, multi-model comparison)
  - Media processing: image metadata (DecodeConfig), audio/video probe占位
  - Debug helpers: request/response compact dump
  - Cloud: NoopDeployer interface for simple deployments
- Integrations registry: register/list/health-check for third‑party services

### 🛠️ Changed
- Airflow toolkit mock schema aligned with Airflow REST API v2 (Context7): `total_entries`, `dag_run_id`, `logical_date`
- Website hero image uses `/logo.png` (fix broken asset)
- README “Multi-provider models” list updated

### 🧪 Tests
- Focused unit tests for new providers and modules (cohere, together, openrouter, lmstudio, vercel, portkey, internlm, sambanova, eval/media/debug/integrations/utils)

### ✅ Compatibility
- Additive features; no breaking changes

---

## [1.2.2] - 2025-10-18

### ✨ Added

#### Reasoning Model Support
- **Enhanced Reasoning Capabilities** - Advanced reasoning support for modern LLM models
  - Automatic detection for Gemini, Anthropic Claude, and VertexAI Claude
  - Structured reasoning output with step-by-step analysis
  - **cmd/examples/reasoning/** - Example program demonstrating reasoning capabilities

#### Batch Operations for PostgreSQL
- **High-Performance Batch Upsert** - Optimized bulk operations
  - 10x faster than individual INSERT/UPDATE operations
  - Transaction-safe with conflict resolution
  - **cmd/examples/batch_upsert/** - Performance comparison example

#### SurrealDB Vector Database Support
- **Modern Vector Database Integration** - Full SurrealDB support
  - Vector similarity search and document embedding storage
  - Real-time query capabilities
  - **cmd/examples/surreal_demo/** - Vector operations example

#### CI/CD Pipeline
- **GitHub Actions CI Workflow** - Automated testing and quality assurance
  - Go module validation and unit tests with race detection
  - Code coverage reporting and security scanning

#### Enhanced Knowledge API
- **Advanced Content Processing** - Improved knowledge ingestion
  - Multi-format content extraction (JSON, Form, Text)
  - Structured data validation and metadata extraction

### 🧪 Testing & Quality
- **Enhanced Test Coverage** - 85% reasoning, 92% batch, 88% SurrealDB
- **Race Condition Detection** - All new code validated with `-race` flag
- **Performance Benchmarks** - Added comprehensive performance tests

### 📊 Performance
- **Batch Operations** - 10x performance improvement for bulk data
- **Reasoning Detection** - Minimal overhead (<1ms)
- **No Regression** - All existing benchmarks maintained

### ✅ Backward Compatibility
- Additive changes only; no breaking changes
- All existing APIs remain unchanged
- Enhanced functionality automatically available for supported models

## [1.2.1] - 2025-10-15

### ✨ Added
- SSE event filtering on streaming endpoints (A2A)
  - `POST /api/v1/agents/:id/run/stream?types=token,complete`
- Content extraction middleware for AgentOS (JSON/Form → context)
- Google Sheets toolkit (service account)
- Minimal knowledge ingestion endpoint (`POST /api/v1/knowledge/content`)

### 🧭 Documentation Reorganization
- Adopted clear separation of docs:
  - `website/` → Implemented, user-facing documentation (VitePress site)
  - `docs/` → Design drafts, WIP, migration plans, developer/internal docs
- Added `docs/README.md` to state policy and entry points
- Added `CONTRIBUTING.md` for contributors (development, testing, docs website)

### 🔗 Link Updates
- README, CLAUDE, CHANGELOG, and release notes now point to canonical pages under `website/advanced/*` and `website/guide/*`
- Removed outdated links to duplicated files under `docs/`

### 🧹 Removed (duplicated implemented docs from docs/)
- Deleted `docs/{API_REFERENCE.md, ARCHITECTURE.md, DEPLOYMENT.md, MULTI_TENANT.md, PERFORMANCE.md, QUICK_START.md, SESSION_STATE.md, WORKFLOW_HISTORY.md, A2A_INTERFACE.md, CHANGELOG.md}`

### 🌐 Website
- Updated API docs to include Knowledge API and configuration on AgentOS page
- Updated website Release Notes with v1.2.1 summary

### ✅ Backward Compatibility
- Additive changes only; no breaking changes

## [1.2.0] - 2025-10-12

### ✨ Added

#### Workflow Session Storage (S005)
- **In-Memory Session Management** - Complete workflow session lifecycle management
  - **pkg/hno/workflow/memory_storage.go** - MemoryStorage implementation (393 lines)
    - Session creation, retrieval, updating, deletion
    - Concurrent-safe with sync.RWMutex
    - Configurable max sessions limit
    - Automatic session pruning
  - **pkg/hno/workflow/session.go** - WorkflowSession structure (300 lines)
    - Session metadata and run history
    - History retrieval with flexible count
    - Statistics tracking (total/completed/success/failed runs)
  - **pkg/hno/workflow/run.go** - WorkflowRun structure (158 lines)
    - Individual run execution tracking
    - Input/output/error recording
    - Timestamp and status management
  - **pkg/hno/workflow/storage.go** - WorkflowStorage interface (141 lines)
    - Abstract storage interface for extensibility
    - Support for custom storage implementations (Redis, PostgreSQL, etc.)

#### Workflow History Injection (S008)
- **Agent Temporary Instructions Support** - Enable history context injection without modifying agent's original configuration
  - **pkg/hno/agent/agent.go** - Enhanced with temporary instructions mechanism
    - `tempInstructions string` - Temporary override for instructions (single execution only)
    - `instructionsMu sync.RWMutex` - Thread-safe concurrent access protection
    - `GetInstructions()` - Retrieves current instructions (temporary takes precedence)
    - `SetInstructions()` - Permanently sets agent instructions
    - `SetTempInstructions()` - Temporarily sets instructions (cleared after Run)
    - `ClearTempInstructions()` - Explicitly clears temporary instructions
    - `updateSystemMessage()` - Updates system message with current instructions
  - **Auto-cleanup mechanism**: `defer a.ClearTempInstructions()` in Run() ensures zero memory leak
  - **Concurrency safety**: RWMutex allows concurrent reads, exclusive writes
  - **Backward compatible**: Empty tempInstructions behaves identically to original implementation

- **History Injection Utilities** - Flexible history formatting and injection helpers
  - **pkg/hno/workflow/history_injection.go** - History injection helper functions (151 lines)
    - `InjectHistoryToAgent()` - Injects formatted history into agent's temporary instructions
    - `buildEnhancedInstructions()` - Combines original instructions with history context
    - `RestoreAgentInstructions()` - Explicitly restores original instructions (optional, auto-cleared)
    - `FormatHistoryForAgent()` - Formats history entries with customizable options
    - `HistoryFormatOptions` - Flexible formatting configuration:
      - Header/Footer tags
      - Include/exclude input/output
      - Optional timestamps
      - Customizable labels
    - `DefaultHistoryFormatOptions()` - Sensible defaults with XML-style tags

- **Step Integration** - Seamless workflow history injection
  - **pkg/hno/workflow/step.go** - Updated Step execution with history support
    - Automatically injects history when `shouldAddHistory()` returns true
    - Retrieves formatted history from `ExecutionContext.GetHistoryContext()`
    - No changes required to existing workflow code

### 🧪 Testing

- **Comprehensive Test Coverage**:
  - **agent_instructions_test.go** - 308 lines, 8 test cases
    - `TestAgent_TempInstructions` - Basic get/set/clear functionality
    - `TestAgent_SetInstructions` - Permanent instruction changes
    - `TestAgent_TempInstructionsPriority` - Temporary overrides permanent
    - `TestAgent_ConcurrentInstructionsAccess` - 100 iterations, 300 goroutines
    - `TestAgent_Run_AutoClearsTempInstructions` - Verify defer cleanup
    - `TestAgent_Run_WithTempInstructionsError` - Cleanup on error
    - `TestAgent_UpdateSystemMessage` - Table-driven tests

  - **history_injection_test.go** - 380 lines, 12 test cases
    - Nil safety, empty history handling
    - Default and custom format options
    - Multiple runs formatting
    - Input/output inclusion control

- **Test Results**:
  - All tests passing with `-race` detector ✅
  - Agent coverage: 75.6% (100% on instruction methods)
  - Workflow coverage: 88.4% (100% on history injection)
  - Zero race conditions detected

### 📊 Performance

- **Performance Targets Met/Exceeded**:
  - `GetInstructions()`: <50ns (RLock optimized)
  - `SetTempInstructions()`: <100ns (Lock optimized)
  - `InjectHistoryToAgent()`: ~200-300ns (vs <500ns target, 2x better)
  - Total injection overhead: <1ms (vs <1ms target)
  - Memory overhead: ~40 bytes per agent (negligible)

- **No Performance Regression**:
  - Agent instantiation: Still ~180ns/op
  - Memory footprint: Still ~1.2KB per agent
  - Existing benchmarks unchanged

### 🔧 Technical Highlights

- **Zero Memory Leak** - defer-based automatic cleanup ensures temp instructions always cleared
- **Concurrency Safe** - sync.RWMutex enables high-performance concurrent access
- **API Design** - Clean, intuitive methods following Go best practices
- **Backward Compatible** - No breaking changes to existing Agent API
- **Bilingual Documentation** - All code comments in English/中文

### 📝 Files Added/Modified

**New Files:**
- `pkg/hno/workflow/history_injection.go` - History injection utilities (151 lines)
- `pkg/hno/agent/agent_instructions_test.go` - Instruction tests (308 lines)
- `pkg/hno/workflow/history_injection_test.go` - Injection tests (380 lines)

**Modified Files:**
- `pkg/hno/agent/agent.go` - Added temporary instructions support
- `pkg/hno/workflow/step.go` - Integrated history injection
- `docs/task/S008-agent-history-injection.md` - Marked as Done

### ✅ Acceptance Criteria

All S008 acceptance criteria met or exceeded:
- ✅ Agent supports temporary instructions
- ✅ Agent.Run auto-clears temporary instructions
- ✅ Concurrent access to instructions is thread-safe
- ✅ History injection doesn't affect Agent's original configuration
- ✅ Flexible history formatting options provided
- ✅ Test coverage >85% (100% on new methods)
- ✅ All tests passing
- ✅ Performance: Injection overhead <1ms (achieved <1μs)

### 🚀 Usage Example

```go
import (
    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/workflow"
)

// Create agent with original instructions
agent, _ := agent.New(agent.Config{
    Model:        model,
    Instructions: "You are a helpful assistant",
})

// Format workflow history
history := []workflow.HistoryEntry{
    {Input: "hello", Output: "hi there"},
    {Input: "how are you", Output: "I'm good"},
}
historyContext := workflow.FormatHistoryForAgent(history, nil)

// Inject history (temporarily enhances instructions)
workflow.InjectHistoryToAgent(agent, historyContext)

// Run agent (temp instructions auto-cleared after execution)
output, err := agent.Run(ctx, "new question")

// Agent's original instructions remain unchanged
fmt.Println(agent.Instructions) // "You are a helpful assistant"
```

## [1.1.1] - 2025-10-08

### ✨ Added

#### Groq Model Integration
- **Groq Ultra-Fast Inference Support** - Industry-leading LLM inference speed
  - **pkg/hno/models/groq/** - Complete Groq model implementation (287 lines)
    - `groq.go` - Main implementation using OpenAI-compatible API
    - `types.go` - Model constants and metadata (7+ models)
    - `groq_test.go` - Comprehensive unit tests (580+ lines)
    - `README.md` - Detailed documentation and examples
  - **Supported Models:**
    - LLaMA 3.1 8B Instant (fastest, recommended)
    - LLaMA 3.1 70B Versatile (most capable)
    - LLaMA 3.3 70B Versatile (latest)
    - Mixtral 8x7B (Mixture of Experts)
    - Gemma 2 9B (compact but powerful)
    - Whisper Large V3 (speech recognition)
    - LLaMA Guard 3 8B (content moderation)
  - **Performance Benefits:**
    - 10x faster inference vs traditional cloud LLM providers
    - Ultra-low first-token latency
    - High concurrent request throughput
  - **Features:**
    - OpenAI-compatible API (reuses go-openai SDK)
    - Full function calling support
    - Streaming and non-streaming modes
    - Configurable timeout, temperature, max tokens
  - **cmd/examples/groq_agent/** - Example program with 4 scenarios
  - **Test Coverage:** 52.4% ✅
  - **Documentation:** Updated CLAUDE.md with Groq configuration

### 📝 Documentation
- Added Groq to supported models list in CLAUDE.md
- Added `GROQ_API_KEY` environment variable documentation
- Updated example programs list with groq_agent
- Added Groq to test coverage table

## [1.1.0] - 2025-10-08

### ✨ Added

#### A2A (Agent-to-Agent) Interface
- **A2A Protocol Support** - Standardized agent-to-agent communication based on JSON-RPC 2.0
  - **pkg/agentos/a2a/** - Complete A2A interface implementation (1001 lines)
    - `types.go` - JSON-RPC 2.0 type definitions (154 lines)
    - `validator.go` - Request validation logic (108 lines)
    - `mapper.go` - A2A ↔ RunInput/Output conversion (317 lines)
    - `a2a.go` - Main interface and entity management (148 lines)
    - `handlers.go` - HTTP handlers for send/stream endpoints (274 lines)
  - REST API endpoints:
    - `POST /a2a/message/send` - Non-streaming message send
    - `POST /a2a/message/stream` - Streaming via Server-Sent Events (SSE)
  - Multi-media support: Text, images (URI/bytes), files, JSON data
  - Compatible with Python Agno A2A implementation
  - **cmd/examples/a2a_server/** - Example A2A server
  - **pkg/agentos/a2a/README.md** - Complete bilingual documentation (English/中文)

#### Workflow Session State Management
- **Thread-Safe Session State** - Cross-step session management with race condition fix
  - **pkg/hno/workflow/session_state.go** - SessionState implementation (192 lines)
    - Thread-safe with sync.RWMutex
    - Deep copy via JSON serialization for parallel branch isolation
    - Smart merging: only applies actual changes (last-write-wins)
  - **ExecutionContext Enhancement** - Added session support fields:
    - `SessionState *SessionState` - Cross-step persistent state
    - `SessionID string` - Unique session identifier
    - `UserID string` - Multi-tenant user identifier
  - **Parallel Execution Fix** - Solved Python Agno v2.1.2 race condition:
    - Clone SessionState for each parallel branch
    - Merge modified states after parallel execution
    - Prevents data races in concurrent workflow steps
  - **pkg/hno/workflow/SESSION_STATE.md** - Comprehensive documentation
  - **Test Coverage:** 79.4% with race detector validation ✅

#### Multi-Tenant Memory Support
- **User-Isolated Memory Storage** - Multi-tenant conversation history
  - Enhanced Memory interface with optional `userID` parameter:
    - `Add(message, userID...)` - Add message for specific user
    - `GetMessages(userID...)` - Get user-specific messages
    - `Clear(userID...)` - Clear user messages
    - `Size(userID...)` - Get user message count
  - `InMemory` implementation:
    - Per-user message storage: `map[string][]*types.Message`
    - Independent maxSize limit per user
    - Backward compatible: empty userID defaults to "default" user
  - Agent integration:
    - Added `UserID string` field to Agent and Config
    - All memory operations pass agent's UserID
  - New `ClearAll()` method to clear all users
  - **Tests:** Multi-tenant isolation, backward compatibility, race detection ✅

#### Model Timeout Configuration
- **Configurable Request Timeout** - Fine-grained timeout control for LLM calls
  - **OpenAI Model** (`pkg/hno/models/openai/openai.go`):
    - Added `Timeout time.Duration` to Config
    - Default: 60 seconds
    - Applied to underlying HTTP client
  - **Anthropic Claude** (`pkg/hno/models/anthropic/anthropic.go`):
    - Added `Timeout time.Duration` to Config
    - Default: 60 seconds
    - Applied to HTTP client
  - Usage example:
    ```go
    claude := anthropic.New("claude-3-opus", anthropic.Config{
        APIKey:  apiKey,
        Timeout: 30 * time.Second, // Custom timeout
    })
    ```

### 🐛 Fixed

- **Workflow Race Condition** - Fixed parallel step execution data race
  - Python Agno v2.1.2 had shared `session_state` dict causing overwrites
  - Go implementation uses independent SessionState clones per branch
  - Smart merge strategy prevents data loss in concurrent execution

### 🧪 Testing

- **New Test Suites:**
  - `session_state_test.go` - 543 lines of session state tests
  - `memory_test.go` - Multi-tenant memory tests (4 new test cases)
  - `agent_test.go` - Multi-tenant agent test (TestAgent_MultiTenant)
  - `openai_test.go` - Timeout configuration test
  - `anthropic_test.go` - Timeout configuration test

- **Test Results:**
  - All tests passing with `-race` detector ✅
  - Workflow coverage: 79.4%
  - Memory coverage: maintained at 93.1%
  - Agent coverage: maintained at 74.7%

### 📊 Performance

- **No Performance Regression** - All benchmarks remain consistent:
  - Agent instantiation: ~180ns/op
  - Memory footprint: ~1.2KB per agent
  - Thread-safe concurrent operations

### 🔧 Technical Highlights

- **Python Agno v2.1.2 Compatibility** - Migrated features from commits:
  - `7e487eb` → `bf3286bb` (23 commits, 5 major features)
  - A2A utils implementation
  - Session state race condition fix
  - Multi-tenant memory support
  - Model timeout parameters

- **Bilingual Documentation** - All new features documented in English/中文:
  - Inline code comments
  - README files
  - API documentation

### 📝 Files Added/Modified

**New Files:**
- `pkg/agentos/a2a/*.go` - A2A interface (5 files, 1001 lines)
- `pkg/hno/workflow/session_state.go` - Session state (192 lines)
- `pkg/hno/workflow/session_state_test.go` - Tests (543 lines)
- `pkg/agentos/a2a/README.md` - A2A documentation
- `pkg/hno/workflow/SESSION_STATE.md` - Session state guide
- `cmd/examples/a2a_server/main.go` - A2A example server

**Modified Files:**
- `pkg/hno/memory/memory.go` - Multi-tenant support
- `pkg/hno/memory/memory_test.go` - New multi-tenant tests
- `pkg/hno/agent/agent.go` - UserID support
- `pkg/hno/agent/agent_test.go` - Multi-tenant test
- `pkg/hno/workflow/workflow.go` - SessionState fields
- `pkg/hno/workflow/parallel.go` - Race condition fix
- `pkg/hno/models/openai/openai.go` - Timeout support
- `pkg/hno/models/anthropic/anthropic.go` - Timeout support

### ✅ Migration Status

Completed migration from Python Agno v2.1.2:
- ✅ A2A interface implementation
- ✅ Workflow session state management (race condition fix)
- ✅ Multi-tenant memory support (userID)
- ✅ Model timeout parameters (OpenAI, Anthropic)

### 🚀 Upgrade Guide

**Multi-Tenant Memory:**
```go
// Old (single-tenant)
agent := agent.New(agent.Config{
    Memory: memory.NewInMemory(100),
})

// New (multi-tenant)
agent := agent.New(agent.Config{
    UserID: "user-123",  // Add UserID
    Memory: memory.NewInMemory(100),
})
```

**Workflow Session State:**
```go
// Create context with session info
ctx := workflow.NewExecutionContextWithSession(
    "input",
    "session-id",
    "user-id",
)

// Access session state
ctx.SetSessionState("key", "value")
value, _ := ctx.GetSessionState("key")
```

**A2A Interface:**
```go
// Create A2A interface
a2a := a2a.New(a2a.Config{
    Agents: []a2a.Entity{myAgent},
    Prefix: "/a2a",
})

// Register routes (Gin)
router := gin.Default()
a2a.RegisterRoutes(router)
```

### 📖 Documentation

- [A2A README](pkg/agentos/a2a/README.md) - Complete A2A protocol guide
- [Session State Guide](pkg/hno/workflow/SESSION_STATE.md) - Workflow session management
- [CHANGELOG.md](CHANGELOG.md) - This file

## [1.0.3] - 2025-10-06

### 🧪 Improved

#### Testing & Quality
- **Enhanced JSON Serialization Tests** - Achieved 100% test coverage for utils/serialize package
  - Added error handling tests for unserializable types (channels, functions)
  - Added panic behavior tests for MustToJSONString
  - Added edge case tests (nil pointers, empty collections)
  - Test coverage: 92.3% → 100% ✅

#### Performance Benchmarks
- **Optimized Performance Tests** - Aligned with Python Agno performance testing patterns
  - Simplified agent instantiation benchmark (removed unnecessary variable)
  - Cleaned up tool registration patterns
  - Renamed test for consistency: "Tool Instantiation Performance" → "Agent Instantiation"

#### Documentation
- **Comprehensive Package Documentation** - Added bilingual (English/中文) documentation
  - Package-level overview with usage examples
  - Detailed function documentation with examples
  - Performance metrics included in package docs
  - All public APIs now fully documented

### 📊 Performance

Current benchmark results on Apple M3:
- **ToJSON**: ~600ns/op, 760B/op, 15 allocs/op
- **ConvertValue**: ~180ns/op, 392B/op, 5 allocs/op
- **Agent Creation**: ~180ns/op (16x faster than Python)

### 🔧 Technical Highlights

- **100% Test Coverage** - utils/serialize package now has complete test coverage
- **Better Error Handling** - Comprehensive tests for edge cases and error conditions
- **Production Ready** - Serialization utilities validated for WebSocket and API usage
- **Python Compatibility** - Prevents the JSON serialization bug found in Python Agno (commit aea0fc129)

### 📝 Files Changed

- `pkg/hno/utils/serialize.go` - Enhanced documentation with examples and performance notes
- `pkg/hno/utils/serialize_test.go` - Added 3 new test cases for error handling
- `pkg/hno/agent/agent_bench_test.go` - Simplified benchmark following Python patterns

### ✅ Migration Status

Completed migration items from Python Agno:
- ✅ JSON serialization bug fix (aea0fc129) - Already prevented in Go implementation
- ✅ Performance test optimization (e639f4996) - Applied to Go benchmarks
- 🔄 Custom route prefix (06baed104) - Deferred to Week 7 (AgentOS expansion)
- 🔄 HN tools update (24c3ee688) - Documentation only, no action needed

## [1.0.2] - 2025-10-05

### ✨ Added

#### New LLM Provider
- **GLM (智谱AI)** - Full integration with Zhipu AI's GLM models
  - Support for GLM-4, GLM-4V (vision), GLM-3-Turbo
  - Custom JWT authentication (HMAC-SHA256)
  - Synchronous API calls (`Invoke`)
  - Streaming responses (`InvokeStream`)
  - Tool/Function calling support
  - Test coverage: 57.2%

#### Implementation Details
- **pkg/hno/models/glm/glm.go** - Main model implementation (410 lines)
- **pkg/hno/models/glm/auth.go** - JWT authentication logic (59 lines)
- **pkg/hno/models/glm/types.go** - GLM API type definitions (105 lines)
- **pkg/hno/models/glm/glm_test.go** - Comprehensive unit tests (320 lines)
- **pkg/hno/models/glm/README.md** - Complete usage documentation

#### Examples & Documentation
- **cmd/examples/glm_agent/** - GLM agent example with calculator tools
  - Chinese language support demonstration
  - Multi-step calculation examples
  - Tool calling integration
- Updated README.md with GLM provider information
- Updated CLAUDE.md with GLM configuration and usage
- Added bilingual comments (English/中文) throughout codebase

### 🔧 Technical Highlights

- **Custom JWT Authentication** - Implemented GLM-specific JWT token generation
  - 7-day token expiration
  - Secure HMAC-SHA256 signing
  - Automatic token regeneration per request

- **OpenAI-Compatible Format** - API structure similar to OpenAI for easy integration
  - Request/response format alignment
  - Tool calling compatibility
  - Streaming support via Server-Sent Events (SSE)

- **Type Safety** - Full Go type system integration
  - Strongly-typed request/response structures
  - Error handling with custom error types
  - Context support for cancellation

### 📊 Test Results

- ✅ All 7 GLM tests passing
- ✅ 57.2% code coverage
- ✅ Race detector: PASS
- ✅ Build verification: SUCCESS

### 🌍 Environment Variables

New environment variable for GLM:
```bash
export ZHIPUAI_API_KEY=your-key-id.your-key-secret
```

### 📦 Dependencies

Added:
- `github.com/golang-jwt/jwt/v5 v5.3.0` - For JWT authentication

### 🎯 Supported Models

Total LLM providers increased from 6 to 7:
- OpenAI (GPT-4, GPT-3.5, GPT-4 Turbo)
- Anthropic (Claude 3.5 Sonnet, Claude 3 Opus/Sonnet/Haiku)
- **GLM (智谱AI: GLM-4, GLM-4V, GLM-3-Turbo)** ⭐ NEW
- Ollama (Local models)
- DeepSeek (DeepSeek-V2, DeepSeek-Coder)
- Google Gemini (Gemini Pro, Flash)
- ModelScope (Qwen, Yi models)

### 📝 Documentation Updates

- README.md - Added GLM to supported models list with example code
- CLAUDE.md - Added GLM environment variables and configuration
- Created pkg/hno/models/glm/README.md with comprehensive usage guide
- All code comments are bilingual (English/中文)

## [1.0.0] - 2025-10-02

### 🎉 Initial Release

Agno-Go v1.0 is a high-performance Go implementation of the Agno multi-agent framework, designed for building production-ready AI agent systems.

### ✨ Features

#### Core Agent System
- **Agent** - Single autonomous agent with tool support
  - LLM model integration
  - Tool/function calling
  - Conversation memory
  - System instructions
  - Max loop protection
  - Coverage: 74.7%

- **Team** - Multi-agent collaboration with 4 coordination modes:
  - `Sequential` - Agents work one after another
  - `Parallel` - All agents work simultaneously
  - `LeaderFollower` - Leader delegates tasks to followers
  - `Consensus` - Agents discuss until reaching agreement
  - Coverage: 92.3%

- **Workflow** - Step-based orchestration with 5 primitives:
  - `Step` - Basic workflow step (agent or function)
  - `Condition` - Conditional branching
  - `Loop` - Iterative loops
  - `Parallel` - Parallel execution
  - `Router` - Dynamic routing
  - Coverage: 80.4%

#### LLM Providers
- **OpenAI** - GPT-4, GPT-3.5, GPT-4 Turbo
- **Anthropic** - Claude 3.5 Sonnet, Claude 3 Opus/Sonnet/Haiku
- **Ollama** - Local model support (llama3, mistral, etc.)

#### Tools
- **Calculator** - Basic math operations
- **HTTP** - GET/POST requests
- **File** - File operations with safety controls

#### Storage & Memory
- **Memory** - In-memory conversation storage with auto-truncation
  - Configurable message limits
  - Thread-safe operations
  - Coverage: 93.1%

- **Session** - Session management for multi-turn conversations
  - In-memory storage (default)
  - PostgreSQL support (via schema)
  - Redis caching support

#### Vector Database
- **ChromaDB** - Vector storage for RAG applications
  - Document embedding
  - Semantic search
  - Collection management

#### AgentOS - Production Server
- **RESTful API** - Full-featured HTTP server
  - Session CRUD operations
  - Agent registration and execution
  - Health check endpoint
  - Structured logging with slog
  - CORS support
  - Request timeout handling
  - Coverage: 65.0%

- **Agent Registry** - Thread-safe agent management
  - Dynamic agent registration
  - Concurrent access support
  - Agent lifecycle management

#### Developer Experience
- **Types** - Comprehensive type system
  - Message types (System, User, Assistant, Tool)
  - Error types with codes
  - Model request/response types
  - 100% test coverage ⭐

- **Documentation**
  - Complete API documentation (OpenAPI 3.0)
  - Deployment guide (Docker, K8s, native)
  - Architecture documentation
  - Performance benchmarks
  - Code examples

#### Deployment
- **Docker** - Production-ready Dockerfile
  - Multi-stage build (~15MB final image)
  - Non-root user
  - Health checks
  - Security best practices

- **Docker Compose** - Full stack deployment
  - AgentOS server
  - PostgreSQL database
  - Redis cache
  - ChromaDB (optional)
  - Ollama (optional)

- **Kubernetes** - K8s manifests included
  - Deployment, Service, ConfigMap, Secret
  - Health probes
  - Resource limits
  - Horizontal Pod Autoscaling ready

### 📊 Performance

- **Agent Creation:** ~180ns/op (16x faster than Python)
- **Memory Footprint:** ~1.2KB per agent
- **Test Coverage:** 80.8% average across core packages
- **Concurrent Operations:** Fully thread-safe with RWMutex

### 🧪 Testing

- **85+ test cases** across all core packages
- **100% pass rate** ✅
- All packages exceed 70% coverage target
- Comprehensive integration tests
- Concurrent access tests
- Performance benchmarks

### 📚 Examples

- `simple_agent` - Basic agent with calculator
- `claude_agent` - Anthropic Claude integration
- `ollama_agent` - Local model support
- `team_demo` - Multi-agent collaboration
- `workflow_demo` - Workflow orchestration
- `rag_demo` - RAG with ChromaDB

### 🔧 Technical Details

**Dependencies:**
- Go 1.21+
- Gin web framework
- PostgreSQL 15+ (optional)
- Redis 7+ (optional)
- ChromaDB (optional)

**Project Structure:**
```
agno-Go/
├── pkg/hno/          # Core framework
│   ├── agent/         # Agent implementation
│   ├── team/          # Team coordination
│   ├── workflow/      # Workflow engine
│   ├── models/        # LLM providers
│   ├── tools/         # Tool integrations
│   ├── memory/        # Conversation memory
│   └── types/         # Core types
├── pkg/agentos/       # Production server
│   ├── server.go      # HTTP server
│   ├── registry.go    # Agent registry
│   └── openapi.yaml   # API specification
├── cmd/examples/      # Example programs
└── docs/              # Documentation
```

### 🎯 Design Philosophy

Agno-Go follows the **KISS principle** (Keep It Simple, Stupid):
- Focus on quality over quantity
- Clear, maintainable code
- Comprehensive testing
- Production-ready from day one

### 🔒 Security

- Non-root Docker container
- Secret management best practices
- Input validation
- Error handling
- Rate limiting support
- HTTPS/TLS ready

### 📖 Documentation

- [README.md](README.md) - Getting started
- [website/advanced/architecture.md](website/advanced/architecture.md) - Architecture overview
- [website/advanced/deployment.md](website/advanced/deployment.md) - Deployment guide
- [website/advanced/performance.md](website/advanced/performance.md) - Performance benchmarks
- [docs/DEVELOPMENT.md#testing-standards](docs/DEVELOPMENT.md#testing-standards) - Test coverage standards
- [pkg/agentos/README.md](pkg/agentos/README.md) - AgentOS API guide
- [pkg/agentos/openapi.yaml](pkg/agentos/openapi.yaml) - OpenAPI specification

### 🙏 Acknowledgments

Agno-Go is inspired by and compatible with the design philosophy of:
- [Agno](https://github.com/agno-agi/agno) - Python multi-agent framework

### 📝 Migration from Python Agno

Agno-Go maintains API compatibility where possible, making migration straightforward:

**Python:**
```python
from agno.agent import Agent
from agno.models.openai import OpenAI

agent = Agent(
    name="Assistant",
    model=OpenAI(id="gpt-4"),
)
response = agent.run("Hello!")
```

**Go:**
```go
import (
    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
)

model, _ := openai.New("gpt-4", openai.Config{...})
ag, _ := agent.New(agent.Config{
    Name: "Assistant",
    Model: model,
})
output, _ := ag.Run(ctx, "Hello!")
```

### 🚀 Getting Started

**Installation:**
```bash
go get github.com/rexleimo/agno-go
```

**Quick Start:**
```bash
# Clone repository
git clone https://github.com/rexleimo/agno-go
cd agno-Go

# Run example
export OPENAI_API_KEY=sk-...
go run cmd/examples/simple_agent/main.go

# Or use Docker
docker-compose up -d
curl http://localhost:8080/health
```

### 🛣️ Roadmap

**v1.1** (Planned)
- Streaming response support
- More tool integrations
- Additional vector databases
- Enhanced monitoring/metrics

**v1.2** (Planned)
- gRPC API support
- WebSocket for real-time updates
- Plugin system
- Advanced workflow features

### 📄 License

MIT License - See [LICENSE](LICENSE) for details.

### 🔗 Links

- **GitHub:** https://github.com/rexleimo/agno-go
- **Documentation:** https://docs.agno.com
- **Issues:** https://github.com/rexleimo/agno-go/issues
- **Discussions:** https://github.com/rexleimo/agno-go/discussions

---

**Full Changelog:** https://github.com/rexleimo/agno-go/commits/v1.0.0
