# Team コラボレーションの例

## 概要

この例では、HNO のマルチエージェントチーム協力機能を示します。Team を使用すると、複数のエージェントが異なる協調モードで協力できます: Sequential、Parallel、Leader-Follower、Consensus。各モードは、異なる種類のタスクと協力パターンに適しています。

## 学べること

- マルチエージェントチームの作成方法
- 4 つのチーム協調モードとそれぞれをいつ使用するか
- エージェントがコンテキストを共有し、互いの作業に基づいて構築する方法
- 個別のエージェント出力にアクセスする方法

## 前提条件

- Go 1.21 以降
- OpenAI API キー

## セットアップ

```bash
export OPENAI_API_KEY=sk-your-api-key-here
cd cmd/examples/team_demo
```

## 完全なコード

完全な例には 4 つのデモが含まれています - 詳細はコードの説明を参照してください。

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/openai"
	"github.com/rexleimo/agno-go/pkg/hno/team"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	ctx := context.Background()

	// Demo 1: Sequential Team
	fmt.Println("=== Demo 1: Sequential Team ===")
	runSequentialDemo(ctx, apiKey)

	fmt.Println("\n=== Demo 2: Parallel Team ===")
	runParallelDemo(ctx, apiKey)

	fmt.Println("\n=== Demo 3: Leader-Follower Team ===")
	runLeaderFollowerDemo(ctx, apiKey)

	fmt.Println("\n=== Demo 4: Consensus Team ===")
	runConsensusDemo(ctx, apiKey)
}
```

## Team 協調モード

### 1. Sequential モード

エージェントは順番に作業し、各エージェントが前のエージェントの出力に基づいて構築します。

```go
func runSequentialDemo(ctx context.Context, apiKey string) {
	model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

	// Create 3 agents for sequential processing
	researcher, _ := agent.New(agent.Config{
		Name:         "Researcher",
		Model:        model,
		Instructions: "You are a research expert. Analyze the topic and provide key facts.",
	})

	analyst, _ := agent.New(agent.Config{
		Name:         "Analyst",
		Model:        model,
		Instructions: "You are an analyst. Take research findings and extract insights.",
	})

	writer, _ := agent.New(agent.Config{
		Name:         "Writer",
		Model:        model,
		Instructions: "You are a writer. Take insights and write a concise summary.",
	})

	// Create sequential team
	t, _ := team.New(team.Config{
		Name:   "Content Pipeline",
		Agents: []*agent.Agent{researcher, analyst, writer},
		Mode:   team.ModeSequential,
	})

	// Run team
	output, _ := t.Run(ctx, "Analyze the benefits of AI in healthcare")

	fmt.Printf("Final Output: %s\n", output.Content)
	fmt.Printf("Agents involved: %d\n", len(output.AgentOutputs))
}
```

**フロー:**
1. **Researcher** がトピックを分析 → 調査結果を生成
2. **Analyst** が結果を受け取る → 洞察を抽出
3. **Writer** が洞察を受け取る → 最終要約を書く

**ユースケース:**
- コンテンツ作成パイプライン
- データ処理ワークフロー
- 多段階分析タスク

### 2. Parallel モード

すべてのエージェントが同じ入力で同時に作業し、出力を組み合わせます。

```go
func runParallelDemo(ctx context.Context, apiKey string) {
	model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

	// Create agents with different specializations
	techAgent, _ := agent.New(agent.Config{
		Name:         "Tech Specialist",
		Model:        model,
		Instructions: "You are a technology expert. Focus on technical aspects.",
	})

	bizAgent, _ := agent.New(agent.Config{
		Name:         "Business Specialist",
		Model:        model,
		Instructions: "You are a business expert. Focus on business implications.",
	})

	ethicsAgent, _ := agent.New(agent.Config{
		Name:         "Ethics Specialist",
		Model:        model,
		Instructions: "You are an ethics expert. Focus on ethical considerations.",
	})

	// Create parallel team
	t, _ := team.New(team.Config{
		Name:   "Multi-Perspective Analysis",
		Agents: []*agent.Agent{techAgent, bizAgent, ethicsAgent},
		Mode:   team.ModeParallel,
	})

	output, _ := t.Run(ctx, "Evaluate the impact of autonomous vehicles")
	fmt.Printf("Combined Analysis:\n%s\n", output.Content)
}
```

**フロー:**
1. すべてのエージェントが同時に同じ入力を受け取る
2. 各エージェントが自分の視点を提供
3. 出力が包括的な分析に組み合わされる

**ユースケース:**
- 多視点分析
- ブレインストーミングセッション
- 独立した評価
- 並列データ処理

### 3. Leader-Follower モード

リーダーエージェントがフォロワーエージェントにタスクを委譲し、結果を統合します。

```go
func runLeaderFollowerDemo(ctx context.Context, apiKey string) {
	model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

	// Create leader
	leader, _ := agent.New(agent.Config{
		Name:         "Team Leader",
		Model:        model,
		Instructions: "You are a team leader. Delegate tasks and synthesize results.",
	})

	// Create followers with tools
	calcAgent, _ := agent.New(agent.Config{
		Name:         "Calculator Agent",
		Model:        model,
		Toolkits:     []toolkit.Toolkit{calculator.New()},
		Instructions: "You perform calculations as requested.",
	})

	dataAgent, _ := agent.New(agent.Config{
		Name:         "Data Agent",
		Model:        model,
		Instructions: "You analyze and present data.",
	})

	// Create leader-follower team
	t, _ := team.New(team.Config{
		Name:   "Project Team",
		Leader: leader,
		Agents: []*agent.Agent{calcAgent, dataAgent},
		Mode:   team.ModeLeaderFollower,
	})

	output, _ := t.Run(ctx, "Calculate the ROI for a $100,000 investment with 15% annual return over 5 years")
	fmt.Printf("Leader's Final Report: %s\n", output.Content)
}
```

**フロー:**
1. **Leader** がタスクを分析し、フォロワーに委譲
2. **Followers** が割り当てられたサブタスクを実行
3. **Leader** が結果を統合し、最終出力を提供

**ユースケース:**
- 複雑なタスク分解
- 階層的ワークフロー
- プロジェクト管理シナリオ
- 特殊なツールの使用

### 4. Consensus モード

エージェントは合意に達するか最大ラウンド数に達するまで議論します。

```go
func runConsensusDemo(ctx context.Context, apiKey string) {
	model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

	// Create agents with different perspectives
	optimist, _ := agent.New(agent.Config{
		Name:         "Optimist",
		Model:        model,
		Instructions: "You are optimistic and focus on opportunities.",
	})

	realist, _ := agent.New(agent.Config{
		Name:         "Realist",
		Model:        model,
		Instructions: "You are realistic and balanced in your views.",
	})

	critic, _ := agent.New(agent.Config{
		Name:         "Critic",
		Model:        model,
		Instructions: "You are critical and focus on potential problems.",
	})

	// Create consensus team
	t, _ := team.New(team.Config{
		Name:      "Decision Team",
		Agents:    []*agent.Agent{optimist, realist, critic},
		Mode:      team.ModeConsensus,
		MaxRounds: 2,
	})

	output, _ := t.Run(ctx, "Should we invest in renewable energy for our company?")

	fmt.Printf("Consensus Result: %s\n", output.Content)
	fmt.Printf("Total discussion rounds: %v\n", output.Metadata["rounds"])
}
```

**フロー:**
1. **ラウンド 1**: 各エージェントが最初の視点を提供
2. **ラウンド 2**: エージェントが他者の見解を見て立場を改善
3. **最終**: システムがコンセンサスまたは最終的な立場を統合

**ユースケース:**
- 意思決定
- 議論シミュレーション
- 多視点分析
- リスク評価

## Team 設定

### 基本設定

```go
team.Config{
	Name:   "My Team",           // チーム識別子
	Agents: []*agent.Agent{...}, // チームメンバー
	Mode:   team.ModeSequential, // 協調モード
}
```

### 高度な設定

```go
team.Config{
	Name:      "Decision Team",
	Leader:    leaderAgent,      // Leader-Follower モード用
	Agents:    followerAgents,   // チームメンバー
	Mode:      team.ModeConsensus,
	MaxRounds: 3,                // Consensus モード用
}
```

## 結果へのアクセス

### Team 出力

```go
output, err := t.Run(ctx, "Your query here")

