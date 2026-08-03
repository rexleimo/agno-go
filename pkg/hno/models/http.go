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

// PostJSON sends a JSON POST request, verifies the status code, and decodes
// the response body into out. It is the shared HTTP skeleton used by
// providers that talk raw HTTP (Anthropic, Gemini, etc.); providers using an
// SDK client (OpenAI) keep their own call path.
//
// PostJSON 发送 JSON POST 请求、校验状态码、并将响应体解码到 out。
// 它是使用裸 HTTP 的 provider（Anthropic、Gemini 等）共享的 HTTP 骨架；
// 使用 SDK 客户端的 provider（OpenAI）保留自己的调用路径。
func PostJSON(ctx context.Context, client *http.Client, url string, headers map[string]string, payload, out interface{}) error {
	resp, err := PostJSONRaw(ctx, client, url, headers, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return types.NewAPIError(fmt.Sprintf("API error (status %d): %s", resp.StatusCode, string(body)), nil)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return types.NewAPIError("failed to decode response", err)
		}
	}
	return nil
}

// PostJSONRaw sends a JSON POST request and returns the raw response. The
// caller owns the response body and must close it. Used by streaming paths
// that need the live body for SSE parsing.
//
// PostJSONRaw 发送 JSON POST 请求并返回原始响应。调用方拥有响应体并必须
// 关闭它。供需要实时响应体做 SSE 解析的流式路径使用。
func PostJSONRaw(ctx context.Context, client *http.Client, url string, headers map[string]string, payload interface{}) (*http.Response, error) {
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, types.NewAPIError("failed to marshal request", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, types.NewAPIError("failed to create HTTP request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, types.NewAPIError("failed to call API", err)
	}
	return resp, nil
}
