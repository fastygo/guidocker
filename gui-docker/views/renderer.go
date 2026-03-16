package views

import (
	"gui-docker/domain"
	"gui-docker/pages"
	"time"
)

type Renderer struct{}

func NewRenderer() (*Renderer, error) {
	return &Renderer{}, nil
}

type LayoutData = pages.LayoutData
type ContainerView = pages.ContainerView
type AppListItem = pages.AppListItem
type AppDetailView = pages.AppDetailView
type AppDetailPaasView = pages.AppDetailPaasView
type ComposeView = pages.ComposeView
type LogsView = pages.LogsView
type ScanSummary = pages.ScanSummary
type ScanResourceView = pages.ScanResourceView
type ScanView = pages.ScanView
type ComposeContainerView = pages.ComposeContainerView
type OverviewView = pages.OverviewView
type AppsView = pages.AppsView

func ContainerToView(c domain.Container) ContainerView {
	return pages.ContainerToView(c)
}

func ContainersToViews(containers []domain.Container) []ContainerView {
	return pages.ContainersToViews(containers)
}

func TimeLayout() string {
	return pages.TimeLayout()
}

func FormatTime(t time.Time) string {
	return pages.FormatTime(t)
}
