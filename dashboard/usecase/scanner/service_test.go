package scanner

import (
	"context"
	"dashboard/domain"
	"os"
	"path/filepath"
	"testing"
)

func TestRunScan_AllManaged(t *testing.T) {
	t.Parallel()

	appRepo := newFakeScannerAppRepository()
	stackDir := t.TempDir()
	appID := "web-app-1234"
	appDir := filepath.Join(stackDir, appID)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("create stack dir: %v", err)
	}

	appRepo.items[appID] = &domain.App{
		ID:     appID,
		Name:   "Web App",
		Dir:    appDir,
		Status: domain.AppStatusCreated,
	}

	dockerRepo := &fakeScannerDockerRepository{
		listAllFn: func(_ context.Context) ([]domain.Container, error) {
			return []domain.Container{
				{ID: "id-web", Name: "web-app-1234_web", Status: domain.ContainerStatusRunning},
			}, nil
		},
		inspectFn: func(_ context.Context, ids []string) ([]domain.ContainerDetail, error) {
			if len(ids) != 1 || ids[0] != "id-web" {
				t.Fatalf("unexpected IDs: %#v", ids)
			}
			return []domain.ContainerDetail{
				{
					ID:     "id-web",
					Name:   "web-app-1234_web",
					Image:  "nginx:alpine",
					Labels: map[string]string{"com.docker.compose.project": appID},
					Status: domain.ContainerStatusRunning,
				},
			}, nil
		},
	}

	service := NewScannerService(dockerRepo, appRepo, stackDir)
	report, err := service.RunScan(context.Background())
	if err != nil {
		t.Fatalf("RunScan() error = %v", err)
	}

	if len(report.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(report.Resources))
	}
	if report.Resources[0].Kind != domain.ResourceManaged {
		t.Fatalf("expected managed kind, got %q", report.Resources[0].Kind)
	}
}

func TestRunScan_OrphanDir(t *testing.T) {
	t.Parallel()

	appRepo := newFakeScannerAppRepository()
	stackDir := t.TempDir()
	orphanDir := filepath.Join(stackDir, "orphan-stack")
	if err := os.MkdirAll(orphanDir, 0o755); err != nil {
		t.Fatalf("create orphan dir: %v", err)
	}

	dockerRepo := &fakeScannerDockerRepository{
		listAllFn: func(_ context.Context) ([]domain.Container, error) {
			return []domain.Container{}, nil
		},
		inspectFn: func(_ context.Context, _ []string) ([]domain.ContainerDetail, error) {
			return []domain.ContainerDetail{}, nil
		},
	}

	service := NewScannerService(dockerRepo, appRepo, stackDir)
	report, err := service.RunScan(context.Background())
	if err != nil {
		t.Fatalf("RunScan() error = %v", err)
	}

	resource := findResourceByKind(report.Resources, domain.ResourceOrphanDir)
	if resource == nil {
		t.Fatalf("expected orphan_dir resource")
	}
	if resource.Dir != orphanDir {
		t.Fatalf("expected orphan dir %q, got %q", orphanDir, resource.Dir)
	}
}

func TestRunScan_OrphanRuntime(t *testing.T) {
	t.Parallel()

	appRepo := newFakeScannerAppRepository()
	dockerRepo := &fakeScannerDockerRepository{
		listAllFn: func(_ context.Context) ([]domain.Container, error) {
			return []domain.Container{
				{ID: "id-legacy", Name: "legacy", Status: domain.ContainerStatusRunning},
			}, nil
		},
		inspectFn: func(_ context.Context, _ []string) ([]domain.ContainerDetail, error) {
			return []domain.ContainerDetail{
				{
					ID:     "id-legacy",
					Name:   "legacy-service",
					Image:  "busybox",
					Status: domain.ContainerStatusRunning,
				},
			}, nil
		},
	}

	service := NewScannerService(dockerRepo, appRepo, t.TempDir())
	report, err := service.RunScan(context.Background())
	if err != nil {
		t.Fatalf("RunScan() error = %v", err)
	}

	resource := findResourceByKind(report.Resources, domain.ResourceOrphanRuntime)
	if resource == nil {
		t.Fatalf("expected orphan_runtime resource")
	}
}

