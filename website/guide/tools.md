# Tools - Extend Agent Capabilities

Give your agents access to external functionality with tools.

---

## What are Tools?

Tools are functions that agents can call to interact with external systems, perform calculations, fetch data, and more. HNO provides a flexible toolkit system for extending agent capabilities.

### Built-in Tools

- **Calculator**: Basic math operations
- **HTTP**: Make web requests
- **File**: Read/write files with safety controls
- **Google Sheets** ⭐: Read/write Google Sheets data
- **Claude Agent Skills** ⭐ NEW (v1.2.6): Call Anthropic Agent Skills via `invoke_claude_skill`
- **Tavily** ⭐ NEW (v1.2.6): Perform quick answers and reader-mode extractions
- **Gmail** ⭐ NEW (v1.2.6): Mark messages as read or archive via Gmail API
- **Jira Worklog** ⭐ NEW (v1.2.6): Summarise and export Jira Cloud worklogs
- **ElevenLabs Voice** ⭐ NEW (v1.2.6): Generate speech audio clips on demand

---

## Using Tools

### Basic Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
    "github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
    "github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    agent, _ := agent.New(agent.Config{
        Name:     "Math Assistant",
        Model:    model,
        Toolkits: []toolkit.Toolkit{calculator.New()},
    })

    output, _ := agent.Run(context.Background(), "What is 23 * 47?")
    fmt.Println(output.Content) // Agent uses calculator automatically
}
```

---

## Calculator Tool

Perform mathematical operations.

### Operations

- `add(a, b)` - Addition
- `subtract(a, b)` - Subtraction
- `multiply(a, b)` - Multiplication
- `divide(a, b)` - Division

### Example

```go
import "github.com/rexleimo/agno-go/pkg/hno/tools/calculator"

agent, _ := agent.New(agent.Config{
    Model:    model,
    Toolkits: []toolkit.Toolkit{calculator.New()},
})

// Agent will automatically use calculator
output, _ := agent.Run(ctx, "Calculate 15% tip on $85")
```

---

## HTTP Tool

Make HTTP requests to external APIs.

### Methods

- `get(url)` - HTTP GET request
- `post(url, body)` - HTTP POST request

### Example

```go
import "github.com/rexleimo/agno-go/pkg/hno/tools/http"

agent, _ := agent.New(agent.Config{
    Model:    model,
    Toolkits: []toolkit.Toolkit{http.New()},
})

// Agent can fetch data from APIs
output, _ := agent.Run(ctx, "Get the latest GitHub status from https://www.githubstatus.com/api/v2/status.json")
```

### Configuration

Control allowed domains for security:

```go
httpTool := http.New(http.Config{
    AllowedDomains: []string{"api.github.com", "api.weather.com"},
    Timeout:        10 * time.Second,
})
```

---

## File Tool

For Agent-facing file access, prefer a root-bound sandbox. The sandbox separates read and write capabilities, accepts root-relative paths only, uses `os.Root` for the actual filesystem operation, applies byte limits, and can emit audit records.

```go
import (
    "github.com/rexleimo/agno-go/pkg/hno/tools/file"
    "github.com/rexleimo/agno-go/pkg/hno/tools/filegen"
)

sandbox, err := file.NewSandbox(
    file.WithReadRoots("./agent-inputs"),
    file.WithWriteRoot("./agent-workspace"),
)
if err != nil {
    log.Fatal(err)
}
defer sandbox.Close()

agent, err := agent.New(agent.Config{
    Model: model,
    Toolkits: []toolkit.Toolkit{
        file.NewWithSandbox(sandbox),
        filegen.NewWithSandbox(sandbox),
    },
})
```

In this configuration, `"report.txt"` and `"drafts/summary.md"` are valid Agent paths, while absolute paths, `..` traversal, and escaping symbolic links are denied. Existing generated files require both `overwrite: true` in a `create_file` call and `file.WithAllowOverwrite(true)` in the sandbox configuration. When multiple toolkits share one `Sandbox`, call `sandbox.Close()` once after all work finishes; closing either toolkit closes the shared sandbox.

::: warning
`file.New()` remains unrestricted for compatibility and is suitable only for trusted callers. A file-path sandbox is not an OS/process sandbox; use per-run volumes, ACLs, execution isolation, quotas, and egress controls for untrusted or multi-tenant workloads.
:::

See the [Sandboxed File I/O guide](/guide/sandboxed-file-io) for the architecture, full API contract, and deployment boundary.

---

## Multiple Tools

Agents can use multiple tools:

```go
sandbox, err := file.NewSandbox(
    file.WithReadRoots("./data"),
    file.WithWriteRoot("./workspace"),
)
if err != nil {
    log.Fatal(err)
}
defer sandbox.Close()

