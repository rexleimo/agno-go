package skills

import (
	"fmt"
	"io/fs"
	"path"
)

// Loader loads skills from a filesystem (os.DirFS or embed.FS).
// Loader 从文件系统（os.DirFS 或 embed.FS）加载技能。
type Loader struct {
	fsys fs.FS
}

// NewLoader creates a Loader over the given filesystem.
// NewLoader 在给定文件系统上创建 Loader。
func NewLoader(fsys fs.FS) *Loader {
	return &Loader{fsys: fsys}
}

// List returns the names of skill directories under root.
// List 返回 root 下的技能目录名。
func (l *Loader) List(root string) ([]string, error) {
	entries, err := fs.ReadDir(l.fsys, root)
	if err != nil {
		return nil, fmt.Errorf("skills: read dir %q: %w", root, err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// Load reads and parses the SKILL.md of a skill directory.
// Load 读取并解析技能目录的 SKILL.md。
func (l *Loader) Load(root, name string) (*Skill, error) {
	// path.Join normalizes "./skill" to "skill"; fs.ReadFile rejects the
	// leading "./" on some platforms (Windows).
	// path.Join 将 "./skill" 规范化为 "skill"；fs.ReadFile 在某些平台
	// （Windows）会拒绝前导 "./"。
	dir := path.Join(root, name)
	data, err := fs.ReadFile(l.fsys, dir+"/"+SkillFileName)
	if err != nil {
		return nil, fmt.Errorf("skills: load %q: %w", name, err)
	}

	meta, err := ParseMetadata(data, name)
	if err != nil {
		return nil, err
	}

	return &Skill{
		Metadata: *meta,
		Dir:      dir,
		Body:     ExtractBody(data),
		fsys:     l.fsys,
	}, nil
}
