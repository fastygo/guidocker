package bbolt

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Options mirrors the subset of bbolt options used by this project.
type Options struct {
	Timeout time.Duration
}

// DB is a lightweight file-backed key/value store with a small bbolt-compatible API.
type DB struct {
	path   string
	mode   os.FileMode
	mu     sync.RWMutex
	data   map[string]map[string][]byte
	closed bool
}

// Tx represents a read or write transaction.
type Tx struct {
	db       *DB
	writable bool
	data     map[string]map[string][]byte
}

// Bucket represents a logical key/value namespace.
type Bucket struct {
	tx   *Tx
	name string
}

// Open opens or creates a file-backed database.
func Open(path string, mode os.FileMode, _ *Options) (*DB, error) {
	db := &DB{
		path: path,
		mode: mode,
		data: map[string]map[string][]byte{},
	}

	if err := db.load(); err != nil {
		return nil, err
	}

	return db, nil
}

// Close marks the database as closed.
func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.closed = true
	return nil
}

// View executes a read-only transaction.
func (db *DB) View(fn func(*Tx) error) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return errors.New("bbolt: database is closed")
	}

	return fn(&Tx{
		db:       db,
		writable: false,
		data:     cloneData(db.data),
	})
}

// Update executes a writable transaction and persists changes on success.
func (db *DB) Update(fn func(*Tx) error) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return errors.New("bbolt: database is closed")
	}

	tx := &Tx{
		db:       db,
		writable: true,
		data:     cloneData(db.data),
	}

	if err := fn(tx); err != nil {
		return err
	}

	db.data = tx.data
	return db.persist()
}

// CreateBucketIfNotExists creates a bucket in a writable transaction.
func (tx *Tx) CreateBucketIfNotExists(name []byte) (*Bucket, error) {
	if !tx.writable {
		return nil, errors.New("bbolt: transaction is read-only")
	}

	bucketName := string(name)
	if _, ok := tx.data[bucketName]; !ok {
		tx.data[bucketName] = map[string][]byte{}
	}

	return &Bucket{tx: tx, name: bucketName}, nil
}

// Bucket returns an existing bucket.
func (tx *Tx) Bucket(name []byte) *Bucket {
	bucketName := string(name)
	if _, ok := tx.data[bucketName]; !ok {
		return nil
	}

	return &Bucket{tx: tx, name: bucketName}
}

// Get returns a bucket value by key.
func (b *Bucket) Get(key []byte) []byte {
	if b == nil {
		return nil
	}

	value, ok := b.tx.data[b.name][string(key)]
	if !ok {
		return nil
	}

	return append([]byte(nil), value...)
}

// Put stores a key/value pair in a writable bucket.
func (b *Bucket) Put(key, value []byte) error {
	if b == nil {
		return errors.New("bbolt: bucket is nil")
	}
	if !b.tx.writable {
		return errors.New("bbolt: transaction is read-only")
	}

	if _, ok := b.tx.data[b.name]; !ok {
		b.tx.data[b.name] = map[string][]byte{}
	}

	b.tx.data[b.name][string(key)] = append([]byte(nil), value...)
	return nil
}

// Delete removes a key from a writable bucket.
func (b *Bucket) Delete(key []byte) error {
	if b == nil {
		return errors.New("bbolt: bucket is nil")
	}
	if !b.tx.writable {
		return errors.New("bbolt: transaction is read-only")
	}

	delete(b.tx.data[b.name], string(key))
	return nil
}

// ForEach iterates over all key/value pairs in a bucket.
func (b *Bucket) ForEach(fn func(k, v []byte) error) error {
	if b == nil {
		return nil
	}

	for key, value := range b.tx.data[b.name] {
		if err := fn([]byte(key), append([]byte(nil), value...)); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) load() error {
	if err := os.MkdirAll(filepath.Dir(db.path), 0o755); err != nil {
		return err
	}

	payload, err := os.ReadFile(db.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(payload) == 0 {
		return nil
	}

	return json.Unmarshal(payload, &db.data)
}

func (db *DB) persist() error {
	if err := os.MkdirAll(filepath.Dir(db.path), 0o755); err != nil {
		return err
	}

	payload, err := json.Marshal(db.data)
	if err != nil {
		return err
	}

	tempFile := db.path + ".tmp"
	if err := os.WriteFile(tempFile, payload, db.mode); err != nil {
		return err
	}

	return os.Rename(tempFile, db.path)
}

func cloneData(source map[string]map[string][]byte) map[string]map[string][]byte {
	cloned := make(map[string]map[string][]byte, len(source))
	for bucket, items := range source {
		bucketCopy := make(map[string][]byte, len(items))
		for key, value := range items {
			bucketCopy[key] = append([]byte(nil), value...)
		}
		cloned[bucket] = bucketCopy
	}

	return cloned
}
