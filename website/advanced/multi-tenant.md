---
title: Multi-Tenant Support
description: Multi-tenant data isolation for serving multiple users with a single agent instance
outline: deep
---

# Multi-Tenant Support

**Multi-Tenant Support** enables Agno-Go to serve multiple users with a single Agent instance while ensuring complete isolation of conversation history and session state between users.

---

## Overview

Multi-tenant architecture allows a single application instance to serve multiple users (tenants) with complete data isolation:

```
                 ┌─────────────────┐
                 │  Agent Instance │
                 └────────┬────────┘
                          │
         ┌────────────────┼────────────────┐
         ▼                ▼                ▼
   ┌──────────┐     ┌──────────┐     ┌──────────┐
   │  User A  │     │  User B  │     │  User C  │
   │ Messages │     │ Messages │     │ Messages │
   └──────────┘     └──────────┘     └──────────┘
```

---

## What is Multi-Tenancy?

Multi-tenancy is an architecture pattern where a single application instance serves multiple isolated users or organizations. Each tenant's data is completely separated from others.

### Without Multi-Tenant

```go
// ❌ Each user needs a separate Agent instance
userAgents := make(map[string]*agent.Agent)

agent1, _ := agent.New(config)  // User 1
agent2, _ := agent.New(config)  // User 2
agent3, _ := agent.New(config)  // User 3
// ... 1000+ users = 1000+ Agent instances
```

**Problems:**
- High memory usage: 1000 users = 1000 Agent instances
- Difficult to manage: Manual agent lifecycle management
- Resource waste: Each agent has duplicate configuration

### With Multi-Tenant

```go
// ✅ Single Agent instance serves all users
sharedAgent, _ := agent.New(config)

// Different users use different userID
output1, _ := sharedAgent.Run(ctx, "user-1 input", "user-1")
output2, _ := sharedAgent.Run(ctx, "user-2 input", "user-2")
output3, _ := sharedAgent.Run(ctx, "user-3 input", "user-3")
```

**Advantages:**
- ✅ Low memory usage: Single Agent instance
- ✅ Easy management: Unified configuration and updates
- ✅ Efficient resource utilization: Shared model and tools

---

## Quick Start

### 1. Create Multi-Tenant Agent

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
    // Create model
    model, _ := openai.New("gpt-4", openai.Config{
        APIKey: "your-api-key",
    })

    // Create multi-tenant Memory
    mem := memory.NewInMemory(100)  // Automatically supports multi-tenancy

    // Create Agent
    myAgent, _ := agent.New(&agent.Config{
        Name:         "customer-service",
        Model:        model,
        Memory:       mem,
        Instructions: "You are a helpful customer service agent.",
    })

    // Conversations for different users
    ctx := context.Background()

    // User A's conversation
    myAgent.UserID = "user-a"
    output1, _ := myAgent.Run(ctx, "My name is Alice")
    fmt.Printf("User A: %s\n", output1.Content)

    output2, _ := myAgent.Run(ctx, "What's my name?")  // "Your name is Alice"
    fmt.Printf("User A: %s\n", output2.Content)

    // User B's conversation
    myAgent.UserID = "user-b"
    output3, _ := myAgent.Run(ctx, "My name is Bob")
    fmt.Printf("User B: %s\n", output3.Content)

    output4, _ := myAgent.Run(ctx, "What's my name?")  // "Your name is Bob"
    fmt.Printf("User B: %s\n", output4.Content)

    // User A talks again
    myAgent.UserID = "user-a"
    output5, _ := myAgent.Run(ctx, "What's my name?")  // "Your name is Alice"
    fmt.Printf("User A: %s\n", output5.Content)
}
```

### 2. Web API Example

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
    // Initialize Agent
    sharedAgent, _ = agent.New(&agent.Config{
        Name:   "api-agent",
        Model:  model,
        Memory: memory.NewInMemory(100),
    })

    // Setup routes
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

    // Set current user ID
    sharedAgent.UserID = req.UserID

    // Run conversation
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

**Test:**
```bash
# User A's conversation
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-a", "message": "My name is Alice"}'

curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-a", "message": "What is my name?"}'
# Response: {"user_id":"user-a","reply":"Your name is Alice"}

# User B's conversation
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-b", "message": "My name is Bob"}'

curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-b", "message": "What is my name?"}'
# Response: {"user_id":"user-b","reply":"Your name is Bob"}
```

---

## Memory Management

### Memory Interface

The Memory interface supports optional `userID` parameter:

```go
// pkg/hno/memory/memory.go

type Memory interface {
    // Add message (supports optional userID)
    Add(message *types.Message, userID ...string)

    // Get message history (supports optional userID)
    GetMessages(userID ...string) []*types.Message

    // Clear messages for specific user
    Clear(userID ...string)

    // Clear messages for all users
    ClearAll()

    // Get message count for specific user
    Size(userID ...string) int
}
```

### InMemory Implementation

```go
type InMemory struct {
    userMessages map[string][]*types.Message  // User ID → Message list
    maxSize      int
    mu           sync.RWMutex
}

// Default user ID
const defaultUserID = "default"

// Get user ID (backward compatible)
func getUserID(userID ...string) string {
    if len(userID) > 0 && userID[0] != "" {
        return userID[0]
    }
    return defaultUserID
}
```

### Usage Examples

#### Basic Usage

```go
mem := memory.NewInMemory(100)

// User A's messages
mem.Add(types.NewUserMessage("Hello from Alice"), "user-a")
mem.Add(types.NewAssistantMessage("Hi Alice!"), "user-a")

// User B's messages
mem.Add(types.NewUserMessage("Hello from Bob"), "user-b")
mem.Add(types.NewAssistantMessage("Hi Bob!"), "user-b")

// Get messages for each user
messagesA := mem.GetMessages("user-a")  // 2 messages
messagesB := mem.GetMessages("user-b")  // 2 messages

fmt.Printf("User A has %d messages\n", len(messagesA))  // 2
fmt.Printf("User B has %d messages\n", len(messagesB))  // 2
```

#### Backward Compatibility

```go
mem := memory.NewInMemory(100)

// No userID specified (uses default "default")
mem.Add(types.NewUserMessage("Hello"))
messages := mem.GetMessages()

// Equivalent to:
mem.Add(types.NewUserMessage("Hello"), "default")
messages := mem.GetMessages("default")
```

#### Clear Operations

```go
mem := memory.NewInMemory(100)

// Add messages for different users
mem.Add(types.NewUserMessage("User A msg"), "user-a")
mem.Add(types.NewUserMessage("User B msg"), "user-b")

// Clear specific user
mem.Clear("user-a")
fmt.Printf("User A: %d messages\n", mem.Size("user-a"))  // 0
fmt.Printf("User B: %d messages\n", mem.Size("user-b"))  // 1

// Clear all users
mem.ClearAll()
fmt.Printf("User A: %d messages\n", mem.Size("user-a"))  // 0
fmt.Printf("User B: %d messages\n", mem.Size("user-b"))  // 0
```

---

## Agent Integration

### Agent Configuration

```go
type Agent struct {
    ID           string
    Name         string
    Model        models.Model
    Toolkits     []toolkit.Toolkit
    Memory       memory.Memory
    Instructions string
    MaxLoops     int

    UserID string  // ⭐ NEW: Multi-tenant user ID
}

type Config struct {
    Name         string
    Model        models.Model
    Toolkits     []toolkit.Toolkit
    Memory       memory.Memory
    Instructions string
    MaxLoops     int

    UserID string  // ⭐ NEW: Multi-tenant user ID
}
```

### Run Method Implementation

```go
// pkg/hno/agent/agent.go

