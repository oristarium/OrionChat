package avatar

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/oristarium/orionchat/types"
	"go.etcd.io/bbolt"
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
func (s *Storage) SaveAvatar(avatar types.Avatar) error {
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
func (s *Storage) GetAvatar(id string) (types.Avatar, error) {
	var avatar types.Avatar
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
func (s *Storage) ListAvatars() ([]types.Avatar, error) {
	var avatars []types.Avatar
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(AvatarBucket))
		if b == nil {
			return fmt.Errorf("bucket not found")
		}

		return b.ForEach(func(k, v []byte) error {
			if string(k) == ConfigKey {
				return nil // Skip config
			}

			// Skip if value appears to be a file path (starts with /)
			if len(v) > 0 && v[0] == '/' {
				return nil
			}

			var avatar types.Avatar
			if err := json.Unmarshal(v, &avatar); err != nil {
				log.Printf("Error unmarshaling avatar data %q: %v", string(v), err)
				return fmt.Errorf("unmarshal avatar: %w", err)
			}
			avatars = append(avatars, avatar)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	// Sort avatars by sort_order
	sort.Slice(avatars, func(i, j int) bool {
		// If sort_order is the same, sort by creation time
		if avatars[i].SortOrder == avatars[j].SortOrder {
			return avatars[i].CreatedAt < avatars[j].CreatedAt
		}
		return avatars[i].SortOrder < avatars[j].SortOrder
	})

	return avatars, nil
}

// SaveConfig saves avatar configuration
func (s *Storage) SaveConfig(config types.AvatarList) error {
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
func (s *Storage) GetConfig() (types.AvatarList, error) {
	var config types.AvatarList
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

// SaveAvatarImage records a new avatar image in the database
func (s *Storage) SaveAvatarImage(image types.AvatarImage) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(ImagesBucket))
		if err != nil {
			return fmt.Errorf("create images bucket: %w", err)
		}

		// Use path as key
		data, err := json.Marshal(image)
		if err != nil {
			return fmt.Errorf("marshal image: %w", err)
		}

		return b.Put([]byte(image.Path), data)
	})
}

// GetAvatarImage retrieves an avatar image by path
func (s *Storage) GetAvatarImage(path string) (types.AvatarImage, error) {
	var image types.AvatarImage
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(ImagesBucket))
		if b == nil {
			return fmt.Errorf("images bucket not found")
		}

		data := b.Get([]byte(path))
		if data == nil {
			return fmt.Errorf("image not found")
		}

		return json.Unmarshal(data, &image)
	})
	return image, err
}

// ListAvatarImages returns all available avatar images
func (s *Storage) ListAvatarImages() ([]types.AvatarImage, error) {
	var images []types.AvatarImage
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(ImagesBucket))
		if b == nil {
			return nil // No images yet
		}

		return b.ForEach(func(k, v []byte) error {
			var image types.AvatarImage
			if err := json.Unmarshal(v, &image); err != nil {
				return fmt.Errorf("unmarshal image: %w", err)
			}
			images = append(images, image)
			return nil
		})
	})
	return images, err
}

// DeleteAvatarImage removes an avatar image from the database
func (s *Storage) DeleteAvatarImage(path string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(ImagesBucket))
		if b == nil {
			return fmt.Errorf("images bucket not found")
		}

		return b.Delete([]byte(path))
	})
}

// DeleteAvatar removes an avatar from storage
func (s *Storage) DeleteAvatar(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(AvatarBucket))
		if b == nil {
			return fmt.Errorf("bucket not found")
		}
		
		return b.Delete([]byte(id))
	})
} 