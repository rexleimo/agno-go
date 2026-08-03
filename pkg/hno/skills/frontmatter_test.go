package skills

import (
	"testing"
)

// testFS builds a fake filesystem with skills for tests.
// testFS 构建包含测试技能的伪文件系统。
func TestParseMetadata_Valid(t *testing.T) {
	meta, err := ParseMetadata([]byte(`---
name: web-research
description: Research a topic.
license: MIT
---
Body`), "web-research")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Name != "web-research" || meta.Description != "Research a topic." || meta.License != "MIT" {
		t.Errorf("unexpected metadata: %+v", meta)
	}
}

func TestParseMetadata_InvalidName(t *testing.T) {
	cases := []struct {
		name string
		dir  string
	}{
		{"Uppercase", "Uppercase"}, // uppercase not allowed
		{"has--dash", "has--dash"}, // "--" not allowed
		{"has space", "has space"}, // space not allowed
		{"toolong1234567890123456789012345678901234567890123456789012345678901234567890", "toolong1234567890123456789012345678901234567890123456789012345678901234567890"}, // >64 chars
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseMetadata([]byte("---\nname: "+tt.name+"\ndescription: test\n---\n"), tt.dir)
			if err == nil {
				t.Errorf("expected error for name %q", tt.name)
			}
		})
	}
}

func TestParseMetadata_NameMismatch(t *testing.T) {
	_, err := ParseMetadata([]byte("---\nname: other\n---\n"), "mydir")
	if err == nil {
		t.Error("expected error for name mismatch")
	}
}

func TestParseMetadata_MissingDescription(t *testing.T) {
	_, err := ParseMetadata([]byte("---\nname: test\n---\n"), "test")
	if err == nil {
		t.Error("expected error for missing description")
	}
}

func TestParseMetadata_MissingFrontmatter(t *testing.T) {
	_, err := ParseMetadata([]byte("no frontmatter here"), "test")
	if err == nil {
		t.Error("expected error for missing frontmatter")
	}
}
