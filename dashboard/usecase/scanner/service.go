package scanner

import (
	"context"
	"dashboard/domain"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Service implements domain.ScannerUseCase for read-only Docker audits.
type Service struct {
	dockerRepository domain.DockerRepository
	appRepository    domain.AppRepository
	stacksDir        string
}

// NewScannerService creates a scanner service.
func NewScannerService(dockerRepository domain.DockerRepository, appRepository domain.AppRepository, stacksDir string) *Service {
	return &Service{
		dockerRepository: dockerRepository,
		appRepository:    appRepository,
		stacksDir:        stacksDir,
	}
}

// RunScan discovers and classifies resources at runtime, in BoltDB, and in filesystem.
func (s *Service) RunScan(ctx context.Context) (*domain.ScanReport, error) {
	if s.appRepository == nil {
		return nil, domain.ErrMissingAppRepository
	}
	if s.dockerRepository == nil {
		return nil, domain.ErrMissingDockerRepository
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	stacksDir := strings.TrimSpace(s.stacksDir)
	apps, err := s.appRepository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list managed applications: %w", err)
	}

	appByID := make(map[string]*domain.App, len(apps))
	managedDirByName := make(map[string]string)
	managedDirSet := make(map[string]bool)
	for _, app := range apps {
		if app == nil || app.ID == "" {
			continue
		}
		appByID[app.ID] = app

		dir := resolveAppDir(app, stacksDir)
		if dir == "" {
			continue
		}
		managedDirByName[app.ID] = dir
		managedDirSet[filepath.Base(dir)] = true
	}

	stacks, err := os.ReadDir(stacksDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read stacks directory %q: %w", stacksDir, err)
	}

	dirsByName := make(map[string]string, len(stacks))
	for _, entry := range stacks {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		dirsByName[name] = filepath.Join(stacksDir, name)
	}

	rawContainers, err := s.dockerRepository.ListAllContainers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list runtime containers: %w", err)
	}
	containerIDs := make([]string, 0, len(rawContainers))
	for _, container := range rawContainers {
		if strings.TrimSpace(container.ID) != "" {
			containerIDs = append(containerIDs, container.ID)
		}
	}

	details, err := s.dockerRepository.InspectContainers(ctx, containerIDs)
	if err != nil {
		return nil, fmt.Errorf("inspect runtime containers: %w", err)
	}

	matchedByApp := map[string][]domain.ContainerDetail{}
	consumed := map[string]bool{}
	for _, detail := range details {
		appID := findManagedAppMatch(detail, appByID, stacksDir)
		if appID == "" {
			continue
		}
		matchedByApp[appID] = append(matchedByApp[appID], detail)
		consumed[detail.ID] = true
	}

	resources := make([]domain.ScanResource, 0, len(apps)+len(details)+len(dirsByName))
	selfAdminName := strings.TrimSpace(os.Getenv("HOSTNAME"))
	selfAdminID := readCurrentContainerID()

	for appID, app := range appByID {
		matched := matchedByApp[appID]
		appDir := managedDirByName[appID]
		dirExists := appDir != "" && fileExists(appDir)
		confidence, kind := classifyManagedResource(app, dirExists, matched, len(dirsByName) > 0)

		name := app.Name
		if strings.TrimSpace(name) == "" {
			name = appID
		}

		reason := "Application is tracked in BoltDB with matching directory and running containers."
		if kind == domain.ResourceBrokenApp {
			reason = buildBrokenReason(dirExists, matched)
		}

		resource := domain.ScanResource{
			Kind:           kind,
			Confidence:     confidence,
			Name:           name,
			AppID:          appID,
			ComposeProject: appID,
			Dir:            appDir,
			Ports:          uniqSorted(sliceFromContainerDetails(matched, "ports")),
			Status:         aggregateStatus(matched),
			Reason:         reason,
			ContainerNames: uniqSorted(sliceFromContainerDetails(matched, "names")),
			CleanupCmds:    managedCleanupCommands(kind, matched, appDir, appID),
		}
		resources = append(resources, resource)
	}

	for _, detail := range details {
		if consumed[detail.ID] {
			continue
		}

		if isStaleAdmin(detail) {
			resource := domain.ScanResource{
				Kind:           domain.ResourceStaleAdmin,
				Confidence:     classifyStaleAdminConfidence(detail),
				Name:           detail.Name,
				AppID:          "",
				ContainerNames: []string{detail.Name},
				ComposeProject: composeProjectFromDetail(detail),
				Ports:          detail.Ports,
				Status:         detail.Status,
				Reason:         staleAdminReason(detail),
				CleanupCmds:    staleAdminCleanupCommands(detail),
				IsCurrentAdmin:  isCurrentAdmin(detail, selfAdminName, selfAdminID),
			}
			resources = append(resources, resource)
			consumed[detail.ID] = true
		}
	}

	orphanRuntimeByKey := map[string][]domain.ContainerDetail{}
	for _, detail := range details {
		if consumed[detail.ID] {
			continue
		}

		project := composeProjectFromDetail(detail)
		key := "container:" + detail.ID
		if project != "" {
			key = "project:" + project
		}
		orphanRuntimeByKey[key] = append(orphanRuntimeByKey[key], detail)
		consumed[detail.ID] = true
	}

	for key, cDetails := range orphanRuntimeByKey {
		containers := cDetails
		project := strings.TrimPrefix(key, "project:")
		if !strings.HasPrefix(key, "project:") {
			project = ""
		}

		name := containers[0].Name
		if project != "" {
			name = project
		}

		resource := domain.ScanResource{
			Kind:           domain.ResourceOrphanRuntime,
			Confidence:     classifyOrphanRuntimeConfidence(project, containers),
			Name:           name,
			AppID:          "",
			ContainerNames: uniqSorted(sliceFromContainerDetails(containers, "names")),
			ComposeProject: project,
			Ports:          uniqSorted(sliceFromContainerDetails(containers, "ports")),
			Status:         aggregateStatus(containers),
			Reason:         buildOrphanRuntimeReason(containers, project),
			CleanupCmds:    orphanRuntimeCleanupCommands(project, containers),
		}
		resources = append(resources, resource)
	}

	for dirName, dirPath := range dirsByName {
		if managedDirSet[dirName] {
			continue
		}

		resource := domain.ScanResource{
			Kind:       domain.ResourceOrphanDir,
			Confidence: domain.ConfidenceMedium,
			Name:       dirName,
			Dir:        dirPath,
			Status:     "missing",
			Reason:     fmt.Sprintf("Directory exists in stacks directory but is not tracked in BoltDB."),
			CleanupCmds: []string{
				fmt.Sprintf("rm -rf %q", dirPath),
			},
		}
		resources = append(resources, resource)
	}

	for i := range resources {
		if resources[i].Kind == domain.ResourceUnknown {
			resources[i].CleanupCmds = []string{"# unknown resource, manual review required"}
		}
	}

	sort.Slice(resources, func(i, j int) bool {
		if resources[i].Kind == resources[j].Kind {
			return resources[i].Name < resources[j].Name
		}
		return resources[i].Kind < resources[j].Kind
	})

	result := &domain.ScanReport{
		ScannedAt: time.Now().UTC(),
		Resources: resources,
		StacksDir: stacksDir,
	}

	return result, nil
}

