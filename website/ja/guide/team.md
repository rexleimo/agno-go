# Team - マルチエージェントコラボレーション

4つの協調モードで強力なマルチエージェントシステムを構築します。

---

## Teamとは？

**Team**は、複雑なタスクを解決するために協力するAgentのコレクションです。異なる協調モードにより、様々なコラボレーションパターンが可能になります。

### 主な機能

- **4つの協調モード**: Sequential、Parallel、Leader-Follower、Consensus
- **動的メンバーシップ**: 実行時にAgentを追加/削除
- **柔軟な設定**: モードごとに動作をカスタマイズ
- **型安全**: 完全なGo型チェック

---

## Teamの作成

### 基本的な例

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/team"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
)

func main() {
    // モデルを作成
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: "your-api-key",
    })

    // チームメンバーを作成
    researcher, _ := agent.New(agent.Config{
        Name:         "Researcher",
        Model:        model,
        Instructions: "You are a research expert.",
    })

    writer, _ := agent.New(agent.Config{
        Name:         "Writer",
        Model:        model,
        Instructions: "You are a technical writer.",
    })

    // チームを作成
    t, err := team.New(team.Config{
        Name:   "Content Team",
        Agents: []*agent.Agent{researcher, writer},
        Mode:   team.ModeSequential,
    })
    if err != nil {
        log.Fatal(err)
    }

    // チームを実行
    output, _ := t.Run(context.Background(), "Write about AI")
    fmt.Println(output.Content)
}
```

---

## 協調モード

### 1. Sequential Mode

Agentが順番に実行され、出力が次のAgentに渡されます。

```go
t, _ := team.New(team.Config{
    Name:   "Pipeline",
    Agents: []*agent.Agent{agent1, agent2, agent3},
    Mode:   team.ModeSequential,
})
```

**ユースケース:**
- コンテンツパイプライン（リサーチ → 執筆 → 編集）
- データ処理ワークフロー
- 多段階推論

**動作の仕組み:**
1. Agent 1が入力を処理 → 出力A
2. Agent 2が出力Aを処理 → 出力B
3. Agent 3が出力Bを処理 → 最終出力

---

### 2. Parallel Mode

すべてのAgentが同時に実行され、結果が統合されます。

```go
t, _ := team.New(team.Config{
    Name:   "Multi-Perspective",
    Agents: []*agent.Agent{techAgent, bizAgent, ethicsAgent},
    Mode:   team.ModeParallel,
})
```

**ユースケース:**
- 多角的分析
- 並列データ処理
- 多様な意見の生成

**動作の仕組み:**
1. すべてのAgentが同じ入力を受け取る
2. 並行実行（Goのgoroutine）
3. 結果を単一の出力に統合

---

### 3. Leader-Follower Mode

リーダーがフォロワーにタスクを委任し、結果を統合します。

```go
t, _ := team.New(team.Config{
    Name:   "Project Team",
    Leader: leaderAgent,
    Agents: []*agent.Agent{follower1, follower2},
    Mode:   team.ModeLeaderFollower,
})
```

**ユースケース:**
- タスクの委任
- 階層的ワークフロー
- エキスパートへの相談

**動作の仕組み:**
1. リーダーがタスクを分析してサブタスクを作成
2. 適切なフォロワーに委任
3. フォロワーの出力を最終結果に統合

---

### 4. Consensus Mode

Agentが合意に達するまで議論します。

```go
t, _ := team.New(team.Config{
    Name:      "Decision Team",
    Agents:    []*agent.Agent{optimist, realist, critic},
    Mode:      team.ModeConsensus,
    MaxRounds: 3,  // 最大議論ラウンド数
})
```

**ユースケース:**
- 意思決定
- 品質保証
- 議論と改善

**動作の仕組み:**
1. すべてのAgentが初期意見を提供
2. Agentが他の意見をレビュー
3. 合意または最大ラウンドまで繰り返し
4. 最終的な合意出力

---

## 設定

### Config構造体

```go
type Config struct {
    // 必須
    Agents []*agent.Agent  // チームメンバー

    // オプション
    Name      string              // チーム名（デフォルト: "Team"）
    Mode      CoordinationMode    // 協調モード（デフォルト: Sequential）
    Leader    *agent.Agent        // リーダー（LeaderFollowerモード用）
    MaxRounds int                 // 最大ラウンド数（Consensusモード用、デフォルト: 3）
}
```

### 協調モード

```go
const (
    ModeSequential     CoordinationMode = "sequential"
    ModeParallel       CoordinationMode = "parallel"
    ModeLeaderFollower CoordinationMode = "leader-follower"
    ModeConsensus      CoordinationMode = "consensus"
)
```

---

## APIリファレンス

### team.New

新しいチームインスタンスを作成します。

**シグネチャ:**
```go
func New(config Config) (*Team, error)
```

**戻り値:**
- `*Team`: 作成されたチームインスタンス
- `error`: Agentリストが空または設定が無効な場合のエラー

---

### Team.Run

入力でチームを実行します。

**シグネチャ:**
```go
func (t *Team) Run(ctx context.Context, input string) (*RunOutput, error)
```

**パラメータ:**
- `ctx`: キャンセル/タイムアウト用のContext
- `input`: ユーザー入力文字列

**戻り値:**
```go
type RunOutput struct {
    Content      string                 // 最終的なチーム出力
    AgentOutputs []AgentOutput          // 個々のAgent出力
    Metadata     map[string]interface{} // 追加メタデータ
}
```

---

### Team.AddAgent / RemoveAgent

チームメンバーを動的に管理します。

**シグネチャ:**
```go
func (t *Team) AddAgent(ag *agent.Agent)
func (t *Team) RemoveAgent(name string) error
func (t *Team) GetAgents() []*agent.Agent
```

**例:**
```go
// 新しいAgentを追加
t.AddAgent(newAgent)

