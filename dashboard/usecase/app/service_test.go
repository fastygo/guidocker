package app

import (
	"context"
	"dashboard/domain"
	"errors"
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
	getLogsFn   func(context.Context, string, int) (string, error)
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

func (r *fakeDockerRepository) GetLogs(ctx context.Context, appID string, lines int) (string, error) {
	if r.getLogsFn != nil {
		return r.getLogsFn(ctx, appID, lines)
	}
	return "", nil
}

func (r *fakeDockerRepository) ListRunning(context.Context) ([]domain.Container, error) {
	return nil, nil
}

func TestService_CreateApp(t *testing.T) {
	repo := newFakeAppRepository()
	service := NewAppService(repo, &fakeDockerRepository{}, "/opt/stacks")

	app, err := service.CreateApp(context.Background(), "Demo App", `version: "3.9"
services:
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
	if app.Dir != "/opt/stacks/"+app.ID {
		t.Fatalf("unexpected app dir %q", app.Dir)
	}
	if app.Status != domain.AppStatusCreated {
		t.Fatalf("expected created status, got %q", app.Status)
	}
	if len(app.Ports) != 1 || app.Ports[0] != "8080:80" {
		t.Fatalf("unexpected ports: %+v", app.Ports)
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
	}, "/opt/stacks")

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
	}, "/opt/stacks")

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
	}, "/opt/stacks")

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
