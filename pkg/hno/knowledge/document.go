package knowledge

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Document represents a document with metadata
type Document struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Source   string                 `json:"source,omitempty"` // File path or URL
}

// Chunk represents a chunk of a document
type Chunk struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Index    int                    `json:"index"` // Position in original document
}

// Loader interface for loading documents from different sources
type Loader interface {
	Load() ([]Document, error)
}

// TextLoader loads documents from text files
type TextLoader struct {
	FilePath string
}

// NewTextLoader creates a new text file loader
func NewTextLoader(filePath string) *TextLoader {
	return &TextLoader{FilePath: filePath}
}

// Load loads a text file
func (l *TextLoader) Load() ([]Document, error) {
	content, err := os.ReadFile(l.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", l.FilePath, err)
	}

	doc := Document{
		ID:      filepath.Base(l.FilePath),
		Content: string(content),
		Source:  l.FilePath,
		Metadata: map[string]interface{}{
			"filename": filepath.Base(l.FilePath),
			"path":     l.FilePath,
			"ext":      filepath.Ext(l.FilePath),
		},
	}

	return []Document{doc}, nil
}

// DirectoryLoader loads documents from a directory
type DirectoryLoader struct {
	DirPath    string
	Pattern    string // File pattern to match (e.g., "*.txt", "*.md")
	Recursive  bool   // Whether to search subdirectories
	extensions map[string]bool
}

// NewDirectoryLoader creates a new directory loader
func NewDirectoryLoader(dirPath string, pattern string, recursive bool) *DirectoryLoader {
	return &DirectoryLoader{
		DirPath:   dirPath,
		Pattern:   pattern,
		Recursive: recursive,
	}
}

// Load loads all matching files from a directory
func (l *DirectoryLoader) Load() ([]Document, error) {
	var documents []Document

	// Parse pattern to get extensions
	if l.Pattern != "" && strings.Contains(l.Pattern, "*") {
		ext := strings.TrimPrefix(l.Pattern, "*")
		l.extensions = map[string]bool{ext: true}
	}

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			if !l.Recursive && path != l.DirPath {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file matches pattern
		if l.extensions != nil {
			ext := filepath.Ext(path)
			if !l.extensions[ext] {
				return nil
			}
		}

		// Load file
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		doc := Document{
			ID:      filepath.Base(path),
			Content: string(content),
			Source:  path,
			Metadata: map[string]interface{}{
				"filename": filepath.Base(path),
				"path":     path,
				"ext":      filepath.Ext(path),
			},
		}

		documents = append(documents, doc)
		return nil
	}

	err := filepath.Walk(l.DirPath, walkFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", l.DirPath, err)
	}

	return documents, nil
}

// ReaderLoader loads documents from an io.Reader
type ReaderLoader struct {
	Reader   io.Reader
	ID       string
	Metadata map[string]interface{}
}

// NewReaderLoader creates a new reader loader
func NewReaderLoader(reader io.Reader, id string, metadata map[string]interface{}) *ReaderLoader {
	return &ReaderLoader{
		Reader:   reader,
		ID:       id,
		Metadata: metadata,
	}
}

// Load loads content from a reader
func (l *ReaderLoader) Load() ([]Document, error) {
	content, err := io.ReadAll(l.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	doc := Document{
		ID:       l.ID,
		Content:  string(content),
		Metadata: l.Metadata,
	}

	return []Document{doc}, nil
}
