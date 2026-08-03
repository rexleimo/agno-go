# Team 협업 예제

## 개요

이 예제는 HNO의 멀티 Agent 팀 협업 기능을 보여줍니다. Team을 사용하면 여러 Agent가 Sequential, Parallel, Leader-Follower, Consensus 등 다양한 협업 모드를 사용하여 함께 작업할 수 있습니다. 각 모드는 서로 다른 유형의 작업 및 협업 패턴에 적합합니다.

## 학습 내용

- 멀티 Agent 팀을 만드는 방법
- 네 가지 팀 협업 모드와 각각을 사용해야 하는 경우
- Agent가 컨텍스트를 공유하고 서로의 작업을 기반으로 구축하는 방법
- 개별 Agent 출력에 액세스하는 방법

## 사전 요구 사항

- Go 1.21 이상
- OpenAI API 키

## 설정

```bash
export OPENAI_API_KEY=sk-your-api-key-here
cd cmd/examples/team_demo
```

## 전체 코드

전체 예제에는 4개의 데모가 포함되어 있습니다 - 자세한 내용은 아래 코드 설명을 참조하세요.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/openai"
	"github.com/rexleimo/agno-go/pkg/hno/team"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	ctx := context.Background()

	// Demo 1: Sequential Team
	fmt.Println("=== Demo 1: Sequential Team ===")
	runSequentialDemo(ctx, apiKey)

	fmt.Println("\n=== Demo 2: Parallel Team ===")
	runParallelDemo(ctx, apiKey)

	fmt.Println("\n=== Demo 3: Leader-Follower Team ===")
	runLeaderFollowerDemo(ctx, apiKey)

	fmt.Println("\n=== Demo 4: Consensus Team ===")
	runConsensusDemo(ctx, apiKey)
}
```

## Team 협업 모드

### 1. Sequential 모드

Agent들이 차례로 작업하며, 각 Agent가 이전 Agent의 출력을 기반으로 구축합니다.

```go
func runSequentialDemo(ctx context.Context, apiKey string) {
	model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

	// Create 3 agents for sequential processing
	researcher, _ := agent.New(agent.Config{
		Name:         "Researcher",
		Model:        model,
		Instructions: "You are a research expert. Analyze the topic and provide key facts.",
	})

	analyst, _ := agent.New(agent.Config{
		Name:         "Analyst",
		Model:        model,
		Instructions: "You are an analyst. Take research findings and extract insights.",
	})

	writer, _ := agent.New(agent.Config{
		Name:         "Writer",
		Model:        model,
		Instructions: "You are a writer. Take insights and write a concise summary.",
	})

	// Create sequential team
	t, _ := team.New(team.Config{
		Name:   "Content Pipeline",
		Agents: []*agent.Agent{researcher, analyst, writer},
		Mode:   team.ModeSequential,
	})

	// Run team
	output, _ := t.Run(ctx, "Analyze the benefits of AI in healthcare")

	fmt.Printf("Final Output: %s\n", output.Content)
	fmt.Printf("Agents involved: %d\n", len(output.AgentOutputs))
}
```

**흐름:**
1. **Researcher**가 주제 분석 → 연구 결과 생성
2. **Analyst**가 결과 수신 → 인사이트 추출
3. **Writer**가 인사이트 수신 → 최종 요약 작성

**사용 사례:**
- 콘텐츠 생성 파이프라인
- 데이터 처리 워크플로우
- 다단계 분석 작업

### 2. Parallel 모드

모든 Agent가 동일한 입력에 대해 동시에 작업하고 출력을 결합합니다.

```go
func runParallelDemo(ctx context.Context, apiKey string) {
	model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

	// Create agents with different specializations
	techAgent, _ := agent.New(agent.Config{
		Name:         "Tech Specialist",
		Model:        model,
		Instructions: "You are a technology expert. Focus on technical aspects.",
	})

	bizAgent, _ := agent.New(agent.Config{
		Name:         "Business Specialist",
		Model:        model,
		Instructions: "You are a business expert. Focus on business implications.",
	})

	ethicsAgent, _ := agent.New(agent.Config{
		Name:         "Ethics Specialist",
		Model:        model,
		Instructions: "You are an ethics expert. Focus on ethical considerations.",
	})

	// Create parallel team
	t, _ := team.New(team.Config{
		Name:   "Multi-Perspective Analysis",
		Agents: []*agent.Agent{techAgent, bizAgent, ethicsAgent},
		Mode:   team.ModeParallel,
	})

	output, _ := t.Run(ctx, "Evaluate the impact of autonomous vehicles")
	fmt.Printf("Combined Analysis:\n%s\n", output.Content)
}
```

**흐름:**
1. 모든 Agent가 동일한 입력을 동시에 수신
2. 각 Agent가 자신의 관점 제공
3. 출력이 포괄적인 분석으로 결합

**사용 사례:**
- 다각적 분석
- 브레인스토밍 세션
- 독립적인 평가
- 병렬 데이터 처리

### 3. Leader-Follower 모드

리더 Agent가 팔로워 Agent에게 작업을 위임하고 결과를 종합합니다.

```go
func runLeaderFollowerDemo(ctx context.Context, apiKey string) {
	model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

	// Create leader
	leader, _ := agent.New(agent.Config{
		Name:         "Team Leader",
		Model:        model,
		Instructions: "You are a team leader. Delegate tasks and synthesize results.",
	})

	// Create followers with tools
	calcAgent, _ := agent.New(agent.Config{
		Name:         "Calculator Agent",
		Model:        model,
		Toolkits:     []toolkit.Toolkit{calculator.New()},
		Instructions: "You perform calculations as requested.",
	})

	dataAgent, _ := agent.New(agent.Config{
		Name:         "Data Agent",
		Model:        model,
		Instructions: "You analyze and present data.",
	})

	// Create leader-follower team
	t, _ := team.New(team.Config{
		Name:   "Project Team",
		Leader: leader,
		Agents: []*agent.Agent{calcAgent, dataAgent},
		Mode:   team.ModeLeaderFollower,
	})

	output, _ := t.Run(ctx, "Calculate the ROI for a $100,000 investment with 15% annual return over 5 years")
	fmt.Printf("Leader's Final Report: %s\n", output.Content)
}
```

**흐름:**
1. **Leader**가 작업을 분석하고 팔로워에게 위임
2. **Follower**가 할당된 하위 작업 실행
3. **Leader**가 결과를 종합하고 최종 출력 제공

**사용 사례:**
- 복잡한 작업 분해
- 계층적 워크플로우
- 프로젝트 관리 시나리오
- 전문 도구 사용

### 4. Consensus 모드

Agent들이 합의에 도달하거나 최대 라운드에 도달할 때까지 토론합니다.

```go
func runConsensusDemo(ctx context.Context, apiKey string) {
	model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

	// Create agents with different perspectives
	optimist, _ := agent.New(agent.Config{
		Name:         "Optimist",
		Model:        model,
		Instructions: "You are optimistic and focus on opportunities.",
	})

	realist, _ := agent.New(agent.Config{
		Name:         "Realist",
		Model:        model,
		Instructions: "You are realistic and balanced in your views.",
	})

	critic, _ := agent.New(agent.Config{
		Name:         "Critic",
		Model:        model,
		Instructions: "You are critical and focus on potential problems.",
	})

	// Create consensus team
	t, _ := team.New(team.Config{
		Name:      "Decision Team",
		Agents:    []*agent.Agent{optimist, realist, critic},
		Mode:      team.ModeConsensus,
		MaxRounds: 2,
	})

	output, _ := t.Run(ctx, "Should we invest in renewable energy for our company?")

	fmt.Printf("Consensus Result: %s\n", output.Content)
	fmt.Printf("Total discussion rounds: %v\n", output.Metadata["rounds"])
}
```

**흐름:**
1. **라운드 1**: 각 Agent가 초기 관점 제공
2. **라운드 2**: Agent들이 다른 사람의 의견을 보고 입장 개선
3. **최종**: 시스템이 합의 또는 최종 입장 종합

**사용 사례:**
- 의사 결정
- 토론 시뮬레이션
- 다관점 분석
- 위험 평가

## Team 구성

### 기본 구성

```go
team.Config{
	Name:   "My Team",           // 팀 식별자
	Agents: []*agent.Agent{...}, // 팀 멤버
	Mode:   team.ModeSequential, // 협업 모드
}
```

### 고급 구성

```go
team.Config{
	Name:      "Decision Team",
	Leader:    leaderAgent,      // Leader-Follower 모드용
	Agents:    followerAgents,   // 팀 멤버
	Mode:      team.ModeConsensus,
	MaxRounds: 3,                // Consensus 모드용
}
```

## 결과 액세스

### Team 출력

```go
output, err := t.Run(ctx, "Your query here")

