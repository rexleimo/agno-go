
# 高度なガイド：マルチプロバイダルーティング（Go ベース）

このガイドでは、既存のプロバイダクライアントと `internal/model.Router` を使って
**自分の Go サービスの中で** 複数のモデルプロバイダ間でルーティングする方法を説明します。
ここでの例はすべて Go コードであり、未完成の HTTP ランタイムには依存しません。

## 1. どんなときに使うか

代表的なユースケース：

- 一般的なチャットにはあるプロバイダを、低レイテンシ/低コスト用途には別のプロバイダを使う  
- メインプロバイダがダウンしたりレートリミットされたときに別プロバイダへフェイルオーバーする  
- アプリケーション側のインターフェイスを変えずに、新しいモデルを A/B テストする  

本質的なアイデアは、**複数の `ChatProvider` 実装を 1 つの API の裏側に隠す** ことです。

## 2. コアコンポーネント

- `go/pkg/providers/*` – 各モデルプロバイダの Go クライアント。
  `model.ChatProvider` / `model.EmbeddingProvider` を実装します。  
- `internal/model.Router` – `ChatRequest` / `EmbeddingRequest` を登録済みプロバイダに
  ルーティングするディスパッチャ。  
- `agent.ModelConfig` – どのプロバイダ／モデルを使うかを表す設定。  

## 3. 例：OpenAI と Gemini の両方を扱う Router

次の例は HTTP サーバを立てずに、Go プロセス内だけで OpenAI と Gemini を扱う
ルーティングロジックを示します。

```go
package main

import (
  "context"
  "fmt"
  "log"
  "os"
  "time"

  "github.com/rexleimo/agno-go/internal/agent"
  "github.com/rexleimo/agno-go/internal/model"
  "github.com/rexleimo/agno-go/pkg/providers/gemini"
  "github.com/rexleimo/agno-go/pkg/providers/openai"
)

func main() {
  ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
  defer cancel()

  openaiKey := os.Getenv("OPENAI_API_KEY")
  geminiKey := os.Getenv("GEMINI_API_KEY")
  if openaiKey == "" && geminiKey == "" {
    log.Fatal("at least one of OPENAI_API_KEY or GEMINI_API_KEY must be set")
  }

  router := model.NewRouter(
    model.WithMaxConcurrency(16),
    model.WithTimeout(30*time.Second),
  )

  if openaiKey != "" {
    router.RegisterChatProvider(openai.New("", openaiKey, nil))
  }
  if geminiKey != "" {
    router.RegisterChatProvider(gemini.New("", geminiKey, nil))
  }

  // まず OpenAI を試し、ダメなら Gemini にフォールバックする
  providers := []agent.Provider{agent.ProviderOpenAI, agent.ProviderGemini}

  var lastErr error
  for _, prov := range providers {
    req := model.ChatRequest{
      Model: agent.ModelConfig{
        Provider: prov,
        ModelID:  "gpt-4o-mini", // prov==Gemini のときは適切な Gemini モデル ID に置き換える
        Stream:   false,
      },
      Messages: []agent.Message{
        {Role: agent.RoleUser, Content: "小さな社内ツール向けに、安くて速いモデルを推薦し理由を説明してください。"},
      },
    }

    resp, err := router.Chat(ctx, req)
    if err != nil {
      lastErr = err
      log.Printf("provider %s failed: %v", prov, err)
      continue
    }

    fmt.Printf("provider=%s reply=%s\n", prov, resp.Message.Content)
    return
  }

  log.Fatalf("all providers failed, last error: %v", lastErr)
}
```

プロバイダ固有のモデル ID やキー、エンドポイントなどは設定ファイルや環境変数に閉じ込め、
アプリケーションロジック側は「どのプロバイダを優先するか」だけに集中できます。

## 4. ストリーミング版

同じ Router はストリーミングにも使えます。

```go
req := model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
    Stream:   true,
  },
  Messages: []agent.Message{
    {Role: agent.RoleUser, Content: "短い挨拶をストリーミングで出力してください。"},
  },
}

err := router.Stream(ctx, req, func(ev model.ChatStreamEvent) error {
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

フォールバックが必要な場合は、非ストリーミングの例と同様に Provider ごとに
`ChatRequest` を作り直して順番に試すだけです。

## 5. 設定上の注意点

Go 側でマルチプロバイダルーティングを構成する際は：

- API キーやエンドポイントは `.env` や `config/default.yaml` に置き、
  アプリケーションコードにはハードコードしない  
- [プロバイダマトリクス](../providers/matrix) を参照して、有効化する
  プロバイダとモデルを選ぶ  
- Router を「プロバイダの具体的な事情を知っている唯一の場所」とし、それ以外のコードは
  `agent.ModelConfig` と `model.ChatRequest` のみに依存させる  

## 6. HTTP ランタイムとの関係

仕様に記載されている HTTP ランタイム（agents / sessions / messages）は、
ここで紹介した概念（モデル設定やマルチプロバイダルーティング）を反映した設計ですが、
実装はまだ安定していません。

そのため現時点では：

- 本ガイドのように、`go/pkg/providers/*` と `internal/model.Router` を用いて
  自前のサービス内でルーティングを実装する  
- `specs/001-go-agno-rewrite/contracts` にある HTTP 契約は、データ形状の参考としてのみ
  使用し、すでに稼働中の外部 API とはみなさない  

といった扱いを推奨します。HTTP 層が安定したあと、このルーティングパターンは自然と
公開される `agents/sessions/messages` エンドポイントに対応する形になります。