func classifyManagedResource(app *domain.App, dirExists bool, matched []domain.ContainerDetail, hasDirs bool) (domain.Confidence, domain.ResourceKind) {
	if app == nil {
		return domain.ConfidenceLow, domain.ResourceUnknown
	}
	if len(matched) == 0 {
		if dirExists {
			return domain.ConfidenceHigh, domain.ResourceBrokenApp
		}
		if hasDirs {
			return domain.ConfidenceLow, domain.ResourceBrokenApp
		}
		return domain.ConfidenceMedium, domain.ResourceBrokenApp
	}

	if hasRunningContainer(matched) {
		return domain.ConfidenceHigh, domain.ResourceManaged
	}
	return domain.ConfidenceMedium, domain.ResourceBrokenApp
}

func buildBrokenReason(dirExists bool, containers []domain.ContainerDetail) string {
	if dirExists && len(containers) == 0 {
		return "Application exists in BoltDB and directory exists, but no runtime containers are running."
	}
	if dirExists {
		return "Application exists in BoltDB and directory exists, but containers are stopped."
	}
	if len(containers) == 0 {
		return "Application exists in BoltDB, but both runtime data and stack directory are missing."
	}
	return "Application exists in BoltDB but runtime status is inconsistent."
}

func buildOrphanRuntimeReason(containers []domain.ContainerDetail, project string) string {
	if project != "" {
		return "Compose project is not tracked in BoltDB."
	}
	return "Container is running outside managed app records."
}