// 최종 결과
fmt.Println(output.Content)

// 개별 Agent 출력
for _, agentOut := range output.AgentOutputs {
	fmt.Printf("%s: %s\n", agentOut.AgentName, agentOut.Content)
}

// 메타데이터
fmt.Printf("Rounds: %v\n", output.Metadata["rounds"])
```

### 개별 Agent 출력

```go
// 특정 Agent의 기여 액세스
if len(output.AgentOutputs) > 0 {
	firstAgent := output.AgentOutputs[0]
	fmt.Printf("Agent: %s\n", firstAgent.AgentName)
	fmt.Printf("Output: %s\n", firstAgent.Content)
}
```

## 예제 실행

```bash
go run main.go
```

## 예상 출력

```
=== Demo 1: Sequential Team ===
Final Output: AI in healthcare offers significant benefits including improved diagnostic accuracy through machine learning, personalized treatment plans, reduced administrative burden, and enhanced patient monitoring through IoT devices.
Agents involved: 3

=== Demo 2: Parallel Team ===
Combined Analysis:
Technical: Autonomous vehicles use advanced sensors, AI algorithms, and real-time processing...
Business: Market disruption, new revenue models, infrastructure investment needs...
Ethics: Privacy concerns, liability questions, job displacement, safety standards...

=== Demo 3: Leader-Follower Team ===
Leader's Final Report: Based on calculations, a $100,000 investment at 15% annual return over 5 years yields $201,136, representing a 101% ROI.

