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
	"dashboard/views"
)

var testRenderer *views.Renderer

func init() {
	var err error
	testRenderer, err = views.NewRenderer()
	if err != nil {
		panic("test renderer: " + err.Error())
	}
}

func newTestHandler(useCase domain.DashboardUseCase) *DashboardHandler {
	return NewDashboardHandler(useCase, testRenderer)
}

type fakeDashboardUseCase struct {
	data     *domain.DashboardData
	getErr   error
	updateFn func(context.Context, string, string) error
}

type fakeAppUseCase struct {
	createFn    func(context.Context, string, string) (*domain.App, error)
	updateFn    func(context.Context, string, string, string) (*domain.App, error)
	updateConfigFn func(context.Context, string, domain.AppConfig) (*domain.App, error)
	importFn    func(context.Context, domain.ImportRepoInput) (*domain.App, error)
	deleteFn    func(context.Context, string) error
	getFn       func(context.Context, string) (*domain.App, error)
	listFn      func(context.Context) ([]*domain.App, error)
	deployFn    func(context.Context, string) error
	stopFn      func(context.Context, string) error
	restartFn   func(context.Context, string) error
	getStatusFn func(context.Context, string) (string, error)
	getLogsFn   func(context.Context, *domain.App, int) (string, error)
}

type fakePlatformSettingsUseCase struct {
	getSettingsFn  func(context.Context) (*domain.PlatformSettings, error)
	updateSettingsFn func(context.Context, domain.PlatformSettings) (*domain.PlatformSettings, error)
}

func (m *fakePlatformSettingsUseCase) GetPlatformSettings(ctx context.Context) (*domain.PlatformSettings, error) {
	if m.getSettingsFn != nil {
		return m.getSettingsFn(ctx)
	}
	return nil, domain.ErrMissingPlatformSettingsRepository
}

