package filegen

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// FileGenToolkit provides file generation capabilities
// This is a basic implementation for development tools

// FileGenToolkit provides file generation capabilities
type FileGenToolkit struct {
	*toolkit.BaseToolkit
}

// New creates a new FileGen toolkit
func New() *FileGenToolkit {
	t := &FileGenToolkit{
		BaseToolkit: toolkit.NewBaseToolkit("file_generation"),
	}

	// Register file creation function
	t.RegisterFunction(&toolkit.Function{
		Name:        "create_file",
		Description: "Create a new file with specified content",
		Parameters: map[string]toolkit.Parameter{
			"file_path": {
				Type:        "string",
				Description: "The path where the file should be created",
				Required:    true,
			},
			"content": {
				Type:        "string",
				Description: "The content to write to the file",
				Required:    true,
			},
			"overwrite": {
				Type:        "boolean",
				Description: "Whether to overwrite existing file (default: false)",
				Required:    false,
				Default:     false,
			},
		},
		Handler: t.createFile,
	})

	// Register directory creation function
	t.RegisterFunction(&toolkit.Function{
		Name:        "create_directory",
		Description: "Create a new directory",
		Parameters: map[string]toolkit.Parameter{
			"dir_path": {
				Type:        "string",
				Description: "The path where the directory should be created",
				Required:    true,
			},
		},
		Handler: t.createDirectory,
	})

	// Register template generation function
	t.RegisterFunction(&toolkit.Function{
		Name:        "generate_from_template",
		Description: "Generate content from a template with variables",
		Parameters: map[string]toolkit.Parameter{
			"template": {
				Type:        "string",
				Description: "The template string with {{variable}} placeholders",
				Required:    true,
			},
			"variables": {
				Type:        "object",
				Description: "Variables to substitute in the template",
				Required:    false,
				Default:     map[string]interface{}{},
			},
		},
		Handler: t.generateFromTemplate,
	})

	return t
}

// createFile creates a new file with specified content
func (f *FileGenToolkit) createFile(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	filePath, ok := args["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path must be a string")
	}

	content, ok := args["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content must be a string")
	}

	overwrite := false
	if overwriteArg, ok := args["overwrite"].(bool); ok {
		overwrite = overwriteArg
	}

	// Check if file already exists
	if _, err := os.Stat(filePath); err == nil && !overwrite {
		return nil, fmt.Errorf("file already exists and overwrite is false: %s", filePath)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return map[string]interface{}{
		"file_path": filePath,
		"filepath":  filePath,
		"size":      len(content),
		"created":   true,
	}, nil
}

// createDirectory creates a new directory
func (f *FileGenToolkit) createDirectory(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	dirPath, ok := args["dir_path"].(string)
	if !ok {
		return nil, fmt.Errorf("dir_path must be a string")
	}

	// Check if directory already exists
	if _, err := os.Stat(dirPath); err == nil {
		return nil, fmt.Errorf("directory already exists: %s", dirPath)
	}

	// Create directory
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	return map[string]interface{}{
		"dir_path": dirPath,
		"created":  true,
	}, nil
}

// generateFromTemplate generates content from a template with variables
func (f *FileGenToolkit) generateFromTemplate(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	templateStr, ok := args["template"].(string)
	if !ok {
		return nil, fmt.Errorf("template must be a string")
	}

	variables := make(map[string]interface{})
	if varsArg, ok := args["variables"].(map[string]interface{}); ok {
		variables = varsArg
	}

	// Simple template substitution
	result := templateStr
	for key, value := range variables {
		placeholder := "{{" + key + "}}"
		valueStr := fmt.Sprintf("%v", value)
		result = strings.ReplaceAll(result, placeholder, valueStr)
	}

	return map[string]interface{}{
		"template":  templateStr,
		"result":    result,
		"variables": variables,
	}, nil
}
