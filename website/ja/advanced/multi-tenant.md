---
title: マルチテナントサポート
description: 単一エージェントインスタンスで複数のユーザーにサービスを提供するためのマルチテナントデータ分離
outline: deep
---

# マルチテナントサポート

**マルチテナントサポート**により、Agno-Goは単一のAgentインスタンスで複数のユーザーにサービスを提供し、ユーザー間の会話履歴とセッション状態の完全な分離を保証します。

---

## 概要

マルチテナントアーキテクチャにより、単一のアプリケーションインスタンスで複数のユーザー（テナント）に完全なデータ分離を提供できます：

```
                 ┌─────────────────┐
                 │ Agent Instance  │
                 └────────┬────────┘
                          │
         ┌────────────────┼────────────────┐
         ▼                ▼                ▼
   ┌──────────┐     ┌──────────┐     ┌──────────┐
   │ User A   │     │ User B   │     │ User C   │
   │ Messages │     │ Messages │     │ Messages │
   └──────────┘     └──────────┘     └──────────┘
```

---

## マルチテナンシーとは？

マルチテナンシーは、単一のアプリケーションインスタンスが複数の分離されたユーザーまたは組織にサービスを提供するアーキテクチャパターンです。各テナントのデータは他のテナントから完全に分離されます。

### マルチテナントなし

```go
// ❌ 各ユーザーに個別のAgentインスタンスが必要
userAgents := make(map[string]*agent.Agent)

agent1, _ := agent.New(config)  // User 1
agent2, _ := agent.New(config)  // User 2
agent3, _ := agent.New(config)  // User 3
// ... 1000+ユーザー = 1000+ Agentインスタンス
```

**問題点:**
- 高メモリ使用量: 1000ユーザー = 1000 Agentインスタンス
- 管理が困難: 手動でのエージェントライフサイクル管理
- リソースの無駄: 各エージェントに重複した設定

### マルチテナントあり

```go
// ✅ 単一のAgentインスタンスがすべてのユーザーにサービスを提供
sharedAgent, _ := agent.New(config)

// 異なるユーザーは異なるuserIDを使用
output1, _ := sharedAgent.Run(ctx, "user-1 input", "user-1")
output2, _ := sharedAgent.Run(ctx, "user-2 input", "user-2")
output3, _ := sharedAgent.Run(ctx, "user-3 input", "user-3")
```

**メリット:**
- ✅ 低メモリ使用量: 単一のAgentインスタンス
- ✅ 簡単な管理: 統一された設定と更新
- ✅ 効率的なリソース利用: 共有モデルとツール

---

## クイックスタート

### 1. マルチテナントAgentの作成

```go
package main

import (
    "context"
    "fmt"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/memory"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
)

func main() {
    // モデルを作成
    model, _ := openai.New("gpt-4", openai.Config{
        APIKey: "your-api-key",
    })

    // マルチテナントMemoryを作成
    mem := memory.NewInMemory(100)  // 自動的にマルチテナンシーをサポート

    // Agentを作成
    myAgent, _ := agent.New(&agent.Config{
        Name:         "customer-service",
        Model:        model,
        Memory:       mem,
        Instructions: "You are a helpful customer service agent.",
    })

    // 異なるユーザーの会話
    ctx := context.Background()

    // ユーザーAの会話
    myAgent.UserID = "user-a"
    output1, _ := myAgent.Run(ctx, "My name is Alice")
    fmt.Printf("User A: %s\n", output1.Content)

    output2, _ := myAgent.Run(ctx, "What's my name?")  // "Your name is Alice"
    fmt.Printf("User A: %s\n", output2.Content)

    // ユーザーBの会話
    myAgent.UserID = "user-b"
    output3, _ := myAgent.Run(ctx, "My name is Bob")
    fmt.Printf("User B: %s\n", output3.Content)

    output4, _ := myAgent.Run(ctx, "What's my name?")  // "Your name is Bob"
    fmt.Printf("User B: %s\n", output4.Content)

    // ユーザーAが再度話す
    myAgent.UserID = "user-a"
    output5, _ := myAgent.Run(ctx, "What's my name?")  // "Your name is Alice"
    fmt.Printf("User A: %s\n", output5.Content)
}
```

### 2. Web API の例

