package interfaces

import (
	"context"
	"dashboard/domain"
	"encoding/json"
	"errors"
	"html"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type appPayload struct {
	Name        string `json:"name"`
	ComposeYAML string `json:"compose_yaml"`
}

func (h *DashboardHandler) Login(w http.ResponseWriter, r *http.Request) {
	if h.loginHandler != nil {
		h.loginHandler(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>PaaS Login</title></head>
<body style="font-family:Arial,sans-serif;background:#f8fafc;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;">
	<form method="post" action="/login" style="width:100%;max-width:420px;background:#fff;border:1px solid #e2e8f0;border-radius:12px;padding:24px;display:flex;flex-direction:column;gap:16px;">
		<div><div style="font-size:14px;color:#475569;">PaaS Console</div><h1 style="margin:8px 0 0;">Sign in</h1></div>
		<label style="display:flex;flex-direction:column;gap:8px;"><span>Username</span><input name="username" type="text" required style="padding:10px 12px;border:1px solid #cbd5e1;border-radius:8px;"></label>
		<label style="display:flex;flex-direction:column;gap:8px;"><span>Password</span><input name="password" type="password" required style="padding:10px 12px;border:1px solid #cbd5e1;border-radius:8px;"></label>
		<button type="submit" style="padding:12px 16px;border:none;border-radius:8px;background:#0f172a;color:#fff;">Login</button>
	</form>
</body>
</html>`))
}

func (h *DashboardHandler) APIApps(w http.ResponseWriter, r *http.Request) {
	if h.appUseCase == nil {
		h.writeErrorResponse(w, http.StatusNotImplemented, "App management is not configured")
		return
	}

	switch r.Method {
	case http.MethodGet:
		apps, err := h.appUseCase.ListApps(r.Context())
		if err != nil {
			log.Printf("Failed to list apps: %v", err)
			h.writeErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}

		h.writeJSON(w, http.StatusOK, map[string]any{"apps": apps})
	case http.MethodPost:
		payload, ok := h.decodeAppPayload(w, r)
		if !ok {
			return
		}

		app, err := h.appUseCase.CreateApp(r.Context(), payload.Name, payload.ComposeYAML)
		if err != nil {
			h.writeAppError(w, err)
			return
		}

		h.writeJSON(w, http.StatusCreated, app)
	default:
		h.writeMethodNotAllowed(w)
	}
}

func (h *DashboardHandler) APIAppRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/apps/"), "/")
	if path == "" {
		http.NotFound(w, r)
		return
	}

	parts := strings.Split(path, "/")
	if len(parts) == 1 {
		h.APIAppDetail(w, r)
		return
	}

	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}

	switch parts[1] {
	case "deploy":
		h.APIDeploy(w, r)
	case "stop":
		h.APIStop(w, r)
	case "restart":
		h.APIRestart(w, r)
	case "logs":
		h.APILogs(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *DashboardHandler) APIAppDetail(w http.ResponseWriter, r *http.Request) {
	if h.appUseCase == nil {
		h.writeErrorResponse(w, http.StatusNotImplemented, "App management is not configured")
		return
	}

	id := h.extractAppID(r)
	if id == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "App ID required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		app, err := h.appUseCase.GetApp(r.Context(), id)
		if err != nil {
			h.writeAppError(w, err)
			return
		}

		h.writeJSON(w, http.StatusOK, app)
	case http.MethodPut:
		payload, ok := h.decodeAppPayload(w, r)
		if !ok {
			return
		}

		app, err := h.appUseCase.UpdateApp(r.Context(), id, payload.Name, payload.ComposeYAML)
		if err != nil {
			h.writeAppError(w, err)
			return
		}

		h.writeJSON(w, http.StatusOK, app)
	case http.MethodDelete:
		if err := h.appUseCase.DeleteApp(r.Context(), id); err != nil {
			h.writeAppError(w, err)
			return
		}

		h.writeJSON(w, http.StatusOK, map[string]bool{"success": true})
	default:
		h.writeMethodNotAllowed(w)
	}
}

func (h *DashboardHandler) APIDeploy(w http.ResponseWriter, r *http.Request) {
	h.handleAppAction(w, r, func(ctx context.Context, id string) error {
		return h.appUseCase.DeployApp(ctx, id)
	})
}

func (h *DashboardHandler) APIStop(w http.ResponseWriter, r *http.Request) {
	h.handleAppAction(w, r, func(ctx context.Context, id string) error {
		return h.appUseCase.StopApp(ctx, id)
	})
}

func (h *DashboardHandler) APIRestart(w http.ResponseWriter, r *http.Request) {
	h.handleAppAction(w, r, func(ctx context.Context, id string) error {
		return h.appUseCase.RestartApp(ctx, id)
	})
}

func (h *DashboardHandler) APILogs(w http.ResponseWriter, r *http.Request) {
	if h.appUseCase == nil {
		h.writeErrorResponse(w, http.StatusNotImplemented, "App management is not configured")
		return
	}
	if r.Method != http.MethodGet {
		h.writeMethodNotAllowed(w)
		return
	}

	id := h.extractAppID(r)
	if id == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "App ID required")
		return
	}

	lines := 100
	if value := strings.TrimSpace(r.URL.Query().Get("lines")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			lines = parsed
		}
	}

	logsText, err := h.appUseCase.GetAppLogs(r.Context(), id, lines)
	if err != nil {
		h.writeAppError(w, err)
		return
	}

	status, statusErr := h.appUseCase.GetAppStatus(r.Context(), id)
	if statusErr != nil {
		status = domain.AppStatusError
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"id":     id,
		"logs":   logsText,
		"status": status,
	})
}

func (h *DashboardHandler) loadDashboardData(ctx context.Context) (*domain.DashboardData, error) {
	if h.appUseCase != nil {
		apps, err := h.appUseCase.ListApps(ctx)
		if err != nil {
			return nil, err
		}

		return buildDashboardDataFromApps(apps), nil
	}

	if h.dashboardUseCase == nil {
		return &domain.DashboardData{
			Title:      "PaaS Dashboard",
			Subtitle:   "Compose-managed apps",
			Containers: []domain.Container{},
		}, nil
	}

	return h.dashboardUseCase.GetDashboardData(ctx)
}

func (h *DashboardHandler) updateAppFromContainerAction(ctx context.Context, appID, status string) error {
	if h.appUseCase == nil {
		return domain.ErrMissingAppRepository
	}

	switch normalized, ok := domain.ParseStatusForUpdate(status); {
	case ok && normalized == domain.ContainerStatusStopped:
		return h.appUseCase.StopApp(ctx, appID)
	case ok && normalized == domain.ContainerStatusRunning:
		if strings.EqualFold(strings.TrimSpace(status), "restart") {
			return h.appUseCase.RestartApp(ctx, appID)
		}
		return h.appUseCase.DeployApp(ctx, appID)
	default:
		if strings.EqualFold(strings.TrimSpace(status), "restart") {
			return h.appUseCase.RestartApp(ctx, appID)
		}
		return domain.ErrInvalidContainerStatus
	}
}

func (h *DashboardHandler) handleAppAction(w http.ResponseWriter, r *http.Request, action func(context.Context, string) error) {
	if h.appUseCase == nil {
		h.writeErrorResponse(w, http.StatusNotImplemented, "App management is not configured")
		return
	}
	if r.Method != http.MethodPost {
		h.writeMethodNotAllowed(w)
		return
	}

	id := h.extractAppID(r)
	if id == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "App ID required")
		return
	}

	if err := action(r.Context(), id); err != nil {
		h.writeAppError(w, err)
		return
	}

	status, err := h.appUseCase.GetAppStatus(r.Context(), id)
	if err != nil {
		status = domain.AppStatusError
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"id":      id,
		"status":  status,
	})
}

func (h *DashboardHandler) decodeAppPayload(w http.ResponseWriter, r *http.Request) (*appPayload, bool) {
	var payload appPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return nil, false
	}

	return &payload, true
}

func (h *DashboardHandler) writeAppError(w http.ResponseWriter, err error) {
	log.Printf("App operation failed: %v", err)
	switch {
	case errors.Is(err, domain.ErrAppNotFound):
		h.writeErrorResponse(w, http.StatusNotFound, "App not found")
	case errors.Is(err, domain.ErrInvalidAppName):
		h.writeErrorResponse(w, http.StatusBadRequest, "App name is required")
	case errors.Is(err, domain.ErrInvalidComposeYAML):
		h.writeErrorResponse(w, http.StatusBadRequest, "Compose YAML is invalid")
	default:
		h.writeErrorResponse(w, http.StatusInternalServerError, "Internal Server Error")
	}
}

func (h *DashboardHandler) extractAppID(r *http.Request) string {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/apps/"), "/")
	if path == "" {
		return ""
	}

	return strings.Split(path, "/")[0]
}

func buildDashboardDataFromApps(apps []*domain.App) *domain.DashboardData {
	containers := make([]domain.Container, 0, len(apps))
	stats := domain.Stats{TotalContainers: len(apps)}

	for _, app := range apps {
		status := app.NormalizedStatus()
		switch status {
		case domain.AppStatusRunning:
			stats.RunningContainers++
		case domain.AppStatusStopped, domain.AppStatusCreated:
			stats.StoppedContainers++
		case domain.AppStatusDeploying:
			stats.PausedContainers++
		}

		containers = append(containers, domain.Container{
			ID:          app.ID,
			Name:        app.Name,
			Image:       extractImage(app.ComposeYAML),
			Status:      mapAppStatusToContainerStatus(status),
			Ports:       append([]string(nil), app.Ports...),
			LastUpdated: app.UpdatedAt,
		})
	}

	return &domain.DashboardData{
		Title:      "PaaS Dashboard",
		Subtitle:   "Compose-managed applications",
		Stats:      stats,
		Containers: containers,
		System:     domain.System{},
	}
}

func mapAppStatusToContainerStatus(status string) string {
	switch status {
	case domain.AppStatusRunning:
		return domain.ContainerStatusRunning
	case domain.AppStatusDeploying:
		return domain.ContainerStatusPaused
	default:
		return domain.ContainerStatusStopped
	}
}

func extractImage(composeYAML string) string {
	for _, line := range strings.Split(composeYAML, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "image:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "image:"))
		}
	}

	return "custom"
}

func (h *DashboardHandler) renderAppsPage(apps []*domain.App) string {
	data := &domain.DashboardData{
		Title:    "PaaS Dashboard",
		Subtitle: "Managed applications",
	}

	var rows strings.Builder
	if len(apps) == 0 {
		rows.WriteString(`<div class="rounded border border-dashed border-border bg-card p-6 text-sm text-muted-foreground">No applications yet. Create your first stack from the Compose screen.</div>`)
	} else {
		for _, app := range apps {
			rows.WriteString(`<a href="/apps/` + html.EscapeString(app.ID) + `" class="block rounded border border-border bg-card px-4 py-3 hover:bg-accent">
				<div class="flex items-center justify-between gap-3">
					<div class="min-w-0">
						<div class="font-medium truncate">` + html.EscapeString(app.Name) + `</div>
						<div class="text-xs text-muted-foreground truncate">` + html.EscapeString(strings.Join(app.Ports, ", ")) + `</div>
					</div>
					` + renderAppStatusBadge(app.Status) + `
				</div>
			</a>`)
		}
	}

	content := `<section class="space-y-4">
		<div class="flex items-center justify-between gap-3">
			<div>
				<div class="text-sm text-muted-foreground">Compose managed workloads</div>
				<div class="text-2xl font-semibold">Apps</div>
			</div>
			<a href="/apps/new" class="px-3 py-2 rounded border border-border text-sm">Create app</a>
		</div>
		<div class="grid gap-2">` + rows.String() + `</div>
	</section>`

	return h.renderDashboardShell(data, "/apps", content)
}

func (h *DashboardHandler) renderAppDetailPage(app *domain.App) string {
	data := &domain.DashboardData{
		Title:    "PaaS Dashboard",
		Subtitle: "Application details",
	}

	content := `<section class="space-y-4">
		<div class="flex items-start justify-between gap-3">
			<div>
				<div class="text-sm text-muted-foreground">App ID: ` + html.EscapeString(app.ID) + `</div>
				<h2 class="text-2xl font-semibold">` + html.EscapeString(app.Name) + `</h2>
			</div>
			` + renderAppStatusBadge(app.Status) + `
		</div>
		<div class="grid gap-3 md:grid-cols-2">
			<div class="rounded border border-border bg-card p-4 space-y-2">
				<div class="font-semibold">Runtime</div>
				<p class="text-sm text-muted-foreground">Stack directory: ` + html.EscapeString(app.Dir) + `</p>
				<p class="text-sm text-muted-foreground">Ports: ` + html.EscapeString(strings.Join(app.Ports, ", ")) + `</p>
				<p class="text-sm text-muted-foreground">Updated: ` + html.EscapeString(app.UpdatedAt.Format(timeLayout())) + `</p>
			</div>
			<div class="rounded border border-border bg-card p-4 space-y-3">
				<div class="font-semibold">Actions</div>
				<div class="flex flex-wrap gap-2">
					<button class="px-3 py-2 rounded border border-border text-sm" onclick="runAppAction('deploy')">Deploy</button>
					<button class="px-3 py-2 rounded border border-border text-sm" onclick="runAppAction('restart')">Restart</button>
					<button class="px-3 py-2 rounded border border-border text-sm" onclick="runAppAction('stop')">Stop</button>
				</div>
				<div id="app-feedback" class="text-sm text-muted-foreground"></div>
				<div class="flex flex-wrap gap-2">
					<a href="/apps/` + html.EscapeString(app.ID) + `/compose" class="px-3 py-2 rounded border border-border text-sm">Compose</a>
					<a href="/apps/` + html.EscapeString(app.ID) + `/logs" class="px-3 py-2 rounded border border-border text-sm">Logs</a>
				</div>
			</div>
		</div>
	</section>
	<script>
		async function runAppAction(action) {
			const feedback = document.getElementById('app-feedback');
			if (feedback) feedback.textContent = 'Running ' + action + '...';
			const response = await fetch('/api/apps/' + encodeURIComponent(` + jsQuote(app.ID) + `) + '/' + action, { method: 'POST' });
			const payload = await response.json().catch(() => ({}));
			if (!response.ok) {
				if (feedback) feedback.textContent = payload.error || 'Action failed';
				return;
			}
			window.location.reload();
		}
	</script>`

	return h.renderDashboardShell(data, "/apps", content)
}

func (h *DashboardHandler) renderComposePage(app *domain.App) string {
	title := "Create application"
	subtitle := "New compose deployment"
	actionLabel := "Save compose"
	name := ""
	appID := ""
	composeYAML := `version: "3.9"
services:
  app:
    image: nginx:alpine
    ports:
      - "8080:80"
    restart: unless-stopped`

	if app != nil {
		title = "Edit application"
		subtitle = "Update existing compose stack"
		actionLabel = "Save changes"
		name = app.Name
		appID = app.ID
		composeYAML = app.ComposeYAML
	}

	content := `<section class="space-y-4">
		<div>
			<div class="text-sm text-muted-foreground">` + html.EscapeString(subtitle) + `</div>
			<div class="text-2xl font-semibold">` + html.EscapeString(title) + `</div>
		</div>
		<div class="grid gap-3 lg:grid-cols-[2fr_1fr]">
			<div class="rounded border border-border bg-card p-4 space-y-3">
				<div class="font-semibold">Compose pack</div>
				<form id="compose-form" class="space-y-3">
					<div>
						<label class="text-sm text-muted-foreground">App name</label>
						<input id="app-name" class="mt-1 w-full rounded border border-border bg-background px-2 py-2 text-sm" value="` + html.EscapeString(name) + `" />
					</div>
					<div>
						<label class="text-sm text-muted-foreground">Docker Compose YAML</label>
						<textarea id="compose-yaml" class="mt-1 w-full rounded border border-border bg-background p-2 text-sm" rows="18">` + html.EscapeString(composeYAML) + `</textarea>
					</div>
					<div id="compose-status" class="text-sm text-muted-foreground"></div>
					<div class="flex flex-wrap gap-2">
						<button type="button" class="px-3 py-2 rounded border border-border text-sm" onclick="saveCompose(false)">` + html.EscapeString(actionLabel) + `</button>
						<button type="button" class="px-3 py-2 rounded border border-border text-sm" onclick="saveCompose(true)">Deploy now</button>
					</div>
				</form>
			</div>
			<div class="rounded border border-border bg-card p-4 space-y-2">
				<div class="font-semibold">Shortcuts</div>
				<p class="text-sm text-muted-foreground">Use valid Docker Compose syntax. The app will be persisted in BoltDB and stored under the configured stacks directory on deploy.</p>
				<p class="text-sm text-muted-foreground">After saving you can continue editing, inspect logs, or deploy later from the app details page.</p>
			</div>
		</div>
	</section>
	<script>
		const currentAppID = ` + jsQuote(appID) + `;
		async function saveCompose(shouldDeploy) {
			const statusNode = document.getElementById('compose-status');
			if (statusNode) statusNode.textContent = shouldDeploy ? 'Saving and deploying...' : 'Saving...';
			const payload = {
				name: document.getElementById('app-name').value,
				compose_yaml: document.getElementById('compose-yaml').value
			};
			const method = currentAppID ? 'PUT' : 'POST';
			const endpoint = currentAppID ? '/api/apps/' + encodeURIComponent(currentAppID) : '/api/apps';
			const response = await fetch(endpoint, {
				method,
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload)
			});
			const appPayload = await response.json().catch(() => ({}));
			if (!response.ok) {
				if (statusNode) statusNode.textContent = appPayload.error || 'Save failed';
				return;
			}
			if (!shouldDeploy) {
				window.location.href = '/apps/' + encodeURIComponent(appPayload.id) + '/compose';
				return;
			}
			const deployResponse = await fetch('/api/apps/' + encodeURIComponent(appPayload.id) + '/deploy', { method: 'POST' });
			const deployPayload = await deployResponse.json().catch(() => ({}));
			if (!deployResponse.ok) {
				if (statusNode) statusNode.textContent = deployPayload.error || 'Deploy failed';
				return;
			}
			window.location.href = '/apps/' + encodeURIComponent(appPayload.id);
		}
	</script>`

	return h.renderDashboardShell(&domain.DashboardData{
		Title:    "PaaS Dashboard",
		Subtitle: subtitle,
	}, "/apps", content)
}

func (h *DashboardHandler) renderAppLogsPage(app *domain.App) string {
	content := `<section class="space-y-4">
		<div>
			<div class="text-sm text-muted-foreground">` + html.EscapeString(app.ID) + `</div>
			<div class="text-2xl font-semibold">` + html.EscapeString(app.Name) + ` logs</div>
		</div>
		<div id="app-feedback" class="text-sm text-muted-foreground"></div>
		<div class="flex gap-2">
			<button onclick="loadAppLogs()" class="px-3 py-2 rounded border border-border text-sm">Refresh</button>
			<a href="/apps/` + html.EscapeString(app.ID) + `/compose" class="px-3 py-2 rounded border border-border text-sm">Open compose</a>
		</div>
		<div class="rounded border border-border bg-card p-4 overflow-x-auto">
			<pre id="app-logs-content" class="text-xs text-muted-foreground whitespace-pre-wrap">Loading logs...</pre>
		</div>
	</section>
	<script>
		async function loadAppLogs() {
			const logsNode = document.getElementById('app-logs-content');
			const feedback = document.getElementById('app-feedback');
			if (feedback) feedback.textContent = 'Loading logs...';
			const response = await fetch('/api/apps/' + encodeURIComponent(` + jsQuote(app.ID) + `) + '/logs?lines=100');
			const payload = await response.json().catch(() => ({}));
			if (!response.ok) {
				if (logsNode) logsNode.textContent = payload.error || 'Failed to load logs';
				if (feedback) feedback.textContent = 'Unable to fetch logs';
				return;
			}
			if (logsNode) logsNode.textContent = payload.logs || 'No logs yet';
			if (feedback) feedback.textContent = 'Status: ' + (payload.status || 'unknown');
		}
		document.addEventListener('DOMContentLoaded', loadAppLogs);
	</script>`

	return h.renderDashboardShell(&domain.DashboardData{
		Title:    "PaaS Dashboard",
		Subtitle: "Application logs",
	}, "/apps", content)
}

func renderAppStatusBadge(status string) string {
	className := "inline-flex rounded px-2 py-1 text-xs "

	switch domain.NormalizeAppStatus(status) {
	case domain.AppStatusRunning:
		className += "bg-primary text-primary-foreground"
	case domain.AppStatusDeploying:
		className += "bg-accent text-accent-foreground"
	case domain.AppStatusStopped, domain.AppStatusCreated:
		className += "bg-muted text-muted-foreground"
	default:
		className += "bg-destructive text-destructive-foreground"
	}

	return `<span class="` + className + `">` + html.EscapeString(domain.StatusLabel(status)) + `</span>`
}

func jsQuote(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func timeLayout() string {
	return "2006-01-02 15:04:05 MST"
}
