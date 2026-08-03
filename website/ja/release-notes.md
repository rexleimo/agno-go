---
title: リリースノート
description: HNOのバージョン履歴とリリースノート
outline: deep
---

# リリースノート

## Version 1.2.9 (2025-11-14)

### ✨ ハイライト
- **EvoLink プロバイダー**: `pkg/hno/providers/evolink` および `pkg/hno/models/evolink/*` により、テキスト・画像・動画の生成をサポートし、EvoLink ドキュメントに沿ったオプション構造体と非同期タスクポーリングを提供します。
- **EvoLink メディアエージェント例**: `website/examples/evolink-media-agents.md` と中国語ページで、テキスト → 画像 → 動画へつなぐワークフローの組み立て方を解説します。
- **ナレッジアップロードのチャンク分割**: `POST /api/v1/knowledge/content` が JSON、`text/plain`（クエリパラメータ）、multipart アップロードで `chunk_size` と `chunk_overlap` を受け付け、各チャンクのメタデータに `chunk_size`・`chunk_overlap`・`chunker_type` を保存します（Python AgentOS と整合）。
- **AgentOS HTTP Tips**: AgentOS API ドキュメントに、カスタムヘルスチェックパス、`/openapi.yaml`・`/docs` ルート、およびルーター変更後に呼び出す `server.Resync()` のガイダンスを追加しました。

### 📚 ドキュメント
- `website/api/agentos.md` と各言語版を更新し、Knowledge セクションにチャンクパラメータと例を追加し、ベストプラクティスに HTTP Tips を含めました。
- EvoLink 例ページでは、必要な環境変数、モデル一覧、HTTPS コールバックやモデレーション上の注意点を整理しました。

### ✅ 互換性
- 追加的なリリースであり、公開 API に破壊的変更はありません。`chunk_size`・`chunk_overlap` はオプションで、省略時は従来どおりの挙動を保ちます。

## Version 1.2.8 (2025-11-10)

### ✨ ハイライト
- Run Context が実行全体（hooks → tools → telemetry）に伝播し、ストリーミングイベントに `run_context_id` を含めてトレース相関を容易にします。
- セッション状態に `AGUI` サブステートを永続化し、`GET /sessions/{id}` で UI 状態を返却。
- ベクター索引：
  - プラガブルな VectorDB プロバイダー（Chroma デフォルト；Redis はオプションで強依存なし）。
  - VectorDB マイグレーション CLI（`migrate up/down`）で冪等な作成/ロールバックを提供。
- Embeddings：VLLM プロバイダー（ローカル/リモート）が共通インターフェースを実装。
- MCPTools：任意の `tool_name_prefix` で登録ツール名に接頭辞を付与。

### 🔧 改善
- Redis をデフォルトの VectorDB 依存から分離。未設定時は無影響で、設定時のみプロバイダーを登録。
- チームのモデル継承はプライマリモデルのみを伝搬。補助フラグはエージェント側で明示的に有効化が必要。

### 🐛 修正
- モデル応答をアクティブステップに正しくバインドし、履歴の未バインド/ゼロ値を修正。
- チームのツール判定が OS スキーマに整合し、メンバーのツール集合を保持。
- 非同期 DB のナレッジフィルターが複合述語とタイムアウトを尊重し、goroutine リークを防止。
- ツールキットのインポート解決でモジュール欠如時に構造化エラーを返却（panic 回避）。
- AgentOS のエラー応答を標準化し、契約テストを安定化。

### 🧪 テスト
- Run Context、AGUI 永続化、チームのプライマリモデル継承、MCP 接頭辞、VLLM 埋め込みを追加カバー。
- オプションの Redis テストは依存関係が無い環境ではスキップ。

### ✅ 互換性
- 追加的な更新でパブリック API は不変。オプション機能はデフォルトで無効。

## Version 1.2.7 (2025-11-03)

### ✨ ハイライト
- Go ネイティブのセッションサービスが Python AgentOS の `/sessions` API を完全に再現し、Postgres ベースの CRUD、Chi ルーター、ヘルスチェックを提供（[ガイド](/ja/guide/session-service)）。
- あらゆる環境向けのデプロイ資産: 専用 Dockerfile、Postgres 同梱の Docker Compose スタック、Kubernetes 用 Helm チャート。
- ドキュメントと `test-session-api.sh` スクリプトを更新し、ローカルおよび CI でエンドポイント検証を実施可能に。

### 🔧 改善
- Postgres ストア実装が型付き DTO とトランザクションセーフな処理を備え、既存の AgentOS スキーマと整合しました。
- DSN 配線、環境変数、ワークフロースクリプトを解説する新しい構成ガイドで Go セッションランタイムの導入を支援。

