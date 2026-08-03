# Examples

Working examples demonstrating all features of HNO.

## Available Examples

### 1. Simple Agent

Basic agent with calculator tools.

**Location**: `cmd/examples/simple_agent/`

**Features**:
- OpenAI GPT-4o-mini integration
- Calculator toolkit
- Basic conversation

**Run**:
```bash
export OPENAI_API_KEY=sk-your-key
go run cmd/examples/simple_agent/main.go
```

[View Source](https://github.com/rexleimo/HNO/tree/main/cmd/examples/simple_agent)

---

### 2. Claude Agent

Anthropic Claude integration with tools.

**Location**: `cmd/examples/claude_agent/`

**Features**:
- Anthropic Claude 3.5 Sonnet
- HTTP and Calculator tools
- Error handling examples

**Run**:
```bash
export ANTHROPIC_API_KEY=sk-ant-your-key
go run cmd/examples/claude_agent/main.go
```

[View Source](https://github.com/rexleimo/HNO/tree/main/cmd/examples/claude_agent)

---

### 3. Ollama Agent

Local model support with Ollama.

**Location**: `cmd/examples/ollama_agent/`

**Features**:
- Local Llama 3 model
- Privacy-focused (no API calls)
- File operations toolkit

**Setup**:
```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Pull model
ollama pull llama3

# Run example
go run cmd/examples/ollama_agent/main.go
```

[View Source](https://github.com/rexleimo/HNO/tree/main/cmd/examples/ollama_agent)

---

### 4. Team Demo

Multi-agent collaboration with different coordination modes.

**Location**: `cmd/examples/team_demo/`

**Features**:
- 4 coordination modes (Sequential, Parallel, Leader-Follower, Consensus)
- Researcher + Writer team
- Real-world workflow

**Run**:
```bash
export OPENAI_API_KEY=sk-your-key
go run cmd/examples/team_demo/main.go
```

[View Source](https://github.com/rexleimo/HNO/tree/main/cmd/examples/team_demo)

---

### 5. Workflow Demo

Step-based orchestration with control flow primitives.

**Location**: `cmd/examples/workflow_demo/`

**Features**:
- 5 workflow primitives (Step, Condition, Loop, Parallel, Router)
- Sentiment analysis workflow
- Conditional routing

**Run**:
```bash
export OPENAI_API_KEY=sk-your-key
go run cmd/examples/workflow_demo/main.go
```

[View Source](https://github.com/rexleimo/HNO/tree/main/cmd/examples/workflow_demo)

---

### 6. RAG Demo

Retrieval-Augmented Generation with ChromaDB.

**Location**: `cmd/examples/rag_demo/`

**Features**:
- ChromaDB vector database
- OpenAI embeddings
- Semantic search
- Document Q&A

**Setup**:
```bash
# Start ChromaDB (Docker)
docker run -d -p 8000:8000 chromadb/chroma

# Set API keys
export OPENAI_API_KEY=sk-your-key

# Run example
go run cmd/examples/rag_demo/main.go
```

[View Source](https://github.com/rexleimo/HNO/tree/main/cmd/examples/rag_demo)

---

### 7. Logfire Observability

Stream reasoning metadata and token usage to Logfire via OpenTelemetry.

**Location**: `cmd/examples/logfire_observability/`

**Features**:
- OTLP/HTTP exporter with configurable endpoint (EU/US)
- Reasoning content and token metrics as span events
- Works with any reasoning-capable model (OpenAI o-series, Gemini 2.5, Claude w/ thinking)

**Run**:
```bash
export OPENAI_API_KEY=sk-your-key
export LOGFIRE_WRITE_TOKEN=lf_your_token
go run -tags logfire cmd/examples/logfire_observability/main.go
```

[View Source](https://github.com/rexleimo/HNO/tree/main/cmd/examples/logfire_observability)

---

## Code Snippets

### Agent with Multiple Tools

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/rexleimo/HNO/pkg/hno/agent"
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
    "github.com/rexleimo/HNO/pkg/hno/tools/calculator"
    "github.com/rexleimo/HNO/pkg/hno/tools/http"
    "github.com/rexleimo/HNO/pkg/hno/tools/toolkit"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    ag, _ := agent.New(agent.Config{
        Name:  "Smart Assistant",
        Model: model,
        Toolkits: []toolkit.Toolkit{
            calculator.New(),
            http.New(),
        },
        Instructions: "You can do math and make HTTP requests",
    })

    output, _ := ag.Run(context.Background(),
        "Calculate 15 * 23 and fetch https://api.github.com")
    fmt.Println(output.Content)
}
```

### Multi-Agent Team

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/rexleimo/HNO/pkg/hno/agent"
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
    "github.com/rexleimo/HNO/pkg/hno/team"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    researcher, _ := agent.New(agent.Config{
        Name:         "Researcher",
        Model:        model,
        Instructions: "Research and gather information",
    })

    writer, _ := agent.New(agent.Config{
        Name:         "Writer",
        Model:        model,
        Instructions: "Create compelling content",
    })

    tm, _ := team.New(team.Config{
        Name:   "Content Team",
        Agents: []*agent.Agent{researcher, writer},
        Mode:   team.ModeSequential,
    })

    output, _ := tm.Run(context.Background(),
        "Write a short article about Go programming")
    fmt.Println(output.Content)
}
```

### Workflow with Conditions

```go
package main

import (
    "context"
    "fmt"
    "os"
    "strings"

    "github.com/rexleimo/HNO/pkg/hno/agent"
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
    "github.com/rexleimo/HNO/pkg/hno/workflow"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    classifier, _ := agent.New(agent.Config{
        Name:         "Classifier",
        Model:        model,
        Instructions: "Classify sentiment as positive or negative",
    })

    positiveHandler, _ := agent.New(agent.Config{
        Name:         "Positive Handler",
        Model:        model,
        Instructions: "Respond enthusiastically",
    })

    negativeHandler, _ := agent.New(agent.Config{
        Name:         "Negative Handler",
        Model:        model,
        Instructions: "Respond empathetically",
    })

    wf, _ := workflow.New(workflow.Config{
        Name: "Sentiment Workflow",
        Steps: []workflow.Primitive{
            workflow.NewStep("classify", classifier),
            workflow.NewCondition("route",
                func(ctx *workflow.ExecutionContext) bool {
                    result := ctx.GetResult("classify")
                    return strings.Contains(result.Content, "positive")
                },
                workflow.NewStep("positive", positiveHandler),
                workflow.NewStep("negative", negativeHandler),
            ),
        },
    })

    output, _ := wf.Run(context.Background(), "I love this!")
    fmt.Println(output.Content)
}
```

## Learn More

- [Quick Start](/guide/quick-start) - Get started in 5 minutes
- [Agent Guide](/guide/agent) - Learn about agents
- [Team Guide](/guide/team) - Multi-agent collaboration
- [Workflow Guide](/guide/workflow) - Orchestration patterns
- [API Reference](/api/) - Complete API documentation

## Contributing Examples

Have an interesting example? Contribute to the repository:

1. Fork the repository
2. Create your example in `cmd/examples/your_example/`
3. Add README.md with description and usage
4. Submit a pull request

[Contribution Guidelines](https://github.com/rexleimo/HNO/blob/main/CONTRIBUTING.md)
