package interfaces

import (
	"context"
	"dashboard/domain"
	"dashboard/pkg/twsx"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// DashboardHandler handles HTTP requests for dashboard
type DashboardHandler struct {
	dashboardUseCase domain.DashboardUseCase
	styleRegistry    *twsx.StyleRegistry
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(useCase domain.DashboardUseCase) *DashboardHandler {
	return &DashboardHandler{
		dashboardUseCase: useCase,
		styleRegistry:    twsx.NewStyleRegistry(),
	}
}

// Dashboard serves the main dashboard page
func (h *DashboardHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	dashboardData, err := h.dashboardUseCase.GetDashboardData(context.Background())
	if err != nil {
		log.Printf("Failed to get dashboard data: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Generate HTML with inline styles
	html := h.renderDashboard(dashboardData)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// APIGetDashboard returns dashboard data as JSON
func (h *DashboardHandler) APIGetDashboard(w http.ResponseWriter, r *http.Request) {
	dashboardData, err := h.dashboardUseCase.GetDashboardData(context.Background())
	if err != nil {
		log.Printf("Failed to get dashboard data: %v", err)
		http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(dashboardData)
	if err != nil {
		log.Printf("Failed to marshal dashboard data: %v", err)
		http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

// APIUpdateContainer updates container status
func (h *DashboardHandler) APIUpdateContainer(w http.ResponseWriter, r *http.Request) {
	containerID := strings.TrimPrefix(r.URL.Path, "/api/containers/")
	if containerID == "" {
		http.Error(w, `{"error": "Container ID required"}`, http.StatusBadRequest)
		return
	}

	var request struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if err := h.dashboardUseCase.UpdateContainerStatus(context.Background(), containerID, request.Status); err != nil {
		log.Printf("Failed to update container status: %v", err)
		http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": true}`))
}

// renderDashboard generates HTML for dashboard with semantic CSS classes
func (h *DashboardHandler) renderDashboard(data *domain.DashboardData) string {
	// Pre-register all CSS classes to ensure they're available for all render functions
	h.registerAllStyles()

	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + data.Title + `</title>
    <style>` + h.styleRegistry.GenerateCSS() + `</style>
</head>
<body class="body">
    <!-- Header -->
    <header class="header">
        <div class="container">
            <h1 class="title">` + data.Title + `</h1>
            <p class="subtitle">` + data.Subtitle + `</p>
        </div>
    </header>

    <!-- Main Content -->
    <main class="main">
        <!-- Stats Cards -->
        <div class="stats-grid">
            ` + h.renderStatsCards(data.Stats) + `
        </div>

        <!-- Containers Section -->
        <div class="containers-section">
            <div class="containers-header">
                <h2 class="section-title">Containers</h2>
            </div>
            <div class="table-container">
                ` + h.renderContainersTable(data.Containers) + `
            </div>
        </div>
    </main>

    <!-- Footer -->
    <footer class="footer">
        <div class="footer-content">
            <p class="footer-text">Dashboard powered by Go & semantic CSS</p>
        </div>
    </footer>

    <script>async function updateContainerStatus(e,t){try{const n=await fetch("/api/containers/"+e,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify({status:t})}),a=await n.json();a.success&&location.reload()}catch(n){console.error("Error:",n)}}</script>
</body>
</html>`)

	return sb.String()
}

// registerAllStyles pre-registers all CSS classes used in the dashboard
func (h *DashboardHandler) registerAllStyles() {
	// Base layout styles
	h.styleRegistry.CLASS("body", twsx.TWSX("min-h-screen bg-gray-50 font-sans"))
	h.styleRegistry.CLASS("header", twsx.TWSX("bg-blue-600 text-white shadow-lg"))
	h.styleRegistry.CLASS("container", twsx.TWSX("max-w-7xl mx-auto px-4 py-6"))
	h.styleRegistry.CLASS("title", twsx.TWSX("text-3xl font-bold"))
	h.styleRegistry.CLASS("subtitle", twsx.TWSX("text-blue-100 mt-2"))

	// Main content
	h.styleRegistry.CLASS("main", twsx.TWSX("max-w-7xl mx-auto px-4 py-8"))
	h.styleRegistry.CLASS("stats-grid", twsx.TWSX("grid gap-6 mb-8"))

	// Stats cards
	h.styleRegistry.CLASS("stats-card", twsx.TWSX("bg-white rounded-lg shadow p-6"))
	h.styleRegistry.CLASS("stats-card-header", twsx.TWSX("flex items-center justify-between"))
	h.styleRegistry.CLASS("stats-card-icon", twsx.TWSX("text-2xl"))
	h.styleRegistry.CLASS("stats-card-value", twsx.TWSX("text-3xl font-bold text-gray-800"))
	h.styleRegistry.CLASS("stats-card-label", twsx.TWSX("text-sm text-gray-600 mt-2"))

	// Containers section
	h.styleRegistry.CLASS("containers-section", twsx.TWSX("bg-white rounded-lg shadow-lg overflow-hidden"))
	h.styleRegistry.CLASS("containers-header", twsx.TWSX("px-6 py-4 bg-gray-50 border-b"))
	h.styleRegistry.CLASS("section-title", twsx.TWSX("text-xl font-semibold text-gray-800"))
	h.styleRegistry.CLASS("table-container", twsx.TWSX("overflow-x-auto"))

	// Table styles
	h.styleRegistry.CLASS("table", twsx.TWSX("w-full"))
	h.styleRegistry.CLASS("table-header", twsx.TWSX("bg-gray-50"))
	h.styleRegistry.CLASS("table-header-cell", twsx.TWSX("px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"))
	h.styleRegistry.CLASS("table-body", twsx.TWSX("bg-white divide-y divide-gray-200"))
	h.styleRegistry.CLASS("table-cell", twsx.TWSX("px-6 py-4 whitespace-nowrap text-sm text-gray-500"))
	h.styleRegistry.CLASS("table-cell-primary", twsx.TWSX("px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900"))
	h.styleRegistry.CLASS("table-cell-status", twsx.TWSX("px-6 py-4 whitespace-nowrap"))
	h.styleRegistry.CLASS("table-cell-actions", twsx.TWSX("px-6 py-4 whitespace-nowrap text-sm font-medium"))

	// Status badges
	h.styleRegistry.CLASS("status-running", twsx.TWSX("px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-green-100 text-green-800"))
	h.styleRegistry.CLASS("status-stopped", twsx.TWSX("px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-red-100 text-red-800"))
	h.styleRegistry.CLASS("status-paused", twsx.TWSX("px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-yellow-100 text-yellow-800"))
	h.styleRegistry.CLASS("status-unknown", twsx.TWSX("px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-gray-100 text-gray-800"))

	// Buttons
	h.styleRegistry.CLASS("btn", twsx.TWSX("px-3 py-1 text-xs font-medium rounded mr-2 text-white cursor-pointer"))
	h.styleRegistry.CLASS("btn-start", twsx.TWSX("bg-green-600"))
	h.styleRegistry.CLASS("btn-stop", twsx.TWSX("bg-red-600"))
	h.styleRegistry.CLASS("btn-pause", twsx.TWSX("bg-yellow-600"))
	h.styleRegistry.CLASS("btn-restart", twsx.TWSX("bg-blue-600"))

	// Footer
	h.styleRegistry.CLASS("footer", twsx.TWSX("bg-gray-800 text-white mt-12"))
	h.styleRegistry.CLASS("footer-content", twsx.TWSX("max-w-7xl mx-auto px-4 py-6 text-center"))
	h.styleRegistry.CLASS("footer-text", twsx.TWSX("text-gray-400"))
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
            <div class="stats-card">
                <div class="stats-card-header">
                    <div class="stats-card-icon">%s</div>
                    <div class="stats-card-value">%d</div>
                </div>
                <div class="stats-card-label">%s</div>
            </div>`,
			card.icon,
			card.value,
			card.title,
		))
	}
	return sb.String()
}

// renderContainersTable generates HTML for containers table
func (h *DashboardHandler) renderContainersTable(containers []domain.Container) string {
	var sb strings.Builder

	sb.WriteString(`<table class="table">
        <thead class="table-header">
            <tr>
                <th class="table-header-cell">Name</th>
                <th class="table-header-cell">Image</th>
                <th class="table-header-cell">Status</th>
                <th class="table-header-cell">Ports</th>
                <th class="table-header-cell">CPU</th>
                <th class="table-header-cell">Memory</th>
                <th class="table-header-cell">Actions</th>
            </tr>
        </thead>
        <tbody class="table-body">`)

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
        <td class="table-cell-primary">%s</td>
        <td class="table-cell">%s</td>
        <td class="table-cell-status">%s</td>
        <td class="table-cell">%s</td>
        <td class="table-cell">%s</td>
        <td class="table-cell">%s</td>
        <td class="table-cell-actions">%s</td>
    </tr>`,
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

	switch status {
	case "running":
		badgeClass = "status-running"
	case "stopped":
		badgeClass = "status-stopped"
	case "paused":
		badgeClass = "status-paused"
	default:
		badgeClass = "status-unknown"
	}

	return fmt.Sprintf(`<span class="%s">%s</span>`,
		badgeClass,
		strings.Title(status),
	)
}

// renderContainerActions generates HTML for container action buttons
func (h *DashboardHandler) renderContainerActions(container domain.Container) string {
	var actions []string

	switch container.Status {
	case "running":
		actions = []string{"stop", "pause", "restart"}
	case "stopped":
		actions = []string{"start"}
	case "paused":
		actions = []string{"unpause", "stop"}
	}

	var sb strings.Builder
	for _, action := range actions {
		var btnClass string

		switch action {
		case "start":
			btnClass = "btn btn-start"
		case "stop":
			btnClass = "btn btn-stop"
		case "pause":
			btnClass = "btn btn-pause"
		case "unpause", "restart":
			btnClass = "btn btn-restart"
		}

		sb.WriteString(fmt.Sprintf(`<button class="%s" onclick="updateContainerStatus('%s', '%s')">%s</button>`,
			btnClass,
			container.ID,
			action,
			strings.Title(action),
		))
	}

	return sb.String()
}
