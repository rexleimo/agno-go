# アーキテクチャ

HNOは、シンプルさ、効率性、拡張性を重視した、クリーンでモジュール化されたアーキテクチャに従っています。

## コア哲学

**シンプル、効率的、スケーラブル**

## 全体アーキテクチャ

```
┌─────────────────────────────────────────┐
│          Application Layer              │
│  (CLI Tools, Web API, Custom Apps)      │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│         Core Abstractions               │
│  ┌─────────┐  ┌──────┐  ┌──────────┐   │
│  │  Agent  │  │ Team │  │ Workflow │   │
│  └─────────┘  └──────┘  └──────────┘   │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│        Foundation Layer                  │
│  ┌────────┐ ┌───────┐ ┌──────┐         │
│  │ Models │ │ Tools │ │Memory│ ...     │
│  └────────┘ └───────┘ └──────┘         │
└─────────────────────────────────────────┘
```

## コアインターフェース

### 1. Model インターフェース

```go
type Model interface {
    // 同期呼び出し
    Invoke(ctx context.Context, req *InvokeRequest) (*ModelResponse, error)

    // ストリーミング呼び出し
    InvokeStream(ctx context.Context, req *InvokeRequest) (<-chan ResponseChunk, error)

    // メタデータ
    GetProvider() string
    GetID() string
}
```

### 2. Toolkit インターフェース

```go
type Toolkit interface {
    Name() string
    Functions() map[string]*Function
}

type Function struct {
    Name        string
    Description string
    Parameters  map[string]Parameter
    Handler     func(context.Context, map[string]interface{}) (interface{}, error)
}
```

### 3. Memory インターフェース

```go
type Memory interface {
    Add(message types.Message) error
    GetMessages() []types.Message
    Clear() error
}
```

## コンポーネント詳細

### Agent

**ファイル**: `pkg/hno/agent/agent.go`

以下の機能を持つ自律的なAIエンティティ:
- 推論にLLMを使用
- ツールを呼び出せる
- 会話記憶を維持
- フックで入出力を検証

**主要メソッド**:
```go
New(config Config) (*Agent, error)
Run(ctx context.Context, input string) (*RunOutput, error)
ClearMemory()
```

### Team

**ファイル**: `pkg/hno/team/team.go`

4つの協調モードを持つマルチエージェント協調:

1. **Sequential** - エージェントが順番に作業
2. **Parallel** - 全エージェントが同時に作業
3. **LeaderFollower** - リーダーがフォロワーに委任
4. **Consensus** - エージェントが合意まで議論

### Workflow

**ファイル**: `pkg/hno/workflow/workflow.go`

5つのプリミティブを持つステップベースのオーケストレーション:

1. **Step** - エージェントまたは関数を実行
2. **Condition** - コンテキストに基づいて分岐
3. **Loop** - 終了条件で反復
4. **Parallel** - ステップを並行実行
5. **Router** - 動的ルーティング

### Models

**ディレクトリ**: `pkg/hno/models/`

LLMプロバイダーの実装:
- `openai/` - OpenAI GPTモデル
- `anthropic/` - Anthropic Claudeモデル
- `ollama/` - Ollamaローカルモデル
- `deepseek/`, `gemini/`, `modelscope/` - その他のプロバイダー

### Tools

**ディレクトリ**: `pkg/hno/tools/`

拡張可能なツールキットシステム:
- `calculator/` - 数学演算
- `http/` - HTTPリクエスト
- `file/` - ファイル操作
- `search/` - Web検索

## AgentOS プロダクションサーバー

**ディレクトリ**: `pkg/agentos/`

以下の機能を持つプロダクション対応HTTPサーバー:

- RESTful APIエンドポイント
- セッション管理
- エージェントレジストリ
- ヘルスモニタリング
- CORSサポート
- リクエストタイムアウト処理

**アーキテクチャ**:
```
┌─────────────────────┐
│   HTTP Handlers     │
│  (API Endpoints)    │
└──────────┬──────────┘
           │
┌──────────▼──────────┐
│  Agent Registry     │
│  (Thread-safe map)  │
└──────────┬──────────┘
           │
┌──────────▼──────────┐
│ Session Manager     │
│  (In-memory store)  │
└──────────┬──────────┘
           │
┌──────────▼──────────┐
│  Agent Instances    │
│  (Runtime agents)   │
└─────────────────────┘
```

## 設計パターン

### 1. インターフェースベース設計

