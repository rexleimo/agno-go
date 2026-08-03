---
title: Session State Management
description: Thread-safe state management for workflow execution with intelligent parallel branch merging
outline: deep
---

# Session State Management

**Session State Management** provides state management capabilities during workflow execution, supporting cross-step data sharing, concurrency-safe state access, and intelligent merging of parallel branches.

## Why Session State?

In complex workflows, steps need to share data:

```
Step1: Get user info
  ↓
  Save to SessionState: {"user_id": "123", "name": "Alice"}
  ↓
Step2: Query orders based on user info
  ↓
  Read from SessionState: user_id = "123"
  Save to SessionState: {"orders": [...]}
  ↓
Step3: Generate report
  ↓
  Read from SessionState: user_id, name, orders
```

Without session state, you'd need to pass data through step outputs, creating tight coupling and complexity. Session state acts as a shared memory space accessible to all steps.

## Core Features

1. **Thread-Safe**: Protected by `sync.RWMutex` for concurrent access
2. **Deep Copy**: Parallel branches get independent state copies
3. **Smart Merge**: Automatic state merging after parallel execution
4. **Flexible Types**: Supports any `interface{}` type data

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"

    "github.com/rexleimo/agno-go/pkg/hno/workflow"
)

func main() {
    // Create execution context with session state
    execCtx := workflow.NewExecutionContextWithSession(
        "initial input",
        "session-123",  // sessionID
        "user-456",     // userID
    )

    // Set session state
    execCtx.SetSessionState("user_name", "Alice")
    execCtx.SetSessionState("user_age", 30)
    execCtx.SetSessionState("preferences", map[string]string{
        "language": "zh-CN",
        "theme":    "dark",
    })

    // Get session state
    if name, ok := execCtx.GetSessionState("user_name"); ok {
        fmt.Printf("User Name: %s\n", name)
    }

    if age, ok := execCtx.GetSessionState("user_age"); ok {
        fmt.Printf("User Age: %d\n", age)
    }
}
```

### Using in Workflow

```go
// Create workflow
wf := workflow.NewWorkflow("user-workflow")

// Step 1: Get user info
step1 := workflow.NewStep("get-user", func(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
    // Simulate fetching user info
    userInfo := map[string]interface{}{
        "id":    "user-123",
        "name":  "Alice",
        "email": "alice@example.com",
    }

    // Save to SessionState
    execCtx.SetSessionState("user_info", userInfo)

    return execCtx, nil
})

// Step 2: Get user orders
step2 := workflow.NewStep("get-orders", func(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
    // Read user info from SessionState
    userInfoRaw, ok := execCtx.GetSessionState("user_info")
    if !ok {
        return execCtx, fmt.Errorf("user_info not found in session state")
    }

    userInfo := userInfoRaw.(map[string]interface{})
    userID := userInfo["id"].(string)

    // Simulate fetching orders
    orders := []string{"order-1", "order-2", "order-3"}
    execCtx.SetSessionState("orders", orders)

    fmt.Printf("Got %d orders for user %s\n", len(orders), userID)

    return execCtx, nil
})

// Chain steps
step1.Then(step2)
wf.AddStep(step1)

// Execute workflow
execCtx := workflow.NewExecutionContextWithSession("", "session-123", "user-456")
result, err := wf.Execute(context.Background(), execCtx)
if err != nil {
    panic(err)
}

// Check final state
orders, _ := result.GetSessionState("orders")
fmt.Printf("Final orders: %v\n", orders)
```

## Parallel Execution and State Merging

### The Challenge

During parallel execution, multiple branches may modify SessionState simultaneously:

```
              ┌─→ Branch A: Set("key1", "value_A")
Parallel Step ├─→ Branch B: Set("key2", "value_B")
              └─→ Branch C: Set("key1", "value_C")  // ⚠️ Conflict!
```

This creates a race condition in traditional implementations. How do we handle conflicting writes?

### The Solution

HNO uses a **deep copy + last-write-wins** strategy:

1. Each parallel branch gets an independent SessionState copy
2. Branches execute independently without interference
3. After completion, all changes are merged in order
4. If conflicts exist, later branches override earlier ones

```go
// pkg/hno/workflow/parallel.go

