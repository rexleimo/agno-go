package agentos

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/rexleimo/agno-go/pkg/hno/media"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

func TestEventTypes(t *testing.T) {
	// 测试所有事件类型常量
	// Test all event type constants
	assert.Equal(t, EventType("run_start"), EventRunStart)
	assert.Equal(t, EventType("reasoning"), EventReasoning)
	assert.Equal(t, EventType("tool_call"), EventToolCall)
	assert.Equal(t, EventType("token"), EventToken)
	assert.Equal(t, EventType("step_start"), EventStepStart)
	assert.Equal(t, EventType("step_end"), EventStepEnd)
	assert.Equal(t, EventType("error"), EventError)
	assert.Equal(t, EventType("complete"), EventComplete)
}

func TestNewEvent(t *testing.T) {
	data := RunStartData{
		Input:     "test input",
		SessionID: "session123",
	}

	event := NewEvent(EventRunStart, data)

	assert.Equal(t, EventRunStart, event.Type)
	assert.NotNil(t, event.Timestamp)
	assert.Equal(t, data, event.Data)
}

func TestEventToSSE(t *testing.T) {
	data := CompleteData{
		Output:   "test output",
		Duration: 1.5,
	}

	event := &Event{
		Type:      EventComplete,
		Timestamp: time.Now(),
		Data:      data,
		SessionID: "session123",
		AgentID:   "agent456",
	}

	sse := event.ToSSE()

	// 验证 SSE 格式
	// Verify SSE format
	assert.True(t, strings.HasPrefix(sse, "event: complete\n"))
	assert.True(t, strings.Contains(sse, "data: "))
	assert.True(t, strings.HasSuffix(sse, "\n\n"))

	// 验证可以解析为 JSON
	// Verify can be parsed as JSON
	lines := strings.Split(sse, "\n")
	var dataLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			dataLine = strings.TrimPrefix(line, "data: ")
			break
		}
	}

	var parsedEvent Event
	err := json.Unmarshal([]byte(dataLine), &parsedEvent)
	assert.NoError(t, err)
	assert.Equal(t, EventComplete, parsedEvent.Type)
	assert.Equal(t, "session123", parsedEvent.SessionID)
	assert.Equal(t, "agent456", parsedEvent.AgentID)
}

func TestEventToSSE_MarshalError(t *testing.T) {
	// 创建一个无法序列化的数据
	// Create data that cannot be serialized
	event := &Event{
		Type:      EventError,
		Timestamp: time.Now(),
		Data:      make(chan int), // channels 不能被 JSON 序列化
	}

	sse := event.ToSSE()

	// 应该返回错误事件
	// Should return error event
	assert.Contains(t, sse, "event: error")
	assert.Contains(t, sse, "failed to marshal event")
}

func TestNewEventFilter_Empty(t *testing.T) {
	// 空过滤器应该允许所有事件
	// Empty filter should allow all events
	filter := NewEventFilter([]string{})

	event1 := NewEvent(EventRunStart, nil)
	event2 := NewEvent(EventComplete, nil)
	event3 := NewEvent(EventError, nil)

	assert.True(t, filter.ShouldSend(event1))
	assert.True(t, filter.ShouldSend(event2))
	assert.True(t, filter.ShouldSend(event3))
}

func TestNewEventFilter_Specific(t *testing.T) {
	// 指定特定事件类型
	// Specify specific event types
	filter := NewEventFilter([]string{"token", "complete"})

	tokenEvent := NewEvent(EventToken, TokenData{Token: "test", Index: 0})
	completeEvent := NewEvent(EventComplete, CompleteData{Output: "done"})
	errorEvent := NewEvent(EventError, ErrorData{Error: "test error"})
	startEvent := NewEvent(EventRunStart, RunStartData{Input: "test"})

	assert.True(t, filter.ShouldSend(tokenEvent))
	assert.True(t, filter.ShouldSend(completeEvent))
	assert.False(t, filter.ShouldSend(errorEvent))
	assert.False(t, filter.ShouldSend(startEvent))
}

