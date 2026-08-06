package filegen

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rexleimo/agno-go/pkg/hno/tools/file"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// FileGenToolkit provides file generation capabilities.
type FileGenToolkit struct {
	*toolkit.BaseToolkit
	sandbox    *file.Sandbox
	sandboxErr error
}

// New creates unrestricted file generation tools for trusted environments only.
// Production agents should use NewWithSandbox.
func New() *FileGenToolkit {
	t := &FileGenToolkit{BaseToolkit: toolkit.NewBaseToolkit("file_generation")}
	t.registerFunctions()
	return t
}

// NewWithSandbox creates file generation tools confined to a sandbox write root.
// File and directory paths must be relative to that root.
func NewWithSandbox(sandbox *file.Sandbox) *FileGenToolkit {
	t := New()
	if sandbox == nil {
		t.sandboxErr = fmt.Errorf("sandbox cannot be nil")
		return t
	}
	t.sandbox = sandbox
	return t
}

// Close closes the configured sandbox. When a sandbox is shared by multiple
// toolkits, close it only after all of them have finished.
func (f *FileGenToolkit) Close() error {
	if f.sandbox == nil {
		return nil
	}
	return f.sandbox.Close()
}

func (f *FileGenToolkit) sandboxConfigurationError() error {
	if f.sandboxErr == nil {
		return nil
	}
	return fmt.Errorf("file generation sandbox configuration: %w", f.sandboxErr)
}

func (t *FileGenToolkit) registerFunctions() {
	t.RegisterFunction(&toolkit.Function{
		Name:        "create_file",
		Description: "Create a new file with specified content. In sandboxed mode, file_path must be relative to the configured write root.",
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
		Description: "Create a new directory. In sandboxed mode, dir_path must be relative to the configured write root.",
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

	if err := f.sandboxConfigurationError(); err != nil {
		return nil, err
	}

	if f.sandbox != nil {
		if err := f.sandbox.CreateFile(filePath, []byte(content), 0644, overwrite); err != nil {
			return nil, fmt.Errorf("failed to create file: %w", err)
		}
	} else {
		if _, err := os.Stat(filePath); err == nil && !overwrite {
			return nil, fmt.Errorf("file already exists and overwrite is false: %s", filePath)
		}

		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write file: %w", err)
		}
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

	if err := f.sandboxConfigurationError(); err != nil {
		return nil, err
	}

	if f.sandbox != nil {
		if err := f.sandbox.CreateDirectory(dirPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	} else {
		if _, err := os.Stat(dirPath); err == nil {
			return nil, fmt.Errorf("directory already exists: %s", dirPath)
		}
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
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