=== Demo 4: Consensus Team ===
Consensus Result: After thorough discussion, the team recommends investing in renewable energy with careful planning for upfront costs and long-term savings.
Total discussion rounds: 2
```

## 모드 비교

| 모드 | 사용 시기 | Agent 수 | 통신 패턴 |
|------|-------------|-------------|----------------------|
| **Sequential** | 파이프라인 작업, 순서가 있는 단계 | 2-10 | 선형: A → B → C |
| **Parallel** | 독립적인 작업, 여러 관점 | 2-20 | 브로드캐스트: 모두 같은 입력 |
| **Leader-Follower** | 복잡한 위임, 계층 구조 | 1 리더 + 1-10 팔로워 | 허브-스포크: 리더가 조정 |
| **Consensus** | 의사 결정, 토론 | 2-5 | 라운드 로빈 토론 |

## 모범 사례

### 1. 올바른 모드 선택

```go
// Sequential: 순서가 중요할 때
team.ModeSequential  // 연구 → 분석 → 작성

// Parallel: 여러 관점이 필요할 때
team.ModeParallel    // 기술 + 비즈니스 + 법률 분석

// Leader-Follower: 위임이 필요할 때
team.ModeLeaderFollower  // 복잡한 작업 분해

// Consensus: 합의가 필요할 때
team.ModeConsensus   // 의사 결정, 토론
```

### 2. 명확한 Agent 역할 설계

```go
// ✅ 좋음: 구체적이고 뚜렷한 역할
researcher := "You are a research expert. Focus on facts and data."
analyst := "You are an analyst. Extract insights from research."

