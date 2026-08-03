# Tools - Agent機能を拡張

ツールでAgentに外部機能へのアクセスを提供します。

---

## ツールとは？

ツールは、Agentが外部システムとやり取りしたり、計算を実行したり、データを取得したりするために呼び出すことができる関数です。Agno-Goは、Agent機能を拡張するための柔軟なツールキットシステムを提供しています。

### 組み込みツール

- **Calculator**: 基本的な数学演算
- **HTTP**: Webリクエストを実行
- **File**: 安全制御付きのファイル読み書き
- **Google Sheets** ⭐ 新機能 (v1.2.1): Google Sheets データの読み書き
- **Claude Agent Skills** ⭐ 新機能 (v1.2.6): `invoke_claude_skill` で Anthropic Agent Skills を呼び出し
- **Tavily** ⭐ 新機能 (v1.2.6): クイックアンサーとリーダーモード抽出
- **Gmail** ⭐ 新機能 (v1.2.6): Gmail API で既読処理やアーカイブを実行
- **Jira Worklog** ⭐ 新機能 (v1.2.6): Jira Cloud の作業ログを要約・エクスポート
- **ElevenLabs Voice** ⭐ 新機能 (v1.2.6): 音声クリップをオンデマンドで生成

---

## ツールの使用

### 基本的な例

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
    fmt.Println(output.Content) // Agentは自動的に計算機を使用
}
```

---

## Calculator Tool

数学演算を実行します。

### 演算

- `add(a, b)` - 加算
- `subtract(a, b)` - 減算
- `multiply(a, b)` - 乗算
- `divide(a, b)` - 除算

### 例

```go
import "github.com/rexleimo/agno-go/pkg/hno/tools/calculator"

agent, _ := agent.New(agent.Config{
    Model:    model,
    Toolkits: []toolkit.Toolkit{calculator.New()},
})

// Agentは自動的に計算機を使用
output, _ := agent.Run(ctx, "Calculate 15% tip on $85")
```

---

## HTTP Tool

外部APIへのHTTPリクエストを実行します。

### メソッド

- `get(url)` - HTTP GETリクエスト
- `post(url, body)` - HTTP POSTリクエスト

### 例

```go
import "github.com/rexleimo/agno-go/pkg/hno/tools/http"

agent, _ := agent.New(agent.Config{
    Model:    model,
    Toolkits: []toolkit.Toolkit{http.New()},
})

// AgentはAPIからデータを取得可能
output, _ := agent.Run(ctx, "Get the latest GitHub status from https://www.githubstatus.com/api/v2/status.json")
```

### 設定

セキュリティのために許可ドメインを制御:

```go
httpTool := http.New(http.Config{
    AllowedDomains: []string{"api.github.com", "api.weather.com"},
    Timeout:        10 * time.Second,
})
```

---

## File Tool

組み込みの安全制御でファイルを読み書きします。

### 操作

- `read_file(path)` - ファイル内容を読み取り
- `write_file(path, content)` - ファイルに内容を書き込み
- `list_directory(path)` - ディレクトリ内容をリスト
- `delete_file(path)` - ファイルを削除

### 例

```go
import "github.com/rexleimo/agno-go/pkg/hno/tools/file"

fileTool := file.New(file.Config{
    AllowedPaths: []string{"/tmp", "./data"},  // アクセスを制限
    MaxFileSize:  1024 * 1024,                 // 1MB制限
})

agent, _ := agent.New(agent.Config{
    Model:    model,
    Toolkits: []toolkit.Toolkit{fileTool},
})

output, _ := agent.Run(ctx, "Read the contents of ./data/report.txt")
```

### 安全機能

- パス制限（ホワイトリスト）
- ファイルサイズ制限
- 読み取り専用モードオプション
- 自動パスサニタイゼーション

---

## 複数のツール

Agentは複数のツールを使用できます:

```go
agent, _ := agent.New(agent.Config{
    Name:  "Multi-Tool Agent",
    Model: model,
    Toolkits: []toolkit.Toolkit{
        calculator.New(),
        http.New(),
        file.New(file.Config{
            AllowedPaths: []string{"./data"},
        }),
    },
})

// Agentは計算、データ取得、ファイル読み取りが可能
output, _ := agent.Run(ctx,
    "Fetch weather data, calculate average temperature, and save to file")
```

---

## カスタムツールの作成

Toolkitインターフェースを実装して独自のツールを構築します。

### ステップ1: Toolkit構造体を作成

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

    // 関数を登録
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

### ステップ2: ハンドラーを実装

```go
func (t *MyToolkit) greet(args map[string]interface{}) (interface{}, error) {
    name, ok := args["name"].(string)
    if !ok {
        return nil, fmt.Errorf("name must be a string")
    }

    return fmt.Sprintf("Hello, %s!", name), nil
}
```

### ステップ3: ツールを使用

```go
agent, _ := agent.New(agent.Config{
    Model:    model,
    Toolkits: []toolkit.Toolkit{mytool.New()},
})

output, _ := agent.Run(ctx, "Greet Alice")
// Agentはgreet("Alice")を呼び出し、"Hello, Alice!"で応答
```

---

## 高度なカスタムツールの例

データベースクエリツール:

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

## ツールのベストプラクティス

### 1. 明確な説明

Agentがいつツールを使用するかを理解できるように:

```go
// 良い例 ✅
Description: "Calculate the square root of a number. Use when user asks for square roots."

