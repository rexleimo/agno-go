package skills

import (
	"testing"
)

// testFS builds a fake filesystem with skills for tests.
// testFS 构建包含测试技能的伪文件系统。
func TestRegistry_CatalogSkipsInvalid(t *testing.T) {
	r, err := NewRegistry(NewLoader(testFS()), "skills")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the 2 valid skills are cataloged; invalid ones skipped.
	// 只编目 2 个有效技能；无效的被跳过。
	infos := r.Catalog()
	if len(infos) != 2 {
		t.Fatalf("expected 2 cataloged skills, got %d: %v", len(infos), infos)
	}
	if infos[0].Name != "pdf-summary" || infos[1].Name != "web-research" {
		t.Errorf("unexpected order: %v", infos)
	}
}

func TestRegistry_CatalogTextAndActivate(t *testing.T) {
	r, err := NewRegistry(NewLoader(testFS()), "skills")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Catalog text is compact (progressive disclosure).
	// 目录文本紧凑（渐进式披露）。
	text := r.CatalogText()
	if !contains(text, "web-research") || !contains(text, "Research a topic") {
		t.Errorf("catalog text incomplete: %q", text)
	}
	if len(text) > 500 {
		t.Errorf("catalog text too large (%d chars)", len(text))
	}

	// Activate loads the full body.
	// 激活加载完整正文。
	skill, err := r.Activate("web-research")
	if err != nil {
		t.Fatalf("activate: %v", err)
	}
	if !contains(skill.Body, "# Web Research") {
		t.Errorf("activated body incomplete")
	}

	// Activating unknown skill fails.
	// 激活未知技能失败。
	if _, err := r.Activate("nope"); err == nil {
		t.Error("expected error for unknown skill")
	}

	if !r.Has("web-research") || r.Has("nope") {
		t.Error("Has() incorrect")
	}
}
