---
title: セッション状態管理
description: ワークフロー実行のためのスレッドセーフな状態管理と並列ブランチのインテリジェントマージ
outline: deep
---

# セッション状態管理

**セッション状態管理**は、ワークフロー実行中の状態管理機能を提供し、ステップ間のデータ共有、並行安全な状態アクセス、並列ブランチのインテリジェントマージをサポートします。

## なぜセッション状態が必要か？

複雑なワークフローでは、ステップ間でデータを共有する必要があります：

```
Step1: ユーザー情報を取得
  ↓
  SessionStateに保存: {"user_id": "123", "name": "Alice"}
  ↓
Step2: ユーザー情報に基づいて注文を照会
  ↓
  SessionStateから読み取り: user_id = "123"
  SessionStateに保存: {"orders": [...]}
  ↓
Step3: レポートを生成
  ↓
  SessionStateから読み取り: user_id, name, orders
```

セッション状態がない場合、ステップの出力を通じてデータを渡す必要があり、結合が強くなり複雑になります。セッション状態は、すべてのステップがアクセスできる共有メモリスペースとして機能します。

## コア機能

1. **スレッドセーフ**: `sync.RWMutex`で並行アクセスを保護
2. **ディープコピー**: 並列ブランチは独立した状態コピーを取得
3. **スマートマージ**: 並列実行後の自動状態マージ
4. **柔軟な型**: 任意の`interface{}`型のデータをサポート

## クイックスタート

### 基本的な使用法

```go
package main

import (
    "context"
    "fmt"

    "github.com/rexleimo/agno-go/pkg/hno/workflow"
)

func main() {
    // セッション状態を持つ実行コンテキストを作成
    execCtx := workflow.NewExecutionContextWithSession(
        "initial input",
        "session-123",  // sessionID
        "user-456",     // userID
    )

    // セッション状態を設定
    execCtx.SetSessionState("user_name", "Alice")
    execCtx.SetSessionState("user_age", 30)
    execCtx.SetSessionState("preferences", map[string]string{
        "language": "zh-CN",
        "theme":    "dark",
    })

    // セッション状態を取得
    if name, ok := execCtx.GetSessionState("user_name"); ok {
        fmt.Printf("User Name: %s\n", name)
    }

    if age, ok := execCtx.GetSessionState("user_age"); ok {
        fmt.Printf("User Age: %d\n", age)
    }
}
```

### ワークフローでの使用

```go
// ワークフローを作成
wf := workflow.NewWorkflow("user-workflow")

// ステップ1: ユーザー情報を取得
step1 := workflow.NewStep("get-user", func(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
    // ユーザー情報の取得をシミュレート
    userInfo := map[string]interface{}{
        "id":    "user-123",
        "name":  "Alice",
        "email": "alice@example.com",
    }

    // SessionStateに保存
    execCtx.SetSessionState("user_info", userInfo)

    return execCtx, nil
})

// ステップ2: ユーザーの注文を取得
step2 := workflow.NewStep("get-orders", func(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
    // SessionStateからユーザー情報を読み取る
    userInfoRaw, ok := execCtx.GetSessionState("user_info")
    if !ok {
        return execCtx, fmt.Errorf("user_info not found in session state")
    }

    userInfo := userInfoRaw.(map[string]interface{})
    userID := userInfo["id"].(string)

    // 注文の取得をシミュレート
    orders := []string{"order-1", "order-2", "order-3"}
    execCtx.SetSessionState("orders", orders)

    fmt.Printf("Got %d orders for user %s\n", len(orders), userID)

    return execCtx, nil
})

// ステップを連鎖
step1.Then(step2)
wf.AddStep(step1)

// ワークフローを実行
execCtx := workflow.NewExecutionContextWithSession("", "session-123", "user-456")
result, err := wf.Execute(context.Background(), execCtx)
if err != nil {
    panic(err)
}

// 最終状態を確認
orders, _ := result.GetSessionState("orders")
fmt.Printf("Final orders: %v\n", orders)
```

## 並列実行と状態マージ

### 課題

並列実行中、複数のブランチがSessionStateを同時に変更する可能性があります：

```
              ┌─→ ブランチA: Set("key1", "value_A")
並列ステップ  ├─→ ブランチB: Set("key2", "value_B")
              └─→ ブランチC: Set("key1", "value_C")  // ⚠️ 競合！
```

### 解決策

HNOは**ディープコピー + last-write-wins**戦略を使用します：

1. 各並列ブランチは独立したSessionStateコピーを取得
2. ブランチは干渉なしで独立して実行
3. 完了後、すべての変更が順番にマージされます
4. 競合が存在する場合、後のブランチが前のブランチを上書き

```go
// pkg/hno/workflow/parallel.go

func (p *Parallel) Execute(ctx context.Context, execCtx *ExecutionContext) (*ExecutionContext, error) {
    // 1. 各ブランチに独立したSessionStateコピーを作成
    sessionStateCopies := make([]*SessionState, len(p.Nodes))
    for i := range p.Nodes {
        if execCtx.SessionState != nil {
            sessionStateCopies[i] = execCtx.SessionState.Clone()  // ディープコピー
        } else {
            sessionStateCopies[i] = NewSessionState()
        }
    }

    // 2. ブランチを並列実行
    // ... (goroutines実行)

    // 3. すべてのブランチ状態の変更をマージ
    execCtx.SessionState = MergeParallelSessionStates(
        originalSessionState,
        modifiedSessionStates,
    )

    return execCtx, nil
}
```

### マージ戦略

