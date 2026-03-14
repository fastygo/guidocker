package bolt

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"dashboard/domain"
)

func TestPlatformSettingsRepository_LoadReturnsDefaultWhenMissing(t *testing.T) {
	repo, err := NewPlatformSettingsRepository(filepath.Join(t.TempDir(), "platform.db"))
	if err != nil {
		t.Fatalf("NewPlatformSettingsRepository() error = %v", err)
	}
	defer func() { _ = repo.Close() }()

	settings, err := repo.LoadPlatformSettings(context.Background())
	if err != nil {
		t.Fatalf("LoadPlatformSettings() error = %v", err)
	}
	if settings.AdminHost != "" {
		t.Fatalf("expected empty admin host for missing settings, got %q", settings.AdminHost)
	}
	if settings.AdminPort != 0 {
		t.Fatalf("expected empty admin port for missing settings, got %d", settings.AdminPort)
	}
}

func TestPlatformSettingsRepository_CompatibleWithAppDatabase(t *testing.T) {
	dbFile := filepath.Join(t.TempDir(), "platform.db")
	appRepo, err := NewAppRepository(dbFile)
	if err != nil {
		t.Fatalf("NewAppRepository() error = %v", err)
	}
	appRepo.Close()

	settingsRepo, err := NewPlatformSettingsRepository(dbFile)
	if err != nil {
		t.Fatalf("NewPlatformSettingsRepository() error = %v", err)
	}
	defer func() { _ = settingsRepo.Close() }()

	settings, err := settingsRepo.LoadPlatformSettings(context.Background())
	if err != nil {
		t.Fatalf("LoadPlatformSettings() error = %v", err)
	}
	if settings == nil {
		t.Fatal("expected platform settings struct, got nil")
	}
	if settings.AdminHost != "" {
		t.Fatalf("expected empty admin host for legacy DB, got %q", settings.AdminHost)
	}
}

func TestPlatformSettingsRepository_SaveAndLoad(t *testing.T) {
	repo, err := NewPlatformSettingsRepository(filepath.Join(t.TempDir(), "platform.db"))
	if err != nil {
		t.Fatalf("NewPlatformSettingsRepository() error = %v", err)
	}
	defer func() { _ = repo.Close() }()

	now := time.Now().UTC().Truncate(time.Second)
	expected := &domain.PlatformSettings{
		AdminHost:   "127.0.0.1",
		AdminPort:   3000,
		AdminDomain: "example.com",
		AdminUseTLS: true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := repo.SavePlatformSettings(context.Background(), expected); err != nil {
		t.Fatalf("SavePlatformSettings() error = %v", err)
	}

	loaded, err := repo.LoadPlatformSettings(context.Background())
	if err != nil {
		t.Fatalf("LoadPlatformSettings() error = %v", err)
	}
	if loaded.AdminHost != expected.AdminHost {
		t.Fatalf("expected admin host %q, got %q", expected.AdminHost, loaded.AdminHost)
	}
	if loaded.AdminPort != expected.AdminPort {
		t.Fatalf("expected admin port %d, got %d", expected.AdminPort, loaded.AdminPort)
	}
	if loaded.AdminDomain != expected.AdminDomain {
		t.Fatalf("expected admin domain %q, got %q", expected.AdminDomain, loaded.AdminDomain)
	}
	if loaded.AdminUseTLS != expected.AdminUseTLS {
		t.Fatalf("expected admin tls %v, got %v", expected.AdminUseTLS, loaded.AdminUseTLS)
	}
	if loaded.CreatedAt.IsZero() {
		t.Fatalf("expected created at to be persisted")
	}
	if !loaded.UpdatedAt.Equal(expected.UpdatedAt) {
		t.Fatalf("expected updated at %s, got %s", expected.UpdatedAt, loaded.UpdatedAt)
	}
}
