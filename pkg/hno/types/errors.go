package types

import "fmt"

// ErrorCode represents different types of errors in the system
type ErrorCode string

const (
	ErrCodeModelTimeout      ErrorCode = "MODEL_TIMEOUT"
	ErrCodeToolExecution     ErrorCode = "TOOL_ERROR"
	ErrCodeInvalidInput      ErrorCode = "INVALID_INPUT"
	ErrCodeInvalidConfig     ErrorCode = "INVALID_CONFIG"
	ErrCodeAPIError          ErrorCode = "API_ERROR"
	ErrCodeRateLimitError    ErrorCode = "RATE_LIMIT"
	ErrCodeInputCheck        ErrorCode = "INPUT_CHECK"
	ErrCodeOutputCheck       ErrorCode = "OUTPUT_CHECK"
	ErrCodePromptInjection   ErrorCode = "PROMPT_INJECTION"
	ErrCodePIIDetected       ErrorCode = "PII_DETECTED"
	ErrCodeContentModeration ErrorCode = "CONTENT_MODERATION"
	ErrCodeCancelled         ErrorCode = "RUN_CANCELLED"
	ErrCodeUnknown           ErrorCode = "UNKNOWN"
)

// HnoError represents a structured error in the Agno system
type HnoError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

// Error implements the error interface
func (e *HnoError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause
func (e *HnoError) Unwrap() error {
	return e.Cause
}

// NewError creates a new HnoError
func NewError(code ErrorCode, message string, cause error) *HnoError {
	return &HnoError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// NewModelTimeoutError creates a model timeout error
func NewModelTimeoutError(message string, cause error) *HnoError {
	return NewError(ErrCodeModelTimeout, message, cause)
}

// NewToolExecutionError creates a tool execution error
func NewToolExecutionError(message string, cause error) *HnoError {
	return NewError(ErrCodeToolExecution, message, cause)
}

// NewInvalidInputError creates an invalid input error
func NewInvalidInputError(message string, cause error) *HnoError {
	return NewError(ErrCodeInvalidInput, message, cause)
}

// NewInvalidConfigError creates an invalid config error
func NewInvalidConfigError(message string, cause error) *HnoError {
	return NewError(ErrCodeInvalidConfig, message, cause)
}

// NewAPIError creates an API error
func NewAPIError(message string, cause error) *HnoError {
	return NewError(ErrCodeAPIError, message, cause)
}

// NewRateLimitError creates a rate limit error
func NewRateLimitError(message string, cause error) *HnoError {
	return NewError(ErrCodeRateLimitError, message, cause)
}

// NewInputCheckError creates an input validation check error
func NewInputCheckError(message string, cause error) *HnoError {
	return NewError(ErrCodeInputCheck, message, cause)
}

// NewOutputCheckError creates an output validation check error
func NewOutputCheckError(message string, cause error) *HnoError {
	return NewError(ErrCodeOutputCheck, message, cause)
}

// NewPromptInjectionError creates a prompt injection detection error
func NewPromptInjectionError(message string, cause error) *HnoError {
	return NewError(ErrCodePromptInjection, message, cause)
}

// NewPIIDetectedError creates a PII detection error
func NewPIIDetectedError(message string, cause error) *HnoError {
	return NewError(ErrCodePIIDetected, message, cause)
}

// NewContentModerationError creates a content moderation error
func NewContentModerationError(message string, cause error) *HnoError {
	return NewError(ErrCodeContentModeration, message, cause)
}

// NewCancellationError creates a cancellation error when a run is cancelled.
func NewCancellationError(message string, cause error) *HnoError {
	return NewError(ErrCodeCancelled, message, cause)
}
