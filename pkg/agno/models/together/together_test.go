package together

import (
	"testing"
)

func TestNew_MissingAPIKey(t *testing.T) {
	_, err := New("meta-llama/Meta-Llama-3.1-8B-Instruct-Turbo", Config{})
	if err == nil {
		t.Fatalf("expected error for missing API key")
	}
}