// 最終結果
fmt.Println(output.Content)

// 個別エージェント出力
for _, agentOut := range output.AgentOutputs {
	fmt.Printf("%s: %s\n", agentOut.AgentName, agentOut.Content)
}

// メタデータ
fmt.Printf("Rounds: %v\n", output.Metadata["rounds"])
```

### 個別エージェント出力

```go
// 特定のエージェントの貢献にアクセス
if len(output.AgentOutputs) > 0 {
	firstAgent := output.AgentOutputs[0]
	fmt.Printf("Agent: %s\n", firstAgent.AgentName)
	fmt.Printf("Output: %s\n", firstAgent.Content)
}
```

## サンプルの実行

```bash
go run main.go
```

## 期待される出力

```
=== Demo 1: Sequential Team ===
Final Output: AI in healthcare offers significant benefits including improved diagnostic accuracy through machine learning, personalized treatment plans, reduced administrative burden, and enhanced patient monitoring through IoT devices.
Agents involved: 3

=== Demo 2: Parallel Team ===
Combined Analysis:
Technical: Autonomous vehicles use advanced sensors, AI algorithms, and real-time processing...
Business: Market disruption, new revenue models, infrastructure investment needs...
Ethics: Privacy concerns, liability questions, job displacement, safety standards...

=== Demo 3: Leader-Follower Team ===
Leader's Final Report: Based on calculations, a $100,000 investment at 15% annual return over 5 years yields $201,136, representing a 101% ROI.

