---
title: 실행 컨텍스트
description: hooks, 도구, 모델 호출 전반에 run_context_id를 전파하고 이벤트에서 상관합니다.
---

# 실행 컨텍스트 (Run Context)

HNO는 실행 단위 컨텍스트 식별자를 전파하며, SSE 이벤트에 `run_context_id`를 포함해 엔드투엔드 상관을 지원합니다.

## SSE 이벤트

`POST /api/v1/agents/{id}/run?stream_events=true`는 `run_start`·`reasoning`·`token`·`complete`·`error`를 출력하며 모두 `run_context_id`를 포함합니다.

## 코드 예시

```go
ctx := agent.WithRunContext(context.Background(), "rc-123")
out, err := myAgent.Run(ctx, "hello")
```

Hook / Toolkit에서 읽기:

```go
id, _ := agent.RunContextID(ctx)
```

## 참고

- 명시하지 않으면 HTTP 레이어가 실행마다 자동으로 주입합니다.
- 취소/타임아웃은 Context를 통해 hooks·tools·모델 호출로 전파됩니다.