fileTool := file.NewWithSandbox(sandbox)

ag, err := agent.New(agent.Config{
    Name:  "Multi-Tool Agent",
    Model: model,
    Toolkits: []toolkit.Toolkit{
        calculator.New(),
        http.New(),
        fileTool,
    },
})

// Agent can now calculate, fetch data, and read files
output, _ := ag.Run(ctx,
    "Fetch weather data, calculate average temperature, and save to file")
```

---

## External API Tools Use Live Data

The integrations below do not fall back to sample records. They call their
provider and return an error when credentials are missing or the provider
rejects the request. Configure credentials outside prompts and source code:

| Toolkit | Provider | Credentials |
| --- | --- | --- |
| `bitbucket.New()` | Bitbucket Cloud REST API | `username` + app password in the tool call |
| `confluence.New()` | Confluence REST API | `base_url`, username, and API token in the tool call |
| `airflow.New()` | Airflow REST API | `base_url`, username, and password in the tool call |
| `youtube.New()` | YouTube Data API v3 | `YOUTUBE_API_KEY` |
| `websearch.New()` | Tavily Search API | `TAVILY_API_KEY` |
| `hackernews.New()` | Hacker News Firebase + Algolia Search API | none |

For tests or API-compatible gateways, the YouTube and Tavily toolkits expose
`NewWithConfig`, while Bitbucket and Hacker News expose `NewWithBase` (Hacker
News also has `NewWithBases`). These constructors only change the endpoint;
they do not reintroduce mock data.

```go
youtubeTool := youtube.NewWithConfig(youtube.Config{
    APIKey: os.Getenv("YOUTUBE_API_KEY"),
})
searchTool := websearch.NewWithConfig(websearch.Config{
    APIKey: os.Getenv("TAVILY_API_KEY"),
})
```

---

## Creating Custom Tools

Build your own tools by implementing the Toolkit interface.

### Step 1: Create Toolkit Struct

```go
package mytool

import "github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"

type MyToolkit struct {
    *toolkit.BaseToolkit
}

func New() *MyToolkit {
    t := &MyToolkit{
        BaseToolkit: toolkit.NewBaseToolkit("my_tools"),
    }

    // Register functions
    t.RegisterFunction(&toolkit.Function{
        Name:        "greet",
        Description: "Greet a person by name",
        Parameters: map[string]toolkit.Parameter{
            "name": {
                Type:        "string",
                Description: "Person's name",
                Required:    true,
            },
        },
        Handler: t.greet,
    })

    return t
}
```

### Step 2: Implement Handler

```go
func (t *MyToolkit) greet(args map[string]interface{}) (interface{}, error) {
    name, ok := args["name"].(string)
    if !ok {
        return nil, fmt.Errorf("name must be a string")
    }

    return fmt.Sprintf("Hello, %s!", name), nil
}
```

### Step 3: Use Your Tool

```go
agent, _ := agent.New(agent.Config{
    Model:    model,
    Toolkits: []toolkit.Toolkit{mytool.New()},
})

output, _ := agent.Run(ctx, "Greet Alice")
// Agent calls greet("Alice") and responds with "Hello, Alice!"
```

---

## Advanced Custom Tool Example

Database query tool:

```go
type DatabaseToolkit struct {
    *toolkit.BaseToolkit
    db *sql.DB
}

func NewDatabaseToolkit(db *sql.DB) *DatabaseToolkit {
    t := &DatabaseToolkit{
        BaseToolkit: toolkit.NewBaseToolkit("database"),
        db:          db,
    }

    t.RegisterFunction(&toolkit.Function{
        Name:        "query_users",
        Description: "Query users from database",
        Parameters: map[string]toolkit.Parameter{
            "limit": {
                Type:        "integer",
                Description: "Maximum number of results",
                Required:    false,
            },
        },
        Handler: t.queryUsers,
    })

    return t
}

func (t *DatabaseToolkit) queryUsers(args map[string]interface{}) (interface{}, error) {
    limit := 10
    if l, ok := args["limit"].(float64); ok {
        limit = int(l)
    }

    rows, err := t.db.Query("SELECT id, name FROM users LIMIT ?", limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var users []map[string]interface{}
    for rows.Next() {
        var id int
        var name string
        rows.Scan(&id, &name)
        users = append(users, map[string]interface{}{
            "id":   id,
            "name": name,
        })
    }

    return users, nil
}
```

---

## Tool Best Practices

### 1. Clear Descriptions

Help the agent understand when to use tools:

```go
// Good ✅
Description: "Calculate the square root of a number. Use when user asks for square roots."

// Bad ❌
Description: "Math function"
```

### 2. Validate Input

Always validate tool parameters:

```go
func (t *MyToolkit) divide(args map[string]interface{}) (interface{}, error) {
    b, ok := args["divisor"].(float64)
    if !ok || b == 0 {
        return nil, fmt.Errorf("divisor must be a non-zero number")
    }
    // ... perform division
}
```

### 3. Error Handling

Return meaningful errors:

```go
func (t *MyToolkit) fetchData(args map[string]interface{}) (interface{}, error) {
    resp, err := http.Get(url)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch data: %w", err)
    }
    // ... process response
}
```

### 4. Security

Restrict tool capabilities:

```go
// A sandbox with read roots but no write root is read-only.
readOnlySandbox, err := file.NewSandbox(
    file.WithReadRoots("/safe/path"),
)
if err != nil {
    log.Fatal(err)
}
defer readOnlySandbox.Close()
fileTool := file.NewWithSandbox(readOnlySandbox)

