package app

import (
	"context"
	"crypto/rand"
	"dashboard/domain"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var portLinePattern = regexp.MustCompile(`(?m)^\s*-\s*["']?([0-9]+(?::[0-9]+)+)["']?\s*$`)

// Service implements domain.AppUseCase.
type Service struct {
	repository       domain.AppRepository
	dockerRepository domain.DockerRepository
	stacksDir        string
}

// NewAppService creates a new app service with dependency injection.
func NewAppService(repository domain.AppRepository, dockerRepository domain.DockerRepository, stacksDir string) *Service {
	return &Service{
		repository:       repository,
		dockerRepository: dockerRepository,
		stacksDir:        stacksDir,
	}
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
	app.Ports = extractPorts(composeYAML)
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

func (s *Service) GetAppLogs(ctx context.Context, id string, lines int) (string, error) {
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

	return s.dockerRepository.GetLogs(ctx, app.ID, normalizeLogLines(lines))
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

func validateAppInput(name, composeYAML string) error {
	if strings.TrimSpace(name) == "" {
		return domain.ErrInvalidAppName
	}
	if strings.TrimSpace(composeYAML) == "" {
		return domain.ErrInvalidComposeYAML
	}
	if !strings.Contains(composeYAML, "services:") {
		return domain.ErrInvalidComposeYAML
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
