package content

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/mcp/protocol"
)

func TestNew(t *testing.T) {
	handler := New()
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
}

func TestHandler_ExtractText(t *testing.T) {
	handler := New()

	tests := []struct {
		name     string
		contents []protocol.Content
		want     string
	}{
		{
			name: "single text content",
			contents: []protocol.Content{
				{Type: protocol.ContentTypeText, Text: "Hello, world!"},
			},
			want: "Hello, world!",
		},
		{
			name: "multiple text contents",
			contents: []protocol.Content{
				{Type: protocol.ContentTypeText, Text: "First line"},
				{Type: protocol.ContentTypeText, Text: "Second line"},
			},
			want: "First line\nSecond line",
		},
		{
			name: "mixed content types",
			contents: []protocol.Content{
				{Type: protocol.ContentTypeText, Text: "Text content"},
				{Type: protocol.ContentTypeImage, Data: "base64data", MimeType: "image/png"},
				{Type: protocol.ContentTypeText, Text: "More text"},
			},
			want: "Text content\nMore text",
		},
		{
			name: "no text content",
			contents: []protocol.Content{
				{Type: protocol.ContentTypeImage, Data: "base64data", MimeType: "image/png"},
			},
			want: "",
		},
		{
			name:     "empty contents",
			contents: []protocol.Content{},
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.ExtractText(tt.contents)
			if got != tt.want {
				t.Errorf("ExtractText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHandler_ExtractImages(t *testing.T) {
	handler := New()

	// Create valid base64 encoded test data
	// 创建有效的 base64 编码测试数据
	testData := []byte("test image data")
	encodedData := base64.StdEncoding.EncodeToString(testData)

	tests := []struct {
		name      string
		contents  []protocol.Content
		wantCount int
		wantErr   bool
	}{
		{
			name: "single image",
			contents: []protocol.Content{
				{Type: protocol.ContentTypeImage, Data: encodedData, MimeType: "image/png"},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "multiple images",
			contents: []protocol.Content{
				{Type: protocol.ContentTypeImage, Data: encodedData, MimeType: "image/png"},
				{Type: protocol.ContentTypeImage, Data: encodedData, MimeType: "image/jpeg"},
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "mixed content types",
			contents: []protocol.Content{
				{Type: protocol.ContentTypeText, Text: "Text"},
				{Type: protocol.ContentTypeImage, Data: encodedData, MimeType: "image/png"},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "invalid base64",
			contents: []protocol.Content{
				{Type: protocol.ContentTypeImage, Data: "not-valid-base64!!!", MimeType: "image/png"},
			},
			wantCount: 0,
			wantErr:   true,
		},
		{
			name: "no images",
			contents: []protocol.Content{
				{Type: protocol.ContentTypeText, Text: "Text"},
			},
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			images, err := handler.ExtractImages(tt.contents)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractImages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(images) != tt.wantCount {
				t.Errorf("ExtractImages() returned %d images, want %d", len(images), tt.wantCount)
			}
		})
	}
}

func TestHandler_ExtractResources(t *testing.T) {
	handler := New()

	tests := []struct {
		name      string
		contents  []protocol.Content
		wantCount int
	}{
		{
			name: "single resource",
			contents: []protocol.Content{
				{Type: protocol.ContentTypeResource, URI: "file:///path/to/file", MimeType: "text/plain"},
			},
			wantCount: 1,
		},
		{
			name: "multiple resources",
			contents: []protocol.Content{
				{Type: protocol.ContentTypeResource, URI: "file:///path1", MimeType: "text/plain"},
				{Type: protocol.ContentTypeResource, URI: "file:///path2", MimeType: "application/json"},
			},
			wantCount: 2,
		},
		{
			name: "mixed content types",
			contents: []protocol.Content{
				{Type: protocol.ContentTypeText, Text: "Text"},
				{Type: protocol.ContentTypeResource, URI: "file:///path", MimeType: "text/plain"},
			},
			wantCount: 1,
		},
		{
			name: "no resources",
			contents: []protocol.Content{
				{Type: protocol.ContentTypeText, Text: "Text"},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resources := handler.ExtractResources(tt.contents)
			if len(resources) != tt.wantCount {
				t.Errorf("ExtractResources() returned %d resources, want %d", len(resources), tt.wantCount)
			}
		})
	}
}

func TestHandler_CreateTextContent(t *testing.T) {
	handler := New()

	content := handler.CreateTextContent("Hello, world!")

	if content.Type != protocol.ContentTypeText {
		t.Errorf("Expected type %s, got %s", protocol.ContentTypeText, content.Type)
	}
	if content.Text != "Hello, world!" {
		t.Errorf("Expected text 'Hello, world!', got %s", content.Text)
	}
}

func TestHandler_CreateImageContent(t *testing.T) {
	handler := New()

	testData := []byte("test image data")
	content := handler.CreateImageContent(testData, "image/png")

	if content.Type != protocol.ContentTypeImage {
		t.Errorf("Expected type %s, got %s", protocol.ContentTypeImage, content.Type)
	}
	if content.MimeType != "image/png" {
		t.Errorf("Expected mime type 'image/png', got %s", content.MimeType)
	}

	// Verify data is base64 encoded
	// 验证数据是 base64 编码
	decoded, err := base64.StdEncoding.DecodeString(content.Data)
	if err != nil {
		t.Errorf("Failed to decode base64 data: %v", err)
	}
	if string(decoded) != string(testData) {
		t.Errorf("Decoded data doesn't match original")
	}
}

func TestHandler_CreateResourceContent(t *testing.T) {
	handler := New()

	content := handler.CreateResourceContent("file:///path/to/file", "text/plain")

	if content.Type != protocol.ContentTypeResource {
		t.Errorf("Expected type %s, got %s", protocol.ContentTypeResource, content.Type)
	}
	if content.URI != "file:///path/to/file" {
		t.Errorf("Expected URI 'file:///path/to/file', got %s", content.URI)
	}
	if content.MimeType != "text/plain" {
		t.Errorf("Expected mime type 'text/plain', got %s", content.MimeType)
	}
}

func TestHandler_FormatAsString(t *testing.T) {
	handler := New()

	testData := []byte("image")
	encodedData := base64.StdEncoding.EncodeToString(testData)

	contents := []protocol.Content{
		{Type: protocol.ContentTypeText, Text: "Text content"},
		{Type: protocol.ContentTypeImage, Data: encodedData, MimeType: "image/png"},
		{Type: protocol.ContentTypeResource, URI: "file:///path", MimeType: "text/plain"},
	}

	result := handler.FormatAsString(contents)

	if !strings.Contains(result, "Text content") {
		t.Error("Expected result to contain 'Text content'")
	}
	if !strings.Contains(result, "[Image") {
		t.Error("Expected result to contain '[Image'")
	}
	if !strings.Contains(result, "[Resource:") {
		t.Error("Expected result to contain '[Resource:'")
	}
}

func TestHandler_ValidateContent(t *testing.T) {
	handler := New()

	testData := []byte("test")
	encodedData := base64.StdEncoding.EncodeToString(testData)

	tests := []struct {
		name    string
		content protocol.Content
		wantErr bool
	}{
		{
			name:    "valid text content",
			content: protocol.Content{Type: protocol.ContentTypeText, Text: "Hello"},
			wantErr: false,
		},
		{
			name:    "invalid text content - empty",
			content: protocol.Content{Type: protocol.ContentTypeText, Text: ""},
			wantErr: true,
		},
		{
			name:    "valid image content",
			content: protocol.Content{Type: protocol.ContentTypeImage, Data: encodedData, MimeType: "image/png"},
			wantErr: false,
		},
		{
			name:    "invalid image content - no data",
			content: protocol.Content{Type: protocol.ContentTypeImage, Data: "", MimeType: "image/png"},
			wantErr: true,
		},
		{
			name:    "invalid image content - no mime type",
			content: protocol.Content{Type: protocol.ContentTypeImage, Data: encodedData, MimeType: ""},
			wantErr: true,
		},
		{
			name:    "invalid image content - bad base64",
			content: protocol.Content{Type: protocol.ContentTypeImage, Data: "not-base64!!!", MimeType: "image/png"},
			wantErr: true,
		},
		{
			name:    "valid resource content",
			content: protocol.Content{Type: protocol.ContentTypeResource, URI: "file:///path"},
			wantErr: false,
		},
		{
			name:    "invalid resource content - no URI",
			content: protocol.Content{Type: protocol.ContentTypeResource, URI: ""},
			wantErr: true,
		},
		{
			name:    "unknown content type",
			content: protocol.Content{Type: "unknown"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.ValidateContent(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateContent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_MergeContents(t *testing.T) {
	handler := New()

	contents1 := []protocol.Content{
		{Type: protocol.ContentTypeText, Text: "First"},
	}
	contents2 := []protocol.Content{
		{Type: protocol.ContentTypeText, Text: "Second"},
	}
	contents3 := []protocol.Content{
		{Type: protocol.ContentTypeText, Text: "Third"},
	}

	merged := handler.MergeContents(contents1, contents2, contents3)

	if len(merged) != 3 {
		t.Errorf("Expected 3 contents, got %d", len(merged))
	}

	if merged[0].Text != "First" || merged[1].Text != "Second" || merged[2].Text != "Third" {
		t.Error("Contents not merged in correct order")
	}
}

func TestHandler_FilterByType(t *testing.T) {
	handler := New()

	testData := []byte("test")
	encodedData := base64.StdEncoding.EncodeToString(testData)

	contents := []protocol.Content{
		{Type: protocol.ContentTypeText, Text: "Text 1"},
		{Type: protocol.ContentTypeImage, Data: encodedData, MimeType: "image/png"},
		{Type: protocol.ContentTypeText, Text: "Text 2"},
		{Type: protocol.ContentTypeResource, URI: "file:///path"},
	}

	tests := []struct {
		name        string
		contentType string
		wantCount   int
	}{
		{
			name:        "filter text",
			contentType: protocol.ContentTypeText,
			wantCount:   2,
		},
		{
			name:        "filter image",
			contentType: protocol.ContentTypeImage,
			wantCount:   1,
		},
		{
			name:        "filter resource",
			contentType: protocol.ContentTypeResource,
			wantCount:   1,
		},
		{
			name:        "filter non-existent type",
			contentType: "non-existent",
			wantCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := handler.FilterByType(contents, tt.contentType)
			if len(filtered) != tt.wantCount {
				t.Errorf("FilterByType() returned %d items, want %d", len(filtered), tt.wantCount)
			}
			for _, c := range filtered {
				if c.Type != tt.contentType {
					t.Errorf("Filtered content has wrong type: %s, want %s", c.Type, tt.contentType)
				}
			}
		})
	}
}
