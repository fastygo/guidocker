package interfaces

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dashboard/domain"
)

type fakeDashboardUseCase struct {
	data     *domain.DashboardData
	getErr   error
	updateFn func(context.Context, string, string) error
}

type fakeAppUseCase struct {
	createFn    func(context.Context, string, string) (*domain.App, error)
	updateFn    func(context.Context, string, string, string) (*domain.App, error)
	deleteFn    func(context.Context, string) error
	getFn       func(context.Context, string) (*domain.App, error)
	listFn      func(context.Context) ([]*domain.App, error)
	deployFn    func(context.Context, string) error
	stopFn      func(context.Context, string) error
	restartFn   func(context.Context, string) error
	getStatusFn func(context.Context, string) (string, error)
	getLogsFn   func(context.Context, string, int) (string, error)
}

func (m *fakeDashboardUseCase) GetDashboardData(ctx context.Context) (*domain.DashboardData, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.data == nil {
		return &domain.DashboardData{}, nil
	}
	return m.data, nil
}

func (m *fakeDashboardUseCase) UpdateContainerStatus(ctx context.Context, containerID, status string) error {
	if m.updateFn == nil {
		return nil
	}
	return m.updateFn(ctx, containerID, status)
}

func (m *fakeDashboardUseCase) GetContainerByID(ctx context.Context, id string) (*domain.Container, error) {
	if m.data == nil {
		return nil, domain.ErrContainerNotFound
	}
	for _, container := range m.data.Containers {
		if container.ID == id {
			return &container, nil
		}
	}
	return nil, domain.ErrContainerNotFound
}

func (m *fakeAppUseCase) CreateApp(ctx context.Context, name, composeYAML string) (*domain.App, error) {
	if m.createFn != nil {
		return m.createFn(ctx, name, composeYAML)
	}
	return nil, nil
}

func (m *fakeAppUseCase) UpdateApp(ctx context.Context, id, name, composeYAML string) (*domain.App, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, name, composeYAML)
	}
	return nil, nil
}

func (m *fakeAppUseCase) DeleteApp(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *fakeAppUseCase) GetApp(ctx context.Context, id string) (*domain.App, error) {
	if m.getFn != nil {
		return m.getFn(ctx, id)
	}
	return nil, domain.ErrAppNotFound
}

func (m *fakeAppUseCase) ListApps(ctx context.Context) ([]*domain.App, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, nil
}

func (m *fakeAppUseCase) DeployApp(ctx context.Context, id string) error {
	if m.deployFn != nil {
		return m.deployFn(ctx, id)
	}
	return nil
}

func (m *fakeAppUseCase) StopApp(ctx context.Context, id string) error {
	if m.stopFn != nil {
		return m.stopFn(ctx, id)
	}
	return nil
}

func (m *fakeAppUseCase) RestartApp(ctx context.Context, id string) error {
	if m.restartFn != nil {
		return m.restartFn(ctx, id)
	}
	return nil
}

func (m *fakeAppUseCase) GetAppStatus(ctx context.Context, id string) (string, error) {
	if m.getStatusFn != nil {
		return m.getStatusFn(ctx, id)
	}
	return domain.AppStatusRunning, nil
}

func (m *fakeAppUseCase) GetAppLogs(ctx context.Context, id string, lines int) (string, error) {
	if m.getLogsFn != nil {
		return m.getLogsFn(ctx, id, lines)
	}
	return "", nil
}

func TestDashboardHandler_Dashboard_RendersHTML(t *testing.T) {
	useCase := &fakeDashboardUseCase{
		data: &domain.DashboardData{
			Title:    "Docker Container Dashboard",
			Subtitle: "Real-time container monitoring",
			Stats:    domain.Stats{TotalContainers: 1},
			Containers: []domain.Container{
				{Name: "nginx-web", Image: "nginx:alpine", Status: "running"},
			},
		},
	}

	handler := NewDashboardHandler(useCase)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	handler.Dashboard(recorder, request)

	response := recorder.Result()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "Docker Container Dashboard") {
		t.Fatalf("response should contain dashboard title")
	}
}

