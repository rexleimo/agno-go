# Tools APIリファレンス

## Calculator

基本的な数学演算を提供します。

**作成:**
```go
func New() *Calculator
```

**関数:**
- `add(a, b)`: 加算
- `subtract(a, b)`: 減算
- `multiply(a, b)`: 乗算
- `divide(a, b)`: 除算

**例:**
```go
calc := calculator.New()

ag, _ := agent.New(agent.Config{
    Toolkits: []toolkit.Toolkit{calc},
    // ...
})
```

## HTTP

GET/POSTリクエスト用のHTTPクライアント。

**作成:**
```go
func New() *HTTPToolkit
```

**関数:**
- `http_get(url)`: HTTP GETリクエスト
- `http_post(url, body)`: HTTP POSTリクエスト

**例:**
```go
http := http.New()

ag, _ := agent.New(agent.Config{
    Toolkits: []toolkit.Toolkit{http},
    Instructions: "You can make HTTP requests to fetch data.",
})
```

## File

Agent が指定するパスには明示的な capability を持つ `Sandbox` と
`NewWithSandbox` を使用します。`New()` は信頼できる呼び出し元向けの無制限な
互換 API です。

**作成:**
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

Sandbox は fail-closed です。read root がなければ read 操作、write root が
なければ write 操作を拒否します。Sandboxed path は root 相対であり、絶対パス、
traversal、ルート外に出る symbolic link は拒否されます。

**関数:**
- `read_file(path)`, `list_files(path)`, `file_exists(path)`, `read_pptx(path)`:
  read root の下でのみ実行されます。
- `write_file(path, content)`, `delete_file(path)`: write root の下でのみ実行されます。
- `filegen.NewWithSandbox(sandbox)` は `create_file` と `create_directory` に同じ
  write capability を使用します。既存ファイルの置換には `overwrite: true` と
  `WithAllowOverwrite(true)` の両方が必要です。

詳しくは英語の [Sandboxed File I/O guide](/guide/sandboxed-file-io) を参照してください。

## カスタムツール

カスタムツールを作成:

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
    // 入力を処理
    return result, nil
}
```
