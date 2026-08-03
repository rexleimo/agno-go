package skills

import (
	"container/list"
)

// lruCache is a minimal LRU cache for activated skills.
// lruCache 是激活技能的最小 LRU 缓存。
type lruCache struct {
	capacity int
	items    map[string]*list.Element
	order    *list.List
}

type cacheEntry struct {
	key   string
	skill *Skill
}

func newLRUCache(capacity int) *lruCache {
	return &lruCache{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		order:    list.New(),
	}
}

func (c *lruCache) get(key string) (*Skill, bool) {
	elem, ok := c.items[key]
	if !ok {
		return nil, false
	}
	c.order.MoveToFront(elem)
	return elem.Value.(*cacheEntry).skill, true
}

func (c *lruCache) put(key string, skill *Skill) {
	if elem, ok := c.items[key]; ok {
		elem.Value.(*cacheEntry).skill = skill
		c.order.MoveToFront(elem)
		return
	}
	elem := c.order.PushFront(&cacheEntry{key: key, skill: skill})
	c.items[key] = elem
	if c.order.Len() > c.capacity {
		oldest := c.order.Back()
		if oldest != nil {
			c.order.Remove(oldest)
			delete(c.items, oldest.Value.(*cacheEntry).key)
		}
	}
}