```go
package main

import (
    "context"
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/rexleimo/agno-go/pkg/hno/agent"
)

var sharedAgent *agent.Agent

func main() {
    // Agentを初期化
    sharedAgent, _ = agent.New(&agent.Config{
        Name:   "api-agent",
        Model:  model,
        Memory: memory.NewInMemory(100),
    })

    // ルートを設定
    router := gin.Default()
    router.POST("/chat", handleChat)
    router.Run(":8080")
}

type ChatRequest struct {
    UserID  string `json:"user_id"`
    Message string `json:"message"`
}

type ChatResponse struct {
    UserID  string `json:"user_id"`
    Reply   string `json:"reply"`
}

func handleChat(c *gin.Context) {
    var req ChatRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // 現在のユーザーIDを設定
    sharedAgent.UserID = req.UserID

    // 会話を実行
    output, err := sharedAgent.Run(context.Background(), req.Message)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, ChatResponse{
        UserID: req.UserID,
        Reply:  output.Content,
    })
}
```

**テスト:**
```bash
# ユーザーAの会話
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-a", "message": "My name is Alice"}'

curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-a", "message": "What is my name?"}'
# レスポンス: {"user_id":"user-a","reply":"Your name is Alice"}

# ユーザーBの会話
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-b", "message": "My name is Bob"}'

curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-b", "message": "What is my name?"}'
# レスポンス: {"user_id":"user-b","reply":"Your name is Bob"}
```

---

## メモリ管理

### Memoryインターフェース

Memoryインターフェースはオプションの`userID`パラメータをサポートします：

```go
// pkg/hno/memory/memory.go

type Memory interface {
    // メッセージを追加（オプションのuserIDをサポート）
    Add(message *types.Message, userID ...string)

    // メッセージ履歴を取得（オプションのuserIDをサポート）
    GetMessages(userID ...string) []*types.Message

    // 特定ユーザーのメッセージをクリア
    Clear(userID ...string)

    // すべてのユーザーのメッセージをクリア
    ClearAll()

    // 特定ユーザーのメッセージ数を取得
    Size(userID ...string) int
}
```

### InMemory実装

```go
type InMemory struct {
    userMessages map[string][]*types.Message  // User ID → メッセージリスト
    maxSize      int
    mu           sync.RWMutex
}

// デフォルトユーザーID
const defaultUserID = "default"

// ユーザーIDを取得（下位互換性）
func getUserID(userID ...string) string {
    if len(userID) > 0 && userID[0] != "" {
        return userID[0]
    }
    return defaultUserID
}
```

### 使用例

#### 基本的な使用法

```go
mem := memory.NewInMemory(100)

// ユーザーAのメッセージ
mem.Add(types.NewUserMessage("Hello from Alice"), "user-a")
mem.Add(types.NewAssistantMessage("Hi Alice!"), "user-a")

// ユーザーBのメッセージ
mem.Add(types.NewUserMessage("Hello from Bob"), "user-b")
mem.Add(types.NewAssistantMessage("Hi Bob!"), "user-b")

// 各ユーザーのメッセージを取得
messagesA := mem.GetMessages("user-a")  // 2メッセージ
messagesB := mem.GetMessages("user-b")  // 2メッセージ

fmt.Printf("User A has %d messages\n", len(messagesA))  // 2
fmt.Printf("User B has %d messages\n", len(messagesB))  // 2
```

---

## Agent統合

### Agent設定

```go
type Agent struct {
    ID           string
    Name         string
    Model        models.Model
    Toolkits     []toolkit.Toolkit
    Memory       memory.Memory
    Instructions string
    MaxLoops     int

    UserID string  // ⭐ NEW: マルチテナントユーザーID
}

type Config struct {
    Name         string
    Model        models.Model
    Toolkits     []toolkit.Toolkit
    Memory       memory.Memory
    Instructions string
    MaxLoops     int

    UserID string  // ⭐ NEW: マルチテナントユーザーID
}
```

### Runメソッドの実装

```go
// pkg/hno/agent/agent.go

func (a *Agent) Run(ctx context.Context, input string) (*RunOutput, error) {
    // ...

    // すべてのMemory呼び出しはUserIDを渡す
    userMsg := types.NewUserMessage(input)
    a.Memory.Add(userMsg, a.UserID)  // ⭐ UserIDを渡す

    // ...

    messages := a.Memory.GetMessages(a.UserID)  // ⭐ UserIDを渡す

    // ...

    a.Memory.Add(types.NewAssistantMessage(content), a.UserID)  // ⭐ UserIDを渡す
}
```

---

## データ分離保証

### 1. メモリ分離

