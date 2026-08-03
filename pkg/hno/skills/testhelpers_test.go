package skills

import (
	"testing/fstest"
)

// testFS builds a fake filesystem with skills for tests.
// testFS 构建包含测试技能的伪文件系统。
func testFS() fstest.MapFS {
	return fstest.MapFS{
		"skills/web-research/SKILL.md": &fstest.MapFile{
			Data: []byte(`---
name: web-research
description: Research a topic on the web and summarize findings with sources.
---

# Web Research

Use this skill to research topics. Load references/web_search.md for the search strategy.
`),
		},
		"skills/web-research/references/web_search.md": &fstest.MapFile{
			Data: []byte("# Search Strategy\nQuery formulation guide."),
		},
		"skills/pdf-summary/SKILL.md": &fstest.MapFile{
			Data: []byte(`---
name: pdf-summary
description: Summarize PDF documents.
---

# PDF Summary

Extract text and summarize.
`),
		},
		// Invalid: name does not match directory.
		// 无效：name 与目录名不一致。
		"skills/bad-skill/SKILL.md": &fstest.MapFile{
			Data: []byte(`---
name: different-name
description: Invalid skill.
---
`),
		},
		// Invalid: missing description.
		// 无效：缺少 description。
		"skills/no-desc/SKILL.md": &fstest.MapFile{
			Data: []byte(`---
name: no-desc
---
`),
		},
	}
}

// contains reports whether s contains sub.
// contains 报告 s 是否包含 sub。
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
