package filegen

import (
	"context"
	"os"
	"path/filepath"
	"testing"
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