func (m *fakePlatformSettingsUseCase) UpdatePlatformSettings(ctx context.Context, settings domain.PlatformSettings) (*domain.PlatformSettings, error) {
	if m.updateSettingsFn != nil {
		return m.updateSettingsFn(ctx, settings)
	}
	return nil, domain.ErrMissingPlatformSettingsRepository
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

func (m *fakeAppUseCase) UpdateAppConfig(ctx context.Context, id string, config domain.AppConfig) (*domain.App, error) {
	if m.updateConfigFn != nil {
		return m.updateConfigFn(ctx, id, config)
	}
	return nil, nil
}

func (m *fakeAppUseCase) ImportRepo(ctx context.Context, input domain.ImportRepoInput) (*domain.App, error) {
	if m.importFn != nil {
		return m.importFn(ctx, input)
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

func (m *fakeAppUseCase) GetAppLogs(ctx context.Context, app *domain.App, lines int) (string, error) {
	if m.getLogsFn != nil {
		return m.getLogsFn(ctx, app, lines)
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

	handler := newTestHandler(useCase)
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

	handler := newTestHandler(useCase)
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

	handler := newTestHandler(useCase)
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

	handler := newTestHandler(useCase)
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
	handler := newTestHandler(useCase)
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
	handler := newTestHandler(useCase)
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
	handler := newTestHandler(useCase)
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
	handler := newTestHandler(useCase)

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
	handler := newTestHandler(useCase)

	body, _ := json.Marshal(map[string]string{"status": "broken"})
	request := httptest.NewRequest(http.MethodPut, "/api/containers/web-app-01", bytes.NewBuffer(body))
	recorder := httptest.NewRecorder()

	handler.APIUpdateContainer(recorder, request)

	if recorder.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Result().StatusCode)
	}
}

func TestDashboardHandler_APIUpdateContainer_MethodNotAllowed(t *testing.T) {
	handler := newTestHandler(&fakeDashboardUseCase{})
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
	handler := newTestHandler(useCase)

	body, _ := json.Marshal(map[string]string{"status": "start"})
	request := httptest.NewRequest(http.MethodPut, "/api/containers/web-app-01", bytes.NewBuffer(body))
	recorder := httptest.NewRecorder()

	handler.APIUpdateContainer(recorder, request)

	if recorder.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", recorder.Result().StatusCode)
	}
}

func TestDashboardHandler_LoginScreen(t *testing.T) {
	handler := newTestHandler(&fakeDashboardUseCase{})
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
	handler := newTestHandler(&fakeDashboardUseCase{})
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

func TestDashboardHandler_APIApps_Create_NoServices_ReturnsBadRequest(t *testing.T) {
	handler := newTestHandler(&fakeDashboardUseCase{})
	handler.SetAppUseCase(&fakeAppUseCase{
		createFn: func(_ context.Context, name, composeYAML string) (*domain.App, error) {
			return nil, domain.ErrComposeNoServices
		},
	})

	body := bytes.NewBufferString(`{"name":"demo","compose_yaml":"services_wrong_syntax"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/apps", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.APIApps(recorder, request)

	if recorder.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Result().StatusCode)
	}

	var payload map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload["error"] != "Compose YAML must contain a 'services:' key" {
		t.Fatalf("unexpected error message: %q", payload["error"])
	}
}

func TestDashboardHandler_APIApps_Create_ReservedIngressPort_ReturnsBadRequest(t *testing.T) {
	handler := newTestHandler(&fakeDashboardUseCase{})
	handler.SetAppUseCase(&fakeAppUseCase{
		createFn: func(_ context.Context, name, composeYAML string) (*domain.App, error) {
			return nil, domain.ErrReservedIngressPort
		},
	})

	body := bytes.NewBufferString(`{"name":"demo","compose_yaml":"services:\n  web:\n    image: nginx:alpine\n    ports:\n      - \"80:80\""}`)
	request := httptest.NewRequest(http.MethodPost, "/api/apps", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.APIApps(recorder, request)

	if recorder.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Result().StatusCode)
	}
}

func TestDashboardHandler_APIAppDelete_Success(t *testing.T) {
	var deletedID string
	handler := newTestHandler(&fakeDashboardUseCase{})
	handler.SetAppUseCase(&fakeAppUseCase{
		deleteFn: func(_ context.Context, id string) error {
			deletedID = id
			return nil
		},
	})

	request := httptest.NewRequest(http.MethodDelete, "/api/apps/app-1", nil)
	recorder := httptest.NewRecorder()

	handler.APIAppRoutes(recorder, request)

	response := recorder.Result()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var payload map[string]bool
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !payload["success"] {
		t.Fatalf("expected success=true")
	}
	if deletedID != "app-1" {
		t.Fatalf("expected deleted app app-1, got %q", deletedID)
	}
}

func TestDashboardHandler_APIAppDelete_NotFound(t *testing.T) {
	handler := newTestHandler(&fakeDashboardUseCase{})
	handler.SetAppUseCase(&fakeAppUseCase{
		deleteFn: func(_ context.Context, id string) error {
			if id != "missing-app" {
				t.Fatalf("expected deleted app missing-app, got %q", id)
			}
			return domain.ErrAppNotFound
		},
	})

	request := httptest.NewRequest(http.MethodDelete, "/api/apps/missing-app", nil)
	recorder := httptest.NewRecorder()

	handler.APIAppRoutes(recorder, request)

	if recorder.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Result().StatusCode)
	}
}

func TestDashboardHandler_APIAppDelete_ManualCleanupRequired(t *testing.T) {
	handler := newTestHandler(&fakeDashboardUseCase{})
	handler.SetAppUseCase(&fakeAppUseCase{
		deleteFn: func(_ context.Context, id string) error {
			if id != "app-1" {
				t.Fatalf("expected deleted app app-1, got %q", id)
			}
			return domain.ErrManualCleanupRequired
		},
	})

	request := httptest.NewRequest(http.MethodDelete, "/api/apps/app-1", nil)
	recorder := httptest.NewRecorder()

	handler.APIAppRoutes(recorder, request)

	if recorder.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Result().StatusCode)
	}
}

func TestDashboardHandler_APIImport_Create(t *testing.T) {
	handler := newTestHandler(&fakeDashboardUseCase{})
	handler.SetAppUseCase(&fakeAppUseCase{
		importFn: func(_ context.Context, input domain.ImportRepoInput) (*domain.App, error) {
			if input.Name != "demo" {
				t.Fatalf("expected name demo, got %q", input.Name)
			}
			if input.RepoURL != "https://github.com/example/demo.git" {
				t.Fatalf("unexpected repo url: %q", input.RepoURL)
			}
			if input.Branch != "main" {
				t.Fatalf("unexpected branch: %q", input.Branch)
			}
			if input.ComposePath != "docker-compose.yml" {
				t.Fatalf("expected compose_path docker-compose.yml, got %q", input.ComposePath)
			}
			return &domain.App{
				ID:         "app-1",
				Name:       input.Name,
				ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
				Status:     domain.AppStatusCreated,
			}, nil
		},
	})

	body := bytes.NewBufferString(`{
		"name":"demo",
		"repo_url":"https://github.com/example/demo.git",
		"branch":"main",
		"compose_path":"docker-compose.yml"
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/apps/import", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.APIImport(recorder, request)

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

func TestDashboardHandler_APIImport_ErrorFromUseCase(t *testing.T) {
	handler := newTestHandler(&fakeDashboardUseCase{})
	handler.SetAppUseCase(&fakeAppUseCase{
		importFn: func(_ context.Context, _ domain.ImportRepoInput) (*domain.App, error) {
			return nil, domain.ErrMissingDockerfile
		},
	})

	body := bytes.NewBufferString(`{
		"name":"demo",
		"repo_url":"https://github.com/example/demo.git"
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/apps/import", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.APIImport(recorder, request)

	if recorder.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Result().StatusCode)
	}
}

func TestDashboardHandler_APIDeploy_Success(t *testing.T) {
	deployedID := ""
	handler := newTestHandler(&fakeDashboardUseCase{})
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

func TestDashboardHandler_APISettings_Get(t *testing.T) {
	handler := newTestHandler(&fakeDashboardUseCase{})
	handler.SetPlatformSettingsUseCase(&fakePlatformSettingsUseCase{
		getSettingsFn: func(context.Context) (*domain.PlatformSettings, error) {
			return &domain.PlatformSettings{
				AdminHost:            "127.0.0.1",
				AdminPort:            3001,
				AdminDomain:          "admin.example.com",
				AdminUseTLS:          true,
				CertbotEmail:         "ops@example.com",
				CertbotEnabled:       true,
				CertbotStaging:       true,
				CertbotAutoRenew:     false,
				CertbotTermsAccepted: true,
			}, nil
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	recorder := httptest.NewRecorder()
	handler.APISettings(recorder, request)

	if recorder.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Result().StatusCode)
	}

	var payload domain.PlatformSettings
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.AdminHost != "127.0.0.1" || payload.AdminPort != 3001 {
		t.Fatalf("unexpected platform settings payload: %+v", payload)
	}
}

func TestDashboardHandler_APISettings_Put(t *testing.T) {
	var saved domain.PlatformSettings
	handler := newTestHandler(&fakeDashboardUseCase{})
	handler.SetPlatformSettingsUseCase(&fakePlatformSettingsUseCase{
		updateSettingsFn: func(_ context.Context, input domain.PlatformSettings) (*domain.PlatformSettings, error) {
			saved = input
			return &input, nil
		},
	})

	body := bytes.NewBufferString(`{
		"admin_host": "0.0.0.0",
		"admin_port": 3200,
		"admin_domain": "admin.example.com",
		"admin_use_tls": true,
		"certbot_email": "ops@example.com",
		"certbot_enabled": true,
		"certbot_staging": false,
		"certbot_auto_renew": true,
		"certbot_terms_accepted": true
	}`)
	request := httptest.NewRequest(http.MethodPut, "/api/settings", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.APISettings(recorder, request)
	if recorder.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Result().StatusCode)
	}
	if saved.AdminPort != 3200 || !saved.AdminUseTLS || !saved.CertbotEnabled {
		t.Fatalf("unexpected saved settings: %+v", saved)
	}
}

func TestDashboardHandler_APIAppConfig_GetAndUpdate(t *testing.T) {
	handler := newTestHandler(&fakeDashboardUseCase{})
	handler.SetAppUseCase(&fakeAppUseCase{
		getFn: func(_ context.Context, id string) (*domain.App, error) {
			if id != "app-1" {
				t.Fatalf("expected app id app-1, got %q", id)
			}
			return &domain.App{
				ID:              "app-1",
				Name:            "demo",
				PublicDomain:    "app.example.com",
				ProxyTargetPort: 8080,
				UseTLS:          true,
				ManagedEnv: map[string]string{
					"FOO": "bar",
				},
			}, nil
		},
		updateConfigFn: func(_ context.Context, id string, config domain.AppConfig) (*domain.App, error) {
			if id != "app-1" {
				t.Fatalf("expected app id app-1, got %q", id)
			}
			return &domain.App{
				ID:              id,
				Name:            "demo",
				PublicDomain:    config.PublicDomain,
				ProxyTargetPort: config.ProxyTargetPort,
				UseTLS:          config.UseTLS,
				ManagedEnv:      config.ManagedEnv,
			}, nil
		},
	})

	getRequest := httptest.NewRequest(http.MethodGet, "/api/apps/app-1/config", nil)
	getRecorder := httptest.NewRecorder()
	handler.APIAppRoutes(getRecorder, getRequest)
	if getRecorder.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, getRecorder.Result().StatusCode)
	}
	var getPayload domain.AppConfig
	if err := json.NewDecoder(getRecorder.Body).Decode(&getPayload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if getPayload.PublicDomain != "app.example.com" || getPayload.ProxyTargetPort != 8080 {
		t.Fatalf("unexpected app config payload: %+v", getPayload)
	}

	putBody := bytes.NewBufferString(`{"public_domain":"landing.example.com","proxy_target_port":3000,"use_tls":false,"managed_env":{"API_URL":"https://landing"}}`)
	putRequest := httptest.NewRequest(http.MethodPut, "/api/apps/app-1/config", putBody)
	putRequest.Header.Set("Content-Type", "application/json")
	putRecorder := httptest.NewRecorder()
	handler.APIAppRoutes(putRecorder, putRequest)

	if putRecorder.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, putRecorder.Result().StatusCode)
	}
	var putPayload domain.AppConfig
	if err := json.NewDecoder(putRecorder.Body).Decode(&putPayload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if putPayload.PublicDomain != "landing.example.com" || putPayload.ProxyTargetPort != 3000 || putPayload.UseTLS != false {
		t.Fatalf("unexpected updated app config payload: %+v", putPayload)
	}
}

func TestDashboardHandler_APIAppConfig_Put_InvalidJSON(t *testing.T) {
	handler := newTestHandler(&fakeDashboardUseCase{})
	handler.SetAppUseCase(&fakeAppUseCase{})

	request := httptest.NewRequest(http.MethodPut, "/api/apps/app-1/config", strings.NewReader(`bad-json`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.APIAppRoutes(recorder, request)

	if recorder.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Result().StatusCode)
	}
}

func TestDashboardHandler_APIAppConfig_Get_NotFound(t *testing.T) {
	handler := newTestHandler(&fakeDashboardUseCase{})
	handler.SetAppUseCase(&fakeAppUseCase{
		getFn: func(_ context.Context, id string) (*domain.App, error) {
			if id != "missing" {
				t.Fatalf("expected app id missing, got %q", id)
			}
			return nil, domain.ErrAppNotFound
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/apps/missing/config", nil)
	recorder := httptest.NewRecorder()
	handler.APIAppRoutes(recorder, request)

	if recorder.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Result().StatusCode)
	}
}

func TestDashboardHandler_APIAppConfig_Put_ReturnsBadRequestOnDomainError(t *testing.T) {
	handler := newTestHandler(&fakeDashboardUseCase{})
	handler.SetAppUseCase(&fakeAppUseCase{
		updateConfigFn: func(_ context.Context, id string, config domain.AppConfig) (*domain.App, error) {
			if id != "app-1" {
				t.Fatalf("expected app id app-1, got %q", id)
			}
			if config.PublicDomain == "" {
				t.Fatalf("expected public domain to be provided")
			}
			return nil, domain.ErrInvalidDomain
		},
	})

	putBody := strings.NewReader(`{"public_domain":"bad_domain","proxy_target_port":8080}`)
	request := httptest.NewRequest(http.MethodPut, "/api/apps/app-1/config", putBody)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.APIAppRoutes(recorder, request)
	if recorder.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Result().StatusCode)
	}
}

func TestDashboardHandler_APIAppConfig_MethodNotAllowed(t *testing.T) {
	handler := newTestHandler(&fakeDashboardUseCase{})
	handler.SetAppUseCase(&fakeAppUseCase{})
	request := httptest.NewRequest(http.MethodPatch, "/api/apps/app-1/config", nil)
	recorder := httptest.NewRecorder()

	handler.APIAppRoutes(recorder, request)

	if recorder.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, recorder.Result().StatusCode)
	}
}
