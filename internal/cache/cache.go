package cache

import (
	"context"
	"time"

	"github.com/hungdv136/rio"
	"github.com/hungdv136/rio/internal/log"
	gocache "github.com/patrickmn/go-cache"
)

// The default cache invalidation strategy
// which verifies last updated time in db with updated time in cache
// - Add new or update stub: updated time of new records are greater than cached value
// - Delete a record: Not supported yet, only update to inactive status
// - Cleanup: Database is empty, so the last record is different with store value
func matchWithLatestUpdate[R any](last *rio.LastUpdatedRecord) func(c *cacheItem[R]) bool {
	return func(c *cacheItem[R]) bool {
		return last != nil && c.last != nil && last.ID == c.last.ID && last.UpdatedAt.Unix() <= c.last.UpdatedAt.Unix()
	}
}

type cacheItem[R any] struct {
	last  *rio.LastUpdatedRecord
	items []*R
}

// Wrap gocache to provide a strong type cache using generic
type cache[R any] struct {
	*gocache.Cache
}

func newCache[R any](cacheTTL time.Duration) *cache[R] {
	return &cache[R]{Cache: gocache.New(cacheTTL, time.Minute)}
}

func (c *cache[R]) get(ctx context.Context, key string, isValidCache func(c *cacheItem[R]) bool) *cacheItem[R] {
	r, ok := c.Cache.Get(key)
	if !ok {
		return nil
	}

	cachedItem, ok := r.(*cacheItem[R])
	if !ok {
		c.Cache.Delete(key)
		log.Warn(ctx, "remove incompatible type")
		return nil
	}

	if isValidCache != nil && !isValidCache(cachedItem) {
		c.Cache.Delete(key)
		log.Info(ctx, "remove invalid cache")
		return nil
	}

	return cachedItem
}
