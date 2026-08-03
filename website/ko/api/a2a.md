---
title: A2A 인터페이스
description: JSON-RPC 2.0 기반 표준화된 에이전트 간 통신
outline: deep
---

# Agent-to-Agent Interface (A2A)

## 개요

**A2A (Agent-to-Agent) 인터페이스**는 JSON-RPC 2.0 기반의 표준화된 에이전트 간 통신 프로토콜로, 동기 및 스트리밍 통신 모드를 지원합니다.

### 프로토콜 표준
- **JSON-RPC 2.0**: 업계 표준 RPC 프로토콜
- **Server-Sent Events (SSE)**: 스트리밍 응답 전송
- **RESTful HTTP**: HTTP 기반 엔드포인트 구현

### 핵심 컴포넌트

```
pkg/agentos/a2a/
├── types.go      # 프로토콜 타입 정의
├── validator.go  # 요청 검증
├── mapper.go     # 프로토콜 매핑
├── a2a.go        # A2A 인터페이스 관리
└── handlers.go   # HTTP 핸들러
```

## 빠른 시작

### 1. A2A 인터페이스 생성

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/rexleimo/agno-go/pkg/agentos/a2a"
    "github.com/rexleimo/agno-go/pkg/hno/agent"
)

func main() {
    // 에이전트 생성
    myAgent, _ := agent.New(&agent.Config{
        Name:  "my-agent",
        Model: model,
        // ... 기타 설정
    })

    // A2A 인터페이스 생성
    a2aInterface := a2a.New("/api/v1/a2a")

    // 엔티티로 에이전트 등록
    a2aInterface.RegisterEntity("my-agent", myAgent)

    // Gin 라우트 설정
    router := gin.Default()
    a2aInterface.SetupRoutes(router)

    router.Run(":8080")
}
```

### 2. 동기 메시지 전송

```bash
curl -X POST http://localhost:8080/api/v1/a2a/sendMessage \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "sendMessage",
    "id": "req-001",
    "params": {
      "message": {
        "messageId": "msg-001",
        "role": "user",
        "agentId": "my-agent",
        "contextId": "session-123",
        "parts": [
          {
            "type": "text",
            "content": "Hello, agent!"
          }
        ]
      }
    }
  }'
```

**응답 예시**:
```json
{
  "jsonrpc": "2.0",
  "id": "req-001",
  "result": {
    "task": {
      "taskId": "task-001",
      "status": "completed",
      "messages": [
        {
          "messageId": "msg-002",
          "role": "assistant",
          "agentId": "my-agent",
          "contextId": "session-123",
          "parts": [
            {
              "type": "text",
              "content": "Hello! How can I help you?"
            }
          ]
        }
      ]
    }
  }
}
```

### 3. 스트리밍 메시지 전송

```bash
curl -X POST http://localhost:8080/api/v1/a2a/streamMessage \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "method": "streamMessage",
    "id": "req-002",
    "params": {
      "message": {
        "messageId": "msg-003",
        "role": "user",
        "agentId": "my-agent",
        "contextId": "session-123",
        "parts": [
          {
            "type": "text",
            "content": "Tell me a story"
          }
        ]
      }
    }
  }'
```

**SSE 응답 스트림**:
```
data: {"type":"content","content":"Once"}

data: {"type":"content","content":" upon"}

data: {"type":"content","content":" a"}

data: {"type":"content","content":" time..."}

data: {"type":"done"}
```

## 프로토콜 세부사항

### JSON-RPC 2.0 요청 형식

```go
type JSONRPC2Request struct {
    JSONRPC string        `json:"jsonrpc"`  // "2.0"이어야 함
    Method  string        `json:"method"`   // "sendMessage" 또는 "streamMessage"
    ID      string        `json:"id"`       // 고유 요청 ID
    Params  RequestParams `json:"params"`   // 요청 매개변수
}

