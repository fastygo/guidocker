package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"gui-docker/domain"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Repository executes Docker Compose commands through the Docker CLI.
type Repository struct {
	stacksDir   string
	networkName string
}

const managedEnvFileName = ".platform.env"
const composeOverrideFileName = "docker-compose.override.yml"
const defaultAppNetworkName = "paas-network"

// NewDockerRepository creates a docker-backed repository.
func NewDockerRepository(stacksDir string) *Repository {
	networkName := strings.TrimSpace(os.Getenv("PAAS_APP_NETWORK"))
	if networkName == "" {
		networkName = defaultAppNetworkName
	}
	return &Repository{stacksDir: stacksDir, networkName: networkName}
}

func (r *Repository) EnsureNetwork(ctx context.Context) error {
	networkName := strings.TrimSpace(r.networkName)
	if networkName == "" {
		return nil
	}
	if _, err := r.run(ctx, "docker", "network", "inspect", networkName); err == nil {
		return nil
	}
	if _, err := r.run(ctx, "docker", "network", "create", networkName); err != nil && !strings.Contains(strings.ToLower(err.Error()), "already exists") {
		return fmt.Errorf("ensure app network %s: %w", networkName, err)
	}
	return nil
}

func (r *Repository) ResolveContainerIP(ctx context.Context, app *domain.App) (string, error) {
	if app == nil {
		return "", domain.ErrAppNotFound
	}

	projectID := strings.TrimSpace(app.ID)
	if projectID == "" {
		return "", domain.ErrAppNotFound
	}

	output, err := r.run(ctx, "docker", "ps", "-q", "--filter", "label=com.docker.compose.project="+projectID)
	if err != nil {
		return "", fmt.Errorf("list project containers: %w", err)
	}

	lines := splitDockerPSOutput(output)
	if len(lines) == 0 {
		return "", domain.ErrContainerNotFound
	}

	ids := make([]string, 0, len(lines))
	for _, line := range lines {
		ids = append(ids, string(bytes.TrimSpace(line)))
	}

	items, err := r.inspectRaw(ctx, ids)
	if err != nil {
		return "", err
	}

	ip := pickProxyContainerIP(items, r.networkName, app.ProxyTargetPort)
	if ip == "" {
		return "", domain.ErrContainerNotFound
	}
	return ip, nil
}

