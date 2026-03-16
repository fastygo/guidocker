package app

import (
	"context"
	"dashboard/domain"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeAppRepository struct {
	items map[string]*domain.App
}

func newFakeAppRepository() *fakeAppRepository {
	return &fakeAppRepository{items: map[string]*domain.App{}}
}

func (r *fakeAppRepository) Create(_ context.Context, app *domain.App) error {
	r.items[app.ID] = cloneApp(app)
	return nil
}

func (r *fakeAppRepository) Update(_ context.Context, app *domain.App) error {
	if _, ok := r.items[app.ID]; !ok {
		return domain.ErrAppNotFound
	}
	r.items[app.ID] = cloneApp(app)
	return nil
}

func (r *fakeAppRepository) Delete(_ context.Context, id string) error {
	if _, ok := r.items[id]; !ok {
		return domain.ErrAppNotFound
	}
	delete(r.items, id)
	return nil
}

func (r *fakeAppRepository) GetByID(_ context.Context, id string) (*domain.App, error) {
	app, ok := r.items[id]
	if !ok {
		return nil, domain.ErrAppNotFound
	}
	return cloneApp(app), nil
}

func (r *fakeAppRepository) List(_ context.Context) ([]*domain.App, error) {
	apps := make([]*domain.App, 0, len(r.items))
	for _, app := range r.items {
		apps = append(apps, cloneApp(app))
	}
	return apps, nil
}

type fakeDockerRepository struct {
	ensureNetworkFn func(context.Context) error
	deployFn    func(context.Context, *domain.App) error
	stopFn      func(context.Context, *domain.App) error
	restartFn   func(context.Context, *domain.App) error
	getStatusFn func(context.Context, *domain.App) (string, error)
	getLogsFn   func(context.Context, *domain.App, int) (string, error)
	destroyFn   func(context.Context, *domain.App) error
	listAllFn   func(context.Context) ([]domain.Container, error)
	inspectFn   func(context.Context, []string) ([]domain.ContainerDetail, error)
	resolveContainerIPFn func(context.Context, *domain.App) (string, error)
}

type fakeGitRepository struct {
	cloneFn func(context.Context, string, string, string) (string, error)
}

type fakeHostManager struct {
	removeRoutingFn  func(context.Context, *domain.App, domain.PlatformSettings) error
	applyRoutingFn   func(context.Context, *domain.App, domain.PlatformSettings) error
	validateRoutingF func(context.Context) error
	reloadRoutingF  func(context.Context) error
}

func (h *fakeHostManager) ApplyRouting(ctx context.Context, app *domain.App, settings domain.PlatformSettings) error {
	if h.applyRoutingFn != nil {
		return h.applyRoutingFn(ctx, app, settings)
	}
	return nil
}
func (h *fakeHostManager) RemoveRouting(ctx context.Context, app *domain.App, settings domain.PlatformSettings) error {
	if h.removeRoutingFn != nil {
		return h.removeRoutingFn(ctx, app, settings)
	}
	return nil
}
func (h *fakeHostManager) ValidateRouting(ctx context.Context) error {
	if h.validateRoutingF != nil {
		return h.validateRoutingF(ctx)
	}
	return nil
}
func (h *fakeHostManager) ReloadRouting(ctx context.Context) error {
	if h.reloadRoutingF != nil {
		return h.reloadRoutingF(ctx)
	}
	return nil
}

type fakeCertManager struct {
	ensureCertificateFn func(context.Context, domain.PlatformSettings, string) error
	removeCertificateFn func(context.Context, string) error
}

func (m *fakeCertManager) EnsureCertificate(ctx context.Context, settings domain.PlatformSettings, domainName string) error {
	if m.ensureCertificateFn != nil {
		return m.ensureCertificateFn(ctx, settings, domainName)
	}
	return nil
}
func (m *fakeCertManager) RemoveCertificate(ctx context.Context, domainName string) error {
	if m.removeCertificateFn != nil {
		return m.removeCertificateFn(ctx, domainName)
	}
	return nil
}

type fakePlatformSettingsUseCase struct {
	settings *domain.PlatformSettings
}

func (m *fakePlatformSettingsUseCase) GetPlatformSettings(context.Context) (*domain.PlatformSettings, error) {
	if m.settings == nil {
		return nil, nil
	}
	snapshot := *m.settings
	return &snapshot, nil
}
func (m *fakePlatformSettingsUseCase) UpdatePlatformSettings(context.Context, domain.PlatformSettings) (*domain.PlatformSettings, error) {
	return nil, nil
}

func (r *fakeDockerRepository) Deploy(ctx context.Context, app *domain.App) error {
	if r.deployFn != nil {
		return r.deployFn(ctx, app)
	}
	return nil
}

func (r *fakeDockerRepository) EnsureNetwork(ctx context.Context) error {
	if r.ensureNetworkFn != nil {
		return r.ensureNetworkFn(ctx)
	}
	return nil
}

func (r *fakeDockerRepository) Stop(ctx context.Context, app *domain.App) error {
	if r.stopFn != nil {
		return r.stopFn(ctx, app)
	}
	return nil
}

func (r *fakeDockerRepository) Restart(ctx context.Context, app *domain.App) error {
	if r.restartFn != nil {
		return r.restartFn(ctx, app)
	}
	return nil
}

func (r *fakeDockerRepository) GetStatus(ctx context.Context, app *domain.App) (string, error) {
	if r.getStatusFn != nil {
		return r.getStatusFn(ctx, app)
	}
	return domain.AppStatusRunning, nil
}

func (r *fakeDockerRepository) GetLogs(ctx context.Context, app *domain.App, lines int) (string, error) {
	if r.getLogsFn != nil {
		return r.getLogsFn(ctx, app, lines)
	}
	return "", nil
}

func (r *fakeDockerRepository) ListRunning(context.Context) ([]domain.Container, error) {
	return nil, nil
}

func (r *fakeDockerRepository) Destroy(ctx context.Context, app *domain.App) error {
	if r.destroyFn != nil {
		return r.destroyFn(ctx, app)
	}

	return nil
}

func (r *fakeDockerRepository) ListAllContainers(ctx context.Context) ([]domain.Container, error) {
	if r.listAllFn != nil {
		return r.listAllFn(ctx)
	}
	return []domain.Container{}, nil
}

func (r *fakeDockerRepository) InspectContainers(ctx context.Context, ids []string) ([]domain.ContainerDetail, error) {
	if r.inspectFn != nil {
		return r.inspectFn(ctx, ids)
	}
	return []domain.ContainerDetail{}, nil
}

func (r *fakeDockerRepository) ResolveContainerIP(ctx context.Context, app *domain.App) (string, error) {
	if r.resolveContainerIPFn != nil {
		return r.resolveContainerIPFn(ctx, app)
	}
	return "", domain.ErrContainerNotFound
}

func (r *fakeGitRepository) Clone(ctx context.Context, sourceURL, branch, destination string) (string, error) {
	if r.cloneFn == nil {
		return "", domain.ErrInvalidRepoURL
	}
	return r.cloneFn(ctx, sourceURL, branch, destination)
}

func TestService_DeleteApp_FullCleanup(t *testing.T) {
	repo := newFakeAppRepository()
	baseDir := t.TempDir()
	stackDir := filepath.Join(baseDir, "app-1")
	if err := os.MkdirAll(stackDir, 0o755); err != nil {
		t.Fatalf("failed to create stack dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stackDir, "docker-compose.yml"), []byte(`services:`), 0o644); err != nil {
		t.Fatalf("failed to create compose file: %v", err)
	}

	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Demo",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         stackDir,
		Status:      domain.AppStatusCreated,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	destroyed := false
	service := NewAppService(repo, &fakeDockerRepository{
		destroyFn: func(_ context.Context, app *domain.App) error {
			destroyed = app.ID == "app-1"
			return nil
		},
	}, nil, "/opt/stacks")

	if err := service.DeleteApp(context.Background(), "app-1"); err != nil {
		t.Fatalf("DeleteApp() error = %v", err)
	}

	if !destroyed {
		t.Fatalf("expected destroy to be called")
	}
	if _, err := os.Stat(stackDir); !os.IsNotExist(err) {
		t.Fatalf("expected stack dir removed, stat error: %v", err)
	}
	if _, ok := repo.items["app-1"]; ok {
		t.Fatalf("expected app removed from repository")
	}
}

func TestService_DeleteApp_DockerErrorReturnsError(t *testing.T) {
	repo := newFakeAppRepository()
	stackDir := filepath.Join(t.TempDir(), "app-1")
	if err := os.MkdirAll(stackDir, 0o755); err != nil {
		t.Fatalf("failed to create stack dir: %v", err)
	}
	managedEnvPath := filepath.Join(stackDir, ".platform.env")
	if err := os.WriteFile(managedEnvPath, []byte("FOO=bar\n"), 0o644); err != nil {
		t.Fatalf("failed to create managed env file: %v", err)
	}
	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Demo",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         stackDir,
		PublicDomain: "demo.example.com",
		UseTLS:      false,
		Status:      domain.AppStatusCreated,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	destroyed := false
	service := NewAppService(repo, &fakeDockerRepository{
		destroyFn: func(_ context.Context, app *domain.App) error {
			destroyed = app.ID == "app-1"
			return errors.New("docker unavailable")
		},
	}, nil, "/opt/stacks")

	if err := service.DeleteApp(context.Background(), "app-1"); err == nil {
		t.Fatalf("DeleteApp() expected error")
	}

	if !destroyed {
		t.Fatalf("expected destroy to be called")
	}
	if _, ok := repo.items["app-1"]; !ok {
		t.Fatalf("expected app not removed from repository on cleanup error")
	}
	if _, err := os.Stat(stackDir); !os.IsNotExist(err) {
		t.Fatalf("expected stack dir removed, stat error: %v", err)
	}
	if _, err := os.Stat(managedEnvPath); !os.IsNotExist(err) {
		t.Fatalf("expected managed env file removed, stat error: %v", err)
	}
}

func TestService_DeleteApp_RemovesRoutingEnvAndCertBeforeRecordDeletion(t *testing.T) {
	repo := newFakeAppRepository()
	baseDir := t.TempDir()
	stackDir := filepath.Join(baseDir, "app-1")
	if err := os.MkdirAll(stackDir, 0o755); err != nil {
		t.Fatalf("failed to create stack dir: %v", err)
	}
	managedEnvPath := filepath.Join(stackDir, ".platform.env")
	if err := os.WriteFile(managedEnvPath, []byte("FOO=bar\n"), 0o644); err != nil {
		t.Fatalf("failed to create managed env file: %v", err)
	}

	repo.items["app-1"] = &domain.App{
		ID:              "app-1",
		Name:            "Demo",
		ComposeYAML:     "services:\n  web:\n    image: nginx:alpine",
		Dir:             stackDir,
		PublicDomain:    "demo.example.com",
		ProxyTargetPort: 8080,
		UseTLS:          true,
		Status:          domain.AppStatusCreated,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	callOrder := []string{}
	hostManager := &fakeHostManager{
		removeRoutingFn: func(_ context.Context, app *domain.App, _ domain.PlatformSettings) error {
			if app.ID != "app-1" {
				t.Fatalf("unexpected app id: %s", app.ID)
			}
			callOrder = append(callOrder, "remove-routing")
			if _, err := os.Stat(managedEnvPath); err != nil {
				t.Fatalf("managed env file should still exist while routing is removed: %v", err)
			}
			return nil
		},
	}
	certRemoved := false
	certManager := &fakeCertManager{
		removeCertificateFn: func(_ context.Context, domainName string) error {
			if domainName != "demo.example.com" {
				t.Fatalf("unexpected domain for cert removal: %q", domainName)
			}
			callOrder = append(callOrder, "remove-certificate")
			certRemoved = true
			if _, err := os.Stat(managedEnvPath); err != nil {
				t.Fatalf("managed env file should still exist while cert is removed: %v", err)
			}
			return nil
		},
	}

	service := NewAppService(repo, &fakeDockerRepository{}, nil, baseDir).
		WithHostManagers(hostManager, certManager)

	if err := service.DeleteApp(context.Background(), "app-1"); err != nil {
		t.Fatalf("DeleteApp() error = %v", err)
	}

	if !certRemoved {
		t.Fatalf("expected certificate removal to be called")
	}
	if len(callOrder) < 2 {
		t.Fatalf("expected remove-routing and remove-certificate calls, got: %#v", callOrder)
	}
	if callOrder[0] != "remove-routing" || callOrder[1] != "remove-certificate" {
		t.Fatalf("expected routing removal before certificate removal, got: %#v", callOrder)
	}
	if _, err := os.Stat(managedEnvPath); !os.IsNotExist(err) {
		t.Fatalf("expected managed env file removed, stat error: %v", err)
	}
	if _, ok := repo.items["app-1"]; ok {
		t.Fatalf("expected app removed from repository")
	}
	if _, err := os.Stat(stackDir); !os.IsNotExist(err) {
		t.Fatalf("expected stack dir removed, stat error: %v", err)
	}
}

func TestService_UpdateAppConfig_AllowsInternalProxyPortMatchingAdminPort(t *testing.T) {
	repo := newFakeAppRepository()
	baseDir := t.TempDir()
	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Demo",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         filepath.Join(baseDir, "app-1"),
		Status:      domain.AppStatusCreated,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	service := NewAppService(repo, &fakeDockerRepository{}, nil, baseDir).
		WithPlatformSettingsUseCase(&fakePlatformSettingsUseCase{
			settings: &domain.PlatformSettings{
				AdminPort: 3000,
			},
		})

	_, err := service.UpdateAppConfig(context.Background(), "app-1", domain.AppConfig{
		PublicDomain:    "demo.example.com",
		ProxyTargetPort: 3000,
	})
	if err != nil {
		t.Fatalf("expected internal proxy target port to be allowed, got %v", err)
	}
}

func TestService_UpdateAppConfig_ValidatesDomainAndPort(t *testing.T) {
	repo := newFakeAppRepository()
	baseDir := t.TempDir()
	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Demo",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         filepath.Join(baseDir, "app-1"),
		Status:      domain.AppStatusCreated,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	service := NewAppService(repo, &fakeDockerRepository{}, nil, baseDir)

	_, err := service.UpdateAppConfig(context.Background(), "app-1", domain.AppConfig{
		PublicDomain: "bad_domain",
	})
	if !errors.Is(err, domain.ErrInvalidDomain) {
		t.Fatalf("expected ErrInvalidDomain, got %v", err)
	}
	_, err = service.UpdateAppConfig(context.Background(), "app-1", domain.AppConfig{
		PublicDomain: "example.com",
	})
	if !errors.Is(err, domain.ErrInvalidProxyPort) {
		t.Fatalf("expected ErrInvalidProxyPort, got %v", err)
	}
}

func TestService_UpdateAppConfig_DetectsDomainConflict(t *testing.T) {
	repo := newFakeAppRepository()
	baseDir := t.TempDir()
	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Demo",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         filepath.Join(baseDir, "app-1"),
		Status:      domain.AppStatusCreated,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	repo.items["app-2"] = &domain.App{
		ID:              "app-2",
		Name:            "Another",
		ComposeYAML:     "services:\n  web:\n    image: nginx:alpine",
		Dir:             filepath.Join(baseDir, "app-2"),
		Status:          domain.AppStatusCreated,
		PublicDomain:    "demo.example.com",
		ProxyTargetPort: 8080,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	service := NewAppService(repo, &fakeDockerRepository{}, nil, baseDir)
	_, err := service.UpdateAppConfig(context.Background(), "app-1", domain.AppConfig{
		PublicDomain:    "Demo.Example.Com",
		ProxyTargetPort: 3000,
	})
	if !errors.Is(err, domain.ErrDomainConflict) {
		t.Fatalf("expected ErrDomainConflict, got %v", err)
	}
}

func TestService_UpdateAppConfig_DoesNotCallCertManagerOnHTTPOnly(t *testing.T) {
	repo := newFakeAppRepository()
	baseDir := t.TempDir()
	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Demo",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         filepath.Join(baseDir, "app-1"),
		Status:      domain.AppStatusCreated,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	certCalled := 0
	service := NewAppService(repo, &fakeDockerRepository{}, nil, baseDir).
		WithHostManagers(nil, &fakeCertManager{
			ensureCertificateFn: func(_ context.Context, _ domain.PlatformSettings, _ string) error {
				certCalled++
				return nil
			},
		})

	if _, err := service.UpdateAppConfig(context.Background(), "app-1", domain.AppConfig{
		PublicDomain:    "app.example.com",
		ProxyTargetPort: 8080,
		UseTLS:          false,
	}); err != nil {
		t.Fatalf("UpdateAppConfig() error = %v", err)
	}
	if certCalled != 0 {
		t.Fatalf("expected cert manager to be skipped on HTTP-only config, got calls %d", certCalled)
	}
}

func TestService_UpdateAppConfig_TriggersCertManagerWhenHTTPSRequested(t *testing.T) {
	repo := newFakeAppRepository()
	baseDir := t.TempDir()
	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Demo",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         filepath.Join(baseDir, "app-1"),
		Status:      domain.AppStatusCreated,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	certCalled := 0
	service := NewAppService(repo, &fakeDockerRepository{
		resolveContainerIPFn: func(_ context.Context, app *domain.App) (string, error) {
			if app == nil || app.ID != "app-1" {
				return "", domain.ErrContainerNotFound
			}
			return "10.44.0.8", nil
		},
	}, nil, baseDir).
		WithPlatformSettingsUseCase(&fakePlatformSettingsUseCase{
			settings: &domain.PlatformSettings{
				CertbotEnabled:       true,
				CertbotEmail:         "ops@example.com",
				CertbotTermsAccepted: true,
			},
		}).
		WithHostManagers(nil, &fakeCertManager{
			ensureCertificateFn: func(_ context.Context, _ domain.PlatformSettings, _ string) error {
				certCalled++
				return nil
			},
		})

	if _, err := service.UpdateAppConfig(context.Background(), "app-1", domain.AppConfig{
		PublicDomain:    "app.example.com",
		ProxyTargetPort: 8080,
		UseTLS:          true,
	}); err != nil {
		t.Fatalf("UpdateAppConfig() error = %v", err)
	}
	if certCalled != 1 {
		t.Fatalf("expected cert manager to be called once, got %d", certCalled)
	}
}

func TestService_UpdateAppConfig_RejectsTLSWithoutCertbotSettings(t *testing.T) {
	repo := newFakeAppRepository()
	baseDir := t.TempDir()
	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Demo",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         filepath.Join(baseDir, "app-1"),
		Status:      domain.AppStatusCreated,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	service := NewAppService(repo, &fakeDockerRepository{}, nil, baseDir).
		WithPlatformSettingsUseCase(&fakePlatformSettingsUseCase{
			settings: &domain.PlatformSettings{},
		})

	_, err := service.UpdateAppConfig(context.Background(), "app-1", domain.AppConfig{
		PublicDomain:    "app.example.com",
		ProxyTargetPort: 8080,
		UseTLS:          true,
	})
	if !errors.Is(err, domain.ErrTLSRequiresCertbot) {
		t.Fatalf("expected ErrTLSRequiresCertbot, got %v", err)
	}
}

func TestService_UpdateAppConfig_RejectsTLSWithoutCertbotEmail(t *testing.T) {
	repo := newFakeAppRepository()
	baseDir := t.TempDir()
	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Demo",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         filepath.Join(baseDir, "app-1"),
		Status:      domain.AppStatusCreated,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	service := NewAppService(repo, &fakeDockerRepository{}, nil, baseDir).
		WithPlatformSettingsUseCase(&fakePlatformSettingsUseCase{
			settings: &domain.PlatformSettings{
				CertbotEnabled:       true,
				CertbotTermsAccepted: true,
			},
		})

	_, err := service.UpdateAppConfig(context.Background(), "app-1", domain.AppConfig{
		PublicDomain:    "app.example.com",
		ProxyTargetPort: 8080,
		UseTLS:          true,
	})
	if !errors.Is(err, domain.ErrTLSEmailRequired) {
		t.Fatalf("expected ErrTLSEmailRequired, got %v", err)
	}
}

func TestService_UpdateAppConfig_RejectsTLSWithoutTermsAcceptance(t *testing.T) {
	repo := newFakeAppRepository()
	baseDir := t.TempDir()
	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Demo",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         filepath.Join(baseDir, "app-1"),
		Status:      domain.AppStatusCreated,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	service := NewAppService(repo, &fakeDockerRepository{}, nil, baseDir).
		WithPlatformSettingsUseCase(&fakePlatformSettingsUseCase{
			settings: &domain.PlatformSettings{
				CertbotEnabled: true,
				CertbotEmail:   "ops@example.com",
			},
		})

	_, err := service.UpdateAppConfig(context.Background(), "app-1", domain.AppConfig{
		PublicDomain:    "app.example.com",
		ProxyTargetPort: 8080,
		UseTLS:          true,
	})
	if !errors.Is(err, domain.ErrTLSAgreementRequired) {
		t.Fatalf("expected ErrTLSAgreementRequired, got %v", err)
	}
}

func TestService_UpdateAppConfig_AllowsTLSWhenPrerequisitesMet(t *testing.T) {
	repo := newFakeAppRepository()
	baseDir := t.TempDir()
	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Demo",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         filepath.Join(baseDir, "app-1"),
		Status:      domain.AppStatusCreated,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	service := NewAppService(repo, &fakeDockerRepository{}, nil, baseDir).
		WithPlatformSettingsUseCase(&fakePlatformSettingsUseCase{
			settings: &domain.PlatformSettings{
				CertbotEnabled:       true,
				CertbotEmail:         "ops@example.com",
				CertbotTermsAccepted: true,
			},
		})

	_, err := service.UpdateAppConfig(context.Background(), "app-1", domain.AppConfig{
		PublicDomain:    "app.example.com",
		ProxyTargetPort: 8080,
		UseTLS:          true,
	})
	if err != nil {
		t.Fatalf("UpdateAppConfig() error = %v", err)
	}
}

func TestService_DeleteApp_AggregatesCleanupErrorsButCleansManagedArtifacts(t *testing.T) {
	repo := newFakeAppRepository()
	baseDir := t.TempDir()
	stackDir := filepath.Join(baseDir, "app-1")
	if err := os.MkdirAll(stackDir, 0o755); err != nil {
		t.Fatalf("failed to create stack dir: %v", err)
	}
	managedEnvPath := filepath.Join(stackDir, ".platform.env")
	if err := os.WriteFile(managedEnvPath, []byte("FOO=bar\n"), 0o644); err != nil {
		t.Fatalf("failed to create managed env file: %v", err)
	}
	repo.items["app-1"] = &domain.App{
		ID:              "app-1",
		Name:            "Demo",
		ComposeYAML:     "services:\n  web:\n    image: nginx:alpine",
		Dir:             stackDir,
		PublicDomain:    "demo.example.com",
		ProxyTargetPort: 8080,
		UseTLS:          true,
		Status:          domain.AppStatusCreated,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	callOrder := []string{}
	hostManager := &fakeHostManager{
		removeRoutingFn: func(_ context.Context, _ *domain.App, _ domain.PlatformSettings) error {
			callOrder = append(callOrder, "remove-routing")
			return errors.New("routing failed")
		},
	}
	certManager := &fakeCertManager{
		removeCertificateFn: func(_ context.Context, _ string) error {
			callOrder = append(callOrder, "remove-certificate")
			return errors.New("certificate failed")
		},
	}

	service := NewAppService(repo, &fakeDockerRepository{}, nil, baseDir).
		WithHostManagers(hostManager, certManager)

	err := service.DeleteApp(context.Background(), "app-1")
	if err == nil {
		t.Fatalf("expected cleanup error")
	}
	if !strings.Contains(err.Error(), "2 cleanup errors") {
		t.Fatalf("expected aggregated cleanup error, got %q", err.Error())
	}
	if len(callOrder) != 2 || callOrder[0] != "remove-routing" || callOrder[1] != "remove-certificate" {
		t.Fatalf("expected routing and certificate removal attempts in order, got %#v", callOrder)
	}
	if _, ok := repo.items["app-1"]; !ok {
		t.Fatalf("expected app not removed from repository on cleanup error")
	}
	if _, err := os.Stat(stackDir); !os.IsNotExist(err) {
		t.Fatalf("expected stack dir removed, stat error: %v", err)
	}
	if _, err := os.Stat(managedEnvPath); !os.IsNotExist(err) {
		t.Fatalf("expected managed env file removed, stat error: %v", err)
	}
}

func TestService_DeleteApp_BlockingExternalContainerRequiresManualCleanup(t *testing.T) {
	repo := newFakeAppRepository()
	baseDir := t.TempDir()
	stackDir := filepath.Join(baseDir, "app-1")
	if err := os.MkdirAll(stackDir, 0o755); err != nil {
		t.Fatalf("failed to create stack dir: %v", err)
	}

	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Demo",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         stackDir,
		PublicDomain: "demo.example.com",
		Status:      domain.AppStatusRunning,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	destroyed := false
	hostCalled := false
	certRemoved := false

	service := NewAppService(repo, &fakeDockerRepository{
		listAllFn: func(_ context.Context) ([]domain.Container, error) {
			return []domain.Container{
				{
					ID:   "container-1",
					Name: "legacy-app-1worker",
				},
			}, nil
		},
		inspectFn: func(_ context.Context, _ []string) ([]domain.ContainerDetail, error) {
			return []domain.ContainerDetail{
				{
					ID:     "container-1",
					Name:   "/legacy-app-1worker",
					Labels: map[string]string{},
				},
			}, nil
		},
		destroyFn: func(_ context.Context, app *domain.App) error {
			destroyed = true
			return nil
		},
	}, nil, baseDir)
	service.WithHostManagers(&fakeHostManager{
		removeRoutingFn: func(_ context.Context, _ *domain.App, _ domain.PlatformSettings) error {
			hostCalled = true
			return nil
		},
	}, &fakeCertManager{
		removeCertificateFn: func(_ context.Context, _ string) error {
			certRemoved = true
			return nil
		},
	})

	err := service.DeleteApp(context.Background(), "app-1")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, domain.ErrManualCleanupRequired) {
		t.Fatalf("expected ErrManualCleanupRequired, got %v", err)
	}
	if destroyed {
		t.Fatalf("expected runtime destroy to be skipped")
	}
	if hostCalled {
		t.Fatalf("expected routing cleanup to be skipped on blocked delete preflight")
	}
	if certRemoved {
		t.Fatalf("expected certificate removal to be skipped on blocked delete preflight")
	}
	if _, ok := repo.items["app-1"]; !ok {
		t.Fatalf("expected app still present in repository")
	}
}

func TestService_DeleteApp_DomainOwnedByAdminSkipsCertificateRemoval(t *testing.T) {
	repo := newFakeAppRepository()
	baseDir := t.TempDir()
	stackDir := filepath.Join(baseDir, "app-1")
	if err := os.MkdirAll(stackDir, 0o755); err != nil {
		t.Fatalf("failed to create stack dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stackDir, "docker-compose.yml"), []byte(`services:`), 0o644); err != nil {
		t.Fatalf("failed to create compose file: %v", err)
	}

	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Dashboard",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         stackDir,
		PublicDomain: "admin.example.com",
		UseTLS:      true,
		Status:      domain.AppStatusRunning,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	routingCalls := 0
	certRemoved := false

	service := NewAppService(repo, &fakeDockerRepository{}, nil, baseDir).
		WithPlatformSettingsUseCase(&fakePlatformSettingsUseCase{
			settings: &domain.PlatformSettings{
				AdminDomain: "admin.example.com",
			},
		}).
		WithHostManagers(&fakeHostManager{
		removeRoutingFn: func(_ context.Context, app *domain.App, _ domain.PlatformSettings) error {
			routingCalls++
			if app == nil || app.PublicDomain != "admin.example.com" {
				t.Fatalf("unexpected app for routing cleanup: %#v", app)
			}
			return nil
		},
	}, &fakeCertManager{
		removeCertificateFn: func(_ context.Context, _ string) error {
			certRemoved = true
			return nil
		},
	})

	if err := service.DeleteApp(context.Background(), "app-1"); err != nil {
		t.Fatalf("DeleteApp() error = %v", err)
	}

	if routingCalls != 1 {
		t.Fatalf("expected one routing cleanup call, got %d", routingCalls)
	}
	if certRemoved {
		t.Fatalf("expected certificate removal to be skipped for admin domain")
	}
	if _, ok := repo.items["app-1"]; ok {
		t.Fatalf("expected app removed from repository")
	}
	if _, err := os.Stat(stackDir); !os.IsNotExist(err) {
		t.Fatalf("expected stack dir removed, stat error: %v", err)
	}
}

func TestService_DeleteApp_ExternalComposeResourcesRequireManualCleanup(t *testing.T) {
	repo := newFakeAppRepository()
	baseDir := t.TempDir()
	stackDir := filepath.Join(baseDir, "app-1")
	if err := os.MkdirAll(stackDir, 0o755); err != nil {
		t.Fatalf("failed to create stack dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stackDir, "docker-compose.yml"), []byte(`services:
  web:
    image: postgres:15
    volumes:
      - external-volume:/var/lib/postgresql/data
      - type: volume
        source: old-shared
        target: /data
        external: true`), 0o644); err != nil {
		t.Fatalf("failed to create compose file: %v", err)
	}

	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Stateful",
		ComposeYAML: `services:
  web:
    image: postgres:15
    volumes:
      - external-volume:/var/lib/postgresql/data
      - type: volume
        source: old-shared
        target: /data
        external: true`,
		Dir:         stackDir,
		Status:      domain.AppStatusCreated,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	service := NewAppService(repo, &fakeDockerRepository{}, nil, baseDir)
	if err := service.DeleteApp(context.Background(), "app-1"); err == nil {
		t.Fatalf("expected error")
	} else if !errors.Is(err, domain.ErrManualCleanupRequired) {
		t.Fatalf("expected ErrManualCleanupRequired, got %v", err)
	}
}

func TestService_CreateApp(t *testing.T) {
	repo := newFakeAppRepository()
	baseDir := t.TempDir()
	service := NewAppService(repo, &fakeDockerRepository{}, nil, baseDir)

	app, err := service.CreateApp(context.Background(), "Demo App", `services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"`)
	if err != nil {
		t.Fatalf("CreateApp() error = %v", err)
	}

	if app.ID == "" {
		t.Fatal("expected generated app ID")
	}
	expectedDir := filepath.Join(baseDir, app.ID)
	if app.Dir != expectedDir {
		t.Fatalf("unexpected app dir %q", app.Dir)
	}
	if app.Status != domain.AppStatusCreated {
		t.Fatalf("expected created status, got %q", app.Status)
	}
	if len(app.Ports) != 1 || app.Ports[0] != "8080:80" {
		t.Fatalf("unexpected ports: %+v", app.Ports)
	}
}

func TestService_CreateApp_MissingServices(t *testing.T) {
	repo := newFakeAppRepository()
	service := NewAppService(repo, &fakeDockerRepository{}, nil, t.TempDir())

	_, err := service.CreateApp(context.Background(), "Demo App", `web:
  image: nginx:alpine`)
	if !errors.Is(err, domain.ErrComposeNoServices) {
		t.Fatalf("expected ErrComposeNoServices, got %v", err)
	}
}

func TestService_CreateApp_RejectsReservedIngressPort(t *testing.T) {
	repo := newFakeAppRepository()
	service := NewAppService(repo, &fakeDockerRepository{}, nil, t.TempDir())

	_, err := service.CreateApp(context.Background(), "Demo App", `services:
  web:
    image: nginx:alpine
    ports:
      - "80:80"`)
	if !errors.Is(err, domain.ErrReservedIngressPort) {
		t.Fatalf("expected ErrReservedIngressPort, got %v", err)
	}
}

func TestService_UpdateApp_RejectsReservedIngressPort(t *testing.T) {
	repo := newFakeAppRepository()
	baseDir := t.TempDir()
	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Demo",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         filepath.Join(baseDir, "app-1"),
		Status:      domain.AppStatusCreated,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	service := NewAppService(repo, &fakeDockerRepository{}, nil, baseDir)

	_, err := service.UpdateApp(context.Background(), "app-1", "Demo", `services:
  web:
    image: nginx:alpine
    ports:
      - "443:80"`)
	if !errors.Is(err, domain.ErrReservedIngressPort) {
		t.Fatalf("expected ErrReservedIngressPort, got %v", err)
	}
}

func TestService_ImportRepo_ComposeMode(t *testing.T) {
	baseDir := t.TempDir()
	repo := newFakeAppRepository()
	gitRepo := &fakeGitRepository{
		cloneFn: func(_ context.Context, sourceURL, _, destination string) (string, error) {
			if sourceURL != "https://github.com/example/repo.git" {
				t.Fatalf("unexpected repo URL %q", sourceURL)
			}
			if err := os.MkdirAll(filepath.Join(destination, "sub"), 0o755); err != nil {
				t.Fatalf("failed create nested dir: %v", err)
			}
			if err := os.WriteFile(filepath.Join(destination, "sub", "compose.yaml"), []byte("services:\n  web:\n    image: nginx:alpine"), 0o644); err != nil {
				t.Fatalf("failed create compose: %v", err)
			}
			return "commit-abc", nil
		},
	}

	service := NewAppService(repo, &fakeDockerRepository{}, gitRepo, baseDir).
		WithImportTempPath(".tmp").
		WithComposeValidator(func(context.Context, string) error { return nil })

	app, err := service.ImportRepo(context.Background(), domain.ImportRepoInput{
		Name:        "Imported App",
		RepoURL:     "https://github.com/example/repo.git",
		ComposePath: "sub/compose.yaml",
	})
	if err != nil {
		t.Fatalf("ImportRepo() error = %v", err)
	}
	if app.SourceType != domain.SourceTypeRepoCompose {
		t.Fatalf("expected source type %q, got %q", domain.SourceTypeRepoCompose, app.SourceType)
	}
	if app.ComposePath != filepath.Join("sub", "compose.yaml") {
		t.Fatalf("expected compose path sub/compose.yaml, got %q", app.ComposePath)
	}
	if app.ResolvedCommit != "commit-abc" {
		t.Fatalf("expected resolved commit commit-abc, got %q", app.ResolvedCommit)
	}
	if _, err := repo.GetByID(context.Background(), app.ID); err != nil {
		t.Fatalf("app not persisted: %v", err)
	}
	if _, err := os.Stat(app.Dir); err != nil {
		t.Fatalf("expected app dir exists: %v", err)
	}
	if !filepath.IsAbs(app.Dir) {
		t.Fatalf("expected app dir to be absolute, got %q", app.Dir)
	}
}

func TestService_ImportRepo_DockerfileFallbackWithPort(t *testing.T) {
	baseDir := t.TempDir()
	repo := newFakeAppRepository()
	gitRepo := &fakeGitRepository{
		cloneFn: func(_ context.Context, _, _, destination string) (string, error) {
			if err := os.WriteFile(filepath.Join(destination, "Dockerfile"), []byte("FROM nginx:alpine"), 0o644); err != nil {
				t.Fatalf("failed create dockerfile: %v", err)
			}
			return "commit-def", nil
		},
	}

	service := NewAppService(repo, &fakeDockerRepository{}, gitRepo, baseDir).
		WithComposeValidator(func(context.Context, string) error { return nil })

	app, err := service.ImportRepo(context.Background(), domain.ImportRepoInput{
		Name:     "Static App",
		RepoURL:  "https://github.com/example/static.git",
		AppPort:  8080,
	})
	if err != nil {
		t.Fatalf("ImportRepo() error = %v", err)
	}
	if app.SourceType != domain.SourceTypeRepoDockerfile {
		t.Fatalf("expected source type %q, got %q", domain.SourceTypeRepoDockerfile, app.SourceType)
	}
	if app.ComposePath != "docker-compose.generated.yml" {
		t.Fatalf("expected generated compose path, got %q", app.ComposePath)
	}
	if !strings.Contains(app.ComposeYAML, "expose:\n      - \"8080\"") {
		t.Fatalf("expected generated compose with required exposed port, got %q", app.ComposeYAML)
	}
}

func TestService_ImportRepo_DockerfileFallbackRejectsReservedPort(t *testing.T) {
	baseDir := t.TempDir()
	repo := newFakeAppRepository()
	gitRepo := &fakeGitRepository{
		cloneFn: func(_ context.Context, _, _, destination string) (string, error) {
			if err := os.WriteFile(filepath.Join(destination, "Dockerfile"), []byte("FROM nginx:alpine"), 0o644); err != nil {
				t.Fatalf("failed create dockerfile: %v", err)
			}
			return "commit-ijk", nil
		},
	}

	service := NewAppService(repo, &fakeDockerRepository{}, gitRepo, baseDir).
		WithComposeValidator(func(context.Context, string) error { return nil })

	_, err := service.ImportRepo(context.Background(), domain.ImportRepoInput{
		Name:    "Static App",
		RepoURL: "https://github.com/example/static.git",
		AppPort: 80,
	})
	if !errors.Is(err, domain.ErrReservedIngressPort) {
		t.Fatalf("expected ErrReservedIngressPort, got %v", err)
	}
}

func TestService_ImportRepo_ComposeModeRejectsReservedPorts(t *testing.T) {
	baseDir := t.TempDir()
	repo := newFakeAppRepository()
	gitRepo := &fakeGitRepository{
		cloneFn: func(_ context.Context, sourceURL, _, destination string) (string, error) {
			if sourceURL != "https://github.com/example/repo.git" {
				t.Fatalf("unexpected repo URL %q", sourceURL)
			}
			if err := os.WriteFile(filepath.Join(destination, "compose.yml"), []byte(`services:
  web:
    image: nginx:alpine
    ports:
      - "443:443"`), 0o644); err != nil {
				t.Fatalf("failed create compose file: %v", err)
			}
			return "commit-xyz", nil
		},
	}

	service := NewAppService(repo, &fakeDockerRepository{}, gitRepo, baseDir).
		WithImportTempPath(".tmp").
		WithComposeValidator(func(context.Context, string) error { return nil })

	_, err := service.ImportRepo(context.Background(), domain.ImportRepoInput{
		Name:        "Imported App",
		RepoURL:     "https://github.com/example/repo.git",
		ComposePath: "compose.yml",
	})
	if !errors.Is(err, domain.ErrReservedIngressPort) {
		t.Fatalf("expected ErrReservedIngressPort, got %v", err)
	}
}

func TestService_ImportRepo_MissingDockerfileReturnsError(t *testing.T) {
	baseDir := t.TempDir()
	repo := newFakeAppRepository()
	gitRepo := &fakeGitRepository{
		cloneFn: func(_ context.Context, _, _, _ string) (string, error) {
			return "commit-ghi", nil
		},
	}
	service := NewAppService(repo, &fakeDockerRepository{}, gitRepo, baseDir).
		WithComposeValidator(func(context.Context, string) error { return nil })

	_, err := service.ImportRepo(context.Background(), domain.ImportRepoInput{
		Name:    "Broken App",
		RepoURL: "https://github.com/example/broken.git",
	})
	if !errors.Is(err, domain.ErrMissingDockerfile) {
		t.Fatalf("expected ErrMissingDockerfile, got %v", err)
	}
	if len(repo.items) != 0 {
		t.Fatalf("expected no apps persisted on import error")
	}

	tempEntries, readErr := os.ReadDir(filepath.Join(baseDir, ".tmp"))
	if readErr != nil {
		if !os.IsNotExist(readErr) {
			t.Fatalf("unexpected error reading temp dir: %v", readErr)
		}
	}
	if readErr == nil && len(tempEntries) != 0 {
		t.Fatalf("expected empty import temp directory, got %d entries", len(tempEntries))
	}
}

func TestService_ImportRepo_InvalidComposeValidation(t *testing.T) {
	baseDir := t.TempDir()
	repo := newFakeAppRepository()
	gitRepo := &fakeGitRepository{
		cloneFn: func(_ context.Context, _, _, destination string) (string, error) {
			if err := os.WriteFile(filepath.Join(destination, "docker-compose.yml"), []byte("services:"), 0o644); err != nil {
				t.Fatalf("failed to write compose file: %v", err)
			}
			return "commit-jkl", nil
		},
	}
	service := NewAppService(repo, &fakeDockerRepository{}, gitRepo, baseDir).
		WithComposeValidator(func(context.Context, string) error {
			return fmt.Errorf("%w: invalid compose file", domain.ErrComposeConfigValidation)
		})

	_, err := service.ImportRepo(context.Background(), domain.ImportRepoInput{
		Name:        "Invalid Compose App",
		RepoURL:     "https://github.com/example/invalid.git",
		ComposePath: "docker-compose.yml",
	})
	if !errors.Is(err, domain.ErrComposeConfigValidation) {
		t.Fatalf("expected compose config validation error, got %v", err)
	}
	if len(repo.items) != 0 {
		t.Fatalf("expected no apps persisted on compose validation error")
	}
}

func TestService_DeployApp_Success(t *testing.T) {
	repo := newFakeAppRepository()
	now := time.Now().UTC()
	stackBase := t.TempDir()
	stackDir := filepath.Join(stackBase, "app-1")
	if err := os.MkdirAll(stackDir, 0o755); err != nil {
		t.Fatalf("failed to create stack dir: %v", err)
	}
	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Demo",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         stackDir,
		Status:      domain.AppStatusCreated,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	deployed := false
	appliedRouting := false
	service := NewAppService(repo, &fakeDockerRepository{
		deployFn: func(_ context.Context, app *domain.App) error {
			deployed = app.ID == "app-1"
			return nil
		},
		getStatusFn: func(context.Context, *domain.App) (string, error) {
			return domain.AppStatusRunning, nil
		},
		resolveContainerIPFn: func(_ context.Context, app *domain.App) (string, error) {
			if app == nil || app.ID != "app-1" {
				return "", domain.ErrContainerNotFound
			}
			return "10.55.0.9", nil
		},
	}, nil, stackBase)
	service.WithHostManagers(&fakeHostManager{
		applyRoutingFn: func(_ context.Context, app *domain.App, _ domain.PlatformSettings) error {
			appliedRouting = app != nil && app.ProxyContainerIP == "10.55.0.9"
			return nil
		},
	}, nil)
	repo.items["app-1"].PublicDomain = "app.example.com"
	repo.items["app-1"].ProxyTargetPort = 8080

	if err := service.DeployApp(context.Background(), "app-1"); err != nil {
		t.Fatalf("DeployApp() error = %v", err)
	}

	if !deployed {
		t.Fatal("expected docker deploy to be invoked")
	}
	if got := repo.items["app-1"].Status; got != domain.AppStatusRunning {
		t.Fatalf("expected running status, got %q", got)
	}
	if got := repo.items["app-1"].ProxyContainerIP; got != "10.55.0.9" {
		t.Fatalf("expected resolved proxy container IP, got %q", got)
	}
	if !appliedRouting {
		t.Fatal("expected routing to be applied with resolved proxy container IP")
	}
}

func TestService_DeployApp_ErrorMarksAppFailed(t *testing.T) {
	repo := newFakeAppRepository()
	stackBase := t.TempDir()
	stackDir := filepath.Join(stackBase, "app-1")
	if err := os.MkdirAll(stackDir, 0o755); err != nil {
		t.Fatalf("failed to create stack dir: %v", err)
	}
	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Demo",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         stackDir,
		Status:      domain.AppStatusCreated,
	}

	service := NewAppService(repo, &fakeDockerRepository{
		deployFn: func(context.Context, *domain.App) error {
			return errors.New("docker unavailable")
		},
	}, nil, stackBase)

	if err := service.DeployApp(context.Background(), "app-1"); err == nil {
		t.Fatal("expected deploy error")
	}

	if got := repo.items["app-1"].Status; got != domain.AppStatusError {
		t.Fatalf("expected error status, got %q", got)
	}
}

func TestService_UpdateAppConfig(t *testing.T) {
	repo := newFakeAppRepository()
	stackBase := t.TempDir()
	stackDir := filepath.Join(stackBase, "app-1")
	if err := os.MkdirAll(stackDir, 0o755); err != nil {
		t.Fatalf("failed to create stack dir: %v", err)
	}
	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Demo",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         stackDir,
		Status:      domain.AppStatusCreated,
		ManagedEnv: map[string]string{
			"KEEP_ME": "1",
		},
	}

	service := NewAppService(repo, &fakeDockerRepository{}, nil, stackBase)
	updated, err := service.UpdateAppConfig(context.Background(), "app-1", domain.AppConfig{
		PublicDomain:    "app.example.com",
		ProxyTargetPort: 8080,
		UseTLS:          true,
		ManagedEnv: map[string]string{
			"API_URL": "https://app.example.com",
			"":        "empty",
		},
	})
	if err != nil {
		t.Fatalf("UpdateAppConfig() error = %v", err)
	}

	if updated.PublicDomain != "app.example.com" {
		t.Fatalf("unexpected public domain %q", updated.PublicDomain)
	}
	if updated.ProxyTargetPort != 8080 {
		t.Fatalf("unexpected proxy target port %d", updated.ProxyTargetPort)
	}
	if !updated.UseTLS {
		t.Fatalf("expected tls enabled")
	}
	if updated.ManagedEnv["API_URL"] != "https://app.example.com" {
		t.Fatalf("expected managed env api url, got %v", updated.ManagedEnv)
	}
	if _, hasEmpty := updated.ManagedEnv[""]; hasEmpty {
		t.Fatal("expected empty managed env keys to be removed")
	}
}

func TestService_StopApp(t *testing.T) {
	repo := newFakeAppRepository()
	stackBase := t.TempDir()
	stackDir := filepath.Join(stackBase, "app-1")
	if err := os.MkdirAll(stackDir, 0o755); err != nil {
		t.Fatalf("failed to create stack dir: %v", err)
	}
	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Demo",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         stackDir,
		Status:      domain.AppStatusRunning,
	}

	stopped := false
	service := NewAppService(repo, &fakeDockerRepository{
		stopFn: func(_ context.Context, app *domain.App) error {
			stopped = app.ID == "app-1"
			return nil
		},
	}, nil, stackBase)

	if err := service.StopApp(context.Background(), "app-1"); err != nil {
		t.Fatalf("StopApp() error = %v", err)
	}

	if !stopped {
		t.Fatal("expected docker stop to be called")
	}
	if got := repo.items["app-1"].Status; got != domain.AppStatusStopped {
		t.Fatalf("expected stopped status, got %q", got)
	}
}

func cloneApp(app *domain.App) *domain.App {
	if app == nil {
		return nil
	}

	cloned := *app
	if app.Ports != nil {
		cloned.Ports = append([]string(nil), app.Ports...)
	}
	if app.ManagedEnv != nil {
		cloned.ManagedEnv = map[string]string{}
		for key, value := range app.ManagedEnv {
			cloned.ManagedEnv[key] = value
		}
	}

	return &cloned
}
