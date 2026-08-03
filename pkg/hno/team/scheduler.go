package team

import (
	"context"
	"fmt"
	"sync"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// Scheduler decides which agent runs next in a team execution. Each team mode
// has one scheduler; the team acts as a coordinator that owns the run loop,
// hooks and run context, while schedulers own the collaboration strategy.
//
// Scheduler 决定团队执行中下一个运行的 agent。每种团队模式对应一个
// scheduler；team 作为协调器拥有运行循环、钩子与运行上下文，scheduler
// 只负责协作策略。
type Scheduler interface {
	// Next returns the next agent to run, or nil when the run is complete.
	// Next 返回下一个要运行的 agent；运行完成时返回 nil。
	Next(ctx context.Context, t *Team, history []AgentOutput) (*agent.Agent, error)

	// InputFor returns the input for the next agent given the run's original
	// input and the conversation history.
	// InputFor 根据运行原始输入与对话历史返回下一个 agent 的输入。
	InputFor(input string, history []AgentOutput) string
}

// schedulerFor maps a team mode to its scheduler implementation.
// schedulerFor 将团队模式映射到对应的 scheduler 实现。
func schedulerFor(mode TeamMode) Scheduler {
	switch mode {
	case ModeSequential:
		return &SequentialScheduler{}
	case ModeParallel:
		return &ParallelScheduler{}
	case ModeLeaderFollower:
		return &LeaderFollowerScheduler{}
	case ModeConsensus:
		return &ConsensusScheduler{}
	default:
		return nil
	}
}

// SequentialScheduler runs agents one after another, passing the previous
// agent's output as the next agent's input.
//
// SequentialScheduler 依次运行 agents，将前一 agent 的输出作为下一
// agent 的输入。
type SequentialScheduler struct{}

// Next implements Scheduler.
// Next 实现 Scheduler。
func (s *SequentialScheduler) Next(_ context.Context, t *Team, history []AgentOutput) (*agent.Agent, error) {
	if len(history) >= len(t.Agents) {
		return nil, nil
	}
	return t.Agents[len(history)], nil
}

// InputFor returns the original input for the first agent, otherwise the
// previous agent's output.
// InputFor 首个 agent 返回原始输入，其余返回前一 agent 的输出。
func (s *SequentialScheduler) InputFor(input string, history []AgentOutput) string {
	if len(history) == 0 {
		return input
	}
	return history[len(history)-1].Content
}

// ParallelScheduler runs all agents concurrently and combines their outputs.
// ParallelScheduler 并发运行所有 agents 并合并输出。
type ParallelScheduler struct{}

// InputFor returns the original input for every agent.
// InputFor 所有 agent 都使用原始输入。
func (s *ParallelScheduler) InputFor(input string, history []AgentOutput) string {
	return input
}

// Next implements Scheduler: all agents run in the first step.
// Next 实现 Scheduler：所有 agents 在第一步并发运行。
func (s *ParallelScheduler) Next(_ context.Context, t *Team, history []AgentOutput) (*agent.Agent, error) {
	if len(history) >= len(t.Agents) {
		return nil, nil
	}
	// Return the agent at the current history position; the executor runs
	// all of them concurrently.
	// 返回当前位置的 agent；执行器并发运行全部。
	return t.Agents[len(history)], nil
}

// LeaderFollowerScheduler uses the leader to plan, delegates subtasks to
// followers, and synthesizes the final answer.
//
// LeaderFollowerScheduler 由 leader 规划、委派子任务给 followers 并合成
// 最终答案。
type LeaderFollowerScheduler struct{}

// InputFor returns the original input for planning and followers; the
// synthesis stage receives the concatenated outputs.
// InputFor 规划与 followers 用原始输入；合成阶段接收拼接后的输出。
func (s *LeaderFollowerScheduler) InputFor(input string, history []AgentOutput) string {
	if len(history) == 0 {
		return input
	}
	// Synthesis stage (history has plan + all followers).
	// 合成阶段（历史包含 plan + 全部 followers）。
	combined := ""
	for _, out := range history {
		combined += "\n[" + out.AgentID + "]: " + out.Content
	}
	return "You are a team leader. Synthesize these team member outputs into a final answer:\n\nOriginal Task: " +
		input + "\n\nTeam Outputs:" + combined + "\n\nProvide a comprehensive final answer."
}

// Next implements Scheduler. The leader runs first (planning), then all
// followers, then the leader again (synthesis).
//
// Next 实现 Scheduler。leader 先规划，然后全部 followers，
// 最后 leader 合成。
func (s *LeaderFollowerScheduler) Next(_ context.Context, t *Team, history []AgentOutput) (*agent.Agent, error) {
	if t.Leader == nil {
		return nil, fmt.Errorf("leader_follower mode requires a leader")
	}
	stage := len(history)
	followerCount := len(t.Agents)
	switch {
	case stage == 0:
		// Leader planning.
		// leader 规划。
		return t.Leader, nil
	case stage <= followerCount:
		// Followers execute (one per history entry).
		// followers 执行（每个历史条目对应一个）。
		return t.Agents[stage-1], nil
	case stage == followerCount+1:
		// Leader synthesis.
		// leader 合成。
		return t.Leader, nil
	default:
		return nil, nil
	}
}

// ConsensusScheduler runs agents for multiple rounds until they reach
// consensus (identical outputs) or the round limit is hit.
//
// ConsensusScheduler 多轮运行 agents 直到达成共识（输出一致）
// 或达到轮次上限。
type ConsensusScheduler struct{}

// InputFor returns the original input for the first round; later rounds
// receive the accumulated discussion.
// InputFor 首轮返回原始输入；后续轮次接收累积的讨论内容。
func (s *ConsensusScheduler) InputFor(input string, history []AgentOutput) string {
	if len(history) == 0 {
		return input
	}
	combined := input
	for _, out := range history {
		combined += "\n[" + out.AgentID + "]: " + out.Content
	}
	return combined + "\n\nIf you agree with the above, reply with the same answer."
}

// Next implements Scheduler: agents rotate within a round; the scheduler
// signals completion when outputs converge or rounds are exhausted.
//
// Next 实现 Scheduler：agents 在轮内轮转；输出收敛或轮次耗尽时
// 返回 nil 结束。
func (s *ConsensusScheduler) Next(_ context.Context, t *Team, history []AgentOutput) (*agent.Agent, error) {
	maxRounds := t.MaxRounds
	if maxRounds <= 0 {
		maxRounds = 3
	}
	agentCount := len(t.Agents)
	if agentCount == 0 {
		return nil, nil
	}

	// A "round" is agentCount executions. Stop when rounds are exhausted.
	// 一个"轮"是 agentCount 次执行。轮次耗尽时停止。
	if len(history) >= maxRounds*agentCount {
		return nil, nil
	}

	// After the first round, check for convergence: all agents produced the
	// same content in the last round.
	// 首轮之后检查收敛：上一轮所有 agents 输出一致。
	if len(history) >= agentCount {
		lastRound := history[len(history)-agentCount:]
		first := lastRound[0].Content
		converged := true
		for _, out := range lastRound[1:] {
			if out.Content != first {
				converged = false
				break
			}
		}
		if converged && first != "" {
			return nil, nil
		}
	}

	return t.Agents[len(history)%agentCount], nil
}

// runParallelOnce executes all agents concurrently and returns their outputs
// in deterministic order. It is the shared kernel for parallel execution used
// by the parallel scheduler and the leader-follower follower stage.
//
// runParallelOnce 并发执行所有 agents 并按确定性顺序返回输出。
// 它是并行执行（parallel scheduler 与 leader-follower 的 follower 阶段）
// 共享的内核。
func (t *Team) runParallelOnce(ctx context.Context, agents []*agent.Agent, input string) ([]*AgentOutput, error) {
	type result struct {
		index  int
		output *AgentOutput
		err    error
	}

	results := make(chan result, len(agents))
	var wg sync.WaitGroup
	for i, ag := range agents {
		wg.Add(1)
		go func(idx int, a *agent.Agent) {
			defer wg.Done()
			runOut, err := t.invokeAgent(ctx, a, input)
			var out *AgentOutput
			if runOut != nil {
				out = &AgentOutput{AgentID: a.ID, Content: runOut.Content}
			}
			results <- result{index: idx, output: out, err: err}
		}(i, ag)
	}
	wg.Wait()
	close(results)

	outputs := make([]*AgentOutput, len(agents))
	var firstErr error
	for r := range results {
		if r.err != nil {
			if firstErr == nil {
				firstErr = types.NewError(types.ErrCodeUnknown,
					fmt.Sprintf("agent %s failed", agents[r.index].ID), r.err)
			}
			continue
		}
		outputs[r.index] = r.output
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return outputs, nil
}