// ❌ 나쁨: 겹치고 모호한 역할
agent1 := "You are helpful."
agent2 := "You are smart."
```

### 3. Agent 수 최적화

- **Sequential**: 2-5 Agent (더 많으면 = 더 긴 파이프라인)
- **Parallel**: 2-10 Agent (더 많으면 = 더 풍부한 분석)
- **Leader-Follower**: 1 리더 + 2-5 팔로워
- **Consensus**: 2-4 Agent (더 많으면 = 수렴하기 어려움)

### 4. 오류 처리

```go
output, err := team.Run(ctx, query)
if err != nil {
	log.Printf("Team execution failed: %v", err)
	// 대체 로직
}
```

## 고급 패턴

### 혼합 도구 사용

```go
// 일부 Agent는 도구가 있고 다른 Agent는 없음
calcAgent := agent.New(agent.Config{
	Toolkits: []toolkit.Toolkit{calculator.New()},
})

analysisAgent := agent.New(agent.Config{
	// 도구 없음, 순수 추론
})
```

### 동적 팀 구성

```go
var agents []*agent.Agent

if needsCalculation {
	agents = append(agents, calcAgent)
}
if needsWebSearch {
	agents = append(agents, searchAgent)
}

team, _ := team.New(team.Config{Agents: agents, Mode: team.ModeParallel})
```

### 중첩 팀

```go
// 하위 팀 생성
researchTeam := team.New(team.Config{...})
analysisTeam := team.New(team.Config{...})

// 한 팀의 출력을 다른 팀의 입력으로 사용
researchOutput, _ := researchTeam.Run(ctx, query)
finalOutput, _ := analysisTeam.Run(ctx, researchOutput.Content)
```

## 성능 고려 사항

### Sequential 모드
- **지연 시간**: 모든 Agent 시간의 합 (가장 느림)
- **비용**: 모든 Agent 비용의 합
- **최적 용도**: 순서가 중요할 때

### Parallel 모드
- **지연 시간**: Agent 시간의 최대값 (더 빠름)
- **비용**: 모든 Agent 비용의 합
- **최적 용도**: 속도가 중요할 때

### Leader-Follower 모드
- **지연 시간**: 리더 + 팔로워 (중간)
- **비용**: 리더 + 팔로워 비용
- **최적 용도**: 복잡한 작업 위임

### Consensus 모드
- **지연 시간**: 라운드 × Agent 시간 (가장 느림)
- **비용**: 라운드 × Agent 수
- **최적 용도**: 합의가 중요할 때

## 다음 단계

- [Simple Agent](./simple-agent.md) 기초로 시작
- 제어된 실행을 위한 [Workflow 엔진](./workflow-demo.md) 탐색
- 팀 협업으로 [RAG 시스템](./rag-demo.md) 구축
- 다양한 [모델 공급자](./claude-agent.md) 시도

## 문제 해결

**Agent가 효과적으로 협업하지 않음:**
- 명확성을 위해 Agent 지침 검토
- 모드가 작업에 맞는지 확인
- Agent가 뚜렷한 역할을 가지고 있는지 확인

**Sequential 팀이 너무 느림:**
- Agent 수 줄이기
- 더 작고 빠른 모델 사용
- Parallel 모드 고려

**Consensus가 수렴하지 않음:**
- MaxRounds 늘리기
- 결정 단순화
- Agent 수 줄이기
- Agent 지침 조정

**리더가 제대로 위임하지 않음:**
- 리더의 위임 지침 명확화
- 팔로워가 적절한 도구를 가지고 있는지 확인
- 팔로워 지침이 명확한지 확인