```go
// pkg/hno/workflow/session_state.go

func MergeParallelSessionStates(original *SessionState, modified []*SessionState) *SessionState {
    merged := NewSessionState()

    // 1. 元の状態をコピー
    if original != nil {
        for k, v := range original.data {
            merged.data[k] = v
        }
    }

    // 2. 各ブランチからの変更を順番にマージ
    for _, modState := range modified {
        if modState == nil {
            continue
        }
        for k, v := range modState.data {
            merged.data[k] = v  // Last-write-wins
        }
    }

    return merged
}
```

### 例

```go
// 3つのブランチの並列実行
parallel := workflow.NewParallel()

// ブランチA: counter = 1を設定
branchA := workflow.NewStep("branch-a", func(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
    execCtx.SetSessionState("counter", 1)
    execCtx.SetSessionState("branch_a_result", "done")
    return execCtx, nil
})

// ブランチB: counter = 2を設定
branchB := workflow.NewStep("branch-b", func(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
    execCtx.SetSessionState("counter", 2)
    execCtx.SetSessionState("branch_b_result", "done")
    return execCtx, nil
})

// ブランチC: counter = 3を設定
branchC := workflow.NewStep("branch-c", func(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
    execCtx.SetSessionState("counter", 3)
    execCtx.SetSessionState("branch_c_result", "done")
    return execCtx, nil
})

parallel.AddNode(branchA)
parallel.AddNode(branchB)
parallel.AddNode(branchC)

// 並列ステップを実行
execCtx := workflow.NewExecutionContextWithSession("", "session-123", "user-456")
result, _ := parallel.Execute(context.Background(), execCtx)

// マージされた結果を確認
counter, _ := result.GetSessionState("counter")
fmt.Printf("Counter: %v\n", counter)  // 出力は1、2、または3の可能性（実行順序による）

branchAResult, _ := result.GetSessionState("branch_a_result")
branchBResult, _ := result.GetSessionState("branch_b_result")
branchCResult, _ := result.GetSessionState("branch_c_result")
fmt.Printf("All branches completed: %v, %v, %v\n", branchAResult, branchBResult, branchCResult)
// 出力: All branches completed: done, done, done
```

## API リファレンス

### SessionState型

```go
type SessionState struct {
    mu   sync.RWMutex
    data map[string]interface{}
}
```

#### メソッド

##### NewSessionState()

```go
func NewSessionState() *SessionState
```

新しいSessionStateインスタンスを作成します。

##### Set(key string, value interface{})

```go
func (ss *SessionState) Set(key string, value interface{})
```

キーと値のペアを設定（スレッドセーフ）。

##### Get(key string) (interface{}, bool)

```go
func (ss *SessionState) Get(key string) (interface{}, bool)
```

キーの値を取得（スレッドセーフ）。

##### Clone() *SessionState

```go
func (ss *SessionState) Clone() *SessionState
```

JSONシリアライゼーションを使用してSessionStateをディープコピーします。

##### GetAll() map[string]interface{}

```go
func (ss *SessionState) GetAll() map[string]interface{}
```

すべてのキーと値のペアのコピーを取得（スレッドセーフ）。

## ベストプラクティス

### 1. 大きなデータの保存を避ける

```go
// ❌ 非推奨
execCtx.SetSessionState("all_users", []User{ /* 10000+ users */ })

// ✅ 推奨
execCtx.SetSessionState("user_ids", []string{"id1", "id2", "id3"})
```

**理由**: `Clone()`はJSONシリアライゼーションを使用するため、大きなデータ構造では高コストになります。

### 2. 並列ブランチを賢く使用

```go
// ✅ 並列ブランチは異なるデータを独立して処理
// ブランチA: ユーザーデータを処理
// ブランチB: 注文データを処理
// ブランチC: ログデータを処理

// ⚠️ 並列ブランチが同じキーを変更することを避ける
// （last-write-wins戦略を理解している場合を除く）
```

## トラブルシューティング

### よくある問題

#### 1. SessionStateがnil

**症状**:
```go
panic: runtime error: invalid memory address or nil pointer dereference
```

**原因**: SessionStateが初期化されていません

**解決策**:
```go
// ❌ 間違い
execCtx := &workflow.ExecutionContext{}
execCtx.SetSessionState("key", "value")  // panic!

// ✅ 正しい
execCtx := workflow.NewExecutionContextWithSession("", "session-id", "user-id")
execCtx.SetSessionState("key", "value")  // OK
```

#### 2. 型アサーション失敗

**症状**:
```go
panic: interface conversion: interface {} is string, not int
```

**解決策**:
```go
// ✅ 正しい
execCtx.SetSessionState("age", 30)  // intを保存
raw, ok := execCtx.GetSessionState("age")
if !ok {
    // キーが存在しない
}
age, ok := raw.(int)
if !ok {
    // 型が一致しない
}
```

## テスト

完全なテストカバレッジには以下が含まれます：

- ✅ 基本的なGet/Set操作
- ✅ ディープコピー（Clone）
- ✅ 状態マージ（Merge）
- ✅ 並行安全性（1000 goroutines）
- ✅ ワークフロー統合テスト
- ✅ 並列ブランチ分離

**テストカバレッジ**: 543行のテストコード

テストを実行：
```bash
cd pkg/hno/workflow
go test -v -run TestSessionState
```

## 関連ドキュメント

- [ワークフローガイド](/ja/guide/workflow) - ワークフローエンジンの使用
- [チームガイド](/ja/guide/team) - マルチエージェント協力
- [メモリ管理](/ja/guide/memory) - 会話メモリ

---

**最終更新**: 2025-01-XX