func (r *Repository) Deploy(ctx context.Context, app *domain.App) error {
	if app == nil {
		return domain.ErrAppNotFound
	}
	if err := r.EnsureNetwork(ctx); err != nil {
		return err
	}

	composeFile := r.composeFile(app)
	if err := r.ensureComposeSource(app, composeFile); err != nil {
		return fmt.Errorf("prepare compose file: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(composeFile), 0o755); err != nil {
		return fmt.Errorf("create app directory: %w", err)
	}
	if err := os.WriteFile(composeFile, []byte(app.ComposeYAML), 0o644); err != nil {
		return fmt.Errorf("write compose file: %w", err)
	}
	if err := r.ensureComposeOverride(app, composeFile); err != nil {
		return fmt.Errorf("write compose override: %w", err)
	}

	args := r.composeCommandArgs(app, "up", "-d")
	_, err := r.run(ctx, "docker", args...)
	if err != nil {
		return fmt.Errorf("deploy app %s: %w", app.ID, err)
	}

	return nil
}

func (r *Repository) Stop(ctx context.Context, app *domain.App) error {
	if app == nil {
		return domain.ErrAppNotFound
	}

	composeFile := r.composeFile(app)
	if err := r.ensureComposeSource(app, composeFile); err != nil {
		return fmt.Errorf("prepare compose file: %w", err)
	}

	args := r.composeCommandArgs(app, "down")
	_, err := r.run(ctx, "docker", args...)
	if err != nil {
		return fmt.Errorf("stop app %s: %w", app.ID, err)
	}

	return nil
}

func (r *Repository) Destroy(ctx context.Context, app *domain.App) error {
	if app == nil {
		return domain.ErrAppNotFound
	}

	composeFile := r.composeFile(app)
	if err := r.ensureComposeSource(app, composeFile); err != nil {
		return fmt.Errorf("prepare compose file: %w", err)
	}

	args := r.composeCommandArgs(app, "down", "--volumes", "--remove-orphans", "--timeout", "30")
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

	composeFile := r.composeFile(app)
	if err := r.ensureComposeSource(app, composeFile); err != nil {
		return fmt.Errorf("prepare compose file: %w", err)
	}

	args := r.composeCommandArgs(app, "restart")
	_, err := r.run(ctx, "docker", args...)
	if err != nil {
		return fmt.Errorf("restart app %s: %w", app.ID, err)
	}

	return nil
}

func (r *Repository) GetStatus(ctx context.Context, app *domain.App) (string, error) {
	if app == nil {
		return "", domain.ErrAppNotFound
	}

	composeFile := r.composeFile(app)
	if err := r.ensureComposeSource(app, composeFile); err != nil {
		return "", err
	}
	if _, err := os.Stat(composeFile); err != nil {
		if os.IsNotExist(err) {
			return domain.AppStatusStopped, nil
		}
		return "", err
	}

	output, err := r.run(ctx, "docker", r.composeCommandArgs(app, "ps", "--format", "json")...)
	if err != nil {
		return "", fmt.Errorf("get status for app %s: %w", app.ID, err)
	}

	return parseComposeStatus(output), nil
}

func (r *Repository) GetLogs(ctx context.Context, app *domain.App, lines int) (string, error) {
	if app == nil {
		return "", domain.ErrAppNotFound
	}

	if err := r.ensureComposeSource(app, r.composeFile(app)); err != nil {
		return "", fmt.Errorf("prepare compose file: %w", err)
	}

	output, err := r.run(ctx, "docker", r.composeCommandArgs(app, "logs", "--tail", strconv.Itoa(lines), "--no-color")...)
	if err != nil {
		return "", fmt.Errorf("get logs for app %s: %w", app.ID, err)
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

	items, err := r.inspectRaw(ctx, ids)
	if err != nil {
		return nil, err
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

func (r *Repository) inspectRaw(ctx context.Context, ids []string) ([]dockerInspect, error) {
	args := make([]string, 0, len(ids)+1)
	args = append(args, "inspect")
	args = append(args, ids...)

	output, err := r.run(ctx, "docker", args...)
	if err != nil {
		return nil, fmt.Errorf("inspect containers: %w", err)
	}

	output = bytes.TrimSpace(output)
	if len(output) == 0 {
		return []dockerInspect{}, nil
	}

	var items []dockerInspect
	if err := json.Unmarshal(output, &items); err != nil {
		return nil, fmt.Errorf("decode docker inspect: %w", err)
	}
	return items, nil
}

func (r *Repository) composeFile(app *domain.App) string {
	if app == nil {
		return filepath.Join(r.stacksDir, "unknown", "docker-compose.yml")
	}

	appID := strings.TrimSpace(app.ID)
	appDir := strings.TrimSpace(app.Dir)
	if appDir == "" {
		appDir = filepath.Join(r.stacksDir, appID)
	}
	if composePath := strings.TrimSpace(app.ComposePath); composePath != "" {
		return filepath.Join(appDir, filepath.Clean(composePath))
	}

	return filepath.Join(appDir, "docker-compose.yml")
}

func (r *Repository) composeOverrideFile(app *domain.App) string {
	return filepath.Join(filepath.Dir(r.composeFile(app)), composeOverrideFileName)
}

func (r *Repository) ensureComposeSource(app *domain.App, composeFile string) error {
	if app == nil {
		return domain.ErrAppNotFound
	}

	if _, err := os.Stat(composeFile); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	composeContent := strings.TrimSpace(app.ComposeYAML)
	if composeContent == "" {
		return domain.ErrInvalidComposeYAML
	}

	if err := os.MkdirAll(filepath.Dir(composeFile), 0o755); err != nil {
		return fmt.Errorf("create app directory: %w", err)
	}
	if err := os.WriteFile(composeFile, []byte(app.ComposeYAML), 0o644); err != nil {
		return fmt.Errorf("write compose file: %w", err)
	}
	return nil
}

func (r *Repository) ensureComposeOverride(app *domain.App, composeFile string) error {
	if app == nil {
		return domain.ErrAppNotFound
	}

	serviceNames, err := serviceNamesFromCompose(app.ComposeYAML)
	if err != nil {
		return err
	}
	overrideFile := filepath.Join(filepath.Dir(composeFile), composeOverrideFileName)
	if err := os.MkdirAll(filepath.Dir(overrideFile), 0o755); err != nil {
		return fmt.Errorf("create override directory: %w", err)
	}
	if err := os.WriteFile(overrideFile, []byte(renderComposeOverride(serviceNames, r.networkName)), 0o644); err != nil {
		return fmt.Errorf("write override file: %w", err)
	}
	return nil
}

func (r *Repository) composeEnvFile(app *domain.App) string {
	appDir := strings.TrimSpace(app.Dir)
	if appDir == "" {
		appID := strings.TrimSpace(app.ID)
		appDir = filepath.Join(r.stacksDir, appID)
	}
	return filepath.Join(appDir, managedEnvFileName)
}

func (r *Repository) composeCommandArgs(app *domain.App, command string, extra ...string) []string {
	args := []string{"compose"}
	composeFile := r.composeFile(app)
	if composeFile != "" {
		args = append(args, "-f", composeFile)
	}
	overrideFile := r.composeOverrideFile(app)
	if _, err := os.Stat(overrideFile); err == nil {
		args = append(args, "-f", overrideFile)
	}
	envFile := r.composeEnvFile(app)
	if _, err := os.Stat(envFile); err == nil {
		args = append(args, "--env-file", envFile)
	}
	args = append(args, "--ansi", "never")
	projectID := strings.TrimSpace(app.ID)
	if projectID != "" {
		args = append(args, "-p", projectID)
	}
	args = append(args, command)
	args = append(args, extra...)
	return args
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
	ID              string               `json:"Id"`
	Name            string               `json:"Name"`
	Image           string               `json:"Image"`
	Config          dockerInspectConfig  `json:"Config"`
	State           dockerInspectState   `json:"State"`
	Mounts          []dockerInspectMount `json:"Mounts"`
	NetworkSettings dockerInspectNetwork `json:"NetworkSettings"`
}

type dockerInspectConfig struct {
	Image        string            `json:"Image"`
	Labels       map[string]string `json:"Labels"`
	Env          []string          `json:"Env"`
	ExposedPorts map[string]any    `json:"ExposedPorts"`
}

type dockerInspectState struct {
	Status string `json:"Status"`
}

type dockerInspectMount struct {
	Destination string `json:"Destination"`
	Target      string `json:"Target"`
}

type dockerInspectNetwork struct {
	Ports    map[string][]dockerInspectPortBinding `json:"Ports"`
	Networks map[string]dockerInspectEndpoint      `json:"Networks"`
}

type dockerInspectEndpoint struct {
	IPAddress string `json:"IPAddress"`
}

type dockerInspectPortBinding struct {
	HostIP   string `json:"HostIP"`
	HostPort string `json:"HostPort"`
}

func serviceNamesFromCompose(raw string) ([]string, error) {
	lines := strings.Split(raw, "\n")
	inServices := false
	servicesIndent := -1
	serviceIndent := -1
	names := make([]string, 0)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " "))
		if !inServices {
			if indent == 0 && trimmed == "services:" {
				inServices = true
				servicesIndent = indent
			}
			continue
		}

		if indent <= servicesIndent {
			break
		}
		if strings.HasPrefix(trimmed, "-") || !strings.HasSuffix(trimmed, ":") {
			continue
		}

		name := strings.TrimSpace(strings.TrimSuffix(trimmed, ":"))
		if name == "" {
			continue
		}
		if strings.Contains(name, " ") {
			continue
		}

		if serviceIndent == -1 {
			serviceIndent = indent
		}
		if indent != serviceIndent {
			continue
		}

		names = append(names, name)
	}

	if !inServices || len(names) == 0 {
		return nil, domain.ErrComposeNoServices
	}

	sort.Strings(names)
	return names, nil
}

func renderComposeOverride(serviceNames []string, networkName string) string {
	var builder strings.Builder
	builder.WriteString("networks:\n")
	builder.WriteString("  " + networkName + ":\n")
	builder.WriteString("    external: true\n")
	builder.WriteString("services:\n")
	for _, serviceName := range serviceNames {
		builder.WriteString("  " + serviceName + ":\n")
		builder.WriteString("    networks:\n")
		builder.WriteString("      - default\n")
		builder.WriteString("      - " + networkName + "\n")
	}
	return builder.String()
}

func pickProxyContainerIP(items []dockerInspect, networkName string, targetPort int) string {
	if len(items) == 0 {
		return ""
	}

	sort.SliceStable(items, func(i, j int) bool {
		left := strings.TrimSpace(items[i].Config.Labels["com.docker.compose.service"])
		right := strings.TrimSpace(items[j].Config.Labels["com.docker.compose.service"])
		if left == right {
			return strings.TrimSpace(items[i].Name) < strings.TrimSpace(items[j].Name)
		}
		return left < right
	})

	fallback := ""
	for _, item := range items {
		if strings.TrimSpace(item.State.Status) != "running" {
			continue
		}
		endpoint, ok := item.NetworkSettings.Networks[networkName]
		if !ok {
			continue
		}
		ip := strings.TrimSpace(endpoint.IPAddress)
		if ip == "" {
			continue
		}
		if fallback == "" {
			fallback = ip
		}
		if containerMatchesTargetPort(item, targetPort) {
			return ip
		}
	}

	return fallback
}

func containerMatchesTargetPort(item dockerInspect, targetPort int) bool {
	if targetPort <= 0 {
		return true
	}
	portKeys := []string{
		fmt.Sprintf("%d/tcp", targetPort),
		fmt.Sprintf("%d/udp", targetPort),
	}
	for _, key := range portKeys {
		if _, ok := item.NetworkSettings.Ports[key]; ok {
			return true
		}
		if _, ok := item.Config.ExposedPorts[key]; ok {
			return true
		}
	}
	return false
}

var _ domain.DockerRepository = (*Repository)(nil)
