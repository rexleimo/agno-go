package store

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	// MaxNamespaceDepth 与 MaxSegmentLen 是 ADJ-2=A 的段规则取值（非空 + 合法 UTF-8 + 不含保留分隔符 + 段长/深度上限），属实现承诺。
	MaxNamespaceDepth = 16
	MaxSegmentLen     = 128
	// UnitSeparator 是规范编码的保留分隔符：段内出现即拒绝，编码因此无碰撞。
	UnitSeparator = "\x1f"
)

// ValidateNamespace 校验层级 namespace：段非空、合法 UTF-8、不含保留分隔符、
// 段长 ≤ MaxSegmentLen、深度 ≤ MaxNamespaceDepth。空 namespace（根层）合法。
func ValidateNamespace(ns []string) error {
	if len(ns) > MaxNamespaceDepth {
		return fmt.Errorf("%w: depth %d > %d", ErrInvalidNamespace, len(ns), MaxNamespaceDepth)
	}
	for i, seg := range ns {
		if seg == "" {
			return fmt.Errorf("%w: empty segment at index %d", ErrInvalidNamespace, i)
		}
		if !utf8.ValidString(seg) {
			return fmt.Errorf("%w: segment %d is not valid UTF-8", ErrInvalidNamespace, i)
		}
		if len(seg) > MaxSegmentLen {
			return fmt.Errorf("%w: segment %d too long", ErrInvalidNamespace, i)
		}
		if strings.Contains(seg, UnitSeparator) {
			return fmt.Errorf("%w: segment %d contains reserved separator", ErrInvalidNamespace, i)
		}
	}
	return nil
}

// ValidateKey 校验键：非空、合法 UTF-8、不含保留分隔符。
func ValidateKey(key string) error {
	if key == "" {
		return ErrInvalidKey
	}
	if !utf8.ValidString(key) {
		return fmt.Errorf("%w: not valid UTF-8", ErrInvalidKey)
	}
	if strings.Contains(key, UnitSeparator) {
		return fmt.Errorf("%w: contains reserved separator", ErrInvalidKey)
	}
	return nil
}

// Composite 是 (ns,key) 的无碰撞规范编码：长度前缀 + 保留分隔符（D4 身份锚）。
// naive 的 '.' 压平会把 ["a.b","c"] 与 ["a","b.c"] 撞成同一串，这里不会。
func Composite(ns []string, key string) string {
	var b strings.Builder
	for _, seg := range ns {
		b.WriteString(strconv.Itoa(len(seg)))
		b.WriteString(UnitSeparator)
		b.WriteString(seg)
		b.WriteString(UnitSeparator)
	}
	b.WriteString(strconv.Itoa(len(key)))
	b.WriteString(UnitSeparator)
	b.WriteString(key)
	return b.String()
}

// namespaceEquals 是精确匹配语义（ADJ-2=A）：层级与前缀匹配留 v3.1。
func namespaceEquals(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
