package media

import (
	"fmt"
	"strings"
)

// Attachment 表示外部媒体资源。
type Attachment struct {
	Type        string                 `json:"type"`
	URL         string                 `json:"url,omitempty"`
	Path        string                 `json:"path,omitempty"`
	ContentType string                 `json:"content_type,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Normalize 将多种输入形态规范化为 Attachment 列表。
// 支持以下形式：
//   - nil 或空值（返回 nil）
//   - []Attachment
//   - []*Attachment
//   - []map[string]interface{}
//   - map[string]interface{}，包含 `media` 列表或本身代表单个媒体
func Normalize(input interface{}) ([]Attachment, error) {
	if input == nil {
		return nil, nil
	}

	switch v := input.(type) {
	case []Attachment:
		return normalizeSlice(v)
	case []*Attachment:
		return normalizePtrSlice(v)
	case []map[string]interface{}:
		return normalizeMapSlice(v)
	case []interface{}:
		return normalizeInterfaceSlice(v)
	case map[string]interface{}:
		if raw, ok := v["media"]; ok {
			return Normalize(raw)
		}
		return normalizeSingleMap(v)
	default:
		return nil, fmt.Errorf("unsupported media payload type %T", input)
	}
}

func normalizeSlice(items []Attachment) ([]Attachment, error) {
	if len(items) == 0 {
		return nil, nil
	}
	normalized := make([]Attachment, len(items))
	for i, item := range items {
		if err := validate(&item); err != nil {
			return nil, fmt.Errorf("media[%d]: %w", i, err)
		}
		normalized[i] = sanitize(item)
	}
	return normalized, nil
}

func normalizePtrSlice(items []*Attachment) ([]Attachment, error) {
	if len(items) == 0 {
		return nil, nil
	}
	normalized := make([]Attachment, len(items))
	for i, item := range items {
		if item == nil {
			return nil, fmt.Errorf("media[%d]: attachment cannot be nil", i)
		}
		if err := validate(item); err != nil {
			return nil, fmt.Errorf("media[%d]: %w", i, err)
		}
		normalized[i] = sanitize(*item)
	}
	return normalized, nil
}

func normalizeMapSlice(items []map[string]interface{}) ([]Attachment, error) {
	if len(items) == 0 {
		return nil, nil
	}
	normalized := make([]Attachment, len(items))
	for i, item := range items {
		if item == nil {
			return nil, fmt.Errorf("media[%d]: entry cannot be nil", i)
		}
		attachment, err := mapToAttachment(item)
		if err != nil {
			return nil, fmt.Errorf("media[%d]: %w", i, err)
		}
		normalized[i] = attachment
	}
	return normalized, nil
}

func normalizeInterfaceSlice(items []interface{}) ([]Attachment, error) {
	if len(items) == 0 {
		return nil, nil
	}

	normalized := make([]Attachment, 0, len(items))
	for i, item := range items {
		switch val := item.(type) {
		case Attachment:
			if err := validate(&val); err != nil {
				return nil, fmt.Errorf("media[%d]: %w", i, err)
			}
			normalized = append(normalized, sanitize(val))
		case *Attachment:
			if val == nil {
				return nil, fmt.Errorf("media[%d]: attachment cannot be nil", i)
			}
			if err := validate(val); err != nil {
				return nil, fmt.Errorf("media[%d]: %w", i, err)
			}
			normalized = append(normalized, sanitize(*val))
		case map[string]interface{}:
			att, err := mapToAttachment(val)
			if err != nil {
				return nil, fmt.Errorf("media[%d]: %w", i, err)
			}
			normalized = append(normalized, att)
		default:
			return nil, fmt.Errorf("media[%d]: unsupported entry type %T", i, item)
		}
	}

	return normalized, nil
}

func normalizeSingleMap(item map[string]interface{}) ([]Attachment, error) {
	if item == nil {
		return nil, nil
	}
	attachment, err := mapToAttachment(item)
	if err != nil {
		return nil, err
	}
	return []Attachment{attachment}, nil
}

func mapToAttachment(value map[string]interface{}) (Attachment, error) {
	var att Attachment

	if rawType, ok := value["type"]; ok {
		if str, ok := rawType.(string); ok {
			att.Type = str
		}
	}

	if rawURL, ok := value["url"]; ok {
		if str, ok := rawURL.(string); ok {
			att.URL = str
		}
	}

	if rawPath, ok := value["path"]; ok {
		if str, ok := rawPath.(string); ok {
			att.Path = str
		}
	}

	if rawContentType, ok := value["content_type"]; ok {
		if str, ok := rawContentType.(string); ok {
			att.ContentType = str
		}
	}

	if rawName, ok := value["name"]; ok {
		if str, ok := rawName.(string); ok {
			att.Name = str
		}
	}

	if rawMetadata, ok := value["metadata"]; ok {
		if m, ok := rawMetadata.(map[string]interface{}); ok {
			att.Metadata = m
		}
	}

	if err := validate(&att); err != nil {
		return Attachment{}, err
	}

	return sanitize(att), nil
}

func validate(att *Attachment) error {
	if att == nil {
		return fmt.Errorf("attachment cannot be nil")
	}

	if att.Type == "" {
		return fmt.Errorf("type is required")
	}

	att.Type = strings.ToLower(strings.TrimSpace(att.Type))
	switch att.Type {
	case "image", "audio", "video", "file":
	default:
		return fmt.Errorf("unsupported type %q", att.Type)
	}

	if att.URL == "" && att.Path == "" {
		return fmt.Errorf("either url or path must be provided")
	}

	return nil
}

func sanitize(att Attachment) Attachment {
	if att.Metadata == nil {
		att.Metadata = make(map[string]interface{})
	}
	att.Type = strings.ToLower(att.Type)
	return att
}
