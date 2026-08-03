package memory

import (
    "sync"

    "github.com/rexleimo/agno-go/pkg/hno/types"
    "github.com/google/uuid"
)

// Memory manages conversation history
// Memory 管理对话历史
type Memory interface {
	// Add appends a message to memory for a specific user
	// Add 为特定用户添加消息到内存
	// userID can be empty string for non-multi-tenant scenarios
	// userID 可以为空字符串（非多租户场景）
	Add(message *types.Message, userID ...string)

	// GetMessages returns all messages for a specific user
	// GetMessages 返回特定用户的所有消息
	// userID can be empty string for non-multi-tenant scenarios
	// userID 可以为空字符串（非多租户场景）
	GetMessages(userID ...string) []*types.Message

	// Clear removes all messages for a specific user (or all users if userID is empty)
	// Clear 删除特定用户的所有消息（如果userID为空则删除所有用户）
	Clear(userID ...string)

	// Size returns the number of messages for a specific user
	// Size 返回特定用户的消息数量
	Size(userID ...string) int
}

// InMemory provides simple in-memory message storage with multi-tenant support
// InMemory 提供简单的内存消息存储，支持多租户
type InMemory struct {
	// userMessages stores messages per user (key: userID, value: messages)
	// userMessages 按用户存储消息（键：userID，值：消息列表）
	userMessages map[string][]*types.Message
	maxSize      int
	mu           sync.RWMutex
}

// NewInMemory creates a new in-memory storage
// NewInMemory 创建新的内存存储
func NewInMemory(maxSize int) *InMemory {
	if maxSize <= 0 {
		maxSize = 100 // default
	}
	return &InMemory{
		userMessages: make(map[string][]*types.Message),
		maxSize:      maxSize,
	}
}

// getUserID returns the userID from variadic parameter, defaults to "default"
// getUserID 从可变参数获取userID，默认为"default"
func getUserID(userID ...string) string {
	if len(userID) > 0 && userID[0] != "" {
		return userID[0]
	}
	return "default"
}

// Add appends a message to memory for a specific user
// Add 为特定用户添加消息到内存
func (m *InMemory) Add(message *types.Message, userID ...string) {
    m.mu.Lock()
    defer m.mu.Unlock()

    uid := getUserID(userID...)

    // Ensure message has an ID for traceability
    if message != nil && message.ID == "" {
        message.ID = "msg-" + uuid.NewString()
    }

	// Initialize user's message list if not exists
	// 如果用户的消息列表不存在则初始化
	if m.userMessages[uid] == nil {
		m.userMessages[uid] = make([]*types.Message, 0, m.maxSize)
	}

	m.userMessages[uid] = append(m.userMessages[uid], message)

	// Trim if exceeds max size (keep recent messages)
	// 如果超过最大大小则修剪（保留最近的消息）
	if len(m.userMessages[uid]) > m.maxSize {
		// Keep system messages and recent messages
		// 保留系统消息和最近的消息
		systemMsgs := make([]*types.Message, 0)
		for _, msg := range m.userMessages[uid] {
			if msg.Role == types.RoleSystem {
				systemMsgs = append(systemMsgs, msg)
			}
		}

		// Keep system messages + most recent messages
		// 保留系统消息 + 最近的消息
		keepCount := m.maxSize - len(systemMsgs)
		if keepCount > 0 {
			recentMsgs := m.userMessages[uid][len(m.userMessages[uid])-keepCount:]
			m.userMessages[uid] = append(systemMsgs, recentMsgs...)
		} else {
			m.userMessages[uid] = systemMsgs
		}
	}
}

// GetMessages returns all messages for a specific user
// GetMessages 返回特定用户的所有消息
func (m *InMemory) GetMessages(userID ...string) []*types.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uid := getUserID(userID...)

	userMsgs := m.userMessages[uid]
	if userMsgs == nil {
		return []*types.Message{}
	}

	// Return a deep copy to prevent external modification
	// 返回深拷贝以防止外部修改
	messages := make([]*types.Message, len(userMsgs))
	for i, msg := range userMsgs {
		// Create a copy of the message
		msgCopy := *msg
		messages[i] = &msgCopy
	}
	return messages
}

// Clear removes all messages for a specific user
// Clear 删除特定用户的所有消息
// If called without userID, clears the default user (for backward compatibility)
// 如果不带userID调用，清除默认用户（向后兼容）
// To clear ALL users, call ClearAll()
// 要清除所有用户，调用ClearAll()
func (m *InMemory) Clear(userID ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	uid := getUserID(userID...)
	m.userMessages[uid] = make([]*types.Message, 0, m.maxSize)
}

// ClearAll removes all messages for all users
// ClearAll 删除所有用户的所有消息
func (m *InMemory) ClearAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.userMessages = make(map[string][]*types.Message)
}

// Size returns the number of messages for a specific user
// Size 返回特定用户的消息数量
func (m *InMemory) Size(userID ...string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uid := getUserID(userID...)
	if m.userMessages[uid] == nil {
		return 0
	}
	return len(m.userMessages[uid])
}
