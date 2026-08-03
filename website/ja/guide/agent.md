# Agent

**Agent**は、ツールを使用し、会話コンテキストを維持し、タスクを独立して実行できる自律型AIエンティティです。

## 概要

```go
import "github.com/rexleimo/HNO/pkg/hno/agent"

agent, err := agent.New(agent.Config{
    Name:         "My Agent",
    Model:        model,
    Toolkits:     []toolkit.Toolkit{calculator.New()},
    Instructions: "You are a helpful assistant",
    MaxLoops:     10,
})

output, err := agent.Run(context.Background(), "What is 2+2?")
```

## 設定

### Config構造体

```go
type Config struct {
    Name         string            // Agentの名前
    Model        models.Model      // LLMモデル
    Toolkits     []toolkit.Toolkit // 利用可能なツール
    Memory       memory.Memory     // 会話メモリ
    Instructions string            // システム指示
    MaxLoops     int               // 最大ツール呼び出しループ数（デフォルト: 10）
    PreHooks     []hooks.Hook      // 実行前フック
    PostHooks    []hooks.Hook      // 実行後フック
}
```

### パラメータ

- **Name**（必須）: 人間が読める形式のAgent識別子
- **Model**（必須）: LLMモデルインスタンス（OpenAI、Claude等）
- **Toolkits**（オプション）: 利用可能なツールのリスト
- **Memory**（オプション）: デフォルトは100メッセージ制限のメモリ内ストレージ
- **Instructions**（オプション）: システムプロンプト/ペルソナ
- **MaxLoops**（オプション）: 無限ツール呼び出しループを防止（デフォルト: 10）
- **PreHooks**（オプション）: 実行前の検証フック
- **PostHooks**（オプション）: 実行後の検証フック

## 基本的な使い方

### シンプルなAgent

```go
package main

import (
    "context"
    "fmt"
    "github.com/rexleimo/HNO/pkg/hno/agent"
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    ag, _ := agent.New(agent.Config{
        Name:         "Assistant",
        Model:        model,
        Instructions: "You are a helpful assistant",
    })

    output, _ := ag.Run(context.Background(), "Hello!")
    fmt.Println(output.Content)
}
```

### ツール付きAgent

```go
import (
    "github.com/rexleimo/HNO/pkg/hno/tools/calculator"
    "github.com/rexleimo/HNO/pkg/hno/tools/http"
)

ag, _ := agent.New(agent.Config{
    Name:  "Smart Assistant",
    Model: model,
    Toolkits: []toolkit.Toolkit{
        calculator.New(),
        http.New(),
    },
    Instructions: "You can do math and make HTTP requests",
})

output, _ := ag.Run(ctx, "Calculate 15 * 23 and fetch https://api.github.com")
```

## 高度な機能

### カスタムメモリ

```go
import "github.com/rexleimo/HNO/pkg/hno/memory"

// カスタム制限でメモリを作成
mem := memory.NewInMemory(50) // 最新50メッセージを保持

ag, _ := agent.New(agent.Config{
    Memory: mem,
    // ... その他の設定
})
```

### フック＆ガードレール

フックで入力と出力を検証:

```go
import "github.com/rexleimo/HNO/pkg/hno/guardrails"

// 組み込みのプロンプトインジェクションガード
promptGuard := guardrails.NewPromptInjectionGuardrail()

// カスタム検証フック
customHook := func(ctx context.Context, input *hooks.HookInput) error {
    if len(input.Input) > 1000 {
        return fmt.Errorf("input too long")
    }
    return nil
}

ag, _ := agent.New(agent.Config{
    PreHooks:  []hooks.Hook{customHook, promptGuard},
    PostHooks: []hooks.Hook{outputValidator},
    // ... その他の設定
})
```

### コンテキストとタイムアウト

```go
import "time"

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

output, err := ag.Run(ctx, "Complex task...")
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        fmt.Println("Timeout!")
    }
}
```

### レスポンスキャッシュ (v1.2.6)

キャッシュを有効化するとモデル出力を再利用でき、決定的なレスポンスを得られます:

