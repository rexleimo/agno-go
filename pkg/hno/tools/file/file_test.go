package file

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	ft := New()
	if ft == nil {
		t.Fatal("New() returned nil")
	}

	funcs := ft.Functions()
	expectedFuncs := []string{"read_file", "write_file", "list_files", "delete_file", "file_exists", "read_pptx"}

	for _, name := range expectedFuncs {
		if _, exists := funcs[name]; !exists {
			t.Errorf("Expected function %s not found", name)
		}
	}
}

func TestReadFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"

	// Create test file
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ft := New()
	result, err := ft.readFile(context.Background(), map[string]interface{}{
		"path": testFile,
	})

	if err != nil {
		t.Errorf("readFile() error = %v", err)
	}

	resultMap := result.(map[string]interface{})
	if resultMap["content"] != testContent {
		t.Errorf("readFile() content = %v, want %v", resultMap["content"], testContent)
	}
}

func TestWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "subdir", "test.txt")
	testContent := "Test content"

	ft := New()
	result, err := ft.writeFile(context.Background(), map[string]interface{}{
		"path":    testFile,
		"content": testContent,
	})

	if err != nil {
		t.Errorf("writeFile() error = %v", err)
	}

	resultMap := result.(map[string]interface{})
	if !resultMap["success"].(bool) {
		t.Error("writeFile() success = false")
	}

	// Verify file was written
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("Failed to verify written file: %v", err)
	}
	if string(content) != testContent {
		t.Errorf("File content = %v, want %v", string(content), testContent)
	}
}

func TestListFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, name := range testFiles {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	ft := New()
	result, err := ft.listFiles(context.Background(), map[string]interface{}{
		"path": tmpDir,
	})

	if err != nil {
		t.Errorf("listFiles() error = %v", err)
	}

	resultMap := result.(map[string]interface{})
	files := resultMap["files"].([]map[string]interface{})

	if len(files) != len(testFiles) {
		t.Errorf("listFiles() count = %v, want %v", len(files), len(testFiles))
	}
}

func TestDeleteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ft := New()
	result, err := ft.deleteFile(context.Background(), map[string]interface{}{
		"path": testFile,
	})

	if err != nil {
		t.Errorf("deleteFile() error = %v", err)
	}

	resultMap := result.(map[string]interface{})
	if !resultMap["success"].(bool) {
		t.Error("deleteFile() success = false")
	}

	// Verify file was deleted
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("File was not deleted")
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	ft := New()

	// Test non-existent file
	result, err := ft.fileExists(context.Background(), map[string]interface{}{
		"path": testFile,
	})
	if err != nil {
		t.Errorf("fileExists() error = %v", err)
	}

	resultMap := result.(map[string]interface{})
	if resultMap["exists"].(bool) {
		t.Error("fileExists() = true for non-existent file")
	}

	// Create file and test again
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result, err = ft.fileExists(context.Background(), map[string]interface{}{
		"path": testFile,
	})
	if err != nil {
		t.Errorf("fileExists() error = %v", err)
	}

	resultMap = result.(map[string]interface{})
	if !resultMap["exists"].(bool) {
		t.Error("fileExists() = false for existing file")
	}
}

func TestReadPPTX(t *testing.T) {
	tmpDir := t.TempDir()
	pptxPath := filepath.Join(tmpDir, "deck.pptx")

	if err := createTestPPTX(pptxPath, []string{"Title Slide", "Second slide content"}); err != nil {
		t.Fatalf("failed to create test pptx: %v", err)
	}

	ft := New()
	result, err := ft.readPPTX(context.Background(), map[string]interface{}{
		"path": pptxPath,
	})
	if err != nil {
		t.Fatalf("readPPTX error: %v", err)
	}

	resultMap := result.(map[string]interface{})
	slides := resultMap["slides"].([]map[string]interface{})
	if len(slides) != 2 {
		t.Fatalf("expected 2 slides, got %d", len(slides))
	}
	if slides[0]["text"] != "Title Slide" {
		t.Fatalf("unexpected first slide text %v", slides[0]["text"])
	}
	if slides[1]["text"] != "Second slide content" {
		t.Fatalf("unexpected second slide text %v", slides[1]["text"])
	}
}

func createTestPPTX(path string, slides []string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	defer zw.Close()

	for idx, content := range slides {
		name := fmt.Sprintf("ppt/slides/slide%d.xml", idx+1)
		writer, err := zw.Create(name)
		if err != nil {
			return err
		}
		xml := fmt.Sprintf(`<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:cSld><p:spTree><p:sp><p:txBody><a:p><a:r><a:t>%s</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld></p:sld>`, content)
		if _, err := writer.Write([]byte(xml)); err != nil {
			return err
		}
	}

	// Add minimal content types to keep structure valid for other tools
	ct, err := zw.Create("[Content_Types].xml")
	if err != nil {
		return err
	}
	if _, err := ct.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="xml" ContentType="application/xml"/>
</Types>`)); err != nil {
		return err
	}

	return nil
}

func TestValidatePath(t *testing.T) {
	tmpDir := t.TempDir()
	ft := NewWithBaseDir(tmpDir)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid path inside base dir",
			path:    filepath.Join(tmpDir, "test.txt"),
			wantErr: false,
		},
		{
			name:    "valid relative path",
			path:    filepath.Join(tmpDir, "subdir", "test.txt"),
			wantErr: false,
		},
		{
			name:    "invalid path outside base dir",
			path:    "/tmp/test.txt",
			wantErr: true,
		},
		{
			name:    "invalid path with parent directory",
			path:    filepath.Join(tmpDir, "..", "test.txt"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ft.validatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
