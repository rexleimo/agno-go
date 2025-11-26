# クイックスタート：Go から Agno-Go プロバイダを呼び出す

このガイドでは、本リポジトリに含まれる **Go プロバイダクライアント（`go/pkg/providers/*`）** を使って、
最小限のチャット呼び出しを行う方法を紹介します。現在のバージョンでは：

1. プロバイダ（例：OpenAI）を設定する  
2. `go/pkg/providers/openai` を使って Go コードから呼び出す  
3. 返ってきたメッセージを確認する  

> 注意：AgentOS の HTTP ランタイム（`/agents`、`/sessions`、`/messages` など）はまだ安定していません。
> このクイックスタートは **HTTP サーバや curl に依存せず**、テストで実際に使われている Go クライアントだけにフォーカスします。

## 前提条件

1. Go 1.25.1 がインストールされていること
2. プロジェクトルートで環境変数ファイルを準備します：

```bash
cd <your-project-root>
cp .env.example .env
```

`.env` に OpenAI のキーを設定します：

```bash
OPENAI_API_KEY=あなたの-openai-key
```

## 最小例：OpenAI Chat を呼び出す

以下のコードは、リポジトリ内のテストと同じ形で、

- `internal/agent`  
- `internal/model`  
- `go/pkg/providers/openai`  

をそのまま利用しています。

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
  "github.com/rexleimo/agno-go/pkg/providers/openai"
)

func main() {
  ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
  defer cancel()

  apiKey := os.Getenv("OPENAI_API_KEY")
  if apiKey == "" {
    log.Fatal("OPENAI_API_KEY not set")
  }

  // デフォルトの OpenAI エンドポイントを使用（プロキシを使う場合は .env で OPENAI_ENDPOINT を上書き）
  client := openai.New("", apiKey, nil)

  resp, err := client.Chat(ctx, model.ChatRequest{
    Model: agent.ModelConfig{
      Provider: agent.ProviderOpenAI,
      ModelID:  "gpt-4o-mini",
      Stream:   false,
    },
    Messages: []agent.Message{
      {Role: agent.RoleUser, Content: "Agno-Go を短く紹介してください。"},
    },
  })
  if err != nil {
    log.Fatalf("chat error: %v", err)
  }

  fmt.Println("assistant:", resp.Message.Content)
}
```

上記を次のように保存します：

```bash
<your-project-root>/examples/openai_quickstart/main.go
```

プロジェクトルートで実行します：

```bash
cd <your-project-root>
go run ./examples/openai_quickstart
```

モデルによって内容は変わりますが、次のような出力が得られるはずです。

```text
assistant: Agno-Go は Go で実装された AgentOS であり、...
```

## 次のステップ

- [設定とセキュリティ](./config-and-security) を読み、各プロバイダのキーやエンドポイント、ランタイム設定をどのように管理するか確認してください  
- [プロバイダマトリクス](./providers/matrix) で、各プロバイダがサポートする Chat / Embedding / Streaming の能力と必要な環境変数を確認してください  
- AgentOS の HTTP ランタイム（agents / sessions / messages）はまだ仕様を調整中です。安定したら別途ドキュメント化しますが、それまでは `go/pkg/providers/*` をメインの公開エントリポイントと考えてください  

