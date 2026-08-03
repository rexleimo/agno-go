package skills

import (
	"testing"
)

// testFS builds a fake filesystem with skills for tests.
// testFS 构建包含测试技能的伪文件系统。
func TestLoader_Load(t *testing.T) {
	l := NewLoader(testFS())
	skill, err := l.Load("skills", "web-research")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill.Metadata.Name != "web-research" {
		t.Errorf("name = %q", skill.Metadata.Name)
	}
	if !contains(skill.Body, "# Web Research") {
		t.Errorf("body missing content: %q", skill.Body)
	}

	// Read reference on demand.
	// 按需读取参考资料。
	ref, err := skill.ReadReference("web_search.md")
	if err != nil {
		t.Fatalf("read reference: %v", err)
	}
	if !contains(string(ref), "Query formulation") {
		t.Errorf("reference content mismatch: %q", string(ref))
	}

	// Path traversal must be rejected.
	// 路径穿越必须被拒绝。
	if _, err := skill.ReadReference("../../secret"); err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestLoader_List(t *testing.T) {
	l := NewLoader(testFS())
	names, err := l.List("skills")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 4 {
		t.Errorf("expected 4 dirs, got %v", names)
	}
}
