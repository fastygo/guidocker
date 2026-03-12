package bolt

import (
	"context"
	"dashboard/domain"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestAppRepository_CRUD(t *testing.T) {
	repo, err := NewAppRepository(filepath.Join(t.TempDir(), "apps.db"))
	if err != nil {
		t.Fatalf("NewAppRepository() error = %v", err)
	}
	defer func() { _ = repo.Close() }()

	now := time.Now().UTC()
	input := &domain.App{
		ID:          "app-1",
		Name:        "Demo",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		Dir:         "/opt/stacks/app-1",
		Status:      domain.AppStatusCreated,
		Ports:       []string{"8080:80"},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := repo.Create(context.Background(), input); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	stored, err := repo.GetByID(context.Background(), "app-1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored.Name != "Demo" {
		t.Fatalf("expected app name Demo, got %q", stored.Name)
	}

	stored.Status = domain.AppStatusRunning
	if err := repo.Update(context.Background(), stored); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	apps, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(apps) != 1 || apps[0].Status != domain.AppStatusRunning {
		t.Fatalf("unexpected apps list: %+v", apps)
	}

	if err := repo.Delete(context.Background(), "app-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := repo.GetByID(context.Background(), "app-1"); !errors.Is(err, domain.ErrAppNotFound) {
		t.Fatalf("expected app not found after delete, got %v", err)
	}
}

func TestAppRepository_UpdateMissingApp(t *testing.T) {
	repo, err := NewAppRepository(filepath.Join(t.TempDir(), "apps.db"))
	if err != nil {
		t.Fatalf("NewAppRepository() error = %v", err)
	}
	defer func() { _ = repo.Close() }()

	err = repo.Update(context.Background(), &domain.App{ID: "missing"})
	if !errors.Is(err, domain.ErrAppNotFound) {
		t.Fatalf("expected ErrAppNotFound, got %v", err)
	}
}
