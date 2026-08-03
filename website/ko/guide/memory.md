# Memory - 대화 히스토리

에이전트를 위한 대화 히스토리 및 컨텍스트를 관리하세요.

---

## Memory란?

Memory는 대화 히스토리를 저장하여, 에이전트가 여러 상호작용에 걸쳐 컨텍스트를 유지할 수 있게 합니다. HNO는 자동 절삭 기능이 있는 내장 메모리 관리를 제공합니다.

### 주요 기능

- **자동 히스토리**: 대화가 자동으로 저장됨
- **구성 가능한 제한**: 메모리 크기 제어
- **메시지 타입**: System, User, Assistant, Tool 메시지
- **수동 제어**: 프로그래밍 방식으로 메모리 지우기 또는 관리

---

## 기본 사용법

### 기본 메모리

에이전트는 기본적으로 메모리가 활성화되어 있습니다:

```go
package main

import (
    "context"
    "fmt"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: "your-api-key",
    })

    agent, _ := agent.New(agent.Config{
        Model: model,
    })

    // 첫 번째 상호작용
    agent.Run(context.Background(), "My name is Alice")
    // 응답: "Nice to meet you, Alice!"

    // 두 번째 상호작용 - 에이전트가 기억함
    output, _ := agent.Run(context.Background(), "What's my name?")
    fmt.Println(output.Content)
    // 응답: "Your name is Alice."
}
```

---

## 구성

### 커스텀 메모리 제한

저장할 최대 메시지 수 설정:

```go
import "github.com/rexleimo/agno-go/pkg/hno/memory"

customMemory := memory.New(memory.Config{
    MaxMessages: 50,  // 최대 50개 메시지 저장
})

agent, _ := agent.New(agent.Config{
    Model:  model,
    Memory: customMemory,
})
```

### 메모리 없음

상태 비저장 에이전트를 위한 메모리 비활성화:

```go
agent, _ := agent.New(agent.Config{
    Model:  model,
    Memory: nil,  // 대화 히스토리 없음
})
```

---

## 메모리 작업

### 메모리 지우기

대화 히스토리 재설정:

```go
// 모든 히스토리 지우기
agent.ClearMemory()

// 새로운 대화 시작
agent.Run(ctx, "New conversation")
```

### 메시지 히스토리 가져오기

저장된 메시지 액세스:

```go
messages := agent.Memory.GetMessages()
for _, msg := range messages {
    fmt.Printf("%s: %s\n", msg.Role, msg.Content)
}
```

### 커스텀 메시지 추가

메모리에 수동으로 메시지 추가:

```go
import "github.com/rexleimo/agno-go/pkg/hno/types"

// 시스템 메시지 추가
agent.Memory.AddMessage(types.Message{
    Role:    types.RoleSystem,
    Content: "You are a helpful assistant.",
})

// 사용자 메시지 추가
agent.Memory.AddMessage(types.Message{
    Role:    types.RoleUser,
    Content: "Hello!",
})
```

---

## 메시지 타입

### System 메시지

에이전트를 위한 지침:

```go
types.Message{
    Role:    types.RoleSystem,
    Content: "You are a Python expert. Help with coding questions.",
}
```

### User 메시지

사용자 입력:

```go
types.Message{
    Role:    types.RoleUser,
    Content: "How do I read a file in Python?",
}
```

### Assistant 메시지

에이전트 응답:

```go
types.Message{
    Role:    types.RoleAssistant,
    Content: "Use the open() function: open('file.txt', 'r')",
}
```

### Tool 메시지

도구 실행 결과:

```go
types.Message{
    Role:       types.RoleTool,
    Content:    "Result: 42",
    ToolCallID: "call_123",
}
```

---

## 메모리 패턴

### 세션 기반 메모리

세션 간 메모리 지우기:

```go
func handleSession(agent *agent.Agent, sessionID string) {
    // 세션 히스토리 로드 (데이터베이스 등에서)
    loadSessionHistory(agent, sessionID)

    // 대화 처리
    output, _ := agent.Run(ctx, userInput)

    // 세션 히스토리 저장
    saveSessionHistory(agent, sessionID)

    // 정리
    agent.ClearMemory()
}
```

### 슬라이딩 윈도우

최근 메시지만 유지:

```go
memory := memory.New(memory.Config{
    MaxMessages: 20,  // 최근 20개 메시지 유지
})

// 오래된 메시지 자동 절삭
agent, _ := agent.New(agent.Config{
    Memory: memory,
})
```

### 영구 메모리

대화 저장 및 복원:

```go
// 대화 저장
messages := agent.Memory.GetMessages()
saveToDatabase(sessionID, messages)

// 대화 복원
savedMessages := loadFromDatabase(sessionID)
for _, msg := range savedMessages {
    agent.Memory.AddMessage(msg)
}
```

---

## 고급 사용법

### 멀티 에이전트 메모리 공유

에이전트 간 컨텍스트 공유:

```go
// 공유 메모리 생성
sharedMemory := memory.New(memory.Config{
    MaxMessages: 100,
})

// 두 에이전트 모두 동일한 메모리 사용
agent1, _ := agent.New(agent.Config{
    Name:   "Agent1",
    Model:  model,
    Memory: sharedMemory,
})

agent2, _ := agent.New(agent.Config{
    Name:   "Agent2",
    Model:  model,
    Memory: sharedMemory,
})

// Agent1 대화가 Agent2에게 보임
agent1.Run(ctx, "Store this information: X=42")
output, _ := agent2.Run(ctx, "What is X?")
// Agent2가 Agent1의 대화를 볼 수 있음
```

