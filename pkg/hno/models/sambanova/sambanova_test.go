package sambanova

import (
	"testing"
)

func TestNew_MissingAPIKey(t *testing.T) {
	_, err := New("Meta-Llama-3.1-70B-Instruct", Config{})
	if err == nil {
		t.Fatalf("expected error for missing API key")
	}
}
