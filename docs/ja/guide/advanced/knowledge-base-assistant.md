
# 高度なガイド：ナレッジベースアシスタント（Go ベース）

このガイドでは、**自分のドキュメントと検索基盤** を使って質問に回答するアシスタントを、
Agno-Go の Go プロバイダクライアントとデータモデルを用いて構築する方法を説明します。
未完成の HTTP ランタイムには依存せず、あくまで Go コードにフォーカスします。

## 1. シナリオ概要

アシスタントに答えてほしい対象としては例えば：

- プロダクトドキュメント  
- 社内ガイドラインやポリシー  
- ナレッジベース記事  

典型的なパターンは次の通りです。

1. オフラインでドキュメントを embedding し、ベクトル＋メタデータをベクトルストアや
   データベースに保存する  
2. 質問を受け取ったら、ベクトル検索などで関連度の高いパッセージを取得する  
3. 取得したパッセージを `agent.Message.Content` に埋め込んでモデルに回答させる  

Agno-Go 自体はベクトルストアを提供せず、embedding と chat の呼び出しだけを担います。

## 2. プロバイダクライアントでドキュメントを embedding する

`model.EmbeddingProvider` を実装している任意のクライアントを利用できます。
具体的なモデル ID や対応状況は Provider Matrix と `.env.example` を参照してください。

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

func embedDoc(ctx context.Context, text string) ([]float64, error) {
  apiKey := os.Getenv("OPENAI_API_KEY")
  if apiKey == "" {
    return nil, fmt.Errorf("OPENAI_API_KEY not set")
  }

  client := openai.New("", apiKey, nil)

  resp, err := client.Embed(ctx, model.EmbeddingRequest{
    Model: agent.ModelConfig{
      Provider: agent.ProviderOpenAI,
      ModelID:  "text-embedding-3-small", // 適切な embedding モデルを選択
    },
    Input: []string{text},
  })
  if err != nil {
    return nil, err
  }
  if len(resp.Vectors) == 0 {
    return nil, fmt.Errorf("empty embedding response")
  }
  return resp.Vectors[0], nil
}
```

ベクトルをどこに保存するか（Postgres、ClickHouse、専用ベクトル DB など）は
アプリケーション側の自由です。

## 3. 取得したコンテキスト付きで質問に答える

質問に対して関連パッセージ（`[]string`）が取得できたら、次のようにプロンプトを構築して
モデルに渡します。

```go
func answerWithContext(
  ctx context.Context,
  client model.ChatProvider,
  provider agent.Provider,
  modelID string,
  question string,
  passages []string,
) (string, error) {
  var contextText string
  for _, p := range passages {
    contextText += "- " + p + "\n"
  }

  prompt := fmt.Sprintf(
    "あなたは親切なアシスタントです。\n\nコンテキスト:\n%s\n質問: %s\n\n必ずコンテキストの内容に基づいて回答し、情報がない場合は「わかりません」と答えてください。",
    contextText,
    question,
  )

  resp, err := client.Chat(ctx, model.ChatRequest{
    Model: agent.ModelConfig{
      Provider: provider,
      ModelID:  modelID,
    },
    Messages: []agent.Message{
      {Role: agent.RoleUser, Content: prompt},
    },
  })
  if err != nil {
    return "", err
  }
  return resp.Message.Content, nil
}
```

ここで `client` は OpenAI, Gemini, Groq など任意の `ChatProvider` で構いません。
対応する環境変数を `.env` に設定しておけば利用できます。

## 4. 全体像をまとめる

完全なナレッジベースアシスタントは通常、次の 3 つの部品から構成されます。

- **インデクサ** – ドキュメントを読み込み、`Embed` を呼び出してベクトル＋メタデータを保存  
- **リトリーバ** – 質問に対して関連度の高いパッセージを検索して返す  
- **アンサラ** – 上記のパターンでコンテキストをプロンプトに埋め込み、`Chat` を呼び出して回答を生成  

このうち Agno-Go の責務は以下に限られます。

- 一貫した `ChatRequest` / `EmbeddingRequest` の形を提供する  
- 共通インターフェイスを実装したプロバイダクライアントを提供する  
- エラーハンドリングやプロバイダステータス表現をある程度揃える  

ストレージやインデックス、ランキングなどはアプリケーション側で自由に設計してください。

## 5. 他ドキュメントとの関係

- [プロバイダマトリクス](../providers/matrix) を使って、長いコンテキストに向いた
  プロバイダとモデルを選択します。  
- [設定とセキュリティ](../config-and-security) を参照し、`OPENAI_API_KEY` や
  `GEMINI_API_KEY` など必要な環境変数を設定します。  
- 仕様にある HTTP ランタイム設計はここで説明したアイデア（コンテキスト + チャット）
  をそのまま反映したものですが、実装が安定するまでは「データ形状の参考」として扱い、
  コピーしてそのまま使える実装とはみなさないでください。  

