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
	Deploy(ctx context.Context, app *App) error
	Stop(ctx context.Context, app *App) error
	Restart(ctx context.Context, app *App) error
	GetStatus(ctx context.Context, app *App) (string, error)
	GetLogs(ctx context.Context, appID string, lines int) (string, error)
	ListRunning(ctx context.Context) ([]Container, error)
}
