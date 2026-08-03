---
title: 멀티 테넌트 지원
description: 단일 에이전트 인스턴스로 여러 사용자에게 서비스를 제공하기 위한 멀티 테넌트 데이터 격리
outline: deep
---

# 멀티 테넌트 지원

**멀티 테넌트 지원**을 통해 Agno-Go는 단일 Agent 인스턴스로 여러 사용자에게 서비스를 제공하면서 사용자 간 대화 기록 및 세션 상태의 완전한 격리를 보장합니다.

---

## 개요

멀티 테넌트 아키텍처를 통해 단일 애플리케이션 인스턴스로 여러 사용자(테넌트)에게 완전한 데이터 격리를 제공할 수 있습니다:

```
                 ┌─────────────────┐
                 │ Agent Instance  │
                 └────────┬────────┘
                          │
         ┌────────────────┼────────────────┐
         ▼                ▼                ▼
   ┌──────────┐     ┌──────────┐     ┌──────────┐
   │ User A   │     │ User B   │     │ User C   │
   │ Messages │     │ Messages │     │ Messages │
   └──────────┘     └──────────┘     └──────────┘
```

---

## 멀티 테넌시란?

멀티 테넌시는 단일 애플리케이션 인스턴스가 여러 격리된 사용자 또는 조직에 서비스를 제공하는 아키텍처 패턴입니다. 각 테넌트의 데이터는 다른 테넌트로부터 완전히 분리됩니다.

### 멀티 테넌트 없이

```go
// ❌ 각 사용자에게 별도의 Agent 인스턴스가 필요
userAgents := make(map[string]*agent.Agent)

agent1, _ := agent.New(config)  // User 1
agent2, _ := agent.New(config)  // User 2
agent3, _ := agent.New(config)  // User 3
// ... 1000+ 사용자 = 1000+ Agent 인스턴스
```

**문제점:**
- 높은 메모리 사용량: 1000 사용자 = 1000 Agent 인스턴스
- 관리 어려움: 수동 에이전트 라이프사이클 관리
- 리소스 낭비: 각 에이전트에 중복된 구성

### 멀티 테넌트 사용

```go
// ✅ 단일 Agent 인스턴스가 모든 사용자에게 서비스 제공
sharedAgent, _ := agent.New(config)

// 다른 사용자는 다른 userID 사용
output1, _ := sharedAgent.Run(ctx, "user-1 input", "user-1")
output2, _ := sharedAgent.Run(ctx, "user-2 input", "user-2")
output3, _ := sharedAgent.Run(ctx, "user-3 input", "user-3")
```

**장점:**
- ✅ 낮은 메모리 사용량: 단일 Agent 인스턴스
- ✅ 쉬운 관리: 통합된 구성 및 업데이트
- ✅ 효율적인 리소스 활용: 공유 모델 및 도구

---

## 빠른 시작

### 1. 멀티 테넌트 Agent 생성

```go
package main

import (
    "context"
    "fmt"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/memory"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
)

func main() {
    // 모델 생성
    model, _ := openai.New("gpt-4", openai.Config{
        APIKey: "your-api-key",
    })

    // 멀티 테넌트 Memory 생성
    mem := memory.NewInMemory(100)  // 자동으로 멀티 테넌시 지원

    // Agent 생성
    myAgent, _ := agent.New(&agent.Config{
        Name:         "customer-service",
        Model:        model,
        Memory:       mem,
        Instructions: "You are a helpful customer service agent.",
    })

    // 다른 사용자의 대화
    ctx := context.Background()

    // 사용자 A의 대화
    myAgent.UserID = "user-a"
    output1, _ := myAgent.Run(ctx, "My name is Alice")
    fmt.Printf("User A: %s\n", output1.Content)

    output2, _ := myAgent.Run(ctx, "What's my name?")  // "Your name is Alice"
    fmt.Printf("User A: %s\n", output2.Content)

    // 사용자 B의 대화
    myAgent.UserID = "user-b"
    output3, _ := myAgent.Run(ctx, "My name is Bob")
    fmt.Printf("User B: %s\n", output3.Content)

    output4, _ := myAgent.Run(ctx, "What's my name?")  // "Your name is Bob"
    fmt.Printf("User B: %s\n", output4.Content)

    // 사용자 A가 다시 대화
    myAgent.UserID = "user-a"
    output5, _ := myAgent.Run(ctx, "What's my name?")  // "Your name is Alice"
    fmt.Printf("User A: %s\n", output5.Content)
}
```

### 2. Web API 예제

