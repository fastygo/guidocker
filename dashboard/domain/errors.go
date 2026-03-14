package domain

import "errors"

var (
	ErrContainerNotFound       = errors.New("container not found")
	ErrInvalidContainerStatus  = errors.New("invalid container status")
	ErrMissingRepository       = errors.New("missing dashboard repository")
	ErrAppNotFound             = errors.New("app not found")
	ErrMissingPlatformSettingsRepository = errors.New("missing platform settings repository")
	ErrInvalidAppName          = errors.New("invalid app name")
	ErrComposeNoServices       = errors.New("compose yaml must contain a 'services:' key")
	ErrInvalidComposeYAML      = errors.New("invalid compose yaml")
	ErrMissingAppRepository    = errors.New("missing app repository")
	ErrMissingDockerRepository = errors.New("missing docker repository")
	ErrMissingGitRepository    = errors.New("missing git repository")
	ErrInvalidRepoURL          = errors.New("invalid repository URL")
	ErrUnsupportedRepoURL      = errors.New("unsupported repository URL")
	ErrRepoBranchNotFound      = errors.New("branch not found")
	ErrInvalidComposePath      = errors.New("invalid compose path")
	ErrMissingComposeFile      = errors.New("compose file is missing")
	ErrMissingDockerfile       = errors.New("missing Dockerfile")
	ErrInvalidAppPort          = errors.New("invalid app port")
	ErrComposeConfigValidation = errors.New("compose file validation failed")
)
