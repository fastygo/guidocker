package docker

import (
	"context"
	"dashboard/domain"
)

// MockRepository implements domain.DockerRepository with no-op behavior.
// Use for UI testing without Docker (e.g. DASHBOARD_DEV_MODE=true).
var _ domain.DockerRepository = (*MockRepository)(nil)

type MockRepository struct{}

// NewMockRepository creates a mock Docker repository for dev mode.
func NewMockRepository() *MockRepository {
	return &MockRepository{}
}

func (r *MockRepository) EnsureNetwork(ctx context.Context) error {
	return nil
}

func (r *MockRepository) Deploy(ctx context.Context, app *domain.App) error {
	return nil
}

func (r *MockRepository) Stop(ctx context.Context, app *domain.App) error {
	return nil
}

func (r *MockRepository) Restart(ctx context.Context, app *domain.App) error {
	return nil
}

func (r *MockRepository) Destroy(ctx context.Context, app *domain.App) error {
	return nil
}

func (r *MockRepository) GetStatus(ctx context.Context, app *domain.App) (string, error) {
	return domain.AppStatusStopped, nil
}

func (r *MockRepository) GetLogs(ctx context.Context, app *domain.App, lines int) (string, error) {
	return "", nil
}

func (r *MockRepository) ListRunning(ctx context.Context) ([]domain.Container, error) {
	return nil, nil
}

func (r *MockRepository) ListAllContainers(ctx context.Context) ([]domain.Container, error) {
	return []domain.Container{}, nil
}

func (r *MockRepository) InspectContainers(ctx context.Context, ids []string) ([]domain.ContainerDetail, error) {
	return []domain.ContainerDetail{}, nil
}

func (r *MockRepository) ResolveContainerIP(ctx context.Context, app *domain.App) (string, error) {
	return "", domain.ErrContainerNotFound
}
