# Tools API Reference

## Calculator

Basic math operations.

**Create:**
```go
func New() *Calculator
```

**Functions:**
- `add(a, b)`: Addition
- `subtract(a, b)`: Subtraction
- `multiply(a, b)`: Multiplication
- `divide(a, b)`: Division

**Example:**
```go
calc := calculator.New()

ag, _ := agent.New(agent.Config{
    Toolkits: []toolkit.Toolkit{calc},
    // ...
})
```

## HTTP

HTTP client for GET/POST requests.

**Create:**
```go
func New() *HTTPToolkit
```

**Functions:**
- `http_get(url)`: HTTP GET request
- `http_post(url, body)`: HTTP POST request

**Example:**
```go
http := http.New()

ag, _ := agent.New(agent.Config{
    Toolkits: []toolkit.Toolkit{http},
    Instructions: "You can make HTTP requests to fetch data.",
})
```

## File

Sandboxed file operations with explicit capabilities. For Agent-facing paths, use a `Sandbox` and `NewWithSandbox`; `New()` remains unrestricted for trusted callers only.

**Create:**
```go
func New() *FileTools
func NewWithBaseDir(baseDir string) *FileTools
func NewWithSandbox(sandbox *Sandbox) *FileTools

func NewSandbox(options ...SandboxOption) (*Sandbox, error)
```

**Sandbox options:**
```go
file.WithReadRoots("./inputs", "./templates")
file.WithWriteRoot("./workspace")
file.WithMaxReadBytes(10 << 20)
file.WithMaxWriteBytes(5 << 20)
file.WithAllowOverwrite(true)
file.WithAudit(func(entry file.AuditEntry) { /* record audit event */ })
```

`NewSandbox` is fail closed: no read root means read operations are denied, and no write root means write operations are denied. Sandboxed paths are relative to their configured root. Absolute paths, traversal, escaping symbolic links, and unsafe Windows path forms are rejected.

**Functions:**
- `read_file(path)`: Read one regular file below a read root.
- `write_file(path, content)`: Create or replace a file below the write root according to the sandbox overwrite policy.
- `list_files(path)`: List one directory below a read root.
- `file_exists(path)`: Check metadata below a read root.
- `read_pptx(path)`: Extract PPTX slide text from a file opened through a read root.
- `delete_file(path)`: Remove a file or empty directory below the write root.

**File generation:**
```go
func filegen.New() *filegen.FileGenToolkit
func filegen.NewWithSandbox(sandbox *file.Sandbox) *filegen.FileGenToolkit
```

- `create_file(file_path, content, overwrite)`: Creates a generated artifact below the write root. Replacing an existing file requires both `overwrite: true` and `WithAllowOverwrite(true)`.
- `create_directory(dir_path)`: Creates a directory below the write root; an existing target is an error.
- `generate_from_template(template, variables)`: Performs in-memory substitution and does not access the filesystem.

**Example:**
```go
sandbox, err := file.NewSandbox(
    file.WithReadRoots("./inputs"),
    file.WithWriteRoot("./workspace"),
)
if err != nil {
    log.Fatal(err)
}
defer sandbox.Close()

ag, err := agent.New(agent.Config{
    Toolkits: []toolkit.Toolkit{
        file.NewWithSandbox(sandbox),
        filegen.NewWithSandbox(sandbox),
    },
    // ...
})
```

See the [Sandboxed File I/O guide](/guide/sandboxed-file-io) for architecture, lifecycle, limitations, and deployment guidance.

## Custom Tools

Create custom tools:

```go
type MyToolkit struct {
    *toolkit.BaseToolkit
}

func NewMyToolkit() *MyToolkit {
    t := &MyToolkit{
        BaseToolkit: toolkit.NewBaseToolkit("my_tools"),
    }

    t.RegisterFunction(&toolkit.Function{
        Name:        "my_function",
        Description: "Description of what this function does",
        Parameters: map[string]toolkit.Parameter{
            "input": {
                Type:        "string",
                Description: "Input parameter description",
                Required:    true,
            },
            "optional": {
                Type:        "number",
                Description: "Optional parameter",
                Required:    false,
            },
        },
        Handler: t.myHandler,
    })

    return t
}

func (t *MyToolkit) myHandler(args map[string]interface{}) (interface{}, error) {
    input := args["input"].(string)
    // Process input
    return result, nil
}
```
