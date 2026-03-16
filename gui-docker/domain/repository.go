package domain

import "context"

type DashboardRepository interface {
	LoadDashboardData(ctx context.Context) (*DashboardData, error)
	SaveDashboardData(ctx context.Context, dashboard *DashboardData) error
}

type AppRepository interface {
	Create(ctx context.Context, app *App) error
	Update(ctx context.Context, app *App) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*App, error)
	List(ctx context.Context) ([]*App, error)
}

type DockerRepository interface {
	EnsureNetwork(ctx context.Context) error
	Deploy(ctx context.Context, app *App) error
	Stop(ctx context.Context, app *App) error
	Restart(ctx context.Context, app *App) error
	Destroy(ctx context.Context, app *App) error
	GetStatus(ctx context.Context, app *App) (string, error)
	GetLogs(ctx context.Context, app *App, lines int) (string, error)
	ListRunning(ctx context.Context) ([]Container, error)
	ListAllContainers(ctx context.Context) ([]Container, error)
	InspectContainers(ctx context.Context, ids []string) ([]ContainerDetail, error)
	ResolveContainerIP(ctx context.Context, app *App) (string, error)
}

type GitRepository interface {
	Clone(ctx context.Context, sourceURL, branch, destination string) (string, error)
}

type PlatformSettingsRepository interface {
	LoadPlatformSettings(ctx context.Context) (*PlatformSettings, error)
	SavePlatformSettings(ctx context.Context, settings *PlatformSettings) error
}

type ContainerDetail struct {
	ID     string
	Name   string
	Image  string
	Labels map[string]string
	Mounts []string
	Envs   []string
	Status string
	Ports  []string
}
