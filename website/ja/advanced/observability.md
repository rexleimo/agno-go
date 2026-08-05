# 可観測性（Observability）

HNO はランタイムの挙動を観測できるように、SSE イベントと OpenTelemetry を備えています。

## AgentOS の SSE ストリーム

`POST /api/v1/agents/{id}/run/stream` は Server‑Sent Events (SSE) を返します。`types` クエリでイベントを絞り込めます（例: `types=run_start,token,reasoning,complete`）。

| イベント | 説明 |
| --- | --- |
| `run_start` | 入力とセッション情報 |
| `token` | モデルからのストリーミングトークン |
| `tool_call` | ツール実行メタデータ（名前・引数・結果） |
| `reasoning` | 推論スナップショット（内容・トークン数・マスク済みテキスト等） |
| `complete` | 最終出力・所要時間・集計済みトークン使用量 |
| `error` | エラーオブジェクト |

## Logfire 連携（OpenTelemetry）

サンプル `cmd/examples/logfire_observability` では、OpenTelemetry を使って Logfire にトレースを送信します。

```bash
# OpenAI/API トークン等を設定
export OPENAI_API_KEY=sk-...
export LOGFIRE_WRITE_TOKEN=lf_...

# OpenTelemetry 依存を含めるために build tag を有効化
go run -tags logfire cmd/examples/logfire_observability/main.go
```

詳細な手順は GitHub のドキュメントをご参照ください：

- https://github.com/rexleimo/agno-go/blob/main/docs/release/logfire_observability.md

次のステップ:
- SSE を任意の APM/監視基盤へ転送
- 推論トークンとコストのダッシュボード連携
- ツール実行やユーザー要求のレイテンシを OpenTelemetry で計測
