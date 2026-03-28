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
	if strings.TrimSpace(stored.CertbotEmail) == "" {
		stored.CertbotEmail = strings.TrimSpace(fallback.CertbotEmail)
	}
	if !stored.CertbotEnabled {
		stored.CertbotEnabled = fallback.CertbotEnabled
	}
	if !stored.CertbotStaging {
		stored.CertbotStaging = fallback.CertbotStaging
	}
	if !stored.CertbotAutoRenew {
		stored.CertbotAutoRenew = fallback.CertbotAutoRenew
	}
	if !stored.CertbotTermsAccepted {
		stored.CertbotTermsAccepted = fallback.CertbotTermsAccepted
	}
	return stored
}

var _ domain.PlatformSettingsUseCase = (*Service)(nil)
