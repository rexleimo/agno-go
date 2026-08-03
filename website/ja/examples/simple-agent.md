# Simple Agent サンプル

## 概要

このサンプルは、HNOを使用してツール呼び出し機能を持つシンプルなAIエージェントを作成する基本的な使い方を示します。エージェントはOpenAIのGPT-4o-miniモデルを使用し、数学的な演算を実行するための計算ツールキットを装備しています。

## 学べること

- OpenAIモデルの作成と設定方法
- ツールを持つエージェントのセットアップ方法
- ユーザークエリでエージェントを実行する方法
- 実行メタデータ（ループ、トークン使用量）へのアクセス方法

## 前提条件

- Go 1.21以上
- OpenAI APIキー

## セットアップ

1. OpenAI APIキーを設定:
```bash
export OPENAI_API_KEY=sk-your-api-key-here
```

2. サンプルディレクトリに移動:
```bash
cd cmd/examples/simple_agent
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
	"github.com/rexleimo/agno-go/pkg/hno/models/openai"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	// 環境からAPIキーを取得
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// OpenAIモデルを作成
	model, err := openai.New("gpt-4o-mini", openai.Config{
		APIKey:      apiKey,
		Temperature: 0.7,
		MaxTokens:   1000,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// 計算ツールキットを作成
	calc := calculator.New()

	// エージェントを作成
	ag, err := agent.New(agent.Config{
		Name:         "Math Assistant",
		Model:        model,
		Toolkits:     []toolkit.Toolkit{calc},
		Instructions: "You are a helpful math assistant. Use the calculator tools to help users with mathematical calculations.",
		MaxLoops:     10,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// エージェントを実行
	ctx := context.Background()
	output, err := ag.Run(ctx, "What is 25 multiplied by 4, then add 15?")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}

	// 結果を出力
	fmt.Println("Agent Response:")
	fmt.Println(output.Content)
	fmt.Println("\nMetadata:")
	fmt.Printf("Loops: %v\n", output.Metadata["loops"])
	fmt.Printf("Usage: %+v\n", output.Metadata["usage"])
}
```

## コードの説明

### 1. モデル設定

```go
model, err := openai.New("gpt-4o-mini", openai.Config{
	APIKey:      apiKey,
	Temperature: 0.7,
	MaxTokens:   1000,
})
```

- GPT-4o-miniを使用するOpenAIモデルインスタンスを作成
- `Temperature: 0.7`はバランスの取れた創造性と一貫性を提供
- `MaxTokens: 1000`は応答長を制限

### 2. 計算ツールキット

```go
calc := calculator.New()
```

計算ツールキットは4つの関数を提供:
- `add` - 2つの数値の加算
- `subtract` - 2つの数値の減算
- `multiply` - 2つの数値の乗算
- `divide` - 2つの数値の除算

### 3. エージェント設定

```go
ag, err := agent.New(agent.Config{
	Name:         "Math Assistant",
	Model:        model,
	Toolkits:     []toolkit.Toolkit{calc},
	Instructions: "You are a helpful math assistant...",
	MaxLoops:     10,
})
```

- `Name` - エージェントを識別
- `Model` - 推論に使用するLLM
- `Toolkits` - エージェントが利用できるツールコレクションの配列
- `Instructions` - エージェントの動作を定義するシステムプロンプト
- `MaxLoops` - ツール呼び出しの最大反復回数（無限ループを防ぐ）

### 4. エージェントの実行

```go
output, err := ag.Run(ctx, "What is 25 multiplied by 4, then add 15?")
```

エージェントは以下を行います:
1. ユーザークエリを分析
2. 計算ツールを使用する必要があると判断
3. `multiply(25, 4)`を呼び出して100を取得
4. `add(100, 15)`を呼び出して115を取得
5. 自然言語の応答を返す

## サンプルの実行

```bash
# オプション1: 直接実行
go run main.go

# オプション2: ビルドして実行
go build -o simple_agent
./simple_agent
```

## 期待される出力

```
Agent Response:
The result of 25 multiplied by 4 is 100, and when you add 15 to that, you get 115.

Metadata:
Loops: 2
Usage: map[completion_tokens:45 prompt_tokens:234 total_tokens:279]
```

## 重要な概念

### ツール呼び出しループ

`MaxLoops`パラメータは、エージェントがツールを呼び出せる回数を制御:

1. **ループ1**: エージェントが`multiply(25, 4)`を呼び出し → 結果: 100
2. **ループ2**: エージェントが`add(100, 15)`を呼び出し → 結果: 115
3. **最終**: エージェントが自然言語の応答を生成

各ループは、1回のツール呼び出しと結果処理を表します。

### メタデータ

`output.Metadata`には有用な実行情報が含まれます:
- `loops` - 実行されたツール呼び出しの反復回数
- `usage` - トークン消費量（プロンプト、完了、合計）

## 次のステップ

- Anthropic統合については[Claude Agentサンプル](./claude-agent.md)を探索
- 複数のエージェントによる[チーム協調](./team-demo.md)について学ぶ
- 複雑なプロセスのための[ワークフローエンジン](./workflow-demo.md)を試す
- ナレッジ検索を使用した[RAGアプリケーション](./rag-demo.md)を構築

## トラブルシューティング

**エラー: "OPENAI_API_KEY environment variable is required"**
- APIキーをエクスポートしたことを確認: `export OPENAI_API_KEY=sk-...`

**エラー: "model not found"**
- GPT-4o-miniモデルへのアクセスがあることを確認
- 代替として"gpt-3.5-turbo"を試す

**エラー: "max loops exceeded"**
- エージェントがMaxLoops制限（10）に達した
- `MaxLoops`を増やすかクエリを簡略化
