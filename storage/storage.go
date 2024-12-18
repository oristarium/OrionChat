package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/oristarium/orionchat/avatar"
	"github.com/oristarium/orionchat/types"
	"go.etcd.io/bbolt"
)

const (
	GeneralBucket = "general"
)

// BBoltStorage implements FileStorage using BBolt
type BBoltStorage struct {
	db *bbolt.DB
}

func NewBBoltStorage(dbPath string) (*BBoltStorage, error) {
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, err
	}

	// Initialize required buckets
	err = db.Update(func(tx *bbolt.Tx) error {
		// Create avatar bucket
		if _, err := tx.CreateBucketIfNotExists([]byte(avatar.AvatarBucket)); err != nil {
			return fmt.Errorf("create avatar bucket: %w", err)
		}
		
		// Create images bucket
		if _, err := tx.CreateBucketIfNotExists([]byte(avatar.ImagesBucket)); err != nil {
			return fmt.Errorf("create images bucket: %w", err)
		}
		
		// Add general bucket
		if _, err := tx.CreateBucketIfNotExists([]byte(GeneralBucket)); err != nil {
			return fmt.Errorf("create general bucket: %w", err)
		}
		
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("initialize buckets: %w", err)
	}

	return &BBoltStorage{db: db}, nil
}

func (s *BBoltStorage) Upload(file *types.File, filename string, directory string) (string, error) {
	filepath := filepath.Join(avatar.AssetsDir, directory, filename)
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(directory, 0755); err != nil {
		return "", err
	}

	dst, err := os.Create(filepath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, *file); err != nil {
		return "", err
	}

	return fmt.Sprintf("/%s/%s", directory, filename), nil
}

func (s *BBoltStorage) Get(key string, bucket string) (string, error) {
	var value string
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		value = string(b.Get([]byte(key)))
		return nil
	})
	return value, err
}

func (s *BBoltStorage) Save(key string, value string, bucket string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}
		return b.Put([]byte(key), []byte(value))
	})
}

// GetDB returns the underlying bbolt database
func (s *BBoltStorage) GetDB() *bbolt.DB {
	return s.db
} 