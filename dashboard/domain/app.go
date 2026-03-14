package domain

import (
	"strings"
	"time"
)

type SourceType string

const (
	SourceTypeManual         SourceType = "manual"
	SourceTypeRepoCompose    SourceType = "repo_compose"
	SourceTypeRepoDockerfile SourceType = "repo_dockerfile"
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
	ID          string `json:"id"`
	Name        string `json:"name"`
	ComposeYAML string `json:"compose_yaml"`
	SourceType  SourceType `json:"source_type"`
	RepoURL     string `json:"repo_url"`
	RepoBranch  string `json:"repo_branch"`
	ComposePath string `json:"compose_path"`
	ResolvedCommit string `json:"resolved_commit"`
	AppPort     int    `json:"app_port"`

	PublicDomain    string           `json:"public_domain"`
	ProxyTargetPort int              `json:"proxy_target_port"`
	ManagedEnv      map[string]string `json:"managed_env"`
	UseTLS          bool             `json:"use_tls"`

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
