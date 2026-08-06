package filegen

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/tools/file"
)

func TestFileGenToolkit_New(t *testing.T) {
	toolkit := New()

	if toolkit == nil {
		t.Fatal("Expected toolkit to be created, got nil")
	}

	if toolkit.Name() != "file_generation" {
		t.Errorf("Expected toolkit name 'file_generation', got '%s'", toolkit.Name())
	}

	functions := toolkit.Functions()
	if len(functions) != 3 {
		t.Errorf("Expected 3 functions, got %d", len(functions))
	}

	if _, exists := functions["create_file"]; !exists {
		t.Error("Expected 'create_file' function to exist")
	}

	if _, exists := functions["create_directory"]; !exists {
		t.Error("Expected 'create_directory' function to exist")
	}

	if _, exists := functions["generate_from_template"]; !exists {
		t.Error("Expected 'generate_from_template' function to exist")
	}
}

func TestFileGenToolkit_CreateFile(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	// Test creating a file
	result, err := toolkit.Execute(ctx, "create_file", map[string]interface{}{
		"file_path": testFile,
		"content":   "Hello, World!",
	})

	if err != nil {
		t.Fatalf("Create file failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["file_path"] != testFile {
		t.Errorf("Expected file_path '%s', got '%v'", testFile, resultMap["file_path"])
	}

	if resultMap["size"] != 13 {
		t.Errorf("Expected size 13, got %v", resultMap["size"])
	}

	if resultMap["created"] != true {
		t.Error("Expected created to be true")
	}

	// Verify file was actually created
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != "Hello, World!" {
		t.Errorf("Expected file content 'Hello, World!', got '%s'", string(content))
	}
}

func TestFileGenToolkit_CreateFileWithOverwrite(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	// Create initial file
	if err := os.WriteFile(testFile, []byte("Initial content"), 0644); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Test overwriting the file
	result, err := toolkit.Execute(ctx, "create_file", map[string]interface{}{
		"file_path": testFile,
		"content":   "New content",
		"overwrite": true,
	})

	if err != nil {
		t.Fatalf("Create file with overwrite failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["created"] != true {
		t.Error("Expected created to be true")
	}

	// Verify file was overwritten
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read overwritten file: %v", err)
	}

	if string(content) != "New content" {
		t.Errorf("Expected file content 'New content', got '%s'", string(content))
	}
}

func TestFileGenToolkit_CreateFileWithoutOverwrite(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	// Create initial file
	if err := os.WriteFile(testFile, []byte("Initial content"), 0644); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Test creating file without overwrite (should fail)
	_, err := toolkit.Execute(ctx, "create_file", map[string]interface{}{
		"file_path": testFile,
		"content":   "New content",
		"overwrite": false,
	})

	if err == nil {
		t.Error("Expected error when creating file without overwrite on existing file")
	}
}

func TestFileGenToolkit_CreateDirectory(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	newDir := filepath.Join(tempDir, "new_subdir")

	// Test creating a directory
	result, err := toolkit.Execute(ctx, "create_directory", map[string]interface{}{
		"dir_path": newDir,
	})

	if err != nil {
		t.Fatalf("Create directory failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["dir_path"] != newDir {
		t.Errorf("Expected dir_path '%s', got '%v'", newDir, resultMap["dir_path"])
	}

	if resultMap["created"] != true {
		t.Error("Expected created to be true")
	}

	// Verify directory was actually created
	if _, err := os.Stat(newDir); err != nil {
		t.Fatalf("Failed to verify directory creation: %v", err)
	}
}

func TestFileGenToolkit_CreateDirectoryAlreadyExists(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Test creating a directory that already exists (should fail)
	_, err := toolkit.Execute(ctx, "create_directory", map[string]interface{}{
		"dir_path": tempDir,
	})

	if err == nil {
		t.Error("Expected error when creating directory that already exists")
	}
}

func TestFileGenToolkit_GenerateFromTemplate(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test template generation
	result, err := toolkit.Execute(ctx, "generate_from_template", map[string]interface{}{
		"template": "Hello {{name}}, welcome to {{project}}!",
		"variables": map[string]interface{}{
			"name":    "Alice",
			"project": "Agno-Go",
		},
	})

	if err != nil {
		t.Fatalf("Generate from template failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["result"] != "Hello Alice, welcome to Agno-Go!" {
		t.Errorf("Expected result 'Hello Alice, welcome to Agno-Go!', got '%v'", resultMap["result"])
	}

	if resultMap["template"] != "Hello {{name}}, welcome to {{project}}!" {
		t.Errorf("Expected template 'Hello {{name}}, welcome to {{project}}!', got '%v'", resultMap["template"])
	}
}

func TestFileGenToolkit_GenerateFromTemplateNoVariables(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test template generation without variables
	result, err := toolkit.Execute(ctx, "generate_from_template", map[string]interface{}{
		"template": "Hello World!",
	})

	if err != nil {
		t.Fatalf("Generate from template failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["result"] != "Hello World!" {
		t.Errorf("Expected result 'Hello World!', got '%v'", resultMap["result"])
	}
}

func TestFileGenToolkit_CreateFileMissingParameters(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test missing required parameters
	_, err := toolkit.Execute(ctx, "create_file", map[string]interface{}{})

	if err == nil {
		t.Error("Expected error for missing file_path parameter")
	}
}

func TestFileGenToolkit_CreateDirectoryMissingParameters(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test missing required parameters
	_, err := toolkit.Execute(ctx, "create_directory", map[string]interface{}{})

	if err == nil {
		t.Error("Expected error for missing dir_path parameter")
	}
}

func TestFileGenToolkit_GenerateFromTemplateMissingParameters(t *testing.T) {
	toolkit := New()
	ctx := context.Background()

	// Test missing required parameters
	_, err := toolkit.Execute(ctx, "generate_from_template", map[string]interface{}{})

	if err == nil {
		t.Error("Expected error for missing template parameter")
	}
}

func TestFileGenToolkit_NewWithSandbox_ConfinesCreateOperations(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	sandbox, err := file.NewSandbox(file.WithWriteRoot(root))
	if err != nil {
		t.Fatal(err)
	}
	toolkit := NewWithSandbox(sandbox)
	defer func() { _ = toolkit.Close() }()

	ctx := context.Background()
	filePath := filepath.Join("reports", "today.txt")
	if _, err := toolkit.Execute(ctx, "create_file", map[string]interface{}{
		"file_path": filePath,
		"content":   "sandboxed",
	}); err != nil {
		t.Fatalf("create root-relative file: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(root, filePath))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "sandboxed" {
		t.Fatalf("content = %q, want sandboxed", content)
	}

	if _, err := toolkit.Execute(ctx, "create_directory", map[string]interface{}{
		"dir_path": filepath.Join("generated", "nested"),
	}); err != nil {
		t.Fatalf("create root-relative directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "generated", "nested")); err != nil {
		t.Fatalf("stat created directory: %v", err)
	}

	for _, request := range []struct {
		name string
		args map[string]interface{}
	}{
		{
			name: "absolute file path",
			args: map[string]interface{}{
				"file_path": filepath.Join(outside, "blocked.txt"),
				"content":   "blocked",
			},
		},
		{
			name: "traversal file path",
			args: map[string]interface{}{
				"file_path": filepath.Join("..", "blocked.txt"),
				"content":   "blocked",
			},
		},
		{
			name: "absolute directory path",
			args: map[string]interface{}{
				"dir_path": filepath.Join(outside, "blocked"),
			},
		},
		{
			name: "traversal directory path",
			args: map[string]interface{}{
				"dir_path": filepath.Join("..", "blocked"),
			},
		},
	} {
		t.Run(request.name, func(t *testing.T) {
			function := "create_file"
			if _, ok := request.args["dir_path"]; ok {
				function = "create_directory"
			}
			if _, err := toolkit.Execute(ctx, function, request.args); err == nil {
				t.Fatalf("expected %s to be rejected", request.name)
			}
		})
	}
}

func TestFileGenToolkit_NewWithSandbox_OverwriteRequiresDualOptIn(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "existing.txt")
	if err := os.WriteFile(target, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	withoutPolicy, err := file.NewSandbox(file.WithWriteRoot(root))
	if err != nil {
		t.Fatal(err)
	}
	withoutPolicyTools := NewWithSandbox(withoutPolicy)
	if _, err := withoutPolicyTools.Execute(ctx, "create_file", map[string]interface{}{
		"file_path": "existing.txt",
		"content":   "new",
		"overwrite": true,
	}); err == nil {
		t.Fatal("expected overwrite to require sandbox policy")
	}
	if err := withoutPolicyTools.Close(); err != nil {
		t.Fatal(err)
	}

	withPolicy, err := file.NewSandbox(file.WithWriteRoot(root), file.WithAllowOverwrite(true))
	if err != nil {
		t.Fatal(err)
	}
	withPolicyTools := NewWithSandbox(withPolicy)
	defer func() { _ = withPolicyTools.Close() }()
	if _, err := withPolicyTools.Execute(ctx, "create_file", map[string]interface{}{
		"file_path": "existing.txt",
		"content":   "new",
	}); err == nil {
		t.Fatal("expected overwrite to require the tool argument")
	}
	if _, err := withPolicyTools.Execute(ctx, "create_file", map[string]interface{}{
		"file_path": "existing.txt",
		"content":   "new",
		"overwrite": true,
	}); err != nil {
		t.Fatalf("overwrite with both opt-ins: %v", err)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "new" {
		t.Fatalf("content = %q, want new", content)
	}
}

func TestFileGenToolkit_NewWithSandbox_RejectsDirectorySymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "out")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	sandbox, err := file.NewSandbox(file.WithWriteRoot(root))
	if err != nil {
		t.Fatal(err)
	}
	toolkit := NewWithSandbox(sandbox)
	defer func() { _ = toolkit.Close() }()

	if _, err := toolkit.Execute(context.Background(), "create_file", map[string]interface{}{
		"file_path": filepath.Join("out", "escaped.txt"),
		"content":   "blocked",
	}); err == nil {
		t.Fatal("expected file creation through a directory symlink to fail")
	}
	if _, err := toolkit.Execute(context.Background(), "create_directory", map[string]interface{}{
		"dir_path": filepath.Join("out", "escaped-dir"),
	}); err == nil {
		t.Fatal("expected directory creation through a directory symlink to fail")
	}
	if _, err := os.Stat(filepath.Join(outside, "escaped.txt")); !os.IsNotExist(err) {
		t.Fatalf("outside file was created or could not be checked: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "escaped-dir")); !os.IsNotExist(err) {
		t.Fatalf("outside directory was created or could not be checked: %v", err)
	}
}

func TestFileGenToolkit_NewWithSandbox_NilSandboxFailsClosed(t *testing.T) {
	toolkit := NewWithSandbox(nil)
	if _, err := toolkit.Execute(context.Background(), "create_file", map[string]interface{}{
		"file_path": "blocked.txt",
		"content":   "blocked",
	}); err == nil {
		t.Fatal("expected nil sandbox configuration to fail closed")
	}
}
