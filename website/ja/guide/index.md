# HNOとは？

**HNO**は、Goで構築された高性能なマルチエージェントシステムフレームワークです。Python Agnoフレームワークの設計思想を継承し、Goの並行処理モデルとパフォーマンスの利点を活用して、効率的でスケーラブルなAIエージェントシステムを構築します。

## 主な機能

### 🚀 極限のパフォーマンス

- **エージェントのインスタンス化**: 平均約180ns（Python版の16倍高速）
- **メモリフットプリント**: エージェントあたり約1.2KB（Pythonより5.4倍少ない）
- **ネイティブ並行処理**: GIL制限なしの完全なgoroutineサポート

### 🤖 本番環境対応

HNOには、本番環境HTTPサーバーである**AgentOS**が含まれています：

- OpenAPI 3.0仕様を備えたRESTful API
- マルチターン会話のためのセッション管理
- スレッドセーフなエージェントレジストリ
- ヘルスモニタリングと構造化ロギング
- CORSサポートとリクエストタイムアウト処理

### 🧩 柔軟なアーキテクチャ

異なるユースケースのための3つのコア抽象：

1. **Agent** - ツールサポートとメモリを持つ自律的AIエージェント
2. **Team** - 4つの協調モードによるマルチエージェント協力
   - Sequential、Parallel、Leader-Follower、Consensus
3. **Workflow** - 5つのプリミティブを使用したステップベースのオーケストレーション
   - Step、Condition、Loop、Parallel、Router

### 🔌 マルチモデルサポート

6つの主要LLMプロバイダーの組み込みサポート：

- **OpenAI** - GPT-4、GPT-3.5 Turboなど
- **Anthropic** - Claude 3.5 Sonnet、Claude 3 Opus/Sonnet/Haiku
- **Ollama** - ローカルモデル（Llama 3、Mistral、CodeLlamaなど）
- **DeepSeek** - DeepSeek-V2、DeepSeek-Coder
- **Google Gemini** - Gemini Pro、Flash
- **ModelScope** - Qwen、Yiモデル

### 🔧 拡張可能なツール

KISSの原則に従い、高品質な必須ツールを提供：

- **Calculator** - 基本的な数学演算（テストカバレッジ75.6%）
- **HTTP** - HTTP GET/POSTリクエストの実行（カバレッジ88.9%）
- **File Operations** - セキュリティコントロール付きの読み取り、書き込み、一覧表示、削除（カバレッジ76.2%）
- **Search** - DuckDuckGo Webサーチ（カバレッジ92.1%）

カスタムツールの作成も簡単 - [ツールガイド](/guide/tools)を参照してください。

### 💾 RAG & ナレッジ

ナレッジベースを持つインテリジェントエージェントの構築：

- **ChromaDB** - ベクトルデータベース統合
- **OpenAI Embeddings** - text-embedding-3-small/largeサポート
- 自動埋め込み生成とセマンティック検索

完全な例については[RAGデモ](/examples/rag-demo)を参照してください。

## 設計思想

### KISSの原則

**Keep It Simple, Stupid** - 量より質を重視：

- **3つのコアLLMプロバイダー**（45+ではなく）
- **必須ツール**（115+ではなく）
- **1つのベクトルデータベース**（15+ではなく）

この集中的なアプローチにより以下が保証されます：
- より良いコード品質
- より簡単なメンテナンス
- 本番環境対応の機能

### Goの利点

なぜGoでマルチエージェントシステムを構築するのか？

1. **パフォーマンス** - コンパイル言語、高速実行
2. **並行処理** - ネイティブgoroutine、GILなし
3. **型安全性** - コンパイル時にエラーをキャッチ
4. **シングルバイナリ** - 簡単なデプロイ、ランタイム依存関係なし
5. **優れたツール** - 組み込みのテスト、プロファイリング、レース検出

## ユースケース

HNOは以下に最適です：

- **本番環境AIアプリケーション** - AgentOS HTTPサーバーでデプロイ
- **マルチエージェントシステム** - 複数のAIエージェントの調整
- **高性能ワークフロー** - 数千のリクエストの処理
- **ローカルAI開発** - プライバシー重視のアプリケーションにOllamaを使用
- **RAGアプリケーション** - ナレッジベースのAIアシスタント構築

## 品質メトリクス

- **テストカバレッジ**: コアパッケージ全体で平均80.8%
- **テストケース**: 85以上のテスト、合格率100%
- **ドキュメント**: 完全なガイド、APIリファレンス、例
- **本番環境対応**: Docker、K8sマニフェスト、デプロイガイド

## 次のステップ

始める準備はできましたか？

1. [クイックスタート](/guide/quick-start) - 5分で最初のエージェントを構築
2. [インストール](/guide/installation) - 詳細なセットアップ手順
3. [コアコンセプト](/guide/agent) - Agent、Team、Workflowについて学ぶ

## クイックリンク

- 埋め込み（Embeddings）：[OpenAI / VLLM の使い方](/ja/guide/embeddings)
- ベクターインデックス：[Chroma + Redis（オプション）+ マイグレーション CLI](/ja/advanced/vector-indexing)

## コミュニティ

- **GitHub**: [rexleimo/HNO](https://github.com/rexleimo/HNO)
- **Issues**: [バグ報告](https://github.com/rexleimo/HNO/issues)
- **Discussions**: [質問する](https://github.com/rexleimo/HNO/discussions)

## ライセンス

HNOは[MITライセンス](https://github.com/rexleimo/HNO/blob/main/LICENSE)の下でリリースされています。

[Agno (Python)](https://github.com/agno-agi/agno)フレームワークから着想を得ています。
