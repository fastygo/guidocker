package app

import (
	"context"
	"crypto/rand"
	"dashboard/domain"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var portLinePattern = regexp.MustCompile(`(?m)^\s*-\s*["']?([0-9]+(?::[0-9]+)+)["']?\s*$`)

const (
	defaultComposePath      = "docker-compose.yml"
	defaultGeneratedCompose = "docker-compose.generated.yml"
	defaultImportTimeout    = 5 * time.Minute
	defaultImportTempPath   = ".tmp"
	maxAppPort             = 65535
)

// Service implements domain.AppUseCase.
type Service struct {
	repository       domain.AppRepository
	dockerRepository domain.DockerRepository
	stacksDir        string
	gitRepository    domain.GitRepository
	importTimeout    time.Duration
	importTempPath   string
	composeValidator composeValidatorFunc
}

type composeValidatorFunc func(context.Context, string) error

// NewAppService creates a new app service with dependency injection.
func NewAppService(repository domain.AppRepository, dockerRepository domain.DockerRepository, gitRepository domain.GitRepository, stacksDir string) *Service {
	validator := func(ctx context.Context, composePath string) error {
		cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composePath, "config")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %s", domain.ErrComposeConfigValidation, strings.TrimSpace(string(output)))
		}
		return nil
	}

	return &Service{
		repository:       repository,
		dockerRepository: dockerRepository,
		gitRepository:    gitRepository,
		stacksDir:        stacksDir,
		importTimeout:    defaultImportTimeout,
		importTempPath:   defaultImportTempPath,
		composeValidator: validator,
	}
}

func (s *Service) WithImportTimeout(timeout time.Duration) *Service {
	if timeout > 0 {
		s.importTimeout = timeout
	}
	return s
}

func (s *Service) WithImportTempPath(path string) *Service {
	if strings.TrimSpace(path) != "" {
		s.importTempPath = strings.TrimSpace(path)
	}
	return s
}

func (s *Service) WithComposeValidator(validator composeValidatorFunc) *Service {
	if validator != nil {
		s.composeValidator = validator
	}
	return s
}

func (s *Service) CreateApp(ctx context.Context, name, composeYAML string) (*domain.App, error) {
	if s.repository == nil {
		return nil, domain.ErrMissingAppRepository
	}
	if err := validateAppInput(name, composeYAML); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	appID, err := newAppID(name)
	if err != nil {
		return nil, err
	}

	app := &domain.App{
		ID:          appID,
		Name:        strings.TrimSpace(name),
		ComposeYAML: strings.TrimSpace(composeYAML),
		SourceType:  domain.SourceTypeManual,
		ComposePath: defaultComposePath,
		Dir:         filepath.Join(s.stacksDir, appID),
		Status:      domain.AppStatusCreated,
		Ports:       extractPorts(composeYAML),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repository.Create(ctx, app); err != nil {
		return nil, err
	}

	return app, nil
}

func (s *Service) UpdateApp(ctx context.Context, id, name, composeYAML string) (*domain.App, error) {
	if s.repository == nil {
		return nil, domain.ErrMissingAppRepository
	}
	if err := validateAppInput(name, composeYAML); err != nil {
		return nil, err
	}

	app, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	app.Name = strings.TrimSpace(name)
	app.ComposeYAML = strings.TrimSpace(composeYAML)
	if strings.TrimSpace(app.ComposePath) == "" {
		app.ComposePath = defaultComposePath
	}
	if app.SourceType == "" {
		app.SourceType = domain.SourceTypeManual
	}
	app.Ports = extractPorts(composeYAML)
	app.UpdatedAt = time.Now().UTC()

	if err := s.repository.Update(ctx, app); err != nil {
		return nil, err
	}

	return app, nil
}

func (s *Service) UpdateAppConfig(ctx context.Context, id string, config domain.AppConfig) (*domain.App, error) {
	if s.repository == nil {
		return nil, domain.ErrMissingAppRepository
	}

	app, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	app.PublicDomain = strings.TrimSpace(config.PublicDomain)
	app.ProxyTargetPort = config.ProxyTargetPort
	app.UseTLS = config.UseTLS
	if config.ManagedEnv != nil {
		app.ManagedEnv = cloneManagedEnv(config.ManagedEnv)
	}
	app.UpdatedAt = time.Now().UTC()

	if err := s.repository.Update(ctx, app); err != nil {
		return nil, err
	}

	return app, nil
}

func (s *Service) DeleteApp(ctx context.Context, id string) error {
	if s.repository == nil {
		return domain.ErrMissingAppRepository
	}

	app, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if s.dockerRepository != nil {
		_ = s.dockerRepository.Destroy(ctx, app)
	}

	stackDir := strings.TrimSpace(app.Dir)
	if stackDir == "" {
		stackDir = filepath.Join(s.stacksDir, id)
	}
	if stackDir != "" {
		_ = os.RemoveAll(stackDir)
	}

	return s.repository.Delete(ctx, id)
}

func (s *Service) GetApp(ctx context.Context, id string) (*domain.App, error) {
	if s.repository == nil {
		return nil, domain.ErrMissingAppRepository
	}

	app, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.refreshAppStatus(ctx, app)
}

func (s *Service) ListApps(ctx context.Context) ([]*domain.App, error) {
	if s.repository == nil {
		return nil, domain.ErrMissingAppRepository
	}

	apps, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	for i := range apps {
		apps[i], _ = s.refreshAppStatus(ctx, apps[i])
	}

	return apps, nil
}

func (s *Service) DeployApp(ctx context.Context, id string) error {
	if s.repository == nil {
		return domain.ErrMissingAppRepository
	}
	if s.dockerRepository == nil {
		return domain.ErrMissingDockerRepository
	}

	app, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if strings.TrimSpace(app.ComposeYAML) == "" {
		return domain.ErrInvalidComposeYAML
	}

	app.Status = domain.AppStatusDeploying
	app.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, app); err != nil {
		return err
	}

	if err := s.dockerRepository.Deploy(ctx, app); err != nil {
		app.Status = domain.AppStatusError
		app.UpdatedAt = time.Now().UTC()
		_ = s.repository.Update(ctx, app)
		return err
	}

	status, err := s.dockerRepository.GetStatus(ctx, app)
	if err != nil {
		app.Status = domain.AppStatusRunning
		app.UpdatedAt = time.Now().UTC()
		_ = s.repository.Update(ctx, app)
		return nil
	}

	app.Status = domain.NormalizeAppStatus(status)
	app.UpdatedAt = time.Now().UTC()
	return s.repository.Update(ctx, app)
}

