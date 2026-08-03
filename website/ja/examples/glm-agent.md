# GLM Agent サンプル

本サンプルは、HNOで中国を代表する国産LLMプラットフォームであるGLM（智谱AI）を使用する方法を示します。

## 概要

GLM（智谱AI）は、清華大学知識工程グループが開発した先進的な言語モデルです。以下の特徴があります：

- **中国語に最適化**: 中国語タスクで優れたパフォーマンス
- **GLM-4**: 128Kコンテキストのメイン会話モデル
- **GLM-4V**: ビジョン対応のマルチモーダル機能
- **GLM-3-Turbo**: 高速でコスト効率の良い変種

## 前提条件

1. **Go 1.21+** がインストールされていること
2. https://open.bigmodel.cn/ から **GLM APIキー** を取得

## APIキーの取得

1. https://open.bigmodel.cn/ にアクセス
2. サインアップまたはログイン
3. APIキーセクションに移動
4. 新しいAPIキーを作成

APIキーの形式: `{key_id}.{key_secret}`

## インストール

```bash
go get github.com/rexleimo/agno-go
```

## 環境設定

`.env`ファイルを作成するか、環境変数をエクスポートします：

```bash
export ZHIPUAI_API_KEY=your-key-id.your-key-secret
```

## 基本的な例

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/glm"
)

func main() {
    // GLMモデルを作成
    model, err := glm.New("glm-4", glm.Config{
        APIKey:      os.Getenv("ZHIPUAI_API_KEY"),
        Temperature: 0.7,
        MaxTokens:   1024,
    })
    if err != nil {
        log.Fatalf("GLMモデルの作成に失敗しました: %v", err)
    }

    // Agentを作成
    agent, err := agent.New(agent.Config{
        Name:         "GLM アシスタント",
        Model:        model,
        Instructions: "あなたは役立つAIアシスタントです。",
    })
    if err != nil {
        log.Fatalf("Agentの作成に失敗しました: %v", err)
    }

    // Agentを実行
    output, err := agent.Run(context.Background(), "こんにちは！自己紹介してください。")
    if err != nil {
        log.Fatalf("Agent実行失敗: %v", err)
    }

    fmt.Println(output.Content)
}
```

## ツールを使用した例

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/glm"
    "github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
    "github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
    // GLMモデルを作成
    model, err := glm.New("glm-4", glm.Config{
        APIKey:      os.Getenv("ZHIPUAI_API_KEY"),
        Temperature: 0.7,
        MaxTokens:   1024,
    })
    if err != nil {
        log.Fatal(err)
    }

    // 計算ツールを持つAgentを作成
    agent, err := agent.New(agent.Config{
        Name:         "GLM 計算アシスタント",
        Model:        model,
        Toolkits:     []toolkit.Toolkit{calculator.New()},
        Instructions: "あなたは計算を実行できる役立つAIアシスタントです。",
    })
    if err != nil {
        log.Fatal(err)
    }

    // 計算をテスト
    output, err := agent.Run(context.Background(), "123 × 456 はいくつですか？")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("結果: %s\n", output.Content)
}
```

## 中国語の例

GLMは中国語タスクで優れています：

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/glm"
)

func main() {
    model, err := glm.New("glm-4", glm.Config{
        APIKey:      os.Getenv("ZHIPUAI_API_KEY"),
        Temperature: 0.7,
    })
    if err != nil {
        log.Fatal(err)
    }

    agent, err := agent.New(agent.Config{
        Name:         "中文助手",
        Model:        model,
        Instructions: "你是一个有用的中文AI助手。",
    })
    if err != nil {
        log.Fatal(err)
    }

    // 中国語で質問
    output, err := agent.Run(context.Background(), "请用中文介绍一下人工智能的发展历史。")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(output.Content)
}
```

## サンプルの実行

1. リポジトリをクローン：
```bash
git clone https://github.com/rexleimo/agno-go.git
cd HNO
```

2. APIキーを設定：
```bash
export ZHIPUAI_API_KEY=your-key-id.your-key-secret
```

3. サンプルを実行：
```bash
go run cmd/examples/glm_agent/main.go
```

## 設定オプション

```go
glm.Config{
    APIKey:      string  // 必須: {key_id}.{key_secret} 形式
    BaseURL:     string  // オプション: カスタムAPIエンドポイント
    Temperature: float64 // オプション: 0.0-1.0（デフォルト: 0.7）
    MaxTokens:   int     // オプション: 最大応答トークン数
    TopP:        float64 // オプション: Top-pサンプリングパラメータ
    DoSample:    bool    // オプション: サンプリングを有効化
}
```

## 認証

GLMはJWT（JSON Web Token）認証を使用します：

- APIキーは`key_id`と`key_secret`に分割されます
- HMAC-SHA256署名を使用してJWTトークンを生成します
- トークンの有効期限は7日間です
- SDKによって自動的に処理されます

## サポートされているモデル

| モデル | コンテキスト | 最適な用途 |
|-------|---------|----------|
| `glm-4` | 128K | 一般的な会話、中国語 |
| `glm-4v` | 128K | ビジョンタスク、マルチモーダル |
| `glm-3-turbo` | 128K | 高速応答、コスト効率 |

## よくある問題

### 無効なAPIキー形式

**問題**: `API key must be in format {key_id}.{key_secret}`

**解決方法**: APIキーにkey_idとkey_secretの間にドット（.）区切りが含まれていることを確認してください。

### 認証失敗

**問題**: `GLM API error: Invalid API key`

**解決方法**:
- APIキーが正しいことを確認
- https://open.bigmodel.cn/ でAPIキーが有効か確認
- 環境変数に余分なスペースがないか確認

### レート制限

**問題**: `GLM API error: Rate limit exceeded`

**解決方法**:
- 指数バックオフでリトライロジックを実装
- リクエスト頻度を下げる
- 必要に応じてAPIプランをアップグレード

## 次のステップ

- 他のLLMオプションについては[Models](/ja/guide/models)を参照
- 機能を強化するために[Tools](/ja/guide/tools)を追加
- 複数のAgentで[Teams](/ja/guide/team)を構築
- 複雑なプロセスのために[Workflows](/ja/guide/workflow)を探索

## 関連例

- [Simple Agent](/ja/examples/simple-agent) - OpenAIの例
- [Claude Agent](/ja/examples/claude-agent) - Anthropicの例
- [Team Demo](/ja/examples/team-demo) - マルチエージェント協調

## リソース

- [GLM 公式ウェブサイト](https://www.bigmodel.cn/)
- [GLM API ドキュメント](https://open.bigmodel.cn/dev/api)
- [HNO リポジトリ](https://github.com/rexleimo/agno-go)
