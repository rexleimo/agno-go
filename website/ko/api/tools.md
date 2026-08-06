# Tools API 레퍼런스

## Calculator

기본 수학 연산입니다.

**생성:**
```go
func New() *Calculator
```

**함수:**
- `add(a, b)`: 덧셈
- `subtract(a, b)`: 뺄셈
- `multiply(a, b)`: 곱셈
- `divide(a, b)`: 나눗셈

**예제:**
```go
calc := calculator.New()

ag, _ := agent.New(agent.Config{
    Toolkits: []toolkit.Toolkit{calc},
    // ...
})
```

## HTTP

GET/POST 요청을 위한 HTTP 클라이언트입니다.

**생성:**
```go
func New() *HTTPToolkit
```

**함수:**
- `http_get(url)`: HTTP GET 요청
- `http_post(url, body)`: HTTP POST 요청

**예제:**
```go
http := http.New()

ag, _ := agent.New(agent.Config{
    Toolkits: []toolkit.Toolkit{http},
    Instructions: "You can make HTTP requests to fetch data.",
})
```

## File

Agent가 지정하는 경로에는 명시적 capability를 가진 `Sandbox`와
`NewWithSandbox`를 사용합니다. `New()`는 신뢰할 수 있는 호출자를 위한 제한 없는
호환 API입니다.

**생성:**
```go
func New() *FileTools
func NewWithBaseDir(baseDir string) *FileTools
func NewWithSandbox(sandbox *Sandbox) *FileTools
func NewSandbox(options ...SandboxOption) (*Sandbox, error)
```

```go
sandbox, err := file.NewSandbox(
    file.WithReadRoots("./inputs", "./templates"),
    file.WithWriteRoot("./workspace"),
    file.WithMaxReadBytes(10 << 20),
    file.WithMaxWriteBytes(5 << 20),
)
```

Sandbox는 fail-closed입니다. read root가 없으면 read 작업을, write root가 없으면
write 작업을 거부합니다. Sandboxed path는 root 상대이며 절대 경로, traversal,
루트 밖으로 나가는 symbolic link는 거부됩니다.

**함수:**
- `read_file(path)`, `list_files(path)`, `file_exists(path)`, `read_pptx(path)`:
  read root 아래에서만 실행됩니다.
- `write_file(path, content)`, `delete_file(path)`: write root 아래에서만 실행됩니다.
- `filegen.NewWithSandbox(sandbox)`는 `create_file`과 `create_directory`에 같은
  write capability를 사용합니다. 기존 파일 대체에는 `overwrite: true`와
  `WithAllowOverwrite(true)`가 모두 필요합니다.

자세한 내용은 영어 [Sandboxed File I/O guide](/guide/sandboxed-file-io)를 참조하세요.

## 커스텀 도구

커스텀 도구를 생성합니다:

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
    // 입력 처리
    return result, nil
}
```