// 名前でAgentを削除
err := t.RemoveAgent("OldAgent")

// すべてのAgentを取得
agents := t.GetAgents()
```

---

## 完全な例

### Sequential Teamの例

リサーチ → 分析 → 執筆のコンテンツ作成パイプライン。

```go
func createContentPipeline(apiKey string) {
    model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

    researcher, _ := agent.New(agent.Config{
        Name:         "Researcher",
        Model:        model,
        Instructions: "Research the topic and provide key facts.",
    })

    analyst, _ := agent.New(agent.Config{
        Name:         "Analyst",
        Model:        model,
        Instructions: "Analyze research findings and extract insights.",
    })

    writer, _ := agent.New(agent.Config{
        Name:         "Writer",
        Model:        model,
        Instructions: "Write a concise summary based on insights.",
    })

    t, _ := team.New(team.Config{
        Name:   "Content Pipeline",
        Agents: []*agent.Agent{researcher, analyst, writer},
        Mode:   team.ModeSequential,
    })

    output, _ := t.Run(context.Background(),
        "Write about the benefits of AI in healthcare")

    fmt.Println(output.Content)
}
```

### Parallel Teamの例

並行実行による多角的分析。

```go
func multiPerspectiveAnalysis(apiKey string) {
    model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

    techAgent, _ := agent.New(agent.Config{
        Name:         "Tech Specialist",
        Model:        model,
        Instructions: "Focus on technical aspects.",
    })

    bizAgent, _ := agent.New(agent.Config{
        Name:         "Business Specialist",
        Model:        model,
        Instructions: "Focus on business implications.",
    })

    ethicsAgent, _ := agent.New(agent.Config{
        Name:         "Ethics Specialist",
        Model:        model,
        Instructions: "Focus on ethical considerations.",
    })

    t, _ := team.New(team.Config{
        Name:   "Analysis Team",
        Agents: []*agent.Agent{techAgent, bizAgent, ethicsAgent},
        Mode:   team.ModeParallel,
    })

    output, _ := t.Run(context.Background(),
        "Evaluate the impact of autonomous vehicles")

    fmt.Println(output.Content)
}
```

---

## ベストプラクティス

### 1. 適切なモードを選択

- **Sequential**: 出力が前のステップに依存する場合に使用
- **Parallel**: 視点が独立している場合に使用
- **Leader-Follower**: タスクの委任が必要な場合に使用
- **Consensus**: 品質と合意が重要な場合に使用

### 2. Agentの専門化

各Agentに明確で具体的な指示を与えます:

```go
// 良い例 ✅
Instructions: "You are a Python expert. Focus on code quality."

// 悪い例 ❌
Instructions: "You help with coding."
```

### 3. エラー処理

チーム操作からのエラーを常に処理します:

```go
output, err := t.Run(ctx, input)
if err != nil {
    log.Printf("Team execution failed: %v", err)
    return
}
```

### 4. コンテキスト管理

タイムアウトとキャンセルにContextを使用します:

```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

output, err := t.Run(ctx, input)
```

---

## パフォーマンスに関する考慮事項

### 並列実行

ParallelモードはGoのgoroutineを使用して真の並行性を実現します:

```go
// 3つのAgentが同時に実行
t, _ := team.New(team.Config{
    Agents: []*agent.Agent{a1, a2, a3},
    Mode:   team.ModeParallel,
})

// 合計時間 ≈ 最も遅いAgent（すべての合計ではない）
```

### メモリ使用量

各Agentは独自のメモリを維持します。大規模なチームの場合:

```go
// 各実行後にメモリをクリア
output, _ := t.Run(ctx, input)
for _, ag := range t.GetAgents() {
    ag.ClearMemory()
}
```

---

## 次のステップ

- ステップベースのオーケストレーションについては[Workflow](/guide/workflow)を参照
- さまざまなLLMプロバイダーについては[Models](/guide/models)を参照
- Agent機能を拡張するには[Tools](/guide/tools)を追加
- 詳細なAPIドキュメントは[Team APIリファレンス](/api/team)を確認

---

## 関連例

- [Team Demo](/examples/team-demo) - 完全な動作例
- [Leader-Followerパターン](/examples/team-demo#leader-follower)
- [Consensus意思決定](/examples/team-demo#consensus)
