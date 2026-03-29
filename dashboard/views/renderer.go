package views

import (
	"dashboard/domain"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"strings"
	"time"
)

//go:embed *.html partials/*.html
var viewsFS embed.FS

// LayoutData is the base data passed to layout (Title, Subtitle, Active).
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

// AppListItem is a list item for apps page (container or app).
type AppListItem struct {
	ID          string
	Name        string
	Status      string
	StatusLabel string
	PortsStr    string
}

// AppDetailView is for app detail page (container-based).
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

// AppDetailPaasView is for app detail page (App/Paas-based).
type AppDetailPaasView struct {
	LayoutData
	ID              string
	Name            string
	Dir             string
	PortsStr        string
	Status          string
	UpdatedAt       string
	PublicDomain    string
	ProxyTargetPort int
	UseTLS          bool
	ManagedEnvStr   string
	FlashMessage    string
	FlashError      string
}

// SettingsView is for the platform settings page.
type SettingsView struct {
	LayoutData
	CertbotEmail         string
	CertbotEnabled       bool
	CertbotStaging       bool
	CertbotAutoRenew     bool
	CertbotTermsAccepted bool
	FlashMessage         string
	FlashError           string
}

// DeleteConfirmView is for the managed app delete confirmation screen.
type DeleteConfirmView struct {
	LayoutData
	ID   string
	Name string
}

// ComposeView is for compose edit page.
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
	RepoAutoDeploy bool
	RepoMode       bool
	IsExistingApp  bool
	FlashMessage   string
	FlashError     string
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
	Status      string
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
	Kind        string
	Confidence  string
	Containers  []string
	ComposeProject string
	Dir         string
	Status      string
	Reason      string
	Name        string
	Ports       []string
	CleanupCmds []string
	IsCurrent   bool
}

type ScanView struct {
	LayoutData
	ScannedAt string
	Resources []ScanResourceView
	StacksDir string
	Summary   ScanSummary
}

// ComposeContainerView is for container-only compose page (no PaaS API).
type ComposeContainerView struct {
	LayoutData
	Title       string
	Subtitle    string
	Name        string
	Image       string
	ComposeYAML string
}

// OverviewView is for overview page.
type OverviewView struct {
	LayoutData
	Stats      domain.Stats
	Containers []ContainerView
}

// AppsView is for apps list page.
type AppsView struct {
	LayoutData
	Items        []AppListItem
	FlashMessage string
	FlashError   string
}

// Renderer loads and executes HTML templates.
type Renderer struct {
	templates map[string]*template.Template
	funcMap   template.FuncMap
}

// NewRenderer creates a renderer and parses all templates from embedded FS.
func NewRenderer() (*Renderer, error) {
	funcMap := template.FuncMap{
		"join": strings.Join,
		"timeLayout": func() string {
			return "2006-01-02 15:04:05"
		},
	}

	root, err := fs.Sub(viewsFS, ".")
	if err != nil {
		return nil, err
	}

	baseNames := []string{
		"layout.html",
		"partials/nav.html",
		"partials/stats_cards.html",
		"partials/containers_table.html",
		"partials/container_row.html",
		"partials/container_actions.html",
		"partials/status_badge.html",
	}

	pages := map[string]string{
		"overview":          "overview.html",
		"apps":              "apps.html",
		"app_detail":        "app_detail.html",
		"app_detail_paas":   "app_detail_paas.html",
		"app_delete_confirm": "app_delete_confirm.html",
		"compose":           "compose.html",
		"compose_container": "compose_container.html",
		"logs":              "logs.html",
		"logs_container":    "logs_container.html",
		"settings":          "settings.html",
		"scan":              "scan.html",
	}

	templates := make(map[string]*template.Template)
	for name, pageFile := range pages {
		allFiles := append(append([]string{}, baseNames...), pageFile)
		t := template.New("").Funcs(funcMap)
		for _, f := range allFiles {
			content, e := fs.ReadFile(root, f)
			if e != nil {
				return nil, fmt.Errorf("read %s: %w", f, e)
			}
			_, e = t.Parse(string(content))
			if e != nil {
				return nil, fmt.Errorf("parse %s: %w", f, e)
			}
		}
		templates[name] = t
	}

	return &Renderer{templates: templates, funcMap: funcMap}, nil
}

// Execute renders the named template with data into the layout.
func (r *Renderer) Execute(name string, data interface{}) (string, error) {
	t, ok := r.templates[name]
	if !ok {
		return "", fmt.Errorf("unknown template: %s", name)
	}
	var sb strings.Builder
	if err := t.ExecuteTemplate(&sb, "layout", data); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// ContainerToView converts domain.Container to ContainerView.
func ContainerToView(c domain.Container) ContainerView {
	return ContainerView{
		ID:         c.ID,
		Name:       c.Name,
		Image:      c.Image,
		Status:     domain.NormalizeStoredStatus(c.Status),
		PortsStr:   strings.Join(c.Ports, ", "),
		CPUPercent: c.GetCPUUsagePercent(),
		MemoryMB:   c.GetMemoryUsageMB(),
	}
}

// ContainersToViews converts a slice of containers.
func ContainersToViews(containers []domain.Container) []ContainerView {
	out := make([]ContainerView, len(containers))
	for i, c := range containers {
		out[i] = ContainerToView(c)
	}
	return out
}

// TimeLayout returns the time format string.
func TimeLayout() string {
	return "2006-01-02 15:04:05"
}

// FormatTime formats t for display.
func FormatTime(t time.Time) string {
	return t.Format(TimeLayout())
}
