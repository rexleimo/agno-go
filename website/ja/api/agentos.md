# AgentOS Server APIリファレンス

## NewServer

HTTPサーバーを作成します。

**シグネチャ:**
```go
func NewServer(config *Config) (*Server, error)

type Config struct {
    Address        string           // サーバーアドレス (デフォルト: :8080)
    Prefix         string           // Optional mount prefix; empty exposes /api/v1
    SessionStorage session.Storage  // セッションストレージ (デフォルト: memory)
    Logger         *slog.Logger     // ロガー (デフォルト: slog.Default())
    Debug          bool             // デバッグモード (デフォルト: false)
    AllowOrigins   []string         // CORSオリジン
    AllowMethods   []string         // CORSメソッド
    AllowHeaders   []string         // CORSヘッダー
    RequestTimeout time.Duration    // リクエストタイムアウト (デフォルト: 30秒)
    MaxRequestSize int64            // 最大リクエストサイズ (デフォルト: 10MB)

    // ナレッジ API (オプション) / Knowledge API (optional)
    VectorDBConfig  *VectorDBConfig  // ベクトルDB構成（例: chromadb）
    EmbeddingConfig *EmbeddingConfig // 埋め込みモデル構成（例: OpenAI）
    KnowledgeAPI    *KnowledgeAPIOptions // ナレッジエンドポイントの有効/無効

    // セッションサマリー (オプション) / Session summaries (optional)
    SummaryManager *session.SummaryManager // 同期/非同期サマリーを構成
}

type VectorDBConfig struct {
    Type           string // 例: "chromadb"
    BaseURL        string // ベクトルDBエンドポイント
    CollectionName string // 既定のコレクション
    Database       string // 任意のデータベース
    Tenant         string // 任意のテナント
}

type EmbeddingConfig struct {
    Provider string // 例: "openai"
    APIKey   string
    Model    string // 例: "text-embedding-3-small"
    BaseURL  string // 例: "https://api.openai.com/v1"
}
```

**例:**
```go
server, err := agentos.NewServer(&agentos.Config{
    Address: ":8080",
    Debug:   true,
    RequestTimeout: 60 * time.Second,
})
```

## Server.RegisterAgent

エージェントを登録します。

**シグネチャ:**
```go
func (s *Server) RegisterAgent(agentID string, ag *agent.Agent) error
```

**例:**
```go
err := server.RegisterAgent("assistant", myAgent)
```

## Server.Start / Shutdown

サーバーを起動および停止します。

**シグネチャ:**
```go
func (s *Server) Start() error
func (s *Server) Shutdown(ctx context.Context) error
```

**例:**
```go
go func() {
    if err := server.Start(); err != nil {
        log.Fatal(err)
    }
}()

// グレースフルシャットダウン
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
server.Shutdown(ctx)
```

## APIエンドポイント

完全なAPIドキュメントは[OpenAPI仕様](../../pkg/agentos/openapi.yaml)を参照してください。

**コアエンドポイント:**
- `GET /health` - ヘルスチェック
- `POST /api/v1/sessions` - セッション作成
- `GET /api/v1/sessions/{id}` - セッション取得
- `PUT /api/v1/sessions/{id}` - セッション更新
- `DELETE /api/v1/sessions/{id}` - セッション削除
- `GET /api/v1/sessions` - セッション一覧
- `POST /api/v1/sessions/{id}/reuse` - セッションを共有
- `GET /api/v1/sessions/{id}/summary` - サマリー取得（準備中は 404）
- `POST /api/v1/sessions/{id}/summary?async=true|false` - 同期/非同期サマリー生成
- `GET /api/v1/sessions/{id}/history` - 履歴取得（`num_messages`、`stream_events` フィルター）
- `GET /api/v1/agents` - エージェント一覧
- `POST /api/v1/agents/{id}/run` - エージェント実行

### セッションサマリーと再利用 (v1.2.6)

`session.SummaryManager` を設定すると同期/非同期サマリーを有効化できます:

```go
// import (
//     "github.com/rexleimo/agno-go/pkg/hno/models/openai"
//     "github.com/rexleimo/agno-go/pkg/hno/session"
// )
summaryModel, _ := openai.New("gpt-4o-mini", openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
})

summary := session.NewSummaryManager(
    session.WithSummaryModel(summaryModel),
    session.WithSummaryTimeout(45*time.Second),
)

server, err := agentos.NewServer(&agentos.Config{
    Address:        ":8080",
    SummaryManager: summary,
})
```

