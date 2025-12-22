package domain

import (
	"fmt"
	"time"
)

// DashboardData represents the main dashboard data structure
type DashboardData struct {
	Title    string    `json:"title"`
	Subtitle string    `json:"subtitle"`
	Stats    Stats     `json:"stats"`
	Containers []Container `json:"containers"`
	System   System   `json:"system"`
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
	return c.Status == "running"
}

// IsStopped returns true if container is stopped
func (c *Container) IsStopped() bool {
	return c.Status == "stopped"
}

// IsPaused returns true if container is paused
func (c *Container) IsPaused() bool {
	return c.Status == "paused"
}

// GetStatusColor returns color class for status
func (c *Container) GetStatusColor() string {
	switch c.Status {
	case "running":
		return "text-green-600"
	case "stopped":
		return "text-red-600"
	case "paused":
		return "text-yellow-600"
	default:
		return "text-gray-600"
	}
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
