# Tools API 参考 / Tools API Reference

## Calculator

基础数学运算。/ Basic math operations.

**创建 / Create:**
```go
func New() *Calculator
```

**函数 / Functions:**
- `add(a, b)`: 加法 / Addition
- `subtract(a, b)`: 减法 / Subtraction
- `multiply(a, b)`: 乘法 / Multiplication
- `divide(a, b)`: 除法 / Division

**示例 / Example:**
```go
calc := calculator.New()

ag, _ := agent.New(agent.Config{
    Toolkits: []toolkit.Toolkit{calc},
    // ...
})
```

## HTTP

用于 GET/POST 请求的 HTTP 客户端。/ HTTP client for GET/POST requests.

**创建 / Create:**
```go
func New() *HTTPToolkit
```

**函数 / Functions:**
- `http_get(url)`: HTTP GET 请求 / HTTP GET request
- `http_post(url, body)`: HTTP POST 请求 / HTTP POST request

**示例 / Example:**
```go
http := http.New()

ag, _ := agent.New(agent.Config{
    Toolkits: []toolkit.Toolkit{http},
    Instructions: "You can make HTTP requests to fetch data.",
})
```

## File

带显式能力控制的沙盒化文件操作。对于 Agent 面向的路径，请使用 `Sandbox` 和 `NewWithSandbox`；`New()` 仅为可信调用方保留不受限制的旧行为。

**创建：**
```go
func New() *FileTools
func NewWithBaseDir(baseDir string) *FileTools
func NewWithSandbox(sandbox *Sandbox) *FileTools

func NewSandbox(options ...SandboxOption) (*Sandbox, error)
```

**Sandbox 选项：**
```go
file.WithReadRoots("./inputs", "./templates")
file.WithWriteRoot("./workspace")
file.WithMaxReadBytes(10 << 20)
file.WithMaxWriteBytes(5 << 20)
file.WithAllowOverwrite(true)
file.WithAudit(func(entry file.AuditEntry) { /* 记录审计事件 */ })
```

`NewSandbox` 默认 fail-closed：没有读根就拒绝读取，没有写根就拒绝写入。沙盒路径必须相对其配置根目录。绝对路径、路径穿越、逃逸性符号链接和 Windows 特殊路径都会被拒绝。

**函数：**
- `read_file(path)`：读取读根下的一个普通文件。
- `write_file(path, content)`：根据 sandbox 覆盖策略，在写根下创建或替换文件。
- `list_files(path)`：列出读根下的一个目录。
- `file_exists(path)`：查询读根下的文件元数据。
- `read_pptx(path)`：从通过读根打开的文件中提取 PPTX 幻灯片文本。
- `delete_file(path)`：删除写根下的文件或空目录。

**文件生成：**
```go
func filegen.New() *filegen.FileGenToolkit
func filegen.NewWithSandbox(sandbox *file.Sandbox) *filegen.FileGenToolkit
```

- `create_file(file_path, content, overwrite)`：在写根下创建产物。覆盖已有文件必须同时满足 `overwrite: true` 和 `WithAllowOverwrite(true)`。
- `create_directory(dir_path)`：在写根下创建目录；目标已存在时返回错误。
- `generate_from_template(template, variables)`：仅在内存中替换模板，不访问文件系统。

**示例：**
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

架构、生命周期、限制和部署建议请参阅[沙盒化文件 I/O 指南](/zh/guide/sandboxed-file-io)。

## 自定义工具 / Custom Tools

创建自定义工具 / Create custom tools:

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
    // 处理输入 / Process input
    return result, nil
}
```
