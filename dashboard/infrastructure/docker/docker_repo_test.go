package docker

import (
	"os"
	"path/filepath"
	"testing"

	"dashboard/domain"
)

func TestRepository_EnsureComposeSource_RecreatesFromStoredCompose(t *testing.T) {
	t.Parallel()

	repo := NewDockerRepository(t.TempDir())
	app := &domain.App{
		ID:         "app-1",
		Dir:        filepath.Join(t.TempDir(), "stack"),
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		ComposePath: "nested/docker-compose.yml",
	}
	composeFile := repo.composeFile(app)

	if err := repo.ensureComposeSource(app, composeFile); err != nil {
		t.Fatalf("ensureComposeSource() error = %v", err)
	}
	content, err := os.ReadFile(composeFile)
	if err != nil {
		t.Fatalf("expected compose file to be created: %v", err)
	}
	if string(content) != app.ComposeYAML {
		t.Fatalf("expected stored compose content, got %q", string(content))
	}
}

func TestRepository_EnsureComposeSource_UsesExistingFile(t *testing.T) {
	t.Parallel()

	repo := NewDockerRepository(t.TempDir())
	stackDir := filepath.Join(t.TempDir(), "stack")
	existing := "services:\n  api:\n    image: busybox:latest"
	app := &domain.App{
		ID:          "app-1",
		Dir:         stackDir,
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine",
		ComposePath: "nested/docker-compose.yml",
	}
	composeFile := repo.composeFile(app)
	if err := os.MkdirAll(filepath.Dir(composeFile), 0o755); err != nil {
		t.Fatalf("prepare directory: %v", err)
	}
	if err := os.WriteFile(composeFile, []byte(existing), 0o644); err != nil {
		t.Fatalf("write existing compose file: %v", err)
	}

	if err := repo.ensureComposeSource(app, composeFile); err != nil {
		t.Fatalf("ensureComposeSource() error = %v", err)
	}
	content, err := os.ReadFile(composeFile)
	if err != nil {
		t.Fatalf("read compose file: %v", err)
	}
	if string(content) != existing {
		t.Fatalf("expected existing content preserved, got %q", string(content))
	}
}

func TestRepository_EnsureComposeSource_FailsWithoutStoredCompose(t *testing.T) {
	t.Parallel()

	repo := NewDockerRepository(t.TempDir())
	app := &domain.App{
		ID:         "app-1",
		Dir:        filepath.Join(t.TempDir(), "stack"),
		ComposeYAML: "",
	}
	composeFile := repo.composeFile(app)

	if err := repo.ensureComposeSource(app, composeFile); err == nil {
		t.Fatalf("expected error when compose source is missing")
	}
}

