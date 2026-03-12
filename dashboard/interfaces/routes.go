package interfaces

import "net/http"

func RegisterRoutes(mux *http.ServeMux, handler *DashboardHandler) {
	mux.HandleFunc("/", handler.Dashboard)
	mux.HandleFunc("/api/dashboard", handler.APIGetDashboard)
	mux.HandleFunc("/api/containers/", handler.APIUpdateContainer)
}
