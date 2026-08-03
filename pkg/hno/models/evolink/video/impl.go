package video

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	evolink "github.com/rexleimo/agno-go/pkg/hno/models/evolink"
)

// Model enumerates supported EvoLink video engines.
type Model string

const (
	ModelVeo31Fast         Model = "veo3.1-fast"
	ModelSora2             Model = "sora-2"
	ModelSora2Pro          Model = "sora-2-pro"
	ModelWan25TextToVideo  Model = "wan2.5-text-to-video"
	ModelWan25ImageToVideo Model = "wan2.5-image-to-video"
	ModelSeedance10ProFast Model = "doubao-seedance-1.0-pro-fast"
)

var allowedModels = map[Model]struct{}{
	ModelVeo31Fast:         {},
	ModelSora2:             {},
	ModelSora2Pro:          {},
	ModelWan25TextToVideo:  {},
	ModelWan25ImageToVideo: {},
	ModelSeedance10ProFast: {},
}

var (
	limitedAspectRatios = []string{"16:9", "9:16"}
	seedanceTextRatios  = []string{"16:9", "9:16", "1:1", "4:3", "3:4", "21:9"}
	seedanceImageRatios = append(append([]string{}, seedanceTextRatios...), "keep_ratio", "adaptive")
	wanDurations        = []int{5, 10}
	soraDurations       = []int{10, 15}
)

// Options defines EvoLink video generation parameters.
type Options struct {
	Model           Model
	Prompt          string
	AspectRatio     string
	DurationSeconds int
	ImageURLs       []string
	RemoveWatermark *bool
	Quality         string
	CallbackURL     string
}

func (o *Options) validate() error {
	if _, ok := allowedModels[o.Model]; !ok {
		if o.Model == "" {
			return fmt.Errorf("model is required")
		}
		return fmt.Errorf("unsupported model: %s", o.Model)
	}
	if strings.TrimSpace(o.Prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	for _, img := range o.ImageURLs {
		if _, err := url.ParseRequestURI(img); err != nil {
			return fmt.Errorf("invalid image url: %w", err)
		}
	}
	if o.CallbackURL != "" {
		u, err := url.ParseRequestURI(o.CallbackURL)
		if err != nil {
			return fmt.Errorf("invalid callback url: %w", err)
		}
		if u.Scheme != "https" {
			return fmt.Errorf("callback url must be https")
		}
	}
	return o.applyModelDefaults()
}

// Response is a minimal video task completion payload
type Response struct {
	TaskID string                 `json:"task_id"`
	Status string                 `json:"status"`
	Data   map[string]interface{} `json:"data"`
}

// Generate creates a video generation task and polls until completion
func Generate(ctx context.Context, c *evolink.Client, opts Options) (*Response, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}
	payload := opts.buildPayload()

	var createResp struct {
		TaskID string `json:"task_id"`
	}
	if err := c.PostJSON(ctx, "/v1/videos/generations", payload, &createResp); err != nil {
		return nil, err
	}
	tr, err := c.PollTask(ctx, createResp.TaskID, 2*time.Second)
	if err != nil {
		return nil, err
	}
	return &Response{TaskID: tr.ID, Status: tr.Status, Data: tr.Data}, nil
}

