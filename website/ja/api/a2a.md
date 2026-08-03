---
title: A2A インターフェース
description: JSON-RPC 2.0 ベースの標準化されたエージェント間通信
outline: deep
---

# Agent-to-Agent Interface (A2A)

## 概要

**A2A (Agent-to-Agent) インターフェース**は、JSON-RPC 2.0に基づく標準化されたエージェント間通信プロトコルで、同期およびストリーミング通信モードをサポートしています。

### プロトコル標準
- **JSON-RPC 2.0**: 業界標準のRPCプロトコル
- **Server-Sent Events (SSE)**: ストリーミングレスポンス転送
- **RESTful HTTP**: HTTPベースのエンドポイント実装

### コアコンポーネント

```
pkg/agentos/a2a/
├── types.go      # プロトコル型定義
├── validator.go  # リクエスト検証
├── mapper.go     # プロトコルマッピング
├── a2a.go        # A2Aインターフェース管理
└── handlers.go   # HTTPハンドラ
```

## クイックスタート

### 1. A2Aインターフェースの作成

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/rexleimo/agno-go/pkg/agentos/a2a"
    "github.com/rexleimo/agno-go/pkg/hno/agent"
)

func main() {
    // エージェントを作成
    myAgent, _ := agent.New(&agent.Config{
        Name:  "my-agent",
        Model: model,
        // ... その他の設定
    })

    // A2Aインターフェースを作成
    a2aInterface := a2a.New("/api/v1/a2a")

    // エージェントをエンティティとして登録
    a2aInterface.RegisterEntity("my-agent", myAgent)

    // Ginルートを設定
    router := gin.Default()
    a2aInterface.SetupRoutes(router)

    router.Run(":8080")
}
```

### 2. 同期メッセージの送信

```bash
curl -X POST http://localhost:8080/api/v1/a2a/sendMessage \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "sendMessage",
    "id": "req-001",
    "params": {
      "message": {
        "messageId": "msg-001",
        "role": "user",
        "agentId": "my-agent",
        "contextId": "session-123",
        "parts": [
          {
            "type": "text",
            "content": "Hello, agent!"
          }
        ]
      }
    }
  }'
```

**レスポンス例**:
```json
{
  "jsonrpc": "2.0",
  "id": "req-001",
  "result": {
    "task": {
      "taskId": "task-001",
      "status": "completed",
      "messages": [
        {
          "messageId": "msg-002",
          "role": "assistant",
          "agentId": "my-agent",
          "contextId": "session-123",
          "parts": [
            {
              "type": "text",
              "content": "Hello! How can I help you?"
            }
          ]
        }
      ]
    }
  }
}
```

### 3. ストリーミングメッセージの送信

```bash
curl -X POST http://localhost:8080/api/v1/a2a/streamMessage \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "method": "streamMessage",
    "id": "req-002",
    "params": {
      "message": {
        "messageId": "msg-003",
        "role": "user",
        "agentId": "my-agent",
        "contextId": "session-123",
        "parts": [
          {
            "type": "text",
            "content": "Tell me a story"
          }
        ]
      }
    }
  }'
```

**SSEレスポンスストリーム**:
```
data: {"type":"content","content":"Once"}

data: {"type":"content","content":" upon"}

data: {"type":"content","content":" a"}

data: {"type":"content","content":" time..."}

data: {"type":"done"}
```

## プロトコル詳細

### JSON-RPC 2.0リクエストフォーマット

```go
type JSONRPC2Request struct {
    JSONRPC string        `json:"jsonrpc"`  // "2.0"である必要があります
    Method  string        `json:"method"`   // "sendMessage"または"streamMessage"
    ID      string        `json:"id"`       // 一意のリクエストID
    Params  RequestParams `json:"params"`   // リクエストパラメータ
}

type RequestParams struct {
    Message Message `json:"message"`  // メッセージ内容
}
```

### メッセージ構造

```go
type Message struct {
    MessageID string `json:"messageId"`  // 一意のメッセージID
    Role      string `json:"role"`       // "user"または"assistant"
    AgentID   string `json:"agentId"`    // ターゲットエージェントID
    ContextID string `json:"contextId"`  // セッションコンテキストID
    Parts     []Part `json:"parts"`      // メッセージパーツ
}

type Part struct {
    Type    string `json:"type"`              // "text"または"data"
    Content string `json:"content,omitempty"` // テキスト内容
    Data    string `json:"data,omitempty"`    // 構造化データ (JSON)
}
```

### レスポンスフォーマット

#### 成功レスポンス

```go
type JSONRPC2Response struct {
    JSONRPC string        `json:"jsonrpc"`  // "2.0"
    ID      string        `json:"id"`       // マッチするリクエストID
    Result  *ResultObject `json:"result"`   // 結果オブジェクト
}

type ResultObject struct {
    Task Task `json:"task"`  // タスク情報
}

type Task struct {
    TaskID   string    `json:"taskId"`   // タスクID
    Status   string    `json:"status"`   // "completed"または"failed"
    Messages []Message `json:"messages"` // レスポンスメッセージ
}
```

#### エラーレスポンス

```go
type JSONRPC2Response struct {
    JSONRPC string       `json:"jsonrpc"`
    ID      string       `json:"id"`
    Error   *ErrorObject `json:"error"`
}

