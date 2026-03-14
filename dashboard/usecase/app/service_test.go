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
	deployFn    func(context.Context, *domain.App) error
	stopFn      func(context.Context, *domain.App) error
	restartFn   func(context.Context, *domain.App) error
	getStatusFn func(context.Context, *domain.App) (string, error)
	getLogsFn   func(context.Context, *domain.App, int) (string, error)
	destroyFn   func(context.Context, *domain.App) error
	listAllFn   func(context.Context) ([]domain.Container, error)
	inspectFn   func(context.Context, []string) ([]domain.ContainerDetail, error)
}

type fakeGitRepository struct {
	cloneFn func(context.Context, string, string, string) (string, error)
}

func (r *fakeDockerRepository) Deploy(ctx context.Context, app *domain.App) error {
	if r.deployFn != nil {
		return r.deployFn(ctx, app)
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

func TestService_DeleteApp_DockerErrorIgnored(t *testing.T) {
	repo := newFakeAppRepository()
	stackDir := filepath.Join(t.TempDir(), "app-1")
	if err := os.MkdirAll(stackDir, 0o755); err != nil {
		t.Fatalf("failed to create stack dir: %v", err)
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
			return errors.New("docker unavailable")
		},
	}, nil, "/opt/stacks")

	if err := service.DeleteApp(context.Background(), "app-1"); err != nil {
		t.Fatalf("DeleteApp() error = %v", err)
	}

	if !destroyed {
		t.Fatalf("expected destroy to be called")
	}
	if _, ok := repo.items["app-1"]; ok {
		t.Fatalf("expected app removed from repository")
	}
	if _, err := os.Stat(stackDir); !os.IsNotExist(err) {
		t.Fatalf("expected stack dir removed, stat error: %v", err)
	}
}

func TestService_CreateApp(t *testing.T) {
	repo := newFakeAppRepository()
	service := NewAppService(repo, &fakeDockerRepository{}, nil, "/opt/stacks")

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
	expectedDir := filepath.Join("/opt/stacks", app.ID)
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
	service := NewAppService(repo, &fakeDockerRepository{}, nil, "/opt/stacks")

	_, err := service.CreateApp(context.Background(), "Demo App", `web:
  image: nginx:alpine`)
	if !errors.Is(err, domain.ErrComposeNoServices) {
		t.Fatalf("expected ErrComposeNoServices, got %v", err)
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
	if !strings.Contains(app.ComposeYAML, "\"8080:8080\"") {
		t.Fatalf("expected generated compose with required port mapping, got %q", app.ComposeYAML)
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
	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Demo",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         "/opt/stacks/app-1",
		Status:      domain.AppStatusCreated,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	deployed := false
	service := NewAppService(repo, &fakeDockerRepository{
		deployFn: func(_ context.Context, app *domain.App) error {
			deployed = app.ID == "app-1"
			return nil
		},
		getStatusFn: func(context.Context, *domain.App) (string, error) {
			return domain.AppStatusRunning, nil
		},
	}, nil, "/opt/stacks")

	if err := service.DeployApp(context.Background(), "app-1"); err != nil {
		t.Fatalf("DeployApp() error = %v", err)
	}

	if !deployed {
		t.Fatal("expected docker deploy to be invoked")
	}
	if got := repo.items["app-1"].Status; got != domain.AppStatusRunning {
		t.Fatalf("expected running status, got %q", got)
	}
}

func TestService_DeployApp_ErrorMarksAppFailed(t *testing.T) {
	repo := newFakeAppRepository()
	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Demo",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         "/opt/stacks/app-1",
		Status:      domain.AppStatusCreated,
	}

	service := NewAppService(repo, &fakeDockerRepository{
		deployFn: func(context.Context, *domain.App) error {
			return errors.New("docker unavailable")
		},
	}, nil, "/opt/stacks")

	if err := service.DeployApp(context.Background(), "app-1"); err == nil {
		t.Fatal("expected deploy error")
	}

	if got := repo.items["app-1"].Status; got != domain.AppStatusError {
		t.Fatalf("expected error status, got %q", got)
	}
}

func TestService_StopApp(t *testing.T) {
	repo := newFakeAppRepository()
	repo.items["app-1"] = &domain.App{
		ID:          "app-1",
		Name:        "Demo",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         "/opt/stacks/app-1",
		Status:      domain.AppStatusRunning,
	}

	stopped := false
	service := NewAppService(repo, &fakeDockerRepository{
		stopFn: func(_ context.Context, app *domain.App) error {
			stopped = app.ID == "app-1"
			return nil
		},
	}, nil, "/opt/stacks")

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

	return &cloned
}
