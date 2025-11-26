# 퀵스타트: Basics 시나리오 (docs.agno.com/basics 대응)

범위: `./go` 모듈의 Basics 5 시나리오(basic / memory / rag / tool+HITL / workflow). 런타임 명령은 `go/`에서 실행, Make 는 리포지토리 루트. Go 1.25.1.  
**TODO: Korean translation polish; mirrors EN.**

## 1) 전제 및 환경
리포지토리 루트:
```bash
cp .env.example .env
```
Go 모듈 루트로 이동:
```bash
cd go
export GOCACHE=$PWD/../.cache/go-build   # optional
```
- `AGNO_API_KEY` 필수 (`X-API-Key`만 허용, FR-004).
- 프로바이더 변수(`*_API_KEY`, `*_CHAT_MODEL`, `*_EMBED_MODEL`)는 선택. 없으면 “미설정”으로 스킵 처리.
- 설정: `../config/default.yaml` (타임아웃/재시도/동시성, memory store, env 에서 endpoint 로드).

## 2) 5 시나리오 실행 (CLI/fixtures)
```bash
cd go
go run ./cmd/agno --config ../config/default.yaml \
  --scenario <basic|memory|rag|tool|workflow> \
  --fixtures ../specs/001-agno-agents-refactor/contracts/fixtures
```
- basic: 단발, stub 리플레이
- memory: 멀티턴, MemoryStore 히스토리 저장
- rag: 검색 불가 → 힌트/에러 폴백
- tool: 툴 + HITL + 가드레일 (스트리밍 포함)
- workflow: 분기/워크플로 플레이스홀더 (멀티모달 확장 여지)

## 3) 테스트 및 데모
- 계약 테스트(목표 95% 이상):
  ```bash
  cd go
  go test ./tests/contract -run Basics
  ```
- 프로바이더 데모(스킵 이유 로그):
  ```bash
  cd go
  go run ./cmd/agno --demo --providers openai,gemini \
    --parallel --providers-log ../specs/001-agno-agents-refactor/artifacts/coverage/providers.log
  ```
- 회귀 + 문서 빌드(리포지토리 루트):
  ```bash
  make constitution-check
  ```

## 4) 프로바이더 클라이언트 예시 (Go)
`go/pkg/providers/<provider>` 를 직접 사용. 예시(OpenAI):
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
    Messages: []agent.Message{{Role: agent.RoleUser, Content: "Agno-Go를 한 문장으로 소개해 주세요."}},
  })
  if err != nil {
    log.Fatalf("chat error: %v", err)
  }
  log.Println("assistant:", resp.Message.Content)
}
```
다른 프로바이더도 동일한 형태로 `pkg/providers/<provider>` 사용, 필수 env 설정 필요.

## 5) 폴백, 보안, 인증
- RAG: hint 모드는 히스토리 미기록 가이던스, error 모드는 `ErrUnavailable`.
- 가드레일: PII / 프롬프트 인젝션 시 `ErrGuardrailViolation` (히스토리 미기록).
- 인증: `X-API-Key`만 허용. Basic/Bearer/OAuth/커스텀 Authorization 거부.

## 6) 프로바이더 스킵 및 차이
- 미설정/접속 불가 → `specs/001-agno-agents-refactor/artifacts/coverage/providers.log` 에 스킵 이유 기록.
- Python 대비 차이 → `specs/001-agno-agents-refactor/contracts/deviations.md` (fixtures 다수는 shape 플레이스홀더).
- 벤치: `specs/001-agno-agents-refactor/artifacts/baseline/python-bench.json` 과 비교(존재 시). 현행 Go 샘플은 `specs/001-agno-agents-refactor/artifacts/bench.txt`, 목표 p95 -20%, 피크 메모리 -25%.

## 7) 문서 빌드 (VitePress)
```bash
cd docs
npm run docs:build   # 최초 npm install 필요할 수 있음
# 또는:
DOCS_DIR=$(pwd)/docs make docs
```
