# M1 设计草案：会话持久化与迁移（SQLite）

目标
- 将 `session.Storage` 从内存扩展到 SQLite，满足企业级持久化与并发稳定性。
- 保留并验证 `updated_at` 语义，确保更新时序一致。

范围
- 新增 SQLite 存储实现与表结构（含索引、触发器）。
- 在 AgentOS 启动时可配置使用 SQLite 存储。
- 提供最小迁移/导入工具与测试用例。

目录结构（拟）
- `pkg/hno/session/sqlite/`
  - `storage.go`（实现 `session.Storage`）
  - `models.go`（行对象与扫描器）
  - `schema.sql`（表结构/索引/触发器）
  - `options.go`（DSN、连接池、超时配置）
- `scripts/init-db.sql`（生产/本地初始化）
- `cmd/tools/session_migrate/main.go`（可选：导入 JSON → SQLite）

表结构（建议）
```sql
CREATE TABLE IF NOT EXISTS sessions (
  session_id   TEXT PRIMARY KEY,
  agent_id     TEXT NOT NULL,
  user_id      TEXT,
  name         TEXT,
  metadata     TEXT,            -- JSON 字符串
  created_at   DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  updated_at   DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_sessions_agent ON sessions(agent_id);
CREATE INDEX IF NOT EXISTS idx_sessions_user  ON sessions(user_id);

CREATE TRIGGER IF NOT EXISTS trg_sessions_updated
AFTER UPDATE ON sessions
FOR EACH ROW
BEGIN
  UPDATE sessions SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE session_id = NEW.session_id;
END;
```

接口映射
- `Create(ctx, *Session) error` → INSERT
- `Get(ctx, id string) (*Session, error)` → SELECT by PK
- `Update(ctx, *Session) error` → UPDATE（触发器自动更新 `updated_at`）
- `Delete(ctx, id string) error` → DELETE
- `List(ctx, filters map[string]interface{}) ([]*Session, error)` → 动态 WHERE（支持 `agent_id`/`user_id`）
- `ListByAgent(ctx, agentID string)` / `ListByUser(ctx, userID string)` → 便捷查询
- `Close() error` → 关闭连接

AgentOS 集成
- 在 `pkg/agentos/server.go` 的 `NewServer` 中：
  - 读取 `AGENTOS_SESSION_STORAGE`（`memory`|`sqlite`）与 `AGENTOS_SQLITE_DSN`（例如 `file:agentos.db?_busy_timeout=5000&_journal_mode=WAL`）。
  - 当配置为 `sqlite` 时注入 `sqlite.NewStorage(dsn, sqlite.Options{BusyTimeout: 5s})`。

最小用法示例（伪代码）
```go
// main.go
cfg := &agentos.Config{}
if os.Getenv("AGENTOS_SESSION_STORAGE") == "sqlite" {
    dsn := getenv("AGENTOS_SQLITE_DSN", "file:agentos.db?_busy_timeout=5000&_journal_mode=WAL")
    st, err := sqlite.NewStorage(dsn, sqlite.Options{BusyTimeout: 5 * time.Second})
    if err != nil { log.Fatal(err) }
    cfg.SessionStorage = st
}
server, _ := agentos.NewServer(cfg)
server.Start()
```

测试计划（关键用例）
- 单元测试
  - CRUD：创建/查询/更新/删除/列表；空输入与不存在 ID。
  - 过滤：`List` 按 `agent_id`/`user_id`，边界值（空、未知）。
  - 时间语义：更新后 `updated_at` 严格大于等于创建时间；多次更新单调递增。
- 并发与可靠性
  - 并发创建与更新（100 并发），无死锁；重试策略（`busy_timeout`）生效。
  - 重启恢复：写入后关闭连接，再次打开读取一致。

迁移/导入（可选）
- 工具：`cmd/tools/session_migrate` 支持从 JSON dump 导入到 SQLite：
  - 入参：`--from-json sessions.json --to-dsn file:agentos.db?...`；
  - 行为：逐条校验（ID 唯一、时间格式）、批量插入、失败回滚。

运维与安全
- DSN 建议启用 WAL、busy_timeout；文件权限限制（0600）。
- 备份：定期复制 `*.db`；恢复前做一致性校验。

里程碑验收标准
- 全部单测通过；新增存储被 `server_test.go` 集成使用且通过。
- 本地端到端验证：创建/查询/更新/删除 Session 经 REST 接口可见变化。
- 文档与示例：在 README/DEVELOPMENT 中记录环境变量与用法。
