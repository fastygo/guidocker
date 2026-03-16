package infrastructure

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"gui-docker/domain"
)

func TestDashboardRepository_LoadDashboardData_MissingFileReturnsDefault(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "no-data", "dashboard.json")
	repo := NewDashboardRepository(dataPath)

	dashboard, err := repo.LoadDashboardData(context.Background())
	if err != nil {
		t.Fatalf("LoadDashboardData() error = %v", err)
	}

	if dashboard.Title != "Dashboard" {
		t.Fatalf("expected default title Dashboard, got %q", dashboard.Title)
	}
	if dashboard.Stats.TotalContainers != 0 {
		t.Fatalf("expected zero containers count for default dashboard, got %d", dashboard.Stats.TotalContainers)
	}
}

func TestDashboardRepository_SaveAndLoadRoundTrip(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "roundtrip", "dashboard.json")
	repo := NewDashboardRepository(dataPath)

	input := &domain.DashboardData{
		Title:    "Test Dashboard",
		Subtitle: "Test subtitle",
		Containers: []domain.Container{
			{ID: "c1", Name: "api", Image: "golang:1.21", Status: "running"},
		},
		System: domain.System{CPUCores: 4, TotalMemory: 8192, UsedMemory: 4096, DiskUsage: 1024},
	}

	if err := repo.SaveDashboardData(context.Background(), input); err != nil {
		t.Fatalf("SaveDashboardData() error = %v", err)
	}

	loaded, err := repo.LoadDashboardData(context.Background())
	if err != nil {
		t.Fatalf("LoadDashboardData() error = %v", err)
	}

	if loaded.Title != input.Title {
		t.Fatalf("title mismatch: expected %q, got %q", input.Title, loaded.Title)
	}

	if len(loaded.Containers) != len(input.Containers) {
		t.Fatalf("expected %d containers, got %d", len(input.Containers), len(loaded.Containers))
	}

	if loaded.Containers[0].ID != input.Containers[0].ID {
		t.Fatalf("expected container id %q, got %q", input.Containers[0].ID, loaded.Containers[0].ID)
	}

	if !reflect.DeepEqual(loaded.System, input.System) {
		t.Fatalf("system mismatch: %+v != %+v", loaded.System, input.System)
	}
}
