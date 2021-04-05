package storages

import (
	"github.com/qsnetwork/sds/utils/cache"
	"sync"
	"time"
)

// ServerCache
type ServerCache struct {
	Cache cache.Cache
	sync.Mutex
}

// IsOK
func (d *ServerCache) IsOK() error {
	return d.Cache.IsOK()
}

// Load
func (d *ServerCache) Load(data Data) error {
	return d.Cache.Get(data.GetCacheKey(), data)
}

// Remove
func (d *ServerCache) Remove(key string) error {
	return d.Cache.Delete(key)
}

// Store
func (d *ServerCache) Store(data Data, expire time.Duration) error {
	return d.Cache.Set(data.GetCacheKey(), data, expire)
}

// GetCache
func (d *ServerCache) GetCache() cache.Cache {
	return d.Cache
}

// NewServerCache
func NewServerCache(cache cache.Cache) *ServerCache {
	cd := new(ServerCache)
	cd.Cache = cache
	return cd
}

// Data
type Data interface {
	GetCacheKey() string
}
