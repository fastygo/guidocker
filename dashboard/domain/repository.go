package domain

import "context"

type DashboardRepository interface {
	LoadDashboardData(ctx context.Context) (*DashboardData, error)
	SaveDashboardData(ctx context.Context, dashboard *DashboardData) error
}
