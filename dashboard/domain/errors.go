package domain

import "errors"

var (
	ErrContainerNotFound      = errors.New("container not found")
	ErrInvalidContainerStatus = errors.New("invalid container status")
	ErrMissingRepository     = errors.New("missing dashboard repository")
)
