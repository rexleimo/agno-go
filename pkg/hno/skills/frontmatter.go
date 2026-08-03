package skills

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// namePattern follows the open standard: <=64 chars, lowercase letters,
// digits and hyphens only, must match the parent directory name, and must
// not contain "--".
// namePattern 遵循开放标准：<=64 字符，仅小写字母、数字和连字符，
// 必须与父目录名一致，且不能包含 "--"。
var namePattern = regexp.MustCompile(`^[a-z0-9-]{1,64}$`)

// ParseMetadata validates and parses the frontmatter of a SKILL.md file.
// dirName must be the name of the skill's parent directory; per the
// standard, the frontmatter name must equal it.
//
// ParseMetadata 校验并解析 SKILL.md 文件的 frontmatter。
// dirName 必须是技能父目录名；按标准，frontmatter 的 name 必须与其一致。
func ParseMetadata(data []byte, dirName string) (*Metadata, error) {
	body := string(data)

	// Frontmatter must be delimited by "---" lines.
	// frontmatter 必须以 "---" 行分隔。
	if !strings.HasPrefix(body, "---") {
		return nil, fmt.Errorf("skills: %s missing frontmatter", SkillFileName)
	}
	rest := strings.TrimPrefix(body, "---")
	rest = strings.TrimPrefix(rest, "\r\n")
	rest = strings.TrimPrefix(rest, "\n")

	end := strings.Index(rest, "---")
	if end == -1 {
		return nil, fmt.Errorf("skills: %s frontmatter not closed", SkillFileName)
	}
	fm := rest[:end]

	var meta Metadata
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return nil, fmt.Errorf("skills: parse frontmatter: %w", err)
	}

	// Validate name per the open standard.
	// 按开放标准校验 name。
	if !namePattern.MatchString(meta.Name) {
		return nil, fmt.Errorf("skills: invalid name %q: must be <=64 chars of [a-z0-9-], no \"--\"", meta.Name)
	}
	if strings.Contains(meta.Name, "--") {
		return nil, fmt.Errorf("skills: invalid name %q: must not contain \"--\"", meta.Name)
	}
	if meta.Name != dirName {
		return nil, fmt.Errorf("skills: name %q must match directory name %q", meta.Name, dirName)
	}
	if strings.TrimSpace(meta.Description) == "" {
		return nil, fmt.Errorf("skills: description is required")
	}
	if len(meta.Description) > 1024 {
		return nil, fmt.Errorf("skills: description exceeds 1024 chars")
	}
	return &meta, nil
}

// ExtractBody returns the SKILL.md content after the frontmatter block.
// ExtractBody 返回 frontmatter 块之后的 SKILL.md 内容。
func ExtractBody(data []byte) string {
	body := string(data)
	rest := strings.TrimPrefix(body, "---")
	rest = strings.TrimPrefix(rest, "\r\n")
	rest = strings.TrimPrefix(rest, "\n")
	end := strings.Index(rest, "---")
	if end == -1 {
		return ""
	}
	return strings.TrimPrefix(rest[end+3:], "\n")
}
