
# 高度なガイド：メモリ拡張チャット（Go ベース）

このガイドでは、既存のプロバイダクライアントと独自のストレージを使って、
**自分の Go アプリケーション内で** 複数ターン・複数セッションにまたがる
「メモリを持ったチャット」を構築する方法を説明します。例はすべて Go コードであり、
HTTP ランタイムに依存しません。

## 1. メモリの 3 層

メモリは大きく 3 層に分けて考えると整理しやすくなります。

- **会話履歴** – 現在の会話セッション内の直近のやり取り  
- **ユーザープロファイル** – 長期的な嗜好・設定（学習スタイル、言語、プランなど）  
- **ドメイン知識レコード** – サポートチケット、購入履歴、重要なイベントなどの事実  

Agno-Go が直接扱うのは 1 層目（会話履歴）で、2・3 層目はアプリケーションや
バックエンドシステム側で管理します。

## 2. Go で会話履歴を表現する

Go では、会話履歴を単純に `[]agent.Message` で表現できます。

```go
var history []agent.Message

history = append(history,
  agent.Message{Role: agent.RoleUser, Content: "短い学習セッションが好きです。"},
)

// モデルからの応答も履歴に追加
history = append(history,
  agent.Message{Role: agent.RoleAssistant, Content: "わかりました。1 回あたり 30 分以内を目安にします。"},
)
```

モデルを呼び出す際は、この履歴の一部または全部を `ChatRequest.Messages` に渡します。

```go
resp, err := client.Chat(ctx, model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
  },
  Messages: history,
})
```

どの程度の履歴を保持するか、いつ要約するか、どこに保存するかなどはアプリケーション側の判断になります。

## 3. 長期メモリを組み込む

長期メモリは通常、外部ストレージに保存し、必要なときだけプロンプトに注入します。

```go
type UserProfile struct {
  ID          string
  Preferences string // 自然言語での短いサマリ
}

func buildPrompt(profile UserProfile, recent []agent.Message) string {
  var buf strings.Builder
  buf.WriteString("あなたは親切なアシスタントです。\n\n")
  buf.WriteString("【ユーザープロファイル】\n")
  buf.WriteString(profile.Preferences)
  buf.WriteString("\n\n【最近の会話】\n")
  for _, m := range recent {
    buf.WriteString(string(m.Role))
    buf.WriteString(": ")
    buf.WriteString(m.Content)
    buf.WriteString("\n")
  }
  buf.WriteString("\n上記の情報に基づいてユーザーの質問に答えてください。\n")
  return buf.String()
}
```

生成したプロンプトを 1 つの `user` メッセージとして送信します。

```go
prompt := buildPrompt(profile, recentHistory)

resp, err := client.Chat(ctx, model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
  },
  Messages: []agent.Message{
    {Role: agent.RoleUser, Content: prompt},
  },
})
```

アプリケーション側では：

- データベースから `UserProfile` を読み書きする  
- いつ要約し、いつ完全な履歴を保持するかを決める  
- 敏感な情報が安全かつコンプライアンスに沿って扱われているかを確認する  

必要があります。

## 4. ストリーミングとの組み合わせ

メモリ拡張チャットはストリーミング出力とも相性が良いです。

```go
req := model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
    Stream:   true,
  },
  Messages: history,
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

最終的なアシスタント返信を `history` に追加しておけば、次回のリクエストでは更新された
履歴を参照できます。

## 5. ストレージと設定

メモリを多く扱うシナリオでは：

- ユーザープロファイルや会話の断片を永続化するために、データベースやキャッシュ
  （Postgres, Redis, キーバリューストアなど）を利用する  
- [設定とセキュリティ](../config-and-security) を参照し、どのプロバイダを有効化し
  API キーをどう管理するか決める  
- 可能な限り大量の生ログをそのままプロンプトに入れず、要約された重要情報だけを渡す  

といった方針を推奨します。Agno-Go は特定のストレージを強制せず、メッセージとリクエストの
形だけを定義します。

## 6. 他ドキュメントとの関係

- [クイックスタート](../quickstart) は最もシンプルな「ステートレス」な呼び出しフローです。  
- 本ガイドはその上にアプリケーションレベルのメモリとストレージロジックを追加したものです。  
- 仕様に記載されている HTTP ランタイムもセッションやメモリの概念を扱いますが、
  実装が安定するまでは内部設計とみなし、「そのままコピーして動く機能」とは考えないでください。  

