# Ollama Agent の例

## 概要

この例では、Ollama を通じてローカル LLM を Agno-Go で使用する方法を示します。Ollama を使用すると、強力な言語モデルをローカルマシンで実行でき、プライバシー、コスト削減、オフライン機能を提供します。これは開発、テスト、プライバシーに配慮したアプリケーションに最適です。

## 学べること

- Ollama を Agno-Go と統合する方法
- ローカル LLM でエージェントを実行する方法
- ローカルモデルでツール呼び出しを使用する方法
- ローカルモデルの利点と制限

## 前提条件

- Go 1.21 以降
- Ollama インストール済み ([ollama.ai](https://ollama.ai))
- ローカルモデルがプル済み (例: llama2、mistral、codellama)

## Ollama のセットアップ

### 1. Ollama のインストール

**macOS/Linux:**
```bash
curl -fsSL https://ollama.ai/install.sh | sh
```

**Windows:**
[ollama.ai/download](https://ollama.ai/download) からダウンロード

### 2. モデルのプル

```bash
# Llama 2 をプル (7B パラメータ、約 4GB)
ollama pull llama2

# または他のモデルを試す:
ollama pull mistral      # Mistral 7B
ollama pull codellama    # コード特化
ollama pull llama2:13b   # より大きく、より高性能
```

### 3. Ollama サーバーの起動

```bash
ollama serve
```

サーバーはデフォルトで `http://localhost:11434` で実行されます。

### 4. インストールの確認

```bash
# モデルをテスト
ollama run llama2 "Hello, how are you?"
```

## 完全なコード

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/ollama"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	// Create Ollama model (uses local Ollama instance)
	// Make sure Ollama is running: ollama serve
	model, err := ollama.New("llama2", ollama.Config{
		BaseURL:     "http://localhost:11434",
		Temperature: 0.7,
		MaxTokens:   2000,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Create calculator toolkit
	calc := calculator.New()

	// Create agent with Ollama
	ag, err := agent.New(agent.Config{
		Name:         "Ollama Assistant",
		Model:        model,
		Toolkits:     []toolkit.Toolkit{calc},
		Instructions: "You are a helpful AI assistant running on Ollama. You can use calculator tools to help with math. Be concise and friendly.",
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
	output, err = ag.Run(ctx, "What is 456 multiplied by 789?")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	// Example 3: Complex calculation
	fmt.Println("=== Example 3: Complex Calculation ===")
	output, err = ag.Run(ctx, "Calculate: (100 + 50) * 2 - 75")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	fmt.Println("✅ All examples completed successfully!")
}
```

## コードの説明

### 1. Ollama モデルの設定

```go
model, err := ollama.New("llama2", ollama.Config{
	BaseURL:     "http://localhost:11434",
	Temperature: 0.7,
	MaxTokens:   2000,
})
```

**設定オプション:**
- **モデル名**: プルされたモデルと一致する必要があります (例: "llama2"、"mistral")
- **BaseURL**: Ollama サーバーアドレス (デフォルト: `http://localhost:11434`)
- **Temperature**: 0.0 (決定論的) から 2.0 (非常に創造的)
- **MaxTokens**: 最大応答長

### 2. API キー不要

OpenAI や Anthropic とは異なり、Ollama はローカルで実行されます:
- ✅ API キー不要
- ✅ 使用コストなし
- ✅ 完全なプライバシー
- ✅ オフラインで動作

### 3. ツールサポート

ローカルモデルはクラウドモデルと同様にツールを使用できます:
```go
Toolkits: []toolkit.Toolkit{calc}
```

エージェントは必要に応じて計算機関数を呼び出します。

## サンプルの実行

### ステップ 1: Ollama の起動
```bash
# ターミナル 1
ollama serve
```

### ステップ 2: サンプルの実行
```bash
# ターミナル 2
cd cmd/examples/ollama_agent
go run main.go
```

## 期待される出力

```
=== Example 1: Simple Conversation ===
Agent: I'm a helpful AI assistant running on Ollama, here to assist you with questions and tasks.

=== Example 2: Calculator Tool Usage ===
Agent: Let me calculate that for you. 456 multiplied by 789 equals 359,784.

=== Example 3: Complex Calculation ===
Agent: Let me solve this step by step:
- First: 100 + 50 = 150
- Then: 150 * 2 = 300
- Finally: 300 - 75 = 225

The answer is 225.

✅ All examples completed successfully!
```

## 利用可能なモデル

### 汎用

| モデル | サイズ | RAM | 説明 |
|-------|------|-----|-------------|
| llama2 | 7B | 8GB | Meta の Llama 2、汎用 |
| llama2:13b | 13B | 16GB | より大きく、より高性能なバージョン |
| mistral | 7B | 8GB | Mistral AI、優れた品質 |
| mixtral | 47B | 32GB | Mixture of experts、非常に高性能 |

### 特化型

| モデル | 用途 |
|-------|----------|
| codellama | コード生成と分析 |
| llama2-uncensored | コンテンツ制限が少ない |
| orca-mini | より小さく、より速い (3B) |
| vicuna | 会話とチャット |

### 利用可能なモデルのリスト表示
```bash
ollama list
```

### 特定のモデルのプル
```bash
ollama pull mistral
ollama pull codellama:13b
```

## 設定例

### 速度重視 (小さいモデル)
```go
ollama.Config{
	Model:       "orca-mini",
	Temperature: 0.5,
	MaxTokens:   500,
}
```

### 品質重視 (大きいモデル)
```go
ollama.Config{
	Model:       "mixtral",
	Temperature: 0.7,
	MaxTokens:   3000,
}
```

### コードタスク用
```go
ollama.Config{
	Model:       "codellama",
	Temperature: 0.3,  // コードでより決定論的
	MaxTokens:   2000,
}
```

### カスタム Ollama サーバー
```go
ollama.Config{
	BaseURL:     "http://192.168.1.100:11434",  // リモート Ollama
	Model:       "llama2",
	Temperature: 0.7,
}
```

## パフォーマンスの考慮事項

### 速度要因

1. **モデルサイズ**: 小さいモデル (7B) は大きいモデル (70B) より速い
2. **ハードウェア**: GPU は推論を大幅に高速化
3. **コンテキスト長**: 長い会話は応答を遅くする

### 典型的な応答時間

| モデル | ハードウェア | 速度 |
|-------|----------|-------|
| llama2 (7B) | Mac M1 | 約 1-2 秒 |
| mistral (7B) | Mac M1 | 約 1-2 秒 |
| mixtral (47B) | Mac M1 | 約 5-10 秒 |
| llama2 (13B) | NVIDIA 3090 | 約 0.5-1 秒 |

## ローカルモデルの利点

### ✅ メリット

1. **プライバシー**: データがマシンから離れることはない
2. **コスト**: API 料金なし、無制限の使用
3. **オフライン**: インターネットなしで動作
4. **コントロール**: モデルとデータの完全な制御
5. **カスタマイズ**: 特定のタスク用にモデルを微調整可能

### ⚠️ 制限

1. **品質**: 一般的に GPT-4 や Claude Opus より低い
2. **速度**: クラウド API より遅い (ハイエンド GPU がない限り)
3. **リソース**: RAM/VRAM が必要 (4-16GB+)
4. **メンテナンス**: モデルと更新を管理する必要がある

## ベストプラクティス

### 1. 適切なモデルを選択

```bash
# 開発/テスト用
ollama pull orca-mini  # 速い、3B パラメータ

# 本番環境用
ollama pull mistral    # 速度/品質のバランスが良い

# 複雑なタスク用
ollama pull mixtral    # 高品質、より多くのリソースが必要
```

### 2. 指示を最適化

ローカルモデルは簡潔で明確な指示から利益を得ます:

```go
// ✅ 良い
Instructions: "You are a math assistant. Use calculator tools for calculations. Be concise."

// ❌ 冗長すぎる
Instructions: "You are an extremely sophisticated mathematical assistant with deep knowledge..."
```

### 3. リソース使用を監視

```bash
# Ollama のステータスを確認
ollama ps

# モデル情報を表示
ollama show llama2
```

### 4. エラーを適切に処理

```go
output, err := ag.Run(ctx, userQuery)
if err != nil {
	// Ollama がダウンしている可能性
	log.Printf("Ollama error: %v. Is the server running?", err)
	// クラウドモデルにフォールバックまたはエラーを返す
}
```

## 統合パターン

### ハイブリッドアプローチ

開発には Ollama を、本番環境にはクラウドを使用:

```go
var model models.Model

if os.Getenv("ENV") == "production" {
	model, _ = openai.New("gpt-4o-mini", openai.Config{...})
} else {
	model, _ = ollama.New("llama2", ollama.Config{...})
}
```

### プライバシー優先アプリケーション

```go
// 機密データには Ollama を使用
sensitiveAgent, _ := agent.New(agent.Config{
	Model: ollamaModel,
	Instructions: "Handle user PII securely...",
})
```

## トラブルシューティング

### エラー: "connection refused"
```bash
# Ollama が実行されているか確認
ollama serve

# またはプロセスを確認
ps aux | grep ollama
```

### エラー: "model not found"
```bash
# 最初にモデルをプル
ollama pull llama2

# 利用可能か確認
ollama list
```

### 応答が遅い
```bash
# より小さいモデルを試す
ollama pull orca-mini

# またはハードウェアアクセラレーションを確認
ollama show llama2 | grep -i gpu
```

### メモリ不足
```bash
# より小さいモデルを使用
ollama pull orca-mini  # 7B の代わりに 3B

# またはスワップスペースを増やす (Linux)
# または他のアプリケーションを閉じる
```

## 次のステップ

- [OpenAI Agent](./simple-agent.md) と [Claude Agent](./claude-agent.md) と比較
- [マルチエージェントチーム](./team-demo.md) でローカルモデルを使用
- ローカル埋め込みを使った[プライバシー保護 RAG](./rag-demo.md) を構築
- ローカルモデルで [Workflows](./workflow-demo.md) を探索

## 追加リソース

- [Ollama ドキュメント](https://github.com/ollama/ollama/blob/main/README.md)
- [Ollama モデルライブラリ](https://ollama.ai/library)
- [ハードウェア要件](https://github.com/ollama/ollama/blob/main/docs/gpu.md)
- [モデル比較](https://ollama.ai/blog/model-comparison)
