package grpc

import (
	"context"
	"net"
	"os/signal"
	"syscall"

	"github.com/hungdv136/rio"
	"github.com/hungdv136/rio/internal/log"
	fs "github.com/hungdv136/rio/internal/storage"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip" // blank import for gzip decompress
	health "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server serves grpc incoming requests
type Server struct {
	listener   net.Listener
	grpcServer *grpc.Server
}

func NewServer(stubStore rio.StubStore, fileStorage fs.FileStorage, descriptor *ServiceDescriptor) *Server {
	handler := newHandler(stubStore, fileStorage, descriptor)
	grpcServer := grpc.NewServer(grpc.UnknownServiceHandler(handler.handleRequest))
	health.RegisterHealthServer(grpcServer, &HealthService{})
	reflection.Register(grpcServer)
	return &Server{grpcServer: grpcServer}
}

// Start starts the grpc server
func (s *Server) Start(ctx context.Context, addr string) error {
	if err := s.prepareServer(ctx, addr); err != nil {
		return err
	}

	return s.serve(ctx)
}

// StartAsync starts server in a different goroutine
func (s *Server) StartAsync(ctx context.Context, addr string) error {
	if err := s.prepareServer(ctx, addr); err != nil {
		return err
	}

	go func() { _ = s.serve(ctx) }()
	return nil
}

func (s *Server) prepareServer(ctx context.Context, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error(ctx, err)
		return err
	}

	s.listener = listener
	return nil
}

func (s *Server) serve(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()

	go func() {
		<-ctx.Done()

		log.Info(ctx, "shutting down server")
		s.grpcServer.GracefulStop()
	}()

	log.Info(ctx, "starting server at address", s.listener.Addr().String())
	return s.grpcServer.Serve(s.listener)
}
