# パフォーマンス

Agno-Goは極限のパフォーマンスを目指して設計されており、Python Agnoと比較してエージェント インスタンス化が16倍高速です。

## エグゼクティブサマリー

✅ **パフォーマンス目標達成**:
- ✅ エージェントインスタンス化: **~180ns** (<1μs 目標)
- ✅ メモリフットプリント: **~1.2KB/agent** (<3KB 目標)
- ✅ 並行性: 競合なしの線形スケーリング

## ベンチマーク結果

### エージェント作成パフォーマンス

| ベンチマーク | 時間/op | メモリ/op | アロケーション/op |
|-----------|---------|-----------|-----------|
| **シンプルエージェント** | 184.5 ns | 1,272 B (1.2 KB) | 8 |
| **ツール付き** | 193.0 ns | 1,288 B (1.3 KB) | 9 |
| **メモリ付き** | 111.9 ns | 312 B (0.3 KB) | 6 |

**主な知見**:
- ⚡ エージェント作成: **<200ナノ秒** (1μs目標の5倍良い!)
- 💾 メモリ使用量: **1.2-1.3KB** (3KB目標の60%良い)
- 🎯 ツール追加のオーバーヘッドはわずか8.5ns
- 🎯 メモリは軽量（わずか312B）

### 実行パフォーマンス

| ベンチマーク | スループット |
|-----------|------------|
| **シンプル実行** | ~6M ops/sec |
| **ツール呼び出し付き** | ~0.5M ops/sec |
| **メモリ操作** | ~1M ops/sec |

**注**: 実際のパフォーマンスはLLM APIレイテンシ（100-1000ms）に制限されます。上記の結果はモックモデルを使用しています。

### 並行パフォーマンス

| ベンチマーク | 時間/op | メモリ/op | スケーリング |
|-----------|---------|-----------|---------|
| **並列作成** | 191.0 ns | 1,272 B | ✅ 線形 |
| **並列実行** | 同様 | 同様 | ✅ 線形 |

**主な知見**:
- ✅ 並行実行とシングルスレッドパフォーマンスが同一
- ✅ ロック競合や競合状態なし
- ✅ 高並行性シナリオに最適

## パフォーマンス比較

### vs Python Agno

| 指標 | Go | Python | 改善 |
|--------|-----|--------|-------------|
| **インスタンス化** | ~180ns | ~3μs | **16倍高速** |
| **メモリ/エージェント** | ~1.2KB | ~6.5KB | **5分の1** |
| **並行性** | ネイティブgoroutines | GIL制限あり | **優位** |

## 実世界シナリオ

### シナリオ1: バッチエージェント作成

1,000エージェントの作成:
- **時間**: 1,000 × 180ns = **0.18ms**
- **メモリ**: 1,000 × 1.2KB = **1.2MB**

### シナリオ2: 高並行性APIサービス

10,000 req/sの処理:
- **リクエストあたり**: 1エージェントインスタンス
- **メモリオーバーヘッド**: 10,000 × 1.2KB = **12MB**
- **レイテンシ**: <1ms (LLM API呼び出しを除く)

### シナリオ3: マルチエージェントワークフロー

100エージェントの協調:
- **総メモリ**: 100 × 1.2KB = **120KB**
- **起動時間**: 100 × 180ns = **18μs**

## 最適化技術

### 1. 低アロケーション数

- エージェントあたりわずか8-9個のヒープアロケーション
- 不要なインターフェース変換なし
- 事前割り当てされたスライス容量

### 2. 効率的なメモリレイアウト

```go
type Agent struct {
    ID           string        // 16B
    Name         string        // 16B
    Model        Model         // 16B (interface)
    Tools        []Toolkit     // 24B (slice header)
    Memory       Memory        // 16B (interface)
    Instructions string        // 16B
    MaxLoops     int           // 8B
    // 合計: ~112B 構造体 + ヒープアロケーション
}
```

### 3. ゼロコピー操作

- 文字列参照（コピーなし）
- インターフェースポインタ（コピーなし）
- スライスビュー（コピーなし）

## ボトルネック分析

### 現在のボトルネック

1. **LLM APIレイテンシ** (100-1000ms)
   - 解決策: ストリーミング、キャッシング、バッチリクエスト

