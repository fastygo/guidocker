package bolt

import (
	"context"
	"encoding/json"
	"errors"
	"gui-docker/domain"
	"path/filepath"
	"testing"
	"time"

	bbolt "go.etcd.io/bbolt"
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

func TestAppRepository_BackwardCompatibleLoad(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "apps.db")
	repo, err := NewAppRepository(dbPath)
	if err != nil {
		t.Fatalf("NewAppRepository() error = %v", err)
	}
	defer func() { _ = repo.Close() }()

	legacy := struct {
		ID          string    `json:"id"`
		Name        string    `json:"name"`
		ComposeYAML string    `json:"compose_yaml"`
		SourceType  string    `json:"source_type"`
		RepoURL     string    `json:"repo_url"`
		Dir         string    `json:"dir"`
		Status      string    `json:"status"`
		Ports       []string  `json:"ports"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
	}{
		ID:          "legacy-app",
		Name:        "Legacy",
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		SourceType:  "manual",
		Dir:         "/opt/stacks/legacy-app",
		Status:      domain.AppStatusCreated,
		Ports:       []string{"8080:80"},
		CreatedAt:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 3, 1, 0, 10, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal legacy record = %v", err)
	}

	if err := repo.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(appsBucket))
		return bucket.Put([]byte("legacy-app"), raw)
	}); err != nil {
		t.Fatalf("seed legacy record = %v", err)
	}

	app, err := repo.GetByID(context.Background(), "legacy-app")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if app.PublicDomain != "" {
		t.Fatalf("expected backward compatible app with empty public domain, got %q", app.PublicDomain)
	}
	if app.ProxyTargetPort != 0 {
		t.Fatalf("expected backward compatible app with empty proxy port, got %d", app.ProxyTargetPort)
	}
	if app.UseTLS {
		t.Fatalf("expected backward compatible app with tls disabled")
	}
	if app.ManagedEnv != nil {
		t.Fatalf("expected backward compatible app with no managed env map, got %v", app.ManagedEnv)
	}
}