- `POST /api/v1/sessions/{id}/summary`
  - `async=true` バックグラウンドジョブをスケジュールし `202 Accepted` を返す
  - `async=false` 同期実行しサマリーを返す
- `GET /api/v1/sessions/{id}/summary` は最新スナップショットを返す（準備完了前は 404）
- `POST /api/v1/sessions/{id}/reuse` はエージェント、チーム、ワークフロー、ユーザー間でセッションを共有

### 履歴フィルターと実行メタデータ

- `GET /api/v1/sessions/{id}/history?num_messages=20&stream_events=true` で最新 N 件にトリミングし、Python ランタイムの SSE トグルと同等に動作します。
- セッションレスポンスには実行メタデータ（`runs[*].status`、タイムスタンプ、キャンセル理由、`cache_hit`）が含まれ、監査やキャッシュ可観測性を強化します。

**知識エンドポイント（オプション） / Knowledge Endpoints (optional):**
- `POST /api/v1/knowledge/search` — ナレッジベースでベクトル類似検索 / Vector similarity search
- `GET  /api/v1/knowledge/config` — 利用可能なチャンクャー、VectorDB、埋め込みモデル情報 / Available chunkers, VectorDBs, embedding model
- `POST /api/v1/knowledge/content` — チャンクサイズを指定できる最小取り込み（`chunk_size` と `chunk_overlap`、text/plain または application/json）/ Minimal ingestion with configurable chunking (`chunk_size`, `chunk_overlap`).

`POST /api/v1/knowledge/content` は JSON と multipart アップロードの両方で `chunk_size` と `chunk_overlap` を受け付けます。`text/plain` リクエストではクエリパラメータで指定し、ファイルをストリーミングする場合はフォームフィールド（例: `chunk_size=2000&chunk_overlap=250`）として渡します。これらの値は reader メタデータに保存され、下流のパイプラインが文書の分割方法を確認できます。 / `POST /api/v1/knowledge/content` accepts `chunk_size` and `chunk_overlap` in both JSON and multipart uploads. Provide them as query parameters for `text/plain` requests or as form fields (`chunk_size=2000&chunk_overlap=250`) when streaming files. Both values propagate into the reader metadata so downstream pipelines can inspect how documents were segmented.

```bash
curl -X POST http://localhost:8080/api/v1/knowledge/content \
  -F file=@docs/guide.md \
  -F chunk_size=1800 \
  -F chunk_overlap=200 \
  -F metadata='{"source_url":"https://example.com/guide"}'
```

各チャンクには `chunk_size`、`chunk_overlap`、使用された `chunker_type` が記録され、Python 版 AgentOS のレスポンスと揃います。 / Each stored chunk records `chunk_size`, `chunk_overlap`, and the `chunker_type` used—mirroring the AgentOS Python responses.

リクエスト例 / Example:
```bash
curl -X POST http://localhost:8080/api/v1/knowledge/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "エージェントの作成方法は?",
    "limit": 5,
    "filters": {"source": "documentation"}
  }'
```

最小サーバー構成（ナレッジ API 有効化）/ Minimal server config (enable Knowledge API):
```go
server, err := agentos.NewServer(&agentos.Config{
  Address: ":8080",
  VectorDBConfig: &agentos.VectorDBConfig{
    Type:           "chromadb",
    BaseURL:        os.Getenv("CHROMADB_URL"),
    CollectionName: "agno_knowledge",
  },
  EmbeddingConfig: &agentos.EmbeddingConfig{
    Provider: "openai",
    APIKey:   os.Getenv("OPENAI_API_KEY"),
    Model:    "text-embedding-3-small",
  },
})
```

実行可能なサンプル / Runnable example: `cmd/examples/knowledge_api/`

## ベストプラクティス

### 1. 常にContextを使用

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

output, err := ag.Run(ctx, input)
```

### 2. エラーを適切に処理

```go
output, err := ag.Run(ctx, input)
if err != nil {
    switch {
    case types.IsInvalidInputError(err):
        // 無効な入力を処理
    case types.IsRateLimitError(err):
        // バックオフして再試行
    default:
        // その他のエラーを処理
    }
}
```

### 3. メモリを管理

```go
// 新しいトピックを開始する際にクリア
ag.ClearMemory()

// または制限付きメモリを使用
mem := memory.NewInMemory(50)
```

### 4. 適切なタイムアウトを設定

```go
server, _ := agentos.NewServer(&agentos.Config{
    RequestTimeout: 60 * time.Second, // 複雑なエージェント用
})
```
