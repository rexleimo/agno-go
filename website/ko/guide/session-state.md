---
title: 세션 상태 관리
description: 워크플로우 실행을 위한 스레드 안전 상태 관리와 지능형 병렬 브랜치 병합
outline: deep
---

# 세션 상태 관리

**세션 상태 관리**는 워크플로우 실행 중 상태 관리 기능을 제공하며, 단계 간 데이터 공유, 동시성 안전 상태 액세스, 병렬 브랜치의 지능형 병합을 지원합니다.

## 왜 세션 상태가 필요한가?

복잡한 워크플로우에서 단계 간 데이터를 공유해야 합니다:

```
Step1: 사용자 정보 가져오기
  ↓
  SessionState에 저장: {"user_id": "123", "name": "Alice"}
  ↓
Step2: 사용자 정보를 기반으로 주문 조회
  ↓
  SessionState에서 읽기: user_id = "123"
  SessionState에 저장: {"orders": [...]}
  ↓
Step3: 보고서 생성
  ↓
  SessionState에서 읽기: user_id, name, orders
```

세션 상태가 없으면 단계 출력을 통해 데이터를 전달해야 하며, 이는 강한 결합과 복잡성을 초래합니다. 세션 상태는 모든 단계가 액세스할 수 있는 공유 메모리 공간 역할을 합니다.

## 핵심 기능

1. **스레드 안전**: `sync.RWMutex`로 동시 액세스 보호
2. **딥 카피**: 병렬 브랜치는 독립적인 상태 복사본을 가져옴
3. **스마트 병합**: 병렬 실행 후 자동 상태 병합
4. **유연한 타입**: 모든 `interface{}` 타입 데이터 지원

## 빠른 시작

### 기본 사용법

```go
package main

import (
    "context"
    "fmt"

    "github.com/rexleimo/agno-go/pkg/hno/workflow"
)

func main() {
    // 세션 상태로 실행 컨텍스트 생성
    execCtx := workflow.NewExecutionContextWithSession(
        "initial input",
        "session-123",  // sessionID
        "user-456",     // userID
    )

    // 세션 상태 설정
    execCtx.SetSessionState("user_name", "Alice")
    execCtx.SetSessionState("user_age", 30)
    execCtx.SetSessionState("preferences", map[string]string{
        "language": "zh-CN",
        "theme":    "dark",
    })

    // 세션 상태 가져오기
    if name, ok := execCtx.GetSessionState("user_name"); ok {
        fmt.Printf("User Name: %s\n", name)
    }

    if age, ok := execCtx.GetSessionState("user_age"); ok {
        fmt.Printf("User Age: %d\n", age)
    }
}
```

### 워크플로우에서 사용

```go
// 워크플로우 생성
wf := workflow.NewWorkflow("user-workflow")

// 단계 1: 사용자 정보 가져오기
step1 := workflow.NewStep("get-user", func(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
    // 사용자 정보 가져오기 시뮬레이션
    userInfo := map[string]interface{}{
        "id":    "user-123",
        "name":  "Alice",
        "email": "alice@example.com",
    }

    // SessionState에 저장
    execCtx.SetSessionState("user_info", userInfo)

    return execCtx, nil
})

// 단계 2: 사용자 주문 가져오기
step2 := workflow.NewStep("get-orders", func(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
    // SessionState에서 사용자 정보 읽기
    userInfoRaw, ok := execCtx.GetSessionState("user_info")
    if !ok {
        return execCtx, fmt.Errorf("user_info not found in session state")
    }

    userInfo := userInfoRaw.(map[string]interface{})
    userID := userInfo["id"].(string)

    // 주문 가져오기 시뮬레이션
    orders := []string{"order-1", "order-2", "order-3"}
    execCtx.SetSessionState("orders", orders)

    fmt.Printf("Got %d orders for user %s\n", len(orders), userID)

    return execCtx, nil
})

// 단계 연결
step1.Then(step2)
wf.AddStep(step1)

// 워크플로우 실행
execCtx := workflow.NewExecutionContextWithSession("", "session-123", "user-456")
result, err := wf.Execute(context.Background(), execCtx)
if err != nil {
    panic(err)
}

// 최종 상태 확인
orders, _ := result.GetSessionState("orders")
fmt.Printf("Final orders: %v\n", orders)
```

