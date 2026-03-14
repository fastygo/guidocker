package settings

import (
	"context"
	"dashboard/domain"
	"fmt"
	"strings"
	"time"
)

// Service implements domain.PlatformSettingsUseCase.
type Service struct {
	repository domain.PlatformSettingsRepository
	fallback   domain.PlatformSettings
}

// NewPlatformSettingsService creates a service with default fallback values.
func NewPlatformSettingsService(repository domain.PlatformSettingsRepository, fallback domain.PlatformSettings) *Service {
	if fallback.AdminPort == 0 {
		fallback.AdminPort = 3000
	}
	if strings.TrimSpace(fallback.AdminHost) == "" {
		fallback.AdminHost = "0.0.0.0"
	}

	return &Service{
		repository: repository,
		fallback:   fallback,
	}
}

// GetPlatformSettings loads platform settings with safe defaults.
func (s *Service) GetPlatformSettings(ctx context.Context) (*domain.PlatformSettings, error) {
	if s == nil || s.repository == nil {
		return nil, domain.ErrMissingPlatformSettingsRepository
	}

	stored, err := s.repository.LoadPlatformSettings(ctx)
	if err != nil {
		return nil, err
	}
	if stored == nil {
		stored = &domain.PlatformSettings{}
	}

	merged := mergePlatformSettings(*stored, s.fallback)
	return &merged, nil
}

// UpdatePlatformSettings saves platform settings and returns the merged result.
func (s *Service) UpdatePlatformSettings(ctx context.Context, settings domain.PlatformSettings) (*domain.PlatformSettings, error) {
	if s == nil || s.repository == nil {
		return nil, domain.ErrMissingPlatformSettingsRepository
	}

	now := time.Now().UTC()
	current, err := s.GetPlatformSettings(ctx)
	if err != nil {
		return nil, err
	}
	if current == nil {
		current = &domain.PlatformSettings{}
	}

	settings.AdminHost = strings.TrimSpace(settings.AdminHost)
	if settings.AdminHost == "" {
		settings.AdminHost = current.AdminHost
		if settings.AdminHost == "" {
			settings.AdminHost = s.fallback.AdminHost
		}
	}

	if settings.AdminPort <= 0 {
		settings.AdminPort = current.AdminPort
		if settings.AdminPort <= 0 {
			settings.AdminPort = s.fallback.AdminPort
		}
	}

	if strings.TrimSpace(settings.AdminDomain) == "" {
		settings.AdminDomain = current.AdminDomain
	}
	settings.CertbotEmail = strings.TrimSpace(settings.CertbotEmail)
	if settings.CertbotEmail == "" {
		settings.CertbotEmail = current.CertbotEmail
	}

	if current.CreatedAt.IsZero() {
		settings.CreatedAt = now
	} else {
		settings.CreatedAt = current.CreatedAt
	}
	settings.UpdatedAt = now

	if err := s.repository.SavePlatformSettings(ctx, &settings); err != nil {
		return nil, fmt.Errorf("save platform settings: %w", err)
	}

	return &settings, nil
}

func mergePlatformSettings(stored, fallback domain.PlatformSettings) domain.PlatformSettings {
	if strings.TrimSpace(stored.AdminHost) == "" {
		stored.AdminHost = fallback.AdminHost
	}
	if stored.AdminPort == 0 {
		stored.AdminPort = fallback.AdminPort
	}
	return stored
}

var _ domain.PlatformSettingsUseCase = (*Service)(nil)
