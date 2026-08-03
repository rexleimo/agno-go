# Team - 멀티 에이전트 협업

4가지 협업 모드로 강력한 멀티 에이전트 시스템을 구축하세요.

---

## Team이란?

**Team**은 복잡한 작업을 해결하기 위해 함께 작업하는 에이전트 모음입니다. 다양한 협업 모드는 다양한 협업 패턴을 가능하게 합니다.

### 주요 기능

- **4가지 협업 모드**: Sequential, Parallel, Leader-Follower, Consensus
- **동적 멤버십**: 런타임 시 에이전트 추가/제거
- **유연한 구성**: 모드별 동작 커스터마이즈
- **타입 안전**: 완전한 Go 타입 체크

---

## Team 생성

### 기본 예제

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/team"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
)

func main() {
    // 모델 생성
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: "your-api-key",
    })

    // 팀 멤버 생성
    researcher, _ := agent.New(agent.Config{
        Name:         "Researcher",
        Model:        model,
        Instructions: "You are a research expert.",
    })

    writer, _ := agent.New(agent.Config{
        Name:         "Writer",
        Model:        model,
        Instructions: "You are a technical writer.",
    })

    // 팀 생성
    t, err := team.New(team.Config{
        Name:   "Content Team",
        Agents: []*agent.Agent{researcher, writer},
        Mode:   team.ModeSequential,
    })
    if err != nil {
        log.Fatal(err)
    }

    // 팀 실행
    output, _ := t.Run(context.Background(), "Write about AI")
    fmt.Println(output.Content)
}
```

---

## 협업 모드

### 1. Sequential 모드

에이전트가 순차적으로 실행되며, 출력이 다음 에이전트로 전달됩니다.

```go
t, _ := team.New(team.Config{
    Name:   "Pipeline",
    Agents: []*agent.Agent{agent1, agent2, agent3},
    Mode:   team.ModeSequential,
})
```

**사용 사례:**
- 콘텐츠 파이프라인 (연구 → 작성 → 편집)
- 데이터 처리 워크플로우
- 다단계 추론

**작동 방식:**
1. Agent 1이 입력 처리 → 출력 A
2. Agent 2가 출력 A 처리 → 출력 B
3. Agent 3이 출력 B 처리 → 최종 출력

---

### 2. Parallel 모드

모든 에이전트가 동시에 실행되며, 결과가 결합됩니다.

```go
t, _ := team.New(team.Config{
    Name:   "Multi-Perspective",
    Agents: []*agent.Agent{techAgent, bizAgent, ethicsAgent},
    Mode:   team.ModeParallel,
})
```

**사용 사례:**
- 다각도 분석
- 병렬 데이터 처리
- 다양한 의견 생성

**작동 방식:**
1. 모든 에이전트가 동일한 입력 수신
2. 동시 실행 (Go goroutine 사용)
3. 결과가 단일 출력으로 결합

---

### 3. Leader-Follower 모드

리더가 팔로워에게 작업을 위임하고 결과를 통합합니다.

```go
t, _ := team.New(team.Config{
    Name:   "Project Team",
    Leader: leaderAgent,
    Agents: []*agent.Agent{follower1, follower2},
    Mode:   team.ModeLeaderFollower,
})
```

**사용 사례:**
- 작업 위임
- 계층적 워크플로우
- 전문가 자문

**작동 방식:**
1. 리더가 작업을 분석하고 하위 작업 생성
2. 적절한 팔로워에게 위임
3. 팔로워 출력을 최종 결과로 통합

---

### 4. Consensus 모드

에이전트가 합의에 도달할 때까지 토론합니다.

```go
t, _ := team.New(team.Config{
    Name:      "Decision Team",
    Agents:    []*agent.Agent{optimist, realist, critic},
    Mode:      team.ModeConsensus,
    MaxRounds: 3,  // 최대 토론 라운드
})
```

**사용 사례:**
- 의사 결정
- 품질 보증
- 토론 및 개선

**작동 방식:**
1. 모든 에이전트가 초기 의견 제공
2. 에이전트가 다른 의견 검토
3. 합의 또는 최대 라운드까지 반복
4. 최종 합의 출력

---

## 구성

### Config 구조체

```go
type Config struct {
    // 필수
    Agents []*agent.Agent  // 팀 멤버

    // 선택
    Name      string              // 팀 이름 (기본값: "Team")
    Mode      CoordinationMode    // 협업 모드 (기본값: Sequential)
    Leader    *agent.Agent        // 리더 (LeaderFollower 모드용)
    MaxRounds int                 // 최대 라운드 (Consensus 모드용, 기본값: 3)
}
```

### 협업 모드

```go
const (
    ModeSequential     CoordinationMode = "sequential"
    ModeParallel       CoordinationMode = "parallel"
    ModeLeaderFollower CoordinationMode = "leader-follower"
    ModeConsensus      CoordinationMode = "consensus"
)
```

---

## API 레퍼런스

### team.New

새 팀 인스턴스를 생성합니다.

**시그니처:**
```go
func New(config Config) (*Team, error)
```

**반환값:**
- `*Team`: 생성된 팀 인스턴스
- `error`: 에이전트 목록이 비어있거나 구성이 잘못된 경우 오류

---

### Team.Run

입력으로 팀을 실행합니다.

**시그니처:**
```go
func (t *Team) Run(ctx context.Context, input string) (*RunOutput, error)
```

**매개변수:**
- `ctx`: 취소/타임아웃을 위한 컨텍스트
- `input`: 사용자 입력 문자열

**반환값:**
```go
type RunOutput struct {
    Content      string                 // 최종 팀 출력
    AgentOutputs []AgentOutput          // 개별 에이전트 출력
    Metadata     map[string]interface{} // 추가 메타데이터
}
```

---

### Team.AddAgent / RemoveAgent

팀 멤버를 동적으로 관리합니다.

**시그니처:**
```go
func (t *Team) AddAgent(ag *agent.Agent)
func (t *Team) RemoveAgent(name string) error
func (t *Team) GetAgents() []*agent.Agent
```

**예제:**
```go
// 새 에이전트 추가
t.AddAgent(newAgent)

