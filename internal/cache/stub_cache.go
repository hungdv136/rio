package cache

import (
	"context"

	"github.com/hungdv136/rio"
	"github.com/hungdv136/rio/internal/config"
	"github.com/hungdv136/rio/internal/log"
)

const (
	strategyAside = "aside"
)

const (
	protoKey   = "protos"
	stubPrefix = "stub_"
)

var _ rio.StubStore = (*stubCache)(nil)

func NewStubCache(sourceStore rio.StubStore, statusStore rio.StatusStore, cfg *config.Config) rio.StubStore {
	switch cfg.StubCacheStrategy {
	case strategyAside:
		return &stubAsideCache{
			StubStore:  sourceStore,
			stubCache:  newCache[rio.Stub](cfg.StubCacheTTL),
			protoCache: newCache[rio.Proto](cfg.StubCacheTTL),
		}

	default:
		return &stubCache{
			StubStore:   sourceStore,
			statusStore: statusStore,
			stubCache:   newCache[rio.Stub](cfg.StubCacheTTL),
			protoCache:  newCache[rio.Proto](cfg.StubCacheTTL),
		}
	}
}

// stubCache is a default stub cache since it reduces the stale data
// which verifies the latest update information in database and cache
// invalidate cache if there is any update(insert, update and delete) in database
type stubCache struct {
	rio.StubStore

	statusStore rio.StatusStore
	stubCache   *cache[rio.Stub]
	protoCache  *cache[rio.Proto]
}

func (s *stubCache) GetAll(ctx context.Context, namespace string) ([]*rio.Stub, error) {
	last, err := s.statusStore.GetLastUpdatedStub(ctx, namespace)
	if err != nil {
		return nil, err
	}

	// namespace can be empty, add prefix for safety
	key := stubPrefix + namespace
	isValidCache := matchWithLatestUpdate[rio.Stub](last)
	if item := s.stubCache.get(ctx, key, isValidCache); item != nil {
		return item.items, nil
	}

	log.Info(ctx, "reloading stub from db")
	stubs, err := s.StubStore.GetAll(ctx, namespace)
	if err != nil {
		return nil, err
	}

	cacheItem := &cacheItem[rio.Stub]{last: last, items: stubs}
	s.stubCache.SetDefault(key, cacheItem)

	return stubs, nil
}

func (s *stubCache) GetProtos(ctx context.Context) ([]*rio.Proto, error) {
	last, err := s.statusStore.GetLastUpdatedProto(ctx)
	if err != nil {
		return nil, err
	}

	isValidCache := matchWithLatestUpdate[rio.Proto](last)
	if item := s.protoCache.get(ctx, protoKey, isValidCache); item != nil {
		return item.items, nil
	}

	log.Info(ctx, "reloading protos from db")
	protos, err := s.StubStore.GetProtos(ctx)
	if err != nil {
		return nil, err
	}

	cacheItem := &cacheItem[rio.Proto]{last: last, items: protos}
	s.protoCache.SetDefault(protoKey, cacheItem)

	return protos, nil
}

// stubAsideCache this implements a simple cache aside pattern
// it only bases on TTL to invalidate local cache
// this strategy can be used if we want to do performance testing
// since it does not require any db connection once cache is loaded
type stubAsideCache struct {
	rio.StubStore

	stubCache  *cache[rio.Stub]
	protoCache *cache[rio.Proto]
}

func (s *stubAsideCache) GetAll(ctx context.Context, namespace string) ([]*rio.Stub, error) {
	key := stubPrefix + namespace
	if item := s.stubCache.get(ctx, key, nil); item != nil {
		return item.items, nil
	}

	log.Info(ctx, "reloading stub from db")
	stubs, err := s.StubStore.GetAll(ctx, namespace)
	if err != nil {
		return nil, err
	}

	cacheItem := &cacheItem[rio.Stub]{items: stubs}
	s.stubCache.SetDefault(key, cacheItem)

	return stubs, nil
}

func (s *stubAsideCache) GetProtos(ctx context.Context) ([]*rio.Proto, error) {
	if item := s.protoCache.get(ctx, protoKey, nil); item != nil {
		return item.items, nil
	}

	log.Info(ctx, "reloading protos from db")
	protos, err := s.StubStore.GetProtos(ctx)
	if err != nil {
		return nil, err
	}

	cacheItem := &cacheItem[rio.Proto]{items: protos}
	s.protoCache.SetDefault(protoKey, cacheItem)

	return protos, nil
}
