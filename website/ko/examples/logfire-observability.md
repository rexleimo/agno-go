# Logfire 관측성

OpenTelemetry로 HNO 에이전트를 계측하고 [Logfire](https://logfire.dev)로 스팬을 전송하는 예제입니다. 추론 내용, 토큰 사용량, 도구 실행을 관측 플랫폼에서 상호 연계할 수 있습니다.

## 실행

```bash
export OPENAI_API_KEY=sk-your-key
export LOGFIRE_WRITE_TOKEN=lf_your_token
go run -tags logfire cmd/examples/logfire_observability/main.go
```

> `logfire` 빌드 태그로 OpenTelemetry 의존성을 포함합니다.

## 무엇을 기록하나

1. OTLP/HTTP 익스포터 설정(로그파이어 쓰기 토큰)
2. 추론 지원 모델 실행
3. 런타임, 루프 횟수, 토큰 사용량 속성
4. `reasoning.complete` 이벤트(추론 텍스트와 토큰 수)

## 관련 문서

- 심화 가이드(GitHub): https://github.com/rexleimo/HNO/blob/main/docs/release/logfire_observability.md
- 개요: `/ko/advanced/observability`
