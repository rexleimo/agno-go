package video

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	evolinkp "github.com/rexleimo/agno-go/pkg/hno/models/evolink"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// Config for Evolink video model
type Config struct {
	APIKey     string
	BaseURL    string
	Timeout    time.Duration
	HTTPClient *http.Client
	Model      Model

	// Defaults
	AspectRatio     string
	DurationSeconds int
	RemoveWatermark *bool
	Quality         string
}

// Video implements models.Model for EvoLink video generations
type Video struct {
	models.BaseModel
	client *evolinkp.Client
	config Config
}

// New creates a new Evolink video model
func New(modelID string, cfg Config) (*Video, error) {
	c, err := evolinkp.NewClient(evolinkp.Config{APIKey: cfg.APIKey, BaseURL: cfg.BaseURL, Timeout: cfg.Timeout, HTTPClient: cfg.HTTPClient})
	if err != nil {
		return nil, err
	}
	if cfg.Model == "" {
		cfg.Model = ModelSora2
	}
	return &Video{BaseModel: models.BaseModel{ID: modelID, Provider: "evolink"}, client: c, config: cfg}, nil
}

// Invoke triggers a video generation task
func (v *Video) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	prompt := lastUserPrompt(req)
	model := modelFromExtra(req, "model", v.config.Model)
	ar := stringFromExtra(req, "aspect_ratio", v.config.AspectRatio)
	dur := intFromExtra(req, "duration", v.config.DurationSeconds)
	imageURLs := stringSliceFromExtra(req, "image_urls")
	cb := stringFromExtra(req, "callback_url", "")
	quality := stringFromExtra(req, "quality", v.config.Quality)
	rmw := boolPointerFromExtra(req, "remove_watermark")
	if rmw == nil && v.config.RemoveWatermark != nil {
		rmw = v.config.RemoveWatermark
	}
	resp, err := Generate(ctx, v.client, Options{
		Model:           Model(model),
		Prompt:          prompt,
		AspectRatio:     ar,
		DurationSeconds: dur,
		ImageURLs:       imageURLs,
		RemoveWatermark: rmw,
		Quality:         quality,
		CallbackURL:     cb,
	})
	if err != nil {
		return nil, err
	}
	return &types.ModelResponse{
		ID:       resp.TaskID,
		Content:  "video task completed",
		Model:    v.ID,
		Metadata: types.Metadata{Extra: map[string]interface{}{"status": resp.Status, "data": resp.Data}},
	}, nil
}

// InvokeStream is not supported for video tasks
func (v *Video) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	return nil, types.NewAPIError("streaming not supported", nil)
}

func lastUserPrompt(req *models.InvokeRequest) string {
	if req == nil || len(req.Messages) == 0 {
		return ""
	}
	for i := len(req.Messages) - 1; i >= 0; i-- {
		m := req.Messages[i]
		if string(m.Role) == string(types.RoleUser) && strings.TrimSpace(m.Content) != "" {
			return m.Content
		}
	}
	return req.Messages[len(req.Messages)-1].Content
}

func stringFromExtra(req *models.InvokeRequest, key, def string) string {
	if req != nil && req.Extra != nil {
		if v, ok := req.Extra[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return def
}
func intFromExtra(req *models.InvokeRequest, key string, def int) int {
	if req != nil && req.Extra != nil {
		if v, ok := req.Extra[key]; ok {
			switch t := v.(type) {
			case int:
				if t > 0 {
					return t
				}
			case float64:
				if int(t) > 0 {
					return int(t)
				}
			}
		}
	}
	return def
}
func stringSliceFromExtra(req *models.InvokeRequest, key string) []string {
	if req != nil && req.Extra != nil {
		if v, ok := req.Extra[key]; ok {
			switch vv := v.(type) {
			case []string:
				return vv
			case []interface{}:
				var out []string
				for _, it := range vv {
					if s, ok := it.(string); ok && s != "" {
						out = append(out, s)
					}
				}
				return out
			case string:
				if vv != "" {
					return []string{vv}
				}
			}
		}
	}
	return nil
}

func boolPointerFromExtra(req *models.InvokeRequest, key string) *bool {
	if req != nil && req.Extra != nil {
		if v, ok := req.Extra[key]; ok {
			if b, ok := v.(bool); ok {
				return &b
			}
		}
	}
	return nil
}

func modelFromExtra(req *models.InvokeRequest, key string, def Model) Model {
	if req != nil && req.Extra != nil {
		if v, ok := req.Extra[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return Model(s)
			}
		}
	}
	return def
}
