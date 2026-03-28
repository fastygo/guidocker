package settings

import (
	"context"
	"dashboard/domain"
	"testing"
)

type fakePlatformSettingsRepository struct {
	settings *domain.PlatformSettings
	loadErr  error
	saved    *domain.PlatformSettings
	saveErr  error
}

func (r *fakePlatformSettingsRepository) LoadPlatformSettings(_ context.Context) (*domain.PlatformSettings, error) {
	if r.loadErr != nil {
		return nil, r.loadErr
	}
	return r.settings, nil
}

func (r *fakePlatformSettingsRepository) SavePlatformSettings(_ context.Context, settings *domain.PlatformSettings) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	if settings != nil {
		saved := *settings
		r.saved = &saved
	}
	return nil
}

func TestSettingsService_GetPlatformSettings_AppliesFallbackDefaults(t *testing.T) {
	repo := &fakePlatformSettingsRepository{}
	service := NewPlatformSettingsService(repo, domain.PlatformSettings{
		CertbotEmail:        "ops@example.com",
		CertbotEnabled:      true,
		CertbotAutoRenew:    true,
		CertbotTermsAccepted: true,
	})

	settings, err := service.GetPlatformSettings(context.Background())
	if err != nil {
		t.Fatalf("GetPlatformSettings() error = %v", err)
	}
	if settings.CertbotEmail != "ops@example.com" {
		t.Fatalf("expected fallback certbot email, got %q", settings.CertbotEmail)
	}
	if !settings.CertbotEnabled || !settings.CertbotAutoRenew || !settings.CertbotTermsAccepted {
		t.Fatalf("expected fallback certbot flags to be applied: %+v", settings)
	}
}

func TestSettingsService_UpdatePlatformSettings_PersistsAndMerges(t *testing.T) {
	repo := &fakePlatformSettingsRepository{
		settings: &domain.PlatformSettings{
			CertbotEmail: "ops@example.com",
		},
	}
	service := NewPlatformSettingsService(repo, domain.PlatformSettings{})

	updated, err := service.UpdatePlatformSettings(context.Background(), domain.PlatformSettings{
		CertbotEnabled:       true,
		CertbotStaging:       true,
		CertbotAutoRenew:     true,
		CertbotTermsAccepted: true,
	})
	if err != nil {
		t.Fatalf("UpdatePlatformSettings() error = %v", err)
	}
	if updated.CertbotEmail != "ops@example.com" {
		t.Fatalf("expected existing certbot email to be preserved, got %q", updated.CertbotEmail)
	}
	if !updated.CertbotEnabled || !updated.CertbotStaging || !updated.CertbotAutoRenew || !updated.CertbotTermsAccepted {
		t.Fatalf("expected certbot flags enabled: %+v", updated)
	}
	if repo.saved == nil {
		t.Fatal("expected repository SavePlatformSettings() to be called")
	}
	if repo.saved.CertbotEmail != "ops@example.com" {
		t.Fatalf("expected saved certbot email ops@example.com, got %q", repo.saved.CertbotEmail)
	}
}

func TestSettingsService_UpdatePlatformSettings_UpdatesCertbotFlags(t *testing.T) {
	repo := &fakePlatformSettingsRepository{
		settings: &domain.PlatformSettings{
			CertbotEmail:         "ops@example.com",
			CertbotEnabled:       true,
			CertbotStaging:       true,
			CertbotAutoRenew:     true,
			CertbotTermsAccepted: true,
		},
	}
	service := NewPlatformSettingsService(repo, domain.PlatformSettings{})

	updated, err := service.UpdatePlatformSettings(context.Background(), domain.PlatformSettings{
		CertbotEmail:         "ops2@example.com",
		CertbotEnabled:       false,
		CertbotStaging:       false,
		CertbotAutoRenew:     false,
		CertbotTermsAccepted: false,
	})
	if err != nil {
		t.Fatalf("UpdatePlatformSettings() error = %v", err)
	}
	if updated.CertbotEmail != "ops2@example.com" {
		t.Fatalf("expected certbot email from payload, got %q", updated.CertbotEmail)
	}
	if updated.CertbotEnabled || updated.CertbotStaging || updated.CertbotAutoRenew || updated.CertbotTermsAccepted {
		t.Fatalf("expected payload certbot flags applied: %+v", updated)
	}
}
