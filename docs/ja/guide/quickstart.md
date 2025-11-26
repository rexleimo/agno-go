# クイックスタート：Basics シナリオ（docs.agno.com/basics 対応）

スコープ：`./go` モジュール上の Basics 5 シナリオ（basic / memory / rag / tool+HITL / workflow）。ランタイム系コマンドは `go/` 内で実行、Make はリポジトリルート。Go 1.25.1。  
**TODO: Japanese translation polish; mirrors EN content.**

## 1) 前提・環境
リポジトリルートで:
```bash
cp .env.example .env
```
Go モジュールルートへ:
```bash
cd go
export GOCACHE=$PWD/../.cache/go-build   # optional
```
- `AGNO_API_KEY` 必須（`X-API-Key` のみ、FR-004）。
- プロバイダ変数（`*_API_KEY`、`*_CHAT_MODEL`、`*_EMBED_MODEL`）は任意。欠けている場合はスキップ扱い。
- 設定: `../config/default.yaml`（タイムアウト/リトライ/並列度、memory store、env から endpoint 読み込み）。

## 2) 5 シナリオ実行（CLI/fixtures）
```bash
cd go
go run ./cmd/agno --config ../config/default.yaml \
  --scenario <basic|memory|rag|tool|workflow> \
  --fixtures ../specs/001-agno-agents-refactor/contracts/fixtures
```
- basic: 単発、stub リプレイ
- memory: 複数ターン、MemoryStore へ履歴保存
- rag: 検索不可 → ヒント/エラーでフォールバック
- tool: ツール + HITL + ガードレール（ストリーミング含む）
- workflow: 分岐/ワークフローのプレースホルダー（マルチモーダル拡張余地あり）

## 3) テスト・デモ
- 契約テスト（目標 95%以上）:
  ```bash
  cd go
  go test ./tests/contract -run Basics
  ```
- プロバイダデモ（未設定はスキップ理由を記録）:
  ```bash
  cd go
  go run ./cmd/agno --demo --providers openai,gemini \
    --parallel --providers-log ../specs/001-agno-agents-refactor/artifacts/coverage/providers.log
  ```
- 回帰 + ドキュメントビルド（リポジトリルート）:
  ```bash
  make constitution-check
  ```

## 4) プロバイダクライアント例（Go）
`go/pkg/providers/<provider>` を直接使用。例（OpenAI）:
```go
package main

import (
  "context"
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
  client := openai.New("", apiKey, nil)
  resp, err := client.Chat(ctx, model.ChatRequest{
    Model:    agent.ModelConfig{Provider: agent.ProviderOpenAI, ModelID: "gpt-4o-mini", Stream: false},
    Messages: []agent.Message{{Role: agent.RoleUser, Content: "Agno-Go を簡潔に紹介してください。"}},
  })
  if err != nil {
    log.Fatalf("chat error: %v", err)
  }
  log.Println("assistant:", resp.Message.Content)
}
```
他のプロバイダも同じ形で `pkg/providers/<provider>` を利用し、必要な env を設定してください。

## 5) フォールバック・ガードレール・認証
- RAG: hint モードは履歴を書かずガイドを返却、error モードは `ErrUnavailable` を返し履歴を汚さない。
- ガードレール: PII / プロンプトインジェクション → `ErrGuardrailViolation`（履歴非保存）。
- 認証: `X-API-Key` のみ許可。Basic/Bearer/OAuth/カスタムは拒否。

## 6) スキップと差分
- 未設定/到達不可プロバイダは `specs/001-agno-agents-refactor/artifacts/coverage/providers.log` にスキップ理由を記録。
- Python との差分は `specs/001-agno-agents-refactor/contracts/deviations.md`（fixtures は形状プレースホルダー多め）。
- ベンチ: `specs/001-agno-agents-refactor/artifacts/baseline/python-bench.json` と比較（存在する場合）。現行 Go サンプルは `specs/001-agno-agents-refactor/artifacts/bench.txt`、目標 p95 -20%、ピークメモリ -25%。

## 7) ドキュメントビルド（VitePress）
```bash
cd docs
npm run docs:build   # 初回は npm install
# または:
DOCS_DIR=$(pwd)/docs make docs
```
