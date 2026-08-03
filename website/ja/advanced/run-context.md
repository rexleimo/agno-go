---
title: 実行コンテキスト
description: hooks・ツール・モデル呼び出しに run_context_id を伝播し、イベントで相関します。
---

# 実行コンテキスト（Run Context）

HNO は実行ごとのコンテキスト識別子を伝播し、SSE では `run_context_id` を含めてエンドツーエンドの相関を可能にします。

## SSE イベント

`POST /api/v1/agents/{id}/run?stream_events=true` は `run_start`、`reasoning`、`token`、`complete`、`error` を出力し、すべてに `run_context_id` が含まれます。

## コード例

```go
ctx := agent.WithRunContext(context.Background(), "rc-123")
out, err := myAgent.Run(ctx, "hello")
```

Hook / Toolkit 内での取得：

```go
id, _ := agent.RunContextID(ctx)
```

## 注意

- 明示しない場合、HTTP レイヤーが実行ごとに自動注入します。
- キャンセル/タイムアウトは Context を通じて hooks・tools・モデルに伝播します。

