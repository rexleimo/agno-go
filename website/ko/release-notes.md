---
title: 릴리스 노트
description: Agno-Go의 버전 히스토리 및 릴리스 노트
outline: deep
---

# 릴리스 노트

## Version 1.2.9 (2025-11-14)

### ✨ 하이라이트
- **EvoLink 프로바이더**: `pkg/hno/providers/evolink` 및 `pkg/hno/models/evolink/*` 아래에서 텍스트·이미지·비디오 생성을 지원하고, EvoLink 제약에 맞는 옵션 구조체와 비동기 태스크 폴링을 제공합니다.
- **EvoLink 미디어 에이전트 예제**: `website/examples/evolink-media-agents.md` 와 중국어 페이지를 통해 텍스트 → 이미지 → 비디오로 이어지는 워크플로를 단계별로 설명합니다.
- **지식 업로드 청킹**: `POST /api/v1/knowledge/content` 가 JSON, `text/plain`(쿼리 파라미터), multipart 업로드에서 `chunk_size` 와 `chunk_overlap` 를 수용하고, 각 청크 메타데이터에 `chunk_size`·`chunk_overlap`·`chunker_type` 을 기록해 Python AgentOS와 동작을 맞춥니다.
- **AgentOS HTTP 팁**: AgentOS API 문서에 사용자 정의 헬스 체크 경로, `/openapi.yaml`·`/docs` 라우트, 라우터 변경 후 `server.Resync()` 호출 시점에 대한 안내를 추가했습니다.

### 📚 문서
- `website/api/agentos.md` 및 현지화 페이지를 업데이트해 Knowledge 섹션에 청킹 파라미터와 예제를 추가하고, 베스트 프랙티스에 HTTP 팁을 포함했습니다.
- EvoLink 예제 문서에서 필요한 환경 변수, 모델 테이블, HTTPS 콜백 및 모더레이션 주의사항을 정리했습니다.

### ✅ 호환성
- 증분 릴리스로, 공개 API에는 깨지는 변경 사항이 없습니다. `chunk_size`·`chunk_overlap` 파라미터는 선택 사항이며 생략 시 기존 동작을 유지합니다.

## Version 1.2.8 (2025-11-10)

### ✨ 하이라이트
- Run Context가 실행 전반(hooks → tools → telemetry)에 전파되며, 스트리밍 이벤트에 `run_context_id`를 포함해 트레이스 상관을 용이하게 합니다.
- 세션 상태에 `AGUI` 서브스테이트를 영속화하고 `GET /sessions/{id}`에서 UI 상태를 반환합니다.
- 벡터 인덱싱:
  - 플러그형 VectorDB 프로바이더(Chroma 기본; Redis 선택적, 강한 의존성 아님).
  - VectorDB 마이그레이션 CLI(`migrate up/down`)로 멱등적 생성/롤백 제공.
- Embeddings: VLLM 프로바이더(로컬/원격)가 공통 인터페이스를 구현.
- MCPTools: 선택적 `tool_name_prefix`로 등록 도구명에 접두사 부여.

### 🔧 개선
- Redis를 기본 VectorDB 의존성에서 분리. 미설정 시 영향 없음; 설정 시에만 프로바이더 등록.
- 팀 모델 상속은 기본 모델만 전파; 보조 플래그는 에이전트 측에서 명시적으로 활성화 필요.

### 🐛 수정
- 모델 응답을 활성 단계에 올바르게 바인딩하여 이력의 미바인딩/제로값 문제 수정.
- 팀 도구 결정이 OS 스키마에 정렬되어 멤버 도구 보존.
- 비동기 DB 기반 지식 필터가 복합 조건과 타임아웃을 준수하며 goroutine 누수 방지.
- 툴킷 임포트 시 모듈 누락에 대해 구조화된 오류 반환(패닉 방지).
- AgentOS 오류 응답 표준화로 계약 테스트 안정성 향상.