func (s *Service) StopApp(ctx context.Context, id string) error {
	if s.repository == nil {
		return domain.ErrMissingAppRepository
	}
	if s.dockerRepository == nil {
		return domain.ErrMissingDockerRepository
	}

	app, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.dockerRepository.Stop(ctx, app); err != nil {
		app.Status = domain.AppStatusError
		app.UpdatedAt = time.Now().UTC()
		_ = s.repository.Update(ctx, app)
		return err
	}

	app.Status = domain.AppStatusStopped
	app.UpdatedAt = time.Now().UTC()
	return s.repository.Update(ctx, app)
}

func (s *Service) RestartApp(ctx context.Context, id string) error {
	if s.repository == nil {
		return domain.ErrMissingAppRepository
	}
	if s.dockerRepository == nil {
		return domain.ErrMissingDockerRepository
	}

	app, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.dockerRepository.Restart(ctx, app); err != nil {
		app.Status = domain.AppStatusError
		app.UpdatedAt = time.Now().UTC()
		_ = s.repository.Update(ctx, app)
		return err
	}

	status, err := s.dockerRepository.GetStatus(ctx, app)
	if err != nil {
		app.Status = domain.AppStatusRunning
	} else {
		app.Status = domain.NormalizeAppStatus(status)
	}
	app.UpdatedAt = time.Now().UTC()
	return s.repository.Update(ctx, app)
}

func (s *Service) GetAppStatus(ctx context.Context, id string) (string, error) {
	if s.repository == nil {
		return "", domain.ErrMissingAppRepository
	}
	if s.dockerRepository == nil {
		return "", domain.ErrMissingDockerRepository
	}

	app, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	status, err := s.dockerRepository.GetStatus(ctx, app)
	if err != nil {
		return "", err
	}

	app.Status = domain.NormalizeAppStatus(status)
	app.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, app); err != nil {
		return "", err
	}

	return app.Status, nil
}

func (s *Service) GetAppLogs(ctx context.Context, app *domain.App, lines int) (string, error) {
	if app == nil {
		return "", domain.ErrAppNotFound
	}
	if s.repository == nil {
		return "", domain.ErrMissingAppRepository
	}
	if s.dockerRepository == nil {
		return "", domain.ErrMissingDockerRepository
	}

	return s.dockerRepository.GetLogs(ctx, app, normalizeLogLines(lines))
}

