package pokecache

import (
	"sync"
	"time"
)

type CacheEntry struct {
	createdAt time.Time
	val []byte
}

type Cache struct {
	cacheMap map[string]CacheEntry
	mutex sync.Mutex
	interval time.Duration
}

func NewCache(interval time.Duration) *Cache {
	c := &Cache{
		cacheMap: make(map[string]CacheEntry),
		interval: interval,
	}
	go c.reapLoop()
	return c
}

func (c *Cache) Add(key string, val []byte) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cacheMap[key] = CacheEntry{
		createdAt: time.Now(),
		val:  val,
	}
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry, ok := c.cacheMap[key]
	if !ok {
		return nil, false
	}
	return entry.val, true
}

func (c *Cache) reapLoop() {
	ticker := time.NewTicker(c.interval)
	for range ticker.C {
		c.reap()
	}
}

func (c *Cache) reap() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for key, entry := range c.cacheMap {
		if now.Sub(entry.createdAt) > c.interval {
			delete(c.cacheMap, key)
		}
	}
}
