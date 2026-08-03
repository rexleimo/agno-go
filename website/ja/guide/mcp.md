# MCP 統合

## MCP とは?

**モデルコンテキストプロトコル (Model Context Protocol, MCP)** は、LLM アプリケーションと外部データソースおよびツール間のシームレスな統合を可能にするオープン標準です。Anthropic によって開発された MCP は、標準化されたインターフェイスを通じて AI モデルを様々なサービスに接続するための普遍的なプロトコルを提供します。

**Model Context Protocol (MCP)** is an open standard that enables seamless integration between LLM applications and external data sources and tools.

## HNO で MCP を使用する理由

- **🔌 拡張性** - エージェントを任意の MCP 互換サーバーに接続
  - **Extensibility** - Connect your agents to any MCP-compatible server
- **🔒 セキュリティ** - 組み込みのコマンド検証とシェルインジェクション保護
  - **Security** - Built-in command validation and shell injection protection
- **🚀 パフォーマンス** - 高速初期化 (<100μs) と低メモリフットプリント (<10KB)
  - **Performance** - Fast initialization (<100μs) and low memory footprint (<10KB)
- **📦 再利用性** - 既存の MCP サーバーを活用し、車輪の再発明を避ける
  - **Reusability** - Leverage existing MCP servers

## アーキテクチャ | Architecture

```
pkg/hno/mcp/
├── protocol/       # JSON-RPC 2.0 と MCP メッセージタイプ
├── client/         # MCP クライアントコアとトランスポート
├── security/       # コマンド検証とセキュリティ
├── content/        # コンテンツタイプ処理
└── toolkit/        # agno ツールキットシステムとの統合
```

## クイックスタート

### 前提条件 | Prerequisites

- Go 1.21 以降 | Go 1.21 or later
- MCP サーバー (例: calculator, filesystem, git)

### インストール | Installation

```bash
# MCP サーバー管理用の uvx をインストール
pip install uvx

# サンプル MCP サーバーをインストール
uvx mcp install @modelcontextprotocol/server-calculator
```

### 基本的な使い方 | Basic Usage

```go
import (
    "github.com/rexleimo/agno-go/pkg/hno/mcp/client"
    "github.com/rexleimo/agno-go/pkg/hno/mcp/security"
    mcptoolkit "github.com/rexleimo/agno-go/pkg/hno/mcp/toolkit"
)

// セキュリティバリデーターを作成
// Create security validator
validator := security.NewCommandValidator()

// トランスポートを設定
// Setup transport
transport, err := client.NewStdioTransport(client.StdioConfig{
    Command: "python",
    Args:    []string{"-m", "mcp_server_calculator"},
})

// MCP クライアントを作成
// Create MCP client
mcpClient, err := client.New(transport, client.Config{
    ClientName:    "my-agent",
    ClientVersion: "1.0.0",
})

ctx := context.Background()
mcpClient.Connect(ctx)
defer mcpClient.Disconnect()

// エージェント用の MCP ツールキットを作成
// Create MCP toolkit for agents
toolkit, err := mcptoolkit.New(ctx, mcptoolkit.Config{
    Client: mcpClient,
    Name:   "calculator-tools",
})
defer toolkit.Close()
```

## セキュリティ機能 | Security Features

### コマンドホワイトリスト | Command Whitelist

デフォルトで許可されているコマンド:
- `python`, `python3`
- `node`, `npm`, `npx`
- `uvx`
- `docker`

### シェルインジェクション保護 | Shell Injection Protection

ブロックされる文字 | Blocked characters:
- `;` (コマンド区切り文字)
- `|` (パイプ)
- `&` (バックグラウンド実行)
- `` ` `` (コマンド置換)
- `$` (変数展開)
- `>`, `<` (リダイレクト)

## ツールフィルタリング | Tool Filtering

```go
// 特定のツールのみを含める
// Include only specific tools
toolkit, err := mcptoolkit.New(ctx, mcptoolkit.Config{
    Client:       mcpClient,
    IncludeTools: []string{"add", "subtract", "multiply"},
})

// 特定のツールを除外
// Exclude certain tools
toolkit, err := mcptoolkit.New(ctx, mcptoolkit.Config{
    Client:       mcpClient,
    ExcludeTools: []string{"divide"},
})
```

## 既知の MCP サーバー | Known MCP Servers

| サーバー | 説明 | インストール |
|--------|-----|------------|
| **server-calculator** | 数学演算 | `uvx mcp install @modelcontextprotocol/server-calculator` |
| **server-filesystem** | ファイル操作 | `uvx mcp install @modelcontextprotocol/server-filesystem` |
| **server-git** | Git 操作 | `uvx mcp install @modelcontextprotocol/server-git` |
| **server-sqlite** | SQLite データベース | `uvx mcp install @modelcontextprotocol/server-sqlite` |

## パフォーマンス | Performance

- **MCP クライアント初期化**: <100μs
- **ツール検出**: サーバーあたり <50μs
- **メモリ**: 接続あたり <10KB
- **テストカバレッジ**: >80%

## ベストプラクティス | Best Practices

1. **常にセキュリティ検証を使用** - コマンド検証をバイパスしない
2. **適切にツールをフィルタリング** - エージェントに必要なツールのみを公開
3. **エラーを適切に処理** - MCP サーバーは失敗またはタイムアウトする可能性がある
4. **接続を閉じる** - リソースをクリーンアップするため常に `defer toolkit.Close()`
5. **モックサーバーでテスト** - `pkg/hno/mcp/client/testing.go` のテストユーティリティを使用

## 次のステップ | Next Steps

- [MCP デモ](../examples/mcp-demo.md)を試す
- [MCP 実装ガイド](../../pkg/hno/mcp/IMPLEMENTATION.md)を読む
- [MCP プロトコル仕様](https://spec.modelcontextprotocol.io/)を探索
- [GitHub](https://github.com/rexleimo/HNO/discussions)でディスカッションに参加

## トラブルシューティング | Troubleshooting

**エラー: "command not allowed"**
- コマンドがホワイトリストにあるか確認
- `validator.AddAllowedCommand()` を使用してカスタムコマンドを追加

**エラー: "shell metacharacters detected"**
- コマンド引数に危険な文字が含まれている
- 引数に `;`, `|`, `&` などが含まれていないことを確認

**エラー: "failed to start MCP server"**
- MCP サーバーがインストールされているか確認
- コマンドパスが正しいか確認
- 必要な権限があるか確認
