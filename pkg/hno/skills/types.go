package skills

import (
	"fmt"
	"io"
	"io/fs"
	"strings"
)

// SKILL.md file name (case-insensitive per standard).
// SKILL.md 文件名（标准不区分大小写）。
const SkillFileName = "SKILL.md"

// Metadata is the parsed frontmatter of a SKILL.md.
// Metadata 是 SKILL.md 解析后的 frontmatter。
type Metadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	License     string `yaml:"license,omitempty"`
}

// Skill is a loaded skill: metadata plus the raw SKILL.md body.
// Skill 是已加载的技能：元数据加上原始 SKILL.md 正文。
type Skill struct {
	Metadata Metadata
	// Dir is the skill directory path within its filesystem.
	// Dir 是技能在其文件系统内的目录路径。
	Dir string
	// Body is the full SKILL.md content after the frontmatter.
	// Body 是 frontmatter 之后的完整 SKILL.md 内容。
	Body string
	// fsys is the filesystem the skill was loaded from.
	// fsys 是技能加载自的文件系统。
	fsys fs.FS
}

// ReadReference loads a reference file from the skill's references/
// directory on demand. Path traversal outside the skill directory is
// rejected.
//
// ReadReference 按需加载技能 references/ 目录中的参考文件。
// 拒绝越出技能目录的路径穿越。
func (s *Skill) ReadReference(name string) ([]byte, error) {
	clean := strings.TrimPrefix(name, "/")
	if strings.Contains(clean, "..") {
		return nil, fmt.Errorf("skills: invalid reference path %q", name)
	}
	path := s.Dir + "/references/" + clean
	f, err := s.fsys.Open(path)
	if err != nil {
		return nil, fmt.Errorf("skills: open reference %q: %w", name, err)
	}
	defer f.Close()
	return io.ReadAll(f)
}
