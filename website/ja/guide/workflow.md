# Workflow - ステップベースのオーケストレーション

5つの強力なプリミティブで複雑で制御された多段階プロセスを構築します。

---

## Workflowとは？

**Workflow**は、制御されたAI Agentプロセスを構築するための決定論的でステップベースのオーケストレーションを提供します。Team（自律型）とは異なり、Workflowは実行フローを完全に制御できます。

### 主な機能

- **5つのプリミティブ**: Step、Condition、Loop、Parallel、Router
- **決定論的実行**: 予測可能で再現性のあるフロー
- **完全な制御**: 実行パスを決定
- **コンテキスト共有**: ステップ間でデータを渡す
- **組み合わせ可能**: プリミティブをネストして複雑なロジックを構築

---

## Workflowの作成

### 基本的な例

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/workflow"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: "your-api-key",
    })

    // Agentを作成
    fetchAgent, _ := agent.New(agent.Config{
        Name:  "Fetcher",
        Model: model,
        Instructions: "Fetch data about the topic.",
    })

    processAgent, _ := agent.New(agent.Config{
        Name:  "Processor",
        Model: model,
        Instructions: "Process and analyze the data.",
    })

    // ワークフローステップを作成
    step1, _ := workflow.NewStep(workflow.StepConfig{
        ID:    "fetch",
        Agent: fetchAgent,
    })

    step2, _ := workflow.NewStep(workflow.StepConfig{
        ID:    "process",
        Agent: processAgent,
    })

    // ワークフローを作成
    wf, _ := workflow.New(workflow.Config{
        Name:  "Data Pipeline",
        Steps: []workflow.Node{step1, step2},
    })

    // ワークフローを実行
    result, _ := wf.Run(context.Background(), "AI trends")
    fmt.Println(result.Output)
}
```

---

## Workflowプリミティブ

### 1. Step

Agentまたはカスタム関数を実行します。

```go
// Agentを使用
step, _ := workflow.NewStep(workflow.StepConfig{
    ID:    "analyze",
    Agent: analyzerAgent,
})

// カスタム関数を使用
step, _ := workflow.NewStep(workflow.StepConfig{
    ID: "transform",
    Function: func(ctx *workflow.ExecutionContext) (*workflow.StepOutput, error) {
        input := ctx.Input
        // カスタムロジック
        return &workflow.StepOutput{Output: transformed}, nil
    },
})
```

**ユースケース:**
- Agent実行
- カスタムデータ変換
- API呼び出し
- データ検証

---

### 2. Condition

コンテキストに基づく条件分岐。

```go
condition, _ := workflow.NewCondition(workflow.ConditionConfig{
    ID: "check_sentiment",
    Condition: func(ctx *workflow.ExecutionContext) bool {
        result := ctx.GetStepOutput("classify")
        return strings.Contains(result.Output, "positive")
    },
    ThenStep: positiveHandlerStep,
    ElseStep: negativeHandlerStep,
})
```

**ユースケース:**
- センチメントルーティング
- 品質チェック
- エラー処理
- A/Bテストロジック

---

### 3. Loop

終了条件付きの反復実行。

```go
loop, _ := workflow.NewLoop(workflow.LoopConfig{
    ID: "retry",
    Condition: func(ctx *workflow.ExecutionContext) bool {
        result := ctx.GetStepOutput("attempt")
        return result == nil || strings.Contains(result.Output, "error")
    },
    Body:          retryStep,
    MaxIterations: 5,
})
```

**ユースケース:**
- リトライロジック
- 反復改善
- データ処理ループ
- 段階的改善

---

### 4. Parallel

複数のステップを並行実行。

```go
parallel, _ := workflow.NewParallel(workflow.ParallelConfig{
    ID: "gather_data",
    Steps: []workflow.Node{
        sourceStep1,
        sourceStep2,
        sourceStep3,
    },
})
```

**ユースケース:**
- 並列データ収集
- 複数ソースの集約
- 独立した計算
- パフォーマンス最適化

---

### 5. Router

異なるブランチへの動的ルーティング。

```go
router, _ := workflow.NewRouter(workflow.RouterConfig{
    ID: "route_by_type",
    Routes: map[string]workflow.Node{
        "email": emailHandlerStep,
        "chat":  chatHandlerStep,
        "phone": phoneHandlerStep,
    },
    Selector: func(ctx *workflow.ExecutionContext) string {
        if strings.Contains(ctx.Input, "@") {
            return "email"
        }
        return "chat"
    },
})
```

**ユースケース:**
- 入力タイプルーティング
- 言語検出
- 優先度処理
- 動的ディスパッチ

---

## 実行コンテキスト

ExecutionContextはワークフロー状態へのアクセスを提供します。

### メソッド

```go
type ExecutionContext struct {
    Input string  // ワークフロー入力
}

