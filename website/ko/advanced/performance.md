# 성능

HNO는 극한의 성능을 위해 설계되어, Python Agno 대비 16배 빠른 에이전트 인스턴스화를 달성합니다.

## 요약

✅ **성능 목표 달성**:
- ✅ 에이전트 인스턴스화: **~180ns** (<1μs 목표)
- ✅ 메모리 사용량: **~1.2KB/agent** (<3KB 목표)
- ✅ 동시성: 경합 없는 선형 확장

## 벤치마크 결과

### 에이전트 생성 성능

| 벤치마크 | 시간/op | 메모리/op | 할당/op |
|-----------|---------|-----------|-----------|
| **Simple Agent** | 184.5 ns | 1,272 B (1.2 KB) | 8 |
| **With Tools** | 193.0 ns | 1,288 B (1.3 KB) | 9 |
| **With Memory** | 111.9 ns | 312 B (0.3 KB) | 6 |

**주요 발견**:
- ⚡ 에이전트 생성: **<200 나노초** (1μs 목표 대비 5배 향상!)
- 💾 메모리 사용: **1.2-1.3KB** (3KB 목표 대비 60% 개선)
- 🎯 도구 추가 시 오버헤드 8.5ns만 발생
- 🎯 메모리는 경량 (312B만 사용)

### 실행 성능

| 벤치마크 | 처리량 |
|-----------|------------|
| **Simple Run** | ~6M ops/sec |
| **With Tool Calls** | ~0.5M ops/sec |
| **Memory Operations** | ~1M ops/sec |

**참고**: 실제 성능은 LLM API 지연시간(100-1000ms)에 제한됩니다. 위 결과는 모의 모델 사용.

### 동시 성능

| 벤치마크 | 시간/op | 메모리/op | 확장성 |
|-----------|---------|-----------|---------|
| **Parallel Creation** | 191.0 ns | 1,272 B | ✅ 선형 |
| **Parallel Run** | 유사 | 유사 | ✅ 선형 |

**주요 발견**:
- ✅ 동시 및 단일 스레드 성능 동일
- ✅ 락 경합 또는 경쟁 조건 없음
- ✅ 높은 동시성 시나리오에 완벽

## 성능 비교

### vs Python Agno

| 지표 | Go | Python | 개선 |
|--------|-----|--------|-------------|
| **인스턴스화** | ~180ns | ~3μs | **16배 빠름** |
| **메모리/Agent** | ~1.2KB | ~6.5KB | **5배 적음** |
| **동시성** | 네이티브 고루틴 | GIL 제한 | **우수** |

## 실제 시나리오

### 시나리오 1: 배치 에이전트 생성

1,000개 에이전트 생성:
- **시간**: 1,000 × 180ns = **0.18ms**
- **메모리**: 1,000 × 1.2KB = **1.2MB**

### 시나리오 2: 고동시성 API 서비스

10,000 req/s 처리:
- **요청당**: 1 에이전트 인스턴스
- **메모리 오버헤드**: 10,000 × 1.2KB = **12MB**
- **지연시간**: <1ms (LLM API 호출 제외)

### 시나리오 3: 다중 에이전트 워크플로우

100개 에이전트 협업:
- **총 메모리**: 100 × 1.2KB = **120KB**
- **시작 시간**: 100 × 180ns = **18μs**

## 최적화 기법

### 1. 낮은 할당 횟수

- 에이전트당 8-9회의 힙 할당만
- 불필요한 인터페이스 변환 없음
- 사전 할당된 슬라이스 용량

### 2. 효율적인 메모리 레이아웃

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

### 3. 제로 카피 작업

- 문자열 참조 (복사 없음)
- 인터페이스 포인터 (복사 없음)
- 슬라이스 뷰 (복사 없음)

## 병목 현상 분석

### 현재 병목

1. **LLM API 지연시간** (100-1000ms)
   - 해결책: 스트리밍, 캐싱, 배치 요청

2. **도구 실행 시간** (가변)
   - 해결책: 병렬 실행, 타임아웃 제어

3. **아직 벤치마크되지 않음**:
   - 팀 조정 오버헤드
   - 워크플로우 실행 오버헤드
   - 벡터 DB 쿼리

## 프로덕션 권장사항

### 1. 에이전트 풀링

GC 압력을 줄이기 위해 에이전트 인스턴스 재사용:

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

### 2. 고루틴 제한

리소스 고갈을 방지하기 위한 동시성 제한:

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

### 3. 응답 캐싱

API 호출을 줄이기 위한 LLM 응답 캐싱:

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

### 4. 모니터링

프로덕션에서 주요 지표 모니터링:

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

## 벤치마크 실행

### 모든 벤치마크 실행

```bash
make bench
# 또는
go test -bench=. -benchmem ./...
```

### 특정 벤치마크 실행

```bash
go test -bench=BenchmarkAgentCreation -benchmem ./pkg/hno/agent/
```

### CPU 프로파일 생성

```bash
go test -bench=. -cpuprofile=cpu.prof ./pkg/hno/agent/
go tool pprof cpu.prof
```

### 메모리 프로파일 생성

```bash
go test -bench=. -memprofile=mem.prof ./pkg/hno/agent/
go tool pprof mem.prof
```

## 프로파일링 팁

### 1. CPU 프로파일링

```bash
go test -cpuprofile=cpu.prof -bench=.
go tool pprof -http=:8080 cpu.prof
```

### 2. 메모리 프로파일링

```bash
go test -memprofile=mem.prof -bench=.
go tool pprof -http=:8080 mem.prof
```

### 3. 경쟁 조건 탐지

```bash
go test -race ./...
```

## 향후 최적화

### 계획된 개선 사항

- [ ] 반복 값에 대한 문자열 인터닝
- [ ] 에이전트 재사용을 위한 sync.Pool
- [ ] 배치 도구 실행
- [ ] LLM API용 HTTP/2 연결 풀링
- [ ] 낮은 지연시간을 위한 gRPC 지원

## 결론

HNO는 **성능 목표를 초과 달성**:

- ✅ 목표 대비 5배 빠름 (180ns vs 1μs)
- ✅ 목표 대비 60% 적은 메모리 (1.2KB vs 3KB)
- ✅ Python 대비 16배 빠름, 5배 적은 메모리
- ✅ 완벽한 동시성 확장

**지원**:
- 수천 개의 동시 에이전트
- 10K+ 요청/초
- 저지연 실시간 애플리케이션

## 참고 자료

- [아키텍처](/advanced/architecture)
- [배포](/advanced/deployment)
- [벤치마크 코드](https://github.com/rexleimo/HNO/tree/main/pkg/hno/agent/agent_bench_test.go)
