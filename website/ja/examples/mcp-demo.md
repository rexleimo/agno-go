# MCP デモ例

## 概要

この例では、MCP (Model Context Protocol) サーバーに接続し、HNO MCP クライアントを通じてそのツールを使用する方法を説明します。セキュリティ検証の設定、トランスポートの作成、MCP サーバーへの接続、MCP ツールと HNO エージェントの統合方法を学習します。

## 学習内容

- MCP コマンドのセキュリティ検証の作成と設定方法
- サブプロセス通信用の stdio トランスポートのセットアップ方法
- MCP サーバーへの接続と利用可能なツールの発見方法
- HNO エージェントで使用する MCP ツールキットの作成方法
- MCP ツールを直接呼び出す方法

## 前提条件

- Go 1.21 以降
- インストール済みの MCP サーバー (例: calculator サーバー)

## セットアップ

### 1. MCP サーバーのインストール

```bash
# uvx パッケージマネージャーをインストール
pip install uvx

# calculator MCP サーバーをインストール
uvx mcp install @modelcontextprotocol/server-calculator

# インストールを確認
python -m mcp_server_calculator --help
```

### 2. 例を実行

```bash
# 例のディレクトリに移動
cd cmd/examples/mcp_demo

# 直接実行
go run main.go

# またはビルドして実行
go build -o mcp_demo
./mcp_demo
```

## 完全なコード

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/mcp/client"
	"github.com/rexleimo/agno-go/pkg/hno/mcp/security"
	mcptoolkit "github.com/rexleimo/agno-go/pkg/hno/mcp/toolkit"
)

