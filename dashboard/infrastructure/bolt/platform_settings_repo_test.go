package bolt

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"dashboard/domain"

	bbolt "go.etcd.io/bbolt"
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
	if settings.CertbotEmail != "" {
		t.Fatalf("expected empty certbot email for missing settings, got %q", settings.CertbotEmail)
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
	if settings.CertbotEmail != "" {
		t.Fatalf("expected empty certbot email for legacy DB, got %q", settings.CertbotEmail)
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
		CertbotEmail:        "ops@example.com",
		CertbotEnabled:      true,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := repo.SavePlatformSettings(context.Background(), expected); err != nil {
		t.Fatalf("SavePlatformSettings() error = %v", err)
	}

	loaded, err := repo.LoadPlatformSettings(context.Background())
	if err != nil {
		t.Fatalf("LoadPlatformSettings() error = %v", err)
	}
	if loaded.CertbotEmail != expected.CertbotEmail {
		t.Fatalf("expected certbot email %q, got %q", expected.CertbotEmail, loaded.CertbotEmail)
	}
	if loaded.CertbotEnabled != expected.CertbotEnabled {
		t.Fatalf("expected certbot enabled %v, got %v", expected.CertbotEnabled, loaded.CertbotEnabled)
	}
	if loaded.CreatedAt.IsZero() {
		t.Fatalf("expected created at to be persisted")
	}
	if !loaded.UpdatedAt.Equal(expected.UpdatedAt) {
		t.Fatalf("expected updated at %s, got %s", expected.UpdatedAt, loaded.UpdatedAt)
	}
}

func TestPlatformSettingsRepository_SaveAndLoad_WithCertbotFields(t *testing.T) {
	repo, err := NewPlatformSettingsRepository(filepath.Join(t.TempDir(), "platform.db"))
	if err != nil {
		t.Fatalf("NewPlatformSettingsRepository() error = %v", err)
	}
	defer func() { _ = repo.Close() }()

	now := time.Now().UTC().Truncate(time.Second)
	expected := &domain.PlatformSettings{
		CertbotEmail:         "ops@example.com",
		CertbotEnabled:       true,
		CertbotStaging:       true,
		CertbotAutoRenew:     true,
		CertbotTermsAccepted: true,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := repo.SavePlatformSettings(context.Background(), expected); err != nil {
		t.Fatalf("SavePlatformSettings() error = %v", err)
	}

	loaded, err := repo.LoadPlatformSettings(context.Background())
	if err != nil {
		t.Fatalf("LoadPlatformSettings() error = %v", err)
	}
	if loaded.CertbotEmail != expected.CertbotEmail {
		t.Fatalf("expected certbot email %q, got %q", expected.CertbotEmail, loaded.CertbotEmail)
	}
	if loaded.CertbotEnabled != expected.CertbotEnabled {
		t.Fatalf("expected certbot enabled %v, got %v", expected.CertbotEnabled, loaded.CertbotEnabled)
	}
	if loaded.CertbotStaging != expected.CertbotStaging {
		t.Fatalf("expected certbot staging %v, got %v", expected.CertbotStaging, loaded.CertbotStaging)
	}
	if loaded.CertbotAutoRenew != expected.CertbotAutoRenew {
		t.Fatalf("expected certbot auto renew %v, got %v", expected.CertbotAutoRenew, loaded.CertbotAutoRenew)
	}
	if loaded.CertbotTermsAccepted != expected.CertbotTermsAccepted {
		t.Fatalf("expected certbot terms accepted %v, got %v", expected.CertbotTermsAccepted, loaded.CertbotTermsAccepted)
	}
}

func TestPlatformSettingsRepository_LoadLegacyDataWithoutCertbotFields(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "platform.db")
	repo, err := NewPlatformSettingsRepository(dbPath)
	if err != nil {
		t.Fatalf("NewPlatformSettingsRepository() error = %v", err)
	}
	defer func() { _ = repo.Close() }()

	legacy := map[string]interface{}{
		"certbot_email": "ops@example.com",
	}
	raw, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal legacy settings = %v", err)
	}

	if err := repo.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(platformSettingsBucket))
		return bucket.Put([]byte(platformSettingsRecord), raw)
	}); err != nil {
		t.Fatalf("seed legacy settings = %v", err)
	}

	loaded, err := repo.LoadPlatformSettings(context.Background())
	if err != nil {
		t.Fatalf("LoadPlatformSettings() error = %v", err)
	}
	if loaded.CertbotEmail != "ops@example.com" {
		t.Fatalf("expected certbot email ops@example.com, got %q", loaded.CertbotEmail)
	}
	if loaded.CertbotEnabled {
		t.Fatalf("expected legacy certbot enabled false")
	}
}
