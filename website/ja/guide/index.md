# HNO とは

**HNO** は Go で構築されたマルチエージェント・システムフレームワークです。
Go の並行処理、静的型、デプロイモデル、標準ツールを使います。特定の負荷に
対する性能値は、再現可能な benchmark がある場合だけ公開します。

## 主な機能

### 再現可能な性能測定

- Agent 作成 benchmark と環境は [Performance](/ja/advanced/performance) に記録します。
- benchmark は `MockModel` を使ったフレームワーク割り当ての測定です。
- LLM 遅延、本番スループット、Go と Python の倍率比較は含みません。
- アプリケーションでは Go の goroutine を利用できます。

### AgentOS

AgentOS は HTTP サーバーとして次の機能を提供します。

- OpenAPI 3.0 対応の REST API
- セッション管理
- スレッドセーフな Agent レジストリ
- ヘルスチェックと構造化ログ
- CORS とリクエストタイムアウト

### アーキテクチャ

- **Agent**: Tool と Memory を使う自律エージェント
- **Team**: 複数 Agent の協調
- **Workflow**: Step、Condition、Loop、Parallel、Router による実行
- **Model**: 複数の LLM Provider を共通インターフェースで扱う
- **Tools / Memory / Storage**: 拡張可能なコンポーネント

## Go を選ぶ理由

Go はコンパイル済みの配布物、組み込みの並行処理、静的型、HTTP/JSON 標準
ライブラリ、テスト・profile・race ツールが理由です。特定サービスでの有利さは
そのサービスの負荷で測定する必要があります。

## HNO という名前

HNO は現在のプロジェクト名です。リポジトリには正式な略語の展開が定義されて
いないため、意味を創作しません。Go module path は
`github.com/rexleimo/agno-go` のままで、HNO はモデルやプロトコルではなく
プロジェクトのブランド名です。

## Provider とツール

現在のソースには複数の LLM Provider、Calculator、HTTP、File、検索、MCP、
Skills、RAG などがあります。対応範囲は実装とテストに基づいて確認してください。
実際の外部 Provider の検証には認証情報が必要で、テスト応答を本番データとは扱いません。

## 次のステップ

1. [Quick Start](/ja/guide/quick-start)
2. [Installation](/ja/guide/installation)
3. [Agent、Team、Workflow](/ja/guide/agent)
4. [Tools](/ja/guide/tools)
5. [Performance](/ja/advanced/performance)

## ライセンス

HNO は [MIT License](https://github.com/rexleimo/HNO/blob/main/LICENSE) で公開されています。

Agno Python プロジェクトから着想を得ていますが、HNO はこの Go プロジェクトの現在の名前です。