2. **ツール実行時間** (可変)
   - 解決策: 並列実行、タイムアウト制御

3. **未ベンチマーク**:
   - チーム協調オーバーヘッド
   - ワークフロー実行オーバーヘッド
   - ベクトルDBクエリ

## プロダクション推奨事項

### 1. エージェントプーリング

GC圧力を削減するためにエージェントインスタンスを再利用:

```go
type AgentPool struct {
    agents chan *Agent
}

func NewAgentPool(size int, config agent.Config) *AgentPool {
    pool := &AgentPool{
        agents: make(chan *Agent, size),
    }
    for i := 0; i < size; i++ {
        ag, _ := agent.New(config)
        pool.agents <- ag
    }
    return pool
}

func (p *AgentPool) Get() *Agent {
    return <-p.agents
}

func (p *AgentPool) Put(ag *Agent) {
    ag.ClearMemory()
    p.agents <- ag
}
```

### 2. Goroutine制限

リソース枯渇を避けるために並行性を制限:

```go
semaphore := make(chan struct{}, 100) // 最大100並行

for _, task := range tasks {
    semaphore <- struct{}{}
    go func(t Task) {
        defer func() { <-semaphore }()

        ag, _ := agent.New(config)
        ag.Run(ctx, t.Input)
    }(task)
}
```

### 3. レスポンスキャッシング

API呼び出しを削減するためにLLMレスポンスをキャッシュ:

```go
type CachedModel struct {
    model models.Model
    cache map[string]*types.ModelResponse
    mu    sync.RWMutex
}

func (m *CachedModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
    key := hashRequest(req)

    m.mu.RLock()
    if cached, ok := m.cache[key]; ok {
        m.mu.RUnlock()
        return cached, nil
    }
    m.mu.RUnlock()

    resp, err := m.model.Invoke(ctx, req)
    if err != nil {
        return nil, err
    }

    m.mu.Lock()
    m.cache[key] = resp
    m.mu.Unlock()

    return resp, nil
}
```

### 4. モニタリング

プロダクションで主要指標を監視:

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    agentCreations = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "agno_agent_creations_total",
    })

    agentLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name: "agno_agent_run_duration_seconds",
    })
)
```

## ベンチマークの実行

### 全ベンチマーク実行

```bash
make bench
# または
go test -bench=. -benchmem ./...
```

### 特定ベンチマーク実行

```bash
go test -bench=BenchmarkAgentCreation -benchmem ./pkg/hno/agent/
```

### CPUプロファイル生成

```bash
go test -bench=. -cpuprofile=cpu.prof ./pkg/hno/agent/
go tool pprof cpu.prof
```

### メモリプロファイル生成

```bash
go test -bench=. -memprofile=mem.prof ./pkg/hno/agent/
go tool pprof mem.prof
```

## プロファイリングのヒント

### 1. CPUプロファイリング

```bash
go test -cpuprofile=cpu.prof -bench=.
go tool pprof -http=:8080 cpu.prof
```

### 2. メモリプロファイリング

```bash
go test -memprofile=mem.prof -bench=.
go tool pprof -http=:8080 mem.prof
```

### 3. 競合検出

```bash
go test -race ./...
```

## 将来の最適化

### 計画中の改善

- [ ] 繰り返し値の文字列インターン化
- [ ] エージェント再利用のためのsync.Pool
- [ ] バッチツール実行
- [ ] LLM APIのためのHTTP/2コネクションプーリング
- [ ] 低レイテンシのためのgRPCサポート

## 結論

Agno-Goは**パフォーマンス目標を超えています**:

- ✅ 目標の5倍高速（180ns vs 1μs）
- ✅ 目標の60%少ないメモリ（1.2KB vs 3KB）
- ✅ Pythonの16倍高速、メモリは5分の1
- ✅ 完璧な並行性スケーリング

**サポート**:
- 数千の並行エージェント
- 10K+リクエスト/秒
- 低レイテンシリアルタイムアプリケーション

## 参照

- [アーキテクチャ](/advanced/architecture)
- [デプロイメント](/advanced/deployment)
- [ベンチマークコード](https://github.com/rexleimo/agno-Go/tree/main/pkg/hno/agent/agent_bench_test.go)
