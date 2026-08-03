---
layout: home

hero:
  name: "Agno-Go"
  text: "고성능 멀티 에이전트 프레임워크"
  tagline: "Python보다 16배 빠름 | 180ns 인스턴스화 | 에이전트당 1.2KB 메모리"
  image:
    src: /logo.png
    alt: Agno-Go
  actions:
    - theme: brand
      text: 시작하기
      link: /ko/guide/quick-start
    - theme: alt
      text: GitHub에서 보기
      link: https://github.com/rexleimo/agno-Go

features:
  - icon: 🚀
    title: 극한의 성능
    details: 에이전트 초기화는 약 180ns, 에이전트당 메모리는 약 1.2KB로 Python 런타임보다 16배 빠릅니다.

  - icon: 🤖
    title: 프로덕션 준비 AgentOS
    details: OpenAPI 3.0, 세션 스토리지, 헬스 체크, 구조화 로그, CORS, 요청 타임아웃과 함께 요약/재사용/히스토리 필터의 패리티 엔드포인트를 제공합니다.

  - icon: 🪄
    title: 세션 패리티
    details: 에이전트·팀 간에 세션을 공유하고 동기/비동기 요약을 실행하며 캐시 적중과 취소 사유를 기록, Python의 `stream_events` 스위치와 동일하게 동작합니다.

  - icon: 🧩
    title: 유연한 아키텍처
    details: 에이전트, 팀(4가지 협업 모드), 워크플로(5가지 제어 프리미티브)를 조합하여 기본값 상속과 체크포인트 재시작으로 결정론적으로 오케스트레이션합니다.

  - icon: 🔌
    title: 다중 모델 프로바이더
    details: OpenAI o-series, Anthropic Claude, Google Gemini, DeepSeek, GLM, ModelScope, Ollama, Cohere, Groq, Together, OpenRouter, LM Studio, Vercel, Portkey, InternLM, SambaNova를 지원합니다.

  - icon: 🔧
    title: 확장 가능한 도구
    details: 계산기, HTTP, 파일, 검색에 더해 Claude Agent Skills, Tavily Reader/Search, Gmail 읽음 처리, Jira 워크로그, ElevenLabs 음성, PPTX 리더, MCP 커넥터를 제공합니다.

  - icon: 💾
    title: 지식과 캐시
    details: ChromaDB 통합, 배치 유틸리티, 인제스트 도우미와 함께 동일 모델 호출을 중복 제거하는 응답 캐시를 제공합니다.

  - icon: 🛡️
    title: 가드레일과 관측성
    details: 프롬프트 인젝션 방어, 커스텀 프리/포스트 훅, 미디어 검증, SSE 추론 스트림, Logfire / OpenTelemetry 추적 샘플을 제공합니다.

  - icon: 📦
    title: 손쉬운 배포
    details: 단일 바이너리 또는 Docker / Compose / Kubernetes 매니페스트와 실전 배포 가이드로 즉시 배포할 수 있습니다.
---

## 빠른 예제

단 몇 줄의 코드로 도구를 갖춘 AI 에이전트 생성:

```go
package main

import (
    "context"
    "fmt"
    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
    "github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
)

func main() {
    // 모델 생성
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: "your-api-key",
    })

    // 도구를 갖춘 에이전트 생성
    ag, _ := agent.New(agent.Config{
        Name:     "수학 도우미",
        Model:    model,
        Toolkits: []toolkit.Toolkit{calculator.New()},
    })

    // 에이전트 실행
    output, _ := ag.Run(context.Background(), "25 * 4 + 15는 얼마인가요?")
    fmt.Println(output.Content) // 출력: 115
}
```

## 성능 비교

| 지표 | Python Agno | Agno-Go | 개선 |
|--------|-------------|---------|-------------|
| 에이전트 생성 | ~3μs | ~180ns | **16배 빠름** |
| 메모리/에이전트 | ~6.5KB | ~1.2KB | **5.4배 감소** |
| 동시성 | GIL 제한 | 네이티브 고루틴 | **무제한** |

## 왜 Agno-Go인가?

### 프로덕션을 위한 설계

Agno-Go는 단순한 프레임워크가 아닌 완전한 프로덕션 시스템입니다. 포함된 **AgentOS** 서버는 다음을 제공합니다:

- OpenAPI 3.0 사양을 갖춘 RESTful API
- 다중 턴 대화를 위한 세션 관리
- 스레드 안전한 에이전트 레지스트리
- 상태 모니터링 및 구조화된 로깅
- CORS 지원 및 요청 타임아웃 처리

### KISS 원칙

**Keep It Simple, Stupid** 철학 준수:

- **3개의 핵심 LLM 제공자**(45개 이상이 아님) - OpenAI, Anthropic, Ollama
- **필수 도구**(115개 이상이 아님) - 계산기, HTTP, 파일, 검색
- **양보다 질** - 프로덕션 준비 기능에 집중

### 개발자 경험

- **타입 안전**: Go의 강력한 타입 시스템으로 컴파일 시 오류 감지
- **빠른 빌드**: Go의 컴파일 속도로 신속한 반복 개발
- **손쉬운 배포**: 런타임 의존성이 없는 단일 바이너리
- **우수한 도구**: 내장된 테스트, 프로파일링, 경쟁 상태 감지

## 5분 안에 시작하기

```bash
# 저장소 복제
git clone https://github.com/rexleimo/agno-Go.git
cd agno-Go

# API 키 설정
export OPENAI_API_KEY=sk-your-key-here

# 예제 실행
go run cmd/examples/simple_agent/main.go

# 또는 AgentOS 서버 시작
docker-compose up -d
curl http://localhost:8080/health
```

## 포함된 내용

- **핵심 프레임워크**: Agent, Team(4가지 모드), Workflow(5가지 프리미티브)
- **모델**: OpenAI, Anthropic Claude, Ollama, DeepSeek, Gemini, ModelScope
- **도구**: Calculator(75.6%), HTTP(88.9%), File(76.2%), Search(92.1%)
- **RAG**: ChromaDB 통합 + OpenAI 임베딩
- **AgentOS**: 프로덕션 HTTP 서버(65.0% 커버리지)
- **예제**: 모든 기능을 다루는 6개의 실제 예제
- **문서**: 완전한 가이드, API 레퍼런스, 배포 지침

## 커뮤니티

- **GitHub**: [rexleimo/agno-Go](https://github.com/rexleimo/agno-Go)
- **Issues**: [버그 리포트 및 기능 요청](https://github.com/rexleimo/agno-Go/issues)
- **Discussions**: [질문 및 아이디어 공유](https://github.com/rexleimo/agno-Go/discussions)

## 라이센스

Agno-Go는 [MIT 라이센스](https://github.com/rexleimo/agno-Go/blob/main/LICENSE)로 배포됩니다.

[Agno (Python)](https://github.com/agno-agi/agno) 프레임워크에서 영감을 받았습니다.
