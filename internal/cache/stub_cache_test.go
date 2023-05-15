package cache

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hungdv136/rio"
	"github.com/hungdv136/rio/internal/config"
	"github.com/hungdv136/rio/internal/test/mock"
	"github.com/stretchr/testify/require"
)

func TestStubCache_GetAll(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	cfg := config.NewConfig()
	namespace := uuid.NewString()
	last := &rio.LastUpdatedRecord{}
	stub := rio.NewStub().For("POST", rio.Contains("animal/create")).WithNamespace(namespace)

	statusStore := mock.NewMockStatusStore(ctrl)
	statusStore.EXPECT().GetLastUpdatedStub(gomock.Any(), namespace).Return(last, nil).Times(2)

	stubStore := mock.NewMockStubStore(ctrl)
	stubStore.EXPECT().GetAll(gomock.Any(), namespace).Return([]*rio.Stub{stub}, nil).Times(1)

	cache := NewStubCache(stubStore, statusStore, cfg)
	allStubs, err := cache.GetAll(ctx, namespace)
	require.NoError(t, err)
	require.Len(t, allStubs, 1)
	require.Equal(t, stub.ID, allStubs[0].ID)

	// Assure that no request to db anymore
	allStubs, err = cache.GetAll(ctx, namespace)
	require.NoError(t, err)
	require.Len(t, allStubs, 1)
	require.Equal(t, stub.ID, allStubs[0].ID)
}

func TestStubCache_GetProtos(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	cfg := config.NewConfig()
	last := &rio.LastUpdatedRecord{}
	proto := &rio.Proto{
		Name:    uuid.NewString(),
		FileID:  uuid.NewString(),
		Methods: []string{uuid.NewString(), uuid.NewString()},
	}

	statusStore := mock.NewMockStatusStore(ctrl)
	statusStore.EXPECT().GetLastUpdatedProto(gomock.Any()).Return(last, nil).Times(2)

	stubStore := mock.NewMockStubStore(ctrl)
	stubStore.EXPECT().GetProtos(gomock.Any()).Return([]*rio.Proto{proto}, nil).Times(1)

	cache := NewStubCache(stubStore, statusStore, cfg)

	gotProtos, err := cache.GetProtos(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, gotProtos)

	for _, p := range gotProtos {
		if p.ID == proto.ID {
			require.Equal(t, proto.Name, p.Name)
			require.Equal(t, proto.FileID, p.FileID)
			require.Equal(t, proto.Methods, p.Methods)
		}
	}

	// Assure that no request to db anymore
	gotProtos, err = cache.GetProtos(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, gotProtos)
}