func (o *Options) applyModelDefaults() error {
	switch o.Model {
	case ModelSora2, ModelSora2Pro:
		if o.AspectRatio == "" {
			o.AspectRatio = "16:9"
		}
		if err := ensureStringInSet(o.AspectRatio, limitedAspectRatios, "aspect_ratio", o.Model); err != nil {
			return err
		}
		if o.DurationSeconds == 0 {
			o.DurationSeconds = 10
		}
		if err := ensureIntInSet(o.DurationSeconds, soraDurations, "duration", o.Model); err != nil {
			return err
		}
		if len(o.ImageURLs) > 1 {
			return fmt.Errorf("%s supports at most one image url", o.Model)
		}
		if o.RemoveWatermark == nil {
			b := true
			o.RemoveWatermark = &b
		}
		if o.Model == ModelSora2Pro {
			if o.Quality == "" {
				o.Quality = "standard"
			}
			if err := ensureStringInSet(o.Quality, []string{"standard", "high"}, "quality", o.Model); err != nil {
				return err
			}
		} else if o.Quality != "" {
			return fmt.Errorf("quality is not supported for %s", o.Model)
		}
	case ModelVeo31Fast:
		if o.AspectRatio == "" {
			o.AspectRatio = "16:9"
		}
		if err := ensureStringInSet(o.AspectRatio, limitedAspectRatios, "aspect_ratio", o.Model); err != nil {
			return err
		}
		if len(o.ImageURLs) > 2 {
			return fmt.Errorf("%s supports up to two image urls", o.Model)
		}
		if o.DurationSeconds != 0 {
			return fmt.Errorf("duration is not supported for %s", o.Model)
		}
		if o.RemoveWatermark != nil {
			return fmt.Errorf("remove_watermark is not supported for %s", o.Model)
		}
		if o.Quality != "" {
			return fmt.Errorf("quality is not supported for %s", o.Model)
		}
	case ModelWan25TextToVideo:
		if len(o.ImageURLs) > 0 {
			return fmt.Errorf("image_urls are not supported for %s", o.Model)
		}
		if o.AspectRatio == "" {
			o.AspectRatio = "16:9"
		}
		if err := ensureStringInSet(o.AspectRatio, limitedAspectRatios, "aspect_ratio", o.Model); err != nil {
			return err
		}
		if o.DurationSeconds == 0 {
			o.DurationSeconds = 5
		}
		if err := ensureIntInSet(o.DurationSeconds, wanDurations, "duration", o.Model); err != nil {
			return err
		}
		if o.RemoveWatermark != nil {
			return fmt.Errorf("remove_watermark is not supported for %s", o.Model)
		}
		if o.Quality != "" {
			return fmt.Errorf("quality is not supported for %s", o.Model)
		}
	case ModelWan25ImageToVideo:
		if len(o.ImageURLs) != 1 {
			return fmt.Errorf("%s requires exactly one image url", o.Model)
		}
		if o.AspectRatio != "" {
			return fmt.Errorf("aspect_ratio is not supported for %s", o.Model)
		}
		if o.DurationSeconds == 0 {
			o.DurationSeconds = 5
		}
		if err := ensureIntInSet(o.DurationSeconds, wanDurations, "duration", o.Model); err != nil {
			return err
		}
		if o.RemoveWatermark != nil {
			return fmt.Errorf("remove_watermark is not supported for %s", o.Model)
		}
		if o.Quality != "" {
			return fmt.Errorf("quality is not supported for %s", o.Model)
		}
	case ModelSeedance10ProFast:
		if len(o.ImageURLs) > 1 {
			return fmt.Errorf("%s supports at most one image url", o.Model)
		}
		if o.DurationSeconds == 0 {
			o.DurationSeconds = 5
		}
		if o.DurationSeconds < 2 || o.DurationSeconds > 12 {
			return fmt.Errorf("duration must be between 2 and 12 seconds for %s", o.Model)
		}
		if o.Quality == "" {
			o.Quality = "1080p"
		}
		if err := ensureStringInSet(o.Quality, []string{"720p", "1080p"}, "quality", o.Model); err != nil {
			return err
		}
		allowed := seedanceTextRatios
		if len(o.ImageURLs) > 0 {
			allowed = seedanceImageRatios
			if o.AspectRatio == "" {
				o.AspectRatio = "adaptive"
			}
		} else if o.AspectRatio == "" {
			o.AspectRatio = "16:9"
		}
		if err := ensureStringInSet(o.AspectRatio, allowed, "aspect_ratio", o.Model); err != nil {
			return err
		}
		if o.RemoveWatermark != nil {
			return fmt.Errorf("remove_watermark is not supported for %s", o.Model)
		}
	default:
		return fmt.Errorf("model %s not handled", o.Model)
	}
	return nil
}

func (o Options) buildPayload() map[string]interface{} {
	payload := map[string]interface{}{
		"model":  o.Model,
		"prompt": o.Prompt,
	}
	if len(o.ImageURLs) > 0 {
		payload["image_urls"] = o.ImageURLs
	}
	switch o.Model {
	case ModelSora2:
		payload["aspect_ratio"] = o.AspectRatio
		payload["duration"] = o.DurationSeconds
		payload["remove_watermark"] = *o.RemoveWatermark
	case ModelSora2Pro:
		payload["aspect_ratio"] = o.AspectRatio
		payload["duration"] = o.DurationSeconds
		payload["remove_watermark"] = *o.RemoveWatermark
		payload["quality"] = o.Quality
	case ModelVeo31Fast:
		payload["aspect_ratio"] = o.AspectRatio
	case ModelWan25TextToVideo:
		payload["aspect_ratio"] = o.AspectRatio
		payload["duration"] = o.DurationSeconds
	case ModelWan25ImageToVideo:
		payload["duration"] = o.DurationSeconds
	case ModelSeedance10ProFast:
		payload["duration"] = o.DurationSeconds
		payload["quality"] = o.Quality
		payload["aspect_ratio"] = o.AspectRatio
	}
	if o.CallbackURL != "" {
		payload["callback_url"] = o.CallbackURL
	}
	return payload
}

func ensureStringInSet(val string, allowed []string, field string, model Model) error {
	if val == "" {
		return fmt.Errorf("%s is required for %s", field, model)
	}
	for _, a := range allowed {
		if val == a {
			return nil
		}
	}
	return fmt.Errorf("invalid %s %q for %s", field, val, model)
}

func ensureIntInSet(val int, allowed []int, field string, model Model) error {
	for _, a := range allowed {
		if val == a {
			return nil
		}
	}
	return fmt.Errorf("invalid %s %d for %s", field, val, model)
}