func (s *Service) ImportRepo(ctx context.Context, input domain.ImportRepoInput) (*domain.App, error) {
	if s.repository == nil {
		return nil, domain.ErrMissingAppRepository
	}
	if s.gitRepository == nil {
		return nil, domain.ErrMissingGitRepository
	}
	if err := validateRepoURL(input.RepoURL); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, domain.ErrInvalidAppName
	}

	appID, err := newAppID(name)
	if err != nil {
		return nil, err
	}

	repoURL := strings.TrimSpace(input.RepoURL)
	repoBranch := strings.TrimSpace(input.Branch)
	composePath := strings.TrimSpace(input.ComposePath)
	appPort := input.AppPort

	baseDir := strings.TrimSpace(s.stacksDir)
	if baseDir == "" {
		return nil, fmt.Errorf("stacks directory is not configured")
	}

	tmpBaseName := strings.TrimSpace(s.importTempPath)
	if tmpBaseName == "" {
		tmpBaseName = defaultImportTempPath
	}
	if filepath.IsAbs(tmpBaseName) {
		tmpBaseName = strings.TrimPrefix(tmpBaseName, string(filepath.Separator))
	}
	tmpBaseName = filepath.Clean(tmpBaseName)
	if tmpBaseName == "." || tmpBaseName == ".." || strings.HasPrefix(tmpBaseName, ".."+string(filepath.Separator)) {
		tmpBaseName = defaultImportTempPath
	}
	tmpBase := filepath.Join(baseDir, tmpBaseName)
	if err := os.MkdirAll(tmpBase, 0o755); err != nil {
		return nil, fmt.Errorf("prepare temporary import directory: %w", err)
	}

	workDir := filepath.Join(tmpBase, "import-"+appID)
	finalDir := filepath.Join(baseDir, appID)
	if _, err := os.Stat(finalDir); err == nil {
		return nil, fmt.Errorf("application directory already exists: %s", finalDir)
	}

	cleanupDir := workDir
	defer func() {
		if cleanupDir != "" {
			_ = os.RemoveAll(cleanupDir)
		}
	}()

	if err := os.RemoveAll(workDir); err != nil {
		return nil, fmt.Errorf("prepare temporary import directory: %w", err)
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return nil, fmt.Errorf("prepare temporary import directory: %w", err)
	}

	timeout := s.importTimeout
	if timeout <= 0 {
		timeout = defaultImportTimeout
	}
	cloneCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resolvedCommit, err := s.gitRepository.Clone(cloneCtx, repoURL, repoBranch, workDir)
	if err != nil {
		return nil, fmt.Errorf("clone repository: %w", err)
	}

	sourceType, resolvedComposePath, composeYAML, composeErr := s.resolveCompose(ctx, workDir, composePath, appPort)
	if composeErr != nil {
		return nil, composeErr
	}

	if err := os.Rename(workDir, finalDir); err != nil {
		return nil, fmt.Errorf("promote application directory: %w", err)
	}
	cleanupDir = finalDir

	app := &domain.App{
		ID:             appID,
		Name:           name,
		ComposeYAML:    composeYAML,
		SourceType:     sourceType,
		RepoURL:        repoURL,
		RepoBranch:     repoBranch,
		ComposePath:    resolvedComposePath,
		ResolvedCommit: resolvedCommit,
		AppPort:        appPort,
		Dir:            finalDir,
		Status:         domain.AppStatusCreated,
		Ports:          extractPorts(composeYAML),
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if err := s.repository.Create(ctx, app); err != nil {
		return nil, err
	}
	cleanupDir = ""

	if input.AutoDeploy {
		if err := s.DeployApp(ctx, app.ID); err != nil {
			return app, err
		}
	}

	return app, nil
}

func (s *Service) refreshAppStatus(ctx context.Context, app *domain.App) (*domain.App, error) {
	if app == nil {
		return nil, domain.ErrAppNotFound
	}
	if s.dockerRepository == nil {
		return app, nil
	}

	status, err := s.dockerRepository.GetStatus(ctx, app)
	if err != nil {
		return app, nil
	}

	normalized := domain.NormalizeAppStatus(status)
	if app.Status != normalized {
		app.Status = normalized
		app.UpdatedAt = time.Now().UTC()
		if s.repository != nil {
			_ = s.repository.Update(ctx, app)
		}
	}

	return app, nil
}