### 🧪 테스트
- Run Context, AGUI 영속화, 팀 기본 모델 상속, MCP 접두사, VLLM 임베딩에 대한 커버리지 추가.
- 선택적 Redis 테스트는 의존성 부재 시 건너뜀.

### ✅ 호환성
- 추가 업데이트이며 공개 API는 변경 없음. 선택 기능은 기본적으로 비활성.

## Version 1.2.7 (2025-11-03)

### ✨ 하이라이트
- Go 네이티브 세션 서비스가 Python AgentOS `/sessions` API를 그대로 재현하고, Postgres 기반 CRUD, Chi 라우팅, 헬스 체크를 제공합니다([가이드](/ko/guide/session-service)).
- 전용 Dockerfile, Postgres 포함 Docker Compose 스택, Kubernetes용 Helm 차트 등 모든 환경을 위한 배포 자산을 제공합니다.
- 문서와 `test-session-api.sh` 스크립트를 갱신해 로컬 및 CI 환경에서 엔드포인트 검증을 수행할 수 있습니다.

### 🔧 개선
- Postgres 스토어 구현이 타입 안전 DTO와 트랜잭션 세이프 처리를 제공해 기존 AgentOS 스키마와 정합을 유지합니다.
- DSN 연결, 환경 변수, 워크플로 스크립트를 다루는 새 구성 가이드로 Go 세션 런타임 도입을 지원합니다.

### 🧪 테스트
- Go와 Python 응답을 비교하는 계약 테스트와 Postgres 스토어 전용 테스트를 추가했습니다.

### ✅ 호환성
- 증분 업데이트이며 Go 세션 런타임은 선택 사항으로 기존 Python 서비스와 병행 운용할 수 있습니다.

## Version 1.2.6 (2025-10-31)

### ✨ 하이라이트
- 세션 패리티: 세션 재사용 엔드포인트, 동기/비동기 요약(`GET/POST /sessions/{id}/summary`), 역사 페이지네이션 파라미터(`num_messages`, `stream_events`), 실행 메타데이터(캐시 적중, 취소 사유, 타임스탬프)를 제공.
- 응답 캐시: 에이전트/팀을 위한 메모리 LRU 캐시와 요약 매니저 기본값을 구성 가능.
- 미디어 첨부 파이프라인: Agent/Team/Workflow 전반에서 미디어 첨부 지원, 검증 도우미 및 `WithMediaPayload` 실행 옵션 포함.
- 스토리지 어댑터: MongoDB, SQLite 세션 스토리지를 추가하고 Postgres와 동일한 JSON 계약을 유지.
- 툴킷 확장: Tavily Reader/Search, Claude Agent Skills, Gmail 읽음 처리, Jira 워크로그, ElevenLabs 음성, 향상된 파일 도구.
- 컬처 지식 매니저: 태그 필터와 비동기 작업으로 조직 지식을 관리.

### 🔧 개선
- 워크플로 엔진이 취소 사유를 지속화하고 resume-from 체크포인트 및 미디어 전용 페이로드를 지원.
- AgentOS 세션 API가 요약 엔드포인트, 재사용 시맨틱, SSE 토글을 포함한 역사 페이지네이션을 노출.
- MCP 클라이언트가 능력 매니페스트를 캐시하고 미디어 첨부를 전달하여 호출 지연을 줄임.

### 🧪 테스트
- 캐시 계층, 요약 매니저, 스토리지 드라이버, 워크플로 복구 경로와 신규 툴킷을 위한 커버리지를 추가.

### ✅ 호환성
- 증분 변경으로 기존 API는 역호환을 유지.

## Version 1.2.5 (2025-10-20)

### ✨ 하이라이트
- 모델 제공자 8종 추가: Cohere, Together, OpenRouter, LM Studio, Vercel, Portkey, InternLM, SambaNova (동기/스트리밍, 함수 호출 지원)
- 평가 시스템(시나리오 평가, 지표 집계, 모델 간 비교), 미디어 처리(이미지 메타데이터; 오디오/비디오 프로브 자리표시자), 디버그 도구(요청/응답 덤프), 클라우드 배포 자리표시자(NoopDeployer)
- 통합 레지스트리(등록/목록/헬스체크), 공용 유틸(JSONPretty, Retry)

