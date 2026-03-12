package domain

import (
	"context"
	"fmt"
	"time"
)

// DashboardUseCase defines the business logic interface
type DashboardUseCase interface {
	GetDashboardData(ctx context.Context) (*DashboardData, error)
	UpdateContainerStatus(ctx context.Context, containerID, status string) error
	GetContainerByID(ctx context.Context, id string) (*Container, error)
}

// AppUseCase defines app lifecycle management operations.
type AppUseCase interface {
	CreateApp(ctx context.Context, name, composeYAML string) (*App, error)
	UpdateApp(ctx context.Context, id, name, composeYAML string) (*App, error)
	DeleteApp(ctx context.Context, id string) error
	GetApp(ctx context.Context, id string) (*App, error)
	ListApps(ctx context.Context) ([]*App, error)
	DeployApp(ctx context.Context, id string) error
	StopApp(ctx context.Context, id string) error
	RestartApp(ctx context.Context, id string) error
	GetAppStatus(ctx context.Context, id string) (string, error)
	GetAppLogs(ctx context.Context, id string, lines int) (string, error)
}

// DashboardService implements DashboardUseCase
type DashboardService struct {
	repository DashboardRepository
}

// NewDashboardService creates a new dashboard service
func NewDashboardService(repository DashboardRepository) *DashboardService {
	return &DashboardService{
		repository: repository,
	}
}

// GetDashboardData retrieves dashboard data from JSON file
func (s *DashboardService) GetDashboardData(ctx context.Context) (*DashboardData, error) {
	if s.repository == nil {
		return nil, ErrMissingRepository
	}

	dashboard, err := s.repository.LoadDashboardData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load dashboard data: %w", err)
	}

	for i := range dashboard.Containers {
		dashboard.Containers[i].Status = NormalizeStoredStatus(dashboard.Containers[i].Status)
	}
	dashboard.Stats = BuildStats(dashboard.Containers)

	return dashboard, nil
}

// UpdateContainerStatus updates the status of a specific container
func (s *DashboardService) UpdateContainerStatus(ctx context.Context, containerID, status string) error {
	if s.repository == nil {
		return ErrMissingRepository
	}

	canonicalStatus, valid := ParseStatusForUpdate(status)
	if !valid {
		return fmt.Errorf("%w: %s", ErrInvalidContainerStatus, status)
	}

	dashboard, err := s.GetDashboardData(ctx)
	if err != nil {
		return err
	}

	// Find and update container
	found := false
	for i := range dashboard.Containers {
		if dashboard.Containers[i].ID == containerID {
			dashboard.Containers[i].Status = canonicalStatus
			dashboard.Containers[i].LastUpdated = time.Now().UTC()
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("%w: %s", ErrContainerNotFound, containerID)
	}

	dashboard.Stats = BuildStats(dashboard.Containers)

	// Save updated data
	return s.repository.SaveDashboardData(ctx, dashboard)
}

// GetContainerByID retrieves a specific container by ID
func (s *DashboardService) GetContainerByID(ctx context.Context, id string) (*Container, error) {
	dashboard, err := s.GetDashboardData(ctx)
	if err != nil {
		return nil, err
	}

	for _, container := range dashboard.Containers {
		if container.ID == id {
			return &container, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrContainerNotFound, id)
}
