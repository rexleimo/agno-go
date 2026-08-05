package chromadb

import (
	"testing"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

// TestMetadataToMap_NoPanicOnUnexported verifies the regression fix: the v2
// DocumentMetadataImpl stores its map in an unexported field, so reflect
// reads must never panic the query path (they return nil instead).
//
// TestMetadataToMap_NoPanicOnUnexported 验证回归修复：v2 的
// DocumentMetadataImpl 将 map 存在未导出字段中，因此反射读取绝不能
// panic 查询路径（改为返回 nil）。
func TestMetadataToMap_NoPanicOnUnexported(t *testing.T) {
	// Build a real v2 DocumentMetadataImpl via the official constructor.
	// 通过官方构造器构建真实的 v2 DocumentMetadataImpl。
	meta := chroma.NewDocumentMetadata(
		chroma.NewStringAttribute("topic", "AI Overview"),
		chroma.NewStringAttribute("date", "2025-01-01"),
	)

	// Must not panic; returns nil because the field is unexported.
	// 绝不能 panic；因字段未导出而返回 nil。
	out := metadataToMap(meta)
	_ = out
}

// TestMetadataToMap_Nil covers nil input.
// TestMetadataToMap_Nil 覆盖 nil 输入。
func TestMetadataToMap_Nil(t *testing.T) {
	if out := metadataToMap(nil); out != nil {
		t.Errorf("nil input: got %v, want nil", out)
	}
}
