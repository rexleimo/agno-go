package vercel

import (
	"testing"
)

func TestNew_MissingAPIKey(t *testing.T) {
	_, err := New("gpt-4o-mini", Config{})
	if err == nil {
		t.Fatalf("expected error for missing API key")
	}
}