func TestRunScan_BrokenApp(t *testing.T) {
	t.Parallel()

	stackDir := t.TempDir()
	appRepo := newFakeScannerAppRepository()
	appID := "broken-app-1111"
	appDir := filepath.Join(stackDir, appID)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("create app dir: %v", err)
	}
	appRepo.items[appID] = &domain.App{ID: appID, Name: "Broken App", Dir: appDir, Status: domain.AppStatusCreated}

	dockerRepo := &fakeScannerDockerRepository{
		listAllFn: func(_ context.Context) ([]domain.Container, error) {
			return []domain.Container{}, nil
		},
		inspectFn: func(_ context.Context, _ []string) ([]domain.ContainerDetail, error) {
			return []domain.ContainerDetail{}, nil
		},
	}

	service := NewScannerService(dockerRepo, appRepo, stackDir)
	report, err := service.RunScan(context.Background())
	if err != nil {
		t.Fatalf("RunScan() error = %v", err)
	}

	resource := findResourceByKind(report.Resources, domain.ResourceBrokenApp)
	if resource == nil {
		t.Fatalf("expected broken_app resource")
	}
	if len(resource.CleanupCmds) == 0 {
		t.Fatalf("expected cleanup commands for broken app")
	}
}

func TestRunScan_StaleAdmin(t *testing.T) {
	t.Parallel()

	appRepo := newFakeScannerAppRepository()
	dockerRepo := &fakeScannerDockerRepository{
		listAllFn: func(_ context.Context) ([]domain.Container, error) {
			return []domain.Container{
				{ID: "id-admin", Name: "dashboard-admin", Status: domain.ContainerStatusRunning},
			}, nil
		},
		inspectFn: func(_ context.Context, _ []string) ([]domain.ContainerDetail, error) {
			return []domain.ContainerDetail{
				{
					ID:     "id-admin",
					Name:   "dashboard-admin",
					Image:  "example/dashboard:latest",
					Labels: map[string]string{},
					Mounts: []string{
						"/var/run/docker.sock",
						"/opt/stacks",
					},
					Envs:   []string{"PAAS_ADMIN_USER=admin", "STACKS_DIR=/opt/stacks"},
					Ports:  []string{"3000->80/tcp"},
					Status: domain.ContainerStatusRunning,
				},
			}, nil
		},
	}

	service := NewScannerService(dockerRepo, appRepo, t.TempDir())
	report, err := service.RunScan(context.Background())
	if err != nil {
		t.Fatalf("RunScan() error = %v", err)
	}

	resource := findResourceByKind(report.Resources, domain.ResourceStaleAdmin)
	if resource == nil {
		t.Fatalf("expected stale_admin resource")
	}
}

