package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTextLoader(t *testing.T) {
	// Create temp file
	tmpfile, err := os.CreateTemp("", "test*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := "This is a test document.\nWith multiple lines."
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// Load document
	loader := NewTextLoader(tmpfile.Name())
	docs, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("Expected 1 document, got %d", len(docs))
	}

	doc := docs[0]
	if doc.Content != content {
		t.Errorf("Content = %q, want %q", doc.Content, content)
	}

	if doc.Source != tmpfile.Name() {
		t.Errorf("Source = %q, want %q", doc.Source, tmpfile.Name())
	}

	if doc.Metadata["filename"] != filepath.Base(tmpfile.Name()) {
		t.Error("Metadata filename not set correctly")
	}
}

func TestDirectoryLoader(t *testing.T) {
	// Create temp directory
	tmpdir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	// Create test files
	files := []struct {
		name    string
		content string
	}{
		{"test1.txt", "Content 1"},
		{"test2.txt", "Content 2"},
		{"test3.md", "Markdown content"},
	}

	for _, f := range files {
		path := filepath.Join(tmpdir, f.name)
		if err := os.WriteFile(path, []byte(f.content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Load only .txt files
	loader := NewDirectoryLoader(tmpdir, "*.txt", false)
	docs, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(docs))
	}

	// Verify content
	for _, doc := range docs {
		if !strings.HasSuffix(doc.ID, ".txt") {
			t.Errorf("Document ID %q should end with .txt", doc.ID)
		}
	}
}

func TestReaderLoader(t *testing.T) {
	content := "Test content from reader"
	reader := strings.NewReader(content)

	loader := NewReaderLoader(reader, "test-doc", map[string]interface{}{
		"custom": "metadata",
	})

	docs, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("Expected 1 document, got %d", len(docs))
	}

	doc := docs[0]
	if doc.Content != content {
		t.Errorf("Content = %q, want %q", doc.Content, content)
	}

	if doc.ID != "test-doc" {
		t.Errorf("ID = %q, want 'test-doc'", doc.ID)
	}

	if doc.Metadata["custom"] != "metadata" {
		t.Error("Metadata not set correctly")
	}
}

func TestCharacterChunker(t *testing.T) {
	doc := Document{
		ID:      "test-doc",
		Content: "This is a test document. " + strings.Repeat("More content. ", 50),
		Source:  "test.txt",
	}

	chunker := NewCharacterChunker(100, 20)
	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("Chunk() error = %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}

	// Verify chunk properties
	for i, chunk := range chunks {
		if chunk.ID == "" {
			t.Errorf("Chunk %d has empty ID", i)
		}
		if chunk.Index != i {
			t.Errorf("Chunk %d has index %d", i, chunk.Index)
		}
		if chunk.Metadata["document_id"] != doc.ID {
			t.Errorf("Chunk %d missing document_id metadata", i)
		}
		if len(chunk.Content) > 100+20 { // Allow for overlap
			t.Errorf("Chunk %d exceeds max size: %d", i, len(chunk.Content))
		}
	}

	// Verify overlap
	if len(chunks) > 1 {
		// Check that consecutive chunks overlap
		for i := 0; i < len(chunks)-1; i++ {
			chunk1 := chunks[i]
			chunk2 := chunks[i+1]

			// The beginning of chunk2 should appear near the end of chunk1
			overlap := findOverlap(chunk1.Content, chunk2.Content)
			if overlap == 0 {
				t.Logf("Warning: No overlap found between chunk %d and %d", i, i+1)
			}
		}
	}
}

func TestSentenceChunker(t *testing.T) {
	doc := Document{
		ID: "test-doc",
		Content: "This is the first sentence. This is the second sentence. " +
			"This is the third sentence. This is the fourth sentence.",
		Source: "test.txt",
	}

	chunker := NewSentenceChunker(100, 30)
	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("Chunk() error = %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}

	// Verify chunks contain complete sentences
	for i, chunk := range chunks {
		content := chunk.Content
		if !strings.HasSuffix(content, ".") && !strings.HasSuffix(content, "!") && !strings.HasSuffix(content, "?") {
			t.Errorf("Chunk %d doesn't end with sentence terminator: %q", i, content)
		}
	}
}

func TestParagraphChunker(t *testing.T) {
	doc := Document{
		ID: "test-doc",
		Content: "This is the first paragraph.\nIt has multiple lines.\n\n" +
			"This is the second paragraph.\n\n" +
			"This is the third paragraph.",
		Source: "test.txt",
	}

	chunker := NewParagraphChunker(200)
	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("Chunk() error = %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}

	// Verify chunk metadata
	for _, chunk := range chunks {
		if chunk.Metadata["document_id"] != doc.ID {
			t.Error("Chunk missing document_id metadata")
		}
	}
}

func TestCharacterChunker_EmptyDocument(t *testing.T) {
	doc := Document{
		ID:      "empty",
		Content: "",
	}

	chunker := NewCharacterChunker(100, 20)
	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("Chunk() error = %v", err)
	}

	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for empty document, got %d", len(chunks))
	}
}

func TestCharacterChunker_ShortDocument(t *testing.T) {
	doc := Document{
		ID:      "short",
		Content: "Short content",
	}

	chunker := NewCharacterChunker(100, 20)
	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("Chunk() error = %v", err)
	}

	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk for short document, got %d", len(chunks))
	}

	if chunks[0].Content != doc.Content {
		t.Error("Short document content doesn't match")
	}
}

func TestParagraphChunker_LargeParagraph(t *testing.T) {
	// Create a very large paragraph that exceeds max chunk size
	largePara := strings.Repeat("This is a sentence. ", 100) // ~2000 chars

	doc := Document{
		ID:      "large",
		Content: largePara,
	}

	chunker := NewParagraphChunker(500)
	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("Chunk() error = %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}

	// Large paragraph should be split into multiple chunks
	if len(chunks) < 3 {
		t.Errorf("Expected multiple chunks for large paragraph, got %d", len(chunks))
	}
}

// Helper function to find overlap between two strings
func findOverlap(s1, s2 string) int {
	minLen := len(s1)
	if len(s2) < minLen {
		minLen = len(s2)
	}

	for i := minLen; i > 0; i-- {
		if strings.HasSuffix(s1, s2[:i]) {
			return i
		}
	}
	return 0
}
