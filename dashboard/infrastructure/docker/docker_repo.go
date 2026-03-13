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
	"sort"
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

	lines := splitDockerPSOutput(output)
	if len(lines) == 0 {
		return []domain.Container{}, nil
	}

	return parseDockerPS(lines), nil
}

func (r *Repository) ListAllContainers(ctx context.Context) ([]domain.Container, error) {
	output, err := r.run(ctx, "docker", "ps", "-a", "--format", "{{json .}}")
	if err != nil {
		return nil, fmt.Errorf("list all containers: %w", err)
	}

	lines := splitDockerPSOutput(output)
	if len(lines) == 0 {
		return []domain.Container{}, nil
	}

	return parseDockerPS(lines), nil
}

func (r *Repository) InspectContainers(ctx context.Context, ids []string) ([]domain.ContainerDetail, error) {
	if len(ids) == 0 {
		return []domain.ContainerDetail{}, nil
	}

	args := make([]string, 0, len(ids)+1)
	args = append(args, "inspect")
	args = append(args, ids...)

	output, err := r.run(ctx, "docker", args...)
	if err != nil {
		return nil, fmt.Errorf("inspect containers: %w", err)
	}

	output = bytes.TrimSpace(output)
	if len(output) == 0 {
		return []domain.ContainerDetail{}, nil
	}

	var items []dockerInspect
	if err := json.Unmarshal(output, &items); err != nil {
		return nil, fmt.Errorf("decode docker inspect: %w", err)
	}

	details := make([]domain.ContainerDetail, 0, len(items))
	for _, item := range items {
		labels := item.Config.Labels
		if labels == nil {
			labels = map[string]string{}
		}

		detail := domain.ContainerDetail{
			ID:     item.ID,
			Name:   strings.TrimPrefix(item.Name, "/"),
			Image:  item.Config.Image,
			Labels: labels,
			Envs:   append([]string(nil), item.Config.Env...),
			Status: normalizeInspectStatus(item.State.Status),
			Ports:  parseDockerInspectPorts(item.NetworkSettings.Ports),
		}
		if detail.Image == "" {
			detail.Image = item.Image
		}
		detail.Mounts = parseDockerInspectMounts(item.Mounts)
		details = append(details, detail)
	}

	return details, nil
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

func splitDockerPSOutput(output []byte) [][]byte {
	lines := bytes.Split(bytes.TrimSpace(output), []byte("\n"))
	filtered := make([][]byte, 0, len(lines))
	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		filtered = append(filtered, line)
	}
	return filtered
}

func parseDockerPS(lines [][]byte) []domain.Container {
	containers := make([]domain.Container, 0, len(lines))
	for _, line := range lines {
		var entry dockerPSLine
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		containers = append(containers, domain.Container{
			ID:     entry.ID,
			Name:   strings.TrimSpace(entry.Names),
			Image:  entry.Image,
			Status: normalizeDockerPSStatus(entry.State, entry.Status),
			Ports:  splitPorts(entry.Ports),
		})
	}

	return containers
}

func parseDockerInspectMounts(mounts []dockerInspectMount) []string {
	paths := make([]string, 0, len(mounts))
	for _, mount := range mounts {
		if strings.TrimSpace(mount.Destination) != "" {
			paths = append(paths, mount.Destination)
			continue
		}
		if strings.TrimSpace(mount.Target) != "" {
			paths = append(paths, mount.Target)
		}
	}
	return paths
}

func parseDockerInspectPorts(raw map[string][]dockerInspectPortBinding) []string {
	ports := make([]string, 0, len(raw))
	for containerPort, bindings := range raw {
		if len(bindings) == 0 {
			ports = append(ports, containerPort)
			continue
		}
		for _, binding := range bindings {
			ip := strings.TrimSpace(binding.HostIP)
			entry := fmt.Sprintf("%s->%s", binding.HostPort, containerPort)
			if ip != "" {
				entry = fmt.Sprintf("%s:%s->%s", ip, binding.HostPort, containerPort)
			}
			ports = append(ports, entry)
		}
	}

	sort.Strings(ports)
	return ports
}

func normalizeInspectStatus(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.Contains(value, "running"), strings.Contains(value, "up"):
		return domain.ContainerStatusRunning
	case strings.Contains(value, "paused"):
		return domain.ContainerStatusPaused
	default:
		return domain.ContainerStatusStopped
	}
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

type dockerPSLine struct {
	ID     string `json:"ID"`
	Image  string `json:"Image"`
	Names  string `json:"Names"`
	Ports  string `json:"Ports"`
	Status string `json:"Status"`
	State  string `json:"State"`
}

type dockerInspect struct {
	ID             string                `json:"Id"`
	Name           string                `json:"Name"`
	Image          string                `json:"Image"`
	Config         dockerInspectConfig   `json:"Config"`
	State          dockerInspectState    `json:"State"`
	Mounts         []dockerInspectMount  `json:"Mounts"`
	NetworkSettings dockerInspectNetwork `json:"NetworkSettings"`
}

type dockerInspectConfig struct {
	Image  string            `json:"Image"`
	Labels map[string]string `json:"Labels"`
	Env    []string          `json:"Env"`
}

type dockerInspectState struct {
	Status string `json:"Status"`
}

type dockerInspectMount struct {
	Destination string `json:"Destination"`
	Target      string `json:"Target"`
}

type dockerInspectNetwork struct {
	Ports map[string][]dockerInspectPortBinding `json:"Ports"`
}

type dockerInspectPortBinding struct {
	HostIP   string `json:"HostIP"`
	HostPort string `json:"HostPort"`
}

var _ domain.DockerRepository = (*Repository)(nil)
