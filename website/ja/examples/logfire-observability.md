# Logfire オブザーバビリティ

OpenTelemetry で HNO のエージェントを計測し、[Logfire](https://logfire.dev) にスパンを送信する例です。推論内容、トークン使用量、ツール実行をオブザーバビリティ基盤で相関できます。

## 実行

```bash
export OPENAI_API_KEY=sk-your-key
export LOGFIRE_WRITE_TOKEN=lf_your_token
go run -tags logfire cmd/examples/logfire_observability/main.go
```

> `logfire` ビルドタグで OpenTelemetry 依存を明示的に含めます。

## 何を計測するか

1. OTLP/HTTP エクスポーター（Logfire 書き込みトークン）
2. 推論対応モデルの実行
3. ランタイム・ループ回数・トークン使用量の属性
4. `reasoning.complete` イベント（推論テキストとトークン数）

## 関連ドキュメント

- 深掘りガイド（GitHub）: https://github.com/rexleimo/HNO/blob/main/docs/release/logfire_observability.md
- 概要: `/ja/advanced/observability`
