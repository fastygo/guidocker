package bolt

import (
	"context"
	"dashboard/domain"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bbolt "go.etcd.io/bbolt"
)

const platformSettingsBucket = "platform_settings"
const platformSettingsRecord = "value"

// PlatformSettingsRepository stores platform-level settings in BoltDB.
type PlatformSettingsRepository struct {
	db *bbolt.DB
}

// NewPlatformSettingsRepository creates a Bolt-backed platform settings repository.
func NewPlatformSettingsRepository(dbFile string) (*PlatformSettingsRepository, error) {
	if err := os.MkdirAll(filepath.Dir(dbFile), 0o755); err != nil {
		return nil, fmt.Errorf("create platform settings directory: %w", err)
	}

	db, err := bbolt.Open(dbFile, 0o600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, fmt.Errorf("open platform settings database: %w", err)
	}

	repo := &PlatformSettingsRepository{db: db}
	if err := repo.ensureBucket(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return repo, nil
}

// Close closes the underlying BoltDB handle.
func (r *PlatformSettingsRepository) Close() error {
	if r == nil || r.db == nil {
		return nil
	}

	return r.db.Close()
}

// LoadPlatformSettings loads platform settings from BoltDB.
func (r *PlatformSettingsRepository) LoadPlatformSettings(ctx context.Context) (*domain.PlatformSettings, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("platform settings repository is not initialized")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var settings *domain.PlatformSettings
	err := r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(platformSettingsBucket))
		if bucket == nil {
			settings = &domain.PlatformSettings{}
			return nil
		}

		data := bucket.Get([]byte(platformSettingsRecord))
		if len(data) == 0 {
			settings = &domain.PlatformSettings{}
			return nil
		}

		stored := &domain.PlatformSettings{}
		if err := json.Unmarshal(data, stored); err != nil {
			return fmt.Errorf("decode platform settings: %w", err)
		}
		settings = stored
		return nil
	})
	if err != nil {
		return nil, err
	}

	return settings, nil
}

// SavePlatformSettings saves platform settings to BoltDB.
func (r *PlatformSettingsRepository) SavePlatformSettings(ctx context.Context, settings *domain.PlatformSettings) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("platform settings repository is not initialized")
	}
	if settings == nil {
		return fmt.Errorf("platform settings are required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	payload, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("encode platform settings: %w", err)
	}

	return r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(platformSettingsBucket))
		if bucket == nil {
			return fmt.Errorf("platform settings bucket not found")
		}
		return bucket.Put([]byte(platformSettingsRecord), payload)
	})
}

func (r *PlatformSettingsRepository) ensureBucket() error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(platformSettingsBucket))
		return err
	})
}

var _ domain.PlatformSettingsRepository = (*PlatformSettingsRepository)(nil)