// Validate domains
httpTool := http.New(http.Config{
    AllowedDomains: []string{"api.trusted.com"},
})
```

---

## Tool Execution Flow

1. User sends request to agent
2. Agent (LLM) decides if tools are needed
3. LLM generates tool call with parameters
4. HNO executes tool function
5. Tool result returned to LLM
6. LLM generates final response

### Example Flow

```
User: "What is 25 * 17?"
  ↓
LLM: "I need to use calculator"
  ↓
Tool Call: multiply(25, 17)
  ↓
Tool Result: 425
  ↓
LLM: "The answer is 425"
  ↓
User receives: "The answer is 425"
```

---

## Troubleshooting

### Agent Not Using Tools

Ensure clear instructions:

```go
agent, _ := agent.New(agent.Config{
    Model:        model,
    Toolkits:     []toolkit.Toolkit{calculator.New()},
    Instructions: "Use the calculator tool for any math operations.",
})
```

### Tool Errors

Check tool registration and parameter types:

```go
// Tool expects float64 for numbers
args := map[string]interface{}{
    "value": 42.0,  // ✅ Correct
    // "value": "42"  // ❌ Wrong type
}
```

---

## Google Sheets Tool ⭐ NEW

Read and write data to Google Sheets using service account authentication.

### Operations

- `read_range(spreadsheet_id, range)` - Read data from specified range
- `write_range(spreadsheet_id, range, values)` - Write data to specified range
- `append_rows(spreadsheet_id, range, values)` - Append rows to spreadsheet

### Setup

1. **Create Service Account**:
   - Go to Google Cloud Console
   - Create a service account
   - Download JSON credentials file

2. **Share Spreadsheet**:
   - Share your Google Sheet with the service account email
   - Grant "Editor" permissions

### Usage

```go
import "github.com/rexleimo/agno-go/pkg/hno/tools/googlesheets"

// Load credentials from JSON file
sheetsTool, err := googlesheets.New(googlesheets.Config{
    CredentialsFile: "./service-account.json",
})

// Or use JSON string
sheetsTool, err := googlesheets.New(googlesheets.Config{
    CredentialsJSON: os.Getenv("GOOGLE_SHEETS_CREDENTIALS"),
})

agent, _ := agent.New(agent.Config{
    Model:    model,
    Toolkits: []toolkit.Toolkit{sheetsTool},
})
```

### Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
    "github.com/rexleimo/agno-go/pkg/hno/tools/googlesheets"
    "github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    sheetsTool, err := googlesheets.New(googlesheets.Config{
        CredentialsFile: "./service-account.json",
    })
    if err != nil {
        log.Fatal(err)
    }

    agent, _ := agent.New(agent.Config{
        Name:     "Data Analyst",
        Model:    model,
        Toolkits: []toolkit.Toolkit{sheetsTool},
    })

    // Agent can read and analyze spreadsheet data
    output, _ := agent.Run(context.Background(),
        "Read the sales data from Sheet1!A1:D100 and summarize the total revenue")

    fmt.Println(output.Content)
}
```

### Range Notation

Use standard A1 notation for ranges:
- `Sheet1!A1:B10` - Cells A1 to B10 in Sheet1
- `Sheet2!A:A` - Entire column A in Sheet2
- `Sheet1!1:5` - Rows 1 to 5 in Sheet1

### Security

- Service account authentication (no user interaction)
- Spreadsheet-level permissions
- Read/write access controlled by sharing settings

## Next Steps

- Learn about [Memory](/guide/memory) for conversation state
- Build [Teams](/guide/team) with specialized tool agents
- Explore [Workflow](/guide/workflow) for tool orchestration
- Check [API Reference](/api/tools) for detailed docs

---

## Related Examples

- [Simple Agent](/examples/simple-agent) - Calculator tool usage
- [Search Agent](/examples/search-agent) - HTTP tool for web search
- [File Agent](/examples/file-agent) - File operations
