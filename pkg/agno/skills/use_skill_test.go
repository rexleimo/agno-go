package skills

import (
	"context"
	"testing"
)

func TestUseSkillToolkit_Activate(t *testing.T) {
	r, err := NewRegistry(NewLoader(testFS()), "skills")
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	tk := NewUseSkillToolkit(r)

	if tk.Name() != "skills" {
		t.Errorf("toolkit name = %q", tk.Name())
	}

	// Invoke the handler with a valid skill name.
	// 用有效技能名调用 handler。
	fn := tk.Functions()["use_skill"]
	if fn == nil {
		t.Fatal("use_skill function not registered")
	}
	result, err := fn.Handler(context.Background(), map[string]interface{}{"name": "web-research"})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	body, ok := result.(string)
	if !ok || !contains(body, "# Web Research") {
		t.Errorf("expected skill body, got %v", result)
	}

	// Unknown skill must error.
	// 未知技能必须报错。
	if _, err := fn.Handler(context.Background(), map[string]interface{}{"name": "nope"}); err == nil {
		t.Error("expected error for unknown skill")
	}

	// Missing name must error.
	// 缺少 name 必须报错。
	if _, err := fn.Handler(context.Background(), map[string]interface{}{}); err == nil {
		t.Error("expected error for missing name")
	}
}