func main() {
	fmt.Println("=== HNO MCP Demo ===")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// ステップ 1: セキュリティ検証ツールを作成
	fmt.Println("Step 1: Creating security validator...")
	validator := security.NewCommandValidator()

	command := "python"
	args := []string{"-m", "mcp_server_calculator"}

	if err := validator.Validate(command, args); err != nil {
		log.Fatalf("Command validation failed: %v", err)
	}
	fmt.Printf("✓ Command validated: %s %v\n", command, args)

	// ステップ 2: トランスポートを作成
	fmt.Println("Step 2: Creating transport...")
	transport, err := client.NewStdioTransport(client.StdioConfig{
		Command: command,
		Args:    args,
	})
	if err != nil {
		log.Fatalf("Failed to create transport: %v", err)
	}
	fmt.Println("✓ Stdio transport created")

	// ステップ 3: MCP クライアントを作成
	fmt.Println("Step 3: Creating MCP client...")
	mcpClient, err := client.New(transport, client.Config{
		ClientName:    "agno-go-demo",
		ClientVersion: "0.1.0",
	})
	if err != nil {
		log.Fatalf("Failed to create MCP client: %v", err)
	}
	fmt.Println("✓ MCP client created")

	// ステップ 4: サーバーに接続
	fmt.Println("Step 4: Connecting to MCP server...")
	if err := mcpClient.Connect(ctx); err != nil {
		log.Fatalf("Connection failed: %v", err)
	}
	defer mcpClient.Disconnect()

	fmt.Println("✓ Connected to MCP server")
	if serverInfo := mcpClient.GetServerInfo(); serverInfo != nil {
		fmt.Printf("  Server: %s v%s\n", serverInfo.Name, serverInfo.Version)
	}

	// ステップ 5: ツールを発見
	fmt.Println("Step 5: Discovering tools...")
	tools, err := mcpClient.ListTools(ctx)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	fmt.Printf("✓ Found %d tools:\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}

	// ステップ 6: MCP ツールキットを作成
	fmt.Println("Step 6: Creating MCP toolkit...")
	toolkit, err := mcptoolkit.New(ctx, mcptoolkit.Config{
		Client: mcpClient,
		Name:   "calculator-tools",
	})
	if err != nil {
		log.Fatalf("Failed to create toolkit: %v", err)
	}
	defer toolkit.Close()

	fmt.Println("✓ MCP toolkit created")
	fmt.Printf("  Toolkit name: %s\n", toolkit.Name())
	fmt.Printf("  Available functions: %d\n", len(toolkit.Functions()))

	// ステップ 7: ツールを直接呼び出す
	fmt.Println("Step 7: Calling a tool...")
	result, err := mcpClient.CallTool(ctx, "add", map[string]interface{}{
		"a": 5,
		"b": 3,
	})
	if err != nil {
		log.Fatalf("Failed to call tool: %v", err)
	}

	fmt.Println("✓ Tool call successful")
	fmt.Printf("  Result: %v\n", result.Content)

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("The MCP toolkit can now be passed to an HNO Agent!")
}
```

## コードの説明

### 1. セキュリティ検証

```go
validator := security.NewCommandValidator()
if err := validator.Validate(command, args); err != nil {
    log.Fatalf("Command validation failed: %v", err)
}
```

- デフォルトのホワイトリストを使用してセキュリティ検証ツールを作成
- コマンドが安全に実行できるかを検証
- 危険なシェルメタ文字をブロック

### 2. Stdio トランスポート

```go
transport, err := client.NewStdioTransport(client.StdioConfig{
    Command: "python",
    Args:    []string{"-m", "mcp_server_calculator"},
})
```

- stdin/stdout 経由で通信するトランスポートを作成
- MCP サーバーをサブプロセスとして起動
- 双方向の JSON-RPC 2.0 メッセージを処理

### 3. MCP クライアント

```go
mcpClient, err := client.New(transport, client.Config{
    ClientName:    "agno-go-demo",
    ClientVersion: "0.1.0",
})
```

- アプリケーション識別子を使用して MCP クライアントを作成
- 接続のライフサイクルを管理
- ツールの発見と呼び出しのメソッドを提供

### 4. ツールの発見

```go
tools, err := mcpClient.ListTools(ctx)
for _, tool := range tools {
    fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
}
```

- MCP サーバーから利用可能なツールを問い合わせ
- ツールのメタデータ (名前、説明、パラメーター) を返す
- 動的なツール発見に使用

### 5. MCP ツールキットの作成

```go
toolkit, err := mcptoolkit.New(ctx, mcptoolkit.Config{
    Client: mcpClient,
    Name:   "calculator-tools",
})
defer toolkit.Close()
```

- MCP ツールを HNO ツールキット関数に変換
- MCP スキーマから関数シグネチャを自動生成
- `agent.Config.Toolkits` と互換性あり

### 6. 直接ツール呼び出し

```go
result, err := mcpClient.CallTool(ctx, "add", map[string]interface{}{
    "a": 5,
    "b": 3,
})
fmt.Printf("Result: %v\n", result.Content)
```

- エージェントなしで MCP ツールを直接呼び出し
- パラメーターをマップとして渡す
- 結果の内容を返す

## 予想される出力

```
=== HNO MCP Demo ===

Step 1: Creating security validator...
✓ Command validated: python [-m mcp_server_calculator]

Step 2: Creating transport...
✓ Stdio transport created

Step 3: Creating MCP client...
✓ MCP client created

Step 4: Connecting to MCP server...
✓ Connected to MCP server
  Server: calculator v0.1.0

Step 5: Discovering tools...
✓ Found 4 tools:
  - add: Add two numbers
  - subtract: Subtract two numbers
  - multiply: Multiply two numbers
  - divide: Divide two numbers

Step 6: Creating MCP toolkit...
✓ MCP toolkit created
  Toolkit name: calculator-tools
  Available functions: 4

Step 7: Calling a tool...
✓ Tool call successful
  Result: 8

=== Demo Complete ===
The MCP toolkit can now be passed to an HNO Agent!
```

## HNO エージェントとの使用

MCP ツールキットを取得したら、任意の HNO エージェントで使用できます:

```go
import (
    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
)

// モデルを作成
model, _ := openai.New("gpt-4o-mini", openai.Config{
    APIKey: "your-api-key",
})

// MCP ツールキットを使用してエージェントを作成
ag, _ := agent.New(agent.Config{
    Name:     "MCP Calculator Agent",
    Model:    model,
    Toolkits: []toolkit.Toolkit{toolkit},  // MCP toolkit here!
})

// エージェントを実行
output, _ := ag.Run(context.Background(), "What is 25 * 4 + 15?")
fmt.Println(output.Content)
```

## トラブルシューティング

**エラー: "command not allowed"**
- MCP サーバーコマンドがセキュリティホワイトリストにあることを確認
- `validator.AddAllowedCommand("your-command")` で追加

**エラー: "failed to start process"**
- MCP サーバーがインストールされていることを確認: `python -m mcp_server_calculator --help`
- Python が PATH に含まれていることを確認

**エラー: "connection timeout"**
- MCP サーバーの起動に時間がかかっている可能性
- コンテキストタイムアウトを増やす: `context.WithTimeout(ctx, 60*time.Second)`

**ツール呼び出しがエラーを返す**
- ツールが存在することを確認: `mcpClient.ListTools(ctx)` をチェック
- パラメータータイプがツールスキーマと一致していることを確認

## 次のステップ

- [MCP 統合ガイド](../guide/mcp.md) を読む
- 他の MCP サーバー (filesystem, git, sqlite) への接続を試す
- ユースケースに合わせたカスタム MCP サーバーを構築
- MCP ツールと組み込みの HNO ツールを組み合わせる
