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

// FileTools provides file operation capabilities.
type FileTools struct {
	*toolkit.BaseToolkit
	sandbox    *Sandbox
	sandboxErr error
	pathMapper func(string) (string, error)
}

// New creates unrestricted file tools for trusted environments only. Production
// agents should use NewWithSandbox to grant explicit read and write roots.
func New() *FileTools {
	return newFileTools(nil, nil, nil)
}

// NewWithSandbox creates file tools confined to the supplied sandbox. Agent
// paths are interpreted as names relative to the configured sandbox roots.
func NewWithSandbox(sandbox *Sandbox) *FileTools {
	if sandbox == nil {
		return newFileTools(nil, fmt.Errorf("sandbox cannot be nil"), nil)
	}
	return newFileTools(sandbox, nil, nil)
}

// NewWithBaseDir creates file tools with both read and write access confined to
// baseDir. It preserves legacy support for absolute in-root paths and overwrite
// behavior while adding symlink-safe root confinement and bounded I/O.
func NewWithBaseDir(baseDir string) *FileTools {
	sandbox, err := NewSandbox(
		WithReadRoots(baseDir),
		WithWriteRoot(baseDir),
		WithAllowOverwrite(true),
	)
	if err != nil {
		return newFileTools(nil, err, nil)
	}
	pathMapper, err := newBaseDirPathMapper(baseDir)
	if err != nil {
		_ = sandbox.Close()
		return newFileTools(nil, err, nil)
	}
	return newFileTools(sandbox, nil, pathMapper)
}

// Close releases resources held by the configured sandbox.
func (ft *FileTools) Close() error {
	if ft.sandbox == nil {
		return nil
	}
	return ft.sandbox.Close()
}

func (ft *FileTools) sandboxConfigurationError() error {
	if ft.sandboxErr == nil {
		return nil
	}
	return fmt.Errorf("file sandbox configuration: %w", ft.sandboxErr)
}

func (ft *FileTools) sandboxPath(path string) (string, error) {
	if err := ft.sandboxConfigurationError(); err != nil {
		return "", err
	}
	if ft.sandbox == nil || ft.pathMapper == nil {
		return path, nil
	}
	mapped, err := ft.pathMapper(path)
	if err != nil {
		return "", fmt.Errorf("map file path to sandbox root: %w", err)
	}
	return mapped, nil
}

func newBaseDirPathMapper(baseDir string) (func(string) (string, error), error) {
	base, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("resolve base directory: %w", err)
	}
	return func(path string) (string, error) {
		if !filepath.IsAbs(path) {
			return path, nil
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("resolve file path: %w", err)
		}
		relative, err := filepath.Rel(base, absolute)
		if err != nil {
			return "", fmt.Errorf("make file path relative to base directory: %w", err)
		}
		if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("path %q is outside the base directory", path)
		}
		return relative, nil
	}, nil
}

func newFileTools(sandbox *Sandbox, sandboxErr error, pathMapper func(string) (string, error)) *FileTools {
	ft := &FileTools{
		BaseToolkit: toolkit.NewBaseToolkit("file_operations"),
		sandbox:     sandbox,
		sandboxErr:  sandboxErr,
		pathMapper:  pathMapper,
	}
	ft.registerFunctions()
	return ft
}

func (ft *FileTools) registerFunctions() {
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

}