## 병렬 실행 및 상태 병합

### 문제점

병렬 실행 중 여러 브랜치가 SessionState를 동시에 수정할 수 있습니다:

```
              ┌─→ 브랜치 A: Set("key1", "value_A")
병렬 단계     ├─→ 브랜치 B: Set("key2", "value_B")
              └─→ 브랜치 C: Set("key1", "value_C")  // ⚠️ 충돌!
```

### 해결책

Agno-Go는 **딥 카피 + last-write-wins** 전략을 사용합니다:

1. 각 병렬 브랜치는 독립적인 SessionState 복사본을 가져옵니다
2. 브랜치는 간섭 없이 독립적으로 실행됩니다
3. 완료 후 모든 변경 사항이 순서대로 병합됩니다
4. 충돌이 존재하면 나중 브랜치가 이전 브랜치를 덮어씁니다

```go
// pkg/hno/workflow/parallel.go

func (p *Parallel) Execute(ctx context.Context, execCtx *ExecutionContext) (*ExecutionContext, error) {
    // 1. 각 브랜치에 독립적인 SessionState 복사본 생성
    sessionStateCopies := make([]*SessionState, len(p.Nodes))
    for i := range p.Nodes {
        if execCtx.SessionState != nil {
            sessionStateCopies[i] = execCtx.SessionState.Clone()  // 딥 카피
        } else {
            sessionStateCopies[i] = NewSessionState()
        }
    }

    // 2. 브랜치를 병렬 실행
    // ... (goroutines 실행)

    // 3. 모든 브랜치 상태 변경 사항 병합
    execCtx.SessionState = MergeParallelSessionStates(
        originalSessionState,
        modifiedSessionStates,
    )

    return execCtx, nil
}
```

### 병합 전략

```go
// pkg/hno/workflow/session_state.go

func MergeParallelSessionStates(original *SessionState, modified []*SessionState) *SessionState {
    merged := NewSessionState()

    // 1. 원본 상태 복사
    if original != nil {
        for k, v := range original.data {
            merged.data[k] = v
        }
    }

    // 2. 각 브랜치의 변경 사항을 순서대로 병합
    for _, modState := range modified {
        if modState == nil {
            continue
        }
        for k, v := range modState.data {
            merged.data[k] = v  // Last-write-wins
        }
    }

    return merged
}
```

### 예제

```go
// 3개 브랜치의 병렬 실행
parallel := workflow.NewParallel()

// 브랜치 A: counter = 1 설정
branchA := workflow.NewStep("branch-a", func(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
    execCtx.SetSessionState("counter", 1)
    execCtx.SetSessionState("branch_a_result", "done")
    return execCtx, nil
})

// 브랜치 B: counter = 2 설정
branchB := workflow.NewStep("branch-b", func(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
    execCtx.SetSessionState("counter", 2)
    execCtx.SetSessionState("branch_b_result", "done")
    return execCtx, nil
})

// 브랜치 C: counter = 3 설정
branchC := workflow.NewStep("branch-c", func(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
    execCtx.SetSessionState("counter", 3)
    execCtx.SetSessionState("branch_c_result", "done")
    return execCtx, nil
})

parallel.AddNode(branchA)
parallel.AddNode(branchB)
parallel.AddNode(branchC)

// 병렬 단계 실행
execCtx := workflow.NewExecutionContextWithSession("", "session-123", "user-456")
result, _ := parallel.Execute(context.Background(), execCtx)

// 병합된 결과 확인
counter, _ := result.GetSessionState("counter")
fmt.Printf("Counter: %v\n", counter)  // 출력은 1, 2 또는 3일 수 있음 (실행 순서에 따라)

branchAResult, _ := result.GetSessionState("branch_a_result")
branchBResult, _ := result.GetSessionState("branch_b_result")
branchCResult, _ := result.GetSessionState("branch_c_result")
fmt.Printf("All branches completed: %v, %v, %v\n", branchAResult, branchBResult, branchCResult)
// 출력: All branches completed: done, done, done
```

