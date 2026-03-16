package interfaces

import (
	"context"
	"dashboard/domain"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type appPayload struct {
	Name        string `json:"name"`
	ComposeYAML string `json:"compose_yaml"`
}

type appConfigPayload struct {
	PublicDomain    string            `json:"public_domain"`
	ProxyTargetPort int               `json:"proxy_target_port"`
	ManagedEnv      map[string]string `json:"managed_env"`
	UseTLS          bool              `json:"use_tls"`
}

type importRepoPayload struct {
	Name        string `json:"name"`
	RepoURL     string `json:"repo_url"`
	Branch      string `json:"branch"`
	ComposePath string `json:"compose_path"`
	AppPort     int    `json:"app_port"`
	AutoDeploy  bool   `json:"auto_deploy"`
}

type platformSettingsPayload struct {
	AdminHost     string `json:"admin_host"`
	AdminPort     int    `json:"admin_port"`
	AdminDomain   string `json:"admin_domain"`
	AdminUseTLS   bool   `json:"admin_use_tls"`
	CertbotEmail  string `json:"certbot_email"`
	CertbotEnabled bool  `json:"certbot_enabled"`
	CertbotStaging bool  `json:"certbot_staging"`
	CertbotAutoRenew bool `json:"certbot_auto_renew"`
	CertbotTermsAccepted bool `json:"certbot_terms_accepted"`
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

func (h *DashboardHandler) APIImport(w http.ResponseWriter, r *http.Request) {
	if h.appUseCase == nil {
		h.writeErrorResponse(w, http.StatusNotImplemented, "App management is not configured")
		return
	}

	if r.Method != http.MethodPost {
		h.writeMethodNotAllowed(w)
		return
	}

	var payload importRepoPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	input := domain.ImportRepoInput{
		Name:        payload.Name,
		RepoURL:     payload.RepoURL,
		Branch:      payload.Branch,
		ComposePath: payload.ComposePath,
		AppPort:     payload.AppPort,
		AutoDeploy:  payload.AutoDeploy,
	}
	app, err := h.appUseCase.ImportRepo(r.Context(), input)
	if err != nil {
		h.writeAppError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, app)
}

func (h *DashboardHandler) APICertbotRenew(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeMethodNotAllowed(w)
		return
	}

	if h.certbotManager == nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "certbot manager is not configured")
		return
	}

	if err := h.certbotManager.RenewCertificates(r.Context()); err != nil {
		log.Printf("Certbot renewal failed: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Certificate renewal failed")
		return
	}

	if h.hostManager != nil {
		if err := h.hostManager.ReloadRouting(r.Context()); err != nil {
			log.Printf("Nginx reload after certbot renew failed: %v", err)
			h.writeErrorResponse(w, http.StatusInternalServerError, "Certificate renewed, but nginx reload failed")
			return
		}
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Certificate renewal completed",
	})
}

func (h *DashboardHandler) APISettings(w http.ResponseWriter, r *http.Request) {
	if h.platformSettingsUseCase == nil {
		h.writeErrorResponse(w, http.StatusNotImplemented, "Platform settings are not configured")
		return
	}

	switch r.Method {
	case http.MethodGet:
		settings, err := h.platformSettingsUseCase.GetPlatformSettings(r.Context())
		if err != nil {
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to load platform settings")
			return
		}
		h.writeJSON(w, http.StatusOK, settings)
	case http.MethodPut:
		var payload platformSettingsPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		updated, err := h.platformSettingsUseCase.UpdatePlatformSettings(r.Context(), domain.PlatformSettings{
			AdminHost:   payload.AdminHost,
			AdminPort:   payload.AdminPort,
			AdminDomain: payload.AdminDomain,
			AdminUseTLS: payload.AdminUseTLS,
			CertbotEmail: payload.CertbotEmail,
			CertbotEnabled: payload.CertbotEnabled,
			CertbotStaging: payload.CertbotStaging,
			CertbotAutoRenew: payload.CertbotAutoRenew,
			CertbotTermsAccepted: payload.CertbotTermsAccepted,
		})
		if err != nil {
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update platform settings")
			return
		}
		h.writeJSON(w, http.StatusOK, updated)
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
	case "config":
		h.APIAppConfig(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *DashboardHandler) APIAppConfig(w http.ResponseWriter, r *http.Request) {
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
		h.writeJSON(w, http.StatusOK, domain.AppConfig{
			PublicDomain:    app.PublicDomain,
			ProxyTargetPort: app.ProxyTargetPort,
			ManagedEnv:      app.ManagedEnv,
			UseTLS:          app.UseTLS,
		})
	case http.MethodPut:
		var payload appConfigPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		updated, err := h.appUseCase.UpdateAppConfig(r.Context(), id, domain.AppConfig{
			PublicDomain:    payload.PublicDomain,
			ProxyTargetPort: payload.ProxyTargetPort,
			ManagedEnv:      payload.ManagedEnv,
			UseTLS:          payload.UseTLS,
		})
		if err != nil {
			h.writeAppError(w, err)
			return
		}

		h.writeJSON(w, http.StatusOK, domain.AppConfig{
			PublicDomain:    updated.PublicDomain,
			ProxyTargetPort: updated.ProxyTargetPort,
			ManagedEnv:      updated.ManagedEnv,
			UseTLS:          updated.UseTLS,
		})
	default:
		h.writeMethodNotAllowed(w)
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

	app, err := h.appUseCase.GetApp(r.Context(), id)
	if err != nil {
		h.writeAppError(w, err)
		return
	}

	logsText, err := h.appUseCase.GetAppLogs(r.Context(), app, lines)
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
	case errors.Is(err, domain.ErrInvalidRepoURL),
		errors.Is(err, domain.ErrUnsupportedRepoURL),
		errors.Is(err, domain.ErrRepoBranchNotFound),
		errors.Is(err, domain.ErrInvalidComposePath),
		errors.Is(err, domain.ErrMissingComposeFile),
		errors.Is(err, domain.ErrMissingDockerfile),
		errors.Is(err, domain.ErrInvalidAppPort),
		errors.Is(err, domain.ErrInvalidDomain),
		errors.Is(err, domain.ErrInvalidProxyPort),
		errors.Is(err, domain.ErrTLSEmailRequired),
		errors.Is(err, domain.ErrTLSAgreementRequired),
		errors.Is(err, domain.ErrTLSRequiresCertbot),
		errors.Is(err, domain.ErrAdminPortConflict),
		errors.Is(err, domain.ErrDomainConflict),
		errors.Is(err, domain.ErrManualCleanupRequired),
		errors.Is(err, domain.ErrComposeConfigValidation):
		h.writeErrorResponse(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrMissingGitRepository):
		h.writeErrorResponse(w, http.StatusInternalServerError, "Git repository adapter is not configured")
	case errors.Is(err, domain.ErrInvalidAppName):
		h.writeErrorResponse(w, http.StatusBadRequest, "App name is required")
	case errors.Is(err, domain.ErrComposeNoServices):
		h.writeErrorResponse(w, http.StatusBadRequest, "Compose YAML must contain a 'services:' key")
	case errors.Is(err, domain.ErrInvalidComposeYAML):
		h.writeErrorResponse(w, http.StatusBadRequest, "Compose YAML is invalid")
	case errors.Is(err, domain.ErrReservedIngressPort):
		h.writeErrorResponse(w, http.StatusBadRequest, err.Error())
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
