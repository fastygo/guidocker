package domain

import (
	"fmt"
	"strings"
	"time"
)

const (
	ContainerStatusRunning = "running"
	ContainerStatusStopped = "stopped"
	ContainerStatusPaused  = "paused"
)

var statusAliases = map[string]string{
	"up":                  ContainerStatusRunning,
	"start":               ContainerStatusRunning,
	"running":             ContainerStatusRunning,
	"restart":             ContainerStatusRunning,
	"stop":                ContainerStatusStopped,
	"stopped":             ContainerStatusStopped,
	"exited":              ContainerStatusStopped,
	"pause":               ContainerStatusPaused,
	"paused":              ContainerStatusPaused,
	"unpause":             ContainerStatusRunning,
}

// DashboardData represents the main dashboard data structure
type DashboardData struct {
	Title    string    `json:"title"`
	Subtitle string    `json:"subtitle"`
	Stats    Stats     `json:"stats"`
	Containers []Container `json:"containers"`
	System   System    `json:"system"`
}

// Stats represents dashboard statistics
type Stats struct {
	TotalContainers    int `json:"total_containers"`
	RunningContainers  int `json:"running_containers"`
	StoppedContainers  int `json:"stopped_containers"`
	PausedContainers   int `json:"paused_containers"`
}

// Container represents a Docker container
type Container struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Image        string  `json:"image"`
	Status       string  `json:"status"`
	Ports        []string `json:"ports"`
	CPUUsage     float64 `json:"cpu_usage"`
	MemoryUsage  float64 `json:"memory_usage"`
	CreatedAt    time.Time `json:"-"`
	LastUpdated  time.Time `json:"-"`
}

// System represents system information
type System struct {
	CPUCores    int `json:"cpu_cores"`
	TotalMemory int `json:"total_memory"`
	UsedMemory  int `json:"used_memory"`
	DiskUsage   int `json:"disk_usage"`
}

// IsRunning returns true if container is running
func (c *Container) IsRunning() bool {
	return c.GetCanonicalStatus() == ContainerStatusRunning
}

// IsStopped returns true if container is stopped
func (c *Container) IsStopped() bool {
	return c.GetCanonicalStatus() == ContainerStatusStopped
}

// IsPaused returns true if container is paused
func (c *Container) IsPaused() bool {
	return c.GetCanonicalStatus() == ContainerStatusPaused
}

// GetStatusColor returns color class for status
func (c *Container) GetStatusColor() string {
	switch c.GetCanonicalStatus() {
	case ContainerStatusRunning:
		return "text-green-600"
	case ContainerStatusStopped:
		return "text-red-600"
	case ContainerStatusPaused:
		return "text-yellow-600"
	default:
		return "text-gray-600"
	}
}

// GetCanonicalStatus maps legacy and alias values to canonical dashboard statuses.
func (c *Container) GetCanonicalStatus() string {
	return NormalizeStoredStatus(c.Status)
}

// NormalizeStoredStatus returns a canonical status for rendering and statistics.
func NormalizeStoredStatus(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if canonical, ok := statusAliases[normalized]; ok {
		return canonical
	}

	return normalized
}

// ParseStatusForUpdate validates action/input statuses and returns canonical values for persistence.
func ParseStatusForUpdate(raw string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	canonical, ok := statusAliases[normalized]
	if !ok {
		return "", false
	}

	return canonical, true
}

// FormatStatusLabel returns a human-readable status label for UI output.
func FormatStatusLabel(raw string) string {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return ""
	}

	return strings.ToUpper(value[:1]) + value[1:]
}

// BuildStats calculates dashboard stats from container collection.
func BuildStats(containers []Container) Stats {
	stats := Stats{
		TotalContainers: len(containers),
	}

	for _, container := range containers {
		switch NormalizeStoredStatus(container.Status) {
		case ContainerStatusRunning:
			stats.RunningContainers++
		case ContainerStatusStopped:
			stats.StoppedContainers++
		case ContainerStatusPaused:
			stats.PausedContainers++
		}
	}

	return stats
}

// GetCPUUsagePercent returns CPU usage as formatted percentage
func (c *Container) GetCPUUsagePercent() string {
	if c.CPUUsage == 0 {
		return "0%"
	}
	return fmt.Sprintf("%.1f%%", c.CPUUsage)
}

// GetMemoryUsageMB returns memory usage as formatted MB
func (c *Container) GetMemoryUsageMB() string {
	if c.MemoryUsage == 0 {
		return "0 MB"
	}
	return fmt.Sprintf("%.1f MB", c.MemoryUsage)
}
