package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// HTTPToolkit provides HTTP request capabilities
type HTTPToolkit struct {
	*toolkit.BaseToolkit
	client *http.Client
}

// New creates a new HTTP toolkit
func New() *HTTPToolkit {
	t := &HTTPToolkit{
		BaseToolkit: toolkit.NewBaseToolkit("http"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Register GET function
	t.RegisterFunction(&toolkit.Function{
		Name:        "http_get",
		Description: "Make an HTTP GET request to a URL and return the response",
		Parameters: map[string]toolkit.Parameter{
			"url": {
				Type:        "string",
				Description: "The URL to request",
				Required:    true,
			},
		},
		Handler: t.httpGet,
	})

	// Register POST function
	t.RegisterFunction(&toolkit.Function{
		Name:        "http_post",
		Description: "Make an HTTP POST request to a URL with data",
		Parameters: map[string]toolkit.Parameter{
			"url": {
				Type:        "string",
				Description: "The URL to request",
				Required:    true,
			},
			"body": {
				Type:        "string",
				Description: "The request body",
				Required:    false,
			},
		},
		Handler: t.httpPost,
	})

	return t
}

// httpGet handles HTTP GET requests
func (h *HTTPToolkit) httpGet(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	url, ok := args["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url must be a string")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return map[string]interface{}{
		"status_code": resp.StatusCode,
		"body":        string(body),
		"headers":     resp.Header,
	}, nil
}

// httpPost handles HTTP POST requests
func (h *HTTPToolkit) httpPost(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	url, ok := args["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url must be a string")
	}

	var body io.Reader
	if bodyStr, ok := args["body"].(string); ok && bodyStr != "" {
		body = strings.NewReader(bodyStr)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return map[string]interface{}{
		"status_code": resp.StatusCode,
		"body":        string(respBody),
		"headers":     resp.Header,
	}, nil
}
