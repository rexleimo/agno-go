// Package cloud provides deployment abstractions for publishing agent
// artifacts (skills, prompts, configs) to storage targets. It belongs to the
// deployment layer: local filesystem and HTTP endpoints are supported out of
// the box, and custom targets implement the Deployer interface.
//
// cloud 包提供发布 agent 工件（技能、提示词、配置）到存储目标的部署抽象。
// 它属于部署层：开箱支持本地文件系统与 HTTP 端点，自定义目标实现
// Deployer 接口即可。
package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Deployer publishes an artifact and returns a deployment ID or URL.
// Deployer 发布工件并返回部署 ID 或 URL。
type Deployer interface {
	// Deploy publishes an artifact and returns a deployment ID or URL.
	// Deploy 发布工件并返回部署 ID 或 URL。
	Deploy(ctx context.Context, artifact string, config map[string]string) (string, error)
}

// LocalDirDeployer writes artifacts into a local directory, returning a
// file:// URL. Useful for local testing and offline deployments.
//
// LocalDirDeployer 将工件写入本地目录并返回 file:// URL。
// 用于本地测试与离线部署。
type LocalDirDeployer struct {
	Dir string
}

// NewLocalDirDeployer creates a deployer rooted at dir (created if missing).
// NewLocalDirDeployer 创建根目录为 dir 的部署器（不存在则创建）。
func NewLocalDirDeployer(dir string) (*LocalDirDeployer, error) {
	if dir == "" {
		return nil, fmt.Errorf("cloud: dir is required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("cloud: create dir: %w", err)
	}
	return &LocalDirDeployer{Dir: dir}, nil
}

// Deploy writes the artifact to Dir/name and returns a file:// URL.
// Deploy 将工件写入 Dir/name 并返回 file:// URL。
func (d *LocalDirDeployer) Deploy(_ context.Context, artifact string, config map[string]string) (string, error) {
	name := config["name"]
	if name == "" {
		name = fmt.Sprintf("artifact-%d.json", time.Now().Unix())
	}
	// Sanitize: no path traversal.
	// 净化：禁止路径穿越。
	if strings.Contains(name, "..") || strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("cloud: invalid artifact name %q", name)
	}
	path := filepath.Join(d.Dir, name)
	if err := os.WriteFile(path, []byte(artifact), 0o644); err != nil {
		return "", fmt.Errorf("cloud: write artifact: %w", err)
	}
	return "file://" + filepath.ToSlash(path), nil
}

// HTTPDeployer POSTs the artifact to an endpoint and returns its response
// body as the deployment handle.
//
// HTTPDeployer 将工件 POST 到端点，并以其响应体作为部署句柄。
type HTTPDeployer struct {
	Endpoint string
	Client   *http.Client
}

// NewHTTPDeployer creates a deployer targeting endpoint.
// NewHTTPDeployer 创建指向 endpoint 的部署器。
func NewHTTPDeployer(endpoint string) *HTTPDeployer {
	return &HTTPDeployer{
		Endpoint: endpoint,
		Client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// Deploy POSTs the artifact as JSON {artifact, config} to the endpoint.
// Deploy 将工件以 JSON {artifact, config} POST 到端点。
func (h *HTTPDeployer) Deploy(ctx context.Context, artifact string, config map[string]string) (string, error) {
	payload, err := json.Marshal(map[string]interface{}{
		"artifact": artifact,
		"config":   config,
	})
	if err != nil {
		return "", fmt.Errorf("cloud: marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.Endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return "", fmt.Errorf("cloud: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("cloud: deploy: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("cloud: deploy failed (status %d)", resp.StatusCode)
	}
	buf := make([]byte, 512)
	n, _ := resp.Body.Read(buf)
	return strings.TrimSpace(string(buf[:n])), nil
}

// NoopDeployer is a placeholder deployer for local/testing usage.
// NoopDeployer 是本地/测试用的占位部署器。
type NoopDeployer struct{}

// Deploy returns a pseudo-URL indicating the deployment location.
// Deploy 返回伪 URL 指示部署位置。
func (NoopDeployer) Deploy(_ context.Context, artifact string, _ map[string]string) (string, error) {
	return "local://" + artifact, nil
}
