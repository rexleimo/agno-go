# 고급 주제

Agno-Go의 고급 개념, 성능 최적화, 배포 전략 및 테스트 모범 사례를 심도 있게 알아봅니다.

## 개요

이 섹션에서는 개발자를 위한 고급 주제를 다룹니다:

- 🏗️ **아키텍처 이해하기** - 핵심 설계 원칙과 패턴 학습
- ⚡ **성능 최적화하기** - 마이크로초 이하 에이전트 인스턴스화 달성
- 🚀 **프로덕션에 배포하기** - 프로덕션 배포 모범 사례
- 🧪 **효과적으로 테스트하기** - 포괄적인 테스트 전략과 도구

## 핵심 주제

### [아키텍처](/ko/advanced/architecture)

Agno-Go의 모듈식 아키텍처와 설계 철학에 대해 알아보기:

- 핵심 인터페이스 (Model, Toolkit, Memory)
- 추상화 패턴 (Agent, Team, Workflow)
- Go 동시성 모델 통합
- 오류 처리 전략
- 패키지 구성

**핵심 개념**: 클린 아키텍처, 의존성 주입, 인터페이스 설계

### [성능](/ko/advanced/performance)

성능 특성과 최적화 기법 이해하기:

- 에이전트 인스턴스화 (평균 ~180ns)
- 메모리 사용량 (에이전트당 ~1.2KB)
- 동시성과 병렬성
- 벤치마킹 도구와 방법론
- 다른 프레임워크와의 성능 비교

**핵심 지표**: 처리량, 지연 시간, 메모리 효율성, 확장성

### [배포](/ko/advanced/deployment)

프로덕션 배포 전략과 모범 사례:

- AgentOS HTTP 서버 설정
- 컨테이너 배포 (Docker, Kubernetes)
- 구성 관리
- 모니터링과 관찰 가능성
- 스케일링 전략
- 보안 고려사항

**핵심 기술**: Docker, Kubernetes, Prometheus, 분산 추적

### [테스트](/ko/advanced/testing)

멀티 에이전트 시스템을 위한 포괄적인 테스트 접근 방식:

- 단위 테스트 패턴
- 모의 객체를 사용한 통합 테스트
- 성능 벤치마킹
- 테스트 커버리지 요구사항 (>70%)
- CI/CD 통합
- 테스트 도구와 유틸리티

**핵심 도구**: Go testing, testify, benchmarking, 커버리지 보고서

## 빠른 링크

### 성능 벤치마크

```bash
# 모든 벤치마크 실행
make benchmark

# 특정 벤치마크 실행
go test -bench=BenchmarkAgentCreation -benchmem ./pkg/hno/agent/

# CPU 프로파일 생성
go test -bench=. -cpuprofile=cpu.out ./pkg/hno/agent/
```

[상세한 성능 지표 보기 →](/ko/advanced/performance)

### 프로덕션 배포

```bash
# AgentOS 서버 빌드
cd pkg/agentos && go build -o agentos

# Docker로 실행
docker build -t agno-go-agentos .
docker run -p 8080:8080 -e OPENAI_API_KEY=$OPENAI_API_KEY agno-go-agentos
```

[배포 가이드 보기 →](/ko/advanced/deployment)

### 벡터 인덱싱

```bash
# 벡터 컬렉션 생성/삭제 (기본 Chroma)
go run ./cmd/vectordb_migrate --action up --provider chroma --collection mycol \
  --chroma-url http://localhost:8000 --distance cosine

# Redis Provider (선택, -tags redis 필요)
go run -tags redis ./cmd/vectordb_migrate --action up --provider redis \
  --collection mycol --chroma-url localhost:6379
```

[벡터 인덱싱 보기 →](/ko/advanced/vector-indexing)

### 테스트 커버리지

패키지별 현재 테스트 커버리지:

| 패키지 | 커버리지 | 상태 |
|---------|----------|--------|
| types | 100.0% | ✅ 우수 |
| memory | 93.1% | ✅ 우수 |
| team | 92.3% | ✅ 우수 |
| toolkit | 91.7% | ✅ 우수 |
| workflow | 80.4% | ✅ 양호 |
| agent | 74.7% | ✅ 양호 |

[테스트 가이드 보기 →](/ko/advanced/testing)

## 설계 원칙

### KISS (Keep It Simple, Stupid)

Agno-Go는 단순함을 추구합니다:

- **집중된 범위**: 8+ 대신 3개의 LLM 제공자 (OpenAI, Anthropic, Ollama)
- **필수 도구**: 15+ 대신 5개의 핵심 도구
- **명확한 추상화**: Agent, Team, Workflow
- **최소 의존성**: 표준 라이브러리 우선

### 성능 우선

Go의 동시성 모델이 가능하게 하는 것:

- 병렬 실행을 위한 네이티브 고루틴 지원
- GIL (전역 인터프리터 락) 제한 없음
- 효율적인 메모리 관리
- 컴파일 타임 최적화

### 프로덕션 준비 완료

실제 배포를 위해 구축됨:

- 포괄적인 오류 처리
- 컨텍스트 인식 취소
- 구조화된 로깅
- OpenTelemetry 통합
- 헬스 체크와 메트릭

## 기여하기

Agno-Go에 기여하고 싶으신가요? 확인해보세요:

- [아키텍처 문서](/ko/advanced/architecture) - 코드베이스 이해하기
- [테스트 가이드](/ko/advanced/testing) - 테스트 표준 배우기
- [GitHub 저장소](https://github.com/rexleimo/agno-Go) - PR 제출하기
- [개발 가이드](https://github.com/rexleimo/agno-Go/blob/main/CLAUDE.md) - 개발 환경 설정

## 추가 리소스

### 문서

- [Go 패키지 문서](https://pkg.go.dev/github.com/rexleimo/agno-Go)
- [Python Agno 프레임워크](https://github.com/agno-agi/agno) (영감의 원천)
- [VitePress 문서 소스](https://github.com/rexleimo/agno-Go/tree/main/website)

### 커뮤니티

- [GitHub Issues](https://github.com/rexleimo/agno-Go/issues)
- [GitHub Discussions](https://github.com/rexleimo/agno-Go/discussions)
- [릴리스 노트](/ko/release-notes)

## 다음 단계

1. 📖 [아키텍처](/ko/advanced/architecture)로 핵심 설계 이해하기
2. ⚡ [성능](/ko/advanced/performance) 최적화 기법 배우기
3. 🚀 프로덕션을 위한 [배포](/ko/advanced/deployment) 전략 검토하기
4. 🧪 [테스트](/ko/advanced/testing) 모범 사례 마스터하기

---

**참고**: 이 섹션은 Agno-Go의 기본 개념에 익숙하다고 가정합니다. 처음이라면 [가이드](/ko/guide/) 섹션부터 시작하세요.
