# Tools - 扩展 Agent 能力

通过工具为您的 Agent 提供外部功能访问。

---

## 什么是 Tools?

Tools 是 Agent 可以调用的函数,用于与外部系统交互、执行计算、获取数据等。HNO 提供了灵活的工具系统用于扩展 Agent 能力。

### 内置工具

- **Calculator**: 基础数学运算
- **HTTP**: 发起网络请求
- **File**: 带安全控制的读写文件
- **Google Sheets** ⭐ 新增 (v1.2.1): 读写 Google Sheets 数据
- **Claude Agent Skills** ⭐ 新增 (v1.2.6): 通过 `invoke_claude_skill` 调用 Anthropic Agent Skills
- **Tavily** ⭐ 新增 (v1.2.6): 快速获取答案与 Reader 模式提取
- **Gmail** ⭐ 新增 (v1.2.6): 通过 Gmail API 标记邮件已读或归档
- **Jira Worklog** ⭐ 新增 (v1.2.6): 汇总并导出 Jira Cloud 工时
- **ElevenLabs Voice** ⭐ 新增 (v1.2.6): 按需生成语音音频片段

---

## 使用 Tools

### 基础示例

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

## Calculator 工具

执行数学运算。

### 操作

- `add(a, b)` - 加法
- `subtract(a, b)` - 减法
- `multiply(a, b)` - 乘法
- `divide(a, b)` - 除法

### 示例

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

## HTTP 工具

向外部 API 发起 HTTP 请求。

### 方法

- `get(url)` - HTTP GET 请求
- `post(url, body)` - HTTP POST 请求

### 示例

```go
import "github.com/rexleimo/agno-go/pkg/hno/tools/http"

agent, _ := agent.New(agent.Config{
    Model:    model,
    Toolkits: []toolkit.Toolkit{http.New()},
})

// Agent can fetch data from APIs
output, _ := agent.Run(ctx, "Get the latest GitHub status from https://www.githubstatus.com/api/v2/status.json")
```

### 配置

出于安全考虑控制允许的域名:

```go
httpTool := http.New(http.Config{
    AllowedDomains: []string{"api.github.com", "api.weather.com"},
    Timeout:        10 * time.Second,
})
```

---

## File 工具

对于 Agent 面向的文件访问，请优先使用根句柄绑定的 sandbox。Sandbox 将读写能力分离，只接收根相对路径，使用 `os.Root` 执行真实文件操作，应用字节上限，并可输出审计记录。

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

在此配置中，`"report.txt"` 和 `"drafts/summary.md"` 是有效的 Agent 路径；绝对路径、`..` 穿越以及会逃出根目录的符号链接都会被拒绝。已有生成文件必须同时满足 `create_file` 调用中的 `overwrite: true` 和 sandbox 配置中的 `file.WithAllowOverwrite(true)` 才能覆盖。多个 Toolkit 共用一个 `Sandbox` 时，应在所有工作结束后统一调用 `sandbox.Close()`；关闭任意一个 Toolkit 都会关闭共享 sandbox。

::: warning
为兼容性保留的 `file.New()` 没有文件系统限制，只适用于可信调用方。文件路径 sandbox 不是 OS/进程 sandbox；面对不可信或多租户工作负载，仍需使用每次运行独立卷、ACL、执行隔离、配额和 egress 控制。
:::

完整架构、API 契约和部署边界请参阅[沙盒化文件 I/O 指南](/zh/guide/sandboxed-file-io)。

---

## 多个工具

Agent 可以使用多个工具:

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

## 创建自定义工具

通过实现 Toolkit 接口构建您自己的工具。

### 步骤 1: 创建 Toolkit 结构

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

### 步骤 2: 实现 Handler

```go
func (t *MyToolkit) greet(args map[string]interface{}) (interface{}, error) {
    name, ok := args["name"].(string)
    if !ok {
        return nil, fmt.Errorf("name must be a string")
    }

    return fmt.Sprintf("Hello, %s!", name), nil
}
```

### 步骤 3: 使用您的工具

```go
agent, _ := agent.New(agent.Config{
    Model:    model,
    Toolkits: []toolkit.Toolkit{mytool.New()},
})

output, _ := agent.Run(ctx, "Greet Alice")
// Agent calls greet("Alice") and responds with "Hello, Alice!"
```

---

## 高级自定义工具示例

数据库查询工具:

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

## 工具最佳实践

### 1. 清晰的描述

帮助 Agent 理解何时使用工具:

```go
// Good ✅
Description: "Calculate the square root of a number. Use when user asks for square roots."

// Bad ❌
Description: "Math function"
```

### 2. 验证输入

始终验证工具参数:

```go
func (t *MyToolkit) divide(args map[string]interface{}) (interface{}, error) {
    b, ok := args["divisor"].(float64)
    if !ok || b == 0 {
        return nil, fmt.Errorf("divisor must be a non-zero number")
    }
    // ... perform division
}
```

### 3. 错误处理

返回有意义的错误:

```go
func (t *MyToolkit) fetchData(args map[string]interface{}) (interface{}, error) {
    resp, err := http.Get(url)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch data: %w", err)
    }
    // ... process response
}
```

### 4. 安全性

限制工具能力:

```go
// 只有读根、没有写根的 sandbox 是只读的。
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

## 工具执行流程

1. 用户向 Agent 发送请求
2. Agent (LLM) 决定是否需要工具
3. LLM 生成带参数的工具调用
4. HNO 执行工具函数
5. 工具结果返回给 LLM
6. LLM 生成最终响应

### 示例流程

```
用户: "What is 25 * 17?"
  ↓
LLM: "I need to use calculator"
  ↓
Tool Call: multiply(25, 17)
  ↓
Tool Result: 425
  ↓
LLM: "The answer is 425"
  ↓
用户收到: "The answer is 425"
```

---

## 故障排除

### Agent 不使用工具

确保指令清晰:

```go
agent, _ := agent.New(agent.Config{
    Model:        model,
    Toolkits:     []toolkit.Toolkit{calculator.New()},
    Instructions: "Use the calculator tool for any math operations.",
})
```

### 工具错误

检查工具注册和参数类型:

```go
// Tool expects float64 for numbers
args := map[string]interface{}{
    "value": 42.0,  // ✅ Correct
    // "value": "42"  // ❌ Wrong type
}
```

---

## Google Sheets 工具 ⭐ 新增

使用服务账号认证读写 Google Sheets 数据。

### 操作

- `read_range(spreadsheet_id, range)` - 从指定范围读取数据
- `write_range(spreadsheet_id, range, values)` - 向指定范围写入数据
- `append_rows(spreadsheet_id, range, values)` - 向电子表格追加行

### 设置

1. **创建服务账号**:
   - 前往 Google Cloud Console
   - 创建服务账号
   - 下载 JSON 凭据文件

2. **共享电子表格**:
   - 将您的 Google Sheet 共享给服务账号邮箱
   - 授予"编辑者"权限

### 使用方法

```go
import "github.com/rexleimo/agno-go/pkg/hno/tools/googlesheets"

// 从 JSON 文件加载凭据
sheetsTool, err := googlesheets.New(googlesheets.Config{
    CredentialsFile: "./service-account.json",
})

// 或使用 JSON 字符串
sheetsTool, err := googlesheets.New(googlesheets.Config{
    CredentialsJSON: os.Getenv("GOOGLE_SHEETS_CREDENTIALS"),
})

agent, _ := agent.New(agent.Config{
    Model:    model,
    Toolkits: []toolkit.Toolkit{sheetsTool},
})
```

### 示例

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
        Name:     "数据分析师",
        Model:    model,
        Toolkits: []toolkit.Toolkit{sheetsTool},
    })

    // Agent 可以读取和分析电子表格数据
    output, _ := agent.Run(context.Background(),
        "从 Sheet1!A1:D100 读取销售数据并总结总收入")

    fmt.Println(output.Content)
}
```

### 范围表示法

使用标准 A1 表示法表示范围：
- `Sheet1!A1:B10` - Sheet1 中的 A1 到 B10 单元格
- `Sheet2!A:A` - Sheet2 中的整个 A 列
- `Sheet1!1:5` - Sheet1 中的 1 到 5 行

### 安全性

- 服务账号认证（无需用户交互）
- 电子表格级别权限
- 通过共享设置控制读写访问

## 下一步

- 了解 [Memory](/guide/memory) 的对话状态
- 使用专业化工具 Agent 构建 [Teams](/guide/team)
- 探索 [Workflow](/guide/workflow) 的工具编排
- 查看 [API Reference](/api/tools) 获取详细文档

---

## 相关示例

- [Simple Agent](/examples/simple-agent) - Calculator 工具使用
- [Search Agent](/examples/search-agent) - HTTP 工具进行网络搜索
- [File Agent](/examples/file-agent) - 文件操作