func classifyOrphanRuntimeConfidence(project string, details []domain.ContainerDetail) domain.Confidence {
	if project != "" {
		return domain.ConfidenceHigh
	}
	if len(details) > 1 {
		return domain.ConfidenceMedium
	}
	return domain.ConfidenceLow
}

func managedCleanupCommands(kind domain.ResourceKind, containers []domain.ContainerDetail, dir string, appID string) []string {
	if kind == domain.ResourceManaged {
		return nil
	}
	if kind == domain.ResourceBrokenApp {
		cmds := make([]string, 0, 2)
		if strings.TrimSpace(dir) != "" {
			cmds = append(cmds, fmt.Sprintf("rm -rf %q", dir))
			cmds = append(cmds, "# Also remove BoltDB record by deleting app via API when safe")
		}
		if len(containers) > 0 {
			project := composeProjectFromDetails(containers)
			if project != "" {
				cmds = append(cmds, fmt.Sprintf("docker compose -p %q down --volumes --remove-orphans", project))
			} else {
				for _, detail := range containers {
					cmds = append(cmds, fmt.Sprintf("docker rm -f %s", strings.TrimSpace(detail.Name)))
				}
			}
		}
		if len(cmds) == 0 {
			cmds = append(cmds, fmt.Sprintf("docker ps --filter name=%q", appID))
		}
		return cmds
	}
	return nil
}

func orphanRuntimeCleanupCommands(project string, details []domain.ContainerDetail) []string {
	if project != "" {
		return []string{
			fmt.Sprintf("docker compose -p %q down --volumes --remove-orphans", project),
		}
	}

	names := make([]string, 0, len(details))
	for _, detail := range details {
		name := strings.TrimSpace(detail.Name)
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	return []string{
		fmt.Sprintf("docker rm -f %s", strings.Join(names, " ")),
	}
}

func staleAdminCleanupCommands(detail domain.ContainerDetail) []string {
	name := strings.TrimSpace(detail.Name)
	if name == "" {
		name = strings.TrimSpace(detail.ID)
	}
	return []string{
		fmt.Sprintf("docker logs --tail 100 %q", name),
		fmt.Sprintf("docker rm -f %q", name),
	}
}

func findManagedAppMatch(detail domain.ContainerDetail, appByID map[string]*domain.App, stacksDir string) string {
	project := composeProjectFromDetail(detail)
	if project != "" {
		if _, ok := appByID[project]; ok {
			return project
		}
	}

	for appID, app := range appByID {
		if matchesManagedApp(detail, app, stacksDir, appID) {
			return appID
		}
	}
	return ""
}

func matchesManagedApp(detail domain.ContainerDetail, app *domain.App, stacksDir, appID string) bool {
	if app == nil || appID == "" {
		return false
	}

	if strings.EqualFold(detail.Name, appID) || strings.Contains(strings.ToLower(detail.Name), strings.ToLower(appID)) {
		return true
	}

	dir := resolveAppDir(app, stacksDir)
	if dir != "" {
		for _, mount := range detail.Mounts {
			if strings.Contains(mount, dir) {
				return true
			}
		}
	}
	return false
}

func resolveAppDir(app *domain.App, stacksDir string) string {
	if app == nil {
		return ""
	}
	if strings.TrimSpace(app.Dir) != "" {
		return app.Dir
	}
	if strings.TrimSpace(stacksDir) == "" {
		return ""
	}
	return filepath.Join(stacksDir, app.ID)
}

func composeProjectFromDetail(detail domain.ContainerDetail) string {
	return strings.TrimSpace(detail.Labels["com.docker.compose.project"])
}

func composeProjectFromDetails(details []domain.ContainerDetail) string {
	for _, detail := range details {
		project := composeProjectFromDetail(detail)
		if project != "" {
			return project
		}
	}
	return ""
}

func hasRunningContainer(details []domain.ContainerDetail) bool {
	for _, detail := range details {
		if normalizeContainerStatus(detail.Status) == domain.ContainerStatusRunning {
			return true
		}
	}
	return false
}

func aggregateStatus(details []domain.ContainerDetail) string {
	if hasRunningContainer(details) {
		return domain.ContainerStatusRunning
	}
	if len(details) == 0 {
		return domain.ContainerStatusStopped
	}
	return normalizeContainerStatus(details[0].Status)
}

func normalizeContainerStatus(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.Contains(v, "running"):
		return domain.ContainerStatusRunning
	case strings.Contains(v, "pause"):
		return domain.ContainerStatusPaused
	case strings.Contains(v, "restart"):
		return domain.ContainerStatusRunning
	default:
		return domain.ContainerStatusStopped
	}
}