すべてのコアコンポーネントは柔軟性のためにインターフェースを使用:

```go
type Model interface { /* ... */ }
type Toolkit interface { /* ... */ }
type Memory interface { /* ... */ }
```

### 2. 継承よりコンポジション

エージェントはモデル、ツール、メモリを組み合わせて構成:

```go
type Agent struct {
    Model    Model
    Toolkits []Toolkit
    Memory   Memory
    // ...
}
```

### 3. コンテキスト伝播

すべての操作はキャンセルとタイムアウトのために`context.Context`を受け入れる:

```go
func (a *Agent) Run(ctx context.Context, input string) (*RunOutput, error)
```

### 4. エラーラッピング

ラップされたエラーによる一貫したエラー処理:

```go
if err != nil {
    return nil, fmt.Errorf("failed to run agent: %w", err)
}
```

## パフォーマンス最適化

### 1. 低アロケーション数

- 最小限のヒープアロケーション（エージェントあたり8-9個）
- 事前割り当てされたスライス
- 適切な場所での文字列インターン化

### 2. 効率的なメモリレイアウト

```go
type Agent struct {
    ID           string   // 16B
    Name         string   // 16B
    Model        Model    // 16B (interface)
    // 合計: ~112B 構造体 + ヒープアロケーション
}
```

### 3. Goroutineセーフティ

- グローバルステートなし
- 設計段階からスレッドセーフ
- 可能な限りロックフリー

## 並行性モデル

### Agentの並行性

```go
// 複数のエージェントを同時に作成しても安全
for i := 0; i < 100; i++ {
    go func() {
        ag, _ := agent.New(config)
        output, _ := ag.Run(ctx, input)
    }()
}
```

### Team並列モード

```go
// エージェントは並列ゴルーチンで実行
team := team.New(team.Config{
    Mode: team.ModeParallel,
    Agents: agents,
})
```

### Workflow並列ステップ

```go
// ステップは同時に実行
workflow.NewParallel("tasks", []Primitive{
    step1, step2, step3,
})
```

## 拡張ポイント

### 1. カスタムモデル

`Model`インターフェースを実装:

```go
type MyModel struct{}

func (m *MyModel) Invoke(ctx context.Context, req *InvokeRequest) (*ModelResponse, error) {
    // カスタム実装
}
```

### 2. カスタムツール

`BaseToolkit`を拡張:

```go
type MyToolkit struct {
    *toolkit.BaseToolkit
}

func (t *MyToolkit) RegisterFunctions() {
    t.RegisterFunction(&Function{
        Name: "my_function",
        Handler: t.myHandler,
    })
}
```

### 3. カスタムメモリ

`Memory`インターフェースを実装:

```go
type MyMemory struct{}

func (m *MyMemory) Add(msg types.Message) error {
    // カスタムストレージ
}
```

## テスト戦略

### ユニットテスト

- 各パッケージに`*_test.go`ファイル
- インターフェースのモック実装
- テーブル駆動テスト

### 統合テスト

- エンドツーエンドワークフローテスト
- マルチエージェントシナリオ
- 実際のAPI統合テスト

### ベンチマークテスト

- `*_bench_test.go`のパフォーマンスベンチマーク
- メモリアロケーション追跡
- 並行性ストレステスト

## 依存関係

### コア依存関係

- **Go標準ライブラリ** - ほとんどの機能
- **重いフレームワークなし** - 軽量設計

### オプション依存関係

- LLMプロバイダーSDK（OpenAI、Anthropicなど）
- ベクトルデータベースクライアント（ChromaDB）
- HTTPクライアントライブラリ

## 将来のアーキテクチャ

### 計画中の機能強化

1. **ストリーミングサポート** - リアルタイムレスポンスストリーミング
2. **プラグインシステム** - 動的ツールロード
3. **分散エージェント** - マルチノードデプロイメント
4. **高度なメモリ** - 永続ストレージ、ベクトルメモリ

## ベストプラクティス

### 1. インターフェースを使用

```go
var model models.Model = openai.New(...)
```

### 2. エラー処理

```go
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

### 3. コンテキストを使用

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### 4. シンプルに保つ

KISS原則に従う - 過度な設計を避ける。

## 参照

- [パフォーマンスベンチマーク](/advanced/performance)
- [デプロイメントガイド](/advanced/deployment)
- [APIリファレンス](/api/)
- [ソースコード](https://github.com/rexleimo/agno-go)
