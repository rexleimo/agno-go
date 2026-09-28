package store

import "errors"

// 错误分类学（D3）：未命中、参数非法、检索请求非法三类互斥可区分；后端必须把驱动错误翻译成这里的哨兵，包装后 errors.Is 仍可达。
var (
	ErrNotFound          = errors.New("store: item not found")
	ErrInvalidNamespace  = errors.New("store: invalid namespace")
	ErrInvalidKey        = errors.New("store: invalid key")
	ErrInvalidItem       = errors.New("store: invalid item")
	ErrInvalidSearch     = errors.New("store: invalid search request")
	ErrNoEmbedder        = errors.New("store: embedding function not configured")
	ErrDimensionMismatch = errors.New("store: embedding dimension mismatch")
)
