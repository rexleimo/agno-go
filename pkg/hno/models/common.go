package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// HTTPClient provides common HTTP functionality for model providers
type HTTPClient struct {
	client *http.Client
}

// NewHTTPClient creates a new HTTP client
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{},
	}
}

// PostJSON makes a POST request with JSON body and returns the response
func (c *HTTPClient) PostJSON(ctx context.Context, url string, headers map[string]string, body interface{}) (*http.Response, error) {
	// Marshal request body
	reqBody, err := json.Marshal(body)
	if err != nil {
		return nil, types.NewAPIError("failed to marshal request", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, types.NewAPIError("failed to create HTTP request", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, types.NewAPIError("API request failed", err)
	}

	return resp, nil
}

// ReadJSONResponse reads and decodes a JSON response
func ReadJSONResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return types.NewAPIError(fmt.Sprintf("API error (status %d): %s", resp.StatusCode, string(body)), nil)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return types.NewAPIError("failed to decode response", err)
	}

	return nil
}

// ReadErrorResponse reads an error response and returns an error
func ReadErrorResponse(resp *http.Response) error {
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return types.NewAPIError(fmt.Sprintf("API error (status %d): %s", resp.StatusCode, string(body)), nil)
}

// ConvertMessages converts types.Message to a generic message format
// This is useful for building request messages across different providers
func ConvertMessages(messages []*types.Message) []map[string]interface{} {
	result := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		result[i] = map[string]interface{}{
			"role":    string(msg.Role),
			"content": msg.Content,
		}
	}
	return result
}

// MergeConfig merges request-level config with model-level config
// Request-level values take precedence over model-level values
func MergeConfig(reqTemp, modelTemp float64, reqTokens, modelTokens int) (float64, int) {
	temperature := modelTemp
	if reqTemp > 0 {
		temperature = reqTemp
	}

	maxTokens := modelTokens
	if reqTokens > 0 {
		maxTokens = reqTokens
	}

	return temperature, maxTokens
}

// BuildToolDefinitions converts model tool definitions to a specific format
// This is a helper that can be customized per provider
func BuildToolDefinitions(tools []ToolDefinition) []map[string]interface{} {
	if len(tools) == 0 {
		return nil
	}

	result := make([]map[string]interface{}, len(tools))
	for i, tool := range tools {
		result[i] = map[string]interface{}{
			"type": tool.Type,
			"function": map[string]interface{}{
				"name":        tool.Function.Name,
				"description": tool.Function.Description,
				"parameters":  tool.Function.Parameters,
			},
		}
	}
	return result
}
