# コア機能と Go API 概要

このページでは、現在 Agno-Go に実装されている **Go レベルの API** を概観します。
現時点で安定している公開インターフェイスは主に次の 2 つです。

- `go/pkg/providers/*` 以下の各プロバイダクライアント  
- `internal/agent` と `internal/model` に定義された共通データモデル  

仕様に登場する HTTP ランタイム（`/agents`、`/sessions`、`/messages` など）は
まだ開発中であり、**安定した公開 API とはみなさないでください**。

## 1. 共有データ型

モデルを呼び出す際によく使う型は次のとおりです。

- `agent.ModelConfig` – 利用するプロバイダ／モデルと基本オプション  
  （`Provider` 列挙、`ModelID`、`Stream`、`MaxTokens`、`Temperature` など）。  
- `agent.Message` – 1 つのメッセージ。`Role`（`user` / `assistant` / `system`）と
  `Content`（現在はプレーンテキスト）を持ちます。  
- `model.ChatRequest` – チャットリクエスト：

  ```go
  type ChatRequest struct {
    Model    agent.ModelConfig `json:"model"`
    Messages []agent.Message   `json:"messages"`
    Tools    []agent.ToolCall  `json:"tools,omitempty"`
    Metadata map[string]any    `json:"metadata,omitempty"`
    Stream   bool              `json:"stream,omitempty"`
  }
  ```

- `model.ChatResponse` – 1 回分のアシスタント応答と使用量：

  ```go
  type ChatResponse struct {
    Message      agent.Message `json:"message"`
    Usage        agent.Usage   `json:"usage,omitempty"`
    FinishReason string        `json:"finishReason,omitempty"`
  }
  ```

- `model.ChatStreamEvent` / `model.StreamHandler` – トークン単位のストリーミング用。  
- `model.EmbeddingRequest` / `model.EmbeddingResponse` – embedding 呼び出し用。  
- `model.ChatProvider` / `model.EmbeddingProvider` – 各 provider クライアントが
  実装するインターフェイス。  

## 2. プロバイダクライアント（`go/pkg/providers/*`）

各プロバイダパッケージ（OpenAI、Gemini、Groq など）は `internal/model` の
インターフェイスを実装しています。たとえば OpenAI クライアントは：

- `go/pkg/providers/openai` に配置  
- `New(endpoint, apiKey string, missingEnv []string) *Client` を公開  
- `model.ChatProvider` と `model.EmbeddingProvider` を実装  

最小限の非ストリーミングチャット呼び出しは次のようになります。

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

client := openai.New("", os.Getenv("OPENAI_API_KEY"), nil)

resp, err := client.Chat(ctx, model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
    Stream:   false,
  },
  Messages: []agent.Message{
    {Role: agent.RoleUser, Content: "Agno-Go を簡単に紹介してください。"},
  },
})
if err != nil {
  log.Fatalf("chat error: %v", err)
}

fmt.Println("assistant:", resp.Message.Content)
```

ストリーミング出力が必要な場合は、同じクライアントの `Stream` メソッドと
`model.ChatStreamEvent` を利用します。

```go
req := model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
    Stream:   true,
  },
  Messages: []agent.Message{
    {Role: agent.RoleUser, Content: "短いトークンで挨拶してください。"},
  },
}

err := client.Stream(ctx, req, func(ev model.ChatStreamEvent) error {
  if ev.Type == "token" {
    fmt.Print(ev.Delta)
  }
  if ev.Done {
    fmt.Println()
  }
  return nil
})
if err != nil {
  log.Fatalf("stream error: %v", err)
}
```

Embedding 呼び出しも同様で、`EmbeddingRequest` / `EmbeddingResponse` を使用します。
詳細な例は `go/tests/contract` や `go/tests/providers` を参照してください。

## 3. Router：複数プロバイダの合成

`internal/model.Router` は複数のプロバイダクライアントを 1 つのディスパッチャに
まとめるための仕組みです。

```go
router := model.NewRouter(
  model.WithMaxConcurrency(16),
  model.WithTimeout(30*time.Second),
)

openAI := openai.New("", os.Getenv("OPENAI_API_KEY"), nil)
router.RegisterChatProvider(openAI)

// Gemini や Groq など、他のプロバイダも同様に登録できます。

req := model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
  },
  Messages: []agent.Message{
    {Role: agent.RoleUser, Content: "Hello from router."},
  },
}

resp, err := router.Chat(ctx, req)
if err != nil {
  log.Fatalf("router chat error: %v", err)
}

fmt.Println("assistant:", resp.Message.Content)
```

Router はさらに：

- `router.Stream(ctx, req, handler)` – ストリーミングチャット  
- `router.Embed(ctx, embeddingReq)` – embedding 呼び出し  
- `router.Statuses()` – 各プロバイダのステータス一覧（ヘルスチェック用）  

としても利用できます。内部実装でもこの Router を使っており、自分のサービスの中で
そのまま再利用することもできます。

## 4. HTTP ランタイム（設計メモ・まだ安定していない）

`specs/001-vitepress-docs/contracts/docs-site-openapi.yaml` には、次のような
HTTP ランタイムの設計が記載されています。

- `GET /health` – ヘルスチェックとプロバイダステータス  
- `POST /agents` – エージェント定義の作成  
- `POST /agents/{agentId}/sessions` – セッション作成  
- `POST /agents/{agentId}/sessions/{sessionId}/messages` – メッセージ送信  

ただし、この HTTP インターフェイスは現時点ではまだ設計・実装の途中です。

- Go ランタイム実装が完全には安定していない  
- 一部のフローは、まだ外部公開されていない `go/cmd/agno` の挙動に依存している  

そのため、今は次のような使い方を推奨します。

- `go/pkg/providers/*` 経由で各プロバイダを直接呼び出す  
- 複数プロバイダを組み合わせたい場合は、自分のサービス内で
  `internal/model.Router` を利用する  

HTTP ランタイムと契約が安定した時点で、別途エンドツーエンドの例をドキュメントに
追記する予定です。

## 5. リポジトリ内の参照場所

- `go/pkg/providers/*` – 各プロバイダクライアント（OpenAI / Gemini / Groq など）  
- `go/internal/agent` – エージェント・モデル設定型、使用量集計など  
- `go/internal/model` – リクエスト/レスポンス型、Router、Provider インターフェイス  
- `go/tests/providers` – プロバイダクライアントの具体的な利用例  
- `go/tests/contract` – HTTP 形状のデータモデルを検証する契約テスト  

自分の Go アプリケーションに例を適用する際は、これらのファイルを最終的な
ソース・オブ・トゥルースとして参照してください。

