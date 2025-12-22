package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DashboardUseCase defines the business logic interface
type DashboardUseCase interface {
	GetDashboardData(ctx context.Context) (*DashboardData, error)
	UpdateContainerStatus(ctx context.Context, containerID, status string) error
	GetContainerByID(ctx context.Context, id string) (*Container, error)
}

// DashboardService implements DashboardUseCase
type DashboardService struct {
	dataFile string
}

// NewDashboardService creates a new dashboard service
func NewDashboardService(dataFile string) *DashboardService {
	return &DashboardService{
		dataFile: dataFile,
	}
}

// GetDashboardData retrieves dashboard data from JSON file
func (s *DashboardService) GetDashboardData(ctx context.Context) (*DashboardData, error) {
	data, err := os.ReadFile(s.dataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read dashboard data: %w", err)
	}

	var dashboard DashboardData
	if err := json.Unmarshal(data, &dashboard); err != nil {
		return nil, fmt.Errorf("failed to parse dashboard data: %w", err)
	}

	// Calculate stats if not provided
	if dashboard.Stats.TotalContainers == 0 {
		dashboard.Stats.TotalContainers = len(dashboard.Containers)
		dashboard.Stats.RunningContainers = 0
		dashboard.Stats.StoppedContainers = 0
		dashboard.Stats.PausedContainers = 0

		for _, container := range dashboard.Containers {
			switch container.Status {
			case "running":
				dashboard.Stats.RunningContainers++
			case "stopped":
				dashboard.Stats.StoppedContainers++
			case "paused":
				dashboard.Stats.PausedContainers++
			}
		}
	}

	return &dashboard, nil
}

// UpdateContainerStatus updates the status of a specific container
func (s *DashboardService) UpdateContainerStatus(ctx context.Context, containerID, status string) error {
	dashboard, err := s.GetDashboardData(ctx)
	if err != nil {
		return err
	}

	// Find and update container
	for i := range dashboard.Containers {
		if dashboard.Containers[i].ID == containerID {
			dashboard.Containers[i].Status = status
			dashboard.Containers[i].LastUpdated = time.Now()
			break
		}
	}

	// Recalculate stats
	dashboard.Stats = Stats{
		TotalContainers:   len(dashboard.Containers),
		RunningContainers: 0,
		StoppedContainers: 0,
		PausedContainers:  0,
	}

	for _, container := range dashboard.Containers {
		switch container.Status {
		case "running":
			dashboard.Stats.RunningContainers++
		case "stopped":
			dashboard.Stats.StoppedContainers++
		case "paused":
			dashboard.Stats.PausedContainers++
		}
	}

	// Save updated data
	return s.saveDashboardData(dashboard)
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

	return nil, fmt.Errorf("container with ID %s not found", id)
}

// saveDashboardData saves dashboard data to JSON file
func (s *DashboardService) saveDashboardData(dashboard *DashboardData) error {
	data, err := json.MarshalIndent(dashboard, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dashboard data: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(s.dataFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(s.dataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write dashboard data: %w", err)
	}

	return nil
}
