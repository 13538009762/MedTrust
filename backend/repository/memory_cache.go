package repository

import (
	"sync"
	"time"
)

type memoryCacheItem struct {
	val      interface{}
	expireAt time.Time
}

// MemoryCache 用于在无 Redis 环境下的内存降级容灾实现 (具备惰性淘汰与后台 Ticker 定期回收机制，杜绝内存泄漏)
type MemoryCache struct {
	data sync.Map
	stop chan struct{}
}

func NewMemoryCache(cleanInterval time.Duration) *MemoryCache {
	c := &MemoryCache{
		stop: make(chan struct{}),
	}
	go func() {
		ticker := time.NewTicker(cleanInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				now := time.Now()
				c.data.Range(func(k, v interface{}) bool {
					if item, ok := v.(memoryCacheItem); ok {
						if now.After(item.expireAt) {
							c.data.Delete(k)
						}
					}
					return true
				})
			case <-c.stop:
				return
			}
		}
	}()
	return c
}

var DefaultCache = NewMemoryCache(1 * time.Minute)

func (c *MemoryCache) Set(key string, val interface{}, ttl time.Duration) {
	expireAt := time.Now().Add(ttl)
	c.data.Store(key, memoryCacheItem{val: val, expireAt: expireAt})
}

func (c *MemoryCache) Get(key string) (interface{}, bool) {
	v, ok := c.data.Load(key)
	if !ok {
		return nil, false
	}
	item, ok := v.(memoryCacheItem)
	if !ok {
		return nil, false
	}
	if time.Now().After(item.expireAt) {
		c.data.Delete(key)
		return nil, false
	}
	return item.val, true
}

func (c *MemoryCache) Delete(key string) {
	c.data.Delete(key)
}

func (c *MemoryCache) Close() {
	select {
	case <-c.stop:
	default:
		close(c.stop)
	}
}