```go
package main

import (
    "context"
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/rexleimo/agno-go/pkg/hno/agent"
)

var sharedAgent *agent.Agent

func main() {
    // Agent 초기화
    sharedAgent, _ = agent.New(&agent.Config{
        Name:   "api-agent",
        Model:  model,
        Memory: memory.NewInMemory(100),
    })

    // 라우트 설정
    router := gin.Default()
    router.POST("/chat", handleChat)
    router.Run(":8080")
}

type ChatRequest struct {
    UserID  string `json:"user_id"`
    Message string `json:"message"`
}

type ChatResponse struct {
    UserID  string `json:"user_id"`
    Reply   string `json:"reply"`
}

func handleChat(c *gin.Context) {
    var req ChatRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // 현재 사용자 ID 설정
    sharedAgent.UserID = req.UserID

    // 대화 실행
    output, err := sharedAgent.Run(context.Background(), req.Message)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, ChatResponse{
        UserID: req.UserID,
        Reply:  output.Content,
    })
}
```

**테스트:**
```bash
# 사용자 A의 대화
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-a", "message": "My name is Alice"}'

curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-a", "message": "What is my name?"}'
# 응답: {"user_id":"user-a","reply":"Your name is Alice"}

# 사용자 B의 대화
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-b", "message": "My name is Bob"}'

curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-b", "message": "What is my name?"}'
# 응답: {"user_id":"user-b","reply":"Your name is Bob"}
```

---

## 메모리 관리

### Memory 인터페이스

Memory 인터페이스는 선택적 `userID` 매개변수를 지원합니다:

```go
// pkg/hno/memory/memory.go

type Memory interface {
    // 메시지 추가 (선택적 userID 지원)
    Add(message *types.Message, userID ...string)

    // 메시지 기록 가져오기 (선택적 userID 지원)
    GetMessages(userID ...string) []*types.Message

    // 특정 사용자의 메시지 지우기
    Clear(userID ...string)

    // 모든 사용자의 메시지 지우기
    ClearAll()

    // 특정 사용자의 메시지 수 가져오기
    Size(userID ...string) int
}
```

### InMemory 구현

```go
type InMemory struct {
    userMessages map[string][]*types.Message  // User ID → 메시지 목록
    maxSize      int
    mu           sync.RWMutex
}

// 기본 사용자 ID
const defaultUserID = "default"

// 사용자 ID 가져오기 (하위 호환성)
func getUserID(userID ...string) string {
    if len(userID) > 0 && userID[0] != "" {
        return userID[0]
    }
    return defaultUserID
}
```

### 사용 예제

#### 기본 사용법

```go
mem := memory.NewInMemory(100)

// 사용자 A의 메시지
mem.Add(types.NewUserMessage("Hello from Alice"), "user-a")
mem.Add(types.NewAssistantMessage("Hi Alice!"), "user-a")

// 사용자 B의 메시지
mem.Add(types.NewUserMessage("Hello from Bob"), "user-b")
mem.Add(types.NewAssistantMessage("Hi Bob!"), "user-b")

// 각 사용자의 메시지 가져오기
messagesA := mem.GetMessages("user-a")  // 2개 메시지
messagesB := mem.GetMessages("user-b")  // 2개 메시지

fmt.Printf("User A has %d messages\n", len(messagesA))  // 2
fmt.Printf("User B has %d messages\n", len(messagesB))  // 2
```

---

## Agent 통합

### Agent 구성

```go
type Agent struct {
    ID           string
    Name         string
    Model        models.Model
    Toolkits     []toolkit.Toolkit
    Memory       memory.Memory
    Instructions string
    MaxLoops     int

    UserID string  // ⭐ NEW: 멀티 테넌트 사용자 ID
}

type Config struct {
    Name         string
    Model        models.Model
    Toolkits     []toolkit.Toolkit
    Memory       memory.Memory
    Instructions string
    MaxLoops     int

    UserID string  // ⭐ NEW: 멀티 테넌트 사용자 ID
}
```

### Run 메서드 구현

```go
// pkg/hno/agent/agent.go

func (a *Agent) Run(ctx context.Context, input string) (*RunOutput, error) {
    // ...

    // 모든 Memory 호출은 UserID를 전달
    userMsg := types.NewUserMessage(input)
    a.Memory.Add(userMsg, a.UserID)  // ⭐ UserID 전달

    // ...

    messages := a.Memory.GetMessages(a.UserID)  // ⭐ UserID 전달

    // ...

    a.Memory.Add(types.NewAssistantMessage(content), a.UserID)  // ⭐ UserID 전달
}
```

---

## 데이터 격리 보장

### 1. 메모리 격리

