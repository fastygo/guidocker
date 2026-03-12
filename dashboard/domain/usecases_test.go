package domain

import (
	"context"
	"errors"
	"testing"
)

type fakeDashboardRepository struct {
	loadData *DashboardData
	loadErr  error
	saved    *DashboardData
	saveErr  error
}

func (r *fakeDashboardRepository) LoadDashboardData(ctx context.Context) (*DashboardData, error) {
	if r.loadErr != nil {
		return nil, r.loadErr
	}

	if r.loadData == nil {
		return &DashboardData{}, nil
	}

	data := *r.loadData
	if r.loadData.Containers != nil {
		containers := make([]Container, len(r.loadData.Containers))
		copy(containers, r.loadData.Containers)
		data.Containers = containers
	}

	return &data, nil
}

func (r *fakeDashboardRepository) SaveDashboardData(ctx context.Context, dashboard *DashboardData) error {
	if r.saveErr != nil {
		return r.saveErr
	}

	saved := *dashboard
	if dashboard.Containers != nil {
		containers := make([]Container, len(dashboard.Containers))
		copy(containers, dashboard.Containers)
		saved.Containers = containers
	}

	r.saved = &saved
	return nil
}

func TestDashboardService_GetDashboardData_NormalizesStatusesAndCalculatesStats(t *testing.T) {
	repo := &fakeDashboardRepository{
		loadData: &DashboardData{
			Title: "Docker Container Dashboard",
			Containers: []Container{
				{ID: "web", Name: "nginx", Status: "stop"},
				{ID: "api", Name: "api", Status: "pause"},
				{ID: "db", Name: "postgres", Status: "start"},
				{ID: "cache", Name: "redis", Status: "restart"},
				{ID: "runner", Name: "worker", Status: "running"},
				{ID: "legacy", Name: "legacy", Status: "weird-state"},
			},
		},
	}

	service := NewDashboardService(repo)
	dashboard, err := service.GetDashboardData(context.Background())
	if err != nil {
		t.Fatalf("GetDashboardData() error = %v", err)
	}

	if got := dashboard.Stats.TotalContainers; got != 6 {
		t.Fatalf("expected total containers 6, got %d", got)
	}

	if got := dashboard.Stats.RunningContainers; got != 3 {
		t.Fatalf("expected running containers 3, got %d", got)
	}

	if got := dashboard.Stats.StoppedContainers; got != 1 {
		t.Fatalf("expected stopped containers 1, got %d", got)
	}

	if got := dashboard.Stats.PausedContainers; got != 1 {
		t.Fatalf("expected paused containers 1, got %d", got)
	}

	expectedStatuses := []string{
		"stopped",
		"paused",
		"running",
		"running",
		"running",
		"weird-state",
	}

	for i, expected := range expectedStatuses {
		if dashboard.Containers[i].Status != expected {
			t.Fatalf("container[%d].Status = %q, expected %q", i, dashboard.Containers[i].Status, expected)
		}
	}
}

func TestDashboardService_UpdateContainerStatus_UpdatesStatusAndRecomputesStats(t *testing.T) {
	repo := &fakeDashboardRepository{
		loadData: &DashboardData{
			Containers: []Container{
				{ID: "web-app-01", Name: "nginx", Status: "running"},
				{ID: "api-server-01", Name: "api", Status: "stopped"},
			},
		},
	}

	service := NewDashboardService(repo)
	if err := service.UpdateContainerStatus(context.Background(), "api-server-01", "pause"); err != nil {
		t.Fatalf("UpdateContainerStatus() error = %v", err)
	}

	if repo.saved == nil {
		t.Fatal("repository SaveDashboardData() was not called")
	}

	if got := repo.saved.Containers[1].Status; got != "paused" {
		t.Fatalf("expected updated status paused, got %q", got)
	}

	if got := repo.saved.Stats.PausedContainers; got != 1 {
		t.Fatalf("expected paused containers 1, got %d", got)
	}

	if got := repo.saved.Stats.RunningContainers; got != 1 {
		t.Fatalf("expected running containers 1, got %d", got)
	}

	if repo.saved.Containers[1].LastUpdated.IsZero() {
		t.Fatal("expected LastUpdated to be set")
	}
}

func TestDashboardService_UpdateContainerStatus_RejectsUnknownStatus(t *testing.T) {
	repo := &fakeDashboardRepository{
		loadData: &DashboardData{
			Containers: []Container{{ID: "web-app-01", Name: "nginx", Status: "running"}},
		},
	}

	service := NewDashboardService(repo)
	if err := service.UpdateContainerStatus(context.Background(), "web-app-01", "unknown"); !errors.Is(err, ErrInvalidContainerStatus) {
		t.Fatalf("expected invalid status error, got %v", err)
	}
}

func TestDashboardService_UpdateContainerStatus_ReturnsNotFound(t *testing.T) {
	repo := &fakeDashboardRepository{
		loadData: &DashboardData{
			Containers: []Container{{ID: "web-app-01", Name: "nginx", Status: "running"}},
		},
	}

	service := NewDashboardService(repo)
	if err := service.UpdateContainerStatus(context.Background(), "missing-id", "stop"); !errors.Is(err, ErrContainerNotFound) {
		t.Fatalf("expected container not found error, got %v", err)
	}
}

func TestDashboardService_GetContainerByID_ReturnsContainer(t *testing.T) {
	repo := &fakeDashboardRepository{
		loadData: &DashboardData{
			Containers: []Container{
				{ID: "web-app-01", Name: "nginx", Status: "running"},
			},
		},
	}

	service := NewDashboardService(repo)
	container, err := service.GetContainerByID(context.Background(), "web-app-01")
	if err != nil {
		t.Fatalf("GetContainerByID() error = %v", err)
	}

	if container == nil || container.ID != "web-app-01" {
		t.Fatal("expected container to be returned")
	}
}

func TestDashboardService_GetContainerByID_ReturnsNotFound(t *testing.T) {
	repo := &fakeDashboardRepository{
		loadData: &DashboardData{
			Containers: []Container{
				{ID: "web-app-01", Name: "nginx", Status: "running"},
			},
		},
	}

	service := NewDashboardService(repo)
	if _, err := service.GetContainerByID(context.Background(), "missing-id"); !errors.Is(err, ErrContainerNotFound) {
		t.Fatalf("expected container not found error, got %v", err)
	}
}

func TestDashboardService_WithoutRepository_ReturnsError(t *testing.T) {
	service := NewDashboardService(nil)
	if _, err := service.GetDashboardData(context.Background()); !errors.Is(err, ErrMissingRepository) {
		t.Fatalf("expected missing repository error, got %v", err)
	}
}
