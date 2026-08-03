package bitbucket

import (
	"context"
	"testing"
)

func TestBitbucketToolkit_ListWorkspaces(t *testing.T) {
	toolkit := New()

	// Test with valid parameters
	result, err := toolkit.listWorkspaces(context.Background(), map[string]interface{}{
		"username":     "testuser",
		"app_password": "testpassword",
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check that workspaces are returned
	workspaces, ok := resultMap["workspaces"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected workspaces array, got: %T", resultMap["workspaces"])
	}

	if len(workspaces) == 0 {
		t.Error("Expected at least one workspace")
	}

	// Check count field
	count, ok := resultMap["count"].(int)
	if !ok {
		t.Fatalf("Expected count integer, got: %T", resultMap["count"])
	}

	if count != len(workspaces) {
		t.Errorf("Count mismatch: expected %d, got %d", len(workspaces), count)
	}
}

func TestBitbucketToolkit_ListRepositories(t *testing.T) {
	toolkit := New()

	// Test with valid parameters
	result, err := toolkit.listRepositories(context.Background(), map[string]interface{}{
		"username":     "testuser",
		"app_password": "testpassword",
		"workspace":    "test-workspace",
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check that repositories are returned
	repositories, ok := resultMap["repositories"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected repositories array, got: %T", resultMap["repositories"])
	}

	if len(repositories) == 0 {
		t.Error("Expected at least one repository")
	}

	// Check workspace field
	workspace, ok := resultMap["workspace"].(string)
	if !ok {
		t.Fatalf("Expected workspace string, got: %T", resultMap["workspace"])
	}

	if workspace != "test-workspace" {
		t.Errorf("Expected workspace 'test-workspace', got '%s'", workspace)
	}
}

func TestBitbucketToolkit_ListPullRequests(t *testing.T) {
	toolkit := New()

	// Test with valid parameters
	result, err := toolkit.listPullRequests(context.Background(), map[string]interface{}{
		"username":     "testuser",
		"app_password": "testpassword",
		"workspace":    "test-workspace",
		"repository":   "test-repo",
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	// Check that pull requests are returned
	pullRequests, ok := resultMap["pull_requests"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected pull_requests array, got: %T", resultMap["pull_requests"])
	}

	if len(pullRequests) == 0 {
		t.Error("Expected at least one pull request")
	}

	// Check repository field
	repository, ok := resultMap["repository"].(string)
	if !ok {
		t.Fatalf("Expected repository string, got: %T", resultMap["repository"])
	}

	if repository != "test-repo" {
		t.Errorf("Expected repository 'test-repo', got '%s'", repository)
	}
}

func TestBitbucketToolkit_ListWorkspaces_MissingParameters(t *testing.T) {
	toolkit := New()

	// Test missing username
	_, err := toolkit.listWorkspaces(context.Background(), map[string]interface{}{
		"app_password": "testpassword",
	})

	if err == nil {
		t.Error("Expected error for missing username")
	}

	// Test missing app_password
	_, err = toolkit.listWorkspaces(context.Background(), map[string]interface{}{
		"username": "testuser",
	})

	if err == nil {
		t.Error("Expected error for missing app_password")
	}
}

func TestBitbucketToolkit_New(t *testing.T) {
	toolkit := New()

	if toolkit == nil {
		t.Fatal("Expected toolkit to be created")
	}

	// Check that functions are registered
	functions := toolkit.Functions()
	if len(functions) != 3 {
		t.Errorf("Expected 3 functions, got %d", len(functions))
	}

	expectedFunctions := []string{"list_workspaces", "list_repositories", "list_pull_requests"}
	for _, expected := range expectedFunctions {
		found := false
		for _, function := range functions {
			if function.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected function '%s' not found", expected)
		}
	}
}