// 이름으로 에이전트 제거
err := t.RemoveAgent("OldAgent")

// 모든 에이전트 가져오기
agents := t.GetAgents()
```

---

## 완전한 예제

### Sequential Team 예제

연구 → 분석 → 작성 콘텐츠 생성 파이프라인.

```go
func createContentPipeline(apiKey string) {
    model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

    researcher, _ := agent.New(agent.Config{
        Name:         "Researcher",
        Model:        model,
        Instructions: "Research the topic and provide key facts.",
    })

    analyst, _ := agent.New(agent.Config{
        Name:         "Analyst",
        Model:        model,
        Instructions: "Analyze research findings and extract insights.",
    })

    writer, _ := agent.New(agent.Config{
        Name:         "Writer",
        Model:        model,
        Instructions: "Write a concise summary based on insights.",
    })

    t, _ := team.New(team.Config{
        Name:   "Content Pipeline",
        Agents: []*agent.Agent{researcher, analyst, writer},
        Mode:   team.ModeSequential,
    })

    output, _ := t.Run(context.Background(),
        "Write about the benefits of AI in healthcare")

    fmt.Println(output.Content)
}
```

### Parallel Team 예제

동시 실행을 통한 다각도 분석.

```go
func multiPerspectiveAnalysis(apiKey string) {
    model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

    techAgent, _ := agent.New(agent.Config{
        Name:         "Tech Specialist",
        Model:        model,
        Instructions: "Focus on technical aspects.",
    })

    bizAgent, _ := agent.New(agent.Config{
        Name:         "Business Specialist",
        Model:        model,
        Instructions: "Focus on business implications.",
    })

    ethicsAgent, _ := agent.New(agent.Config{
        Name:         "Ethics Specialist",
        Model:        model,
        Instructions: "Focus on ethical considerations.",
    })

    t, _ := team.New(team.Config{
        Name:   "Analysis Team",
        Agents: []*agent.Agent{techAgent, bizAgent, ethicsAgent},
        Mode:   team.ModeParallel,
    })

    output, _ := t.Run(context.Background(),
        "Evaluate the impact of autonomous vehicles")

    fmt.Println(output.Content)
}
```

---

## 모범 사례

### 1. 올바른 모드 선택

- **Sequential**: 출력이 이전 단계에 의존할 때 사용
- **Parallel**: 관점이 독립적일 때 사용
- **Leader-Follower**: 작업 위임이 필요할 때 사용
- **Consensus**: 품질과 합의가 중요할 때 사용

### 2. 에이전트 전문화

각 에이전트에게 명확하고 구체적인 지침 제공:

```go
// 좋음 ✅
Instructions: "You are a Python expert. Focus on code quality."

// 나쁨 ❌
Instructions: "You help with coding."
```

### 3. 오류 처리

항상 팀 작업의 오류를 처리:

```go
output, err := t.Run(ctx, input)
if err != nil {
    log.Printf("Team execution failed: %v", err)
    return
}
```

### 4. Context 관리

타임아웃 및 취소를 위한 컨텍스트 사용:

```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

output, err := t.Run(ctx, input)
```

---

## 성능 고려사항

### 병렬 실행

Parallel 모드는 진정한 동시성을 위해 Go goroutine 사용:

```go
// 3개 에이전트가 동시에 실행
t, _ := team.New(team.Config{
    Agents: []*agent.Agent{a1, a2, a3},
    Mode:   team.ModeParallel,
})

// 총 시간 ≈ 가장 느린 에이전트 (전체 합이 아님)
```

### 메모리 사용

각 에이전트는 자체 메모리를 유지합니다. 대규모 팀의 경우:

```go
// 각 실행 후 메모리 지우기
output, _ := t.Run(ctx, input)
for _, ag := range t.GetAgents() {
    ag.ClearMemory()
}
```

---

## 다음 단계

- 단계 기반 오케스트레이션을 위한 [Workflow](/guide/workflow) 배우기
- 다양한 LLM 제공업체를 위한 [Models](/guide/models) 탐색
- 에이전트 능력 향상을 위한 [Tools](/guide/tools) 추가
- 자세한 API 문서는 [Team API Reference](/api/team) 확인

---

## 관련 예제

- [Team Demo](/examples/team-demo) - 완전한 작동 예제
- [Leader-Follower Pattern](/examples/team-demo#leader-follower)
- [Consensus Decision Making](/examples/team-demo#consensus)
