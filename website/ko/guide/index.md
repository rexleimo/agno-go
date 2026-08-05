# HNO란?

**HNO**는 Go로 구축된 멀티 에이전트 시스템 프레임워크입니다. Go의 동시성 모델,
정적 타입, 배포 모델과 표준 도구를 사용합니다. 특정 부하의 성능 수치는 재현 가능한
benchmark가 있을 때만 공개합니다.

## 주요 기능

### 재현 가능한 성능 측정

- Agent 생성 benchmark와 환경은 [Performance](/ko/advanced/performance)에 기록합니다.
- benchmark는 `MockModel`을 사용한 프레임워크 할당 측정입니다.
- LLM 지연, 운영 처리량, Go와 Python의 배수 비교는 포함하지 않습니다.
- 애플리케이션에서는 Go goroutine을 사용할 수 있습니다.

### AgentOS

AgentOS는 다음 기능을 제공하는 HTTP 서버입니다.

- OpenAPI 3.0 REST API
- 세션 관리
- 스레드 안전 Agent 레지스트리
- 헬스 체크와 구조화된 로깅
- CORS와 요청 타임아웃

### 아키텍처

- **Agent**: Tool과 Memory를 사용하는 자율 에이전트
- **Team**: 여러 Agent의 협업
- **Workflow**: Step, Condition, Loop, Parallel, Router 기반 실행
- **Model**: 여러 LLM Provider를 공통 인터페이스로 사용
- **Tools / Memory / Storage**: 확장 가능한 구성요소

## 왜 Go인가

Go는 컴파일된 배포물, 내장 동시성, 정적 타입, HTTP/JSON 표준 라이브러리,
테스트/profile/race 도구가 선택 이유입니다. 특정 서비스에서 유리한지는 그 서비스의
부하로 측정해야 합니다.

## 왜 HNO인가

HNO는 현재 프로젝트 이름입니다. 저장소에는 공식 약어 확장이 정의되어 있지 않으므로
의미를 만들어내지 않습니다. Go module path는 `github.com/rexleimo/agno-go`로 유지되며,
HNO는 표준 모델이나 프로토콜이 아니라 프로젝트 브랜드입니다.

## Provider와 도구

현재 소스에는 여러 LLM Provider, Calculator, HTTP, File, 검색, MCP, Skills와 RAG 등이
있습니다. 지원 범위는 구현과 테스트를 기준으로 확인해야 합니다. 실제 외부 Provider 검증에는
인증 정보가 필요하며 테스트 응답을 운영 데이터로 설명하지 않습니다.

## 다음 단계

1. [Quick Start](/ko/guide/quick-start)
2. [Installation](/ko/guide/installation)
3. [Agent, Team, Workflow](/ko/guide/agent)
4. [Tools](/ko/guide/tools)
5. [Performance](/ko/advanced/performance)

## 라이선스

HNO는 [MIT License](https://github.com/rexleimo/HNO/blob/main/LICENSE)로 배포됩니다.

Agno Python 프로젝트에서 영감을 받았지만 HNO는 이 Go 프로젝트의 현재 이름입니다.
