package cloud

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalDirDeployer(t *testing.T) {
	dir := t.TempDir()
	d, err := NewLocalDirDeployer(dir)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	url, err := d.Deploy(context.Background(), `{"skill":"web-research"}`, map[string]string{"name": "my-skill.json"})
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if !strings.HasPrefix(url, "file://") {
		t.Errorf("expected file:// URL, got %q", url)
	}

	// File must exist with content.
	// 文件必须存在且内容正确。
	content, err := os.ReadFile(filepath.Join(dir, "my-skill.json"))
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	if !strings.Contains(string(content), "web-research") {
		t.Errorf("artifact content mismatch: %s", content)
	}
}

func TestLocalDirDeployer_PathTraversalRejected(t *testing.T) {
	d, err := NewLocalDirDeployer(t.TempDir())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = d.Deploy(context.Background(), "x", map[string]string{"name": "../../evil"})
	if err == nil {
		t.Error("expected path traversal to be rejected")
	}
}

func TestLocalDirDeployer_AutoName(t *testing.T) {
	d, err := NewLocalDirDeployer(t.TempDir())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	url, err := d.Deploy(context.Background(), "artifact", nil)
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if !strings.HasPrefix(url, "file://") {
		t.Errorf("expected file:// URL, got %q", url)
	}
}

func TestHTTPDeployer(t *testing.T) {
	var gotPayload string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		gotPayload = string(buf[:n])
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("deployment-42"))
	}))
	defer srv.Close()

	d := NewHTTPDeployer(srv.URL)
	handle, err := d.Deploy(context.Background(), "artifact-data", map[string]string{"env": "prod"})
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if handle != "deployment-42" {
		t.Errorf("handle = %q, want deployment-42", handle)
	}
	if !strings.Contains(gotPayload, "artifact-data") || !strings.Contains(gotPayload, "prod") {
		t.Errorf("payload mismatch: %s", gotPayload)
	}
}

func TestHTTPDeployer_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	d := NewHTTPDeployer(srv.URL)
	if _, err := d.Deploy(context.Background(), "x", nil); err == nil {
		t.Error("expected error on non-2xx status")
	}
}

func TestNoopDeployer(t *testing.T) {
	url, err := (NoopDeployer{}).Deploy(context.Background(), "skill", nil)
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if url != "local://skill" {
		t.Errorf("url = %q", url)
	}
}