// 前のステップの出力を取得
func (ctx *ExecutionContext) GetStepOutput(stepID string) *StepOutput

// カスタムデータを保存
func (ctx *ExecutionContext) SetData(key string, value interface{})

// カスタムデータを取得
func (ctx *ExecutionContext) GetData(key string) interface{}
```

### 例

```go
step := workflow.NewStep(workflow.StepConfig{
    ID: "use_context",
    Function: func(ctx *workflow.ExecutionContext) (*workflow.StepOutput, error) {
        // 前のステップにアクセス
        previous := ctx.GetStepOutput("classify")

        // 共有データにアクセス
        userData := ctx.GetData("user_info")

        // 処理して返す
        result := processData(previous.Output, userData)
        return &workflow.StepOutput{Output: result}, nil
    },
})
```

---

## 完全な例

### 条件分岐Workflow

センチメントベースのルーティング。

```go
func buildSentimentWorkflow(apiKey string) *workflow.Workflow {
    model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

    classifier, _ := agent.New(agent.Config{
        Name:  "Classifier",
        Model: model,
        Instructions: "Classify sentiment as 'positive' or 'negative'.",
    })

    positiveHandler, _ := agent.New(agent.Config{
        Name:  "PositiveHandler",
        Model: model,
        Instructions: "Thank the user warmly.",
    })

    negativeHandler, _ := agent.New(agent.Config{
        Name:  "NegativeHandler",
        Model: model,
        Instructions: "Apologize and offer help.",
    })

    classifyStep, _ := workflow.NewStep(workflow.StepConfig{
        ID:    "classify",
        Agent: classifier,
    })

    positiveStep, _ := workflow.NewStep(workflow.StepConfig{
        ID:    "positive",
        Agent: positiveHandler,
    })

    negativeStep, _ := workflow.NewStep(workflow.StepConfig{
        ID:    "negative",
        Agent: negativeHandler,
    })

    condition, _ := workflow.NewCondition(workflow.ConditionConfig{
        ID: "route",
        Condition: func(ctx *workflow.ExecutionContext) bool {
            result := ctx.GetStepOutput("classify")
            return strings.Contains(result.Output, "positive")
        },
        ThenStep: positiveStep,
        ElseStep: negativeStep,
    })

    wf, _ := workflow.New(workflow.Config{
        Name:  "Sentiment Router",
        Steps: []workflow.Node{classifyStep, condition},
    })

    return wf
}
```

### ループWorkflow

品質改善を伴うリトライ。

```go
func buildRetryWorkflow(apiKey string) *workflow.Workflow {
    model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

    generator, _ := agent.New(agent.Config{
        Name:  "Generator",
        Model: model,
        Instructions: "Generate creative content.",
    })

    validator, _ := agent.New(agent.Config{
        Name:  "Validator",
        Model: model,
        Instructions: "Check if content meets quality standards. Return 'pass' or 'fail'.",
    })

    generateStep, _ := workflow.NewStep(workflow.StepConfig{
        ID:    "generate",
        Agent: generator,
    })

    validateStep, _ := workflow.NewStep(workflow.StepConfig{
        ID:    "validate",
        Agent: validator,
    })

    loop, _ := workflow.NewLoop(workflow.LoopConfig{
        ID: "improve",
        Condition: func(ctx *workflow.ExecutionContext) bool {
            result := ctx.GetStepOutput("validate")
            return result != nil && strings.Contains(result.Output, "fail")
        },
        Body:          generateStep,
        MaxIterations: 3,
    })

    wf, _ := workflow.New(workflow.Config{
        Name:  "Quality Loop",
        Steps: []workflow.Node{generateStep, validateStep, loop},
    })

    return wf
}
```

---

## ベストプラクティス

### 1. 明確なステップID

デバッグのために説明的なステップIDを使用:

```go
// 良い例 ✅
ID: "fetch_user_data"

