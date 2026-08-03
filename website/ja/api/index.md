# APIリファレンス

HNO v1.0の完全なAPIリファレンスです。

## コアモジュール

- [Agent](/api/agent) - 自律型AIエージェント
- [Team](/api/team) - マルチエージェント協調
- [Workflow](/api/workflow) - ステップベースのオーケストレーション
- [Models](/api/models) - LLMプロバイダー統合
- [Tools](/api/tools) - 組み込みツールとカスタムツール
- [Memory](/api/memory) - 会話履歴管理
- [Types](/api/types) - コア型とエラー
- [AgentOS Server](/api/agentos) - 本番環境向けHTTPサーバー

## クイックリンク

### Agent

```go
import "github.com/rexleimo/HNO/pkg/hno/agent"

agent.New(config) (*Agent, error)
agent.Run(ctx, input) (*RunOutput, error)
agent.ClearMemory()
```

[完全なAgent APIドキュメント →](/api/agent)

### Team

```go
import "github.com/rexleimo/HNO/pkg/hno/team"

team.New(config) (*Team, error)
team.Run(ctx, input) (*RunOutput, error)

// モード: Sequential, Parallel, LeaderFollower, Consensus
```

[完全なTeam APIドキュメント →](/api/team)

### Workflow

```go
import "github.com/rexleimo/HNO/pkg/hno/workflow"

workflow.New(config) (*Workflow, error)
workflow.Run(ctx, input) (*RunOutput, error)

// プリミティブ: Step, Condition, Loop, Parallel, Router
```

[完全なWorkflow APIドキュメント →](/api/workflow)

### Models

```go
import (
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
    "github.com/rexleimo/HNO/pkg/hno/models/anthropic"
    "github.com/rexleimo/HNO/pkg/hno/models/ollama"
)

openai.New(modelID, config) (*OpenAI, error)
anthropic.New(modelID, config) (*Anthropic, error)
ollama.New(modelID, config) (*Ollama, error)
```

[完全なModels APIドキュメント →](/api/models)

### Tools

```go
import (
    "github.com/rexleimo/HNO/pkg/hno/tools/calculator"
    "github.com/rexleimo/HNO/pkg/hno/tools/http"
    "github.com/rexleimo/HNO/pkg/hno/tools/file"
)

calculator.New() *Calculator
http.New(config) *HTTP
file.New(config) *File
```

[完全なTools APIドキュメント →](/api/tools)

## 一般的なパターン

### エラーハンドリング

```go
import "github.com/rexleimo/HNO/pkg/hno/types"

output, err := agent.Run(ctx, input)
if err != nil {
    switch {
    case errors.Is(err, types.ErrInvalidInput):
        // 無効な入力を処理
    case errors.Is(err, types.ErrRateLimit):
        // レート制限を処理
    default:
        // その他のエラーを処理
    }
}
```

### Context管理

```go
import (
    "context"
    "time"
)

// タイムアウト付き
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

output, err := agent.Run(ctx, input)
```

### 並行エージェント

```go
var wg sync.WaitGroup

for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()

        ag, _ := agent.New(config)
        output, _ := ag.Run(ctx, input)

        fmt.Printf("Agent %d: %s\n", id, output.Content)
    }(i)
}

wg.Wait()
```

## 型定義

### コア型

```go
// メッセージ型
type Message struct {
    Role    MessageRole
    Content string
    Name    string
}

// 実行出力
type RunOutput struct {
    Content  string
    Messages []Message
    Metadata map[string]interface{}
}

// モデルレスポンス
type ModelResponse struct {
    Content    string
    ToolCalls  []ToolCall
    FinishReason string
}
```

[完全な型リファレンス →](/api/types)

## AgentOS Server API

本番環境デプロイ用のREST APIエンドポイント:

```bash
# ヘルスチェック
GET /health

# エージェント一覧
GET /api/v1/agents

# エージェント実行
POST /api/v1/agents/{agent_id}/run

# セッション作成
POST /api/v1/sessions

# セッション取得
GET /api/v1/sessions/{session_id}

# セッションを共有 (エージェント/チーム間)
POST /api/v1/sessions/{session_id}/reuse

# サマリー生成 (同期/非同期)
POST /api/v1/sessions/{session_id}/summary?async=true|false

# サマリースナップショット取得
GET /api/v1/sessions/{session_id}/summary

# 履歴取得 (num_messages, stream_events フィルター)
GET /api/v1/sessions/{session_id}/history
```

[完全なAgentOS APIドキュメント →](/api/agentos)

## OpenAPI仕様

完全なOpenAPI 3.0仕様が利用可能です:

- [OpenAPI YAML](https://github.com/rexleimo/HNO/blob/main/pkg/agentos/openapi.yaml)
- [Swagger UI](https://github.com/rexleimo/HNO/tree/main/pkg/agentos#api-documentation)

## サンプル

リポジトリ内の動作サンプル:

- [Simple Agent](https://github.com/rexleimo/HNO/tree/main/cmd/examples/simple_agent)
- [Team Demo](https://github.com/rexleimo/HNO/tree/main/cmd/examples/team_demo)
- [Workflow Demo](https://github.com/rexleimo/HNO/tree/main/cmd/examples/workflow_demo)
- [RAG Demo](https://github.com/rexleimo/HNO/tree/main/cmd/examples/rag_demo)

## パッケージドキュメント

完全なGoパッケージドキュメントはpkg.go.devで確認できます:

[pkg.go.dev/github.com/rexleimo/HNO](https://pkg.go.dev/github.com/rexleimo/HNO)