func TestDashboardHandler_Dashboard_ComposeCreateScreen(t *testing.T) {
	useCase := &fakeDashboardUseCase{
		data: &domain.DashboardData{
			Title:    "Docker Container Dashboard",
			Subtitle: "Real-time container monitoring",
			Containers: []domain.Container{
				{Name: "nginx-web", ID: "c1", Image: "nginx:alpine", Status: "running"},
			},
		},
	}

	handler := NewDashboardHandler(useCase)
	request := httptest.NewRequest(http.MethodGet, "/apps/new", nil)
	recorder := httptest.NewRecorder()

	handler.Dashboard(recorder, request)

	body := recorder.Body.String()
	if !strings.Contains(body, "Compose pack") {
		t.Fatalf("expected compose creation screen content")
	}
}

func TestDashboardHandler_Dashboard_ComposeEditScreen(t *testing.T) {
	useCase := &fakeDashboardUseCase{
		data: &domain.DashboardData{
			Containers: []domain.Container{
				{ID: "c1", Name: "nginx-web", Image: "nginx:alpine", Status: "running"},
			},
		},
	}

	handler := NewDashboardHandler(useCase)
	request := httptest.NewRequest(http.MethodGet, "/apps/c1/compose", nil)
	recorder := httptest.NewRecorder()

	handler.Dashboard(recorder, request)

	body := recorder.Body.String()
	if !strings.Contains(body, "App: c1") {
		t.Fatalf("expected compose edit screen for container")
	}
}

func TestDashboardHandler_Dashboard_LogsScreen(t *testing.T) {
	useCase := &fakeDashboardUseCase{
		data: &domain.DashboardData{
			Containers: []domain.Container{
				{ID: "c1", Name: "nginx-web", Image: "nginx:alpine", Status: "running"},
			},
		},
	}

	handler := NewDashboardHandler(useCase)
	request := httptest.NewRequest(http.MethodGet, "/apps/c1/logs", nil)
	recorder := httptest.NewRecorder()

	handler.Dashboard(recorder, request)

	body := recorder.Body.String()
	if !strings.Contains(body, "nginx-web logs") {
		t.Fatalf("expected logs screen content")
	}
}

func TestDashboardHandler_Dashboard_ServiceError(t *testing.T) {
	useCase := &fakeDashboardUseCase{getErr: errors.New("service failure")}
	handler := NewDashboardHandler(useCase)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	handler.Dashboard(recorder, request)

	if recorder.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500 on service error")
	}
}

func TestDashboardHandler_APIGetDashboard_ReturnsJSON(t *testing.T) {
	useCase := &fakeDashboardUseCase{
		data: &domain.DashboardData{
			Title: "JSON Dashboard",
			Stats: domain.Stats{TotalContainers: 1},
		},
	}
	handler := NewDashboardHandler(useCase)
	request := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	recorder := httptest.NewRecorder()

	handler.APIGetDashboard(recorder, request)

	response := recorder.Result()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var payload domain.DashboardData
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	if payload.Title != "JSON Dashboard" {
		t.Fatalf("unexpected title: %q", payload.Title)
	}
}

func TestDashboardHandler_APIGetDashboard_MethodNotAllowed(t *testing.T) {
	useCase := &fakeDashboardUseCase{
		data: &domain.DashboardData{Title: "JSON Dashboard"},
	}
	handler := NewDashboardHandler(useCase)
	request := httptest.NewRequest(http.MethodPost, "/api/dashboard", nil)
	recorder := httptest.NewRecorder()

	handler.APIGetDashboard(recorder, request)

	if recorder.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, recorder.Result().StatusCode)
	}
}

func TestDashboardHandler_APIUpdateContainer_Success(t *testing.T) {
	var updatedID string
	var updatedStatus string
	useCase := &fakeDashboardUseCase{
		updateFn: func(_ context.Context, id, status string) error {
			updatedID = id
			updatedStatus = status
			return nil
		},
	}
	handler := NewDashboardHandler(useCase)

	body, _ := json.Marshal(map[string]string{"status": "stop"})
	request := httptest.NewRequest(http.MethodPut, "/api/containers/web-app-01", bytes.NewBuffer(body))
	recorder := httptest.NewRecorder()

	handler.APIUpdateContainer(recorder, request)

	response := recorder.Result()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
	if updatedID != "web-app-01" {
		t.Fatalf("expected updated ID web-app-01, got %q", updatedID)
	}
	if updatedStatus != "stop" {
		t.Fatalf("expected updated status stop, got %q", updatedStatus)
	}

	var payload map[string]bool
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	if !payload["success"] {
		t.Fatal("expected success=true in response")
	}
}

