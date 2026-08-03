package openrouter

import (
	"testing"
)

func TestNew_MissingAPIKey(t *testing.T) {
	_, err := New("openrouter/auto", Config{})
	if err == nil {
		t.Fatalf("expected error for missing API key")
	}
}
