package avatar

import (
	"encoding/json"
	"fmt"
	"time"

	"go.etcd.io/bbolt"
)

const (
	AvatarBucket = "avatars"
	ConfigKey    = "config"
)

// Storage handles avatar data persistence
type Storage struct {
	db *bbolt.DB
}

// NewStorage creates a new avatar storage instance
func NewStorage(db *bbolt.DB) *Storage {
	return &Storage{db: db}
}

// SaveAvatar saves or updates an avatar
func (s *Storage) SaveAvatar(avatar Avatar) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(AvatarBucket))
		if err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}

		// If new avatar, generate ID and set creation time
		if avatar.ID == "" {
			avatar.ID = fmt.Sprintf("avatar_%d", time.Now().UnixNano())
			avatar.CreatedAt = time.Now().Unix()
		}

		// Serialize avatar
		data, err := json.Marshal(avatar)
		if err != nil {
			return fmt.Errorf("marshal avatar: %w", err)
		}

		return b.Put([]byte(avatar.ID), data)
	})
}

// GetAvatar retrieves an avatar by ID
func (s *Storage) GetAvatar(id string) (Avatar, error) {
	var avatar Avatar
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(AvatarBucket))
		if b == nil {
			return fmt.Errorf("bucket not found")
		}

		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("avatar not found")
		}

		return json.Unmarshal(data, &avatar)
	})
	return avatar, err
}

// ListAvatars returns all avatars
func (s *Storage) ListAvatars() ([]Avatar, error) {
	var avatars []Avatar
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(AvatarBucket))
		if b == nil {
			return nil // No avatars yet
		}

		return b.ForEach(func(k, v []byte) error {
			if string(k) == ConfigKey {
				return nil // Skip config
			}

			var avatar Avatar
			if err := json.Unmarshal(v, &avatar); err != nil {
				return fmt.Errorf("unmarshal avatar: %w", err)
			}
			avatars = append(avatars, avatar)
			return nil
		})
	})
	return avatars, err
}

// SaveConfig saves avatar configuration
func (s *Storage) SaveConfig(config AvatarList) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(AvatarBucket))
		if err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}

		data, err := json.Marshal(config)
		if err != nil {
			return fmt.Errorf("marshal config: %w", err)
		}

		return b.Put([]byte(ConfigKey), data)
	})
}

// GetConfig retrieves avatar configuration
func (s *Storage) GetConfig() (AvatarList, error) {
	var config AvatarList
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(AvatarBucket))
		if b == nil {
			return fmt.Errorf("bucket not found")
		}

		data := b.Get([]byte(ConfigKey))
		if data == nil {
			return fmt.Errorf("config not found")
		}

		return json.Unmarshal(data, &config)
	})
	return config, err
} 