func TestDashboardHandler_APIUpdateContainer_InvalidStatus(t *testing.T) {
	useCase := &fakeDashboardUseCase{
		updateFn: func(context.Context, string, string) error {
			return domain.ErrInvalidContainerStatus
		},
	}
	handler := NewDashboardHandler(useCase)

	body, _ := json.Marshal(map[string]string{"status": "broken"})
	request := httptest.NewRequest(http.MethodPut, "/api/containers/web-app-01", bytes.NewBuffer(body))
	recorder := httptest.NewRecorder()

	handler.APIUpdateContainer(recorder, request)

	if recorder.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Result().StatusCode)
	}
}

func TestDashboardHandler_APIUpdateContainer_MethodNotAllowed(t *testing.T) {
	handler := NewDashboardHandler(&fakeDashboardUseCase{})
	request := httptest.NewRequest(http.MethodGet, "/api/containers/web-app-01", nil)
	recorder := httptest.NewRecorder()

	handler.APIUpdateContainer(recorder, request)

	if recorder.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, recorder.Result().StatusCode)
	}
}

func TestDashboardHandler_APIUpdateContainer_NotFound(t *testing.T) {
	useCase := &fakeDashboardUseCase{
		updateFn: func(context.Context, string, string) error {
			return domain.ErrContainerNotFound
		},
	}
	handler := NewDashboardHandler(useCase)

	body, _ := json.Marshal(map[string]string{"status": "start"})
	request := httptest.NewRequest(http.MethodPut, "/api/containers/web-app-01", bytes.NewBuffer(body))
	recorder := httptest.NewRecorder()

	handler.APIUpdateContainer(recorder, request)

	if recorder.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", recorder.Result().StatusCode)
	}
}

func TestDashboardHandler_LoginScreen(t *testing.T) {
	handler := NewDashboardHandler(&fakeDashboardUseCase{})
	request := httptest.NewRequest(http.MethodGet, "/login", nil)
	recorder := httptest.NewRecorder()

	handler.Login(recorder, request)

	if recorder.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Result().StatusCode)
	}
	if !strings.Contains(recorder.Body.String(), "Sign in") {
		t.Fatalf("expected login screen to contain sign in text")
	}
}

func TestDashboardHandler_APIApps_Create(t *testing.T) {
	handler := NewDashboardHandler(&fakeDashboardUseCase{})
	handler.SetAppUseCase(&fakeAppUseCase{
		createFn: func(_ context.Context, name, composeYAML string) (*domain.App, error) {
			if name != "demo" {
				t.Fatalf("expected name demo, got %q", name)
			}
			if !strings.Contains(composeYAML, "services:") {
				t.Fatalf("expected compose payload to contain services section")
			}
			return &domain.App{ID: "app-1", Name: name, ComposeYAML: composeYAML, Status: domain.AppStatusCreated}, nil
		},
	})

	body := bytes.NewBufferString(`{"name":"demo","compose_yaml":"services:\n  web:\n    image: nginx:alpine"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/apps", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.APIApps(recorder, request)

	if recorder.Result().StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", recorder.Result().StatusCode)
	}

	var payload domain.App
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.ID != "app-1" {
		t.Fatalf("expected app id app-1, got %q", payload.ID)
	}
}

func TestDashboardHandler_APIDeploy_Success(t *testing.T) {
	deployedID := ""
	handler := NewDashboardHandler(&fakeDashboardUseCase{})
	handler.SetAppUseCase(&fakeAppUseCase{
		deployFn: func(_ context.Context, id string) error {
			deployedID = id
			return nil
		},
		getStatusFn: func(context.Context, string) (string, error) {
			return domain.AppStatusRunning, nil
		},
	})

	request := httptest.NewRequest(http.MethodPost, "/api/apps/app-1/deploy", nil)
	recorder := httptest.NewRecorder()

	handler.APIAppRoutes(recorder, request)

	if recorder.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Result().StatusCode)
	}
	if deployedID != "app-1" {
		t.Fatalf("expected deploy for app-1, got %q", deployedID)
	}

	var payload map[string]any
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload["success"] != true {
		t.Fatalf("expected success=true, got %+v", payload)
	}
	if payload["status"] != domain.AppStatusRunning {
		t.Fatalf("expected running status, got %+v", payload["status"])
	}
}
