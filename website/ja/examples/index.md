# サンプル

HNOのすべての機能を実演する実用的なサンプル。

## 利用可能なサンプル

### 1. Simple Agent

計算ツールを持つ基本的なエージェント。

**場所**: `cmd/examples/simple_agent/`

**機能**:
- OpenAI GPT-4o-mini統合
- 計算ツールキット
- 基本的な会話

**実行**:
```bash
export OPENAI_API_KEY=sk-your-key
go run cmd/examples/simple_agent/main.go
```

[ソースを表示](https://github.com/rexleimo/agno-go/tree/main/cmd/examples/simple_agent)

---

### 2. Claude Agent

ツールを使用したAnthropic Claude統合。

**場所**: `cmd/examples/claude_agent/`

**機能**:
- Anthropic Claude 3.5 Sonnet
- HTTPと計算ツール
- エラーハンドリングの例

**実行**:
```bash
export ANTHROPIC_API_KEY=sk-ant-your-key
go run cmd/examples/claude_agent/main.go
```

[ソースを表示](https://github.com/rexleimo/agno-go/tree/main/cmd/examples/claude_agent)

---

### 3. Ollama Agent

Ollamaによるローカルモデルサポート。

**場所**: `cmd/examples/ollama_agent/`

**機能**:
- ローカルLlama 3モデル
- プライバシー重視（API呼び出しなし）
- ファイル操作ツールキット

**セットアップ**:
```bash
# Ollamaをインストール
curl -fsSL https://ollama.com/install.sh | sh

# モデルをプル
ollama pull llama3

# サンプルを実行
go run cmd/examples/ollama_agent/main.go
```

[ソースを表示](https://github.com/rexleimo/agno-go/tree/main/cmd/examples/ollama_agent)

---

### 4. Team Demo

異なる協調モードでのマルチエージェント協調。

**場所**: `cmd/examples/team_demo/`

**機能**:
- 4つの協調モード（Sequential、Parallel、Leader-Follower、Consensus）
- 研究者 + ライターチーム
- 実世界のワークフロー

**実行**:
```bash
export OPENAI_API_KEY=sk-your-key
go run cmd/examples/team_demo/main.go
```

[ソースを表示](https://github.com/rexleimo/agno-go/tree/main/cmd/examples/team_demo)

---

### 5. Workflow Demo

制御フロープリミティブを使用したステップベースのオーケストレーション。

**場所**: `cmd/examples/workflow_demo/`

**機能**:
- 5つのワークフロープリミティブ（Step、Condition、Loop、Parallel、Router）
- 感情分析ワークフロー
- 条件付きルーティング

**実行**:
```bash
export OPENAI_API_KEY=sk-your-key
go run cmd/examples/workflow_demo/main.go
```

[ソースを表示](https://github.com/rexleimo/agno-go/tree/main/cmd/examples/workflow_demo)

---

### 6. RAG Demo

ChromaDBを使用したRetrieval-Augmented Generation。

**場所**: `cmd/examples/rag_demo/`

**機能**:
- ChromaDBベクトルデータベース
- OpenAI埋め込み
- セマンティック検索
- ドキュメントQ&A

**セットアップ**:
```bash
# ChromaDBを開始（Docker）
docker run -d -p 8000:8000 chromadb/chroma

# APIキーを設定
export OPENAI_API_KEY=sk-your-key

# サンプルを実行
go run cmd/examples/rag_demo/main.go
```

[ソースを表示](https://github.com/rexleimo/agno-go/tree/main/cmd/examples/rag_demo)

---

## コードスニペット

### 複数ツールを持つエージェント

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
    "github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
    "github.com/rexleimo/agno-go/pkg/hno/tools/http"
    "github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    ag, _ := agent.New(agent.Config{
        Name:  "Smart Assistant",
        Model: model,
        Toolkits: []toolkit.Toolkit{
            calculator.New(),
            http.New(),
        },
        Instructions: "You can do math and make HTTP requests",
    })

    output, _ := ag.Run(context.Background(),
        "Calculate 15 * 23 and fetch https://api.github.com")
    fmt.Println(output.Content)
}
```

### マルチエージェントチーム

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
    "github.com/rexleimo/agno-go/pkg/hno/team"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    researcher, _ := agent.New(agent.Config{
        Name:         "Researcher",
        Model:        model,
        Instructions: "Research and gather information",
    })

    writer, _ := agent.New(agent.Config{
        Name:         "Writer",
        Model:        model,
        Instructions: "Create compelling content",
    })

    tm, _ := team.New(team.Config{
        Name:   "Content Team",
        Agents: []*agent.Agent{researcher, writer},
        Mode:   team.ModeSequential,
    })

    output, _ := tm.Run(context.Background(),
        "Write a short article about Go programming")
    fmt.Println(output.Content)
}
```

### 条件付きワークフロー

```go
package main

import (
    "context"
    "fmt"
    "os"
    "strings"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
    "github.com/rexleimo/agno-go/pkg/hno/workflow"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    classifier, _ := agent.New(agent.Config{
        Name:         "Classifier",
        Model:        model,
        Instructions: "Classify sentiment as positive or negative",
    })

    positiveHandler, _ := agent.New(agent.Config{
        Name:         "Positive Handler",
        Model:        model,
        Instructions: "Respond enthusiastically",
    })

    negativeHandler, _ := agent.New(agent.Config{
        Name:         "Negative Handler",
        Model:        model,
        Instructions: "Respond empathetically",
    })

    wf, _ := workflow.New(workflow.Config{
        Name: "Sentiment Workflow",
        Steps: []workflow.Primitive{
            workflow.NewStep("classify", classifier),
            workflow.NewCondition("route",
                func(ctx *workflow.ExecutionContext) bool {
                    result := ctx.GetResult("classify")
                    return strings.Contains(result.Content, "positive")
                },
                workflow.NewStep("positive", positiveHandler),
                workflow.NewStep("negative", negativeHandler),
            ),
        },
    })

    output, _ := wf.Run(context.Background(), "I love this!")
    fmt.Println(output.Content)
}
```

## 詳細を学ぶ

- [クイックスタート](/guide/quick-start) - 5分で始める
- [エージェントガイド](/guide/agent) - エージェントについて学ぶ
- [チームガイド](/guide/team) - マルチエージェント協調
- [ワークフローガイド](/guide/workflow) - オーケストレーションパターン
- [APIリファレンス](/api/) - 完全なAPIドキュメント

## サンプルを提供

興味深いサンプルがありますか？リポジトリに貢献してください:

1. リポジトリをフォーク
2. `cmd/examples/your_example/`にサンプルを作成
3. 説明と使用方法を含むREADME.mdを追加
4. プルリクエストを提出

[コントリビューションガイドライン](https://github.com/rexleimo/agno-go/blob/main/CONTRIBUTING.md)