func TestEventFilter_ShouldSend(t *testing.T) {
	tests := []struct {
		name     string
		types    []string
		event    *Event
		expected bool
	}{
		{
			name:     "允许所有事件",
			types:    []string{},
			event:    NewEvent(EventRunStart, nil),
			expected: true,
		},
		{
			name:     "匹配单个类型",
			types:    []string{"token"},
			event:    NewEvent(EventToken, nil),
			expected: true,
		},
		{
			name:     "不匹配类型",
			types:    []string{"token"},
			event:    NewEvent(EventError, nil),
			expected: false,
		},
		{
			name:     "匹配多个类型之一",
			types:    []string{"token", "complete", "error"},
			event:    NewEvent(EventComplete, nil),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewEventFilter(tt.types)
			result := filter.ShouldSend(tt.event)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildReasoningEvents(t *testing.T) {
	messages := []*types.Message{
		{
			Role:             types.RoleAssistant,
			Content:          "answer",
			ReasoningContent: types.NewReasoningContent("step 1").WithTokenCount(42),
		},
		{
			Role:             types.RoleAssistant,
			Content:          "final answer",
			ReasoningContent: types.NewReasoningContent("  ").WithTokenCount(10),
		},
	}

	events, summary := buildReasoningEvents(messages, "provider", "model-x")
	assert.Len(t, events, 1)
	assert.NotNil(t, summary)
	assert.Equal(t, "provider", summary.Provider)
	assert.Equal(t, "model-x", summary.Model)
	if summary.TokenCount == nil {
		t.Fatal("expected summary to include token count")
	}
	assert.Equal(t, 42, *summary.TokenCount)

	reasoningEvent := events[0]
	assert.Equal(t, EventReasoning, reasoningEvent.Type)

	data, ok := reasoningEvent.Data.(ReasoningData)
	if !ok {
		t.Fatalf("expected ReasoningData, got %T", reasoningEvent.Data)
	}
	assert.Equal(t, "step 1", data.Content)
	assert.Equal(t, 0, data.MessageIndex)
	assert.Equal(t, "provider", data.Provider)
	assert.Equal(t, "model-x", data.Model)
	if data.TokenCount == nil {
		t.Fatal("expected token count in reasoning data")
	}
	assert.Equal(t, 42, *data.TokenCount)
}

func TestBuildUsageSummary(t *testing.T) {
	usage := types.Usage{
		PromptTokens:     12,
		CompletionTokens: 34,
		TotalTokens:      46,
	}

	metadata := map[string]interface{}{
		"usage": usage,
	}

	summary := buildUsageSummary(metadata)
	if assert.NotNil(t, summary) {
		assert.Equal(t, 12, summary.PromptTokens)
		assert.Equal(t, 34, summary.CompletionTokens)
		assert.Equal(t, 46, summary.TotalTokens)
	}

	metadata["usage"] = &usage
	summary = buildUsageSummary(metadata)
	assert.NotNil(t, summary)

	metadata["usage"] = "invalid"
	summary = buildUsageSummary(metadata)
	assert.Nil(t, summary)
}

func TestEventDataStructures(t *testing.T) {
	// 测试各种事件数据结构
	// Test various event data structures

	t.Run("RunStartData", func(t *testing.T) {
		data := RunStartData{
			Input:     "test input",
			SessionID: "session123",
			Media: []media.Attachment{
				{Type: "image", URL: "https://example.com/image.png"},
			},
		}
		assert.Equal(t, "test input", data.Input)
		assert.Equal(t, "session123", data.SessionID)
		assert.Len(t, data.Media, 1)
	})

	t.Run("ToolCallData", func(t *testing.T) {
		data := ToolCallData{
			ToolName:  "calculator",
			Arguments: map[string]interface{}{"a": 1, "b": 2},
			Result:    3,
		}
		assert.Equal(t, "calculator", data.ToolName)
		assert.Equal(t, 1, data.Arguments["a"])
		assert.Equal(t, 3, data.Result)
	})

	t.Run("TokenData", func(t *testing.T) {
		data := TokenData{
			Token: "Hello",
			Index: 0,
		}
		assert.Equal(t, "Hello", data.Token)
		assert.Equal(t, 0, data.Index)
	})

	t.Run("StepData", func(t *testing.T) {
		data := StepData{
			StepName:    "step1",
			StepIndex:   0,
			Description: "First step",
		}
		assert.Equal(t, "step1", data.StepName)
		assert.Equal(t, 0, data.StepIndex)
		assert.Equal(t, "First step", data.Description)
	})

	t.Run("ErrorData", func(t *testing.T) {
		data := ErrorData{
			Error:   "something went wrong",
			Code:    "ERR_001",
			Details: map[string]string{"key": "value"},
		}
		assert.Equal(t, "something went wrong", data.Error)
		assert.Equal(t, "ERR_001", data.Code)
		assert.NotNil(t, data.Details)
	})

	t.Run("CompleteData", func(t *testing.T) {
		data := CompleteData{
			Output:     "result",
			Duration:   1.5,
			TokenCount: 100,
		}
		assert.Equal(t, "result", data.Output)
		assert.Equal(t, 1.5, data.Duration)
		assert.Equal(t, 100, data.TokenCount)
	})
}

func TestEventWithSessionAndAgent(t *testing.T) {
	event := NewEvent(EventRunStart, RunStartData{Input: "test"})
	event.SessionID = "session789"
	event.AgentID = "agent123"

	assert.Equal(t, "session789", event.SessionID)
	assert.Equal(t, "agent123", event.AgentID)

	// 验证 SSE 输出包含这些字段
	// Verify SSE output includes these fields
	sse := event.ToSSE()
	assert.Contains(t, sse, "session789")
	assert.Contains(t, sse, "agent123")
}
