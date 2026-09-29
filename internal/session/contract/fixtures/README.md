# Contract Fixtures

Go↔Python 会话契约的对齐夹具（`contract_test.go` 的 9 个测试消费）。

## 来源与状态

这些文件由切片 28 之后的收尾从 **Go service 的当前投影形状** 导出
（`buildSessionDetail` / `toListItem` / `writeError`，internal/session/service），
用于把此前的 9 个 SKIP 变成真实回归测试：夹具字节 = 服务当前输出，
任何投影改动都会被这些测试当场抓住。

**与 agno-python 的真值方向**：Python 侧的 fixture 若与此处不一致，
以 Python 侧为准更新本目录（测试会立刻暴露差异）——本目录当前锁的是
Go 侧行为回归，不是 Python 契约的权威副本。

## 文件

| 文件 | 消费测试 |
|---|---|
| `get_sessions_agent.json` | TestListSessionsMatchesFixture |
| `get_session_detail_agent.json` | TestGetSessionDetailMatchesFixture |
| `get_session_runs.json` | TestGetSessionRunsMatchesFixture |
| `create_session_agent.json` | TestCreateSessionMatchesFixture |
| `rename_session_agent.json` | TestRenameSessionMatchesFixture |
| `get_sessions_invalid_type_error.json` | TestListSessionsInvalidTypeMatchesErrorFixture |
| `get_session_not_found_error.json` | TestGetSessionDetailNotFoundMatchesErrorFixture |
| `list_sessions_database_required_error.json` | TestListSessionsDatabaseRequiredMatchesErrorFixture |
| `list_sessions_database_not_found_error.json` | TestListSessionsDatabaseNotFoundMatchesErrorFixture |
