package interfaces

import (
	"context"
	"dashboard/domain"
	"dashboard/views"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
)

// DashboardHandler handles HTTP requests for dashboard
type DashboardHandler struct {
	dashboardUseCase        domain.DashboardUseCase
	appUseCase              domain.AppUseCase
	scanUseCase             domain.ScannerUseCase
	platformSettingsUseCase domain.PlatformSettingsUseCase
	certbotManager          certbotManager
	hostManager             certReloadManager
	loginHandler            http.HandlerFunc
	renderer                *views.Renderer
}

type certbotManager interface {
	RenewCertificates(context.Context) error
}

type certReloadManager interface {
	ReloadRouting(context.Context) error
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(useCase domain.DashboardUseCase, renderer *views.Renderer) *DashboardHandler {
	return &DashboardHandler{
		dashboardUseCase: useCase,
		renderer:         renderer,
	}
}

// SetRenderer attaches the renderer (for late injection if needed)
func (h *DashboardHandler) SetRenderer(r *views.Renderer) {
	h.renderer = r
}

func (h *DashboardHandler) executeView(w http.ResponseWriter, name string, data interface{}) {
	if h.renderer == nil {
		http.Error(w, "GUI is disabled in API-only mode", http.StatusNotFound)
		return
	}
	html, err := h.renderer.Execute(name, data)
	if err != nil {
		log.Printf("Template %s: %v", name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func appsToView(apps []*domain.App) views.AppsView {
	items := make([]views.AppListItem, len(apps))
	for i, app := range apps {
		norm := domain.NormalizeAppStatus(app.Status)
		items[i] = views.AppListItem{
			ID:          app.ID,
			Name:        app.Name,
			Status:      norm,
			StatusLabel: domain.StatusLabel(norm),
			PortsStr:    strings.Join(app.Ports, ", "),
		}
	}
	return views.AppsView{
		LayoutData: views.LayoutData{
			Title:    "PaaS Dashboard",
			Subtitle: "Managed applications",
			Active:   "/apps",
		},
		Items: items,
	}
}

func containersToAppItems(containers []domain.Container) []views.AppListItem {
	items := make([]views.AppListItem, len(containers))
	for i, c := range containers {
		norm := domain.NormalizeStoredStatus(c.Status)
		items[i] = views.AppListItem{
			ID:          c.ID,
			Name:        c.Name,
			Status:      norm,
			StatusLabel: domain.FormatStatusLabel(norm),
			PortsStr:    strings.Join(c.Ports, ", "),
		}
	}
	return items
}

func defaultComposeYAML() string {
	return `services:
  app:
    image: nginx:alpine
    ports:
      - "8080:80"
    restart: unless-stopped`
}

func containerComposeYAML(c *domain.Container) string {
	return `services:
  ` + c.Name + `:
    image: ` + c.Image + `
    ports:
      - "80:80"
    environment:
      - TZ=UTC
    restart: unless-stopped`
}

func appDetailPaasToView(app *domain.App, flashMsg, flashErr string) views.AppDetailPaasView {
	return views.AppDetailPaasView{
		LayoutData: views.LayoutData{
			Title:    "PaaS Dashboard",
			Subtitle: "Application details",
			Active:   "/apps",
		},
		ID:              app.ID,
		Name:            app.Name,
		Dir:             app.Dir,
		PortsStr:        strings.Join(app.Ports, ", "),
		Status:          domain.NormalizeAppStatus(app.Status),
		UpdatedAt:       views.FormatTime(app.UpdatedAt),
		PublicDomain:    app.PublicDomain,
		ProxyTargetPort: app.ProxyTargetPort,
		UseTLS:          app.UseTLS,
		ManagedEnvStr:   renderManagedEnv(app.ManagedEnv),
		FlashMessage:    flashMsg,
		FlashError:      flashErr,
	}
}

func composePaasToView(app *domain.App) views.ComposeView {
	title := "Create application"
	subtitle := "New compose deployment"
	actionLabel := "Save compose"
	name := ""
	appID := ""
	composeYAML := defaultComposeYAML()
	sourceType := string(domain.SourceTypeManual)
	repoURL := ""
	repoBranch := ""
	composePath := ""
	appPort := 0
	repoAutoDeploy := true
	repoMode := false

	if app != nil {
		title = "Edit application"
		subtitle = "Update existing compose stack"
		actionLabel = "Save changes"
		name = app.Name
		appID = app.ID
		composeYAML = app.ComposeYAML
		sourceType = string(app.SourceType)
		repoURL = app.RepoURL
		repoBranch = app.RepoBranch
		composePath = app.ComposePath
		appPort = app.AppPort
		repoMode = app.SourceType == domain.SourceTypeRepoCompose || app.SourceType == domain.SourceTypeRepoDockerfile
	}

	appIDJS := template.JS("null")
	if appID != "" {
		quoted, _ := json.Marshal(appID)
		appIDJS = template.JS(quoted)
	}

	return views.ComposeView{
		LayoutData: views.LayoutData{
			Title:    "PaaS Dashboard",
			Subtitle: subtitle,
			Active:   "/apps",
		},
		Title:       title,
		Subtitle:    subtitle,
		Name:        name,
		ComposeYAML: composeYAML,
		SourceType:  sourceType,
		RepoURL:     repoURL,
		RepoBranch:  repoBranch,
		ComposePath: composePath,
		AppPort:     appPort,
		RepoAutoDeploy: repoAutoDeploy,
		RepoMode:       repoMode,
		IsExistingApp:  app != nil,
		ActionLabel: actionLabel,
		AppID:       appID,
		AppIDJS:     appIDJS,
	}
}

func logsPaasToView(app *domain.App, logsContent, status string) views.LogsView {
	return views.LogsView{
		LayoutData: views.LayoutData{
			Title:    "PaaS Dashboard",
			Subtitle: "Application logs",
			Active:   "/apps",
		},
		ID:          app.ID,
		Name:        app.Name,
		LogsContent: logsContent,
		Status:      status,
	}
}

// SetAppUseCase attaches the app lifecycle use case to the handler.
func (h *DashboardHandler) SetAppUseCase(useCase domain.AppUseCase) {
	h.appUseCase = useCase
}

// SetScanUseCase attaches the scanner use case to the handler.
func (h *DashboardHandler) SetScanUseCase(useCase domain.ScannerUseCase) {
	h.scanUseCase = useCase
}

// SetPlatformSettingsUseCase attaches platform-level settings use case.
func (h *DashboardHandler) SetPlatformSettingsUseCase(useCase domain.PlatformSettingsUseCase) {
	h.platformSettingsUseCase = useCase
}

func (h *DashboardHandler) SetCertificateOperations(renewManager certbotManager, hostManager certReloadManager) {
	h.certbotManager = renewManager
	h.hostManager = hostManager
}

// SetLoginHandler attaches a dedicated login handler.
func (h *DashboardHandler) SetLoginHandler(handler http.HandlerFunc) {
	h.loginHandler = handler
}

// Dashboard serves the main dashboard page
func (h *DashboardHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		h.handleOverview(w, r)
		return
	case "/apps", "/apps/":
		h.handleApps(w, r)
		return
	case "/settings", "/settings/":
		if r.Method == http.MethodPost {
			h.handleSettingsPost(w, r)
			return
		}
		h.handleSettings(w, r)
		return
	case "/settings/renew":
		if r.Method == http.MethodPost {
			h.handleCertbotRenewPost(w, r)
			return
		}
		h.writeMethodNotAllowed(w)
		return
	default:
		if strings.HasPrefix(r.URL.Path, "/apps/") {
			h.handleAppRoutes(w, r)
			return
		}

		http.NotFound(w, r)
		return
	}
}

func (h *DashboardHandler) handleOverview(w http.ResponseWriter, r *http.Request) {
	dashboardData, err := h.loadDashboardData(r.Context())
	if err != nil {
		log.Printf("Failed to get dashboard data: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	view := views.OverviewView{
		LayoutData: views.LayoutData{
			Title:    dashboardData.Title,
			Subtitle: dashboardData.Subtitle,
			Active:   "/",
		},
		Stats:      dashboardData.Stats,
		Containers: views.ContainersToViews(dashboardData.Containers),
	}
	h.executeView(w, "overview", view)
}

func (h *DashboardHandler) handleApps(w http.ResponseWriter, r *http.Request) {
	flashMsg := r.URL.Query().Get("msg")
	flashErr := r.URL.Query().Get("err")
	if h.appUseCase != nil {
		apps, err := h.appUseCase.ListApps(r.Context())
		if err != nil {
			log.Printf("Failed to get apps: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		view := appsToView(apps)
		view.FlashMessage = flashMsg
		view.FlashError = flashErr
		h.executeView(w, "apps", view)
		return
	}

	dashboardData, err := h.loadDashboardData(r.Context())
	if err != nil {
		log.Printf("Failed to get dashboard data: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	view := views.AppsView{
		LayoutData: views.LayoutData{
			Title:    dashboardData.Title,
			Subtitle: "Managed containers",
			Active:   "/apps",
		},
		Items:        containersToAppItems(dashboardData.Containers),
		FlashMessage: flashMsg,
		FlashError:   flashErr,
	}
	h.executeView(w, "apps", view)
}

func (h *DashboardHandler) handleAppRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/apps/")
	path = strings.Trim(path, "/")
	if path == "" {
		http.Redirect(w, r, "/apps", http.StatusSeeOther)
		return
	}

	parts := strings.Split(path, "/")
	if len(parts) == 1 {
		if parts[0] == "new" {
			h.handleComposeCreate(w, r)
			return
		}

		h.handleAppDetail(w, r, parts[0])
		return
	}

	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}

	id := parts[0]
	action := parts[1]
	if id == "new" && (action == "compose" || action == "deploy" || action == "edit") {
		if r.Method == http.MethodPost && action == "compose" {
			h.handleComposePost(w, r, "")
			return
		}
		h.handleComposeCreate(w, r)
		return
	}

	switch action {
	case "compose", "edit":
		if r.Method == http.MethodPost {
			h.handleComposePost(w, r, id)
			return
		}
		h.handleComposeEdit(w, r, id)
	case "deploy":
		if r.Method == http.MethodPost {
			h.handleAppLifecyclePost(w, r, id, "Application deployed", h.appUseCase.DeployApp)
			return
		}
		h.handleComposeEdit(w, r, id)
	case "logs":
		h.handleAppLogs(w, r, id)
	case "restart":
		if r.Method == http.MethodPost {
			if h.appUseCase != nil {
				h.handleAppLifecyclePost(w, r, id, "Application restarted", h.appUseCase.RestartApp)
			} else {
				h.handleContainerLifecyclePost(w, r, id, "restart")
			}
			return
		}
		http.NotFound(w, r)
	case "stop":
		if r.Method == http.MethodPost {
			if h.appUseCase != nil {
				h.handleAppLifecyclePost(w, r, id, "Application stopped", h.appUseCase.StopApp)
			} else {
				h.handleContainerLifecyclePost(w, r, id, "stop")
			}
			return
		}
		http.NotFound(w, r)
	case "start", "pause", "unpause":
		if r.Method == http.MethodPost && h.appUseCase == nil {
			h.handleContainerLifecyclePost(w, r, id, action)
			return
		}
		http.NotFound(w, r)
	case "delete":
		if r.Method == http.MethodPost {
			h.handleAppDeletePost(w, r, id)
			return
		}
		h.handleAppDeleteConfirm(w, r, id)
	case "config":
		if r.Method == http.MethodPost {
			h.handleAppConfigPost(w, r, id)
			return
		}
		http.NotFound(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *DashboardHandler) handleAppDetail(w http.ResponseWriter, r *http.Request, id string) {
	if id == "" {
		http.Redirect(w, r, "/apps", http.StatusSeeOther)
		return
	}

	if h.appUseCase != nil {
		app, err := h.appUseCase.GetApp(r.Context(), id)
		if err != nil {
			log.Printf("Failed to load app %s: %v", id, err)
			http.NotFound(w, r)
			return
		}

		view := appDetailPaasToView(app, r.URL.Query().Get("msg"), r.URL.Query().Get("err"))
		h.executeView(w, "app_detail_paas", view)
		return
	}

	container, err := h.dashboardUseCase.GetContainerByID(r.Context(), id)
	if err != nil {
		log.Printf("Failed to load app %s: %v", id, err)
		http.NotFound(w, r)
		return
	}

	view := views.AppDetailView{
		LayoutData: views.LayoutData{
			Title:    "Docker Container Dashboard",
			Subtitle: "Container details",
			Active:   "/apps",
		},
		ID:         container.ID,
		Name:       container.Name,
		Image:      container.Image,
		Status:     domain.NormalizeStoredStatus(container.Status),
		PortsStr:   strings.Join(container.Ports, ", "),
		CPUPercent: container.GetCPUUsagePercent(),
		MemoryMB:   container.GetMemoryUsageMB(),
	}
	h.executeView(w, "app_detail", view)
}

func (h *DashboardHandler) handleAppConfigPost(w http.ResponseWriter, r *http.Request, id string) {
	if h.appUseCase == nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		redirectWithFlash(w, r, "/apps/"+url.PathEscape(id), "", "Invalid form submission")
		return
	}

	config := domain.AppConfig{
		PublicDomain:    r.FormValue("public_domain"),
		ProxyTargetPort: parseIntOr(r.FormValue("proxy_target_port"), 0),
		UseTLS:          r.FormValue("use_tls") == "on",
		ManagedEnv:      parseManagedEnvString(r.FormValue("managed_env")),
	}
	if _, err := h.appUseCase.UpdateAppConfig(r.Context(), id, config); err != nil {
		redirectWithFlash(w, r, "/apps/"+url.PathEscape(id), "", err.Error())
		return
	}

	redirectWithFlash(w, r, "/apps/"+url.PathEscape(id), "App settings saved", "")
}

func (h *DashboardHandler) handleAppLifecyclePost(w http.ResponseWriter, r *http.Request, id, successMessage string, fn func(context.Context, string) error) {
	if h.appUseCase == nil || fn == nil {
		http.NotFound(w, r)
		return
	}
	if err := fn(r.Context(), id); err != nil {
		redirectWithFlash(w, r, "/apps/"+url.PathEscape(id), "", err.Error())
		return
	}
	redirectWithFlash(w, r, "/apps/"+url.PathEscape(id), successMessage, "")
}

func (h *DashboardHandler) handleAppDeletePost(w http.ResponseWriter, r *http.Request, id string) {
	if h.appUseCase == nil {
		http.NotFound(w, r)
		return
	}
	if err := h.appUseCase.DeleteApp(r.Context(), id); err != nil {
		redirectWithFlash(w, r, "/apps/"+url.PathEscape(id), "", err.Error())
		return
	}
	redirectWithFlash(w, r, "/apps", "Application deleted", "")
}

func (h *DashboardHandler) handleContainerLifecyclePost(w http.ResponseWriter, r *http.Request, id, status string) {
	if h.dashboardUseCase == nil {
		http.NotFound(w, r)
		return
	}
	if err := h.dashboardUseCase.UpdateContainerStatus(r.Context(), id, status); err != nil {
		redirectBackOrDefault(w, r, "/apps/"+url.PathEscape(id))
		return
	}
	redirectBackOrDefault(w, r, "/apps/"+url.PathEscape(id))
}

func (h *DashboardHandler) handleAppDeleteConfirm(w http.ResponseWriter, r *http.Request, id string) {
	if h.appUseCase == nil {
		http.NotFound(w, r)
		return
	}
	app, err := h.appUseCase.GetApp(r.Context(), id)
	if err != nil {
		log.Printf("Failed to load app %s for delete confirmation: %v", id, err)
		http.NotFound(w, r)
		return
	}

	view := views.DeleteConfirmView{
		LayoutData: views.LayoutData{
			Title:    "PaaS Dashboard",
			Subtitle: "Delete confirmation",
			Active:   "/apps",
		},
		ID:   app.ID,
		Name: app.Name,
	}
	h.executeView(w, "app_delete_confirm", view)
}

func (h *DashboardHandler) handleComposePost(w http.ResponseWriter, r *http.Request, id string) {
	if h.appUseCase == nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form submission", http.StatusBadRequest)
		return
	}

	mode := strings.TrimSpace(r.FormValue("mode"))
	if mode == "" {
		mode = "compose"
	}
	shouldDeploy := r.FormValue("submit_action") == "deploy"
	if mode == "repo" {
		h.handleComposeRepoPost(w, r, id, shouldDeploy)
		return
	}
	h.handleComposeManualPost(w, r, id, shouldDeploy)
}

func (h *DashboardHandler) handleComposeManualPost(w http.ResponseWriter, r *http.Request, id string, shouldDeploy bool) {
	name := strings.TrimSpace(r.FormValue("name"))
	composeYAML := r.FormValue("compose_yaml")

	var (
		app *domain.App
		err error
	)
	if id == "" {
		app, err = h.appUseCase.CreateApp(r.Context(), name, composeYAML)
	} else {
		app, err = h.appUseCase.UpdateApp(r.Context(), id, name, composeYAML)
	}
	if err != nil {
		var existing *domain.App
		if id != "" {
			existing = &domain.App{ID: id}
		}
		h.renderComposeError(w, id, buildComposeViewFromRequest(existing, r, err.Error()))
		return
	}

	if shouldDeploy {
		if err := h.appUseCase.DeployApp(r.Context(), app.ID); err != nil {
			h.renderComposeError(w, app.ID, buildComposeViewFromRequest(app, r, err.Error()))
			return
		}
		redirectWithFlash(w, r, "/apps/"+url.PathEscape(app.ID), "Application deployed", "")
		return
	}

	redirectWithFlash(w, r, "/apps/"+url.PathEscape(app.ID)+"/compose", "Compose saved", "")
}

func (h *DashboardHandler) handleComposeRepoPost(w http.ResponseWriter, r *http.Request, id string, shouldDeploy bool) {
	if id != "" {
		h.renderComposeError(w, id, buildComposeViewFromRequest(&domain.App{ID: id}, r, "Import mode updates existing applications are disabled"))
		return
	}
	autoDeploy := shouldDeploy || r.FormValue("repo_auto_deploy") == "on"

	input := domain.ImportRepoInput{
		Name:        strings.TrimSpace(r.FormValue("name")),
		RepoURL:     strings.TrimSpace(r.FormValue("repo_url")),
		Branch:      strings.TrimSpace(r.FormValue("repo_branch")),
		ComposePath: strings.TrimSpace(r.FormValue("repo_compose_path")),
		AppPort:     parseIntOr(r.FormValue("repo_app_port"), 0),
		AutoDeploy:  autoDeploy,
	}
	app, err := h.appUseCase.ImportRepo(r.Context(), input)
	if err != nil {
		h.renderComposeError(w, "", buildComposeViewFromRequest(nil, r, err.Error()))
		return
	}

	if autoDeploy {
		redirectWithFlash(w, r, "/apps/"+url.PathEscape(app.ID), "Application deployed", "")
		return
	}
	redirectWithFlash(w, r, "/apps", "Application imported", "")
}

func (h *DashboardHandler) renderComposeError(w http.ResponseWriter, id string, view views.ComposeView) {
	if id == "" {
		view.Title = "Create application"
		view.Subtitle = "New compose deployment"
	} else if view.Title == "" {
		view.Title = "Edit application"
		view.Subtitle = "Update existing compose stack"
	}
	h.executeView(w, "compose", view)
}

func buildComposeViewFromRequest(app *domain.App, r *http.Request, flashErr string) views.ComposeView {
	view := composePaasToView(app)
	mode := strings.TrimSpace(r.FormValue("mode"))
	if mode == "" {
		mode = "compose"
	}
	view.Name = strings.TrimSpace(r.FormValue("name"))
	view.ComposeYAML = r.FormValue("compose_yaml")
	view.RepoURL = strings.TrimSpace(r.FormValue("repo_url"))
	view.RepoBranch = strings.TrimSpace(r.FormValue("repo_branch"))
	view.ComposePath = strings.TrimSpace(r.FormValue("repo_compose_path"))
	view.AppPort = parseIntOr(r.FormValue("repo_app_port"), 0)
	view.RepoAutoDeploy = r.FormValue("repo_auto_deploy") == "on"
	view.RepoMode = mode == "repo"
	view.FlashError = flashErr
	view.FlashMessage = ""
	if app == nil {
		view.IsExistingApp = false
		view.AppID = ""
	}
	return view
}

func (h *DashboardHandler) handleComposeCreate(w http.ResponseWriter, r *http.Request) {
	if h.appUseCase != nil {
		view := composePaasToView(nil)
		view.FlashMessage = r.URL.Query().Get("msg")
		view.FlashError = r.URL.Query().Get("err")
		h.executeView(w, "compose", view)
		return
	}

	view := views.ComposeContainerView{
		LayoutData: views.LayoutData{
			Title:    "Docker Container Dashboard",
			Subtitle: "New app compose package",
			Active:   "/apps",
		},
		Title:       "Create application",
		Subtitle:    "New app compose package",
		Name:        "",
		Image:       "",
		ComposeYAML: defaultComposeYAML(),
	}
	h.executeView(w, "compose_container", view)
}

func (h *DashboardHandler) handleComposeEdit(w http.ResponseWriter, r *http.Request, id string) {
	if id == "" {
		http.Redirect(w, r, "/apps/new", http.StatusSeeOther)
		return
	}

	if h.appUseCase != nil {
		app, err := h.appUseCase.GetApp(r.Context(), id)
		if err != nil {
			log.Printf("Failed to load app %s: %v", id, err)
			http.NotFound(w, r)
			return
		}

		view := composePaasToView(app)
		view.FlashMessage = r.URL.Query().Get("msg")
		view.FlashError = r.URL.Query().Get("err")
		h.executeView(w, "compose", view)
		return
	}

	container, err := h.dashboardUseCase.GetContainerByID(r.Context(), id)
	if err != nil {
		log.Printf("Failed to load app %s: %v", id, err)
		http.NotFound(w, r)
		return
	}

	view := views.ComposeContainerView{
		LayoutData: views.LayoutData{
			Title:    "Docker Container Dashboard",
			Subtitle: "App: " + container.ID,
			Active:   "/apps",
		},
		Title:       "Edit application",
		Subtitle:    "App: " + container.ID,
		Name:        container.Name,
		Image:       container.Image,
		ComposeYAML: containerComposeYAML(container),
	}
	h.executeView(w, "compose_container", view)
}

func (h *DashboardHandler) handleAppLogs(w http.ResponseWriter, r *http.Request, id string) {
	if id == "" {
		http.NotFound(w, r)
		return
	}

	if h.appUseCase != nil {
		app, err := h.appUseCase.GetApp(r.Context(), id)
		if err != nil {
			log.Printf("Failed to load app %s: %v", id, err)
			http.NotFound(w, r)
			return
		}

		logsText, err := h.appUseCase.GetAppLogs(r.Context(), app, 100)
		if err != nil {
			log.Printf("Failed to load app logs for %s: %v", id, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		status, statusErr := h.appUseCase.GetAppStatus(r.Context(), id)
		if statusErr != nil {
			status = "unknown"
		}

		view := logsPaasToView(app, logsText, status)
		h.executeView(w, "logs", view)
		return
	}

	container, err := h.dashboardUseCase.GetContainerByID(r.Context(), id)
	if err != nil {
		log.Printf("Failed to load app %s: %v", id, err)
		http.NotFound(w, r)
		return
	}

	logLines := []string{
		"2026-03-12T09:00:01Z | info | " + container.Name + " started",
		"2026-03-12T09:00:02Z | info | pulling image " + container.Image,
		"2026-03-12T09:00:05Z | warn | high memory usage detected",
		"2026-03-12T09:00:07Z | info | health check passed",
		"2026-03-12T09:00:10Z | debug | request latency avg=12ms",
	}
	var logs strings.Builder
	for _, line := range logLines {
		logs.WriteString(line + "\n")
	}
	view := views.LogsView{
		LayoutData: views.LayoutData{
			Title:    "Docker Container Dashboard",
			Subtitle: "Application logs",
			Active:   "/apps",
		},
		ID:          container.ID,
		Name:        container.Name,
		LogsContent: logs.String(),
		Status:      domain.NormalizeStoredStatus(container.Status),
	}
	h.executeView(w, "logs_container", view)
}

func (h *DashboardHandler) handleSettings(w http.ResponseWriter, r *http.Request) {
	view := views.SettingsView{
		LayoutData: views.LayoutData{
			Title:    "Docker Container Dashboard",
			Subtitle: "Panel settings",
			Active:   "/settings",
		},
		FlashMessage: r.URL.Query().Get("msg"),
		FlashError:   r.URL.Query().Get("err"),
	}
	if h.platformSettingsUseCase != nil {
		settings, err := h.platformSettingsUseCase.GetPlatformSettings(r.Context())
		if err != nil {
			log.Printf("Failed to load platform settings: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if settings != nil {
			view.CertbotEmail = settings.CertbotEmail
			view.CertbotEnabled = settings.CertbotEnabled
			view.CertbotStaging = settings.CertbotStaging
			view.CertbotAutoRenew = settings.CertbotAutoRenew
			view.CertbotTermsAccepted = settings.CertbotTermsAccepted
		}
	}
	h.executeView(w, "settings", view)
}

func (h *DashboardHandler) handleSettingsPost(w http.ResponseWriter, r *http.Request) {
	if h.platformSettingsUseCase == nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		redirectWithFlash(w, r, "/settings", "", "Invalid form submission")
		return
	}

	settings := domain.PlatformSettings{
		CertbotEmail:         r.FormValue("certbot_email"),
		CertbotEnabled:       r.FormValue("certbot_enabled") == "on",
		CertbotStaging:       r.FormValue("certbot_staging") == "on",
		CertbotAutoRenew:     r.FormValue("certbot_auto_renew") == "on",
		CertbotTermsAccepted: r.FormValue("certbot_terms") == "on",
	}
	if _, err := h.platformSettingsUseCase.UpdatePlatformSettings(r.Context(), settings); err != nil {
		redirectWithFlash(w, r, "/settings", "", err.Error())
		return
	}

	redirectWithFlash(w, r, "/settings", "Settings saved", "")
}

func (h *DashboardHandler) handleCertbotRenewPost(w http.ResponseWriter, r *http.Request) {
	if h.certbotManager == nil {
		http.NotFound(w, r)
		return
	}
	if err := h.certbotManager.RenewCertificates(r.Context()); err != nil {
		redirectWithFlash(w, r, "/settings", "", "Certificate renewal failed: "+err.Error())
		return
	}
	if h.hostManager != nil {
		if err := h.hostManager.ReloadRouting(r.Context()); err != nil {
			redirectWithFlash(w, r, "/settings", "", "Certificate renewed, but nginx reload failed: "+err.Error())
			return
		}
	}

	redirectWithFlash(w, r, "/settings", "Certificate renewal completed", "")
}

func renderManagedEnv(env map[string]string) string {
	if len(env) == 0 {
		return ""
	}

	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+"="+env[key])
	}
	return strings.Join(lines, "\n")
}

func parseManagedEnvString(text string) map[string]string {
	values := map[string]string{}
	lines := strings.Split(text, "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		separator := strings.Index(line, "=")
		if separator <= 0 {
			continue
		}

		key := strings.TrimSpace(line[:separator])
		value := strings.TrimSpace(line[separator+1:])
		if key == "" {
			continue
		}
		values[key] = value
	}
	return values
}

func parseIntOr(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return value
}

func redirectWithFlash(w http.ResponseWriter, r *http.Request, path, message, flashErr string) {
	values := url.Values{}
	if message != "" {
		values.Set("msg", message)
	}
	if flashErr != "" {
		values.Set("err", flashErr)
	}

	target := path
	if encoded := values.Encode(); encoded != "" {
		target += "?" + encoded
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func redirectBackOrDefault(w http.ResponseWriter, r *http.Request, fallback string) {
	target := strings.TrimSpace(r.Referer())
	if target == "" {
		target = fallback
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

// APIGetDashboard returns dashboard data as JSON
func (h *DashboardHandler) APIGetDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/dashboard" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		h.writeMethodNotAllowed(w)
		return
	}

	dashboardData, err := h.loadDashboardData(r.Context())
	if err != nil {
		log.Printf("Failed to get dashboard data: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	h.writeJSON(w, http.StatusOK, dashboardData)
}

func (h *DashboardHandler) APIHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeMethodNotAllowed(w)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"mode":   healthMode(),
	})
}

type updateContainerRequest struct {
	Status string `json:"status"`
}

// APIUpdateContainer updates container status
func (h *DashboardHandler) APIUpdateContainer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		h.writeMethodNotAllowed(w)
		return
	}

	containerID := strings.TrimPrefix(r.URL.Path, "/api/containers/")
	if containerID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Container ID required")
		return
	}

	containerID = strings.Trim(containerID, "/")
	if containerID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Container ID required")
		return
	}

	var request updateContainerRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if strings.TrimSpace(request.Status) == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Status is required")
		return
	}

	if h.appUseCase != nil {
		if err := h.updateAppFromContainerAction(r.Context(), containerID, request.Status); err != nil {
			log.Printf("Failed to update app status: %v", err)
			switch {
			case errors.Is(err, domain.ErrAppNotFound):
				h.writeErrorResponse(w, http.StatusNotFound, "App not found")
			case errors.Is(err, domain.ErrInvalidContainerStatus), errors.Is(err, domain.ErrInvalidComposeYAML):
				h.writeErrorResponse(w, http.StatusBadRequest, "Invalid app action")
			default:
				h.writeErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
			}
			return
		}
		response := map[string]bool{"success": true}
		h.writeJSON(w, http.StatusOK, response)
		return
	}

	if err := h.dashboardUseCase.UpdateContainerStatus(r.Context(), containerID, request.Status); err != nil {
		log.Printf("Failed to update container status: %v", err)
		switch {
		case errors.Is(err, domain.ErrContainerNotFound):
			h.writeErrorResponse(w, http.StatusNotFound, "Container not found")
		case errors.Is(err, domain.ErrInvalidContainerStatus):
			h.writeErrorResponse(w, http.StatusBadRequest, "Invalid container status")
		default:
			h.writeErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
		}
		return
	}

	response := map[string]bool{"success": true}
	h.writeJSON(w, http.StatusOK, response)
}

func (h *DashboardHandler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

func (h *DashboardHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	response := map[string]string{
		"error": message,
	}
	h.writeJSON(w, statusCode, response)
}

func (h *DashboardHandler) writeMethodNotAllowed(w http.ResponseWriter) {
	h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method Not Allowed")
}

func healthMode() string {
	mode := strings.TrimSpace(os.Getenv("DASHBOARD_MODE"))
	if mode == "" {
		return "gui"
	}
	return mode
}