## API 참조

### SessionState 타입

```go
type SessionState struct {
    mu   sync.RWMutex
    data map[string]interface{}
}
```

#### 메서드

##### NewSessionState()

```go
func NewSessionState() *SessionState
```

새 SessionState 인스턴스를 생성합니다.

##### Set(key string, value interface{})

```go
func (ss *SessionState) Set(key string, value interface{})
```

키-값 쌍 설정 (스레드 안전).

##### Get(key string) (interface{}, bool)

```go
func (ss *SessionState) Get(key string) (interface{}, bool)
```

키의 값 가져오기 (스레드 안전).

##### Clone() *SessionState

```go
func (ss *SessionState) Clone() *SessionState
```

JSON 직렬화를 사용하여 SessionState를 딥 카피합니다.

##### GetAll() map[string]interface{}

```go
func (ss *SessionState) GetAll() map[string]interface{}
```

모든 키-값 쌍의 복사본 가져오기 (스레드 안전).

## 모범 사례

### 1. 큰 데이터 저장 피하기

```go
// ❌ 권장하지 않음
execCtx.SetSessionState("all_users", []User{ /* 10000+ users */ })

// ✅ 권장
execCtx.SetSessionState("user_ids", []string{"id1", "id2", "id3"})
```

**이유**: `Clone()`은 JSON 직렬화를 사용하므로 큰 데이터 구조에서는 비용이 많이 듭니다.

### 2. 병렬 브랜치 현명하게 사용

```go
// ✅ 병렬 브랜치는 서로 다른 데이터를 독립적으로 처리
// 브랜치 A: 사용자 데이터 처리
// 브랜치 B: 주문 데이터 처리
// 브랜치 C: 로그 데이터 처리

// ⚠️ 병렬 브랜치가 같은 키를 수정하는 것을 피함
// (last-write-wins 전략을 이해하는 경우 제외)
```

## 문제 해결

### 일반적인 문제

#### 1. SessionState가 nil

**증상**:
```go
panic: runtime error: invalid memory address or nil pointer dereference
```

**원인**: SessionState가 초기화되지 않음

**해결책**:
```go
// ❌ 잘못됨
execCtx := &workflow.ExecutionContext{}
execCtx.SetSessionState("key", "value")  // panic!

// ✅ 올바름
execCtx := workflow.NewExecutionContextWithSession("", "session-id", "user-id")
execCtx.SetSessionState("key", "value")  // OK
```

#### 2. 타입 어설션 실패

**증상**:
```go
panic: interface conversion: interface {} is string, not int
```

**해결책**:
```go
// ✅ 올바름
execCtx.SetSessionState("age", 30)  // int 저장
raw, ok := execCtx.GetSessionState("age")
if !ok {
    // 키가 존재하지 않음
}
age, ok := raw.(int)
if !ok {
    // 타입 불일치
}
```

## 테스트

완전한 테스트 커버리지에는 다음이 포함됩니다:

- ✅ 기본 Get/Set 작업
- ✅ 딥 카피 (Clone)
- ✅ 상태 병합 (Merge)
- ✅ 동시성 안전성 (1000 goroutines)
- ✅ 워크플로우 통합 테스트
- ✅ 병렬 브랜치 격리

**테스트 커버리지**: 543줄의 테스트 코드

테스트 실행:
```bash
cd pkg/hno/workflow
go test -v -run TestSessionState
```

## 관련 문서

- [워크플로우 가이드](/ko/guide/workflow) - 워크플로우 엔진 사용
- [팀 가이드](/ko/guide/team) - 멀티 에이전트 협력
- [메모리 관리](/ko/guide/memory) - 대화 메모리

---

**최종 업데이트**: 2025-01-XX
