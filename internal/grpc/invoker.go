package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/hungdv136/rio/internal/log"
	"github.com/hungdv136/rio/internal/util"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func invokeGrpc(ctx context.Context, serverAddr string, m *desc.MethodDescriptor, input *dynamic.Message, opts ...grpc.CallOption) (*dynamic.Message, error) {
	connection, close, err := newConnection(ctx, serverAddr)
	if err != nil {
		return nil, err
	}

	// TODO: Keep connection to reuse for many requests
	// Assume that the number of request is not too high,
	// so it's acceptable to open connection for each request
	defer util.CloseSilently(ctx, close)

	output := dynamic.NewMessage(m.GetOutputType())
	if err := connection.Invoke(ctx, getFullMethod(m), input, output, opts...); err != nil {
		log.Error(ctx, err)
		return nil, err
	}

	return output, nil
}

func newConnection(ctx context.Context, addr string) (grpc.ClientConnInterface, func() error, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// TODO: Ability to customize transportation security option
	conn, err := grpc.DialContext(ctx, addr, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error(ctx, err)
		return nil, nil, err
	}

	return conn, conn.Close, nil
}

func getFullMethod(m *desc.MethodDescriptor) string {
	return fmt.Sprintf("/%s/%s", m.GetService().GetFullyQualifiedName(), m.GetName())
}