type RequestParams struct {
    Message Message `json:"message"`  // 메시지 내용
}
```

### 메시지 구조

```go
type Message struct {
    MessageID string `json:"messageId"`  // 고유 메시지 ID
    Role      string `json:"role"`       // "user" 또는 "assistant"
    AgentID   string `json:"agentId"`    // 대상 에이전트 ID
    ContextID string `json:"contextId"`  // 세션 컨텍스트 ID
    Parts     []Part `json:"parts"`      // 메시지 파트
}

type Part struct {
    Type    string `json:"type"`              // "text" 또는 "data"
    Content string `json:"content,omitempty"` // 텍스트 내용
    Data    string `json:"data,omitempty"`    // 구조화된 데이터 (JSON)
}
```

### 응답 형식

#### 성공 응답

```go
type JSONRPC2Response struct {
    JSONRPC string        `json:"jsonrpc"`  // "2.0"
    ID      string        `json:"id"`       // 일치하는 요청 ID
    Result  *ResultObject `json:"result"`   // 결과 객체
}

type ResultObject struct {
    Task Task `json:"task"`  // 작업 정보
}

type Task struct {
    TaskID   string    `json:"taskId"`   // 작업 ID
    Status   string    `json:"status"`   // "completed" 또는 "failed"
    Messages []Message `json:"messages"` // 응답 메시지
}
```

#### 오류 응답

```go
type JSONRPC2Response struct {
    JSONRPC string       `json:"jsonrpc"`
    ID      string       `json:"id"`
    Error   *ErrorObject `json:"error"`
}

type ErrorObject struct {
    Code    int    `json:"code"`    // 오류 코드
    Message string `json:"message"` // 오류 메시지
}
```

**표준 오류 코드**:
- `-32700`: Parse error (JSON 파싱 실패)
- `-32600`: Invalid Request (잘못된 요청 형식)
- `-32601`: Method not found (메서드가 존재하지 않음)
- `-32602`: Invalid params (잘못된 매개변수)
- `-32603`: Internal error (내부 서버 오류)

## 검증 메커니즘

### 요청 검증

A2A 인터페이스는 완전한 요청 검증을 제공합니다:

```go
func ValidateRequest(req *JSONRPC2Request) error {
    // 1. JSON-RPC 버전 확인
    if req.JSONRPC != "2.0" {
        return fmt.Errorf("invalid jsonrpc version, must be 2.0")
    }

    // 2. 메서드 확인
    if req.Method != "sendMessage" && req.Method != "streamMessage" {
        return fmt.Errorf("invalid method, must be sendMessage or streamMessage")
    }

    // 3. 요청 ID 확인
    if req.ID == "" {
        return fmt.Errorf("request id is required")
    }

    // 4. 메시지 검증
    return ValidateMessage(&req.Params.Message)
}
```

## 완전한 예제

### 서버 사이드

```go
package main

import (
    "context"
    "log"

    "github.com/gin-gonic/gin"
    "github.com/rexleimo/agno-go/pkg/agentos/a2a"
    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
)

