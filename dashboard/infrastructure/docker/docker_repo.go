package docker

import (
	"bytes"
	"context"
	"dashboard/domain"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Repository executes Docker Compose commands through the Docker CLI.
type Repository struct {
	stacksDir string
}

// NewDockerRepository creates a docker-backed repository.
func NewDockerRepository(stacksDir string) *Repository {
	return &Repository{stacksDir: stacksDir}
}

func (r *Repository) Deploy(ctx context.Context, app *domain.App) error {
	if app == nil {
		return domain.ErrAppNotFound
	}

	composeFile := r.composeFile(app.ID, app.Dir)
	if err := os.MkdirAll(filepath.Dir(composeFile), 0o755); err != nil {
		return fmt.Errorf("create app directory: %w", err)
	}
	if err := os.WriteFile(composeFile, []byte(app.ComposeYAML), 0o644); err != nil {
		return fmt.Errorf("write compose file: %w", err)
	}

	_, err := r.run(ctx, "docker", "compose", "-f", composeFile, "up", "-d")
	if err != nil {
		return fmt.Errorf("deploy app %s: %w", app.ID, err)
	}

	return nil
}

func (r *Repository) Stop(ctx context.Context, app *domain.App) error {
	if app == nil {
		return domain.ErrAppNotFound
	}

	composeFile := r.composeFile(app.ID, app.Dir)
	_, err := r.run(ctx, "docker", "compose", "-f", composeFile, "down")
	if err != nil {
		return fmt.Errorf("stop app %s: %w", app.ID, err)
	}

	return nil
}

func (r *Repository) Destroy(ctx context.Context, app *domain.App) error {
	if app == nil {
		return domain.ErrAppNotFound
	}

	composeFile := r.composeFile(app.ID, app.Dir)
	args := []string{
		"compose",
		"-p", app.ID,
		"down",
		"--volumes",
		"--remove-orphans",
		"--timeout", "30",
	}
	if _, err := os.Stat(composeFile); err == nil {
		args = append([]string{"compose", "-f", composeFile, "-p", app.ID}, args[2:]...)
	}

	_, err := r.run(ctx, "docker", args...)
	if err != nil {
		return fmt.Errorf("destroy app %s: %w", app.ID, err)
	}

	return nil
}

func (r *Repository) Restart(ctx context.Context, app *domain.App) error {
	if app == nil {
		return domain.ErrAppNotFound
	}

	composeFile := r.composeFile(app.ID, app.Dir)
	_, err := r.run(ctx, "docker", "compose", "-f", composeFile, "restart")
	if err != nil {
		return fmt.Errorf("restart app %s: %w", app.ID, err)
	}

	return nil
}

func (r *Repository) GetStatus(ctx context.Context, app *domain.App) (string, error) {
	if app == nil {
		return "", domain.ErrAppNotFound
	}

	composeFile := r.composeFile(app.ID, app.Dir)
	if _, err := os.Stat(composeFile); err != nil {
		if os.IsNotExist(err) {
			return domain.AppStatusStopped, nil
		}
		return "", err
	}

	output, err := r.run(ctx, "docker", "compose", "-f", composeFile, "ps", "--format", "json")
	if err != nil {
		return "", fmt.Errorf("get status for app %s: %w", app.ID, err)
	}

	return parseComposeStatus(output), nil
}

func (r *Repository) GetLogs(ctx context.Context, appID string, lines int) (string, error) {
	composeFile := r.composeFile(appID, "")
	output, err := r.run(ctx, "docker", "compose", "-f", composeFile, "logs", "--tail", strconv.Itoa(lines), "--no-color")
	if err != nil {
		return "", fmt.Errorf("get logs for app %s: %w", appID, err)
	}

	return string(output), nil
}

func (r *Repository) ListRunning(ctx context.Context) ([]domain.Container, error) {
	output, err := r.run(ctx, "docker", "ps", "--format", "{{json .}}")
	if err != nil {
		return nil, fmt.Errorf("list running containers: %w", err)
	}

	lines := bytes.Split(bytes.TrimSpace(output), []byte("\n"))
	if len(lines) == 1 && len(lines[0]) == 0 {
		return []domain.Container{}, nil
	}

	type dockerPSLine struct {
		ID     string `json:"ID"`
		Image  string `json:"Image"`
		Names  string `json:"Names"`
		Ports  string `json:"Ports"`
		Status string `json:"Status"`
		State  string `json:"State"`
	}

	containers := make([]domain.Container, 0, len(lines))
	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var entry dockerPSLine
		if err := json.Unmarshal(line, &entry); err != nil {
			return nil, fmt.Errorf("decode docker ps line: %w", err)
		}

		containers = append(containers, domain.Container{
			ID:     entry.ID,
			Name:   entry.Names,
			Image:  entry.Image,
			Status: normalizeDockerPSStatus(entry.State, entry.Status),
			Ports:  splitPorts(entry.Ports),
		})
	}

	return containers, nil
}

func (r *Repository) composeFile(appID, appDir string) string {
	if strings.TrimSpace(appDir) == "" {
		appDir = filepath.Join(r.stacksDir, appID)
	}

	return filepath.Join(appDir, "docker-compose.yml")
}

func (r *Repository) run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}

	return output, nil
}

type composeServiceState struct {
	State  string `json:"State"`
	Status string `json:"Status"`
}

func parseComposeStatus(output []byte) string {
	trimmed := bytes.TrimSpace(output)
	if len(trimmed) == 0 {
		return domain.AppStatusStopped
	}

	services := make([]composeServiceState, 0)
	if trimmed[0] == '[' {
		_ = json.Unmarshal(trimmed, &services)
	} else {
		for _, line := range bytes.Split(trimmed, []byte("\n")) {
			var item composeServiceState
			if err := json.Unmarshal(line, &item); err == nil {
				services = append(services, item)
			}
		}
	}

	if len(services) == 0 {
		return domain.AppStatusStopped
	}

	hasRunning := false
	for _, service := range services {
		state := strings.ToLower(strings.TrimSpace(service.State))
		status := strings.ToLower(strings.TrimSpace(service.Status))
		value := state
		if value == "" {
			value = status
		}

		switch {
		case strings.Contains(value, "running"):
			hasRunning = true
		case strings.Contains(value, "restart"),
			strings.Contains(value, "dead"),
			strings.Contains(value, "error"),
			strings.Contains(value, "exit 1"),
			strings.Contains(value, "failed"):
			return domain.AppStatusError
		}
	}

	if hasRunning {
		return domain.AppStatusRunning
	}

	return domain.AppStatusStopped
}

func splitPorts(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	ports := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			ports = append(ports, part)
		}
	}

	return ports
}

func normalizeDockerPSStatus(state, status string) string {
	value := strings.ToLower(strings.TrimSpace(state))
	if value == "" {
		value = strings.ToLower(strings.TrimSpace(status))
	}

	switch {
	case strings.Contains(value, "running"), strings.Contains(value, "up"):
		return domain.ContainerStatusRunning
	case strings.Contains(value, "paused"):
		return domain.ContainerStatusPaused
	default:
		return domain.ContainerStatusStopped
	}
}

var _ domain.DockerRepository = (*Repository)(nil)
