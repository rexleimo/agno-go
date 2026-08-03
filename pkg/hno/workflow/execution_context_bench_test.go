package workflow

import (
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// BenchmarkExecutionContext_GetWorkflowHistory benchmarks GetWorkflowHistory
func BenchmarkExecutionContext_GetWorkflowHistory(b *testing.B) {
	execCtx := NewExecutionContext("test")
	execCtx.SetWorkflowHistory([]HistoryEntry{
		{Input: "input-1", Output: "output-1", Timestamp: time.Now()},
		{Input: "input-2", Output: "output-2", Timestamp: time.Now()},
	})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = execCtx.GetWorkflowHistory()
	}
}

// BenchmarkExecutionContext_SetWorkflowHistory benchmarks SetWorkflowHistory
func BenchmarkExecutionContext_SetWorkflowHistory(b *testing.B) {
	execCtx := NewExecutionContext("test")
	history := []HistoryEntry{
		{Input: "input-1", Output: "output-1", Timestamp: time.Now()},
		{Input: "input-2", Output: "output-2", Timestamp: time.Now()},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		execCtx.SetWorkflowHistory(history)
	}
}

// BenchmarkExecutionContext_HasHistory benchmarks HasHistory
func BenchmarkExecutionContext_HasHistory(b *testing.B) {
	execCtx := NewExecutionContext("test")
	execCtx.SetWorkflowHistory([]HistoryEntry{
		{Input: "input-1", Output: "output-1", Timestamp: time.Now()},
	})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = execCtx.HasHistory()
	}
}

// BenchmarkExecutionContext_GetHistoryCount benchmarks GetHistoryCount
func BenchmarkExecutionContext_GetHistoryCount(b *testing.B) {
	execCtx := NewExecutionContext("test")
	execCtx.SetWorkflowHistory([]HistoryEntry{
		{Input: "input-1", Output: "output-1", Timestamp: time.Now()},
		{Input: "input-2", Output: "output-2", Timestamp: time.Now()},
	})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = execCtx.GetHistoryCount()
	}
}

// BenchmarkExecutionContext_GetLastHistoryEntry benchmarks GetLastHistoryEntry
func BenchmarkExecutionContext_GetLastHistoryEntry(b *testing.B) {
	execCtx := NewExecutionContext("test")
	execCtx.SetWorkflowHistory([]HistoryEntry{
		{Input: "input-1", Output: "output-1", Timestamp: time.Now()},
		{Input: "input-2", Output: "output-2", Timestamp: time.Now()},
	})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = execCtx.GetLastHistoryEntry()
	}
}

// BenchmarkExecutionContext_GetHistoryInput benchmarks GetHistoryInput
func BenchmarkExecutionContext_GetHistoryInput(b *testing.B) {
	execCtx := NewExecutionContext("test")
	execCtx.SetWorkflowHistory([]HistoryEntry{
		{Input: "input-1", Output: "output-1", Timestamp: time.Now()},
		{Input: "input-2", Output: "output-2", Timestamp: time.Now()},
		{Input: "input-3", Output: "output-3", Timestamp: time.Now()},
	})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = execCtx.GetHistoryInput(1)
	}
}

// BenchmarkExecutionContext_GetHistoryInputNegative benchmarks GetHistoryInput with negative index
func BenchmarkExecutionContext_GetHistoryInputNegative(b *testing.B) {
	execCtx := NewExecutionContext("test")
	execCtx.SetWorkflowHistory([]HistoryEntry{
		{Input: "input-1", Output: "output-1", Timestamp: time.Now()},
		{Input: "input-2", Output: "output-2", Timestamp: time.Now()},
		{Input: "input-3", Output: "output-3", Timestamp: time.Now()},
	})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = execCtx.GetHistoryInput(-1)
	}
}

// BenchmarkExecutionContext_GetHistoryOutput benchmarks GetHistoryOutput
func BenchmarkExecutionContext_GetHistoryOutput(b *testing.B) {
	execCtx := NewExecutionContext("test")
	execCtx.SetWorkflowHistory([]HistoryEntry{
		{Input: "input-1", Output: "output-1", Timestamp: time.Now()},
		{Input: "input-2", Output: "output-2", Timestamp: time.Now()},
		{Input: "input-3", Output: "output-3", Timestamp: time.Now()},
	})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = execCtx.GetHistoryOutput(1)
	}
}

// BenchmarkExecutionContext_GetHistoryContext benchmarks GetHistoryContext
func BenchmarkExecutionContext_GetHistoryContext(b *testing.B) {
	execCtx := NewExecutionContext("test")
	execCtx.SetHistoryContext("<workflow_history_context>test</workflow_history_context>")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = execCtx.GetHistoryContext()
	}
}

// BenchmarkExecutionContext_SetHistoryContext benchmarks SetHistoryContext
func BenchmarkExecutionContext_SetHistoryContext(b *testing.B) {
	execCtx := NewExecutionContext("test")
	context := "<workflow_history_context>test</workflow_history_context>"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		execCtx.SetHistoryContext(context)
	}
}

// BenchmarkExecutionContext_AddMessage benchmarks AddMessage
func BenchmarkExecutionContext_AddMessage(b *testing.B) {
	execCtx := NewExecutionContext("test")
	msg := types.NewUserMessage("test message")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		execCtx.ClearMessages()
		b.StartTimer()
		execCtx.AddMessage(msg)
	}
}

// BenchmarkExecutionContext_GetMessages benchmarks GetMessages
func BenchmarkExecutionContext_GetMessages(b *testing.B) {
	execCtx := NewExecutionContext("test")
	execCtx.AddMessage(types.NewUserMessage("test message 1"))
	execCtx.AddMessage(types.NewUserMessage("test message 2"))

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = execCtx.GetMessages()
	}
}
