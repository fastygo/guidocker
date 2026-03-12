package interfaces

import "net/http"

func RegisterRoutes(mux *http.ServeMux, handler *DashboardHandler) {
	mux.HandleFunc("/", handler.Dashboard)
	mux.HandleFunc("/login", handler.Login)
	mux.HandleFunc("/api/dashboard", handler.APIGetDashboard)
	mux.HandleFunc("/api/apps", handler.APIApps)
	mux.HandleFunc("/api/apps/", handler.APIAppRoutes)
	mux.HandleFunc("/api/containers/", handler.APIUpdateContainer)
}
