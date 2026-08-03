package memory

import (
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

func TestInMemory_Add(t *testing.T) {
	mem := NewInMemory(10)

	msg1 := types.NewUserMessage("hello")
	mem.Add(msg1)

	if mem.Size() != 1 {
		t.Errorf("expected size 1, got %d", mem.Size())
	}

	messages := mem.GetMessages()
	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}

	if messages[0].Content != "hello" {
		t.Errorf("expected content 'hello', got '%s'", messages[0].Content)
	}
}

func TestInMemory_MaxSize(t *testing.T) {
	maxSize := 5
	mem := NewInMemory(maxSize)

	// Add system message first
	mem.Add(types.NewSystemMessage("system"))

	// Add more messages than max size
	for i := 0; i < 10; i++ {
		mem.Add(types.NewUserMessage("message"))
	}

	size := mem.Size()
	if size > maxSize {
		t.Errorf("expected size <= %d, got %d", maxSize, size)
	}

	// Check that system message is preserved
	messages := mem.GetMessages()
	hasSystem := false
	for _, msg := range messages {
		if msg.Role == types.RoleSystem {
			hasSystem = true
			break
		}
	}
	if !hasSystem {
		t.Error("expected system message to be preserved")
	}
}

func TestInMemory_Clear(t *testing.T) {
	mem := NewInMemory(10)

	mem.Add(types.NewUserMessage("hello"))
	mem.Add(types.NewUserMessage("world"))

	if mem.Size() != 2 {
		t.Errorf("expected size 2 before clear, got %d", mem.Size())
	}

	mem.Clear()

	if mem.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", mem.Size())
	}
}

func TestInMemory_GetMessages_Copy(t *testing.T) {
	mem := NewInMemory(10)
	mem.Add(types.NewUserMessage("original"))

	messages := mem.GetMessages()
	messages[0].Content = "modified"

	// Original should not be modified
	original := mem.GetMessages()
	if original[0].Content != "original" {
		t.Error("GetMessages should return a copy, not the original slice")
	}
}

// TestInMemory_MultiTenant tests multi-tenant isolation
// 测试多租户隔离
func TestInMemory_MultiTenant(t *testing.T) {
	mem := NewInMemory(10)

	// User 1 adds messages
	mem.Add(types.NewUserMessage("user1 message 1"), "user1")
	mem.Add(types.NewUserMessage("user1 message 2"), "user1")

	// User 2 adds messages
	mem.Add(types.NewUserMessage("user2 message 1"), "user2")
	mem.Add(types.NewUserMessage("user2 message 2"), "user2")

	// User 1 should only see their messages
	user1Messages := mem.GetMessages("user1")
	if len(user1Messages) != 2 {
		t.Errorf("user1 expected 2 messages, got %d", len(user1Messages))
	}
	if user1Messages[0].Content != "user1 message 1" {
		t.Errorf("user1 first message incorrect: got '%s'", user1Messages[0].Content)
	}

	// User 2 should only see their messages
	user2Messages := mem.GetMessages("user2")
	if len(user2Messages) != 2 {
		t.Errorf("user2 expected 2 messages, got %d", len(user2Messages))
	}
	if user2Messages[0].Content != "user2 message 1" {
		t.Errorf("user2 first message incorrect: got '%s'", user2Messages[0].Content)
	}

	// Verify size per user
	if mem.Size("user1") != 2 {
		t.Errorf("user1 size expected 2, got %d", mem.Size("user1"))
	}
	if mem.Size("user2") != 2 {
		t.Errorf("user2 size expected 2, got %d", mem.Size("user2"))
	}
}

// TestInMemory_DefaultUser tests backward compatibility
// 测试向后兼容性
func TestInMemory_DefaultUser(t *testing.T) {
	mem := NewInMemory(10)

	// Add without userID (should use "default")
	mem.Add(types.NewUserMessage("message 1"))
	mem.Add(types.NewUserMessage("message 2"))

	// Get without userID (should get "default" user messages)
	messages := mem.GetMessages()
	if len(messages) != 2 {
		t.Errorf("expected 2 default messages, got %d", len(messages))
	}

	// Verify size without userID
	if mem.Size() != 2 {
		t.Errorf("default size expected 2, got %d", mem.Size())
	}

	// Clear without userID should clear only default user
	mem.Add(types.NewUserMessage("user1 message"), "user1")
	mem.Clear() // Clear default user only

	if mem.Size() != 0 {
		t.Error("default user should be cleared")
	}
	if mem.Size("user1") != 1 {
		t.Error("user1 should still have 1 message")
	}
}

// TestInMemory_ClearAllUsers tests clearing all users
// 测试清除所有用户
func TestInMemory_ClearAllUsers(t *testing.T) {
	mem := NewInMemory(10)

	mem.Add(types.NewUserMessage("user1 message"), "user1")
	mem.Add(types.NewUserMessage("user2 message"), "user2")
	mem.Add(types.NewUserMessage("default message"))

	// ClearAll should clear ALL users
	mem.ClearAll()

	if mem.Size("user1") != 0 {
		t.Error("user1 should be cleared")
	}
	if mem.Size("user2") != 0 {
		t.Error("user2 should be cleared")
	}
	if mem.Size() != 0 {
		t.Error("default user should be cleared")
	}
}

// TestInMemory_MaxSizePerUser tests max size per user
// 测试每个用户的最大大小限制
func TestInMemory_MaxSizePerUser(t *testing.T) {
	maxSize := 3
	mem := NewInMemory(maxSize)

	// Add system message for user1
	mem.Add(types.NewSystemMessage("system"), "user1")

	// Add more messages than max for user1
	for i := 0; i < 5; i++ {
		mem.Add(types.NewUserMessage("message"), "user1")
	}

	// user1 should not exceed maxSize
	if mem.Size("user1") > maxSize {
		t.Errorf("user1 size %d exceeds max %d", mem.Size("user1"), maxSize)
	}

	// System message should be preserved
	messages := mem.GetMessages("user1")
	hasSystem := false
	for _, msg := range messages {
		if msg.Role == types.RoleSystem {
			hasSystem = true
			break
		}
	}
	if !hasSystem {
		t.Error("system message should be preserved for user1")
	}

	// user2 should be independent
	mem.Add(types.NewUserMessage("user2 message"), "user2")
	if mem.Size("user2") != 1 {
		t.Errorf("user2 should have 1 message, got %d", mem.Size("user2"))
	}
}

func TestInMemory_AssignsIDIfMissing(t *testing.T) {
    mem := NewInMemory(10)

    // Manually construct a message without ID
    msg := &types.Message{Role: types.RoleUser, Content: "hi"}
    mem.Add(msg)

    messages := mem.GetMessages()
    if len(messages) != 1 {
        t.Fatalf("expected 1 message, got %d", len(messages))
    }
    if messages[0].ID == "" {
        t.Error("expected memory.Add to assign an ID to message with empty ID")
    }
}
