package agent

import "github.com/rexleimo/agno-go/pkg/hno/types"

func (a *Agent) updateSystemMessage(messages []*types.Message, instructions string) []*types.Message {
	if len(messages) == 0 {
		return []*types.Message{types.NewSystemMessage(instructions)}
	}

	// Create a copy to avoid modifying the original
	// 创建副本以避免修改原始消息
	result := make([]*types.Message, 0, len(messages)+1)

	// Check if first message is system message
	// 检查第一条消息是否为系统消息
	systemMessageFound := false
	for i, msg := range messages {
		if i == 0 && msg.Role == types.RoleSystem {
			// Replace first system message
			// 替换第一条系统消息
			result = append(result, types.NewSystemMessage(instructions))
			systemMessageFound = true
		} else {
			result = append(result, msg)
		}
	}

	// If no system message found, prepend one
	// 如果没有找到系统消息，添加一个到开头
	if !systemMessageFound {
		result = append([]*types.Message{types.NewSystemMessage(instructions)}, result...)
	}

	return result
}

// filterToolMessages removes tool-related messages from the slice.
// It filters out messages with Role == RoleTool and clears tool-related fields from other messages.
// filterToolMessages 从切片中移除工具相关消息
