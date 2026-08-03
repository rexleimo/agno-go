package file

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// FileTools provides file operation capabilities
type FileTools struct {
	*toolkit.BaseToolkit
	baseDir string // Base directory for file operations (for security)
}

// New creates a new FileTools instance
func New() *FileTools {
	ft := &FileTools{
		BaseToolkit: toolkit.NewBaseToolkit("file_operations"),
		baseDir:     "", // empty means no restriction
	}

	ft.RegisterFunction(&toolkit.Function{
		Name:        "read_file",
		Description: "Read contents of a file",
		Parameters: map[string]toolkit.Parameter{
			"path": {
				Type:        "string",
				Description: "Path to the file to read",
				Required:    true,
			},
		},
		Handler: ft.readFile,
	})

	ft.RegisterFunction(&toolkit.Function{
		Name:        "write_file",
		Description: "Write content to a file",
		Parameters: map[string]toolkit.Parameter{
			"path": {
				Type:        "string",
				Description: "Path to the file to write",
				Required:    true,
			},
			"content": {
				Type:        "string",
				Description: "Content to write to the file",
				Required:    true,
			},
		},
		Handler: ft.writeFile,
	})

	ft.RegisterFunction(&toolkit.Function{
		Name:        "list_files",
		Description: "List files in a directory",
		Parameters: map[string]toolkit.Parameter{
			"path": {
				Type:        "string",
				Description: "Directory path to list files from",
				Required:    true,
			},
		},
		Handler: ft.listFiles,
	})

	ft.RegisterFunction(&toolkit.Function{
		Name:        "delete_file",
		Description: "Delete a file",
		Parameters: map[string]toolkit.Parameter{
			"path": {
				Type:        "string",
				Description: "Path to the file to delete",
				Required:    true,
			},
		},
		Handler: ft.deleteFile,
	})

	ft.RegisterFunction(&toolkit.Function{
		Name:        "file_exists",
		Description: "Check if a file exists",
		Parameters: map[string]toolkit.Parameter{
			"path": {
				Type:        "string",
				Description: "Path to check",
				Required:    true,
			},
		},
		Handler: ft.fileExists,
	})

	ft.RegisterFunction(&toolkit.Function{
		Name:        "read_pptx",
		Description: "Extract slide text from a PPTX presentation file",
		Parameters: map[string]toolkit.Parameter{
			"path": {
				Type:        "string",
				Description: "Path to the PPTX file",
				Required:    true,
			},
		},
		Handler: ft.readPPTX,
	})

	return ft
}

// NewWithBaseDir creates a FileTools instance with a base directory restriction
func NewWithBaseDir(baseDir string) *FileTools {
	ft := New()
	ft.baseDir = baseDir
	return ft
}

func (ft *FileTools) readFile(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}

	if err := ft.validatePath(path); err != nil {
		return nil, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return map[string]interface{}{
		"path":    path,
		"content": string(content),
		"size":    len(content),
	}, nil
}

func (ft *FileTools) writeFile(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}

	content, ok := args["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content must be a string")
	}

	if err := ft.validatePath(path); err != nil {
		return nil, err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return map[string]interface{}{
		"path":    path,
		"size":    len(content),
		"success": true,
	}, nil
}

func (ft *FileTools) listFiles(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}

	if err := ft.validatePath(path); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	files := make([]map[string]interface{}, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, map[string]interface{}{
			"name":   entry.Name(),
			"is_dir": entry.IsDir(),
			"size":   info.Size(),
		})
	}

	return map[string]interface{}{
		"path":  path,
		"files": files,
		"count": len(files),
	}, nil
}

func (ft *FileTools) deleteFile(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}

	if err := ft.validatePath(path); err != nil {
		return nil, err
	}

	if err := os.Remove(path); err != nil {
		return nil, fmt.Errorf("failed to delete file: %w", err)
	}

	return map[string]interface{}{
		"path":    path,
		"success": true,
		"message": "File deleted successfully",
	}, nil
}

func (ft *FileTools) fileExists(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}

	if err := ft.validatePath(path); err != nil {
		return map[string]interface{}{
			"path":   path,
			"exists": false,
			"error":  err.Error(),
		}, nil
	}

	info, err := os.Stat(path)
	exists := err == nil

	result := map[string]interface{}{
		"path":   path,
		"exists": exists,
	}

	if exists {
		result["is_dir"] = info.IsDir()
		result["size"] = info.Size()
	}

	return result, nil
}

func (ft *FileTools) readPPTX(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}

	if err := ft.validatePath(path); err != nil {
		return nil, err
	}

	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open pptx: %w", err)
	}
	defer reader.Close()

	type slideFile struct {
		name string
		file *zip.File
	}

	var slideFiles []slideFile
	for _, f := range reader.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			slideFiles = append(slideFiles, slideFile{name: f.Name, file: f})
		}
	}

	if len(slideFiles) == 0 {
		return nil, fmt.Errorf("no slide xml files found in pptx")
	}

	sort.Slice(slideFiles, func(i, j int) bool {
		return slideFiles[i].name < slideFiles[j].name
	})

	slides := make([]map[string]interface{}, 0, len(slideFiles))

	for idx, entry := range slideFiles {
		rc, err := entry.file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open slide %s: %w", entry.name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read slide %s: %w", entry.name, err)
		}

		text := extractSlideText(data)
		slides = append(slides, map[string]interface{}{
			"index": idx,
			"name":  entry.name,
			"text":  text,
		})
	}

	return map[string]interface{}{
		"path":   path,
		"slides": slides,
		"count":  len(slides),
	}, nil
}

func extractSlideText(data []byte) string {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var builder strings.Builder
	var pendingLine strings.Builder

	for {
		tok, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return builder.String()
		}

		switch element := tok.(type) {
		case xml.StartElement:
			if element.Name.Local == "t" {
				if next, err := decoder.Token(); err == nil {
					if char, ok := next.(xml.CharData); ok {
						text := strings.TrimSpace(string(char))
						if text != "" {
							if pendingLine.Len() > 0 {
								pendingLine.WriteString(" ")
							}
							pendingLine.WriteString(text)
						}
					}
				}
			}
		case xml.EndElement:
			if element.Name.Local == "p" && pendingLine.Len() > 0 {
				if builder.Len() > 0 {
					builder.WriteString("\n")
				}
				builder.WriteString(pendingLine.String())
				pendingLine.Reset()
			}
		}
	}

	if pendingLine.Len() > 0 {
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(pendingLine.String())
	}

	return builder.String()
}

// validatePath checks if the path is allowed
func (ft *FileTools) validatePath(path string) error {
	if ft.baseDir == "" {
		return nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	absBaseDir, err := filepath.Abs(ft.baseDir)
	if err != nil {
		return fmt.Errorf("invalid base directory: %w", err)
	}

	relPath, err := filepath.Rel(absBaseDir, absPath)
	if err != nil || filepath.IsAbs(relPath) || len(relPath) > 0 && relPath[0] == '.' {
		return fmt.Errorf("path %s is outside allowed directory", path)
	}

	return nil
}
