package domain

import (
	"strings"
	"time"
)

const (
	AppStatusCreated   = "created"
	AppStatusDeploying = "deploying"
	AppStatusRunning   = "running"
	AppStatusStopped   = "stopped"
	AppStatusError     = "error"
)

// App represents a compose-managed application.
type App struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ComposeYAML string    `json:"compose_yaml"`
	Dir         string    `json:"dir"`
	Status      string    `json:"status"`
	Ports       []string  `json:"ports"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NormalizedStatus returns a canonical app status.
func (a *App) NormalizedStatus() string {
	return NormalizeAppStatus(a.Status)
}

// NormalizeAppStatus normalizes app statuses for rendering and counting.
func NormalizeAppStatus(status string) string {
	value := strings.ToLower(strings.TrimSpace(status))

	switch value {
	case "", AppStatusCreated:
		return AppStatusCreated
	case AppStatusDeploying:
		return AppStatusDeploying
	case AppStatusRunning:
		return AppStatusRunning
	case AppStatusStopped:
		return AppStatusStopped
	default:
		return AppStatusError
	}
}

// StatusLabel returns a human-friendly app status label.
func StatusLabel(status string) string {
	normalized := NormalizeAppStatus(status)
	return strings.ToUpper(normalized[:1]) + normalized[1:]
}