=== Demo 4: Consensus Team ===
Consensus Result: After thorough discussion, the team recommends investing in renewable energy with careful planning for upfront costs and long-term savings.
Total discussion rounds: 2
```

## モード比較

| モード | 使用時期 | エージェント数 | 通信パターン |
|------|-------------|-------------|----------------------|
| **Sequential** | パイプラインタスク、順序付きステップ | 2-10 | 線形: A → B → C |
| **Parallel** | 独立したタスク、複数の視点 | 2-20 | ブロードキャスト: すべて同じ入力 |
| **Leader-Follower** | 複雑な委譲、階層構造 | 1 リーダー + 1-10 フォロワー | ハブアンドスポーク: リーダーが調整 |
| **Consensus** | 意思決定、議論 | 2-5 | ラウンドロビン議論 |

## ベストプラクティス

### 1. 適切なモードを選択

```go
// Sequential: 順序が重要な場合
team.ModeSequential  // 調査 → 分析 → 執筆

// Parallel: 複数の視点が必要な場合
team.ModeParallel    // 技術 + ビジネス + 法律分析

// Leader-Follower: 委譲が必要な場合
team.ModeLeaderFollower  // 複雑なタスク分解

// Consensus: 合意が必要な場合
team.ModeConsensus   // 意思決定、議論
```

### 2. 明確なエージェントの役割を設計

```go
// ✅ 良い: 具体的で異なる役割
researcher := "You are a research expert. Focus on facts and data."
analyst := "You are an analyst. Extract insights from research."

// ❌ 悪い: 重複した、曖昧な役割
agent1 := "You are helpful."
agent2 := "You are smart."
```

### 3. エージェント数を最適化

- **Sequential**: 2-5 エージェント (多い = 長いパイプライン)
- **Parallel**: 2-10 エージェント (多い = より豊かな分析)
- **Leader-Follower**: 1 リーダー + 2-5 フォロワー
- **Consensus**: 2-4 エージェント (多い = 収束が困難)

### 4. エラーを処理

```go
output, err := team.Run(ctx, query)
if err != nil {
	log.Printf("Team execution failed: %v", err)
	// フォールバックロジック
}
```

## 高度なパターン

### 混合ツール使用

```go
// 一部のエージェントにはツールがあり、他にはない
calcAgent := agent.New(agent.Config{
	Toolkits: []toolkit.Toolkit{calculator.New()},
})

analysisAgent := agent.New(agent.Config{
	// ツールなし、純粋な推論
})
```

### 動的チーム構成

```go
var agents []*agent.Agent

if needsCalculation {
	agents = append(agents, calcAgent)
}
if needsWebSearch {
	agents = append(agents, searchAgent)
}

team, _ := team.New(team.Config{Agents: agents, Mode: team.ModeParallel})
```

### ネストされたチーム

```go
// サブチームを作成
researchTeam := team.New(team.Config{...})
analysisTeam := team.New(team.Config{...})

// 1 つのチームの出力を別のチームの入力として使用
researchOutput, _ := researchTeam.Run(ctx, query)
finalOutput, _ := analysisTeam.Run(ctx, researchOutput.Content)
```

## パフォーマンスの考慮事項

### Sequential モード
- **レイテンシー**: すべてのエージェント時間の合計 (最も遅い)
- **コスト**: すべてのエージェントコストの合計
- **最適**: 順序が重要な場合

### Parallel モード
- **レイテンシー**: エージェント時間の最大 (より速い)
- **コスト**: すべてのエージェントコストの合計
- **最適**: 速度が重要な場合

### Leader-Follower モード
- **レイテンシー**: リーダー + フォロワー (中程度)
- **コスト**: リーダー + フォロワーコスト
- **最適**: 複雑なタスク委譲

### Consensus モード
- **レイテンシー**: ラウンド × エージェント時間 (最も遅い)
- **コスト**: ラウンド × エージェント数
- **最適**: コンセンサスが重要な場合

## 次のステップ

- [Simple Agent](./simple-agent.md) の基本から始める
- 制御された実行のために [Workflow Engine](./workflow-demo.md) を探索
- チーム協力で [RAG Systems](./rag-demo.md) を構築
- 異なる [Model Providers](./claude-agent.md) を試す

## トラブルシューティング

**エージェントが効果的に協力していない:**
- 明確性についてエージェントの指示を確認
- モードがタスクに適合しているか確認
- エージェントが異なる役割を持っているか確認

**Sequential チームが遅すぎる:**
- エージェント数を減らす
- より小さい/速いモデルを使用
- Parallel モードを検討

**Consensus が収束しない:**
- MaxRounds を増やす
- 決定を簡素化
- エージェント数を減らす
- エージェントの指示を調整

**Leader が適切に委譲していない:**
- リーダーの委譲指示を明確にする
- フォロワーが適切なツールを持っているか確認
- フォロワーの指示が明確か確認
