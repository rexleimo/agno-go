# クイックスタート

5分以内にHNOを始めましょう！

## 前提条件

- Go 1.21以降
- OpenAI APIキー（またはAnthropic/Ollama）
- AIエージェントの基本的な理解

## インストール

### オプション1: Go Getを使用

```bash
go get github.com/rexleimo/HNO
```

### オプション2: リポジトリのクローン

```bash
git clone https://github.com/rexleimo/HNO.git
cd HNO
go mod download
```

## 最初のエージェント

### 1. シンプルなエージェント（ツールなし）

`main.go`ファイルを作成：

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/HNO/pkg/hno/agent"
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
)

func main() {
    // Get API key from environment
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Fatal("OPENAI_API_KEY environment variable is required")
    }

    // Create OpenAI model
    model, err := openai.New("gpt-4o-mini", openai.Config{
        APIKey: apiKey,
    })
    if err != nil {
        log.Fatalf("Failed to create model: %v", err)
    }

    // Create agent
    ag, err := agent.New(agent.Config{
        Name:         "Assistant",
        Model:        model,
        Instructions: "You are a helpful assistant.",
    })
    if err != nil {
        log.Fatalf("Failed to create agent: %v", err)
    }

    // Run agent
    output, err := ag.Run(context.Background(), "What is the capital of France?")
    if err != nil {
        log.Fatalf("Agent run failed: %v", err)
    }

    fmt.Println("Agent:", output.Content)
}
```

**実行方法:**

```bash
export OPENAI_API_KEY=sk-your-key-here
go run main.go
```

**期待される出力:**

```
Agent: The capital of France is Paris.
```

### 2. ツールを持つエージェント

計算機ツールを追加：

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/HNO/pkg/hno/agent"
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
    "github.com/rexleimo/HNO/pkg/hno/tools/calculator"
    "github.com/rexleimo/HNO/pkg/hno/tools/toolkit"
)

func main() {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Fatal("OPENAI_API_KEY required")
    }

    // Create model
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: apiKey,
    })

    // Create agent WITH tools
    ag, _ := agent.New(agent.Config{
        Name:  "Calculator Agent",
        Model: model,
        Toolkits: []toolkit.Toolkit{
            calculator.New(),
        },
        Instructions: "You are a math assistant. Use the calculator tools for calculations.",
    })

    // Ask a math question
    output, _ := ag.Run(context.Background(), "What is 123 * 456 + 789?")

    fmt.Println("Question: What is 123 * 456 + 789?")
    fmt.Println("Agent:", output.Content)
}
```

**実行方法:**

```bash
go run main.go
```

**期待される出力:**

```
Question: What is 123 * 456 + 789?
Agent: The result is 56,877
```

### 3. マルチターン会話

会話のためのメモリを追加：

```go
package main

import (
    "bufio"
    "context"
    "fmt"
    "log"
    "os"
    "strings"

    "github.com/rexleimo/HNO/pkg/hno/agent"
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
)

func main() {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Fatal("OPENAI_API_KEY required")
    }

    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: apiKey,
    })

    ag, _ := agent.New(agent.Config{
        Name:         "Chat Assistant",
        Model:        model,
        Instructions: "You are a friendly chatbot. Remember context from previous messages.",
    })

    fmt.Println("Chat Assistant (type 'quit' to exit)")
    fmt.Println("=====================================")

    scanner := bufio.NewScanner(os.Stdin)
    for {
        fmt.Print("\nYou: ")
        if !scanner.Scan() {
            break
        }

        input := strings.TrimSpace(scanner.Text())
        if input == "quit" || input == "exit" {
            fmt.Println("Goodbye!")
            break
        }

        if input == "" {
            continue
        }

        // Run agent (memory is automatically maintained)
        output, err := ag.Run(context.Background(), input)
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            continue
        }

        fmt.Printf("Agent: %s\n", output.Content)
    }
}
```

**会話例:**

```
You: My name is Alice
Agent: Nice to meet you, Alice! How can I help you today?

You: What's my name?
Agent: Your name is Alice!

You: quit
Goodbye!
```