func main() {
    // 1. 모델 생성
    model, err := openai.New("gpt-4", openai.Config{
        APIKey: "your-api-key",
    })
    if err != nil {
        log.Fatal(err)
    }

    // 2. 에이전트 생성
    myAgent, err := agent.New(&agent.Config{
        Name:         "customer-service",
        Model:        model,
        Instructions: "You are a helpful customer service agent.",
    })
    if err != nil {
        log.Fatal(err)
    }

    // 3. A2A 인터페이스 생성
    a2aInterface := a2a.New("/api/v1/a2a")
    a2aInterface.RegisterEntity("customer-service", myAgent)

    // 4. 라우트 설정
    router := gin.Default()
    a2aInterface.SetupRoutes(router)

    // 5. 서버 시작
    log.Println("A2A server listening on :8080")
    router.Run(":8080")
}
```

### 클라이언트 사이드 (Go)

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"

    "github.com/rexleimo/agno-go/pkg/agentos/a2a"
)

func main() {
    // 요청 구성
    request := &a2a.JSONRPC2Request{
        JSONRPC: "2.0",
        Method:  "sendMessage",
        ID:      "req-001",
        Params: a2a.RequestParams{
            Message: a2a.Message{
                MessageID: "msg-001",
                Role:      "user",
                AgentID:   "customer-service",
                ContextID: "session-123",
                Parts: []a2a.Part{
                    {
                        Type:    "text",
                        Content: "How do I return a product?",
                    },
                },
            },
        },
    }

    // 직렬화
    requestBody, _ := json.Marshal(request)

    // 요청 전송
    resp, err := http.Post(
        "http://localhost:8080/api/v1/a2a/sendMessage",
        "application/json",
        bytes.NewBuffer(requestBody),
    )
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    // 응답 읽기
    body, _ := io.ReadAll(resp.Body)

    var response a2a.JSONRPC2Response
    json.Unmarshal(body, &response)

    // 결과 처리
    if response.Error != nil {
        fmt.Printf("Error: %s\n", response.Error.Message)
        return
    }

    task := response.Result.Task
    fmt.Printf("Task Status: %s\n", task.Status)
    for _, msg := range task.Messages {
        for _, part := range msg.Parts {
            fmt.Printf("Agent Response: %s\n", part.Content)
        }
    }
}
```

## 모범 사례

### 1. 오류 처리

```go
// 표준 오류 코드 사용
if err := validateInput(); err != nil {
    return &a2a.ErrorObject{
        Code:    -32602, // Invalid params
        Message: err.Error(),
    }
}
```

### 2. ContextID 관리

```go
// 각 세션에 고유한 contextId 사용
contextID := fmt.Sprintf("session-%s-%d", userID, time.Now().Unix())

// 같은 세션의 모든 메시지는 동일한 contextId 사용
message1.ContextID = contextID
message2.ContextID = contextID
```

### 3. 동시 처리

```go
// A2A 인터페이스는 동시성에 안전합니다
// 여러 요청을 동시에 처리할 수 있습니다

for i := 0; i < 10; i++ {
    go func(id int) {
        // 동시에 요청 전송
        sendMessageToAgent(fmt.Sprintf("req-%d", id))
    }(i)
}
```

### 4. 타임아웃 제어

```go
// 요청 타임아웃 설정
client := &http.Client{
    Timeout: 30 * time.Second,
}

resp, err := client.Post(url, contentType, body)
```

## 문제 해결

### 일반적인 문제

#### 1. "Invalid JSON-RPC version"

**원인**: `jsonrpc` 필드가 "2.0"이 아님

**해결책**:
```json
{
  "jsonrpc": "2.0",  // 문자열 "2.0"이어야 함
  "method": "sendMessage",
  ...
}
```

#### 2. "Agent not found"

**원인**: `agentId`가 등록되지 않음

**해결책**:
```go
// 등록된 엔티티 확인
entities := a2aInterface.ListEntities()
fmt.Println(entities)

// 에이전트가 등록되어 있는지 확인
a2aInterface.RegisterEntity("your-agent-id", agent)
```

#### 3. "Invalid message format"

**원인**: 메시지에 필수 필드가 없음

**해결책**:
```json
{
  "messageId": "msg-001",     // ✅ 필수
  "role": "user",             // ✅ 필수
  "agentId": "my-agent",      // ✅ 필수
  "contextId": "session-123", // ⚠️ 선택 사항이지만 권장
  "parts": [                  // ✅ 필수, 최소 1개
    {
      "type": "text",
      "content": "Hello"
    }
  ]
}
```

## 관련 문서

- [세션 상태 관리](/ko/guide/session-state) - 세션 상태 관리
- [멀티 테넌트 지원](/ko/advanced/multi-tenant) - 멀티 테넌트 지원
- [아키텍처 설계](/ko/architecture) - 아키텍처 설계

---

**최종 업데이트**: 2025-01-XX