```go
// テスト: マルチテナント分離
mem := memory.NewInMemory(100)

// ユーザーAが10メッセージを追加
for i := 0; i < 10; i++ {
    mem.Add(types.NewUserMessage(fmt.Sprintf("User A message %d", i)), "user-a")
}

// ユーザーBが5メッセージを追加
for i := 0; i < 5; i++ {
    mem.Add(types.NewUserMessage(fmt.Sprintf("User B message %d", i)), "user-b")
}

// 分離を検証
assert.Equal(t, 10, mem.Size("user-a"))  // ✅
assert.Equal(t, 5, mem.Size("user-b"))   // ✅
assert.Equal(t, 0, mem.Size("user-c"))   // ✅ 存在しないユーザー
```

### 2. 並行安全性

```go
// テスト: 1000並行リクエスト
mem := memory.NewInMemory(100)
var wg sync.WaitGroup

// 10ユーザー、各100並行リクエスト
for userID := 0; userID < 10; userID++ {
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(uid, msgID int) {
            defer wg.Done()
            userIDStr := fmt.Sprintf("user-%d", uid)
            msg := types.NewUserMessage(fmt.Sprintf("Message %d", msgID))
            mem.Add(msg, userIDStr)
        }(userID, i)
    }
}

wg.Wait()

// 各ユーザーが正しいメッセージ数を持つことを検証
for userID := 0; userID < 10; userID++ {
    userIDStr := fmt.Sprintf("user-%d", userID)
    assert.Equal(t, 100, mem.Size(userIDStr))  // ✅
}
```

---

## ベストプラクティス

### 1. UserID命名規則

```go
// ✅ 推奨: 一貫した命名規則を使用
"user-{uuid}"           // user-123e4567-e89b-12d3-a456-426614174000
"org-{org_id}-user-{id}" // org-acme-user-001
"tenant-{id}"           // tenant-12345

// ❌ 避ける: 不安定な識別子を使用
"{ip_address}"          // IPは変更される可能性がある
"{session_id}"          // セッションは期限切れになる
```

### 2. エラー処理

```go
// UserIDを検証
func validateUserID(userID string) error {
    if userID == "" {
        return fmt.Errorf("userID cannot be empty")
    }
    if len(userID) > 255 {
        return fmt.Errorf("userID too long (max 255 chars)")
    }
    return nil
}

// APIレイヤーで検証
func handleChat(c *gin.Context) {
    var req ChatRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    if err := validateUserID(req.UserID); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // ...
}
```

---

## トラブルシューティング

### よくある問題

#### 1. ユーザーデータの混同

**症状:** ユーザーAがユーザーBのメッセージを見る

**原因:** UserIDが適切に渡されていない

**解決策:**
```go
// ❌ 間違い
agent.Run(ctx, input)  // UserIDが設定されていない

// ✅ 正しい
agent.UserID = "user-a"
agent.Run(ctx, input)
```

#### 2. 高メモリ使用量

**症状:** メモリが継続的に増加

**原因:** 非アクティブなユーザーデータがクリーンアップされていない

**解決策:**
```go
// 定期的なクリーンアップ
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        cleanupInactiveUsers(mem, 24*time.Hour)
    }
}()
```

---

## 統合

### A2A Interface + マルチテナント

```go
// A2AリクエストにcontextIDが含まれ、userIDとして使用可能
type Message struct {
    MessageID string `json:"messageId"`
    Role      string `json:"role"`
    AgentID   string `json:"agentId"`
    ContextID string `json:"contextId"`  // ⭐ userIDとして使用可能
    Parts     []Part `json:"parts"`
}

// マッピング中にUserIDを設定
func MapA2ARequestToRunInput(req *JSONRPC2Request) (*RunInput, error) {
    // ...
    agent.UserID = req.Params.Message.ContextID  // ⭐ contextIDをuserIDとして使用
    // ...
}
```

---

## 関連ドキュメント

- [A2Aインターフェース](/ja/api/a2a) - エージェント間通信
- [セッション状態管理](/ja/guide/session-state) - ワークフローセッション管理
- [メモリガイド](/ja/guide/memory) - メモリ使用ガイド

---

## テスト

完全なテストカバレッジには以下が含まれます：

- ✅ マルチユーザーデータ分離
- ✅ 並行安全性（1000 goroutines）
- ✅ Agent統合テスト
- ✅ メモリ容量管理

**テストカバレッジ:** 93.1%（メモリモジュール）

テストを実行：
```bash
cd pkg/hno/memory
go test -v -run TestInMemory

cd pkg/hno/agent
go test -v -run TestAgent_MultiTenant
```

---

**最終更新:** 2025-01-08