func (p *Parallel) Execute(ctx context.Context, execCtx *ExecutionContext) (*ExecutionContext, error) {
    // 1. Create independent SessionState copy for each branch
    sessionStateCopies := make([]*SessionState, len(p.Nodes))
    for i := range p.Nodes {
        if execCtx.SessionState != nil {
            sessionStateCopies[i] = execCtx.SessionState.Clone()  // Deep copy
        } else {
            sessionStateCopies[i] = NewSessionState()
        }
    }

    // 2. Execute branches in parallel
    // ... (goroutines execution)

    // 3. Merge all branch state changes
    execCtx.SessionState = MergeParallelSessionStates(
        originalSessionState,
        modifiedSessionStates,
    )

    return execCtx, nil
}
```

### Merge Strategy

```go
// pkg/hno/workflow/session_state.go

func MergeParallelSessionStates(original *SessionState, modified []*SessionState) *SessionState {
    merged := NewSessionState()

    // 1. Copy original state
    if original != nil {
        for k, v := range original.data {
            merged.data[k] = v
        }
    }

    // 2. Merge changes from each branch in order
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

### Example

```go
// Parallel execution of 3 branches
parallel := workflow.NewParallel()

// Branch A: Set counter = 1
branchA := workflow.NewStep("branch-a", func(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
    execCtx.SetSessionState("counter", 1)
    execCtx.SetSessionState("branch_a_result", "done")
    return execCtx, nil
})

// Branch B: Set counter = 2
branchB := workflow.NewStep("branch-b", func(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
    execCtx.SetSessionState("counter", 2)
    execCtx.SetSessionState("branch_b_result", "done")
    return execCtx, nil
})

// Branch C: Set counter = 3
branchC := workflow.NewStep("branch-c", func(ctx context.Context, execCtx *workflow.ExecutionContext) (*workflow.ExecutionContext, error) {
    execCtx.SetSessionState("counter", 3)
    execCtx.SetSessionState("branch_c_result", "done")
    return execCtx, nil
})

parallel.AddNode(branchA)
parallel.AddNode(branchB)
parallel.AddNode(branchC)

// Execute parallel step
execCtx := workflow.NewExecutionContextWithSession("", "session-123", "user-456")
result, _ := parallel.Execute(context.Background(), execCtx)

// Check merged result
counter, _ := result.GetSessionState("counter")
fmt.Printf("Counter: %v\n", counter)  // Output may be 1, 2, or 3 (depends on execution order)

branchAResult, _ := result.GetSessionState("branch_a_result")
branchBResult, _ := result.GetSessionState("branch_b_result")
branchCResult, _ := result.GetSessionState("branch_c_result")
fmt.Printf("All branches completed: %v, %v, %v\n", branchAResult, branchBResult, branchCResult)
// Output: All branches completed: done, done, done
```

## API Reference

### SessionState Type

```go
type SessionState struct {
    mu   sync.RWMutex
    data map[string]interface{}
}
```

#### Methods

##### NewSessionState()

```go
func NewSessionState() *SessionState
```

Create a new SessionState instance.

##### Set(key string, value interface{})

```go
func (ss *SessionState) Set(key string, value interface{})
```

Set key-value pair (thread-safe).

**Parameters**:
- `key`: The key to set
- `value`: Any value type (interface{})

##### Get(key string) (interface{}, bool)

```go
func (ss *SessionState) Get(key string) (interface{}, bool)
```

Get value for key (thread-safe).

**Returns**:
- `value`: The value associated with the key
- `exists`: Whether the key exists

**Example**:
```go
if value, ok := sessionState.Get("user_id"); ok {
    userID := value.(string)
    fmt.Printf("User ID: %s\n", userID)
}
```

##### Clone() *SessionState

```go
func (ss *SessionState) Clone() *SessionState
```

Deep copy SessionState using JSON serialization.

**Note**: Non-serializable types fall back to shallow copy.

##### GetAll() map[string]interface{}

```go
func (ss *SessionState) GetAll() map[string]interface{}
```

Get a copy of all key-value pairs (thread-safe).

##### MergeFrom(other *SessionState)

```go
func (ss *SessionState) MergeFrom(other *SessionState)
```

Merge data from another SessionState.

##### Clear()

```go
func (ss *SessionState) Clear()
```

Clear all data.

---

### ExecutionContext Extensions

```go
type ExecutionContext struct {
    // Existing fields
    Input    string
    Output   string
    Data     map[string]interface{}
    Metadata map[string]interface{}

    // New fields for session state
    SessionState *SessionState `json:"session_state,omitempty"`
    SessionID    string        `json:"session_id,omitempty"`
    UserID       string        `json:"user_id,omitempty"`
}
```

#### New Methods

##### NewExecutionContextWithSession()

```go
func NewExecutionContextWithSession(input, sessionID, userID string) *ExecutionContext
```

Create execution context with session state.

**Parameters**:
- `input`: Initial input string
- `sessionID`: Session identifier
- `userID`: User identifier

##### SetSessionState(key string, value interface{})

```go
func (ec *ExecutionContext) SetSessionState(key string, value interface{})
```

Set session state (convenience method).

##### GetSessionState(key string) (interface{}, bool)

```go
func (ec *ExecutionContext) GetSessionState(key string) (interface{}, bool)
```

Get session state (convenience method).

## Advanced Usage

### Type-Safe Access

```go
// Use type assertion for type safety
func getUserInfo(execCtx *workflow.ExecutionContext) (map[string]interface{}, error) {
    raw, ok := execCtx.GetSessionState("user_info")
    if !ok {
        return nil, fmt.Errorf("user_info not found")
    }

    userInfo, ok := raw.(map[string]interface{})
    if !ok {
        return nil, fmt.Errorf("user_info has invalid type")
    }

    return userInfo, nil
}
```

### Structured Data Storage

```go
type UserProfile struct {
    ID    string
    Name  string
    Email string
    Age   int
}

// Store struct
profile := UserProfile{
    ID:    "user-123",
    Name:  "Alice",
    Email: "alice@example.com",
    Age:   30,
}
execCtx.SetSessionState("user_profile", profile)

// Read struct
raw, _ := execCtx.GetSessionState("user_profile")
profile := raw.(UserProfile)
fmt.Printf("User: %s (%s)\n", profile.Name, profile.Email)
```

### Nested Data Access

```go
// Store nested data
execCtx.SetSessionState("config", map[string]interface{}{
    "database": map[string]interface{}{
        "host": "localhost",
        "port": 5432,
    },
    "cache": map[string]interface{}{
        "enabled": true,
        "ttl":     300,
    },
})

// Access nested data
configRaw, _ := execCtx.GetSessionState("config")
config := configRaw.(map[string]interface{})
db := config["database"].(map[string]interface{})
fmt.Printf("Database: %s:%d\n", db["host"], db["port"])
```

### Session Isolation

```go
// Different sessions use different ExecutionContext

// Session A
sessionA := workflow.NewExecutionContextWithSession("input-a", "session-a", "user-1")
sessionA.SetSessionState("session_name", "Session A")

// Session B
sessionB := workflow.NewExecutionContextWithSession("input-b", "session-b", "user-2")
sessionB.SetSessionState("session_name", "Session B")

// Session A and Session B states are completely isolated
nameA, _ := sessionA.GetSessionState("session_name")  // "Session A"
nameB, _ := sessionB.GetSessionState("session_name")  // "Session B"
```

## Best Practices

### 1. Avoid Storing Large Data

```go
// ❌ Not recommended
execCtx.SetSessionState("all_users", []User{ /* 10000+ users */ })

// ✅ Recommended
execCtx.SetSessionState("user_ids", []string{"id1", "id2", "id3"})
```

**Reason**: `Clone()` uses JSON serialization, which becomes expensive for large data structures.

### 2. Use References Instead of Copies

```go
// ❌ Not recommended - modification doesn't affect SessionState
data, _ := execCtx.GetSessionState("config")
config := data.(map[string]interface{})
config["new_key"] = "new_value"  // Lost!

// ✅ Recommended - read, modify, then set again
configRaw, _ := execCtx.GetSessionState("config")
config := configRaw.(map[string]interface{})
config["new_key"] = "new_value"
execCtx.SetSessionState("config", config)  // Saved
```

### 3. Use Parallel Branches Wisely

```go
// ✅ Parallel branches process different data independently
// Branch A: Process user data
// Branch B: Process order data
// Branch C: Process log data

// ⚠️ Avoid parallel branches modifying the same key
// (unless you understand last-write-wins strategy)
```

**Best practice**: Have each parallel branch work on different keys:

```go
// Branch A
execCtx.SetSessionState("branch_a_counter", 1)

// Branch B
execCtx.SetSessionState("branch_b_counter", 2)

// Branch C
execCtx.SetSessionState("branch_c_counter", 3)
```

## Performance Considerations

### Deep Copy Performance

`Clone()` uses JSON serialization for deep copying:

```go
// Efficient scenarios
execCtx.SetSessionState("counter", 42)           // Simple types
execCtx.SetSessionState("name", "Alice")         // Strings
execCtx.SetSessionState("enabled", true)         // Booleans

// Inefficient scenarios (avoid frequent cloning)
execCtx.SetSessionState("large_data", hugeSlice) // Large data structures
```

**Benchmark results**:
- Small values (< 1KB): ~1-5 μs per clone
- Medium values (1-10 KB): ~10-50 μs per clone
- Large values (> 10 KB): > 50 μs per clone

## Troubleshooting

### Common Issues

#### 1. SessionState is nil

**Symptom**:
```go
panic: runtime error: invalid memory address or nil pointer dereference
```

**Cause**: SessionState not initialized

**Solution**:
```go
// ❌ Wrong
execCtx := &workflow.ExecutionContext{}
execCtx.SetSessionState("key", "value")  // panic!

// ✅ Correct
execCtx := workflow.NewExecutionContextWithSession("", "session-id", "user-id")
execCtx.SetSessionState("key", "value")  // OK
```

#### 2. Type Assertion Failed

**Symptom**:
```go
panic: interface conversion: interface {} is string, not int
```

**Cause**: Type mismatch

**Solution**:
```go
// ❌ Wrong
execCtx.SetSessionState("age", "30")  // Store string
age := execCtx.GetSessionState("age").(int)  // Try to read as int - panic!

// ✅ Correct
execCtx.SetSessionState("age", 30)  // Store int
raw, ok := execCtx.GetSessionState("age")
if !ok {
    // Key doesn't exist
}
age, ok := raw.(int)
if !ok {
    // Type mismatch
}
```

#### 3. Parallel Branch State Lost

**Symptom**: State set by parallel branches is lost after merging

**Cause**: Didn't understand last-write-wins strategy

**Solution**:
```go
// Avoid parallel branches modifying the same key
// Use different key names

// ✅ Recommended
// Branch A
execCtx.SetSessionState("branch_a_counter", 1)

// Branch B
execCtx.SetSessionState("branch_b_counter", 2)

// Branch C
execCtx.SetSessionState("branch_c_counter", 3)
```

## Testing

Complete test coverage includes:

- ✅ Basic Get/Set operations
- ✅ Deep copy (Clone)
- ✅ State merging (Merge)
- ✅ Concurrency safety (1000 goroutines)
- ✅ Workflow integration tests
- ✅ Parallel branch isolation
- ✅ Multi-tenant isolation

**Test Coverage**: 543 lines of test code

Run tests:
```bash
cd pkg/hno/workflow
go test -v -run TestSessionState
```

## Related Documentation

- [Workflow Guide](/guide/workflow) - Workflow engine usage
- [Team Guide](/guide/team) - Multi-agent collaboration
- [Memory Management](/guide/memory) - Conversation memory

---

**Last Updated**: 2025-01-XX
