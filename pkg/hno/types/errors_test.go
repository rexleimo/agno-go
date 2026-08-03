package types

import (
	"errors"
	"testing"
)

func TestHnoError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *HnoError
		wantMsg string
	}{
		{
			name: "error without cause",
			err: &HnoError{
				Code:    ErrCodeInvalidInput,
				Message: "invalid parameter",
			},
			wantMsg: "[INVALID_INPUT] invalid parameter",
		},
		{
			name: "error with cause",
			err: &HnoError{
				Code:    ErrCodeAPIError,
				Message: "API call failed",
				Cause:   errors.New("connection timeout"),
			},
			wantMsg: "[API_ERROR] API call failed: connection timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %v, want %v", got, tt.wantMsg)
			}
		})
	}
}

func TestHnoError_Unwrap(t *testing.T) {
	causeErr := errors.New("underlying error")
	err := &HnoError{
		Code:    ErrCodeToolExecution,
		Message: "tool failed",
		Cause:   causeErr,
	}

	if unwrapped := err.Unwrap(); unwrapped != causeErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, causeErr)
	}
}

func TestHnoError_Unwrap_NoCause(t *testing.T) {
	err := &HnoError{
		Code:    ErrCodeInvalidInput,
		Message: "invalid input",
	}

	if unwrapped := err.Unwrap(); unwrapped != nil {
		t.Errorf("Unwrap() = %v, want nil", unwrapped)
	}
}

func TestNewError(t *testing.T) {
	code := ErrCodeModelTimeout
	message := "request timeout"
	cause := errors.New("context deadline exceeded")

	err := NewError(code, message, cause)

	if err.Code != code {
		t.Errorf("Code = %v, want %v", err.Code, code)
	}
	if err.Message != message {
		t.Errorf("Message = %v, want %v", err.Message, message)
	}
	if err.Cause != cause {
		t.Errorf("Cause = %v, want %v", err.Cause, cause)
	}
}

func TestNewModelTimeoutError(t *testing.T) {
	message := "model timeout"
	cause := errors.New("deadline exceeded")

	err := NewModelTimeoutError(message, cause)

	if err.Code != ErrCodeModelTimeout {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeModelTimeout)
	}
	if err.Message != message {
		t.Errorf("Message = %v, want %v", err.Message, message)
	}
	if err.Cause != cause {
		t.Errorf("Cause = %v, want %v", err.Cause, cause)
	}
}

func TestNewToolExecutionError(t *testing.T) {
	message := "tool execution failed"
	cause := errors.New("tool crashed")

	err := NewToolExecutionError(message, cause)

	if err.Code != ErrCodeToolExecution {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeToolExecution)
	}
	if err.Message != message {
		t.Errorf("Message = %v, want %v", err.Message, message)
	}
}

func TestNewInvalidInputError(t *testing.T) {
	message := "invalid input provided"

	err := NewInvalidInputError(message, nil)

	if err.Code != ErrCodeInvalidInput {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeInvalidInput)
	}
	if err.Message != message {
		t.Errorf("Message = %v, want %v", err.Message, message)
	}
	if err.Cause != nil {
		t.Errorf("Cause = %v, want nil", err.Cause)
	}
}

func TestNewInvalidConfigError(t *testing.T) {
	message := "configuration is invalid"
	cause := errors.New("missing API key")

	err := NewInvalidConfigError(message, cause)

	if err.Code != ErrCodeInvalidConfig {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeInvalidConfig)
	}
	if err.Message != message {
		t.Errorf("Message = %v, want %v", err.Message, message)
	}
}

func TestNewAPIError(t *testing.T) {
	message := "API request failed"
	cause := errors.New("HTTP 500")

	err := NewAPIError(message, cause)

	if err.Code != ErrCodeAPIError {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeAPIError)
	}
	if err.Message != message {
		t.Errorf("Message = %v, want %v", err.Message, message)
	}
}

func TestNewRateLimitError(t *testing.T) {
	message := "rate limit exceeded"
	cause := errors.New("too many requests")

	err := NewRateLimitError(message, cause)

	if err.Code != ErrCodeRateLimitError {
		t.Errorf("Code = %v, want %v", err.Code, ErrCodeRateLimitError)
	}
	if err.Message != message {
		t.Errorf("Message = %v, want %v", err.Message, message)
	}
}

func TestErrorCode_Constants(t *testing.T) {
	// Test that all error codes are defined correctly
	codes := []ErrorCode{
		ErrCodeModelTimeout,
		ErrCodeToolExecution,
		ErrCodeInvalidInput,
		ErrCodeInvalidConfig,
		ErrCodeAPIError,
		ErrCodeRateLimitError,
		ErrCodeUnknown,
	}

	for _, code := range codes {
		if code == "" {
			t.Errorf("Error code should not be empty")
		}
	}
}

func TestErrorIs(t *testing.T) {
	originalErr := errors.New("original error")
	agnoErr := NewAPIError("API failed", originalErr)

	// Test that errors.Is works with wrapped errors
	if !errors.Is(agnoErr, originalErr) {
		t.Error("errors.Is should recognize wrapped error")
	}
}
