package health

import (
	"context"

	ssov1 "github.com/CheckEZ/protos/gen/go/sso"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Health interface {
	Check(ctx context.Context) error
	Ready(ctx context.Context) error
}

type ServerAPI struct {
	ssov1.UnimplementedHealthServer
	health Health
}

func Register(gRPC *grpc.Server, health Health) {
	ssov1.RegisterHealthServer(gRPC, &ServerAPI{health: health})
}

func (s *ServerAPI) Check(ctx context.Context, req *ssov1.HealthCheckRequest) (*ssov1.HealthCheckResponse, error) {
	err := s.health.Check(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &ssov1.HealthCheckResponse{}, nil
}

func (s *ServerAPI) Ready(ctx context.Context, req *ssov1.HealthReadyRequest) (*ssov1.HealthReadyResponse, error) {
	err := s.health.Ready(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &ssov1.HealthReadyResponse{}, nil
}
