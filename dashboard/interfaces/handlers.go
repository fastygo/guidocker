package interfaces

import (
	"dashboard/domain"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"fmt"
)

// DashboardHandler handles HTTP requests for dashboard
type DashboardHandler struct {
	dashboardUseCase domain.DashboardUseCase
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(useCase domain.DashboardUseCase) *DashboardHandler {
	return &DashboardHandler{
		dashboardUseCase: useCase,
	}
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
	dashboardData, err := h.dashboardUseCase.GetDashboardData(r.Context())
	if err != nil {
		log.Printf("Failed to get dashboard data: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	html := h.renderOverview(dashboardData)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (h *DashboardHandler) handleApps(w http.ResponseWriter, r *http.Request) {
	dashboardData, err := h.dashboardUseCase.GetDashboardData(r.Context())
	if err != nil {
		log.Printf("Failed to get dashboard data: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	html := h.renderApps(dashboardData)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
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

	container, err := h.dashboardUseCase.GetContainerByID(r.Context(), id)
	if err != nil {
		log.Printf("Failed to load app %s: %v", id, err)
		http.NotFound(w, r)
		return
	}

	html := h.renderAppDetail(container)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (h *DashboardHandler) handleComposeCreate(w http.ResponseWriter, r *http.Request) {
	html := h.renderComposeScreen(nil)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (h *DashboardHandler) handleComposeEdit(w http.ResponseWriter, r *http.Request, id string) {
	if id == "" {
		http.Redirect(w, r, "/apps/new", http.StatusSeeOther)
		return
	}

	container, err := h.dashboardUseCase.GetContainerByID(r.Context(), id)
	if err != nil {
		log.Printf("Failed to load app %s: %v", id, err)
		http.NotFound(w, r)
		return
	}

	html := h.renderComposeScreen(container)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (h *DashboardHandler) handleAppLogs(w http.ResponseWriter, r *http.Request, id string) {
	if id == "" {
		http.NotFound(w, r)
		return
	}

	container, err := h.dashboardUseCase.GetContainerByID(r.Context(), id)
	if err != nil {
		log.Printf("Failed to load app %s: %v", id, err)
		http.NotFound(w, r)
		return
	}

	html := h.renderAppLogs(container)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (h *DashboardHandler) handleSettings(w http.ResponseWriter, r *http.Request) {
	html := h.renderSettings()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
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

	dashboardData, err := h.dashboardUseCase.GetDashboardData(r.Context())
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

func (h *DashboardHandler) renderDashboardShell(data *domain.DashboardData, active string, content string) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>` + data.Title + `</title>
	<script src="https://cdn.jsdelivr.net/npm/@tailwindcss/browser@4"></script>
	<style type="text/tailwindcss">
	@import "tailwindcss";

	:root {
		--background: 0 0% 100%;
		--foreground: 222.2 47.4% 11.2%;
		--card: 0 0% 100%;
		--card-foreground: 222.2 47.4% 11.2%;
		--popover: 0 0% 100%;
		--popover-foreground: 222.2 47.4% 11.2%;
		--primary: 222.2 47.4% 11.2%;
		--primary-foreground: 210 40% 98%;
		--secondary: 210 40% 96.4%;
		--secondary-foreground: 222.2 47.4% 11.2%;
		--muted: 210 40% 96.4%;
		--muted-foreground: 215.4 16.3% 46.9%;
		--accent: 210 40% 96.4%;
		--accent-foreground: 222.2 47.4% 11.2%;
		--destructive: 0 84.2% 60.2%;
		--destructive-foreground: 210 40% 98%;
		--border: 214.3 31.8% 91.4%;
		--input: 214.3 31.8% 91.4%;
		--ring: 222.2 47.4% 11.2%;
	}

	.dark {
		--background: 222.2 84% 4.9%;
		--foreground: 210 40% 98%;
		--card: 222.2 84% 4.9%;
		--card-foreground: 210 40% 98%;
		--popover: 222.2 84% 4.9%;
		--popover-foreground: 210 40% 98%;
		--primary: 210 40% 98%;
		--primary-foreground: 222.2 47.4% 11.2%;
		--secondary: 217.2 32.6% 17.5%;
		--secondary-foreground: 210 40% 98%;
		--muted: 217.2 32.6% 17.5%;
		--muted-foreground: 215 20.2% 65.1%;
		--accent: 217.2 32.6% 17.5%;
		--accent-foreground: 210 40% 98%;
		--destructive: 0 62.8% 30.6%;
		--destructive-foreground: 210 40% 98%;
		--border: 217.2 32.6% 17.5%;
		--input: 217.2 32.6% 17.5%;
		--ring: 212.7 26.8% 83.9%;
	}

	body {
		background-color: hsl(var(--background));
		color: hsl(var(--foreground));
	}

	.bg-background { background-color: hsl(var(--background)) !important; }
	.text-foreground { color: hsl(var(--foreground)) !important; }
	.bg-card { background-color: hsl(var(--card)) !important; }
	.text-card-foreground { color: hsl(var(--card-foreground)) !important; }
	.bg-popover { background-color: hsl(var(--popover)) !important; }
	.text-popover-foreground { color: hsl(var(--popover-foreground)) !important; }
	.bg-primary { background-color: hsl(var(--primary)) !important; }
	.text-primary-foreground { color: hsl(var(--primary-foreground)) !important; }
	.bg-secondary { background-color: hsl(var(--secondary)) !important; }
	.text-secondary-foreground { color: hsl(var(--secondary-foreground)) !important; }
	.bg-muted { background-color: hsl(var(--muted)) !important; }
	.text-muted-foreground { color: hsl(var(--muted-foreground)) !important; }
	.bg-accent { background-color: hsl(var(--accent)) !important; }
	.text-accent-foreground { color: hsl(var(--accent-foreground)) !important; }
	.bg-destructive { background-color: hsl(var(--destructive)) !important; }
	.text-destructive-foreground { color: hsl(var(--destructive-foreground)) !important; }
	.border-border { border-color: hsl(var(--border)) !important; }
	.bg-input { background-color: hsl(var(--input)) !important; }
	.text-input { color: hsl(var(--input)) !important; }
	.focus\:bg-ring:focus { background-color: hsl(var(--ring)) !important; }
	.hover\:bg-accent:hover { background-color: hsl(var(--accent)) !important; }
	.hover\:text-accent:hover { color: hsl(var(--accent-foreground)) !important; }
	.dark .dashboard-lucide-icon { filter: invert(1); }
	</style>
	<script>
		(function() {
			const root = document.documentElement;
			const themeIcon = () => document.getElementById('theme-toggle-icon');
			const savedTheme = localStorage.getItem('dashboard-theme');
			const prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
			const shouldUseDark = savedTheme === 'dark' || (!savedTheme && prefersDark);
			const applyThemeIcon = function() {
				const icon = themeIcon();
				if (!icon) {
					return;
				}
				icon.src = root.classList.contains('dark')
					? 'https://unpkg.com/lucide-static@latest/icons/sun.svg'
					: 'https://unpkg.com/lucide-static@latest/icons/moon.svg';
			};

			root.classList.toggle('dark', shouldUseDark);
			applyThemeIcon();

			window.toggleDashboardTheme = function() {
				const isDark = root.classList.contains('dark');
				const nextDark = !isDark;
				root.classList.toggle('dark', nextDark);
				localStorage.setItem('dashboard-theme', nextDark ? 'dark' : 'light');
				applyThemeIcon();
			};

			document.addEventListener('DOMContentLoaded', applyThemeIcon);
		})();
	</script>
</head>
<body class="min-h-screen overflow-x-hidden bg-background text-foreground font-sans">
	<div class="fixed inset-0 hidden z-30" id="mobile-sidebar-backdrop" onclick="closeMobileSidebar()" style="background: rgba(0, 0, 0, 0.45);"></div>
	<aside class="fixed top-0 bottom-0 left-0 w-64 max-w-full border-r border-border bg-card hidden md:hidden z-40" id="mobile-sidebar">
		<div class="px-4 py-4 border-b border-border flex items-center justify-between">
			<div>
				<p class="text-sm text-muted-foreground">PaaS Console</p>
				<p class="text-lg font-semibold">Local Panel</p>
			</div>
			<button onclick="closeMobileSidebar()" class="rounded border border-border px-2 py-1 text-xs">Close</button>
		</div>
		<nav class="p-2 space-y-1">` + h.renderNavigation(active, true) + `</nav>
	</aside>
	<div class="flex min-h-screen w-full">
		<aside class="hidden md:block md:w-64 border-r border-border bg-card">
			<div class="px-4 py-4 border-b border-border">
				<p class="text-sm text-muted-foreground">PaaS Console</p>
				<p class="text-lg font-semibold">Local Panel</p>
			</div>
			<nav class="p-2 space-y-1">` + h.renderNavigation(active, false) + `</nav>
		</aside>
		<div class="flex-1 min-w-0 flex flex-col">
			<header class="flex items-center justify-between gap-2 border-b border-border px-4 py-3">
				<button onclick="openMobileSidebar()" class="rounded border border-border px-2 py-1 md:hidden" aria-label="Open menu">
					<img src="https://unpkg.com/lucide-static@latest/icons/menu.svg" alt="menu" class="h-4 w-4 dashboard-lucide-icon">
				</button>
				<h1 class="text-xl font-semibold truncate flex-1 px-2">` + data.Title + `</h1>
				<button onclick="toggleDashboardTheme()" class="h-8 w-8 inline-flex items-center justify-center rounded border border-border" aria-label="Toggle theme">
					<img id="theme-toggle-icon" src="https://unpkg.com/lucide-static@latest/icons/moon.svg" alt="Theme" class="h-4 w-4 dashboard-lucide-icon">
				</button>
			</header>
		<main class="w-full max-w-full min-w-0 p-4 md:p-6">
				` + content + `
			</main>
		</div>
	</div>
	<script>
		window.openMobileSidebar = function() {
			const sidebar = document.getElementById('mobile-sidebar');
			const backdrop = document.getElementById('mobile-sidebar-backdrop');
			if (!sidebar || !backdrop) {
				return;
			}
			sidebar.classList.remove('hidden');
			backdrop.classList.remove('hidden');
			document.body.style.overflow = 'hidden';
		};
		window.closeMobileSidebar = function() {
			const sidebar = document.getElementById('mobile-sidebar');
			const backdrop = document.getElementById('mobile-sidebar-backdrop');
			if (!sidebar || !backdrop) {
				return;
			}
			sidebar.classList.add('hidden');
			backdrop.classList.add('hidden');
			document.body.style.overflow = '';
		};
		window.addEventListener('keydown', function(event) {
			if (event && event.key === 'Escape') {
				closeMobileSidebar();
			}
		});
		const sidebarElement = document.getElementById('mobile-sidebar');
		if (sidebarElement) {
			let sidebarTouchStartX = 0;
			sidebarElement.addEventListener('touchstart', function(event) {
				if (event.touches.length > 0) {
					sidebarTouchStartX = event.touches[0].clientX;
				}
			}, { passive: true });
			sidebarElement.addEventListener('touchend', function(event) {
				if (!event.changedTouches || event.changedTouches.length === 0) {
					return;
				}
				const touchEndX = event.changedTouches[0].clientX;
				if (sidebarTouchStartX - touchEndX > 60) {
					closeMobileSidebar();
				}
			}, { passive: true });
		}

		window.showComposeMessage = function(message) {
			const composeTarget = document.getElementById('compose-status');
			if (composeTarget) {
				composeTarget.textContent = message;
				return;
			}
			const appFeedback = document.getElementById('app-feedback');
			if (appFeedback) {
				appFeedback.textContent = message;
			}
		};

		async function updateContainerStatus(containerID, status) {
			try {
				const response = await fetch('/api/containers/' + encodeURIComponent(containerID), {
					method: 'PUT',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ status }),
				});
				const data = await response.json();
				if (data && data.success) {
					location.reload();
				}
			} catch (err) {
				console.error('Action failed:', err);
			}
		}
	</script>
</body>
</html>`
}

func (h *DashboardHandler) renderOverview(data *domain.DashboardData) string {
	content := `<section class="space-y-4">
	<div class="text-sm text-muted-foreground">` + data.Subtitle + `</div>
	<div class="text-2xl font-semibold">Overview</div>
	<div class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">` + h.renderStatsCards(data.Stats) + `</div>
	<div class="rounded border border-border bg-card">
		<div class="px-4 py-3 border-b border-border">
			<div class="text-sm font-semibold">Recent containers</div>
		</div>
		<div class="overflow-x-auto">
			` + h.renderContainersTable(data.Containers) + `
		</div>
	</div>
</section>`

	return h.renderDashboardShell(data, "/", content)
}

func (h *DashboardHandler) renderApps(data *domain.DashboardData) string {
	var rows strings.Builder
	for _, container := range data.Containers {
		rows.WriteString(`<a href="/apps/` + container.ID + `" class="block rounded border border-border bg-card px-4 py-3 flex items-center justify-between text-sm gap-2 hover:bg-accent">
			<span class="font-medium truncate">` + container.Name + `</span>
			` + h.getStatusBadge(container.Status) + `
		</a>`)
	}

	content := `<section class="space-y-4">
		<div class="flex items-center justify-between">
			<div>
				<div class="text-sm text-muted-foreground">Managed containers</div>
				<div class="text-2xl font-semibold">Apps</div>
			</div>
			<a href="/apps/new" class="px-3 py-2 rounded border border-border text-sm">Create app</a>
		</div>
		<div class="grid gap-2">` + rows.String() + `</div>
	</section>`

	return h.renderDashboardShell(data, "/apps", content)
}

func (h *DashboardHandler) renderAppDetail(container *domain.Container) string {
	dummy := &domain.DashboardData{
		Title:    "Docker Container Dashboard",
		Subtitle: "Container details",
	}

	content := `<section class="space-y-4">
		<div>
			<h2 class="text-2xl font-semibold">` + container.Name + `</h2>
			<div class="text-sm text-muted-foreground">ID: ` + container.ID + `</div>
		</div>
		<div class="grid gap-3 md:grid-cols-2">
			<div class="rounded border border-border bg-card p-4">
				<div class="font-semibold">Runtime</div>
				<p class="text-sm text-muted-foreground">Image: ` + container.Image + `</p>
				<p class="text-sm text-muted-foreground">Ports: ` + strings.Join(container.Ports, ", ") + `</p>
				<p class="text-sm text-muted-foreground">CPU: ` + container.GetCPUUsagePercent() + ` / Memory: ` + container.GetMemoryUsageMB() + `</p>
				<div class="mt-3">` + h.getStatusBadge(container.Status) + `</div>
			</div>
			<div class="rounded border border-border bg-card p-4">
				<div class="font-semibold">Actions</div>
				<div class="mt-2">` + h.renderContainerActions(*container) + `</div>
				<div class="mt-3 space-y-2">
					<a href="/apps/` + container.ID + `/compose" class="inline-block px-3 py-2 rounded border border-border text-sm">Compose</a><br />
					<a href="/apps/` + container.ID + `/logs" class="inline-block px-3 py-2 rounded border border-border text-sm">Logs</a>
				</div>
			</div>
		</div>
	</section>`

	return h.renderDashboardShell(dummy, "/apps", content)
}

func (h *DashboardHandler) renderComposeScreen(container *domain.Container) string {
	title := "Create application"
	composeTemplate := `version: "3.9"
services:
  app:
    image: nginx:alpine
    ports:
      - "8080:80"
    restart: unless-stopped`
	name := ""
	image := ""
	subtitle := "New app compose package"

	if container != nil {
		title = "Edit application"
		name = container.Name
		image = container.Image
		composeTemplate = `version: "3.9"
services:
  ` + container.Name + `:
    image: ` + container.Image + `
    ports:
      - "80:80"
    environment:
      - TZ=UTC
    restart: unless-stopped`
		subtitle = `App: ` + container.ID
	}

	data := &domain.DashboardData{
		Title:    "Docker Container Dashboard",
		Subtitle: subtitle,
	}

	content := `<section class="space-y-4">
		<div>
			<div class="text-sm text-muted-foreground">` + subtitle + `</div>
			<div class="text-2xl font-semibold">` + title + `</div>
		</div>
		<div class="grid gap-3 md:grid-cols-2">
			<div class="rounded border border-border bg-card p-4">
				<div class="font-semibold">Compose pack</div>
				<form onsubmit="event.preventDefault(); showComposeMessage('Compose saved to local draft');" class="space-y-3">
					<div>
						<label class="text-sm text-muted-foreground">App name</label>
						<input class="mt-1 w-full rounded border border-border bg-background px-2 py-2 text-sm" value="` + name + `" />
					</div>
					<div>
						<label class="text-sm text-muted-foreground">Service image</label>
						<input class="mt-1 w-full rounded border border-border bg-background px-2 py-2 text-sm" value="` + image + `" />
					</div>
					<div>
						<label class="text-sm text-muted-foreground">Docker Compose yaml</label>
						<textarea class="mt-1 w-full rounded border border-border bg-background p-2 text-sm" rows="14">` + composeTemplate + `</textarea>
					</div>
					<div id="compose-status" class="text-sm text-muted-foreground"></div>
					<div class="flex gap-2">
						<button type="submit" class="px-3 py-2 rounded border border-border text-sm">Save compose</button>
						<button type="button" onclick="showComposeMessage('Compose deployment started')" class="px-3 py-2 rounded border border-border text-sm">Deploy now</button>
					</div>
				</form>
			</div>
			<div class="rounded border border-border bg-card p-4">
				<div class="font-semibold">Shortcuts</div>
				<div class="text-sm text-muted-foreground space-y-2 mt-3">
					<p>Use this screen as a blueprint for full compose-driven deployment.</p>
					<p>You can paste env vars, volumes, networks and secrets manually into yaml.</p>
				</div>
			</div>
		</div>
	</section>`

	return h.renderDashboardShell(data, "/apps", content)
}

func (h *DashboardHandler) renderAppLogs(container *domain.Container) string {
	logItems := []string{
		"2026-03-12T09:00:01Z | info | " + container.Name + " started",
		"2026-03-12T09:00:02Z | info | pulling image " + container.Image,
		"2026-03-12T09:00:05Z | warn | high memory usage detected",
		"2026-03-12T09:00:07Z | info | health check passed",
		"2026-03-12T09:00:10Z | debug | request latency avg=12ms",
	}

	var logs strings.Builder
	logs.WriteString(`<div class="rounded border border-border bg-card p-4 overflow-x-auto">`)
	logs.WriteString(`<pre class="text-xs text-muted-foreground">`)
	for _, item := range logItems {
		logs.WriteString(item + "\n")
	}
	logs.WriteString(`</pre></div>`)

	data := &domain.DashboardData{
		Title:    "Docker Container Dashboard",
		Subtitle: "Application logs",
	}

	content := `<section class="space-y-4">
		<div>
			<div class="text-sm text-muted-foreground">` + container.ID + `</div>
			<div class="text-2xl font-semibold">` + container.Name + ` logs</div>
			<div class="text-sm text-muted-foreground">Recent runtime log output</div>
		</div>
		<div id="app-feedback" class="text-sm text-muted-foreground"></div>
		<div class="flex gap-2">
			<button onclick="showComposeMessage('Logs refreshed')" class="px-3 py-2 rounded border border-border text-sm">Refresh</button>
			<a href="/apps/` + container.ID + `/compose" class="px-3 py-2 rounded border border-border text-sm">Open compose</a>
		</div>
		` + logs.String() + `
	</section>`

	return h.renderDashboardShell(data, "/apps", content)
}

func (h *DashboardHandler) renderSettings() string {
	data := &domain.DashboardData{
		Title:    "Docker Container Dashboard",
		Subtitle: "Panel settings",
	}

	content := `<section class="space-y-4">
		<div>
			<div class="text-2xl font-semibold">Settings</div>
			<div class="text-sm text-muted-foreground">Placeholders for core PAAS settings modules.</div>
		</div>
		<div class="grid gap-3 md:grid-cols-2">
			<div class="rounded border border-border bg-card p-4">
				<div class="font-semibold">Display</div>
				<p class="text-sm text-muted-foreground">Dark mode is controlled by the button in the header.</p>
			</div>
			<div class="rounded border border-border bg-card p-4">
				<div class="font-semibold">Security</div>
				<p class="text-sm text-muted-foreground">Token access and audit controls are next sprint.</p>
			</div>
		</div>
	</section>`

	return h.renderDashboardShell(data, "/settings", content)
}

func (h *DashboardHandler) renderNavigation(active string, closeOnClick bool) string {
	items := []struct {
		href string
		name string
		icon string
	}{
		{"/", "Overview", "house"},
		{"/apps", "Apps", "box"},
		{"/settings", "Settings", "settings"},
	}

	var sb strings.Builder
	for _, item := range items {
		className := "flex items-center gap-2 rounded px-3 py-2 text-sm"
		if item.href == active {
			className += " bg-accent text-accent-foreground"
		} else {
			className += " text-muted-foreground hover:bg-accent"
		}
		clickAction := ""
		if closeOnClick {
			clickAction = ` onclick="closeMobileSidebar()"`
		}

		sb.WriteString(fmt.Sprintf(`<a href="%s" class="%s"%s><img src="https://unpkg.com/lucide-static@latest/icons/%s.svg" alt="%s" class="w-4 h-4 dashboard-lucide-icon" /> <span>%s</span></a>`,
			item.href,
			className,
			clickAction,
			item.icon,
			item.name,
			item.name,
		))
	}

	return sb.String()
}

// renderStatsCards generates HTML for statistics cards
func (h *DashboardHandler) renderStatsCards(stats domain.Stats) string {
	cards := []struct {
		title string
		value int
		icon  string
	}{
		{"Total Containers", stats.TotalContainers, "📦"},
		{"Running", stats.RunningContainers, "▶️"},
		{"Stopped", stats.StoppedContainers, "⏹️"},
		{"Paused", stats.PausedContainers, "⏸️"},
	}

	var sb strings.Builder
	for _, card := range cards {
		sb.WriteString(fmt.Sprintf(`
            <article class="rounded border border-border bg-card p-4">
                <div class="text-sm text-muted-foreground">%s %s</div>
                <p class="text-3xl font-semibold mt-2">%d</p>
            </article>`,
			card.icon,
			card.title,
			card.value,
		))
	}
	return sb.String()
}

// renderContainersTable generates HTML for containers table
func (h *DashboardHandler) renderContainersTable(containers []domain.Container) string {
	var sb strings.Builder

	sb.WriteString(`<table class="w-full">
        <thead class="bg-muted">
            <tr>
                <th class="px-4 py-3 text-left text-xs font-semibold text-muted-foreground min-w-0">Name</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-muted-foreground min-w-0">Image</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-muted-foreground min-w-0">Status</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-muted-foreground min-w-0">Ports</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-muted-foreground min-w-0">CPU</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-muted-foreground min-w-0">Memory</th>
                <th class="px-4 py-3 text-left text-xs font-semibold text-muted-foreground">Actions</th>
            </tr>
        </thead>
        <tbody class="divide-y divide-border">`)

	for _, container := range containers {
		sb.WriteString(h.renderContainerRow(container))
	}

	sb.WriteString(`</tbody></table>`)
	return sb.String()
}

// renderContainerRow generates HTML for a single container row
func (h *DashboardHandler) renderContainerRow(container domain.Container) string {
	statusBadge := h.getStatusBadge(container.Status)

	return fmt.Sprintf(`<tr>
        <td class="px-4 py-3 text-sm font-medium min-w-0"><a href="/apps/%s" class="truncate block">%s</a></td>
        <td class="px-4 py-3 text-sm text-muted-foreground min-w-0 truncate">%s</td>
        <td class="px-4 py-3 text-sm min-w-0 truncate">%s</td>
        <td class="px-4 py-3 text-sm text-muted-foreground min-w-0 truncate">%s</td>
        <td class="px-4 py-3 text-sm text-muted-foreground min-w-0 truncate">%s</td>
        <td class="px-4 py-3 text-sm text-muted-foreground min-w-0 truncate">%s</td>
        <td class="px-4 py-3 text-sm min-w-0">%s</td>
    </tr>`,
		container.ID,
		container.Name,
		container.Image,
		statusBadge,
		strings.Join(container.Ports, ", "),
		container.GetCPUUsagePercent(),
		container.GetMemoryUsageMB(),
		h.renderContainerActions(container),
	)
}

// getStatusBadge returns HTML for status badge
func (h *DashboardHandler) getStatusBadge(status string) string {
	var badgeClass string
	normalizedStatus := domain.NormalizeStoredStatus(status)

	switch normalizedStatus {
	case domain.ContainerStatusRunning:
		badgeClass = "inline-flex rounded bg-primary px-2 py-1 text-xs text-primary-foreground"
	case domain.ContainerStatusStopped:
		badgeClass = "inline-flex rounded bg-muted px-2 py-1 text-xs text-muted-foreground"
	case domain.ContainerStatusPaused:
		badgeClass = "inline-flex rounded bg-accent px-2 py-1 text-xs text-accent-foreground"
	default:
		badgeClass = "inline-flex rounded bg-secondary px-2 py-1 text-xs text-secondary-foreground"
	}

	return fmt.Sprintf(`<span class="%s">%s</span>`,
		badgeClass,
		domain.FormatStatusLabel(normalizedStatus),
	)
}

// renderContainerActions generates HTML for container action buttons
func (h *DashboardHandler) renderContainerActions(container domain.Container) string {
	var actions []string
	normalizedStatus := domain.NormalizeStoredStatus(container.Status)

	switch normalizedStatus {
	case domain.ContainerStatusRunning:
		actions = []string{"stop", "pause", "restart"}
	case domain.ContainerStatusStopped:
		actions = []string{"start"}
	case domain.ContainerStatusPaused:
		actions = []string{"unpause", "stop"}
	}

	var sb strings.Builder
	for _, action := range actions {
		var btnClass string

		switch action {
		case "start":
			btnClass = "px-3 py-2 text-sm rounded border border-border bg-primary text-primary-foreground"
		case "stop":
			btnClass = "px-3 py-2 text-sm rounded border border-destructive bg-destructive text-destructive-foreground"
		case "pause":
			btnClass = "px-3 py-2 text-sm rounded border border-border bg-muted text-muted-foreground"
		case "unpause", "restart":
			btnClass = "px-3 py-2 text-sm rounded border border-border bg-accent text-accent-foreground"
		}

		sb.WriteString(fmt.Sprintf(`<button class="%s" onclick="updateContainerStatus('%s', '%s')">%s</button>`,
			btnClass,
			container.ID,
			action,
			domain.FormatStatusLabel(action),
		))
	}

	return sb.String()
}
