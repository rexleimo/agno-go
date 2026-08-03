package internlm

import (
	"testing"
)

func TestNew_RequiresBaseURLAndKey(t *testing.T) {
	if _, err := New("internlm2.5", Config{}); err == nil {
		t.Fatalf("expected error for missing key/baseURL")
	}
	if _, err := New("internlm2.5", Config{APIKey: "k"}); err == nil {
		t.Fatalf("expected error for missing baseURL")
	}
}