// 悪い例 ❌
ID: "step1"
```

### 2. エラー処理

各ステップでエラーを処理:

```go
result, err := wf.Run(ctx, input)
if err != nil {
    log.Printf("Workflow failed at step: %v", err)
    return
}
```

### 3. コンテキストタイムアウト

適切なタイムアウトを設定:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

result, _ := wf.Run(ctx, input)
```

### 4. ループ制限

無限ループを防ぐため、常にMaxIterationsを設定:

```go
loop, _ := workflow.NewLoop(workflow.LoopConfig{
    // ...
    MaxIterations: 10,  // 常に制限を設定
})
```


## 高度な Workflow 機能

### チェックポイントから再開

中断されたワークフローを最後に成功したステップから再開します:

```go
execCtx, err := wf.Run(ctx, input, "session-id",
    workflow.WithResumeFrom("validate-output"),
)
if err != nil {
    log.Fatal(err)
}
```

`EnableHistory` を有効にすると各実行のステップ出力とキャンセル記録が永続化されます。`WithResumeFrom(stepID)` を渡すと、完了済みステップをスキップして指定したチェックポイントから続行します。

### キャンセルの永続化

キャンセル理由や Run ID、タイムスタンプを記録するには履歴を有効化します:

```go
store := workflow.NewMemoryStorage(100)
wf, _ := workflow.New(workflow.Config{
    Name:          "retriable-pipeline",
    Steps:         []workflow.Node{firstStep, finalStep},
    EnableHistory: true,
    HistoryStore:  store,
})
```

ステップがコンテキストをキャンセルした場合、最新の履歴エントリには `RunStatusCancelled` と `CancellationReason` が保存されます。`store.GetSession(ctx, sessionID)` で内容を確認できます。

### メディアペイロード

プロンプトへ埋め込まずに画像・音声・ファイルを添付できます:

```go
// import "github.com/rexleimo/agno-go/pkg/hno/media"
attachments := []media.Attachment{
    {Type: "image", URL: "https://example.com/diagram.png"},
    {Type: "file",  ID:  "spec-v1", Name: "spec.pdf"},
}

execCtx, _ := wf.Run(ctx, "review this", "sess-media",
    workflow.WithMediaPayload(attachments),
)

payload, _ := execCtx.GetSessionState("media_payload")
```

ワークフローは検証済みの添付ファイルをセッション状態 (`media_payload`) に保存し、後続ステップや AgentOS 連携からアクセスできるようにします。

---

## Workflow vs Team

それぞれをいつ使用するか:

### Workflowを使用する場合:
- 決定論的実行が必要
- 特定の順序でステップを実行する必要がある
- きめ細かい制御が必要
- デバッグとテストが重要

### Teamを使用する場合:
- Agentが自律的に動作すべき
- 順序が重要でない（parallel/consensus）
- 創発的な動作が必要
- 制御よりも柔軟性を優先

---

## パフォーマンスのヒント

### 並列実行

独立したステップにはParallelを使用:

```go
// Sequential: 合計3秒
steps := []workflow.Node{step1, step2, step3} // 1s + 1s + 1s

// Parallel: 合計1秒
parallel, _ := workflow.NewParallel(workflow.ParallelConfig{
    Steps: []workflow.Node{step1, step2, step3}, // max(1s, 1s, 1s)
})
```

### コンテキストの再利用

コンテキストデータを再利用して冗長なLLM呼び出しを回避:

```go
func (ctx *ExecutionContext) {
    // 高コストな計算をキャッシュ
    if ctx.GetData("cached") == nil {
        expensive := computeExpensiveData()
        ctx.SetData("cached", expensive)
    }
}
```

---

## 次のステップ

- 自律型Agentについては[Team](/guide/team)と比較
- ワークフローAgentに[Tools](/guide/tools)を追加
- さまざまなLLMプロバイダーについては[Models](/guide/models)を参照
- 詳細なドキュメントは[Workflow APIリファレンス](/api/workflow)を確認

---

## 関連例

- [Workflow Demo](/examples/workflow-demo) - 完全な例
- [条件分岐ルーティング](/examples/workflow-demo#conditional)
- [リトライループパターン](/examples/workflow-demo#loop)
