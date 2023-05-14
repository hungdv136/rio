package setup

import (
	"context"

	"github.com/hungdv136/rio"
	"github.com/hungdv136/rio/internal/cache"
	"github.com/hungdv136/rio/internal/config"
	"github.com/hungdv136/rio/internal/database"
	fs "github.com/hungdv136/rio/internal/storage"
)

func ProvideStubStore(ctx context.Context, cfg *config.Config) (rio.StubStore, error) {
	db, err := database.NewStubDBStore(ctx, cfg.DB)
	if err != nil {
		return nil, err
	}

	return cache.NewStubCache(db, db, cfg), nil
}

func ProvideFileStorage(ctx context.Context, cfg *config.Config) (fs.FileStorage, error) {
	return fs.NewFileStorage(cfg.FileStorage), nil
}
