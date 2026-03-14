package interfaces

import "net/http"

func RegisterRoutes(mux *http.ServeMux, handler *DashboardHandler) {
	// Static assets (compiled CSS) - relative to working dir when run from dashboard/
	mux.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("static"))))

	mux.HandleFunc("/", handler.Dashboard)
	mux.HandleFunc("/login", handler.Login)
	mux.HandleFunc("/api/dashboard", handler.APIGetDashboard)
	mux.HandleFunc("/api/apps", handler.APIApps)
	mux.HandleFunc("/api/apps/import", handler.APIImport)
	mux.HandleFunc("/api/apps/", handler.APIAppRoutes)
	mux.HandleFunc("/api/containers/", handler.APIUpdateContainer)
	mux.HandleFunc("/scan", handler.HandleScan)
	mux.HandleFunc("/api/scan", handler.APIScan)
}
