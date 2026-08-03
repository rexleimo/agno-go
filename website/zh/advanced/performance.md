# 性能 / Performance

HNO 专为极致性能而设计,实现比 Python Agno 快 16 倍的智能体实例化速度。

## 执行摘要 / Executive Summary

✅ **性能目标已达成 / Performance Goals Achieved**:
- ✅ Agent 实例化: **~180ns** (<1μs 目标)
- ✅ 内存占用: **~1.2KB/agent** (<3KB 目标)
- ✅ 并发性: 线性扩展,无竞争

## 基准测试结果 / Benchmark Results

### Agent 创建性能 / Agent Creation Performance

| 基准测试 / Benchmark | 时间/操作 / Time/op | 内存/操作 / Memory/op | 分配次数/操作 / Allocs/op |
|-----------|---------|-----------|-----------|
| **简单 Agent / Simple Agent** | 184.5 ns | 1,272 B (1.2 KB) | 8 |
| **带工具 / With Tools** | 193.0 ns | 1,288 B (1.3 KB) | 9 |
| **带记忆 / With Memory** | 111.9 ns | 312 B (0.3 KB) | 6 |

**关键发现 / Key Findings**:
- ⚡ Agent 创建: **<200 纳秒** (比 1μs 目标快 5 倍!)
- 💾 内存使用: **1.2-1.3KB** (比 3KB 目标低 60%)
- 🎯 添加工具仅增加 8.5ns 开销
- 🎯 记忆非常轻量(仅 312B)

### 执行性能 / Execution Performance

| 基准测试 / Benchmark | 吞吐量 / Throughput |
|-----------|------------|
| **简单运行 / Simple Run** | ~6M ops/sec |
| **带工具调用 / With Tool Calls** | ~0.5M ops/sec |
| **内存操作 / Memory Operations** | ~1M ops/sec |

**注意**: 实际性能受 LLM API 延迟限制(100-1000ms)。以上结果使用模拟模型。

### 并发性能 / Concurrent Performance

| 基准测试 / Benchmark | 时间/操作 / Time/op | 内存/操作 / Memory/op | 扩展性 / Scaling |
|-----------|---------|-----------|---------|
| **并行创建 / Parallel Creation** | 191.0 ns | 1,272 B | ✅ 线性 |
| **并行运行 / Parallel Run** | 相似 / Similar | 相似 / Similar | ✅ 线性 |

**关键发现 / Key Findings**:
- ✅ 并发和单线程性能相同
- ✅ 无锁竞争或竞态条件
- ✅ 完美适合高并发场景

## 性能对比 / Performance Comparison

### vs Python Agno

| 指标 / Metric | Go | Python | 改进 / Improvement |
|--------|-----|--------|-------------|
| **实例化 / Instantiation** | ~180ns | ~3μs | **快 16 倍** |
| **内存/Agent / Memory/Agent** | ~1.2KB | ~6.5KB | **少 5 倍** |
| **并发性 / Concurrency** | 原生 goroutines | GIL 限制 | **更优** |

## 实际场景 / Real-World Scenarios

### 场景 1: 批量 Agent 创建 / Batch Agent Creation

创建 1,000 个 agents:
- **时间**: 1,000 × 180ns = **0.18ms**
- **内存**: 1,000 × 1.2KB = **1.2MB**

### 场景 2: 高并发 API 服务 / High-Concurrency API Service

处理 10,000 请求/秒:
- **每个请求**: 1 个 agent 实例
- **内存开销**: 10,000 × 1.2KB = **12MB**
- **延迟**: <1ms (不包括 LLM API 调用)

### 场景 3: 多智能体工作流 / Multi-Agent Workflow

100 个 agents 协作:
- **总内存**: 100 × 1.2KB = **120KB**
- **启动时间**: 100 × 180ns = **18μs**

## 优化技术 / Optimization Techniques

### 1. 低分配次数 / Low Allocation Count

- 每个 agent 仅 8-9 次堆分配
- 无不必要的接口转换
- 预分配切片容量

