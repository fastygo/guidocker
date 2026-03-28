package interfaces

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterAPIRoutes_OnlyRegistersAPIEndpoints(t *testing.T) {
	t.Setenv("DASHBOARD_MODE", "api")

	mux := http.NewServeMux()
	RegisterAPIRoutes(mux, newTestHandler(&fakeDashboardUseCase{}))

	healthRequest := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	healthRecorder := httptest.NewRecorder()
	mux.ServeHTTP(healthRecorder, healthRequest)
	if healthRecorder.Code != http.StatusOK {
		t.Fatalf("expected /api/health status 200, got %d", healthRecorder.Code)
	}

	var payload map[string]string
	if err := json.NewDecoder(healthRecorder.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}
	if payload["mode"] != "api" {
		t.Fatalf("expected mode api, got %+v", payload)
	}

	rootRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	rootRecorder := httptest.NewRecorder()
	mux.ServeHTTP(rootRecorder, rootRequest)
	if rootRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected root status 404, got %d", rootRecorder.Code)
	}

	loginRequest := httptest.NewRequest(http.MethodGet, "/login", nil)
	loginRecorder := httptest.NewRecorder()
	mux.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected login status 404, got %d", loginRecorder.Code)
	}
}

func TestRegisterRoutes_RegistersGUIAndAPIEndpoints(t *testing.T) {
	t.Setenv("DASHBOARD_MODE", "gui")

	mux := http.NewServeMux()
	RegisterRoutes(mux, newTestHandler(&fakeDashboardUseCase{}))

	loginRequest := httptest.NewRequest(http.MethodGet, "/login", nil)
	loginRecorder := httptest.NewRecorder()
	mux.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d", loginRecorder.Code)
	}

	healthRequest := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	healthRecorder := httptest.NewRecorder()
	mux.ServeHTTP(healthRecorder, healthRequest)
	if healthRecorder.Code != http.StatusOK {
		t.Fatalf("expected /api/health status 200, got %d", healthRecorder.Code)
	}
}