func (a *Agent) Run(ctx context.Context, input string) (*RunOutput, error) {
    // ...

    // All Memory calls pass UserID
    userMsg := types.NewUserMessage(input)
    a.Memory.Add(userMsg, a.UserID)  // ⭐ Pass UserID

    // ...

    messages := a.Memory.GetMessages(a.UserID)  // ⭐ Pass UserID

    // ...

    a.Memory.Add(types.NewAssistantMessage(content), a.UserID)  // ⭐ Pass UserID
}
```

### Usage Patterns

#### Pattern 1: Shared Agent + Switch UserID

```go
agent, _ := agent.New(&agent.Config{
    Name:   "shared-agent",
    Model:  model,
    Memory: memory.NewInMemory(100),
})

// Handle User A's request
agent.UserID = "user-a"
output, _ := agent.Run(ctx, "User A message")

// Handle User B's request
agent.UserID = "user-b"
output, _ := agent.Run(ctx, "User B message")
```

⚠️ **Note**: This approach requires careful UserID switching in concurrent environments

#### Pattern 2: Separate Agent per User (Recommended for High Concurrency)

```go
// Create Agent factory
func createUserAgent(userID string) (*agent.Agent, error) {
    return agent.New(&agent.Config{
        Name:   "user-agent",
        Model:  sharedModel,  // Can share Model
        Memory: memory.NewInMemory(100),
        UserID: userID,  // Set fixed UserID
    })
}

// Use Agent pool
userAgents := make(map[string]*agent.Agent)

// User A
if _, exists := userAgents["user-a"]; !exists {
    userAgents["user-a"], _ = createUserAgent("user-a")
}
output, _ := userAgents["user-a"].Run(ctx, "User A message")

// User B
if _, exists := userAgents["user-b"]; !exists {
    userAgents["user-b"], _ = createUserAgent("user-b")
}
output, _ := userAgents["user-b"].Run(ctx, "User B message")
```

---

## Data Isolation Guarantees

### 1. Memory Isolation

```go
// Test: Multi-tenant isolation
mem := memory.NewInMemory(100)

// User A adds 10 messages
for i := 0; i < 10; i++ {
    mem.Add(types.NewUserMessage(fmt.Sprintf("User A message %d", i)), "user-a")
}

// User B adds 5 messages
for i := 0; i < 5; i++ {
    mem.Add(types.NewUserMessage(fmt.Sprintf("User B message %d", i)), "user-b")
}

// Verify isolation
assert.Equal(t, 10, mem.Size("user-a"))  // ✅
assert.Equal(t, 5, mem.Size("user-b"))   // ✅
assert.Equal(t, 0, mem.Size("user-c"))   // ✅ Non-existent user

messagesA := mem.GetMessages("user-a")
messagesB := mem.GetMessages("user-b")

// User A cannot see User B's messages
for _, msg := range messagesA {
    assert.NotContains(t, msg.Content, "User B")  // ✅
}
```

### 2. Concurrency Safety

```go
// Test: 1000 concurrent requests
mem := memory.NewInMemory(100)
var wg sync.WaitGroup

// 10 users, 100 concurrent requests each
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

// Verify each user has correct number of messages
for userID := 0; userID < 10; userID++ {
    userIDStr := fmt.Sprintf("user-%d", userID)
    assert.Equal(t, 100, mem.Size(userIDStr))  // ✅
}
```

---

## Best Practices

### 1. UserID Naming Convention

```go
// ✅ Recommended: Use consistent naming convention
"user-{uuid}"           // user-123e4567-e89b-12d3-a456-426614174000
"org-{org_id}-user-{id}" // org-acme-user-001
"tenant-{id}"           // tenant-12345

// ❌ Avoid: Use unstable identifiers
"{ip_address}"          // IP may change
"{session_id}"          // Session expires
```

### 2. Error Handling

```go
// Validate UserID
func validateUserID(userID string) error {
    if userID == "" {
        return fmt.Errorf("userID cannot be empty")
    }
    if len(userID) > 255 {
        return fmt.Errorf("userID too long (max 255 chars)")
    }
    // Can add more validation rules
    return nil
}

