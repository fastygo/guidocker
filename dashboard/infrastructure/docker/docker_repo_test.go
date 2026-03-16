package docker

import (
	"os"
	"path/filepath"
	"strings"
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

func TestRepository_EnsureComposeOverride_WritesManagedNetwork(t *testing.T) {
	t.Parallel()

	repo := NewDockerRepository(t.TempDir())
	stackDir := filepath.Join(t.TempDir(), "stack")
	app := &domain.App{
		ID:          "app-1",
		Dir:         stackDir,
		ComposeYAML: "services:\n  web:\n    image: nginx:alpine\n  worker:\n    image: busybox:latest",
		ComposePath: "nested/docker-compose.yml",
	}
	composeFile := repo.composeFile(app)

	if err := repo.ensureComposeOverride(app, composeFile); err != nil {
		t.Fatalf("ensureComposeOverride() error = %v", err)
	}

	overrideFile := repo.composeOverrideFile(app)
	content, err := os.ReadFile(overrideFile)
	if err != nil {
		t.Fatalf("expected override file to be created: %v", err)
	}
	rendered := string(content)
	if !strings.Contains(rendered, "paas-network") {
		t.Fatalf("expected override to include managed network, got %q", rendered)
	}
	if !strings.Contains(rendered, "web:") || !strings.Contains(rendered, "worker:") {
		t.Fatalf("expected override to include all services, got %q", rendered)
	}
}

func TestPickProxyContainerIP_PrefersMatchingPortOnManagedNetwork(t *testing.T) {
	t.Parallel()

	items := []dockerInspect{
		{
			Name: "/app-worker-1",
			Config: dockerInspectConfig{
				Labels: map[string]string{
					"com.docker.compose.service": "worker",
				},
			},
			State: dockerInspectState{Status: "running"},
			NetworkSettings: dockerInspectNetwork{
				Networks: map[string]dockerInspectEndpoint{
					"paas-network": {IPAddress: "10.10.0.12"},
				},
			},
		},
		{
			Name: "/app-web-1",
			Config: dockerInspectConfig{
				Labels: map[string]string{
					"com.docker.compose.service": "web",
				},
				ExposedPorts: map[string]any{
					"8080/tcp": struct{}{},
				},
			},
			State: dockerInspectState{Status: "running"},
			NetworkSettings: dockerInspectNetwork{
				Networks: map[string]dockerInspectEndpoint{
					"paas-network": {IPAddress: "10.10.0.20"},
				},
			},
		},
	}

	if got := pickProxyContainerIP(items, "paas-network", 8080); got != "10.10.0.20" {
		t.Fatalf("expected matching container IP, got %q", got)
	}
}
