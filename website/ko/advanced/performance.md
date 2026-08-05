# 성능

이 페이지는 고정된 로컬 dummy model을 사용한 framework overhead 측정만 공개합니다.
LLM 품질, 네트워크 지연시간 또는 운영 처리량 보장이 아닙니다. 원본 데이터는
`benchmarks/framework_comparison/results/latest.json`에 있습니다.

## 프레임워크 간 측정

기록: 2026-08-04 04:19:54 UTC

- Target: `windows/amd64`
- CPU: `12th Gen Intel(R) Core(TM) i5-12400F`
- Go: `go1.26.4`
- Python: `3.11.15`
- Packages: `agno==2.8.6`, `langgraph==1.2.10`
- Python 20회, Go 10회. 외부 Provider, 네트워크, DB, Token은 사용하지 않습니다.

### HNO와 Agno: Agent 생성

| Operation | Framework | Mean ns/op | Median ns/op | Range ns/op |
| --- | --- | ---: | ---: | ---: |
| Minimal Agent | HNO | 255.8 | 252.4 | 249.2-266.8 |
| Minimal Agent | Agno | 7,105.5 | 6,869.1 | 5,308.8-10,042.1 |
| Agent + 도구 1개 | HNO | 298.4 | 296.8 | 265.9-355.1 |
| Agent + 도구 1개 | Agno | 6,603.7 | 6,394.4 | 5,812.1-8,591.5 |

이 구성 작업에서 Agno/HNO 평균 비는 27.8배와 22.1배입니다. 이 수치는 해당
머신, 버전, 설정과 작업에만 해당하며 운영이나 종단 간 속도 비가 아닙니다.

### HNO와 Agno: fresh local dummy run

새 Agent를 만들고 고정된 응답으로 `ping`을 한 번 실행합니다.

| Framework | Mean ns/op | Median ns/op | Range ns/op |
| --- | ---: | ---: | ---: |
| HNO | 6,431.0 | 6,528.0 | 5,360.0-7,090.0 |
| Agno | 187,208.6 | 180,537.0 | 162,652.0-241,754.0 |

이 작업의 평균 비는 29.1배입니다. 실제 LLM, 네트워크, DB, Token 생성은 포함하지 않습니다.

### LangGraph: 별도 지표

LangGraph는 Agent 생성이 아니라 graph compile을 사용하므로 별도로 보고합니다. 최소
`StateGraph`는 평균 `356,598.2 ns/op`, 중앙값 `352,408.5 ns/op`, 범위
`332,839.0-394,720.0 ns/op`입니다. 이는 완전한 Agent 시스템의 속도 순위가 아닙니다.

## 재현

```bash
uv run --with 'agno==2.8.6' --with 'langgraph==1.2.10' \
  python benchmarks/framework_comparison/compare.py --repeat 20 --number 1000
```

프로토콜, raw JSON, source hash와 한계는 `benchmarks/framework_comparison/README.md`를
참조하세요. Go `B/op`와 Python `timeit`은 다른 측정 체계이며 메모리 비교로 사용하지 않습니다.

## 아직 측정하지 않은 항목

실제 LLM 지연, Token throughput, resident memory, Team/Workflow, 실제 서비스 Tool,
고정 동시성 RPS, 운영 용량과 비용은 측정하지 않았습니다. 동일 Provider, model, prompt,
Tool, output, timeout, concurrency, hardware와 version이 필요합니다.

## 왜 Go, 왜 HNO인가

Go는 컴파일된 배포물, 내장 동시성, 정적 타입, HTTP/JSON 표준 라이브러리와
테스트/profile/race 도구가 선택 이유인 구현 언어입니다. HNO는 현재 프로젝트 이름이며
공식 약어 확장은 저장소에 정의되어 있지 않습니다. Go module path는
`github.com/rexleimo/agno-go`로 유지됩니다.
