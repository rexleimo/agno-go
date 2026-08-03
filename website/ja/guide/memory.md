# Memory - 会話履歴

Agentの会話履歴とコンテキストを管理します。

---

## Memoryとは？

Memoryは会話履歴を保存し、Agentが複数のやり取りにわたってコンテキストを維持できるようにします。Agno-Goは、自動切り捨て機能を備えた組み込みメモリ管理を提供します。

### 主な機能

- **自動履歴**: 会話が自動的に保存される
- **設定可能な制限**: メモリサイズを制御
- **メッセージタイプ**: System、User、Assistant、Toolメッセージ
- **手動制御**: メモリをプログラムでクリアまたは管理

---

## 基本的な使い方

### デフォルトMemory

Agentはデフォルトでメモリが有効:

```go
package main

import (
    "context"
    "fmt"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: "your-api-key",
    })

    agent, _ := agent.New(agent.Config{
        Model: model,
    })

    // 最初のやり取り
    agent.Run(context.Background(), "My name is Alice")
    // 応答: "Nice to meet you, Alice!"

    // 2回目のやり取り - Agentは記憶している
    output, _ := agent.Run(context.Background(), "What's my name?")
    fmt.Println(output.Content)
    // 応答: "Your name is Alice."
}
```

---

## 設定

### カスタムMemory制限

保存する最大メッセージ数を設定:

```go
import "github.com/rexleimo/agno-go/pkg/hno/memory"

customMemory := memory.New(memory.Config{
    MaxMessages: 50,  // 最大50メッセージを保存
})

agent, _ := agent.New(agent.Config{
    Model:  model,
    Memory: customMemory,
})
```

### Memoryなし

ステートレスAgentのためにメモリを無効化:

```go
agent, _ := agent.New(agent.Config{
    Model:  model,
    Memory: nil,  // 会話履歴なし
})
```

---

## Memory操作

### Memoryをクリア

会話履歴をリセット:

```go
// すべての履歴をクリア
agent.ClearMemory()

// 新しい会話を開始
agent.Run(ctx, "New conversation")
```

### メッセージ履歴を取得

保存されたメッセージにアクセス:

```go
messages := agent.Memory.GetMessages()
for _, msg := range messages {
    fmt.Printf("%s: %s\n", msg.Role, msg.Content)
}
```

### カスタムメッセージを追加

手動でメモリにメッセージを追加:

```go
import "github.com/rexleimo/agno-go/pkg/hno/types"

// システムメッセージを追加
agent.Memory.AddMessage(types.Message{
    Role:    types.RoleSystem,
    Content: "You are a helpful assistant.",
})

// ユーザーメッセージを追加
agent.Memory.AddMessage(types.Message{
    Role:    types.RoleUser,
    Content: "Hello!",
})
```

---

## メッセージタイプ

### システムメッセージ

Agentへの指示:

```go
types.Message{
    Role:    types.RoleSystem,
    Content: "You are a Python expert. Help with coding questions.",
}
```

### ユーザーメッセージ

ユーザー入力:

```go
types.Message{
    Role:    types.RoleUser,
    Content: "How do I read a file in Python?",
}
```

### アシスタントメッセージ

Agentの応答:

```go
types.Message{
    Role:    types.RoleAssistant,
    Content: "Use the open() function: open('file.txt', 'r')",
}
```

### ツールメッセージ

ツール実行結果:

```go
types.Message{
    Role:       types.RoleTool,
    Content:    "Result: 42",
    ToolCallID: "call_123",
}
```

---

## Memoryパターン

### セッションベースMemory

セッション間でメモリをクリア:

```go
func handleSession(agent *agent.Agent, sessionID string) {
    // セッション履歴を読み込む（データベース等から）
    loadSessionHistory(agent, sessionID)

    // 会話を処理
    output, _ := agent.Run(ctx, userInput)

    // セッション履歴を保存
    saveSessionHistory(agent, sessionID)

    // クリーンアップ
    agent.ClearMemory()
}
```

### スライディングウィンドウ

最新のメッセージのみを保持:

```go
memory := memory.New(memory.Config{
    MaxMessages: 20,  // 最新20メッセージを保持
})

// 古いメッセージを自動的に切り捨て
agent, _ := agent.New(agent.Config{
    Memory: memory,
})
```

### 永続的Memory

会話を保存して復元:

```go
// 会話を保存
messages := agent.Memory.GetMessages()
saveToDatabase(sessionID, messages)

// 会話を復元
savedMessages := loadFromDatabase(sessionID)
for _, msg := range savedMessages {
    agent.Memory.AddMessage(msg)
}
```

---

## 高度な使用法

### マルチAgentメモリ共有

Agent間でコンテキストを共有:

```go
// 共有メモリを作成
sharedMemory := memory.New(memory.Config{
    MaxMessages: 100,
})

// 両方のAgentが同じメモリを使用
agent1, _ := agent.New(agent.Config{
    Name:   "Agent1",
    Model:  model,
    Memory: sharedMemory,
})

agent2, _ := agent.New(agent.Config{
    Name:   "Agent2",
    Model:  model,
    Memory: sharedMemory,
})

// Agent1の会話はAgent2から見える
agent1.Run(ctx, "Store this information: X=42")
output, _ := agent2.Run(ctx, "What is X?")
// Agent2はAgent1の会話を見ることができる
```

### 条件付きMemory

