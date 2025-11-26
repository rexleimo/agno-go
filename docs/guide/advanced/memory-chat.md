
# Advanced Guide: Memory-Augmented Chat (Go-first)

This guide illustrates how to build a chat experience that takes advantage of
memory across turns and sessions **inside your own Go application**, using the
existing provider clients and your own storage. It does not depend on an HTTP
runtime being available.

## 1. Types of memory

You can think about “memory” in three layers:

- **Conversation history** – recent turns in the current conversation.  
- **User profile** – long-lived information about a user (preferences, profile
  fields, plan, etc.).  
- **Knowledge records** – domain-specific facts (support tickets, purchases,
  important events).  

Agno-Go focuses on the first layer via `ChatRequest`/`ChatResponse` and leaves
the other layers to your own systems.

## 2. Representing conversation state

At the Go API level, the conversation history is just a slice of
`agent.Message` values:

```go
var history []agent.Message

history = append(history,
  agent.Message{Role: agent.RoleUser, Content: "Hi, I like short study sessions."},
)

// later…
history = append(history,
  agent.Message{Role: agent.RoleAssistant, Content: "Great, I will keep sessions under 30 minutes."},
)
```

When you call a provider, you pass the relevant part of this history into
`model.ChatRequest.Messages`:

```go
resp, err := client.Chat(ctx, model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
  },
  Messages: history,
})
```

Your application decides:

- How many past turns to keep.  
- Whether to periodically summarize older messages.  
- How to persist history between requests (database, cache, …).  

## 3. Adding long-term memory

Long-term memory is typically stored outside the model and injected into the
prompt when needed. For example:

```go
type UserProfile struct {
  ID          string
  Preferences string // a short natural-language summary
}

func buildPrompt(profile UserProfile, recent []agent.Message) string {
  var buf strings.Builder
  buf.WriteString("You are a helpful assistant.\n\n")
  buf.WriteString("USER PROFILE:\n")
  buf.WriteString(profile.Preferences)
  buf.WriteString("\n\nRECENT CONVERSATION:\n")
  for _, m := range recent {
    buf.WriteString(string(m.Role))
    buf.WriteString(": ")
    buf.WriteString(m.Content)
    buf.WriteString("\n")
  }
  buf.WriteString("\nBased on the profile and recent conversation, answer the user.\n")
  return buf.String()
}
```

You can then pass this prompt as a single `user` message:

```go
prompt := buildPrompt(profile, recentHistory)

resp, err := client.Chat(ctx, model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
  },
  Messages: []agent.Message{
    {Role: agent.RoleUser, Content: prompt},
  },
})
```

Your own service is responsible for:

- Loading / updating `UserProfile` records from your database.  
- Deciding when to summarize or prune history.  
- Ensuring sensitive data is handled according to your policies.  

## 4. Streaming with memory

Memory-augmented chat works equally well with streaming:

```go
req := model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
    Stream:   true,
  },
  Messages: history,
}

err := client.Stream(ctx, req, func(ev model.ChatStreamEvent) error {
  if ev.Type == "token" {
    fmt.Print(ev.Delta)
  }
  if ev.Done {
    fmt.Println()
  }
  return nil
})
if err != nil {
  log.Fatalf("stream error: %v", err)
}
```

You can append the final assistant message back into `history` so that future
requests see the updated conversation.

## 5. Storage and configuration

For memory-heavy scenarios:

- Use your own database or cache (for example Postgres, Redis, key-value store)
  to persist user profiles and conversation snippets.  
- Use [Configuration & Security](../config-and-security) to decide which
  providers to enable and how to manage API keys.  
- Keep sensitive or long-lived data out of prompt text where possible; prefer
  short summaries over raw logs.  

Agno-Go does not impose a specific storage backend; it only defines the shapes
for messages and requests.

## 6. Relation to other docs

- The [Quickstart](../quickstart) shows the simplest “stateless” call flow.  
- This guide adds application-level memory on top of the same provider clients.  
- The HTTP runtime design in the specs uses the same concepts (sessions,
  messages, metadata), but until it stabilizes, treat it as an internal design,
  not a ready-made feature.  

