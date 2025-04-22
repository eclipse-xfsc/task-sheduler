package health

import (
	"context"

	"github.com/eclipse-xfsc/task-sheduler/gen/health"
)

type Service struct {
	version string
}

func New(version string) *Service {
	return &Service{version: version}
}

func (s *Service) Liveness(_ context.Context) (*health.HealthResponse, error) {
	return &health.HealthResponse{
		Service: "task",
		Status:  "up",
		Version: s.version,
	}, nil
}

func (s *Service) Readiness(_ context.Context) (*health.HealthResponse, error) {
	return &health.HealthResponse{
		Service: "task",
		Status:  "up",
		Version: s.version,
	}, nil
}