func isStaleAdmin(detail domain.ContainerDetail) bool {
	matches := 0

	image := strings.ToLower(detail.Image)
	if strings.Contains(image, "dashboard") || strings.Contains(image, "paas") {
		matches++
	}

	if hasMount(detail.Mounts, "/var/run/docker.sock") {
		matches++
	}

	if hasMount(detail.Mounts, "/opt/stacks") {
		matches++
	}

	if hasEnv(detail.Envs, "PAAS_ADMIN_USER") || hasEnv(detail.Envs, "STACKS_DIR") {
		matches++
	}

	for _, port := range detail.Ports {
		if strings.Contains(port, "7000") {
			matches++
			break
		}
	}

	if strings.Contains(strings.ToLower(detail.Name), "dashboard") ||
		strings.Contains(strings.ToLower(detail.Name), "paas") ||
		strings.Contains(strings.ToLower(detail.Name), "admin") {
		matches++
	}

	return matches >= 3
}

func classifyStaleAdminConfidence(detail domain.ContainerDetail) domain.Confidence {
	matches := 0
	image := strings.ToLower(detail.Image)
	if strings.Contains(image, "dashboard") || strings.Contains(image, "paas") {
		matches++
	}
	if hasMount(detail.Mounts, "/var/run/docker.sock") {
		matches++
	}
	if hasMount(detail.Mounts, "/opt/stacks") {
		matches++
	}
	if hasEnv(detail.Envs, "PAAS_ADMIN_USER") || hasEnv(detail.Envs, "STACKS_DIR") {
		matches++
	}
	for _, port := range detail.Ports {
		if strings.Contains(port, "7000") {
			matches++
			break
		}
	}
	if strings.Contains(strings.ToLower(detail.Name), "dashboard") ||
		strings.Contains(strings.ToLower(detail.Name), "paas") ||
		strings.Contains(strings.ToLower(detail.Name), "admin") {
		matches++
	}

	switch {
	case matches >= 4:
		return domain.ConfidenceHigh
	case matches == 3:
		return domain.ConfidenceMedium
	default:
		return domain.ConfidenceLow
	}
}

func staleAdminReason(detail domain.ContainerDetail) string {
	return "Container strongly matches dashboard admin patterns. Review before manual removal."
}

func isCurrentAdmin(detail domain.ContainerDetail, hostName, selfContainerID string) bool {
	name := strings.TrimSpace(detail.Name)
	if hostName != "" && name != "" {
		if strings.EqualFold(name, hostName) || strings.Contains(strings.ToLower(name), strings.ToLower(hostName)) {
			return true
		}
	}
	if selfContainerID != "" {
		return strings.HasPrefix(strings.TrimSpace(detail.ID), selfContainerID)
	}
	return false
}

func hasMount(mounts []string, expected string) bool {
	for _, mount := range mounts {
		if strings.Contains(strings.ToLower(mount), strings.ToLower(expected)) {
			return true
		}
	}
	return false
}

func hasEnv(envs []string, expected string) bool {
	prefix := strings.ToUpper(expected) + "="
	for _, env := range envs {
		e := strings.ToUpper(env)
		if strings.HasPrefix(e, prefix) {
			return true
		}
	}
	return false
}

func sliceFromContainerDetails(details []domain.ContainerDetail, field string) []string {
	values := make([]string, 0, len(details))
	for _, detail := range details {
		switch field {
		case "names":
			if strings.TrimSpace(detail.Name) != "" {
				values = append(values, detail.Name)
			}
		case "ports":
			for _, port := range detail.Ports {
				if strings.TrimSpace(port) != "" {
					values = append(values, port)
				}
			}
		}
	}
	return values
}

func uniqSorted(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func readCurrentContainerID() string {
	content, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return ""
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		parts := strings.Split(line, "/")
		for i := len(parts) - 1; i >= 0; i-- {
			part := strings.TrimSpace(parts[i])
			if isContainerID(part) {
				if len(part) > 12 {
					return part[:12]
				}
				return part
			}
		}
	}
	return ""
}

func isContainerID(value string) bool {
	if len(value) < 12 || len(value) > 64 {
		return false
	}
	for _, char := range strings.ToLower(value) {
		switch {
		case char >= '0' && char <= '9':
		case char >= 'a' && char <= 'f':
		default:
			return false
		}
	}
	return true
}
