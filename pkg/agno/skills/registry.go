package skills

import (
	"sort"
	"strings"
)

// SkillInfo is the lightweight catalog entry injected into the agent
// context (progressive disclosure: ~100 tokens total).
// SkillInfo 是注入 agent 上下文的轻量目录条目
// （渐进式披露：合计约 100 tokens）。
type SkillInfo struct {
	Name        string
	Description string
}

// Registry catalogs available skills and activates them on demand.
// Registry 编目可用技能并按需激活。
type Registry struct {
	loader  *Loader
	root    string
	cache   *lruCache
	entries map[string]SkillInfo
}

// NewRegistry builds a Registry over root in the loader's filesystem and
// catalogs all skills under it (loading metadata only).
//
// NewRegistry 在 loader 文件系统的 root 上构建 Registry，并编目其下
// 所有技能（仅加载元数据）。
func NewRegistry(loader *Loader, root string) (*Registry, error) {
	names, err := loader.List(root)
	if err != nil {
		return nil, err
	}

	r := &Registry{
		loader:  loader,
		root:    root,
		cache:   newLRUCache(16),
		entries: make(map[string]SkillInfo, len(names)),
	}

	for _, name := range names {
		skill, err := loader.Load(root, name)
		if err != nil {
			// Skip malformed skills; do not fail the whole registry.
			// 跳过格式错误的技能；不使整个 registry 失败。
			continue
		}
		r.entries[name] = SkillInfo{
			Name:        skill.Metadata.Name,
			Description: skill.Metadata.Description,
		}
	}
	return r, nil
}

// Catalog returns all cataloged skills, sorted by name.
// Catalog 返回所有已编目技能，按名称排序。
func (r *Registry) Catalog() []SkillInfo {
	infos := make([]SkillInfo, 0, len(r.entries))
	for _, info := range r.entries {
		infos = append(infos, info)
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Name < infos[j].Name })
	return infos
}

// CatalogText renders the catalog as compact text for context injection.
// CatalogText 将目录渲染为紧凑文本用于上下文注入。
func (r *Registry) CatalogText() string {
	infos := r.Catalog()
	if len(infos) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("Available skills:\n")
	for _, info := range infos {
		sb.WriteString("- ")
		sb.WriteString(info.Name)
		sb.WriteString(": ")
		sb.WriteString(info.Description)
		sb.WriteString("\n")
	}
	return sb.String()
}

// Activate loads the full SKILL.md body of a skill (cached with LRU).
// Activate 加载技能的完整 SKILL.md 正文（LRU 缓存）。
func (r *Registry) Activate(name string) (*Skill, error) {
	if skill, ok := r.cache.get(name); ok {
		return skill, nil
	}

	skill, err := r.loader.Load(r.root, name)
	if err != nil {
		return nil, err
	}
	r.cache.put(name, skill)
	return skill, nil
}

// Has reports whether a skill is cataloged.
// Has 报告技能是否已编目。
func (r *Registry) Has(name string) bool {
	_, ok := r.entries[name]
	return ok
}
