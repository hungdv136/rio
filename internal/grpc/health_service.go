package grpc

import (
	"context"

	health "google.golang.org/grpc/health/grpc_health_v1"
)

// HealthService defines health check service
type HealthService struct{}

// Check checks server status
func (s *HealthService) Check(_ context.Context, req *health.HealthCheckRequest) (*health.HealthCheckResponse, error) {
	return &health.HealthCheckResponse{Status: health.HealthCheckResponse_SERVING}, nil
}

// Watch performs a streaming health-check
func (s *HealthService) Watch(_ *health.HealthCheckRequest, server health.Health_WatchServer) error {
	status := &health.HealthCheckResponse{Status: health.HealthCheckResponse_SERVING}
	return server.Send(status)
}
