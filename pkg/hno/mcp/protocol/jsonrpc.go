package protocol

import "encoding/json"

// JSONRPCVersion is the JSON-RPC 2.0 version identifier
// JSONRPCVersion 是 JSON-RPC 2.0 版本标识符
const JSONRPCVersion = "2.0"

// JSONRPCRequest represents a JSON-RPC 2.0 request
// JSONRPCRequest 表示 JSON-RPC 2.0 请求
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
// JSONRPCResponse 表示 JSON-RPC 2.0 响应
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
	ID      interface{}     `json:"id"`
}

// JSONRPCError represents a JSON-RPC 2.0 error object
// JSONRPCError 表示 JSON-RPC 2.0 错误对象
type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// JSONRPCNotification represents a JSON-RPC 2.0 notification (no ID)
// JSONRPCNotification 表示 JSON-RPC 2.0 通知（无 ID）
type JSONRPCNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Standard JSON-RPC 2.0 error codes
// 标准 JSON-RPC 2.0 错误码
const (
	ErrorCodeParseError     = -32700 // Invalid JSON was received
	ErrorCodeInvalidRequest = -32600 // The JSON sent is not a valid Request object
	ErrorCodeMethodNotFound = -32601 // The method does not exist / is not available
	ErrorCodeInvalidParams  = -32602 // Invalid method parameter(s)
	ErrorCodeInternalError  = -32603 // Internal JSON-RPC error
)

// NewRequest creates a new JSON-RPC 2.0 request with the given parameters.
// NewRequest 使用给定参数创建新的 JSON-RPC 2.0 请求。
func NewRequest(method string, params interface{}, id interface{}) (*JSONRPCRequest, error) {
	var paramsJSON json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		paramsJSON = data
	}

	return &JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		Method:  method,
		Params:  paramsJSON,
		ID:      id,
	}, nil
}

// NewResponse creates a new JSON-RPC 2.0 response with the given result.
// NewResponse 使用给定结果创建新的 JSON-RPC 2.0 响应。
func NewResponse(result interface{}, id interface{}) (*JSONRPCResponse, error) {
	var resultJSON json.RawMessage
	if result != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return nil, err
		}
		resultJSON = data
	}

	return &JSONRPCResponse{
		JSONRPC: JSONRPCVersion,
		Result:  resultJSON,
		ID:      id,
	}, nil
}

// NewErrorResponse creates a new JSON-RPC 2.0 error response.
// NewErrorResponse 创建新的 JSON-RPC 2.0 错误响应。
func NewErrorResponse(code int, message string, data interface{}, id interface{}) (*JSONRPCResponse, error) {
	var dataJSON json.RawMessage
	if data != nil {
		d, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		dataJSON = d
	}

	return &JSONRPCResponse{
		JSONRPC: JSONRPCVersion,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    dataJSON,
		},
		ID: id,
	}, nil
}

// NewNotification creates a new JSON-RPC 2.0 notification.
// NewNotification 创建新的 JSON-RPC 2.0 通知。
func NewNotification(method string, params interface{}) (*JSONRPCNotification, error) {
	var paramsJSON json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		paramsJSON = data
	}

	return &JSONRPCNotification{
		JSONRPC: JSONRPCVersion,
		Method:  method,
		Params:  paramsJSON,
	}, nil
}
