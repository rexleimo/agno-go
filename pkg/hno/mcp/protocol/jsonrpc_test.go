package protocol

import (
	"encoding/json"
	"testing"
)

func TestNewRequest(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		params     interface{}
		id         interface{}
		wantMethod string
		wantErr    bool
	}{
		{
			name:       "valid request with params",
			method:     "tools/list",
			params:     map[string]string{"cursor": "abc"},
			id:         1,
			wantMethod: "tools/list",
			wantErr:    false,
		},
		{
			name:       "valid request without params",
			method:     "initialize",
			params:     nil,
			id:         "req-1",
			wantMethod: "initialize",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := NewRequest(tt.method, tt.params, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if req.JSONRPC != JSONRPCVersion {
					t.Errorf("Expected JSONRPC version %s, got %s", JSONRPCVersion, req.JSONRPC)
				}
				if req.Method != tt.wantMethod {
					t.Errorf("Expected method %s, got %s", tt.wantMethod, req.Method)
				}
				if req.ID != tt.id {
					t.Errorf("Expected ID %v, got %v", tt.id, req.ID)
				}

				// Verify JSON serialization
				data, err := json.Marshal(req)
				if err != nil {
					t.Errorf("Failed to marshal request: %v", err)
				}

				var decoded JSONRPCRequest
				if err := json.Unmarshal(data, &decoded); err != nil {
					t.Errorf("Failed to unmarshal request: %v", err)
				}
			}
		})
	}
}

func TestNewResponse(t *testing.T) {
	tests := []struct {
		name    string
		result  interface{}
		id      interface{}
		wantErr bool
	}{
		{
			name:    "valid response with result",
			result:  map[string]string{"status": "ok"},
			id:      1,
			wantErr: false,
		},
		{
			name:    "valid response without result",
			result:  nil,
			id:      2,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := NewResponse(tt.result, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if resp.JSONRPC != JSONRPCVersion {
					t.Errorf("Expected JSONRPC version %s, got %s", JSONRPCVersion, resp.JSONRPC)
				}
				if resp.ID != tt.id {
					t.Errorf("Expected ID %v, got %v", tt.id, resp.ID)
				}
				if resp.Error != nil {
					t.Errorf("Expected no error, got %v", resp.Error)
				}

				// Verify JSON serialization
				data, err := json.Marshal(resp)
				if err != nil {
					t.Errorf("Failed to marshal response: %v", err)
				}

				var decoded JSONRPCResponse
				if err := json.Unmarshal(data, &decoded); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
				}
			}
		})
	}
}

func TestNewErrorResponse(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		message  string
		data     interface{}
		id       interface{}
		wantCode int
		wantErr  bool
	}{
		{
			name:     "parse error",
			code:     ErrorCodeParseError,
			message:  "Parse error",
			data:     nil,
			id:       nil,
			wantCode: ErrorCodeParseError,
			wantErr:  false,
		},
		{
			name:     "method not found with data",
			code:     ErrorCodeMethodNotFound,
			message:  "Method not found",
			data:     map[string]string{"method": "unknown"},
			id:       1,
			wantCode: ErrorCodeMethodNotFound,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := NewErrorResponse(tt.code, tt.message, tt.data, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewErrorResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if resp.JSONRPC != JSONRPCVersion {
					t.Errorf("Expected JSONRPC version %s, got %s", JSONRPCVersion, resp.JSONRPC)
				}
				if resp.Error == nil {
					t.Fatal("Expected error object, got nil")
				}
				if resp.Error.Code != tt.wantCode {
					t.Errorf("Expected error code %d, got %d", tt.wantCode, resp.Error.Code)
				}
				if resp.Error.Message != tt.message {
					t.Errorf("Expected error message %s, got %s", tt.message, resp.Error.Message)
				}

				// Verify JSON serialization
				data, err := json.Marshal(resp)
				if err != nil {
					t.Errorf("Failed to marshal error response: %v", err)
				}

				var decoded JSONRPCResponse
				if err := json.Unmarshal(data, &decoded); err != nil {
					t.Errorf("Failed to unmarshal error response: %v", err)
				}
			}
		})
	}
}

func TestNewNotification(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		params     interface{}
		wantMethod string
		wantErr    bool
	}{
		{
			name:       "valid notification with params",
			method:     "logging/message",
			params:     map[string]string{"level": "info", "message": "test"},
			wantMethod: "logging/message",
			wantErr:    false,
		},
		{
			name:       "valid notification without params",
			method:     "initialized",
			params:     nil,
			wantMethod: "initialized",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notif, err := NewNotification(tt.method, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNotification() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if notif.JSONRPC != JSONRPCVersion {
					t.Errorf("Expected JSONRPC version %s, got %s", JSONRPCVersion, notif.JSONRPC)
				}
				if notif.Method != tt.wantMethod {
					t.Errorf("Expected method %s, got %s", tt.wantMethod, notif.Method)
				}

				// Verify JSON serialization
				data, err := json.Marshal(notif)
				if err != nil {
					t.Errorf("Failed to marshal notification: %v", err)
				}

				var decoded JSONRPCNotification
				if err := json.Unmarshal(data, &decoded); err != nil {
					t.Errorf("Failed to unmarshal notification: %v", err)
				}

				// Ensure no ID field in serialized JSON
				var raw map[string]interface{}
				if err := json.Unmarshal(data, &raw); err != nil {
					t.Errorf("Failed to unmarshal raw JSON: %v", err)
				}
				if _, hasID := raw["id"]; hasID {
					t.Error("Notification should not have an ID field")
				}
			}
		})
	}
}

func TestErrorCodes(t *testing.T) {
	tests := []struct {
		code int
		name string
	}{
		{ErrorCodeParseError, "Parse Error"},
		{ErrorCodeInvalidRequest, "Invalid Request"},
		{ErrorCodeMethodNotFound, "Method Not Found"},
		{ErrorCodeInvalidParams, "Invalid Params"},
		{ErrorCodeInternalError, "Internal Error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify error code is in valid range
			if tt.code > -32000 || tt.code < -32768 {
				t.Errorf("Error code %d is outside standard range [-32768, -32000]", tt.code)
			}
		})
	}
}
