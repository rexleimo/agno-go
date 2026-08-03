# HNO란 무엇인가요?

**HNO**는 Go로 구축된 고성능 멀티 에이전트 시스템 프레임워크입니다. Python Agno 프레임워크의 설계 철학을 계승하여 Go의 동시성 모델과 성능 이점을 활용하여 효율적이고 확장 가능한 AI 에이전트 시스템을 구축합니다.

## 주요 기능

### 🚀 극한의 성능

- **Agent 인스턴스화**: 평균 ~180ns (Python 버전보다 16배 빠름)
- **메모리 사용량**: 에이전트당 ~1.2KB (Python 대비 5.4배 감소)
- **네이티브 동시성**: GIL 제한 없는 완전한 goroutine 지원

### 🤖 프로덕션 준비 완료

HNO는 프로덕션 HTTP 서버인 **AgentOS**를 포함합니다:

- OpenAPI 3.0 사양을 갖춘 RESTful API
- 다중 턴 대화를 위한 세션 관리
- 스레드 안전 에이전트 레지스트리
- 헬스 모니터링 및 구조화된 로깅
- CORS 지원 및 요청 타임아웃 처리

### 🧩 유연한 아키텍처

다양한 사용 사례를 위한 세 가지 핵심 추상화:

1. **Agent** - 도구 지원 및 메모리를 갖춘 자율 AI 에이전트
2. **Team** - 4가지 협업 모드를 갖춘 멀티 에이전트 협업
   - Sequential, Parallel, Leader-Follower, Consensus
3. **Workflow** - 5가지 기본 요소를 갖춘 단계 기반 오케스트레이션
   - Step, Condition, Loop, Parallel, Router

### 🔌 다중 모델 지원

6개 주요 LLM 제공업체에 대한 내장 지원:

- **OpenAI** - GPT-4, GPT-3.5 Turbo 등
- **Anthropic** - Claude 3.5 Sonnet, Claude 3 Opus/Sonnet/Haiku
- **Ollama** - 로컬 모델 (Llama 3, Mistral, CodeLlama 등)
- **DeepSeek** - DeepSeek-V2, DeepSeek-Coder
- **Google Gemini** - Gemini Pro, Flash
- **ModelScope** - Qwen, Yi 모델

### 🔧 확장 가능한 도구

KISS 원칙을 따라 높은 품질의 필수 도구를 제공합니다:

- **Calculator** - 기본 수학 연산 (테스트 커버리지 75.6%)
- **HTTP** - HTTP GET/POST 요청 (88.9% 커버리지)
- **File Operations** - 보안 제어를 갖춘 읽기, 쓰기, 목록, 삭제 (76.2% 커버리지)
- **Search** - DuckDuckGo 웹 검색 (92.1% 커버리지)

커스텀 도구 생성이 쉽습니다 - [Tools Guide](/guide/tools)를 참조하세요.

### 💾 RAG 및 지식

지식 베이스를 갖춘 지능형 에이전트 구축:

- **ChromaDB** - 벡터 데이터베이스 통합
- **OpenAI Embeddings** - text-embedding-3-small/large 지원
- 자동 임베딩 생성 및 의미 검색

전체 예제는 [RAG Demo](/examples/rag-demo)를 참조하세요.

## 설계 철학

### KISS 원칙

**Keep It Simple, Stupid** - 수량보다 품질에 집중:

- **3개의 핵심 LLM 제공업체** (45개 이상이 아님)
- **필수 도구** (115개 이상이 아님)
- **1개의 벡터 데이터베이스** (15개 이상이 아님)

이러한 집중 접근 방식은 다음을 보장합니다:
- 더 나은 코드 품질
- 더 쉬운 유지보수
- 프로덕션 준비 기능

### Go의 장점

멀티 에이전트 시스템을 Go로 구축하는 이유는?

1. **성능** - 컴파일 언어, 빠른 실행
2. **동시성** - 네이티브 goroutine, GIL 없음
3. **타입 안전성** - 컴파일 시 오류 발견
4. **단일 바이너리** - 쉬운 배포, 런타임 의존성 없음
5. **뛰어난 도구** - 내장 테스트, 프로파일링, 레이스 감지

## 사용 사례

HNO는 다음에 완벽합니다:

- **프로덕션 AI 애플리케이션** - AgentOS HTTP 서버로 배포
- **멀티 에이전트 시스템** - 여러 AI 에이전트 조정
- **고성능 워크플로우** - 수천 개의 요청 처리
- **로컬 AI 개발** - 프라이버시 중심 애플리케이션을 위한 Ollama 사용
- **RAG 애플리케이션** - 지식 기반 AI 어시스턴트 구축

## 품질 지표

- **테스트 커버리지**: 핵심 패키지에서 평균 80.8%
- **테스트 케이스**: 85개 이상의 테스트, 100% 통과율
- **문서**: 완전한 가이드, API 레퍼런스, 예제
- **프로덕션 준비**: Docker, K8s 매니페스트, 배포 가이드

## 다음 단계

시작할 준비가 되셨나요?

1. [Quick Start](/guide/quick-start) - 5분 안에 첫 번째 에이전트 구축
2. [Installation](/guide/installation) - 자세한 설정 지침
3. [Core Concepts](/guide/agent) - Agent, Team, Workflow에 대해 배우기

## 빠른 링크

- 임베딩: [OpenAI/VLLM 사용법](/ko/guide/embeddings)
- 벡터 인덱싱: [Chroma + Redis(선택) + 마이그레이션 CLI](/ko/advanced/vector-indexing)

## 커뮤니티

- **GitHub**: [rexleimo/HNO](https://github.com/rexleimo/HNO)
- **Issues**: [버그 신고](https://github.com/rexleimo/HNO/issues)
- **Discussions**: [질문하기](https://github.com/rexleimo/HNO/discussions)

## 라이선스

HNO는 [MIT License](https://github.com/rexleimo/HNO/blob/main/LICENSE) 하에 릴리스되었습니다.

[Agno (Python)](https://github.com/agno-agi/agno) 프레임워크에서 영감을 받았습니다.