type ErrorObject struct {
    Code    int    `json:"code"`    // エラーコード
    Message string `json:"message"` // エラーメッセージ
}
```

**標準エラーコード**:
- `-32700`: Parse error (JSON解析失敗)
- `-32600`: Invalid Request (無効なリクエストフォーマット)
- `-32601`: Method not found (メソッドが存在しません)
- `-32602`: Invalid params (無効なパラメータ)
- `-32603`: Internal error (内部サーバーエラー)

## 検証メカニズム

### リクエスト検証

A2Aインターフェースは完全なリクエスト検証を提供します：

```go
func ValidateRequest(req *JSONRPC2Request) error {
    // 1. JSON-RPCバージョンをチェック
    if req.JSONRPC != "2.0" {
        return fmt.Errorf("invalid jsonrpc version, must be 2.0")
    }

    // 2. メソッドをチェック
    if req.Method != "sendMessage" && req.Method != "streamMessage" {
        return fmt.Errorf("invalid method, must be sendMessage or streamMessage")
    }

    // 3. リクエストIDをチェック
    if req.ID == "" {
        return fmt.Errorf("request id is required")
    }

    // 4. メッセージを検証
    return ValidateMessage(&req.Params.Message)
}
```

## 完全な例

### サーバーサイド

```go
package main

import (
    "context"
    "log"

    "github.com/gin-gonic/gin"
    "github.com/rexleimo/agno-go/pkg/agentos/a2a"
    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
)

func main() {
    // 1. モデルを作成
    model, err := openai.New("gpt-4", openai.Config{
        APIKey: "your-api-key",
    })
    if err != nil {
        log.Fatal(err)
    }

    // 2. エージェントを作成
    myAgent, err := agent.New(&agent.Config{
        Name:         "customer-service",
        Model:        model,
        Instructions: "You are a helpful customer service agent.",
    })
    if err != nil {
        log.Fatal(err)
    }

    // 3. A2Aインターフェースを作成
    a2aInterface := a2a.New("/api/v1/a2a")
    a2aInterface.RegisterEntity("customer-service", myAgent)

    // 4. ルートを設定
    router := gin.Default()
    a2aInterface.SetupRoutes(router)

    // 5. サーバーを起動
    log.Println("A2A server listening on :8080")
    router.Run(":8080")
}
```

### クライアントサイド (Go)

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"

    "github.com/rexleimo/agno-go/pkg/agentos/a2a"
)

func main() {
    // リクエストを構築
    request := &a2a.JSONRPC2Request{
        JSONRPC: "2.0",
        Method:  "sendMessage",
        ID:      "req-001",
        Params: a2a.RequestParams{
            Message: a2a.Message{
                MessageID: "msg-001",
                Role:      "user",
                AgentID:   "customer-service",
                ContextID: "session-123",
                Parts: []a2a.Part{
                    {
                        Type:    "text",
                        Content: "How do I return a product?",
                    },
                },
            },
        },
    }

    // シリアライズ
    requestBody, _ := json.Marshal(request)

    // リクエストを送信
    resp, err := http.Post(
        "http://localhost:8080/api/v1/a2a/sendMessage",
        "application/json",
        bytes.NewBuffer(requestBody),
    )
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    // レスポンスを読み取る
    body, _ := io.ReadAll(resp.Body)

    var response a2a.JSONRPC2Response
    json.Unmarshal(body, &response)

    // 結果を処理
    if response.Error != nil {
        fmt.Printf("Error: %s\n", response.Error.Message)
        return
    }

    task := response.Result.Task
    fmt.Printf("Task Status: %s\n", task.Status)
    for _, msg := range task.Messages {
        for _, part := range msg.Parts {
            fmt.Printf("Agent Response: %s\n", part.Content)
        }
    }
}
```

## ベストプラクティス

### 1. エラー処理

```go
// 標準エラーコードを使用
if err := validateInput(); err != nil {
    return &a2a.ErrorObject{
        Code:    -32602, // Invalid params
        Message: err.Error(),
    }
}
```

### 2. ContextID管理

```go
// 各セッションに一意のcontextIdを使用
contextID := fmt.Sprintf("session-%s-%d", userID, time.Now().Unix())

// 同じセッション内のすべてのメッセージは同じcontextIdを使用
message1.ContextID = contextID
message2.ContextID = contextID
```

### 3. 並行処理

```go
// A2Aインターフェースは並行安全です
// 複数のリクエストを同時に処理できます

for i := 0; i < 10; i++ {
    go func(id int) {
        // リクエストを並行送信
        sendMessageToAgent(fmt.Sprintf("req-%d", id))
    }(i)
}
```

### 4. タイムアウト制御

```go
// リクエストタイムアウトを設定
client := &http.Client{
    Timeout: 30 * time.Second,
}

resp, err := client.Post(url, contentType, body)
```

## トラブルシューティング

### よくある問題

#### 1. "Invalid JSON-RPC version"

**原因**: `jsonrpc`フィールドが"2.0"ではありません

**解決策**:
```json
{
  "jsonrpc": "2.0",  // 文字列"2.0"である必要があります
  "method": "sendMessage",
  ...
}
```

#### 2. "Agent not found"

**原因**: `agentId`が登録されていません

**解決策**:
```go
// 登録されたエンティティを確認
entities := a2aInterface.ListEntities()
fmt.Println(entities)

// エージェントが登録されていることを確認
a2aInterface.RegisterEntity("your-agent-id", agent)
```

#### 3. "Invalid message format"

**原因**: メッセージに必須フィールドがありません

**解決策**:
```json
{
  "messageId": "msg-001",     // ✅ 必須
  "role": "user",             // ✅ 必須
  "agentId": "my-agent",      // ✅ 必須
  "contextId": "session-123", // ⚠️ オプションですが推奨
  "parts": [                  // ✅ 必須、最低1つ
    {
      "type": "text",
      "content": "Hello"
    }
  ]
}
```

## 関連ドキュメント

- [セッション状態管理](/ja/guide/session-state) - セッション状態管理
- [マルチテナントサポート](/ja/advanced/multi-tenant) - マルチテナントサポート
- [アーキテクチャ設計](/ja/architecture) - アーキテクチャ設計

---

**最終更新**: 2025-01-XX