// 悪い例 ❌
Description: "Math function"
```

### 2. 入力を検証

常にツールパラメータを検証:

```go
func (t *MyToolkit) divide(args map[string]interface{}) (interface{}, error) {
    b, ok := args["divisor"].(float64)
    if !ok || b == 0 {
        return nil, fmt.Errorf("divisor must be a non-zero number")
    }
    // ... 除算を実行
}
```

### 3. エラー処理

意味のあるエラーを返す:

```go
func (t *MyToolkit) fetchData(args map[string]interface{}) (interface{}, error) {
    resp, err := http.Get(url)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch data: %w", err)
    }
    // ... 応答を処理
}
```

### 4. セキュリティ

ツール機能を制限:

```go
// 許可された操作をホワイトリスト化
fileTool := file.New(file.Config{
    AllowedPaths: []string{"/safe/path"},
    ReadOnly:     true,  // 書き込みを防止
})

// ドメインを検証
httpTool := http.New(http.Config{
    AllowedDomains: []string{"api.trusted.com"},
})
```

---

## ツール実行フロー

1. ユーザーがAgentにリクエストを送信
2. Agent（LLM）がツールが必要かどうかを判断
3. LLMがパラメータ付きのツール呼び出しを生成
4. Agno-Goがツール関数を実行
5. ツール結果がLLMに返される
6. LLMが最終応答を生成

### 例のフロー

```
ユーザー: "25 * 17は何ですか？"
  ↓
LLM: "計算機を使う必要があります"
  ↓
ツール呼び出し: multiply(25, 17)
  ↓
ツール結果: 425
  ↓
LLM: "答えは425です"
  ↓
ユーザーが受け取る: "答えは425です"
```

---

## トラブルシューティング

### Agentがツールを使用しない

明確な指示を確認:

```go
agent, _ := agent.New(agent.Config{
    Model:        model,
    Toolkits:     []toolkit.Toolkit{calculator.New()},
    Instructions: "Use the calculator tool for any math operations.",
})
```

### ツールエラー

ツールの登録とパラメータの型を確認:

```go
// ツールは数値にfloat64を期待
args := map[string]interface{}{
    "value": 42.0,  // ✅ 正しい
    // "value": "42"  // ❌ 誤った型
}
```

---

## Google Sheets ツール ⭐ 新機能

サービスアカウント認証を使用してGoogle Sheetsデータを読み書きします。

### 操作

- `read_range(spreadsheet_id, range)` - 指定された範囲からデータを読み取り
- `write_range(spreadsheet_id, range, values)` - 指定された範囲にデータを書き込み
- `append_rows(spreadsheet_id, range, values)` - スプレッドシートに行を追加

### 設定

1. **サービスアカウントを作成**:
   - Google Cloud Consoleにアクセス
   - サービスアカウントを作成
   - JSON認証情報ファイルをダウンロード

2. **スプレッドシートを共有**:
   - Google Sheetをサービスアカウントのメールアドレスと共有
   - "編集者"権限を付与

### 使用方法

```go
import "github.com/rexleimo/agno-go/pkg/hno/tools/googlesheets"

// JSONファイルから認証情報を読み込み
sheetsTool, err := googlesheets.New(googlesheets.Config{
    CredentialsFile: "./service-account.json",
})

// またはJSON文字列を使用
sheetsTool, err := googlesheets.New(googlesheets.Config{
    CredentialsJSON: os.Getenv("GOOGLE_SHEETS_CREDENTIALS"),
})

agent, _ := agent.New(agent.Config{
    Model:    model,
    Toolkits: []toolkit.Toolkit{sheetsTool},
})
```

### 例

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
        Name:     "データアナリスト",
        Model:    model,
        Toolkits: []toolkit.Toolkit{sheetsTool},
    })

    // Agentはスプレッドシートデータを読み取り分析可能
    output, _ := agent.Run(context.Background(),
        "Sheet1!A1:D100から売上データを読み取り、総収入を要約してください")

    fmt.Println(output.Content)
}
```

### 範囲表記

範囲には標準のA1表記を使用：
- `Sheet1!A1:B10` - Sheet1のA1からB10セル
- `Sheet2!A:A` - Sheet2のA列全体
- `Sheet1!1:5` - Sheet1の1から5行

### セキュリティ

- サービスアカウント認証（ユーザー操作不要）
- スプレッドシートレベルの権限
- 共有設定による読み書きアクセスの制御

## 次のステップ

- 会話状態については[Memory](/guide/memory)を参照
- 専門的なツールAgentで[Teams](/guide/team)を構築
- ツールオーケストレーションのために[Workflow](/guide/workflow)を探索
- 詳細なドキュメントは[APIリファレンス](/api/tools)を確認

---

## 関連例

- [Simple Agent](/examples/simple-agent) - Calculator toolの使用
- [Search Agent](/examples/search-agent) - Web検索用のHTTP tool
- [File Agent](/examples/file-agent) - ファイル操作
- [Google Sheets Agent](/examples/googlesheets-agent) - Google Sheets統合