func (ft *FileTools) readFile(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}
	sandboxPath, err := ft.sandboxPath(path)
	if err != nil {
		return nil, err
	}

	var content []byte
	if ft.sandbox != nil {
		content, err = ft.sandbox.ReadFile(sandboxPath)
	} else {
		content, err = os.ReadFile(path)
	}
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
	sandboxPath, err := ft.sandboxPath(path)
	if err != nil {
		return nil, err
	}

	if ft.sandbox != nil {
		if err := ft.sandbox.WriteFile(sandboxPath, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write file: %w", err)
		}
	} else {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write file: %w", err)
		}
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
	sandboxPath, err := ft.sandboxPath(path)
	if err != nil {
		return nil, err
	}

	var entries []os.DirEntry
	if ft.sandbox != nil {
		entries, err = ft.sandbox.ReadDir(sandboxPath)
	} else {
		entries, err = os.ReadDir(path)
	}
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
	sandboxPath, err := ft.sandboxPath(path)
	if err != nil {
		return nil, err
	}

	if ft.sandbox != nil {
		err = ft.sandbox.DeleteFile(sandboxPath)
	} else {
		err = os.Remove(path)
	}
	if err != nil {
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
	sandboxPath, err := ft.sandboxPath(path)
	if err != nil {
		return map[string]interface{}{
			"path":   path,
			"exists": false,
			"error":  err.Error(),
		}, nil
	}

	var info os.FileInfo
	if ft.sandbox != nil {
		info, err = ft.sandbox.Stat(sandboxPath)
	} else {
		info, err = os.Stat(path)
	}
	if err != nil {
		return map[string]interface{}{
			"path":   path,
			"exists": false,
			"error":  err.Error(),
		}, nil
	}

	return map[string]interface{}{
		"path":   path,
		"exists": true,
		"is_dir": info.IsDir(),
		"size":   info.Size(),
	}, nil
}

func (ft *FileTools) readPPTX(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}
	sandboxPath, err := ft.sandboxPath(path)
	if err != nil {
		return nil, err
	}

	if ft.sandbox != nil {
		file, relative, info, err := ft.sandbox.openReadFile(sandboxPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open pptx: %w", err)
		}
		defer file.Close()

		reader, err := zip.NewReader(file, info.Size())
		if err != nil {
			return nil, fmt.Errorf("failed to open pptx: %w", err)
		}
		result, err := readPPTXReader(path, reader)
		if err == nil {
			ft.sandbox.record("read_pptx", relative, info.Size())
		}
		return result, err
	}

	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open pptx: %w", err)
	}
	defer reader.Close()
	return readPPTXReader(path, &reader.Reader)
}

const (
	maxPPTXSlideBytes int64 = 1 << 20  // 1 MiB
	maxPPTXTotalBytes int64 = 10 << 20 // 10 MiB
	maxPPTXSlides           = 1000
)

func readPPTXReader(path string, reader *zip.Reader) (map[string]interface{}, error) {
	type slideFile struct {
		name string
		file *zip.File
	}

	var slideFiles []slideFile
	for _, entry := range reader.File {
		if strings.HasPrefix(entry.Name, "ppt/slides/slide") && strings.HasSuffix(entry.Name, ".xml") {
			slideFiles = append(slideFiles, slideFile{name: entry.Name, file: entry})
		}
	}
	if len(slideFiles) == 0 {
		return nil, fmt.Errorf("no slide xml files found in pptx")
	}
	if len(slideFiles) > maxPPTXSlides {
		return nil, fmt.Errorf("pptx contains more than %d slides", maxPPTXSlides)
	}

	sort.Slice(slideFiles, func(i, j int) bool {
		return slideFiles[i].name < slideFiles[j].name
	})

	slides := make([]map[string]interface{}, 0, len(slideFiles))
	var totalBytes int64
	for index, entry := range slideFiles {
		remaining := maxPPTXTotalBytes - totalBytes
		if remaining <= 0 {
			return nil, fmt.Errorf("pptx extracted text exceeds %d bytes", maxPPTXTotalBytes)
		}
		limit := maxPPTXSlideBytes
		if remaining < limit {
			limit = remaining
		}

		content, err := readPPTXSlide(entry.file, limit)
		if err != nil {
			return nil, fmt.Errorf("read slide %s: %w", entry.name, err)
		}
		totalBytes += int64(len(content))
		slides = append(slides, map[string]interface{}{
			"index": index,
			"name":  entry.name,
			"text":  extractSlideText(content),
		})
	}

	return map[string]interface{}{
		"path":   path,
		"slides": slides,
		"count":  len(slides),
	}, nil
}

func readPPTXSlide(file *zip.File, limit int64) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	content, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > limit {
		return nil, fmt.Errorf("slide exceeds %d bytes", limit)
	}
	return content, nil
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
