package elevenlabs

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenerateSpeech(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/text-to-speech/voice-123" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("xi-api-key") != "api-key" {
			t.Fatalf("missing xi-api-key header")
		}
		audio := []byte{0x00, 0x01, 0x02}
		_, _ = w.Write(audio)
	}))
	defer server.Close()

	tk, err := New(Config{
		APIKey:  "api-key",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	resp, err := tk.Execute(context.Background(), "generate_speech", map[string]interface{}{
		"voice_id":         "voice-123",
		"text":             "Hello world",
		"stability":        0.5,
		"similarity_boost": 0.7,
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	data := resp.(map[string]interface{})
	audioB64 := data["audio_b64"].(string)
	audio, err := base64.StdEncoding.DecodeString(audioB64)
	if err != nil {
		t.Fatalf("invalid base64: %v", err)
	}
	if len(audio) != 3 {
		t.Fatalf("unexpected audio length %d", len(audio))
	}
}
