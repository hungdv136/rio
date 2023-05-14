package main

import (
	"context"

	"github.com/hungdv136/rio/internal/config"
	xgrpc "github.com/hungdv136/rio/internal/grpc"
	"github.com/hungdv136/rio/internal/setup"
)

func main() {
	ctx := context.Background()
	cfg := config.NewConfig()

	fileStore, err := setup.ProvideFileStorage(ctx, cfg)
	if err != nil {
		panic(err)
	}

	stubStore, err := setup.ProvideStubStore(ctx, cfg)
	if err != nil {
		panic(err)
	}

	service := xgrpc.NewServer(stubStore, fileStore, xgrpc.NewServiceDescriptor(fileStore))
	if err := service.Start(ctx, cfg.ServerAddress); err != nil {
		panic(err)
	}
}
