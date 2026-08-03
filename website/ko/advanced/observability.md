# 관측성(Observability) 및 텔레메트리

HNO는 런타임 동작을 파악할 수 있도록 SSE 이벤트와 OpenTelemetry를 제공합니다.

## AgentOS SSE 스트림

`POST /api/v1/agents/{id}/run/stream` 엔드포인트는 Server‑Sent Events (SSE)를 반환합니다. `types` 쿼리로 이벤트를 필터링할 수 있습니다. 예: `types=run_start,token,reasoning,complete`.

| 이벤트 | 설명 |
| --- | --- |
| `run_start` | 입력 페이로드와 세션 메타데이터 |
| `token` | 모델이 스트리밍으로 반환하는 토큰 |
| `tool_call` | 도구 실행 메타데이터(이름, 인자, 결과) |
| `reasoning` | 추론 스냅샷(내용, 토큰 수, 마스킹 텍스트 등) |
| `complete` | 최종 출력, 소요 시간, 집계된 사용량 |
| `error` | 구조화된 에러 객체 |

## Logfire 연동(OpenTelemetry)

샘플 `cmd/examples/logfire_observability`는 OpenTelemetry를 사용하여 Logfire로 트레이스를 전송합니다.

```bash
export OPENAI_API_KEY=sk-...
export LOGFIRE_WRITE_TOKEN=lf_...
go run -tags logfire cmd/examples/logfire_observability/main.go
```

자세한 단계별 가이드는 GitHub 문서를 참고하세요:

- https://github.com/rexleimo/HNO/blob/main/docs/release/logfire_observability.md

다음 단계:
- SSE 이벤트를 원하는 관측/모니터링 백엔드로 전달
- 추론 토큰 지표를 비용 대시보드와 결합
- OpenTelemetry 훅으로 도구 실행과 요청 지연 측정
