package elevenlabs

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

const (
	defaultBaseURL = "https://api.elevenlabs.io"
	defaultTimeout = 45 * time.Second
)

// Config 控制 ElevenLabs 工具包。
type Config struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
	Timeout    time.Duration
}

// Toolkit 提供语音合成功能。
type Toolkit struct {
	*toolkit.BaseToolkit
	client *client
}

type client struct {
	base string
	key  string
	http *http.Client
}

type speechRequest struct {
	Text                     string                 `json:"text"`
	ModelID                  string                 `json:"model_id,omitempty"`
	VoiceSettings            map[string]interface{} `json:"voice_settings,omitempty"`
	OptimiseStreamingLatency int                    `json:"optimize_streaming_latency,omitempty"`
}

// New 创建 ElevenLabs 工具包。
func New(cfg Config) (*Toolkit, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("elevenlabs api key is required")
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	if cfg.Timeout > 0 {
		httpClient.Timeout = cfg.Timeout
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	c := &client{
		base: strings.TrimRight(baseURL, "/"),
		key:  cfg.APIKey,
		http: httpClient,
	}

	tk := &Toolkit{
		BaseToolkit: toolkit.NewBaseToolkit("elevenlabs"),
		client:      c,
	}

	tk.RegisterFunction(&toolkit.Function{
		Name:        "generate_speech",
		Description: "Generate speech audio from text using ElevenLabs",
		Parameters: map[string]toolkit.Parameter{
			"voice_id": {
				Type:        "string",
				Description: "Target voice identifier",
				Required:    true,
			},
			"text": {
				Type:        "string",
				Description: "Text input to synthesise",
				Required:    true,
			},
			"model_id": {
				Type:        "string",
				Description: "Optional model identifier",
			},
			"stability": {
				Type:        "number",
				Description: "Optional stability parameter (0.0 - 1.0)",
			},
			"similarity_boost": {
				Type:        "number",
				Description: "Optional similarity boost parameter (0.0 - 1.0)",
			},
		},
		Handler: tk.generateSpeech,
	})

	return tk, nil
}

func (t *Toolkit) generateSpeech(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	voiceID, ok := args["voice_id"].(string)
	if !ok || voiceID == "" {
		return nil, fmt.Errorf("voice_id is required")
	}

	text, ok := args["text"].(string)
	if !ok || strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("text must be provided")
	}

	req := speechRequest{
		Text: text,
	}

	if modelID, ok := args["model_id"].(string); ok && modelID != "" {
		req.ModelID = modelID
	}

	settings := make(map[string]interface{})
	if stability, ok := args["stability"].(float64); ok {
		settings["stability"] = stability
	}
	if similarity, ok := args["similarity_boost"].(float64); ok {
		settings["similarity_boost"] = similarity
	}
	if len(settings) > 0 {
		req.VoiceSettings = settings
	}

	audio, err := t.client.generate(ctx, voiceID, req)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"voice_id":       voiceID,
		"audio_b64":      audio,
		"length_seconds": len(audio) / 4, // rough indicator
	}, nil
}

func (c *client) generate(ctx context.Context, voiceID string, payload speechRequest) (string, error) {
	url := fmt.Sprintf("%s/v1/text-to-speech/%s", c.base, voiceID)
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to encode speech payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", c.key)
	req.Header.Set("Accept", "audio/mpeg")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("elevenlabs request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("elevenlabs request returned status %d", resp.StatusCode)
	}

	audioBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read audio stream: %w", err)
	}

	return base64.StdEncoding.EncodeToString(audioBytes), nil
}