### 🧪 テスト
- Go と Python のレスポンスを比較する契約テスト、および Postgres ストア専用テストを追加。

### ✅ 互換性
- 追加的な更新で、Go セッションランタイムはオプションとして Python サービスと並行稼働可能です。

## Version 1.2.6 (2025-10-31)

### ✨ ハイライト
- セッションパリティ: セッション再利用エンドポイント、同期/非同期サマリー（`GET/POST /sessions/{id}/summary`）、履歴ページングパラメータ（`num_messages`、`stream_events`）、実行メタデータ（キャッシュヒット、キャンセル理由、タイムスタンプ）を追加。
- レスポンスキャッシュ: エージェント/チーム向けにメモリ LRU キャッシュと設定可能なサマリーマネージャを提供。
- メディア添付パイプライン: Agent/Team/Workflow すべてでメディア添付をサポートし、検証ヘルパーと `WithMediaPayload` 実行オプションを含む。
- ストレージアダプタ: MongoDB と SQLite セッションストレージを追加し、Postgres と同一の JSON コントラクトを維持。
- ツールキット拡張: Tavily Reader/Search、Claude Agent Skills、Gmail 既読処理、Jira Worklog、ElevenLabs 音声、強化されたファイルツール。
- カルチャー知識マネージャ: タグフィルタと非同期処理で組織知識を管理。

### 🔧 改善
- ワークフローエンジンがキャンセル理由を永続化し、resume-from チェックポイントとメディア専用ペイロードをサポート。
- AgentOS セッション API がサマリーエンドポイント、再利用セマンティクス、SSE トグル付きの履歴ページングを公開。
- MCP クライアントがケイパビリティマニフェストをキャッシュし、メディア添付を転送してレイテンシを削減。

### 🧪 テスト
- キャッシュ層、サマリーマネージャ、ストレージドライバ、ワークフロー復帰パス、新ツールキットを対象にしたテストを追加。

### ✅ 互換性
- 追加的な変更のみで後方互換性を維持。

## Version 1.2.5 (2025-10-20)

### ✨ ハイライト
- モデルプロバイダーを8種追加：Cohere、Together、OpenRouter、LM Studio、Vercel、Portkey、InternLM、SambaNova（同期/ストリーミング、関数呼び出し対応）
- 評価システム（シナリオ評価・指標集計・モデル比較）、メディア処理（画像メタデータ；音声/動画プローブのプレースホルダー）、デバッグツール（リクエスト/レスポンスの簡易ダンプ）、クラウド配備プレースホルダー（NoopDeployer）
- 統合レジストリ（登録/一覧/ヘルスチェック）、共通ユーティリティ（JSONPretty、Retry）

### 🔧 修正
- Airflow ツールの返却構造を Airflow REST API v2 に整合：`total_entries`、`dag_run_id`、`logical_date`
- サイトのトップ画像欠落を修正：`/logo.svg` → `/logo.png`

### 🧪 テスト
- 新規モデル/モジュールの単体テストを強化、既存のベンチマークは維持

### ✅ 互換性
- 追加的な変更のみで後方互換性を維持

## バージョン 1.2.1 (2025-10-15)

### 🧭 ドキュメント再編

- 明確に分離：
  - `website/` → 実装済みの対外ドキュメント（VitePress サイト）
  - `docs/` → 設計ドラフト、移行計画、タスク、開発者/内部ドキュメント
- `docs/README.md` を追加（方針と入口を記載）
- 貢献者向けに `CONTRIBUTING.md` を追加

### 🔗 リンク修正

- README、CLAUDE、CHANGELOG、リリースノートのリンクを `website/advanced/*` と `website/guide/*` に統一
- `docs/` 配下の重複実装ドキュメントへの旧リンクを削除

### 🌐 サイト更新

- API：AgentOS ページにナレッジ API を追記（/api/agentos）
- Workflow History と Performance ページを正規参照に

### ✅ 動作変更

- なし（ドキュメントと構成のみ更新）

### ✨ 今回の追加（実装済み）

- A2A ストリーミングのイベント種別フィルタ（SSE）
  - `POST /api/v1/agents/:id/run/stream?types=token,complete`
  - 要求したイベントのみ出力；標準SSE形式；Contextキャンセル対応
- AgentOS コンテンツ抽出ミドルウェア
  - JSON/Form から `content/metadata/user_id/session_id` をContextに注入
  - `MaxRequestSize` によるサイズ保護とスキップパスをサポート
- Google Sheets ツール（サービスアカウント）
  - `read_range`、`write_range`、`append_rows`；JSON/ファイル資格情報対応