func (s *Service) resolveCompose(ctx context.Context, repoDir, requestedPath string, fallbackPort int) (domain.SourceType, string, string, error) {
	composePath, err := s.resolveComposePath(repoDir, requestedPath)
	if err == nil {
		composeFile := filepath.Join(repoDir, composePath)
		if s.composeValidator != nil {
			if validateErr := s.composeValidator(ctx, composeFile); validateErr != nil {
				return "", "", "", validateErr
			}
		}
		composeBytes, err := os.ReadFile(composeFile)
		if err != nil {
			return "", "", "", err
		}
		return domain.SourceTypeRepoCompose, composePath, string(composeBytes), nil
	}

	if requestedPath != "" && errors.Is(err, domain.ErrMissingComposeFile) {
		return "", "", "", err
	}
	if !errors.Is(err, domain.ErrMissingComposeFile) {
		return "", "", "", err
	}

	dockerfilePath := filepath.Join(repoDir, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); err != nil {
		return "", "", "", domain.ErrMissingDockerfile
	}

	if fallbackPort <= 0 || fallbackPort > maxAppPort {
		return "", "", "", domain.ErrInvalidAppPort
	}

	generatedCompose := generateComposeFromDockerfile(fallbackPort)
	generatedPath := filepath.Join(repoDir, defaultGeneratedCompose)
	if err := os.WriteFile(generatedPath, []byte(generatedCompose), 0o644); err != nil {
		return "", "", "", fmt.Errorf("write generated compose file: %w", err)
	}

	if s.composeValidator != nil {
		if validateErr := s.composeValidator(ctx, generatedPath); validateErr != nil {
			return "", "", "", validateErr
		}
	}

	return domain.SourceTypeRepoDockerfile, defaultGeneratedCompose, generatedCompose, nil
}

func (s *Service) resolveComposePath(repoDir, requestedPath string) (string, error) {
	requestedPath = strings.TrimSpace(requestedPath)
	if requestedPath == "" {
		for _, candidate := range []string{defaultComposePath, "compose.yml"} {
			if _, err := os.Stat(filepath.Join(repoDir, candidate)); err == nil {
				return candidate, nil
			}
		}
		return "", domain.ErrMissingComposeFile
	}

	candidate := filepath.Clean(filepath.FromSlash(requestedPath))
	if filepath.IsAbs(candidate) || strings.HasPrefix(candidate, "..") {
		return "", domain.ErrInvalidComposePath
	}

	fullPath := filepath.Join(repoDir, candidate)
	rel, err := filepath.Rel(repoDir, fullPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", domain.ErrInvalidComposePath
	}
	if rel == "." {
		return "", domain.ErrInvalidComposePath
	}
	entry, err := os.Stat(fullPath)
	if err != nil {
		return "", domain.ErrMissingComposeFile
	}
	if entry.IsDir() {
		return "", domain.ErrInvalidComposePath
	}

	return rel, nil
}

func validateRepoURL(rawURL string) error {
	cleanURL := strings.TrimSpace(rawURL)
	if cleanURL == "" {
		return domain.ErrInvalidRepoURL
	}

	parsed, err := url.Parse(cleanURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return domain.ErrInvalidRepoURL
	}
	if parsed.Scheme != "https" {
		return domain.ErrUnsupportedRepoURL
	}

	return nil
}

func generateComposeFromDockerfile(port int) string {
	return fmt.Sprintf(`services:
  web:
    build:
      context: .
    ports:
      - "%d:%d"
    restart: unless-stopped
`, port, port)
}

func validateAppInput(name, composeYAML string) error {
	if strings.TrimSpace(name) == "" {
		return domain.ErrInvalidAppName
	}
	if strings.TrimSpace(composeYAML) == "" {
		return domain.ErrInvalidComposeYAML
	}
	if !strings.Contains(composeYAML, "services:") {
		return domain.ErrComposeNoServices
	}

	return nil
}

func newAppID(name string) (string, error) {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate app id: %w", err)
	}

	slug := slugify(name)
	if slug == "" {
		slug = "app"
	}

	return fmt.Sprintf("%s-%s", slug, hex.EncodeToString(buf)), nil
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))

	var builder strings.Builder
	lastDash := false
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z':
			builder.WriteRune(char)
			lastDash = false
		case char >= '0' && char <= '9':
			builder.WriteRune(char)
			lastDash = false
		default:
			if !lastDash && builder.Len() > 0 {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}

	return strings.Trim(builder.String(), "-")
}

func extractPorts(composeYAML string) []string {
	matches := portLinePattern.FindAllStringSubmatch(composeYAML, -1)
	if len(matches) == 0 {
		return nil
	}

	ports := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		ports = append(ports, match[1])
	}

	return ports
}

func normalizeLogLines(lines int) int {
	if lines <= 0 {
		return 100
	}
	if lines > 1000 {
		return 1000
	}
	return lines
}

func cloneManagedEnv(env map[string]string) map[string]string {
	if len(env) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(env))
	for key, value := range env {
		if strings.TrimSpace(key) == "" {
			continue
		}
		cloned[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	return cloned
}
