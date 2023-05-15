package grpc

import (
	"context"
	"testing"

	"github.com/hungdv136/rio/internal/log"
	fs "github.com/hungdv136/rio/internal/storage"
	"github.com/stretchr/testify/require"
)

func TestServiceDescriptor(t *testing.T) {
	t.Parallel()

	ctx := log.SaveID(context.Background(), t.Name())
	storageCfg := fs.LocalStorageConfig{StoragePath: "../../testdata"}
	storage := fs.NewLocalStorage(storageCfg)

	sd := NewServiceDescriptor(storage)
	cleanup(t, sd)

	d, err := sd.GetDescriptor(ctx, "offer_proto")
	require.NoError(t, err)

	method, err := d.GetMethod(ctx, "/offers.v1.OfferService/ValidateOffer")
	require.NoError(t, err)
	require.NotNil(t, method)
	require.Equal(t, "ValidateOffer", method.GetName())
}

func TestDescriptor(t *testing.T) {
	t.Parallel()

	ctx := log.SaveID(context.Background(), t.Name())
	s := NewDescriptor()
	err := s.init(ctx, "../../testdata/proto")
	require.NoError(t, err)
	require.Len(t, s.sdMap, 1)

	method, err := s.GetMethod(ctx, "/offers.v1.OfferService/ValidateOffer")
	require.NoError(t, err)
	require.NotNil(t, method)
	require.Equal(t, "ValidateOffer", method.GetName())
}

func cleanup(t *testing.T, d *ServiceDescriptor) {
	cleanup := func() { _ = d.ClearCache(context.Background()) }
	cleanup()
	t.Cleanup(cleanup)
}
