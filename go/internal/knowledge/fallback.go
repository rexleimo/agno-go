package knowledge

import (
	"errors"
	"fmt"
)

// Mode describes how to respond when knowledge retrieval is unavailable.
type Mode string

const (
	// ModeHint returns a lightweight hint to the user when retrieval fails.
	ModeHint Mode = "hint"
	// ModeError surfaces an explicit error response.
	ModeError Mode = "error"
)

// Strategy encapsulates fallback configuration.
type Strategy struct {
	Mode        Mode
	HintMessage string
	ErrorPrefix string
}

var (
	// ErrUnavailable signals that retrieval could not proceed.
	ErrUnavailable = errors.New("knowledge unavailable")
)

// HandleUnavailable returns a user-facing fallback string or an error depending on strategy.
func HandleUnavailable(strategy Strategy, reason string) (string, error) {
	switch strategy.Mode {
	case ModeHint:
		msg := strategy.HintMessage
		if msg == "" {
			msg = "Knowledge source unavailable; returning best-effort response."
		}
		if reason != "" {
			return fmt.Sprintf("%s (%s)", msg, reason), nil
		}
		return msg, nil
	default:
		prefix := strategy.ErrorPrefix
		if prefix == "" {
			prefix = "Knowledge retrieval failed"
		}
		if reason != "" {
			return "", fmt.Errorf("%w: %s: %s", ErrUnavailable, prefix, reason)
		}
		return "", fmt.Errorf("%w: %s", ErrUnavailable, prefix)
	}
}
