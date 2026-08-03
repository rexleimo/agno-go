package agent

import (
	"context"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/reasoning"
	"github.com/rexleimo/agno-go/pkg/hno/run"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

func (output *RunOutput) appendEvent(evt run.BaseRunOutputEvent) {
	if output == nil || evt == nil {
		return
	}
	output.Events = append(output.Events, evt)
}

// RunStream executes the agent using the model's streaming API and returns
// a pair of channels: one for incremental content events and one that carries
// the final RunOutput once aggregation completes.
//
// The initial behaviour focuses on single-pass streaming responses:
// - Pre-hooks, memory, run-context and post-hooks are honoured.
// - Model.InvokeStream is used to stream content chunks.
// - Tool calls present in streaming chunks are aggregated but not executed.

func (a *Agent) filterToolMessages(messages []*types.Message) []*types.Message {
	if len(messages) == 0 {
		return messages
	}

	// Pre-allocate with same capacity for efficiency / 预分配相同容量以提高效率
	filtered := make([]*types.Message, 0, len(messages))

	for _, msg := range messages {
		// Skip tool response messages entirely / 完全跳过工具响应消息
		if msg.Role == types.RoleTool {
			continue
		}

		// For other messages, clear tool-related fields / 对于其他消息，清除工具相关字段
		if len(msg.ToolCalls) > 0 || msg.ToolCallID != "" {
			// Create a shallow copy to avoid modifying the original message in Memory
			// 创建浅拷贝以避免修改 Memory 中的原始消息
			msgCopy := *msg
			msgCopy.ToolCalls = nil
			msgCopy.ToolCallID = ""
			filtered = append(filtered, &msgCopy)
		} else {
			// No tool data, can use original message / 没有工具数据，可以使用原始消息
			filtered = append(filtered, msg)
		}
	}

	return filtered
}

// filterHistoryMessages removes messages that existed before the current Run.
// It uses the initialCount to determine which messages are historical.
// filterHistoryMessages 移除当前 Run 之前就存在的消息
// 它使用 initialCount 来确定哪些消息是历史消息
func (a *Agent) filterHistoryMessages(messages []*types.Message, initialCount int) []*types.Message {
	// Defensive: handle nil / 防御性: 处理 nil
	if messages == nil {
		return nil
	}

	// Defensive: handle empty / 防御性: 处理空
	if len(messages) == 0 {
		return messages
	}

	// Defensive: handle negative count / 防御性: 处理负数
	if initialCount < 0 {
		initialCount = 0
	}

	// All messages are new / 所有消息都是新的
	if initialCount == 0 {
		return messages
	}

	// All messages are historical / 所有消息都是历史的
	if initialCount >= len(messages) {
		return []*types.Message{}
	}

	// Return new messages (after initialCount) / 返回新消息（initialCount 之后的）
	return messages[initialCount:]
}

// extractReasoning 从响应中提取推理内容(优雅降级)
// extractReasoning extracts reasoning from response (graceful degradation)
func (a *Agent) extractReasoning(ctx context.Context, resp *types.ModelResponse) *types.ReasoningContent {
	// 检查模型是否支持推理 / Check if model supports reasoning
	if !reasoning.IsReasoningModel(a.Model) {
		return nil
	}

	// 提取推理内容 / Extract reasoning content
	reasoningContent, err := reasoning.ExtractReasoning(ctx, a.Model, resp)
	if err != nil {
		// 记录警告但不中断流程 / Log warning but don't interrupt flow
		if a.logger != nil {
			a.logger.Warn("failed to extract reasoning", "error", err)
		}
		return nil
	}

	return reasoningContent
}

// scrubRunOutputWithContext applies filters to RunOutput based on storage configuration.
// It modifies the output in place for performance.
// scrubRunOutputWithContext 根据存储配置对 RunOutput 应用过滤器
// 为了性能考虑，会原地修改输出
func (a *Agent) scrubRunOutputWithContext(output *RunOutput, initialMessageCount int) {
	if output == nil || output.Messages == nil {
		return
	}

	initialCount := len(output.Messages)

	// Filter tool messages first (order matters!) / 先过滤工具消息（顺序很重要！）
	if !a.storeToolMessages {
		// Count tool messages in history (before filtering) to adjust initialMessageCount
		// 计算历史消息中的工具消息数量（在过滤之前），以便调整 initialMessageCount
		toolMessagesInHistory := 0
		if initialMessageCount > 0 && initialMessageCount <= len(output.Messages) {
			for i := 0; i < initialMessageCount; i++ {
				if output.Messages[i].Role == types.RoleTool {
					toolMessagesInHistory++
				}
			}
		}

		output.Messages = a.filterToolMessages(output.Messages)

		// Adjust initialMessageCount by removing count of tool messages that were in history
		// 调整 initialMessageCount，减去历史中的工具消息数量
		if toolMessagesInHistory > 0 {
			initialMessageCount -= toolMessagesInHistory
			if initialMessageCount < 0 {
				initialMessageCount = 0
			}
		}

		initialCount = len(output.Messages)
	}

	// Then filter history messages / 然后过滤历史消息
	if !a.storeHistoryMessages {
		output.Messages = a.filterHistoryMessages(output.Messages, initialMessageCount)
	}

	// Log if messages were filtered / 如果消息被过滤则记录日志
	if len(output.Messages) < initialCount {
		a.logger.Debug("filtered messages from output",
			"original_count", initialCount,
			"filtered_count", len(output.Messages),
			"store_tool_messages", a.storeToolMessages,
			"store_history_messages", a.storeHistoryMessages,
		)
	}
}

func (a *Agent) markRunCancelled(output *RunOutput, loopCount int, cacheHit bool, reason error, initialMessageCount int) *RunOutput {
	if output == nil {
		return nil
	}

	if output.Metadata == nil {
		output.Metadata = map[string]interface{}{}
	}

	output.Status = RunStatusCancelled
	output.CompletedAt = time.Now().UTC()
	output.Metadata["loops"] = loopCount
	output.Metadata["cache_hit"] = cacheHit
	if reason != nil {
		output.CancellationReason = reason.Error()
		output.Metadata["error"] = reason.Error()
	}

	output.Messages = a.Memory.GetMessages(a.UserID)
	a.scrubRunOutputWithContext(output, initialMessageCount)
	return output
}
