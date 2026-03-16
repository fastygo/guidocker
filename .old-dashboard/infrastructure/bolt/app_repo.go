package bolt

import (
	"context"
	"dashboard/domain"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	bbolt "go.etcd.io/bbolt"
)

const appsBucket = "apps"

// AppRepository stores apps in BoltDB.
type AppRepository struct {
	db *bbolt.DB
}

// NewAppRepository creates a Bolt-backed app repository.
func NewAppRepository(dbFile string) (*AppRepository, error) {
	if err := os.MkdirAll(filepath.Dir(dbFile), 0o755); err != nil {
		return nil, fmt.Errorf("create bolt directory: %w", err)
	}

	db, err := bbolt.Open(dbFile, 0o600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, fmt.Errorf("open bolt database: %w", err)
	}

	repo := &AppRepository{db: db}
	if err := repo.ensureBucket(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return repo, nil
}

// Close closes the underlying BoltDB handle.
func (r *AppRepository) Close() error {
	if r == nil || r.db == nil {
		return nil
	}

	return r.db.Close()
}

func (r *AppRepository) Create(ctx context.Context, app *domain.App) error {
	return r.save(ctx, app, true)
}

func (r *AppRepository) Update(ctx context.Context, app *domain.App) error {
	return r.save(ctx, app, false)
}

func (r *AppRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	return r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(appsBucket))
		if bucket.Get([]byte(id)) == nil {
			return domain.ErrAppNotFound
		}
		return bucket.Delete([]byte(id))
	})
}

func (r *AppRepository) GetByID(ctx context.Context, id string) (*domain.App, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var app *domain.App
	err := r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(appsBucket))
		value := bucket.Get([]byte(id))
		if value == nil {
			return domain.ErrAppNotFound
		}

		stored := &domain.App{}
		if err := json.Unmarshal(value, stored); err != nil {
			return fmt.Errorf("decode app: %w", err)
		}
		app = stored
		return nil
	})
	if err != nil {
		return nil, err
	}

	return app, nil
}

func (r *AppRepository) List(ctx context.Context) ([]*domain.App, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	apps := make([]*domain.App, 0)
	err := r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(appsBucket))
		return bucket.ForEach(func(_, value []byte) error {
			stored := &domain.App{}
			if err := json.Unmarshal(value, stored); err != nil {
				return fmt.Errorf("decode app: %w", err)
			}
			apps = append(apps, stored)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(apps, func(i, j int) bool {
		return apps[i].CreatedAt.After(apps[j].CreatedAt)
	})

	return apps, nil
}

func (r *AppRepository) ensureBucket() error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(appsBucket))
		return err
	})
}

func (r *AppRepository) save(ctx context.Context, app *domain.App, createOnly bool) error {
	if app == nil {
		return domain.ErrAppNotFound
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	payload, err := json.Marshal(app)
	if err != nil {
		return fmt.Errorf("encode app: %w", err)
	}

	return r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(appsBucket))
		existing := bucket.Get([]byte(app.ID))

		if createOnly && existing != nil {
			return fmt.Errorf("app already exists: %s", app.ID)
		}
		if !createOnly && existing == nil {
			return domain.ErrAppNotFound
		}

		if err := bucket.Put([]byte(app.ID), payload); err != nil {
			return fmt.Errorf("store app: %w", err)
		}
		return nil
	})
}

var _ domain.AppRepository = (*AppRepository)(nil)
