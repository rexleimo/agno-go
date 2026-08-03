---
layout: home

hero:
  name: "Agno-Go"
  text: "高性能マルチエージェントフレームワーク"
  tagline: "Python より 16 倍高速 | 180ns インスタンス化 | エージェントあたり 1.2KB メモリ"
  image:
    src: /logo.png
    alt: Agno-Go
  actions:
    - theme: brand
      text: はじめる
      link: /ja/guide/quick-start
    - theme: alt
      text: GitHub で見る
      link: https://github.com/rexleimo/agno-Go

features:
  - icon: 🚀
    title: 極限のパフォーマンス
    details: エージェント初期化は約 180ns、エージェントあたり約 1.2KB のメモリで、Python ランタイムより 16 倍高速です。

  - icon: 🤖
    title: 本番対応 AgentOS
    details: OpenAPI 3.0、セッションストレージ、ヘルスチェック、構造化ログ、CORS、タイムアウトに加え、要約・再利用・履歴フィルターのパリティエンドポイントを備えています。

  - icon: 🪄
    title: セッションパリティ
    details: エージェントやチーム間でセッションを共有し、同期 / 非同期サマリー、キャッシュヒットやキャンセル理由を記録しつつ、Python の `stream_events` スイッチとも互換です。

  - icon: 🧩
    title: 柔軟なアーキテクチャ
    details: エージェント、チーム（4 つの協調モード）、ワークフロー（5 つの制御プリミティブ）を組み合わせ、継承デフォルトやチェックポイント復帰で決定論的にオーケストレーションします。

  - icon: 🔌
    title: マルチプロバイダーモデル
    details: OpenAI o-series、Anthropic Claude、Google Gemini、DeepSeek、GLM、ModelScope、Ollama、Cohere、Groq、Together、OpenRouter、LM Studio、Vercel、Portkey、InternLM、SambaNova をサポート。

  - icon: 🔧
    title: 拡張可能なツール
    details: 計算機、HTTP、ファイル、検索に加え、Claude Agent Skills、Tavily Reader/Search、Gmail 既読化、Jira Worklog、ElevenLabs 音声、PPTX リーダー、MCP コネクタを搭載。

  - icon: 💾
    title: ナレッジとキャッシュ
    details: ChromaDB 連携、バッチ投入ユーティリティ、インジェスト支援に加え、同一モデル呼び出しを重複排除するレスポンスキャッシュを提供します。

  - icon: 🛡️
    title: ガードレールと可観測性
    details: プロンプトインジェクション防御、カスタム前後処理フック、メディア検証、SSE 推論ストリーム、Logfire / OpenTelemetry 連携サンプルを提供します。

  - icon: 📦
    title: シンプルなデプロイ
    details: 単一バイナリ、Docker / Compose / Kubernetes マニフェスト、実践的なデプロイガイドですぐに導入できます。
---

## クイック例

わずか数行のコードで、ツール付き AI エージェントを作成:

```go
package main

import (
    "context"
    "fmt"
    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
    "github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
)

func main() {
    // モデルを作成
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: "your-api-key",
    })

    // ツール付きエージェントを作成
    ag, _ := agent.New(agent.Config{
        Name:     "数学アシスタント",
        Model:    model,
        Toolkits: []toolkit.Toolkit{calculator.New()},
    })

    // エージェントを実行
    output, _ := ag.Run(context.Background(), "25 * 4 + 15 はいくつ?")
    fmt.Println(output.Content) // 出力: 115
}
```

## パフォーマンス比較

| 指標 | Python Agno | Agno-Go | 改善 |
|--------|-------------|---------|-------------|
| エージェント作成 | ~3μs | ~180ns | **16 倍高速** |
| メモリ/エージェント | ~6.5KB | ~1.2KB | **5.4 倍削減** |
| 並行性 | GIL 制限 | ネイティブ goroutine | **無制限** |

## なぜ Agno-Go?

### 本番環境向けに構築

Agno-Go は単なるフレームワークではなく、完全な本番システムです。付属の **AgentOS** サーバーは以下を提供:

- OpenAPI 3.0 仕様の RESTful API
- マルチターン会話のセッション管理
- スレッドセーフなエージェントレジストリ
- ヘルスモニタリングと構造化ロギング
- CORS サポートとリクエストタイムアウト処理

### KISS 原則

**Keep It Simple, Stupid** の哲学に従う:

- **3 つのコア LLM プロバイダー**(45+ ではない) - OpenAI、Anthropic、Ollama
- **必須ツール**(115+ ではない) - 計算機、HTTP、ファイル、検索
- **量より質** - 本番環境対応機能に焦点

### 開発者体験

- **型安全**: Go の強い型付けでコンパイル時にエラーを検出
- **高速ビルド**: Go のコンパイル速度で迅速な反復開発
- **簡単なデプロイ**: ランタイム依存なしの単一バイナリ
- **優れたツール**: 組み込みのテスト、プロファイリング、競合検出

## 5 分でスタート

```bash
# リポジトリをクローン
git clone https://github.com/rexleimo/agno-Go.git
cd agno-Go

# API キーを設定
export OPENAI_API_KEY=sk-your-key-here

# サンプルを実行
go run cmd/examples/simple_agent/main.go

# または AgentOS サーバーを起動
docker-compose up -d
curl http://localhost:8080/health
```

## 含まれるもの

- **コアフレームワーク**: Agent、Team(4 モード)、Workflow(5 プリミティブ)
- **モデル**: OpenAI、Anthropic Claude、Ollama、DeepSeek、Gemini、ModelScope
- **ツール**: Calculator(75.6%)、HTTP(88.9%)、File(76.2%)、Search(92.1%)
- **RAG**: ChromaDB 統合 + OpenAI 埋め込み
- **AgentOS**: 本番環境向け HTTP サーバー(65.0% カバレッジ)
- **サンプル**: すべての機能をカバーする 6 つの実用例
- **ドキュメント**: 完全ガイド、API リファレンス、デプロイ手順

## コミュニティ

- **GitHub**: [rexleimo/agno-Go](https://github.com/rexleimo/agno-Go)
- **Issues**: [バグ報告と機能リクエスト](https://github.com/rexleimo/agno-Go/issues)
- **Discussions**: [質問とアイデア共有](https://github.com/rexleimo/agno-Go/discussions)

## ライセンス

Agno-Go は [MIT ライセンス](https://github.com/rexleimo/agno-Go/blob/main/LICENSE) でリリースされています。

[Agno (Python)](https://github.com/agno-agi/agno) フレームワークからインスピレーションを得ています。
