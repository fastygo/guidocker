package domain

import "errors"

var (
	ErrContainerNotFound       = errors.New("container not found")
	ErrInvalidContainerStatus  = errors.New("invalid container status")
	ErrMissingRepository       = errors.New("missing dashboard repository")
	ErrAppNotFound             = errors.New("app not found")
	ErrInvalidAppName          = errors.New("invalid app name")
	ErrComposeNoServices       = errors.New("compose yaml must contain a 'services:' key")
	ErrInvalidComposeYAML      = errors.New("invalid compose yaml")
	ErrMissingAppRepository    = errors.New("missing app repository")
	ErrMissingDockerRepository = errors.New("missing docker repository")
)
