package client

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/rexleimo/agno-go/pkg/hno/mcp/protocol"
)

// MockTransport is a mock implementation of Transport for testing
// It is exported for use in other packages' tests
// MockTransport 是用于测试的 Transport 模拟实现
// 导出供其他包的测试使用
type MockTransport struct {
	mu            sync.RWMutex
	running       bool
	responses     map[int64]*protocol.JSONRPCResponse
	startError    error
	sendError     error
	notifications []*protocol.JSONRPCNotification
	sendFunc      func(context.Context, *protocol.JSONRPCRequest) (*protocol.JSONRPCResponse, error)
	startCount    int
}

// NewMockTransport creates a new mock transport for testing
// NewMockTransport 创建新的模拟传输用于测试
func NewMockTransport() *MockTransport {
	return &MockTransport{
		responses:     make(map[int64]*protocol.JSONRPCResponse),
		notifications: make([]*protocol.JSONRPCNotification, 0),
	}
}

func (m *MockTransport) Start(ctx context.Context) error {
	if m.startError != nil {
		return m.startError
	}
	m.mu.Lock()
	m.running = true
	m.startCount++
	m.mu.Unlock()
	return nil
}

func (m *MockTransport) Stop() error {
	m.mu.Lock()
	m.running = false
	m.mu.Unlock()
	return nil
}

func (m *MockTransport) Send(ctx context.Context, req *protocol.JSONRPCRequest) (*protocol.JSONRPCResponse, error) {
	if m.sendError != nil {
		return nil, m.sendError
	}

	m.mu.RLock()
	sendHook := m.sendFunc
	m.mu.RUnlock()

	if sendHook != nil {
		return sendHook(ctx, req)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if we have a mock response for this request ID
	// 检查是否有此请求 ID 的模拟响应
	if resp, ok := m.responses[req.ID.(int64)]; ok {
		return resp, nil
	}

	// Default successful response
	// 默认成功响应
	return &protocol.JSONRPCResponse{
		JSONRPC: protocol.JSONRPCVersion,
		Result:  json.RawMessage(`{}`),
		ID:      req.ID,
	}, nil
}

func (m *MockTransport) SendNotification(ctx context.Context, notif *protocol.JSONRPCNotification) error {
	m.mu.Lock()
	m.notifications = append(m.notifications, notif)
	m.mu.Unlock()
	return nil
}

func (m *MockTransport) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// SetResponse sets a mock response for a given request ID
// SetResponse 为给定的请求 ID 设置模拟响应
func (m *MockTransport) SetResponse(id int64, resp *protocol.JSONRPCResponse) {
	m.mu.Lock()
	m.responses[id] = resp
	m.mu.Unlock()
}

// SetSendFunc sets a custom function to handle Send calls.
func (m *MockTransport) SetSendFunc(fn func(context.Context, *protocol.JSONRPCRequest) (*protocol.JSONRPCResponse, error)) {
	m.mu.Lock()
	m.sendFunc = fn
	m.mu.Unlock()
}

// StartCount returns how many times Start was called.
func (m *MockTransport) StartCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.startCount
}
