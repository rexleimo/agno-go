package surreal

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
	sqlEndpoint    = "/sql"
)

// ClientConfig 配置 SurrealDB 客户端
type ClientConfig struct {
	BaseURL     string
	Namespace   string
	Database    string
	Username    string
	Password    string
	BearerToken string
	HTTPClient  *http.Client
	Timeout     time.Duration
}

// Client SurrealDB HTTP 客户端
type Client struct {
	httpClient *http.Client
	baseURL    string
	namespace  string
	database   string
	authHeader string
}

// NewClient 创建 SurrealDB HTTP 客户端
func NewClient(cfg ClientConfig) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("surreal: base URL cannot be empty")
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = defaultTimeout
		}
		httpClient = &http.Client{Timeout: timeout}
	}

	client := &Client{
		httpClient: httpClient,
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		namespace:  cfg.Namespace,
		database:   cfg.Database,
		authHeader: buildAuthHeader(cfg),
	}
	return client, nil
}

func buildAuthHeader(cfg ClientConfig) string {
	if token := strings.TrimSpace(cfg.BearerToken); token != "" {
		return "Bearer " + token
	}
	if cfg.Username == "" {
		return ""
	}
	raw := cfg.Username + ":" + cfg.Password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(raw))
}

type responseEnvelope struct {
	Status string          `json:"status"`
	Result json.RawMessage `json:"result"`
	Detail string          `json:"detail"`
}

// execute 发送 SQL 请求并返回响应包
func (c *Client) execute(ctx context.Context, query string, vars map[string]interface{}) ([]responseEnvelope, error) {
	payload := map[string]interface{}{
		"query": query,
	}
	if len(vars) > 0 {
		payload["vars"] = vars
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("surreal: failed to encode payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+sqlEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("surreal: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.namespace != "" {
		req.Header.Set("NS", c.namespace)
	}
	if c.database != "" {
		req.Header.Set("DB", c.database)
	}
	if c.authHeader != "" {
		req.Header.Set("Authorization", c.authHeader)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("surreal: request failed: %w", err)
	}
	defer resp.Body.Close()

	payloadBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("surreal: failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("surreal: request failed with status %d: %s", resp.StatusCode, string(payloadBytes))
	}

	var envelopes []responseEnvelope
	if err := json.Unmarshal(payloadBytes, &envelopes); err != nil {
		return nil, fmt.Errorf("surreal: failed to decode response: %w", err)
	}

	for _, env := range envelopes {
		if strings.ToUpper(env.Status) != "OK" {
			if env.Detail != "" {
				return nil, fmt.Errorf("surreal: statement error: %s", env.Detail)
			}
			return nil, fmt.Errorf("surreal: statement returned status %s", env.Status)
		}
	}
	return envelopes, nil
}

// querySingle 执行 SQL 查询并将第一条非空结果解码到 target
func (c *Client) querySingle(ctx context.Context, query string, vars map[string]interface{}, target interface{}) error {
	envelopes, err := c.execute(ctx, query, vars)
	if err != nil {
		return err
	}

	if target == nil {
		return nil
	}

	for _, env := range envelopes {
		if len(env.Result) == 0 || string(env.Result) == "null" {
			continue
		}
		if err := json.Unmarshal(env.Result, target); err != nil {
			return fmt.Errorf("surreal: failed to decode result: %w", err)
		}
		return nil
	}
	// 如果没有任何结果，也认为成功，但 target 保持零值
	return nil
}