### 🔧 수정
- Airflow 도구 반환 형태를 Airflow REST API v2에 맞춤: `total_entries`, `dag_run_id`, `logical_date`
- 사이트 메인 이미지 누락 수정: `/logo.svg` → `/logo.png`

### 🧪 테스트
- 신규 모델/모듈에 대해 집중 단위 테스트 보강, 기존 벤치마크 유지

### ✅ 호환성
- 완전 역호환(증분 변경)

## 버전 1.2.1 (2025-10-15)

### 🧭 문서 재구성

- 명확한 분리:
  - `website/` → 구현된 대외 문서 (VitePress 사이트)
  - `docs/` → 설계 초안, 마이그레이션 계획, 태스크, 개발자/내부 문서
- 정책과 진입점을 담은 `docs/README.md` 추가
- 기여자 온보딩을 위한 `CONTRIBUTING.md` 추가

### 🔗 링크 수정

- README, CLAUDE, CHANGELOG, 릴리스 노트 링크를 `website/advanced/*`, `website/guide/*`로 정규화
- `docs/` 하위 중복 구현 문서로의 오래된 링크 제거

### 🌐 사이트 업데이트

- API: AgentOS 페이지에 지식 API 추가 (/api/agentos)
- Workflow History, Performance 페이지를 표준 참조로 통일

### ✅ 동작 변경

- 없음 (문서/구조 조정만 포함)

### ✨ 신규 (이번 릴리스에서 구현됨)

- A2A 스트리밍 이벤트 유형 필터 (SSE)
  - `POST /api/v1/agents/:id/run/stream?types=token,complete`
  - 요청한 이벤트만 출력; 표준 SSE 형식; Context 취소 지원
- AgentOS 컨텐츠 추출 미들웨어
  - JSON/Form의 `content/metadata/user_id/session_id`를 Context로 주입
  - `MaxRequestSize` 크기 보호 및 경로 스킵 지원
- Google Sheets 도구 (서비스 계정)
  - `read_range`, `write_range`, `append_rows`; JSON/파일 자격 증명 지원
- 최소 지식 적재 엔드포인트
  - `POST /api/v1/knowledge/content` 는 `text/plain` 및 `application/json` 지원

