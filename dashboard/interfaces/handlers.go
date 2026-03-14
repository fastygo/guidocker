package interfaces

import (
	"dashboard/domain"
	"dashboard/views"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"strings"
)

// DashboardHandler handles HTTP requests for dashboard
type DashboardHandler struct {
	dashboardUseCase domain.DashboardUseCase
	appUseCase       domain.AppUseCase
	scanUseCase      domain.ScannerUseCase
	platformSettingsUseCase domain.PlatformSettingsUseCase
	loginHandler     http.HandlerFunc
	renderer         *views.Renderer
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
		log.Printf("Renderer not initialized")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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

func appDetailPaasToView(app *domain.App) views.AppDetailPaasView {
	return views.AppDetailPaasView{
		LayoutData: views.LayoutData{
			Title:    "PaaS Dashboard",
			Subtitle: "Application details",
			Active:   "/apps",
		},
		ID:        app.ID,
		Name:      app.Name,
		Dir:       app.Dir,
		PortsStr:  strings.Join(app.Ports, ", "),
		Status:    domain.NormalizeAppStatus(app.Status),
		UpdatedAt: views.FormatTime(app.UpdatedAt),
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
		ActionLabel: actionLabel,
		AppID:       appID,
		AppIDJS:     appIDJS,
	}
}

func logsPaasToView(app *domain.App) views.LogsView {
	return views.LogsView{
		LayoutData: views.LayoutData{
			Title:    "PaaS Dashboard",
			Subtitle: "Application logs",
			Active:   "/apps",
		},
		ID:          app.ID,
		Name:        app.Name,
		LogsContent: "Loading logs...",
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
		h.handleSettings(w, r)
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
	if h.appUseCase != nil {
		apps, err := h.appUseCase.ListApps(r.Context())
		if err != nil {
			log.Printf("Failed to get apps: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		view := appsToView(apps)
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
		Items: containersToAppItems(dashboardData.Containers),
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
		h.handleComposeCreate(w, r)
		return
	}

	switch action {
	case "compose", "edit", "deploy":
		h.handleComposeEdit(w, r, id)
	case "logs":
		h.handleAppLogs(w, r, id)
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

		view := appDetailPaasToView(app)
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

func (h *DashboardHandler) handleComposeCreate(w http.ResponseWriter, r *http.Request) {
	if h.appUseCase != nil {
		view := composePaasToView(nil)
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

		view := logsPaasToView(app)
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
	}
	h.executeView(w, "logs_container", view)
}

func (h *DashboardHandler) handleSettings(w http.ResponseWriter, r *http.Request) {
	view := views.LayoutData{
		Title:    "Docker Container Dashboard",
		Subtitle: "Panel settings",
		Active:   "/settings",
	}
	h.executeView(w, "settings", view)
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
