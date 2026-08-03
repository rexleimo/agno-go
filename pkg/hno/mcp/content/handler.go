package content

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/rexleimo/agno-go/pkg/hno/mcp/protocol"
)

// Handler handles different types of MCP content (text, images, resources)
// Handler 处理不同类型的 MCP 内容（文本、图像、资源）
type Handler struct{}

// New creates a new content handler.
// New 创建新的内容处理器。
func New() *Handler {
	return &Handler{}
}

// ExtractText extracts text content from a list of Content items.
// Returns the concatenated text from all text-type content items.
//
// ExtractText 从内容项列表中提取文本内容。
// 返回所有文本类型内容项的连接文本。
func (h *Handler) ExtractText(contents []protocol.Content) string {
	var texts []string
	for _, content := range contents {
		if content.Type == protocol.ContentTypeText && content.Text != "" {
			texts = append(texts, content.Text)
		}
	}
	return strings.Join(texts, "\n")
}

// ExtractImages extracts image content from a list of Content items.
// Returns a slice of ImageContent with decoded image data.
//
// ExtractImages 从内容项列表中提取图像内容。
// 返回包含解码图像数据的 ImageContent 切片。
func (h *Handler) ExtractImages(contents []protocol.Content) ([]ImageContent, error) {
	var images []ImageContent
	for _, content := range contents {
		if content.Type == protocol.ContentTypeImage {
			// Decode base64 data
			// 解码 base64 数据
			data, err := base64.StdEncoding.DecodeString(content.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to decode image data: %w", err)
			}

			images = append(images, ImageContent{
				Data:     data,
				MimeType: content.MimeType,
			})
		}
	}
	return images, nil
}

// ExtractResources extracts resource content from a list of Content items.
// Returns a slice of ResourceContent.
//
// ExtractResources 从内容项列表中提取资源内容。
// 返回 ResourceContent 切片。
func (h *Handler) ExtractResources(contents []protocol.Content) []ResourceContent {
	var resources []ResourceContent
	for _, content := range contents {
		if content.Type == protocol.ContentTypeResource {
			resources = append(resources, ResourceContent{
				URI:      content.URI,
				MimeType: content.MimeType,
				Text:     content.Text,
				Resource: content.Resource,
			})
		}
	}
	return resources
}

// CreateTextContent creates a text-type Content item.
// CreateTextContent 创建文本类型的内容项。
func (h *Handler) CreateTextContent(text string) protocol.Content {
	return protocol.Content{
		Type: protocol.ContentTypeText,
		Text: text,
	}
}

// CreateImageContent creates an image-type Content item.
// The data should be raw bytes, which will be base64-encoded.
//
// CreateImageContent 创建图像类型的内容项。
// 数据应为原始字节，将进行 base64 编码。
func (h *Handler) CreateImageContent(data []byte, mimeType string) protocol.Content {
	encodedData := base64.StdEncoding.EncodeToString(data)
	return protocol.Content{
		Type:     protocol.ContentTypeImage,
		Data:     encodedData,
		MimeType: mimeType,
	}
}

// CreateResourceContent creates a resource-type Content item.
// CreateResourceContent 创建资源类型的内容项。
func (h *Handler) CreateResourceContent(uri, mimeType string) protocol.Content {
	return protocol.Content{
		Type:     protocol.ContentTypeResource,
		URI:      uri,
		MimeType: mimeType,
	}
}

// FormatAsString formats a list of Content items as a human-readable string.
// FormatAsString 将内容项列表格式化为人类可读的字符串。
func (h *Handler) FormatAsString(contents []protocol.Content) string {
	var parts []string

	for i, content := range contents {
		switch content.Type {
		case protocol.ContentTypeText:
			parts = append(parts, content.Text)
		case protocol.ContentTypeImage:
			parts = append(parts, fmt.Sprintf("[Image %d: %s, %d bytes]",
				i+1, content.MimeType, len(content.Data)))
		case protocol.ContentTypeResource:
			parts = append(parts, fmt.Sprintf("[Resource: %s (%s)]",
				content.URI, content.MimeType))
		default:
			parts = append(parts, fmt.Sprintf("[Unknown content type: %s]", content.Type))
		}
	}

	return strings.Join(parts, "\n")
}

// ImageContent represents decoded image content
// ImageContent 表示解码的图像内容
type ImageContent struct {
	Data     []byte
	MimeType string
}

// ResourceContent represents resource content
// ResourceContent 表示资源内容
type ResourceContent struct {
	URI      string
	MimeType string
	Text     string
	Resource interface{}
}

// ValidateContent validates that a Content item has the required fields for its type.
// ValidateContent 验证内容项是否具有其类型所需的字段。
func (h *Handler) ValidateContent(content protocol.Content) error {
	switch content.Type {
	case protocol.ContentTypeText:
		if content.Text == "" {
			return fmt.Errorf("text content must have non-empty Text field")
		}
	case protocol.ContentTypeImage:
		if content.Data == "" {
			return fmt.Errorf("image content must have non-empty Data field")
		}
		if content.MimeType == "" {
			return fmt.Errorf("image content must have MimeType field")
		}
		// Validate base64 encoding
		// 验证 base64 编码
		if _, err := base64.StdEncoding.DecodeString(content.Data); err != nil {
			return fmt.Errorf("image Data must be valid base64: %w", err)
		}
	case protocol.ContentTypeResource:
		if content.URI == "" {
			return fmt.Errorf("resource content must have non-empty URI field")
		}
	default:
		return fmt.Errorf("unknown content type: %s", content.Type)
	}
	return nil
}

// MergeContents merges multiple Content slices into a single slice.
// MergeContents 将多个内容切片合并为单个切片。
func (h *Handler) MergeContents(contentSlices ...[]protocol.Content) []protocol.Content {
	var result []protocol.Content
	for _, contents := range contentSlices {
		result = append(result, contents...)
	}
	return result
}

// FilterByType filters Content items by type.
// FilterByType 按类型过滤内容项。
func (h *Handler) FilterByType(contents []protocol.Content, contentType string) []protocol.Content {
	var filtered []protocol.Content
	for _, content := range contents {
		if content.Type == contentType {
			filtered = append(filtered, content)
		}
	}
	return filtered
}
