package bitbucket

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newBitbucketTestToolkit(t *testing.T) *BitbucketToolkit {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok || user != "testuser" || password != "testpassword" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/workspaces":
			fmt.Fprint(w, `{"values":[{"uuid":"{workspace-uuid}","slug":"test-workspace","name":"Test Workspace"}],"next":"https://next.example/workspaces"}`)
		case "/repositories/test-workspace":
			fmt.Fprint(w, `{"values":[{"uuid":"{repo-uuid}","slug":"test-repo","name":"Test Repository","full_name":"test-workspace/test-repo"}]}`)
		case "/repositories/test-workspace/test-repo/pullrequests":
			fmt.Fprint(w, `{"values":[{"id":7,"title":"Real pull request","state":"OPEN"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	tk := NewWithBase(server.URL)
	return tk
}

func TestBitbucketToolkit_ListWorkspaces(t *testing.T) {
	tk := newBitbucketTestToolkit(t)
	result, err := tk.Execute(context.Background(), "list_workspaces", map[string]interface{}{
		"username": "testuser", "app_password": "testpassword",
	})
	if err != nil {
		t.Fatalf("list_workspaces failed: %v", err)
	}
	resultMap := result.(map[string]interface{})
	workspaces := resultMap["workspaces"].([]map[string]interface{})
	if len(workspaces) != 1 || workspaces[0]["slug"] != "test-workspace" {
		t.Fatalf("unexpected workspaces: %#v", workspaces)
	}
	if resultMap["count"] != 1 || resultMap["next"] == nil {
		t.Fatalf("unexpected pagination result: %#v", resultMap)
	}
}

func TestBitbucketToolkit_ListRepositories(t *testing.T) {
	tk := newBitbucketTestToolkit(t)
	result, err := tk.Execute(context.Background(), "list_repositories", map[string]interface{}{
		"username": "testuser", "app_password": "testpassword", "workspace": "test-workspace",
	})
	if err != nil {
		t.Fatalf("list_repositories failed: %v", err)
	}
	resultMap := result.(map[string]interface{})
	repositories := resultMap["repositories"].([]map[string]interface{})
	if len(repositories) != 1 || repositories[0]["slug"] != "test-repo" {
		t.Fatalf("unexpected repositories: %#v", repositories)
	}
}

func TestBitbucketToolkit_ListPullRequests(t *testing.T) {
	tk := newBitbucketTestToolkit(t)
	result, err := tk.Execute(context.Background(), "list_pull_requests", map[string]interface{}{
		"username": "testuser", "app_password": "testpassword", "workspace": "test-workspace", "repository": "test-repo",
	})
	if err != nil {
		t.Fatalf("list_pull_requests failed: %v", err)
	}
	resultMap := result.(map[string]interface{})
	pullRequests := resultMap["pull_requests"].([]map[string]interface{})
	if len(pullRequests) != 1 || pullRequests[0]["title"] != "Real pull request" {
		t.Fatalf("unexpected pull requests: %#v", pullRequests)
	}
}

func TestBitbucketToolkit_MissingParameters(t *testing.T) {
	tk := New()
	if _, err := tk.Execute(context.Background(), "list_workspaces", map[string]interface{}{"username": "testuser"}); err == nil {
		t.Error("expected missing app_password error")
	}
	if _, err := tk.Execute(context.Background(), "list_repositories", map[string]interface{}{
		"username": "testuser", "app_password": "secret",
	}); err == nil {
		t.Error("expected missing workspace error")
	}
}

func TestBitbucketToolkit_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"rate limited"}`, http.StatusTooManyRequests)
	}))
	t.Cleanup(server.Close)
	_, err := NewWithBase(server.URL).Execute(context.Background(), "list_workspaces", map[string]interface{}{
		"username": "testuser", "app_password": "testpassword",
	})
	if err == nil {
		t.Fatal("expected API error")
	}
}

func TestBitbucketToolkit_New(t *testing.T) {
	tk := New()
	if tk == nil || len(tk.Functions()) != 3 {
		t.Fatalf("expected three registered functions")
	}
	for _, name := range []string{"list_workspaces", "list_repositories", "list_pull_requests"} {
		if _, ok := tk.Functions()[name]; !ok {
			t.Errorf("function %q is not registered", name)
		}
	}
}