```go
ag, _ := agent.New(agent.Config{
    Model:       model,
    EnableCache: true,
    CacheTTL:    2 * time.Minute,
})

first, _ := ag.Run(ctx, "Summarise REST vs gRPC")
second, _ := ag.Run(ctx, "Summarise REST vs gRPC")

if cached, _ := second.Metadata["cache_hit"].(bool); cached {
    // Handle cached response
}
```

共有ストレージや Redis を使う場合は `cache.Provider` を差し替えてください。デフォルトではインメモリ LRU が利用されます。

## 実行出力

`Run`メソッドは`*RunOutput`を返します:

```go
type RunOutput struct {
    Content  string                 // Agentの応答
    Messages []types.Message        // 完全なメッセージ履歴
    Metadata map[string]interface{} // 追加データ
}
```

例:

```go
output, err := ag.Run(ctx, "Tell me a joke")
if err != nil {
    log.Fatal(err)
}

fmt.Println("Response:", output.Content)
fmt.Println("Messages:", len(output.Messages))
fmt.Println("Metadata:", output.Metadata)
```

## メモリ管理

### メモリをクリア

```go
// すべての会話履歴をクリア
ag.ClearMemory()
```

### メモリにアクセス

```go
// 現在のメッセージを取得
messages := ag.GetMemory().GetMessages()
fmt.Println("History:", len(messages))
```

## エラー処理

```go
output, err := ag.Run(ctx, input)
if err != nil {
    switch {
    case errors.Is(err, types.ErrInvalidInput):
        // 無効な入力を処理
    case errors.Is(err, types.ErrRateLimit):
        // レート制限を処理
    case errors.Is(err, context.DeadlineExceeded):
        // タイムアウトを処理
    default:
        // その他のエラーを処理
    }
}
```

## ベストプラクティス

### 1. 常にContextを使用

```go
ctx := context.Background()
output, err := ag.Run(ctx, input)
```

### 2. 適切なMaxLoopsを設定

```go
// シンプルなタスクの場合
MaxLoops: 5

// 複雑な推論の場合
MaxLoops: 15
```

### 3. 明確な指示を提供

```go
Instructions: `You are a customer support agent.
- Be polite and professional
- Use tools to look up information
- If unsure, ask for clarification`
```

### 4. 型安全なツール設定を使用

```go
calc := calculator.New()
httpClient := http.New(http.Config{
    Timeout: 10 * time.Second,
})

ag, _ := agent.New(agent.Config{
    Toolkits: []toolkit.Toolkit{calc, httpClient},
})
```

## パフォーマンスに関する考慮事項

- **Agent作成**: 平均約180ns
- **メモリフットプリント**: Agent1つあたり約1.2KB
- **並行Agent**: 完全にスレッドセーフ、自由にgoroutineを使用可能

```go
// 並行Agent
for i := 0; i < 100; i++ {
    go func(id int) {
        ag, _ := agent.New(config)
        output, _ := ag.Run(ctx, input)
        fmt.Printf("Agent %d: %s\n", id, output.Content)
    }(i)
}
```

## 例

実際の例を参照:

- [Simple Agent](https://github.com/rexleimo/HNO/tree/main/cmd/examples/simple_agent)
- [Claude Agent](https://github.com/rexleimo/HNO/tree/main/cmd/examples/claude_agent)
- [Ollama Agent](https://github.com/rexleimo/HNO/tree/main/cmd/examples/ollama_agent)
- [Agent with Guardrails](https://github.com/rexleimo/HNO/tree/main/cmd/examples/agent_with_guardrails)

## APIリファレンス

完全なAPIドキュメントは[Agent APIリファレンス](/api/agent)を参照してください。

## 次のステップ

- [Team](/guide/team) - マルチエージェントコラボレーション
- [Workflow](/guide/workflow) - ステップベースのオーケストレーション
- [Tools](/guide/tools) - 組み込みツールとカスタムツール
- [Models](/guide/models) - LLMプロバイダー設定
