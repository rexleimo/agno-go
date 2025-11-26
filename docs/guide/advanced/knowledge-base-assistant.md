
# Advanced Guide: Knowledge Base Assistant (Go-first)

This guide outlines how to build an assistant that answers questions based on
your own documents, using **Go provider clients** and your own retrieval stack.
It focuses on how to structure requests to the existing Go API surface, and
intentionally avoids relying on the unfinished HTTP runtime.

## 1. High-level scenario

You want an assistant that can answer questions about:

- Product documentation  
- Internal guidelines or policies  
- Knowledge base articles  

At a high level:

1. Offline, you embed your documents into a vector store (any database or service
   you prefer).  
2. At query time, you retrieve the most relevant passages based on the user’s
   question.  
3. You pass those passages into the model via `agent.Message.Content`.  

Agno-Go does **not** ship a built-in vector store; you bring your own storage
and just use the provider clients to run embeddings and chat.

## 2. Embedding documents with a provider client

You can use any provider that implements `model.EmbeddingProvider`. The exact
model IDs and capabilities are listed in the Provider Matrix and `.env.example`.

```go
package main

import (
  "context"
  "log"
  "os"
  "time"

  "github.com/rexleimo/agno-go/internal/agent"
  "github.com/rexleimo/agno-go/internal/model"
  "github.com/rexleimo/agno-go/pkg/providers/openai"
)

func embedDoc(ctx context.Context, text string) ([]float64, error) {
  apiKey := os.Getenv("OPENAI_API_KEY")
  if apiKey == "" {
    return nil, fmt.Errorf("OPENAI_API_KEY not set")
  }

  client := openai.New("", apiKey, nil)

  resp, err := client.Embed(ctx, model.EmbeddingRequest{
    Model: agent.ModelConfig{
      Provider: agent.ProviderOpenAI,
      ModelID:  "text-embedding-3-small", // choose a suitable embedding model
    },
    Input: []string{text},
  })
  if err != nil {
    return nil, err
  }
  if len(resp.Vectors) == 0 {
    return nil, fmt.Errorf("empty embedding response")
  }
  return resp.Vectors[0], nil
}
```

How you store these vectors (Postgres, ClickHouse, dedicated vector DB, etc.) is
up to you and outside the scope of Agno-Go.

## 3. Answering questions with retrieved context

Once you have a way to retrieve relevant passages (for example, `[]string`
containing the top-k chunks), you can pass them into a chat request:

```go
func answerWithContext(
  ctx context.Context,
  client model.ChatProvider,
  provider agent.Provider,
  modelID string,
  question string,
  passages []string,
) (string, error) {
  var contextText string
  for _, p := range passages {
    contextText += "- " + p + "\n"
  }

  prompt := fmt.Sprintf(
    "You are a helpful assistant.\n\nCONTEXT:\n%s\nQUESTION: %s\n\nAnswer in a concise way and say “I don't know” if the answer is not in the context.",
    contextText,
    question,
  )

  resp, err := client.Chat(ctx, model.ChatRequest{
    Model: agent.ModelConfig{
      Provider: provider,
      ModelID:  modelID,
    },
    Messages: []agent.Message{
      {Role: agent.RoleUser, Content: prompt},
    },
  })
  if err != nil {
    return "", err
  }
  return resp.Message.Content, nil
}
```

You can plug in any `ChatProvider` here (OpenAI, Gemini, Groq, …) as long as
you have configured the corresponding env vars.

## 4. Putting it together

A complete knowledge base assistant in Go typically has three pieces:

- **Indexer** – reads documents, calls `Embed` on a provider client, stores
  vectors + metadata in your own store.  
- **Retriever** – given a question, finds relevant passages and returns text
  chunks.  
- **Answerer** – calls `Chat` with a prompt that includes the retrieved
  context, using the pattern above.  

Agno-Go’s responsibilities in this story are deliberately small:

- Provide consistent `ChatRequest` / `EmbeddingRequest` shapes.  
- Provide provider clients that implement the same interfaces.  
- Normalize basic error handling and provider status.  

Everything else (storage, indexing, ranking) is up to your application.

## 5. Relation to other docs

- Use the [Provider Matrix](../providers/matrix) to pick a provider and model
  with good long-context support.  
- Use [Configuration & Security](../config-and-security) to configure the
  necessary env vars (for example `OPENAI_API_KEY`, `GEMINI_API_KEY`).  
- The HTTP runtime design in the specs mirrors the same ideas (context + chat),
  but until it stabilizes, treat it as a contract reference rather than a
  ready-to-copy implementation.  

