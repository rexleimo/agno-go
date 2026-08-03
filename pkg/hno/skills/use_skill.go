package skills

import (
	"context"
	"fmt"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// NewUseSkillToolkit builds the use_skill tool backed by a Registry.
// It is the second level of progressive disclosure: the model sees the
// catalog (first level) and calls this tool to load a skill's full body.
//
// NewUseSkillToolkit 构建由 Registry 支撑的 use_skill 工具。
// 它是渐进式披露的第二级：模型先看到目录（第一级），再调用此工具
// 加载技能的完整正文。
func NewUseSkillToolkit(reg *Registry) *toolkit.BaseToolkit {
	tk := toolkit.NewBaseToolkit("skills")
	tk.RegisterFunction(&toolkit.Function{
		Name:        "use_skill",
		Description: "Load a skill by name and return its full instructions. Call this when a task matches an available skill.",
		Parameters: map[string]toolkit.Parameter{
			"name": {
				Type:        "string",
				Description: "The skill name to load (from the available skills list)",
				Required:    true,
			},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			name, ok := args["name"].(string)
			if !ok || name == "" {
				return "", fmt.Errorf("use_skill: name is required")
			}
			skill, err := reg.Activate(name)
			if err != nil {
				return "", fmt.Errorf("use_skill: %w", err)
			}
			return skill.Body, nil
		},
	})
	return tk
}