条件に基づいてメモリをクリア:

```go
messageCount := len(agent.Memory.GetMessages())

if messageCount > 100 {
    // システムメッセージのみを保持
    systemMsg := agent.Memory.GetMessages()[0]
    agent.ClearMemory()
    agent.Memory.AddMessage(systemMsg)
}
```

### Memory検査

会話履歴を分析:

```go
messages := agent.Memory.GetMessages()

var userMessages, assistantMessages int
for _, msg := range messages {
    switch msg.Role {
    case types.RoleUser:
        userMessages++
    case types.RoleAssistant:
        assistantMessages++
    }
}

fmt.Printf("User messages: %d, Assistant messages: %d\n",
    userMessages, assistantMessages)
```

---

## Memory設定

### Config構造体

```go
type Config struct {
    MaxMessages int // 保存する最大メッセージ数（デフォルト: 100）
}
```

### デフォルト動作

- すべての会話メッセージを自動的に保存
- 制限に達したときに最も古いメッセージを切り捨て
- 切り捨て中もシステムメッセージは保持

---

## ベストプラクティス

### 1. 適切な制限を設定

コンテキストとパフォーマンスのバランスを取る:

```go
// 短い会話
memory := memory.New(memory.Config{MaxMessages: 20})

// 長い会話
memory := memory.New(memory.Config{MaxMessages: 100})

// 非常に長いコンテキスト
memory := memory.New(memory.Config{MaxMessages: 500})
```

### 2. 戦略的にMemoryをクリア

コンテキストが変わったときにリセット:

```go
// 新しいトピック
if isNewTopic(userInput) {
    agent.ClearMemory()
}

// 新しいセッション
if isNewSession(sessionID) {
    agent.ClearMemory()
}
```

### 3. Memory使用量を監視

会話の長さを追跡:

```go
messages := agent.Memory.GetMessages()
if len(messages) > 80 {
    log.Printf("Warning: Approaching memory limit (%d/100)", len(messages))
}
```

### 4. 重要なコンテキストを保持

システム指示を保持:

```go
// システムメッセージを保存
systemMsg := agent.Memory.GetMessages()[0]

// メモリをクリア
agent.ClearMemory()

// システムメッセージを復元
agent.Memory.AddMessage(systemMsg)
```

---

## Memory vs コンテキストウィンドウ

### Memory（Agno-Go）
- Agno-Goによって管理
- 設定可能なメッセージ制限
- 自動切り捨て

### コンテキストウィンドウ（LLM）
- モデル固有の制限（例: 128Kトークン）
- LLMプロバイダーによって管理
- 超過するとエラーが発生する可能性

**ベストプラクティス**: Memory制限をLLMコンテキストウィンドウより低く保つ。

```go
// GPT-4o-mini: 128Kトークン ≈ 100K単語 ≈ 400メッセージ
memory := memory.New(memory.Config{MaxMessages: 200})
```

---

## トラブルシューティング

### Agentが記憶しない

メモリ設定を確認:

```go
// 悪い例 ❌ - メモリなし
agent, _ := agent.New(agent.Config{
    Model:  model,
    Memory: nil,
})

// 良い例 ✅ - メモリあり
agent, _ := agent.New(agent.Config{
    Model: model,
    // デフォルトでメモリが有効
})
```

### Memoryが大きすぎる

メッセージ制限を減らす:

```go
memory := memory.New(memory.Config{
    MaxMessages: 50,  // より小さい制限
})
```

### コンテキストの喪失

不必要にメモリをクリアしない:

```go
// 悪い例 ❌ - 各メッセージ後にクリア
output, _ := agent.Run(ctx, input)
agent.ClearMemory() // これはしないでください

// 良い例 ✅ - コンテキストを保持
output, _ := agent.Run(ctx, input)
// 次のやり取りのためにメモリを維持
```

---

## 例

### マルチターン会話

```go
agent, _ := agent.New(agent.Config{Model: model})

// ターン1
agent.Run(ctx, "I'm planning a trip to Paris")

// ターン2
agent.Run(ctx, "What's the weather like there?")
// Agentは「there」= Parisと理解

// ターン3
agent.Run(ctx, "What should I pack?")
// AgentはParisと天候について知っている
```

### セッション管理

```go
type SessionManager struct {
    agents map[string]*agent.Agent
}

func (sm *SessionManager) GetAgent(sessionID string) *agent.Agent {
    if ag, exists := sm.agents[sessionID]; exists {
        return ag
    }

    // セッション用の新しいAgentを作成
    ag, _ := agent.New(agent.Config{Model: model})
    sm.agents[sessionID] = ag
    return ag
}

func (sm *SessionManager) EndSession(sessionID string) {
    if ag, exists := sm.agents[sessionID]; exists {
        ag.ClearMemory()
        delete(sm.agents, sessionID)
    }
}
```

---

## 次のステップ

- 共有メモリで[Teams](/guide/team)を構築
- 機能を拡張するには[Tools](/guide/tools)を追加
- コンテキスト渡しで[Workflows](/guide/workflow)を作成
- 詳細なドキュメントは[Memory APIリファレンス](/api/memory)を確認

---

## 関連例

- [Simple Agent](/examples/simple-agent) - 基本的なメモリ使用
- [Multi-Turn Chat](/examples/chat-agent) - 会話例
- [Session Management](/examples/session-demo) - セッションベースのメモリ
