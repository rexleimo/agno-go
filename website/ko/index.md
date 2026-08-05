---
layout: home

hero:
  name: "HNO"
  text: "Go 네이티브 멀티 에이전트 프레임워크"
  tagline: "검증되지 않은 추정이 아니라 명시적이고 재현 가능한 증거로 설명하는 Go 구현."
  actions:
    - theme: brand
      text: 시작하기
      link: /ko/guide/quick-start
    - theme: alt
      text: GitHub에서 보기
      link: https://github.com/rexleimo/agno-go

features:
  - title: 추측이 아닌 측정
    details: Agent 생성 Go benchmark를 포함하며 Performance 페이지에 명령, 환경, 범위와 한계를 기록합니다.

  - title: Provider 어댑터
    details: Provider 구현은 Model 인터페이스 뒤에 있습니다. 현재 소스에는 최상위 Provider 패키지 17개가 있지만 호환성이나 지연시간 보장은 아닙니다.

  - title: 공유 오케스트레이션
    details: Agent, Team, Workflow는 Go 소스의 실행 구성요소와 Tool dispatch 추상화를 공유합니다.

  - title: 관측성 통합
    details: OpenTelemetry와 구조화된 런타임 계측을 사용할 수 있습니다. 계측 비용은 구성에 따라 달라지므로 대상 서비스에서 측정해야 합니다.

  - title: Skills·MCP·메모리
    details: Agent Skills, MCP 브리지, 플러그형 메모리와 세션 스토리지를 선택적 기능으로 제공합니다.

  - title: 정확한 프로토콜 설명
    details: 자동 테스트는 어댑터와 request/response 매핑을 다룹니다. 실제 Provider 검증에는 인증 정보가 필요하며 테스트 응답을 운영 증거로 설명하지 않습니다.

---

## 프레임워크 간 benchmark 스냅샷

이는 통제된 로컬 benchmark이며 운영 환경이나 LLM benchmark가 아닙니다.

| 작업 | HNO 평균 | Agno 평균 | HNO 중앙값 | Agno 중앙값 |
| --- | ---: | ---: | ---: | ---: |
| Agent 생성 | 255.8 ns/op | 7,105.5 ns/op | 252.4 | 6,869.1 |
| Agent 생성(도구 1개) | 298.4 ns/op | 6,603.7 ns/op | 296.8 | 6,394.4 |
| 새 Agent + 로컬 dummy run | 6,431.0 ns/op | 187,208.6 ns/op | 6,528.0 | 180,537.0 |

환경은 Windows amd64, 12th Gen Intel Core i5-12400F, Go 1.26.4, Python 3.11.15,
`agno==2.8.6`, `langgraph==1.2.10`입니다. Python 20회, Go 10회 benchmark를
실행했고 모두 고정된 로컬 dummy 응답을 사용합니다. 구성 작업의 Agno/HNO 평균 비는
27.8배와 22.1배, fresh run은 29.1배였습니다. 이 비율은 이 부하에만 해당하며
종단 간 또는 운영 서비스 속도 비가 아닙니다.

LangGraph는 Agent 객체 생성이 아니라 그래프 컴파일을 사용하므로 별도로 보고합니다.
최소 `StateGraph` 결과는 평균 `356,598.2 ns/op`, 중앙값 `352,408.5 ns/op`,
범위 `332,839.0-394,720.0 ns/op`입니다.

원본 샘플, 소스 해시, 버전과 명령은
[`benchmarks/framework_comparison/`](https://github.com/rexleimo/agno-go/tree/main/benchmarks/framework_comparison)에 있습니다.

```bash
uv run --with 'agno==2.8.6' --with 'langgraph==1.2.10' \
  python benchmarks/framework_comparison/compare.py --repeat 20 --number 1000
```

## 왜 Go, 왜 HNO인가

**왜 Go인가:** 컴파일된 배포물, 내장 동시성, 정적 타입, HTTP/JSON 표준 라이브러리,
기본 제공 테스트와 프로파일 도구가 구현상의 이유입니다. 특정 부하에서의 이점은 그
부하로 측정해야 합니다.

**왜 HNO인가:** HNO는 현재 프로젝트 이름입니다. 저장소에 공식 약어 확장이 정의되어
있지 않으므로 의미를 만들어내지 않습니다. Go module path는
`github.com/rexleimo/agno-go`로 유지되며 HNO는 표준 모델이나 프로토콜이 아니라
프로젝트 브랜드입니다.

**증거 정책:** 평균, 중앙값, 범위, 환경, 버전과 명령을 함께 기록합니다. Go 할당 바이트를
Python 메모리 값으로 취급하지 않습니다. 실제 LLM과 운영 용량 비교에는 동일 Provider와
동일 workload가 필요합니다.