func TestRunScan_CleanupCmds(t *testing.T) {
	t.Parallel()

	stackDir := t.TempDir()
	appRepo := newFakeScannerAppRepository()
	appRepo.items["broken-app-2222"] = &domain.App{
		ID:   "broken-app-2222",
		Name: "Broken App",
		Dir:  filepath.Join(stackDir, "broken-app-2222"),
	}
	if err := os.MkdirAll(appRepo.items["broken-app-2222"].Dir, 0o755); err != nil {
		t.Fatalf("create app dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(stackDir, "orphan-dir"), 0o755); err != nil {
		t.Fatalf("create orphan dir: %v", err)
	}

	dockerRepo := &fakeScannerDockerRepository{
		listAllFn: func(_ context.Context) ([]domain.Container, error) {
			return []domain.Container{
				{ID: "id-orphan", Name: "orphan-run", Status: domain.ContainerStatusRunning},
				{ID: "id-admin", Name: "dashboard-old", Status: domain.ContainerStatusRunning},
			}, nil
		},
		inspectFn: func(_ context.Context, _ []string) ([]domain.ContainerDetail, error) {
			return []domain.ContainerDetail{
				{
					ID:     "id-orphan",
					Name:   "orphan-run",
					Image:  "nginx:alpine",
					Status: domain.ContainerStatusRunning,
				},
				{
					ID:     "id-admin",
					Name:   "dashboard-old",
					Image:  "example/dashboard:latest",
					Mounts: []string{"/var/run/docker.sock", "/opt/stacks"},
					Ports:  []string{"3000->3000/tcp"},
					Envs:   []string{"PAAS_ADMIN_USER=admin"},
					Status: domain.ContainerStatusRunning,
				},
			}, nil
		},
	}

	service := NewScannerService(dockerRepo, appRepo, stackDir)
	report, err := service.RunScan(context.Background())
	if err != nil {
		t.Fatalf("RunScan() error = %v", err)
	}

	var stale, orphanDir, orphanRuntime bool
	for _, resource := range report.Resources {
		switch resource.Kind {
		case domain.ResourceStaleAdmin:
			stale = stale || len(resource.CleanupCmds) > 0
		case domain.ResourceOrphanDir:
			orphanDir = orphanDir || len(resource.CleanupCmds) > 0
		case domain.ResourceOrphanRuntime:
			orphanRuntime = orphanRuntime || len(resource.CleanupCmds) > 0
		}
	}
	if !stale {
		t.Fatal("expected stale_admin resource to include cleanup commands")
	}
	if !orphanDir {
		t.Fatal("expected orphan_dir resource to include cleanup commands")
	}
	if !orphanRuntime {
		t.Fatal("expected orphan_runtime resource to include cleanup commands")
	}
}

func TestRunScan_NonManagedAndManagedMix(t *testing.T) {
	t.Parallel()

	stackDir := t.TempDir()
	appRepo := newFakeScannerAppRepository()
	managedID := "managed-1111"
	managedDir := filepath.Join(stackDir, managedID)
	if err := os.MkdirAll(managedDir, 0o755); err != nil {
		t.Fatalf("create managed dir: %v", err)
	}
	appRepo.items[managedID] = &domain.App{
		ID:   managedID,
		Name: "Managed App",
		Dir:  managedDir,
	}

	dockerRepo := &fakeScannerDockerRepository{
		listAllFn: func(_ context.Context) ([]domain.Container, error) {
			return []domain.Container{
				{ID: "id-managed", Name: "managed-1111_web", Status: domain.ContainerStatusRunning},
				{ID: "id-orphan", Name: "orphan-runtime", Status: domain.ContainerStatusRunning},
			}, nil
		},
		inspectFn: func(_ context.Context, _ []string) ([]domain.ContainerDetail, error) {
			return []domain.ContainerDetail{
				{
					ID:     "id-managed",
					Name:   "managed-1111_web",
					Image:  "nginx:alpine",
					Labels: map[string]string{"com.docker.compose.project": managedID},
					Status: domain.ContainerStatusRunning,
				},
				{
					ID:     "id-orphan",
					Name:   "orphan-runtime",
					Image:  "alpine",
					Status: domain.ContainerStatusRunning,
				},
			}, nil
		},
	}

	service := NewScannerService(dockerRepo, appRepo, stackDir)
	report, err := service.RunScan(context.Background())
	if err != nil {
		t.Fatalf("RunScan() error = %v", err)
	}

	var managedSeen, orphanSeen bool
	for _, resource := range report.Resources {
		if resource.Kind == domain.ResourceManaged {
			managedSeen = true
		}
		if resource.Kind == domain.ResourceOrphanRuntime {
			orphanSeen = true
		}
	}
	if !managedSeen {
		t.Fatal("expected managed resource")
	}
	if !orphanSeen {
		t.Fatal("expected orphan_runtime resource")
	}
}

func findResourceByKind(resources []domain.ScanResource, kind domain.ResourceKind) *domain.ScanResource {
	for _, resource := range resources {
		if resource.Kind == kind {
			return &resource
		}
	}
	return nil
}

type fakeScannerAppRepository struct {
	items map[string]*domain.App
}

func newFakeScannerAppRepository() *fakeScannerAppRepository {
	return &fakeScannerAppRepository{items: map[string]*domain.App{}}
}

func (r *fakeScannerAppRepository) Create(_ context.Context, app *domain.App) error {
	r.items[app.ID] = cloneApp(app)
	return nil
}

func (r *fakeScannerAppRepository) Update(_ context.Context, app *domain.App) error {
	if _, ok := r.items[app.ID]; !ok {
		return domain.ErrAppNotFound
	}
	r.items[app.ID] = cloneApp(app)
	return nil
}

func (r *fakeScannerAppRepository) Delete(_ context.Context, id string) error {
	if _, ok := r.items[id]; !ok {
		return domain.ErrAppNotFound
	}
	delete(r.items, id)
	return nil
}

func (r *fakeScannerAppRepository) GetByID(_ context.Context, id string) (*domain.App, error) {
	app, ok := r.items[id]
	if !ok {
		return nil, domain.ErrAppNotFound
	}
	return cloneApp(app), nil
}

func (r *fakeScannerAppRepository) List(_ context.Context) ([]*domain.App, error) {
	apps := make([]*domain.App, 0, len(r.items))
	for _, app := range r.items {
		apps = append(apps, cloneApp(app))
	}
	return apps, nil
}

type fakeScannerDockerRepository struct {
	listAllFn  func(context.Context) ([]domain.Container, error)
	inspectFn  func(context.Context, []string) ([]domain.ContainerDetail, error)
	deployFn   func(context.Context, *domain.App) error
	stopFn     func(context.Context, *domain.App) error
	restartFn  func(context.Context, *domain.App) error
	destroyFn  func(context.Context, *domain.App) error
	getStatusFn func(context.Context, *domain.App) (string, error)
	getLogsFn   func(context.Context, *domain.App, int) (string, error)
	ensureNetworkFn func(context.Context) error
	resolveContainerIPFn func(context.Context, *domain.App) (string, error)
}

func (r *fakeScannerDockerRepository) Deploy(ctx context.Context, app *domain.App) error {
	if r.deployFn != nil {
		return r.deployFn(ctx, app)
	}
	return nil
}

func (r *fakeScannerDockerRepository) EnsureNetwork(ctx context.Context) error {
	if r.ensureNetworkFn != nil {
		return r.ensureNetworkFn(ctx)
	}
	return nil
}

func (r *fakeScannerDockerRepository) Stop(ctx context.Context, app *domain.App) error {
	if r.stopFn != nil {
		return r.stopFn(ctx, app)
	}
	return nil
}

func (r *fakeScannerDockerRepository) Restart(ctx context.Context, app *domain.App) error {
	if r.restartFn != nil {
		return r.restartFn(ctx, app)
	}
	return nil
}

func (r *fakeScannerDockerRepository) Destroy(ctx context.Context, app *domain.App) error {
	if r.destroyFn != nil {
		return r.destroyFn(ctx, app)
	}
	return nil
}

func (r *fakeScannerDockerRepository) GetStatus(ctx context.Context, app *domain.App) (string, error) {
	if r.getStatusFn != nil {
		return r.getStatusFn(ctx, app)
	}
	return domain.AppStatusRunning, nil
}

func (r *fakeScannerDockerRepository) GetLogs(ctx context.Context, app *domain.App, lines int) (string, error) {
	if r.getLogsFn != nil {
		return r.getLogsFn(ctx, app, lines)
	}
	return "", nil
}

func (r *fakeScannerDockerRepository) ListRunning(context.Context) ([]domain.Container, error) {
	return nil, nil
}

func (r *fakeScannerDockerRepository) ListAllContainers(ctx context.Context) ([]domain.Container, error) {
	if r.listAllFn != nil {
		return r.listAllFn(ctx)
	}
	return []domain.Container{}, nil
}

func (r *fakeScannerDockerRepository) InspectContainers(ctx context.Context, ids []string) ([]domain.ContainerDetail, error) {
	if r.inspectFn != nil {
		return r.inspectFn(ctx, ids)
	}
	return []domain.ContainerDetail{}, nil
}

func (r *fakeScannerDockerRepository) ResolveContainerIP(ctx context.Context, app *domain.App) (string, error) {
	if r.resolveContainerIPFn != nil {
		return r.resolveContainerIPFn(ctx, app)
	}
	return "", domain.ErrContainerNotFound
}

func cloneApp(app *domain.App) *domain.App {
	if app == nil {
		return nil
	}
	copied := *app
	if app.Ports != nil {
		copied.Ports = append([]string{}, app.Ports...)
	}
	return &copied
}
