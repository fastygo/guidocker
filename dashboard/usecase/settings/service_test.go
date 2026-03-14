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
		AdminHost: "127.0.0.1",
		AdminPort: 3000,
	})

	settings, err := service.GetPlatformSettings(context.Background())
	if err != nil {
		t.Fatalf("GetPlatformSettings() error = %v", err)
	}
	if settings.AdminHost != "127.0.0.1" {
		t.Fatalf("expected admin host 127.0.0.1, got %q", settings.AdminHost)
	}
	if settings.AdminPort != 3000 {
		t.Fatalf("expected admin port 3000, got %d", settings.AdminPort)
	}
}

func TestSettingsService_UpdatePlatformSettings_PersistsAndMerges(t *testing.T) {
	repo := &fakePlatformSettingsRepository{
		settings: &domain.PlatformSettings{
			AdminHost: "0.0.0.0",
			AdminPort: 3000,
		},
	}
	service := NewPlatformSettingsService(repo, domain.PlatformSettings{
		AdminHost: "127.0.0.1",
		AdminPort: 3010,
	})

	updated, err := service.UpdatePlatformSettings(context.Background(), domain.PlatformSettings{
		AdminDomain: "dashboard.local",
		AdminUseTLS: true,
	})
	if err != nil {
		t.Fatalf("UpdatePlatformSettings() error = %v", err)
	}
	if updated.AdminHost != "0.0.0.0" {
		t.Fatalf("expected existing admin host to be preserved, got %q", updated.AdminHost)
	}
	if updated.AdminPort != 3000 {
		t.Fatalf("expected existing admin port to be preserved, got %d", updated.AdminPort)
	}
	if updated.AdminDomain != "dashboard.local" {
		t.Fatalf("expected admin domain dashboard.local, got %q", updated.AdminDomain)
}
	if updated.AdminUseTLS != true {
		t.Fatalf("expected admin tls enabled")
	}
	if repo.saved == nil {
		t.Fatal("expected repository SavePlatformSettings() to be called")
	}
	if repo.saved.AdminDomain != "dashboard.local" {
		t.Fatalf("expected saved admin domain dashboard.local, got %q", repo.saved.AdminDomain)
	}
}
