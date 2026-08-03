# Memory - Conversation History

Manage conversation history and context for your agents.

---

## What is Memory?

Memory stores conversation history, allowing agents to maintain context across multiple interactions. HNO provides built-in memory management with automatic truncation.

### Key Features

- **Automatic History**: Conversations stored automatically
- **Configurable Limits**: Control memory size
- **Message Types**: System, User, Assistant, Tool messages
- **Manual Control**: Clear or manage memory programmatically

---

## Basic Usage

### Default Memory

Agents have memory enabled by default:

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

    // First interaction
    agent.Run(context.Background(), "My name is Alice")
    // Response: "Nice to meet you, Alice!"

    // Second interaction - agent remembers
    output, _ := agent.Run(context.Background(), "What's my name?")
    fmt.Println(output.Content)
    // Response: "Your name is Alice."
}
```

---

## Configuration

### Custom Memory Limit

Set maximum number of messages to store:

```go
import "github.com/rexleimo/agno-go/pkg/hno/memory"

customMemory := memory.New(memory.Config{
    MaxMessages: 50,  // Store up to 50 messages
})

agent, _ := agent.New(agent.Config{
    Model:  model,
    Memory: customMemory,
})
```

### No Memory

Disable memory for stateless agents:

```go
agent, _ := agent.New(agent.Config{
    Model:  model,
    Memory: nil,  // No conversation history
})
```

---

## Memory Operations

### Clear Memory

Reset conversation history:

```go
// Clear all history
agent.ClearMemory()

// Start fresh conversation
agent.Run(ctx, "New conversation")
```

### Get Message History

Access stored messages:

```go
messages := agent.Memory.GetMessages()
for _, msg := range messages {
    fmt.Printf("%s: %s\n", msg.Role, msg.Content)
}
```

### Add Custom Messages

Manually add messages to memory:

```go
import "github.com/rexleimo/agno-go/pkg/hno/types"

// Add system message
agent.Memory.AddMessage(types.Message{
    Role:    types.RoleSystem,
    Content: "You are a helpful assistant.",
})

// Add user message
agent.Memory.AddMessage(types.Message{
    Role:    types.RoleUser,
    Content: "Hello!",
})
```

---

## Message Types

### System Messages

Instructions for the agent:

```go
types.Message{
    Role:    types.RoleSystem,
    Content: "You are a Python expert. Help with coding questions.",
}
```

### User Messages

User input:

```go
types.Message{
    Role:    types.RoleUser,
    Content: "How do I read a file in Python?",
}
```

### Assistant Messages

Agent responses:

```go
types.Message{
    Role:    types.RoleAssistant,
    Content: "Use the open() function: open('file.txt', 'r')",
}
```

### Tool Messages

Tool execution results:

```go
types.Message{
    Role:       types.RoleTool,
    Content:    "Result: 42",
    ToolCallID: "call_123",
}
```

---

## Memory Patterns

### Session-Based Memory

Clear memory between sessions:

```go
func handleSession(agent *agent.Agent, sessionID string) {
    // Load session history (from database, etc.)
    loadSessionHistory(agent, sessionID)

    // Handle conversation
    output, _ := agent.Run(ctx, userInput)

    // Save session history
    saveSessionHistory(agent, sessionID)

    // Clean up
    agent.ClearMemory()
}
```

### Sliding Window

Keep only recent messages:

```go
memory := memory.New(memory.Config{
    MaxMessages: 20,  // Keep last 20 messages
})

// Automatically truncates older messages
agent, _ := agent.New(agent.Config{
    Memory: memory,
})
```

### Persistent Memory

Save and restore conversations:

```go
// Save conversation
messages := agent.Memory.GetMessages()
saveToDatabase(sessionID, messages)

// Restore conversation
savedMessages := loadFromDatabase(sessionID)
for _, msg := range savedMessages {
    agent.Memory.AddMessage(msg)
}
```

---

## Advanced Usage

### Multi-Agent Memory Sharing

Share context between agents:

```go
// Create shared memory
sharedMemory := memory.New(memory.Config{
    MaxMessages: 100,
})

// Both agents use same memory
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