엔터프라이즈 검수 절차: [`docs/ENTERPRISE_MIGRATION_PLAN.md`](https://github.com/rexleimo/agno-Go/blob/main/docs/ENTERPRISE_MIGRATION_PLAN.md) 참고.

## 버전 1.1.0 (2025-10-08)

### 🎉 하이라이트

이번 릴리스는 프로덕션 준비 멀티 에이전트 시스템을 위한 강력한 새 기능을 제공합니다:

- **A2A 인터페이스** - 표준화된 에이전트 간 통신 프로토콜
- **세션 상태 관리** - 워크플로우 단계 간 영구 상태
- **멀티 테넌트 지원** - 단일 에이전트 인스턴스로 여러 사용자에게 서비스 제공
- **모델 타임아웃 구성** - LLM 호출을 위한 세밀한 타임아웃 제어

---

### ✨ 새로운 기능

#### A2A (Agent-to-Agent) 인터페이스

JSON-RPC 2.0 기반의 에이전트 간 상호작용을 위한 표준화된 통신 프로토콜.

**주요 기능:**
- RESTful API 엔드포인트 (`/a2a/message/send`, `/a2a/message/stream`)
- 멀티미디어 지원 (텍스트, 이미지, 파일, JSON 데이터)
- 스트리밍을 위한 Server-Sent Events (SSE)
- Python Agno A2A 구현과 호환

**빠른 예제:**
```go
import "github.com/rexleimo/agno-go/pkg/agentos/a2a"

// A2A 인터페이스 생성
a2a := a2a.New(a2a.Config{
    Agents: []a2a.Entity{myAgent},
    Prefix: "/a2a",
})

// 라우트 등록 (Gin)
router := gin.Default()
a2a.RegisterRoutes(router)
```

📚 **자세히 알아보기:** [A2A 인터페이스 문서](/ko/api/a2a)

---

#### 워크플로우 세션 상태 관리

워크플로우 단계 간 상태를 유지하기 위한 스레드 안전 세션 관리.

**주요 기능:**
- 단계 간 영구 상태 저장소
- `sync.RWMutex`를 사용한 스레드 안전성
- 병렬 브랜치 격리를 위한 딥 카피
- 데이터 손실을 방지하는 스마트 병합 전략
- Python Agno v2.1.2의 경쟁 조건 수정

**빠른 예제:**
```go
// 세션 정보로 컨텍스트 생성
execCtx := workflow.NewExecutionContextWithSession(
    "input",
    "session-123",  // 세션 ID
    "user-a",       // 사용자 ID
)

// 세션 상태에 액세스
execCtx.SetSessionState("key", "value")
value, _ := execCtx.GetSessionState("key")
```

📚 **자세히 알아보기:** [세션 상태 문서](/ko/guide/session-state)

---

#### 멀티 테넌트 지원

완전한 데이터 격리를 보장하면서 단일 Agent 인스턴스로 여러 사용자에게 서비스 제공.

**주요 기능:**
- 사용자별로 격리된 대화 기록
- Memory 인터페이스의 선택적 `userID` 매개변수
- 기존 코드와의 하위 호환성
- 스레드 안전 동시 작업
- 정리를 위한 `ClearAll()` 메서드

**빠른 예제:**
```go
// 멀티 테넌트 에이전트 생성
agent, _ := agent.New(&agent.Config{
    Name:   "customer-service",
    Model:  model,
    Memory: memory.NewInMemory(100),
})

// 사용자 A의 대화
agent.UserID = "user-a"
output, _ := agent.Run(ctx, "My name is Alice")

// 사용자 B의 대화
agent.UserID = "user-b"
output, _ := agent.Run(ctx, "My name is Bob")
```

📚 **자세히 알아보기:** [멀티 테넌트 문서](/ko/advanced/multi-tenant)

---

#### 모델 타임아웃 구성

세밀한 제어로 LLM 호출에 대한 요청 타임아웃 구성.

**주요 기능:**
- 기본값: 60초
- 범위: 1초에서 10분
- 지원 모델: OpenAI, Anthropic Claude
- 컨텍스트 인식 타임아웃 처리

**빠른 예제:**
```go
// 사용자 정의 타임아웃이 있는 OpenAI
model, _ := openai.New("gpt-4", openai.Config{
    APIKey:  apiKey,
    Timeout: 30 * time.Second,
})

// 사용자 정의 타임아웃이 있는 Claude
claude, _ := anthropic.New("claude-3-opus", anthropic.Config{
    APIKey:  apiKey,
    Timeout: 45 * time.Second,
})
```

📚 **자세히 알아보기:** [모델 구성](/ko/guide/models#timeout-configuration)

---

### 🐛 버그 수정

- **워크플로우 경쟁 조건** - 병렬 단계 실행 데이터 경쟁 수정
  - Python Agno v2.1.2에는 공유 `session_state` dict로 인한 덮어쓰기 문제가 있었습니다
  - Go 구현은 브랜치당 독립적인 SessionState 복제본을 사용합니다
  - 스마트 병합 전략으로 동시 실행에서 데이터 손실 방지

---

### 📚 문서

모든 새로운 기능에는 포괄적인 이중 언어 문서 (영어/중문)가 포함되어 있습니다:

- [A2A 인터페이스 가이드](/ko/api/a2a) - 완전한 프로토콜 사양
- [세션 상태 가이드](/ko/guide/session-state) - 워크플로우 상태 관리
- [멀티 테넌트 가이드](/ko/advanced/multi-tenant) - 데이터 격리 패턴
- [모델 구성](/ko/guide/models#timeout-configuration) - 타임아웃 설정

---

### 🧪 테스트

**새로운 테스트 스위트:**
- `session_state_test.go` - 세션 상태 테스트 543줄
- `memory_test.go` - 멀티 테넌트 메모리 테스트 (새 테스트 케이스 4개)
- `agent_test.go` - 멀티 테넌트 에이전트 테스트
- `openai_test.go` - 타임아웃 구성 테스트
- `anthropic_test.go` - 타임아웃 구성 테스트

**테스트 결과:**
- ✅ 모든 테스트가 `-race` 감지기로 통과
- ✅ 워크플로우 커버리지: 79.4%
- ✅ 메모리 커버리지: 93.1%
- ✅ 에이전트 커버리지: 74.7%

---

### 📊 성능

**성능 저하 없음** - 모든 벤치마크가 일관됩니다:
- Agent 인스턴스화: ~180ns/op (Python보다 16배 빠름)
- 메모리 사용량: ~1.2KB/에이전트
- 스레드 안전 동시 작업

---

### ⚠️ 호환성 문제

**없음.** 이번 릴리스는 v1.0.x와 완전히 하위 호환됩니다.

---

### 🔄 마이그레이션 가이드

**마이그레이션 불필요** - 모든 새 기능은 추가 기능이며 하위 호환됩니다.

**선택적 개선 사항:**

1. **멀티 테넌트 지원 활성화:**
   ```go
   // 에이전트 구성에 UserID 추가
   agent := agent.New(agent.Config{
       UserID: "user-123",  // NEW
       Memory: memory.NewInMemory(100),
   })
   ```

2. **워크플로우에서 세션 상태 사용:**
   ```go
   // 세션이 있는 컨텍스트 생성
   ctx := workflow.NewExecutionContextWithSession(
       "input",
       "session-id",
       "user-id",
   )
   ```

3. **모델 타임아웃 구성:**
   ```go
   // 모델 구성에 Timeout 추가
   model, _ := openai.New("gpt-4", openai.Config{
       APIKey:  apiKey,
       Timeout: 30 * time.Second,  // NEW
   })
   ```

---

### 📦 설치

```bash
go get github.com/rexleimo/agno-go@v1.1.0
```

---

### 🔗 링크

- **GitHub 릴리스:** [v1.1.0](https://github.com/rexleimo/agno-go/releases/tag/v1.1.0)
- **전체 변경 로그:** [CHANGELOG.md](https://github.com/rexleimo/agno-go/blob/main/CHANGELOG.md)
- **문서:** [https://agno-go.dev](https://agno-go.dev)

---

## 버전 1.0.3 (2025-10-06)

### 🧪 개선

- **JSON 직렬화 테스트 강화** - utils/serialize 패키지에서 100% 테스트 커버리지 달성
- **성능 벤치마크** - Python Agno 성능 테스트 패턴과 정렬
- **포괄적인 문서** - 이중 언어 패키지 문서 추가

---

## 버전 1.0.2 (2025-10-05)

### ✨ 추가

#### GLM (智谱AI) 프로바이더

- Zhipu AI의 GLM 모델과 완전 통합
- GLM-4, GLM-4V (비전), GLM-3-Turbo 지원
- 사용자 정의 JWT 인증 (HMAC-SHA256)
- 동기 및 스트리밍 API 호출
- 도구/함수 호출 지원

---

## 버전 1.0.0 (2025-10-02)

### 🎉 초기 릴리스

Agno-Go v1.0은 Agno 멀티 에이전트 프레임워크의 고성능 Go 구현입니다.

#### 핵심 기능
- **Agent** - 도구 지원이 있는 단일 자율 에이전트
- **Team** - 4가지 모드를 통한 멀티 에이전트 협력
- **Workflow** - 5가지 프리미티브를 통한 단계 기반 오케스트레이션

#### LLM 프로바이더
- OpenAI (GPT-4, GPT-3.5, GPT-4 Turbo)
- Anthropic (Claude 3.5 Sonnet, Claude 3 Opus/Sonnet/Haiku)
- Ollama (로컬 모델)

---

**최종 업데이트:** 2025-10-08
