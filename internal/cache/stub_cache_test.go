package cache

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hungdv136/rio"
	"github.com/hungdv136/rio/internal/config"
	"github.com/hungdv136/rio/internal/database"
	"github.com/stretchr/testify/require"
)

func TestStubCache_GetAll(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := config.NewConfig()
	store, err := database.NewStubDBStore(ctx, cfg.DB)
	require.NoError(t, err)

	cache := NewStubCache(store, store, cfg)
	namespace := uuid.NewString()
	activeStub := rio.NewStub().For("POST", rio.Contains("animal/create")).WithNamespace(namespace)
	inactiveStub := rio.NewStub().For("POST", rio.Contains("animal/create")).WithNamespace(namespace).WithInactive()
	require.NoError(t, store.Create(ctx, activeStub, inactiveStub))

	allStubs, err := cache.GetAll(ctx, namespace)
	require.NoError(t, err)
	require.Len(t, allStubs, 1)
	require.Equal(t, activeStub.ID, allStubs[0].ID)

	allStubs, err = cache.GetAll(ctx, namespace)
	require.NoError(t, err)
	require.Len(t, allStubs, 1)
	require.Equal(t, activeStub.ID, allStubs[0].ID)
}

func TestStubCache_GetProtos(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := config.NewConfig()
	store, err := database.NewStubDBStore(ctx, cfg.DB)
	require.NoError(t, err)

	cache := NewStubCache(store, store, cfg)
	proto := &rio.Proto{
		Name:    uuid.NewString(),
		FileID:  uuid.NewString(),
		Methods: []string{uuid.NewString(), uuid.NewString()},
	}
	require.NoError(t, store.CreateProto(ctx, proto))

	createdProtos, err := cache.GetProtos(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, createdProtos)

	for _, p := range createdProtos {
		if p.ID == proto.ID {
			require.Equal(t, proto.Name, p.Name)
			require.Equal(t, proto.FileID, p.FileID)
			require.Equal(t, proto.Methods, p.Methods)
		}
	}

	createdProtos, err = cache.GetProtos(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, createdProtos)
}