- 最小限のナレッジ取り込みエンドポイント
  - `POST /api/v1/knowledge/content` は `text/plain` と `application/json` をサポート

企業向けの検収手順は [`docs/ENTERPRISE_MIGRATION_PLAN.md`](https://github.com/rexleimo/HNO/blob/main/docs/ENTERPRISE_MIGRATION_PLAN.md) を参照してください。

## バージョン 1.1.0 (2025-10-08)

### 🎉 ハイライト

本リリースは、本番環境対応のマルチエージェントシステムのための強力な新機能をもたらします：

- **A2Aインターフェース** - 標準化されたエージェント間通信プロトコル
- **セッション状態管理** - ワークフローステップ間の永続的な状態
- **マルチテナントサポート** - 単一エージェントインスタンスで複数のユーザーにサービスを提供
- **モデルタイムアウト設定** - LLM呼び出しのための細かいタイムアウト制御

---

### ✨ 新機能

#### A2A (Agent-to-Agent) インターフェース

JSON-RPC 2.0に基づくエージェント間相互作用の標準化された通信プロトコル。

**主な機能:**
- RESTful APIエンドポイント（`/a2a/message/send`, `/a2a/message/stream`）
- マルチメディアサポート（テキスト、画像、ファイル、JSONデータ）
- ストリーミング用Server-Sent Events (SSE)
- Python Agno A2A実装と互換性あり

**クイック例:**
```go
import "github.com/rexleimo/agno-go/pkg/agentos/a2a"

// A2Aインターフェースを作成
a2a := a2a.New(a2a.Config{
    Agents: []a2a.Entity{myAgent},
    Prefix: "/a2a",
})

// ルートを登録（Gin）
router := gin.Default()
a2a.RegisterRoutes(router)
```

📚 **詳細を学ぶ:** [A2Aインターフェースドキュメント](/ja/api/a2a)

---

#### ワークフローセッション状態管理

ワークフローステップ間で状態を維持するためのスレッドセーフなセッション管理。

**主な機能:**
- ステップ間の永続的な状態ストレージ
- `sync.RWMutex`によるスレッドセーフ
- 並列ブランチ分離のためのディープコピー
- データ損失を防ぐスマートマージ戦略
- Python Agno v2.1.2の競合状態を修正

**クイック例:**
```go
// セッション情報付きコンテキストを作成
execCtx := workflow.NewExecutionContextWithSession(
    "input",
    "session-123",  // セッションID
    "user-a",       // ユーザーID
)

// セッション状態にアクセス
execCtx.SetSessionState("key", "value")
value, _ := execCtx.GetSessionState("key")
```

📚 **詳細を学ぶ:** [セッション状態ドキュメント](/ja/guide/session-state)

---

#### マルチテナントサポート

単一のAgentインスタンスで複数のユーザーにサービスを提供し、完全なデータ分離を保証。

**主な機能:**
- ユーザー分離された会話履歴
- Memoryインターフェースのオプション`userID`パラメータ
- 既存コードとの下位互換性
- スレッドセーフな並行操作
- クリーンアップ用`ClearAll()`メソッド

**クイック例:**
```go
// マルチテナントエージェントを作成
agent, _ := agent.New(&agent.Config{
    Name:   "customer-service",
    Model:  model,
    Memory: memory.NewInMemory(100),
})

// ユーザーAの会話
agent.UserID = "user-a"
output, _ := agent.Run(ctx, "My name is Alice")

// ユーザーBの会話
agent.UserID = "user-b"
output, _ := agent.Run(ctx, "My name is Bob")
```

📚 **詳細を学ぶ:** [マルチテナントドキュメント](/ja/advanced/multi-tenant)

---

#### モデルタイムアウト設定

細かい制御でLLM呼び出しのリクエストタイムアウトを設定。

**主な機能:**
- デフォルト: 60秒
- 範囲: 1秒から10分
- サポートモデル: OpenAI、Anthropic Claude
- コンテキストを考慮したタイムアウト処理

**クイック例:**
```go
// カスタムタイムアウト付きOpenAI
model, _ := openai.New("gpt-4", openai.Config{
    APIKey:  apiKey,
    Timeout: 30 * time.Second,
})

// カスタムタイムアウト付きClaude
claude, _ := anthropic.New("claude-3-opus", anthropic.Config{
    APIKey:  apiKey,
    Timeout: 45 * time.Second,
})
```

📚 **詳細を学ぶ:** [モデル設定](/ja/guide/models#timeout-configuration)

---

### 🐛 バグ修正

- **ワークフロー競合状態** - 並列ステップ実行のデータ競合を修正
  - Python Agno v2.1.2には共有`session_state` dictによる上書きの問題がありました
  - Go実装はブランチごとに独立したSessionStateクローンを使用
  - スマートマージ戦略により並行実行でのデータ損失を防止

---

### 📚 ドキュメント

すべての新機能には包括的なバイリンガルドキュメント（英語/中文）が含まれています：

- [A2Aインターフェースガイド](/ja/api/a2a) - 完全なプロトコル仕様
- [セッション状態ガイド](/ja/guide/session-state) - ワークフロー状態管理
- [マルチテナントガイド](/ja/advanced/multi-tenant) - データ分離パターン
- [モデル設定](/ja/guide/models#timeout-configuration) - タイムアウト設定

---

### 🧪 テスト

**新しいテストスイート:**
- `session_state_test.go` - セッション状態テスト543行
- `memory_test.go` - マルチテナントメモリテスト（新しいテストケース4つ）
- `agent_test.go` - マルチテナントエージェントテスト
- `openai_test.go` - タイムアウト設定テスト
- `anthropic_test.go` - タイムアウト設定テスト

**テスト結果:**
- ✅ すべてのテストが`-race`検出器で合格
- ✅ ワークフローカバレッジ: 79.4%
- ✅ メモリカバレッジ: 93.1%
- ✅ エージェントカバレッジ: 74.7%

---

### 📊 パフォーマンス

**パフォーマンス低下なし** - すべてのベンチマークが一貫しています：
- Agent実例化: ~180ns/op（Pythonより16倍高速）
- メモリフットプリント: ~1.2KB/エージェント
- スレッドセーフな並行操作

---

### ⚠️ 破壊的変更

**なし。** 本リリースはv1.0.xと完全に下位互換性があります。

---

### 🔄 移行ガイド

**移行は不要** - すべての新機能は追加的で下位互換性があります。

**オプションの拡張:**

1. **マルチテナントサポートを有効化:**
   ```go
   // エージェント設定にUserIDを追加
   agent := agent.New(agent.Config{
       UserID: "user-123",  // NEW
       Memory: memory.NewInMemory(100),
   })
   ```

2. **ワークフローでセッション状態を使用:**
   ```go
   // セッション付きコンテキストを作成
   ctx := workflow.NewExecutionContextWithSession(
       "input",
       "session-id",
       "user-id",
   )
   ```

3. **モデルタイムアウトを設定:**
   ```go
   // モデル設定にTimeoutを追加
   model, _ := openai.New("gpt-4", openai.Config{
       APIKey:  apiKey,
       Timeout: 30 * time.Second,  // NEW
   })
   ```

---

### 📦 インストール

```bash
go get github.com/rexleimo/agno-go@v1.1.0
```

---

### 🔗 リンク

- **GitHubリリース:** [v1.1.0](https://github.com/rexleimo/agno-go/releases/tag/v1.1.0)
- **完全な変更ログ:** [CHANGELOG.md](https://github.com/rexleimo/agno-go/blob/main/CHANGELOG.md)
- **ドキュメント:** [https://agno-go.dev](https://agno-go.dev)

---

## バージョン 1.0.3 (2025-10-06)

### 🧪 改善

- **JSONシリアライゼーションテストの強化** - utils/serializeパッケージで100%のテストカバレッジを達成
- **パフォーマンスベンチマーク** - Python Agnoパフォーマンステストパターンと整合
- **包括的なドキュメント** - バイリンガルパッケージドキュメントを追加

---

## バージョン 1.0.2 (2025-10-05)

### ✨ 追加

#### GLM (智谱AI) プロバイダー

- Zhipu AIのGLMモデルとの完全統合
- GLM-4、GLM-4V（ビジョン）、GLM-3-Turboのサポート
- カスタムJWT認証（HMAC-SHA256）
- 同期およびストリーミングAPI呼び出し
- ツール/関数呼び出しサポート

---

## バージョン 1.0.0 (2025-10-02)

### 🎉 初回リリース

HNO v1.0は、Agnoマルチエージェントフレームワークの高性能Go実装です。

#### コア機能
- **Agent** - ツールサポート付き単一自律エージェント
- **Team** - 4つのモードによるマルチエージェント協力
- **Workflow** - 5つのプリミティブによるステップベースのオーケストレーション

#### LLMプロバイダー
- OpenAI（GPT-4、GPT-3.5、GPT-4 Turbo）
- Anthropic（Claude 3.5 Sonnet、Claude 3 Opus/Sonnet/Haiku）
- Ollama（ローカルモデル）

---

**最終更新:** 2025-10-08