## AgentOS（HTTPサーバー）の使用

### 1. サーバーの起動

#### Docker Composeを使用（推奨）

```bash
# Copy environment template
cp .env.example .env

# Edit .env and add your API key
nano .env  # Add: OPENAI_API_KEY=sk-your-key

# Start server
docker-compose up -d

# Check health
curl http://localhost:8080/health
```

#### Goを使用（ネイティブ）

```bash
# Build server
go build -o agentos cmd/server/main.go

# Run server
export OPENAI_API_KEY=sk-your-key
./agentos
```

### 2. APIの使用

#### ヘルスチェック

```bash
curl http://localhost:8080/health
```

**レスポンス:**
```json
{
  "status": "healthy",
  "service": "agentos",
  "time": 1704067200
}
```

#### エージェントの実行

```bash
curl -X POST http://localhost:8080/api/v1/agents/assistant/run \
  -H "Content-Type: application/json" \
  -d '{
    "input": "What is 2+2?"
  }'
```

**レスポンス:**
```json
{
  "content": "2 + 2 equals 4.",
  "metadata": {
    "agent_id": "assistant"
  }
}
```

完全なAPIドキュメントについては[AgentOS APIリファレンス](/api/agentos)を参照してください。

## 次のステップ

### さらに学ぶ

- [コアコンセプト](/guide/agent) - Agent、Team、Workflowを理解する
- [ツールガイド](/guide/tools) - 組み込みツールとカスタムツールについて学ぶ
- [モデルガイド](/guide/models) - マルチモデルサポート
- [高度なトピック](/advanced/) - アーキテクチャ、パフォーマンス、デプロイメント

### 例を試す

すべての例は`cmd/examples/`ディレクトリにあります：

```bash
# Simple agent with calculator
go run cmd/examples/simple_agent/main.go

# Anthropic Claude
go run cmd/examples/claude_agent/main.go

# Local models with Ollama
go run cmd/examples/ollama_agent/main.go

# Multi-agent team
go run cmd/examples/team_demo/main.go

# Workflow engine
go run cmd/examples/workflow_demo/main.go

# RAG with ChromaDB
go run cmd/examples/rag_demo/main.go
```

各例の詳細なドキュメントについては[Examples](/examples/)を参照してください。

## トラブルシューティング

### よくある問題

**1. "OPENAI_API_KEY not set"**

```bash
export OPENAI_API_KEY=sk-your-key-here
```

**2. "Module not found"**

```bash
go mod download
go mod tidy
```

**3. "Port 8080 already in use"**

`.env`または設定でポートを変更：
```bash
AGENTOS_ADDRESS=:9090
```

**4. "Context deadline exceeded"**

タイムアウトを増やす：
```bash
export REQUEST_TIMEOUT=60
```

### デバッグログの取得

```bash
export AGENTOS_DEBUG=true
export LOG_LEVEL=debug
```

## クイックリファレンス

### よく使うインポート

```go
import (
    "github.com/rexleimo/HNO/pkg/hno/agent"
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
    "github.com/rexleimo/HNO/pkg/hno/tools/calculator"
    "github.com/rexleimo/HNO/pkg/hno/team"
    "github.com/rexleimo/HNO/pkg/hno/workflow"
    "github.com/rexleimo/HNO/pkg/agentos"
)
```

### エージェント作成テンプレート

```go
model, err := openai.New("gpt-4o-mini", openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
})

ag, err := agent.New(agent.Config{
    Name:         "Agent Name",
    Model:        model,
    Toolkits:     []toolkit.Toolkit{/* tools */},
    Instructions: "System instructions",
    MaxLoops:     10,
})

output, err := ag.Run(context.Background(), "input")
```

## 次: コアコンセプト

3つのコア抽象について学ぶ：

- [Agent](/guide/agent) - 自律的AIエージェント
- [Team](/guide/team) - マルチエージェント協力
- [Workflow](/guide/workflow) - ステップベースのオーケストレーション
