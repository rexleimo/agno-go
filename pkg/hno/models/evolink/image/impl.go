package image

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	evolink "github.com/rexleimo/agno-go/pkg/hno/models/evolink"
)

// Options defines GPT-4O image generation parameters for EvoLink
type Model string

const (
	ModelGPT4O             Model = "gpt-4o-image"
	ModelSeedream40        Model = "doubao-seedream-4.0"
	ModelNanoBanana        Model = "gemini-2.5-flash-image"
	ModelQwenImageEdit     Model = "qwen-image-edit"
	ModelWan25TextToImage  Model = "wan2.5-text-to-image"
	ModelWan25ImageToImage Model = "wan2.5-image-to-image"
)

var allowedModels = map[Model]struct{}{
	ModelGPT4O:             {},
	ModelSeedream40:        {},
	ModelNanoBanana:        {},
	ModelQwenImageEdit:     {},
	ModelWan25TextToImage:  {},
	ModelWan25ImageToImage: {},
}

type Options struct {
	Model       Model
	Prompt      string
	Size        string // 1:1, 2:3, 3:2, 1024x1024, 1024x1536, 1536x1024
	N           int    // 1,2,4
	References  []string
	MaskURL     string // optional, requires exactly one reference; must be .png
	CallbackURL string // optional HTTPS only
}

var allowedSizes = map[string]struct{}{
	"1:1": {}, "2:3": {}, "3:2": {},
	"1024x1024": {}, "1024x1536": {}, "1536x1024": {},
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
	if _, ok := allowedSizes[o.Size]; !ok {
		return fmt.Errorf("invalid size: %s", o.Size)
	}
	if o.N != 1 && o.N != 2 && o.N != 4 {
		return fmt.Errorf("invalid n: %d", o.N)
	}
	if len(o.References) > 5 {
		return fmt.Errorf("too many references: %d", len(o.References))
	}
	for _, r := range o.References {
		if _, err := url.ParseRequestURI(r); err != nil {
			return fmt.Errorf("invalid reference url: %w", err)
		}
	}
	if o.MaskURL != "" {
		if _, err := url.ParseRequestURI(o.MaskURL); err != nil {
			return fmt.Errorf("invalid mask url: %w", err)
		}
		if path.Ext(strings.ToLower(o.MaskURL)) != ".png" {
			return fmt.Errorf("mask must be png")
		}
		if len(o.References) != 1 {
			return fmt.Errorf("exactly one reference required when mask is used")
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
	return nil
}

// Response is a minimal image task completion payload
type Response struct {
	TaskID string                 `json:"task_id"`
	Status string                 `json:"status"`
	Data   map[string]interface{} `json:"data"`
}

// Generate creates an image generation task and polls until completion
func Generate(ctx context.Context, c *evolink.Client, opts Options) (*Response, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"model":  opts.Model,
		"prompt": opts.Prompt,
		"size":   opts.Size,
		"n":      opts.N,
	}
	if len(opts.References) > 0 {
		payload["references"] = opts.References
	}
	if opts.MaskURL != "" {
		payload["mask_url"] = opts.MaskURL
	}
	if opts.CallbackURL != "" {
		payload["callback_url"] = opts.CallbackURL
	}
	var createResp struct {
		TaskID string `json:"task_id"`
	}
	if err := c.PostJSON(ctx, "/v1/images/generations", payload, &createResp); err != nil {
		return nil, err
	}
	tr, err := c.PollTask(ctx, createResp.TaskID, 2*time.Second)
	if err != nil {
		return nil, err
	}
	return &Response{TaskID: tr.ID, Status: tr.Status, Data: tr.Data}, nil
}