### 2. 高效的内存布局 / Efficient Memory Layout

```go
type Agent struct {
    ID           string        // 16B
    Name         string        // 16B
    Model        Model         // 16B (interface)
    Tools        []Toolkit     // 24B (slice header)
    Memory       Memory        // 16B (interface)
    Instructions string        // 16B
    MaxLoops     int           // 8B
    // Total: ~112B struct + heap allocations
}
```

### 3. 零拷贝操作 / Zero-Copy Operations

- 字符串引用(无拷贝)
- 接口指针(无拷贝)
- 切片视图(无拷贝)

## 瓶颈分析 / Bottleneck Analysis

### 当前瓶颈 / Current Bottlenecks

1. **LLM API 延迟 / LLM API Latency** (100-1000ms)
   - 解决方案: 流式传输、缓存、批量请求

2. **工具执行时间 / Tool Execution Time** (因工具而异)
   - 解决方案: 并行执行、超时控制

3. **尚未基准测试 / Not yet benchmarked**:
   - Team 协调开销
   - Workflow 执行开销
   - 向量数据库查询

## 生产环境建议 / Production Recommendations

### 1. Agent 池化 / Agent Pooling

重用 agent 实例以减少 GC 压力:

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

### 2. Goroutine 限制 / Goroutine Limits

限制并发以避免资源耗尽:

```go
semaphore := make(chan struct{}, 100) // Max 100 concurrent

for _, task := range tasks {
    semaphore <- struct{}{}
    go func(t Task) {
        defer func() { <-semaphore }()

        ag, _ := agent.New(config)
        ag.Run(ctx, t.Input)
    }(task)
}
```

### 3. 响应缓存 / Response Caching

缓存 LLM 响应以减少 API 调用:

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

### 4. 监控 / Monitoring

在生产环境中监控关键指标:

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

## 运行基准测试 / Running Benchmarks

### 运行所有基准测试 / Run All Benchmarks

```bash
make bench
# or
go test -bench=. -benchmem ./...
```

### 运行特定基准测试 / Run Specific Benchmark

```bash
go test -bench=BenchmarkAgentCreation -benchmem ./pkg/hno/agent/
```

### 生成 CPU 性能分析 / Generate CPU Profile

```bash
go test -bench=. -cpuprofile=cpu.prof ./pkg/hno/agent/
go tool pprof cpu.prof
```

### 生成内存性能分析 / Generate Memory Profile

```bash
go test -bench=. -memprofile=mem.prof ./pkg/hno/agent/
go tool pprof mem.prof
```

## 性能分析技巧 / Profiling Tips

### 1. CPU 性能分析 / CPU Profiling

```bash
go test -cpuprofile=cpu.prof -bench=.
go tool pprof -http=:8080 cpu.prof
```

### 2. 内存性能分析 / Memory Profiling

```bash
go test -memprofile=mem.prof -bench=.
go tool pprof -http=:8080 mem.prof
```

### 3. 竞态检测 / Race Detection

```bash
go test -race ./...
```

## 未来优化 / Future Optimizations

### 计划的改进 / Planned Improvements

- [ ] 重复值的字符串驻留
- [ ] sync.Pool 用于 agent 重用
- [ ] 批量工具执行
- [ ] LLM API 的 HTTP/2 连接池
- [ ] gRPC 支持以降低延迟

## 结论 / Conclusion

HNO **超越性能目标**:

- ✅ 比目标快 5 倍(180ns vs 1μs)
- ✅ 比目标少 60% 内存(1.2KB vs 3KB)
- ✅ 比 Python 快 16 倍,内存少 5 倍
- ✅ 完美的并发扩展

**支持 / Supports**:
- 数千个并发 agents
- 10K+ 请求/秒
- 低延迟实时应用

## 参考资料 / References

- [架构 / Architecture](/advanced/architecture)
- [部署 / Deployment](/advanced/deployment)
- [基准测试代码 / Benchmark Code](https://github.com/rexleimo/HNO/tree/main/pkg/hno/agent/agent_bench_test.go)