// Validate at API layer
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

### 3. Logging and Monitoring

```go
// Log UserID for each request
logger.Info("Processing request",
    "user_id", userID,
    "input_length", len(input),
    "timestamp", time.Now(),
)

// Monitoring metrics
metrics.RecordUserRequest(userID)
metrics.RecordMessageCount(userID, mem.Size(userID))
```

### 4. Security Considerations

```go
// Use encrypted UserID
func encryptUserID(plainUserID string) string {
    // Use encryption algorithm
    return encryptedID
}

// Access control
func checkUserPermission(userID string, action string) bool {
    // Implement permission check logic
    return hasPermission
}
```

---

## Troubleshooting

### Common Issues

#### 1. User Data Confusion

**Symptom:** User A sees User B's messages

**Cause:** UserID not properly passed

**Solution:**
```go
// ❌ Wrong
agent.Run(ctx, input)  // UserID not set

// ✅ Correct
agent.UserID = "user-a"
agent.Run(ctx, input)
```

#### 2. High Memory Usage

**Symptom:** Memory continuously growing

**Cause:** Inactive user data not cleaned up

**Solution:**
```go
// Periodic cleanup
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        cleanupInactiveUsers(mem, 24*time.Hour)
    }
}()
```

#### 3. Concurrent Race Conditions

**Symptom:** Data occasionally lost or duplicated

**Cause:** Shared Agent's UserID field modified by multiple goroutines

**Solution:**
```go
// ❌ Wrong: Concurrent modification of shared Agent
var sharedAgent *agent.Agent
go func() { sharedAgent.UserID = "user-a"; sharedAgent.Run(ctx, input) }()
go func() { sharedAgent.UserID = "user-b"; sharedAgent.Run(ctx, input) }()

// ✅ Correct: Separate Agent per user
agentA := createUserAgent("user-a")
agentB := createUserAgent("user-b")
go func() { agentA.Run(ctx, input) }()
go func() { agentB.Run(ctx, input) }()
```

---

## Integration with Other Features

### A2A Interface + Multi-Tenant

```go
// A2A request contains contextID, can be used as userID
type Message struct {
    MessageID string `json:"messageId"`
    Role      string `json:"role"`
    AgentID   string `json:"agentId"`
    ContextID string `json:"contextId"`  // ⭐ Can be used as userID
    Parts     []Part `json:"parts"`
}

// Set UserID during mapping
func MapA2ARequestToRunInput(req *JSONRPC2Request) (*RunInput, error) {
    // ...
    agent.UserID = req.Params.Message.ContextID  // ⭐ Use contextID as userID
    // ...
}
```

### Session State + Multi-Tenant

```go
// ExecutionContext supports both SessionID and UserID
execCtx := workflow.NewExecutionContextWithSession(
    "input",
    "session-123",  // SessionID: Identifier for single session
    "user-a",       // UserID: User identifier
)

// SessionID: For session state management
// UserID: For multi-tenant data isolation
```

---

## Related Documentation

- [A2A Interface](/api/a2a) - Agent-to-agent communication
- [Session State Management](/guide/session-state) - Workflow session management
- [Memory Guide](/guide/memory) - Memory usage guide

---

## Testing

Complete test coverage includes:

- ✅ Multi-user data isolation
- ✅ Concurrency safety (1000 goroutines)
- ✅ Agent integration tests
- ✅ Memory capacity management
- ✅ Clear operation correctness

**Test Coverage:** 93.1% (Memory module)

Run tests:
```bash
cd pkg/hno/memory
go test -v -run TestInMemory

cd pkg/hno/agent
go test -v -run TestAgent_MultiTenant
```

---

**Last Updated:** 2025-01-08
