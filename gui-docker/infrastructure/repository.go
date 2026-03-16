package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"gui-docker/domain"
	"os"
	"path/filepath"
)

// DashboardRepository implements data access for dashboard
type DashboardRepository struct {
	dataFile string
}

// NewDashboardRepository creates a new repository instance
func NewDashboardRepository(dataFile string) *DashboardRepository {
	return &DashboardRepository{
		dataFile: dataFile,
	}
}

// LoadDashboardData loads dashboard data from JSON file
func (r *DashboardRepository) LoadDashboardData(ctx context.Context) (*domain.DashboardData, error) {
	_ = ctx
	data, err := os.ReadFile(r.dataFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty dashboard if file doesn't exist
			return &domain.DashboardData{
				Title:      "Dashboard",
				Subtitle:   "Container monitoring",
				Stats:      domain.Stats{},
				Containers: []domain.Container{},
				System:     domain.System{},
			}, nil
		}
		return nil, fmt.Errorf("failed to read dashboard data: %w", err)
	}

	var dashboard domain.DashboardData
	if err := json.Unmarshal(data, &dashboard); err != nil {
		return nil, fmt.Errorf("failed to parse dashboard data: %w", err)
	}

	return &dashboard, nil
}

// SaveDashboardData saves dashboard data to JSON file
func (r *DashboardRepository) SaveDashboardData(ctx context.Context, dashboard *domain.DashboardData) error {
	_ = ctx
	if dashboard == nil {
		return fmt.Errorf("dashboard data is nil")
	}

	data, err := json.MarshalIndent(dashboard, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dashboard data: %w", err)
	}

	dir := filepath.Dir(r.dataFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(r.dataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write dashboard data: %w", err)
	}

	return nil
}