// Agent1 conversation is visible to Agent2
agent1.Run(ctx, "Store this information: X=42")
output, _ := agent2.Run(ctx, "What is X?")
// Agent2 can see Agent1's conversation
```

### Conditional Memory

Clear memory based on conditions:

```go
messageCount := len(agent.Memory.GetMessages())

if messageCount > 100 {
    // Keep only system message
    systemMsg := agent.Memory.GetMessages()[0]
    agent.ClearMemory()
    agent.Memory.AddMessage(systemMsg)
}
```

### Memory Inspection

Analyze conversation history:

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

## Memory Configuration

### Config Struct

```go
type Config struct {
    MaxMessages int // Maximum number of messages to store (default: 100)
}
```

### Default Behavior

- Automatically stores all conversation messages
- Truncates oldest messages when limit reached
- System messages preserved during truncation

---

## Best Practices

### 1. Set Appropriate Limits

Balance context and performance:

```go
// Short conversations
memory := memory.New(memory.Config{MaxMessages: 20})

// Long conversations
memory := memory.New(memory.Config{MaxMessages: 100})

// Very long context
memory := memory.New(memory.Config{MaxMessages: 500})
```

### 2. Clear Memory Strategically

Reset when context changes:

```go
// New topic
if isNewTopic(userInput) {
    agent.ClearMemory()
}

// New session
if isNewSession(sessionID) {
    agent.ClearMemory()
}
```

### 3. Monitor Memory Usage

Track conversation length:

```go
messages := agent.Memory.GetMessages()
if len(messages) > 80 {
    log.Printf("Warning: Approaching memory limit (%d/100)", len(messages))
}
```

### 4. Preserve Important Context

Keep system instructions:

```go
// Save system message
systemMsg := agent.Memory.GetMessages()[0]

// Clear memory
agent.ClearMemory()

// Restore system message
agent.Memory.AddMessage(systemMsg)
```

---

## Memory vs Context Window

### Memory (HNO)
- Managed by HNO
- Configurable message limit
- Automatic truncation

### Context Window (LLM)
- Model-specific limit (e.g., 128K tokens)
- Managed by LLM provider
- Can cause errors if exceeded

**Best Practice**: Keep Memory limit below LLM context window.

```go
// GPT-4o-mini: 128K tokens ≈ 100K words ≈ 400 messages
memory := memory.New(memory.Config{MaxMessages: 200})
```

---

## Troubleshooting

### Agent Doesn't Remember

Check memory configuration:

```go
// Bad ❌ - No memory
agent, _ := agent.New(agent.Config{
    Model:  model,
    Memory: nil,
})

// Good ✅ - With memory
agent, _ := agent.New(agent.Config{
    Model: model,
    // Memory enabled by default
})
```

### Memory Too Large

Reduce message limit:

```go
memory := memory.New(memory.Config{
    MaxMessages: 50,  // Smaller limit
})
```

### Lost Context

Don't clear memory unnecessarily:

```go
// Bad ❌ - Clears after each message
output, _ := agent.Run(ctx, input)
agent.ClearMemory() // DON'T DO THIS

// Good ✅ - Preserves context
output, _ := agent.Run(ctx, input)
// Memory maintained for next interaction
```

---

## Examples

### Multi-Turn Conversation

```go
agent, _ := agent.New(agent.Config{Model: model})

// Turn 1
agent.Run(ctx, "I'm planning a trip to Paris")

// Turn 2
agent.Run(ctx, "What's the weather like there?")
// Agent knows "there" = Paris

// Turn 3
agent.Run(ctx, "What should I pack?")
// Agent knows about Paris and weather
```

### Session Management

```go
type SessionManager struct {
    agents map[string]*agent.Agent
}

func (sm *SessionManager) GetAgent(sessionID string) *agent.Agent {
    if ag, exists := sm.agents[sessionID]; exists {
        return ag
    }

    // Create new agent for session
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

## Next Steps

- Build [Teams](/guide/team) with shared memory
- Add [Tools](/guide/tools) to enhance capabilities
- Create [Workflows](/guide/workflow) with context passing
- Check [Memory API Reference](/api/memory) for detailed docs

---

## Related Examples

- [Simple Agent](/examples/simple-agent) - Basic memory usage
- [Multi-Turn Chat](/examples/chat-agent) - Conversation example
- [Session Management](/examples/session-demo) - Session-based memory