### 조건부 메모리

조건에 따라 메모리 지우기:

```go
messageCount := len(agent.Memory.GetMessages())

if messageCount > 100 {
    // 시스템 메시지만 유지
    systemMsg := agent.Memory.GetMessages()[0]
    agent.ClearMemory()
    agent.Memory.AddMessage(systemMsg)
}
```

### 메모리 검사

대화 히스토리 분석:

```go
messages := agent.Memory.GetMessages()

var userMessages, assistantMessages int
for _, msg := range messages {
    switch msg.Role {
    case types.RoleUser:
        userMessages++
    case types.RoleAssistant:
        assistantMessages++
    }
}

fmt.Printf("User messages: %d, Assistant messages: %d\n",
    userMessages, assistantMessages)
```

---

## 메모리 구성

### Config 구조체

```go
type Config struct {
    MaxMessages int // 저장할 최대 메시지 수 (기본값: 100)
}
```

### 기본 동작

- 모든 대화 메시지 자동 저장
- 제한에 도달하면 가장 오래된 메시지 절삭
- 절삭 중 시스템 메시지 보존

---

## 모범 사례

### 1. 적절한 제한 설정

컨텍스트와 성능의 균형:

```go
// 짧은 대화
memory := memory.New(memory.Config{MaxMessages: 20})

// 긴 대화
memory := memory.New(memory.Config{MaxMessages: 100})

// 매우 긴 컨텍스트
memory := memory.New(memory.Config{MaxMessages: 500})
```

### 2. 전략적으로 메모리 지우기

컨텍스트 변경 시 재설정:

```go
// 새로운 주제
if isNewTopic(userInput) {
    agent.ClearMemory()
}

// 새로운 세션
if isNewSession(sessionID) {
    agent.ClearMemory()
}
```

### 3. 메모리 사용량 모니터링

대화 길이 추적:

```go
messages := agent.Memory.GetMessages()
if len(messages) > 80 {
    log.Printf("Warning: Approaching memory limit (%d/100)", len(messages))
}
```

### 4. 중요한 컨텍스트 보존

시스템 지침 유지:

```go
// 시스템 메시지 저장
systemMsg := agent.Memory.GetMessages()[0]

// 메모리 지우기
agent.ClearMemory()

// 시스템 메시지 복원
agent.Memory.AddMessage(systemMsg)
```

---

## Memory vs Context Window

### Memory (HNO)
- HNO에서 관리
- 구성 가능한 메시지 제한
- 자동 절삭

### Context Window (LLM)
- 모델별 제한 (예: 128K 토큰)
- LLM 제공업체에서 관리
- 초과 시 오류 발생 가능

**모범 사례**: Memory 제한을 LLM 컨텍스트 윈도우 미만으로 유지.

```go
// GPT-4o-mini: 128K 토큰 ≈ 100K 단어 ≈ 400 메시지
memory := memory.New(memory.Config{MaxMessages: 200})
```

---

## 문제 해결

### 에이전트가 기억하지 못함

메모리 구성 확인:

```go
// 나쁨 ❌ - 메모리 없음
agent, _ := agent.New(agent.Config{
    Model:  model,
    Memory: nil,
})

// 좋음 ✅ - 메모리 있음
agent, _ := agent.New(agent.Config{
    Model: model,
    // 기본적으로 메모리 활성화
})
```

### 메모리가 너무 큼

메시지 제한 줄이기:

```go
memory := memory.New(memory.Config{
    MaxMessages: 50,  // 더 작은 제한
})
```

### 컨텍스트 손실

불필요하게 메모리를 지우지 마세요:

```go
// 나쁨 ❌ - 각 메시지 후 지우기
output, _ := agent.Run(ctx, input)
agent.ClearMemory() // 하지 마세요

// 좋음 ✅ - 컨텍스트 보존
output, _ := agent.Run(ctx, input)
// 다음 상호작용을 위해 메모리 유지
```

---

## 예제

### 다중 턴 대화

```go
agent, _ := agent.New(agent.Config{Model: model})

// 턴 1
agent.Run(ctx, "I'm planning a trip to Paris")

// 턴 2
agent.Run(ctx, "What's the weather like there?")
// 에이전트는 "there" = Paris를 앎

// 턴 3
agent.Run(ctx, "What should I pack?")
// 에이전트는 Paris와 날씨에 대해 앎
```

### 세션 관리

```go
type SessionManager struct {
    agents map[string]*agent.Agent
}

func (sm *SessionManager) GetAgent(sessionID string) *agent.Agent {
    if ag, exists := sm.agents[sessionID]; exists {
        return ag
    }

    // 세션을 위한 새 에이전트 생성
    ag, _ := agent.New(agent.Config{Model: model})
    sm.agents[sessionID] = ag
    return ag
}

func (sm *SessionManager) EndSession(sessionID string) {
    if ag, exists := sm.agents[sessionID]; exists {
        ag.ClearMemory()
        delete(sm.agents, sessionID)
    }
}
```

---

## 다음 단계

- 공유 메모리로 [Teams](/guide/team) 구축
- 능력 향상을 위한 [Tools](/guide/tools) 추가
- 컨텍스트 전달로 [Workflows](/guide/workflow) 생성
- 자세한 문서는 [Memory API Reference](/api/memory) 확인

---

## 관련 예제

- [Simple Agent](/examples/simple-agent) - 기본 메모리 사용
- [Multi-Turn Chat](/examples/chat-agent) - 대화 예제
- [Session Management](/examples/session-demo) - 세션 기반 메모리
