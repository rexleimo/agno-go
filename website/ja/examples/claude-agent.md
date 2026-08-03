# Claude Agent の例

## 概要

この例では、Agno-Go で Anthropic の Claude モデルを使用する方法を示します。Claude は、思慮深く詳細な回答と強力な推論能力で知られています。この例では、シンプルな会話、計算機ツールの使用、複雑な計算、数学的推論など、複数のユースケースを紹介します。

## 学べること

- Anthropic Claude を Agno-Go と統合する方法
- Claude モデル (Opus、Sonnet、Haiku) の設定方法
- ツール呼び出し機能を備えた Claude の使用方法
- Claude の指示のベストプラクティス

## 前提条件

- Go 1.21 以降
- Anthropic API キー ([console.anthropic.com](https://console.anthropic.com) で取得)

## セットアップ

1. Anthropic API キーを設定します:
```bash
export ANTHROPIC_API_KEY=sk-ant-your-api-key-here
```

2. サンプルディレクトリに移動します:
```bash
cd cmd/examples/claude_agent
```

## 完全なコード

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/anthropic"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	// Create Anthropic Claude model
	model, err := anthropic.New("claude-3-opus-20240229", anthropic.Config{
		APIKey:      apiKey,
		Temperature: 0.7,
		MaxTokens:   2000,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Create calculator toolkit
	calc := calculator.New()

	// Create agent with Claude
	ag, err := agent.New(agent.Config{
		Name:         "Claude Assistant",
		Model:        model,
		Toolkits:     []toolkit.Toolkit{calc},
		Instructions: "You are Claude, a helpful AI assistant created by Anthropic. Use the calculator tools to help users with mathematical calculations. Be precise and explain your reasoning.",
		MaxLoops:     10,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Example 1: Simple conversation
	fmt.Println("=== Example 1: Simple Conversation ===")
	ctx := context.Background()
	output, err := ag.Run(ctx, "Introduce yourself in one sentence.")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	// Example 2: Using calculator tools
	fmt.Println("=== Example 2: Calculator Tool Usage ===")
	output, err = ag.Run(ctx, "What is 156 multiplied by 23, then subtract 100?")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	// Example 3: Complex calculation
	fmt.Println("=== Example 3: Complex Calculation ===")
	output, err = ag.Run(ctx, "Calculate the following: (45 + 67) * 3 - 89. Show your work step by step.")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	// Example 4: Mathematical reasoning
	fmt.Println("=== Example 4: Mathematical Reasoning ===")
	output, err = ag.Run(ctx, "If I have $500 and spend $123, then earn $250, how much money do I have? Use the calculator to verify.")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	fmt.Println("✅ All examples completed successfully!")
}
```

## コードの説明

### 1. Claude モデルの設定

```go
model, err := anthropic.New("claude-3-opus-20240229", anthropic.Config{
	APIKey:      apiKey,
	Temperature: 0.7,
	MaxTokens:   2000,
})
```

**利用可能な Claude モデル:**
- `claude-3-opus-20240229` - 最も高性能、複雑なタスクに最適
- `claude-3-sonnet-20240229` - パフォーマンスと速度のバランス
- `claude-3-haiku-20240307` - 最速、シンプルなタスクに最適

**設定オプション:**
- `Temperature: 0.7` - バランスの取れた創造性 (0.0 = 決定論的、1.0 = 創造的)
- `MaxTokens: 2000` - 最大応答長

### 2. Claude 固有の指示

```go
Instructions: "You are Claude, a helpful AI assistant created by Anthropic.
Use the calculator tools to help users with mathematical calculations.
Be precise and explain your reasoning."
```

Claude は以下に適切に応答します:
- 明確なアイデンティティと目的
- ツール使用に関する明示的な指示
- 推論と説明の重視

### 3. サンプルシナリオ

#### Example 1: シンプルな会話
ツールなしで基本的な会話能力をテストします。

#### Example 2: 計算機ツールの使用
```
クエリ: "What is 156 multiplied by 23, then subtract 100?"
期待されるフロー:
1. multiply(156, 23) → 3588
2. subtract(3588, 100) → 3488
```

#### Example 3: 複雑な計算
```
クエリ: "Calculate: (45 + 67) * 3 - 89. Show your work step by step."
期待されるフロー:
1. add(45, 67) → 112
2. multiply(112, 3) → 336
3. subtract(336, 89) → 247
Claude は各ステップを説明します
```

#### Example 4: 数学的推論
Claude の能力をテスト:
- 文章問題の分解
- 適切なツールの選択
- 明確な説明の提供

## サンプルの実行

```bash
# オプション 1: 直接実行
go run main.go

# オプション 2: ビルドして実行
go build -o claude_agent
./claude_agent
```

## 期待される出力

```
=== Example 1: Simple Conversation ===
Agent: I'm Claude, an AI assistant created by Anthropic to be helpful, harmless, and honest.

=== Example 2: Calculator Tool Usage ===
Agent: Let me calculate that for you. First, 156 multiplied by 23 equals 3,588. Then, subtracting 100 from 3,588 gives us 3,488.

=== Example 3: Complex Calculation ===
Agent: I'll solve this step by step:
1. First, calculate the parentheses: 45 + 67 = 112
2. Then multiply: 112 * 3 = 336
3. Finally subtract: 336 - 89 = 247

The final answer is 247.

=== Example 4: Mathematical Reasoning ===
Agent: Let me help you track your money:
- Starting amount: $500
- After spending $123: $500 - $123 = $377
- After earning $250: $377 + $250 = $627

You have $627 in total.

✅ All examples completed successfully!
```

## Claude vs OpenAI

### Claude を使用する場合

**最適な用途:**
- 複雑な推論タスク
- 詳細な説明
- 安全性が重要なアプリケーション
- 思慮深く微妙な回答

**特徴:**
- より詳細で説明的
- 強力な倫理的推論
- 複雑な指示に従うのが得意
- 不確実性を認めるのが上手

### OpenAI を使用する場合

**最適な用途:**
- 高速な応答
- コード生成
- クリエイティブライティング
- 大規模な関数呼び出し

## モデル選択ガイド

| モデル | 速度 | 能力 | コスト | 用途 |
|-------|-------|------------|------|----------|
| Claude 3 Opus | 遅い | 最高 | 高 | 複雑な分析、研究 |
| Claude 3 Sonnet | 中程度 | 高 | 中 | 汎用、バランス型 |
| Claude 3 Haiku | 速い | 良好 | 低 | シンプルなタスク、大量処理 |

## 設定のヒント

### 決定論的な出力の場合
```go
anthropic.Config{
	Temperature: 0.0,
	MaxTokens:   1000,
}
```

### 創造的なタスクの場合
```go
anthropic.Config{
	Temperature: 1.0,
	MaxTokens:   3000,
}
```

### 本番環境 (バランス型)
```go
anthropic.Config{
	Temperature: 0.7,
	MaxTokens:   2000,
}
```

## ベストプラクティス

1. **明確な指示**: Claude は詳細で構造化されたプロンプトに適切に応答します
2. **推論の要求**: より良い結果を得るために Claude に「説明」または「作業を示す」ように依頼します
3. **安全性**: Claude はより慎重です - 機密性の高いクエリは適切にフレーム化してください
4. **コンテキスト**: Claude は 200K トークンのコンテキストウィンドウを持っています - 長いドキュメントに使用してください

## 次のステップ

- [OpenAI Simple Agent](./simple-agent.md) と比較
- [Ollama でローカルモデル](./ollama-agent.md)を試す
- [マルチエージェントチーム](./team-demo.md)を構築
- [Claude を使った RAG](./rag-demo.md) を探索

## トラブルシューティング

**エラー: "ANTHROPIC_API_KEY environment variable is required"**
- API キーを設定してください: `export ANTHROPIC_API_KEY=sk-ant-...`

**エラー: "model not found"**
- モデル名が正確に一致することを確認してください: `claude-3-opus-20240229`
- API ティアがモデルへのアクセス権を持っているか確認してください

**Opus で応答が遅い**
- より速い応答のために Sonnet の使用を検討してください
- 長い出力が不要な場合は MaxTokens を減らしてください

**レート制限エラー**
- Anthropic はティアごとに異なるレート制限があります
- 指数バックオフを使用した再試行ロジックを実装してください
- 大量タスクには Haiku の使用を検討してください
