package observability

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
)

// CacheEntry is a cached response for an exact-match prompt.
// CacheEntry 是精确匹配提示词的缓存响应。
type CacheEntry struct {
	Response string
	Metadata map[string]string
}

// PromptCache is a bounded, thread-safe exact-match prompt cache
// (LRU eviction). Use it to short-circuit repeated identical requests.
//
// PromptCache 是有界、线程安全的精确匹配提示词缓存（LRU 淘汰）。
// 用它短路重复的相同请求。
type PromptCache struct {
	mu       sync.Mutex
	capacity int
	items    map[string]*pce
	order    []string // simple FIFO order for eviction / 简单的 FIFO 淘汰顺序
}

type pce struct {
	entry CacheEntry
}

// NewPromptCache creates a cache with the given capacity.
// NewPromptCache 创建给定容量的缓存。
func NewPromptCache(capacity int) *PromptCache {
	if capacity <= 0 {
		capacity = 1000
	}
	return &PromptCache{
		capacity: capacity,
		items:    make(map[string]*pce),
		order:    make([]string, 0, capacity),
	}
}

// Key derives a cache key from the prompt (SHA-256 of the joined text).
// Key 从提示词派生缓存键（拼接文本的 SHA-256）。
func (c *PromptCache) Key(prompt string) string {
	sum := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(sum[:])
}

// Get returns the cached entry for key, if present.
// Get 返回 key 的缓存条目（如有）。
func (c *PromptCache) Get(key string) (CacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[key]
	if !ok {
		return CacheEntry{}, false
	}
	return e.entry, true
}

// Put stores an entry under key, evicting oldest when full.
// Put 在 key 下存储条目，满时淘汰最旧。
func (c *PromptCache) Put(key string, entry CacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.items[key]; ok {
		c.items[key].entry = entry
		return
	}
	c.items[key] = &pce{entry: entry}
	c.order = append(c.order, key)
	if len(c.order) > c.capacity {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.items, oldest)
	}
}

// Len returns the number of cached entries.
// Len 返回缓存条目数。
func (c *PromptCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.items)
}
