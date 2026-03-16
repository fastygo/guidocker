package pages

import (
	"gui-docker/domain"
	"html/template"
)

// LayoutData is the base data passed to the shell layout.
type LayoutData struct {
	Title    string
	Subtitle string
	Active   string
}

// ContainerView is the view model for a container row.
type ContainerView struct {
	ID         string
	Name       string
	Image      string
	Status     string
	PortsStr   string
	CPUPercent string
	MemoryMB   string
}

// AppListItem is a list item for apps page.
type AppListItem struct {
	ID          string
	Name        string
	Status      string
	StatusLabel string
	PortsStr    string
}

// AppDetailView is for the container detail page.
type AppDetailView struct {
	LayoutData
	ID         string
	Name       string
	Image      string
	Status     string
	PortsStr   string
	CPUPercent string
	MemoryMB   string
}

// AppDetailPaasView is for the PaaS detail page.
type AppDetailPaasView struct {
	LayoutData
	ID        string
	Name      string
	Dir       string
	PortsStr  string
	Status    string
	UpdatedAt string
}

// ComposeView is for the compose edit page.
type ComposeView struct {
	LayoutData
	Title       string
	Subtitle    string
	Name        string
	ComposeYAML string
	SourceType  string
	RepoURL     string
	RepoBranch  string
	ComposePath string
	AppPort     int
	ActionLabel string
	AppID       string
	AppIDJS     template.JS
}

// LogsView is for app logs page.
type LogsView struct {
	LayoutData
	ID          string
	Name        string
	LogsContent string
}

type ScanSummary struct {
	Managed       int
	Broken        int
	OrphanRuntime int
	OrphanDir     int
	StaleAdmin    int
	Unknown       int
}

type ScanResourceView struct {
	Kind           string
	Confidence     string
	Containers     []string
	ComposeProject string
	Dir            string
	Status         string
	Reason         string
	Name           string
	Ports          []string
	CleanupCmds    []string
	IsCurrent      bool
}

type ScanView struct {
	LayoutData
	ScannedAt string
	Resources []ScanResourceView
	StacksDir string
	Summary   ScanSummary
}

// ComposeContainerView is for the container-only compose page.
type ComposeContainerView struct {
	LayoutData
	Title       string
	Subtitle    string
	Name        string
	Image       string
	ComposeYAML string
}

// OverviewView is for the overview page.
type OverviewView struct {
	LayoutData
	Stats      domain.Stats
	Containers []ContainerView
}

// AppsView is for the apps list page.
type AppsView struct {
	LayoutData
	Items []AppListItem
}

// LoginView is for the standalone login page.
type LoginView struct {
	Message string
}