```go
// 테스트: 멀티 테넌트 격리
mem := memory.NewInMemory(100)

// 사용자 A가 10개 메시지 추가
for i := 0; i < 10; i++ {
    mem.Add(types.NewUserMessage(fmt.Sprintf("User A message %d", i)), "user-a")
}

// 사용자 B가 5개 메시지 추가
for i := 0; i < 5; i++ {
    mem.Add(types.NewUserMessage(fmt.Sprintf("User B message %d", i)), "user-b")
}

// 격리 검증
assert.Equal(t, 10, mem.Size("user-a"))  // ✅
assert.Equal(t, 5, mem.Size("user-b"))   // ✅
assert.Equal(t, 0, mem.Size("user-c"))   // ✅ 존재하지 않는 사용자
```

### 2. 동시성 안전성

```go
// 테스트: 1000 동시 요청
mem := memory.NewInMemory(100)
var wg sync.WaitGroup

// 10명 사용자, 각각 100개 동시 요청
for userID := 0; userID < 10; userID++ {
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(uid, msgID int) {
            defer wg.Done()
            userIDStr := fmt.Sprintf("user-%d", uid)
            msg := types.NewUserMessage(fmt.Sprintf("Message %d", msgID))
            mem.Add(msg, userIDStr)
        }(userID, i)
    }
}

wg.Wait()

// 각 사용자가 올바른 메시지 수를 가지는지 검증
for userID := 0; userID < 10; userID++ {
    userIDStr := fmt.Sprintf("user-%d", userID)
    assert.Equal(t, 100, mem.Size(userIDStr))  // ✅
}
```

---

## 모범 사례

### 1. UserID 명명 규칙

```go
// ✅ 권장: 일관된 명명 규칙 사용
"user-{uuid}"           // user-123e4567-e89b-12d3-a456-426614174000
"org-{org_id}-user-{id}" // org-acme-user-001
"tenant-{id}"           // tenant-12345

// ❌ 피하기: 불안정한 식별자 사용
"{ip_address}"          // IP가 변경될 수 있음
"{session_id}"          // 세션이 만료됨
```

### 2. 오류 처리

```go
// UserID 검증
func validateUserID(userID string) error {
    if userID == "" {
        return fmt.Errorf("userID cannot be empty")
    }
    if len(userID) > 255 {
        return fmt.Errorf("userID too long (max 255 chars)")
    }
    return nil
}

// API 계층에서 검증
func handleChat(c *gin.Context) {
    var req ChatRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    if err := validateUserID(req.UserID); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // ...
}
```

---

## 문제 해결

### 일반적인 문제

#### 1. 사용자 데이터 혼동

**증상:** 사용자 A가 사용자 B의 메시지를 봄

**원인:** UserID가 제대로 전달되지 않음

**해결책:**
```go
// ❌ 잘못됨
agent.Run(ctx, input)  // UserID가 설정되지 않음

// ✅ 올바름
agent.UserID = "user-a"
agent.Run(ctx, input)
```

#### 2. 높은 메모리 사용량

**증상:** 메모리가 계속 증가

**원인:** 비활성 사용자 데이터가 정리되지 않음

**해결책:**
```go
// 주기적인 정리
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        cleanupInactiveUsers(mem, 24*time.Hour)
    }
}()
```

---

## 통합

### A2A Interface + 멀티 테넌트

```go
// A2A 요청에 contextID가 포함되어 userID로 사용 가능
type Message struct {
    MessageID string `json:"messageId"`
    Role      string `json:"role"`
    AgentID   string `json:"agentId"`
    ContextID string `json:"contextId"`  // ⭐ userID로 사용 가능
    Parts     []Part `json:"parts"`
}

// 매핑 중 UserID 설정
func MapA2ARequestToRunInput(req *JSONRPC2Request) (*RunInput, error) {
    // ...
    agent.UserID = req.Params.Message.ContextID  // ⭐ contextID를 userID로 사용
    // ...
}
```

---

## 관련 문서

- [A2A 인터페이스](/ko/api/a2a) - 에이전트 간 통신
- [세션 상태 관리](/ko/guide/session-state) - 워크플로우 세션 관리
- [메모리 가이드](/ko/guide/memory) - 메모리 사용 가이드

---

## 테스트

완전한 테스트 커버리지에는 다음이 포함됩니다:

- ✅ 멀티 사용자 데이터 격리
- ✅ 동시성 안전성 (1000 goroutines)
- ✅ Agent 통합 테스트
- ✅ 메모리 용량 관리

**테스트 커버리지:** 93.1% (메모리 모듈)

테스트 실행:
```bash
cd pkg/hno/memory
go test -v -run TestInMemory

cd pkg/hno/agent
go test -v -run TestAgent_MultiTenant
```

---

**최종 업데이트:** 2025-01-08
