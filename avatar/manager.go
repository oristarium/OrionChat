package avatar

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/oristarium/orionchat/types"
)

// Manager handles avatar business logic
type Manager struct {
	Storage *Storage
	config  types.AvatarList
}

// NewManager creates a new avatar manager
func NewManager(storage *Storage) (*Manager, error) {
	m := &Manager{
		Storage: storage,
		config:  types.AvatarList{},
	}

	// Load or initialize config
	config, err := storage.GetConfig()
	if err != nil {
		if err.Error() != "bucket not found" && err.Error() != "config not found" {
			return nil, fmt.Errorf("get config: %w", err)
		}
		// Initialize empty config if not found
		config = types.AvatarList{}
		if err := storage.SaveConfig(config); err != nil {
			return nil, fmt.Errorf("save initial config: %w", err)
		}
	}
	m.config = config

	// Check for default avatar
	hasDefault, err := m.hasDefaultAvatar()
	if err != nil {
		return nil, fmt.Errorf("check default avatar: %w", err)
	}

	if !hasDefault {
		log.Printf("initializing default")
		if err := m.initializeDefaultAvatar(); err != nil {
			return nil, fmt.Errorf("initialize default avatar: %w", err)
		}
	}

	return m, nil
}

// hasDefaultAvatar checks if there's already a default avatar in storage
func (m *Manager) hasDefaultAvatar() (bool, error) {
	avatars, err := m.Storage.ListAvatars()
	if err != nil {
		return false, fmt.Errorf("list avatars: %w", err)
	}

	for _, avatar := range avatars {
		if avatar.IsDefault {
			return true, nil
		}
	}

	// list avatars
	log.Printf("avatars: %+v", avatars)

	return false, nil
}

// initializeDefaultAvatar creates the default avatar
func (m *Manager) initializeDefaultAvatar() error {
	log.Printf("Creating default avatar")
	// Register default images first
	defaultImages := []struct {
		path string
		typ  string
	}{
		{fmt.Sprintf("/%s/idle.png", AvatarAssetsDir), "png"},
		{fmt.Sprintf("/%s/talking.gif", AvatarAssetsDir), "gif"},
	}

	for _, img := range defaultImages {
		if err := m.RegisterAvatarImage(img.path); err != nil {
			log.Printf("Warning: Failed to register default image %s: %v", img.path, err)
		}
	}

	// Get all existing avatars to find highest ID
	existingAvatars, err := m.Storage.ListAvatars()
	if err != nil {
		return fmt.Errorf("failed to list avatars: %w", err)
	}

	maxID := 0
	for _, a := range existingAvatars {
		if id, err := strconv.Atoi(a.ID); err == nil && id > maxID {
			maxID = id
		}
	}

	defaultAvatar := types.Avatar{
		ID:          fmt.Sprintf("%d", maxID+1),
		Name:        "Default",
		Description: "Default avatar",
		States: map[types.AvatarState]string{
			types.StateIdle:    fmt.Sprintf("/%s/idle.png", AvatarAssetsDir),
			types.StateTalking: fmt.Sprintf("/%s/talking.gif", AvatarAssetsDir),
		},
		IsDefault: true,
		CreatedAt: time.Now().Unix(),
		TTSVoices: []types.TTSVoice{
			{
				VoiceID: "id_male_darma",
				Provider: "tiktok",
			},
		},
	}

	log.Printf("Saving default avatar with ID: %s", defaultAvatar.ID)
	if err := m.Storage.SaveAvatar(defaultAvatar); err != nil {
		return fmt.Errorf("save default avatar: %w", err)
	}

	m.config.Avatars = []types.Avatar{defaultAvatar}
	// m.config.HasDefault = true
	// m.config.CurrentID = defaultAvatar.ID

	log.Printf("Saving config with default avatar")
	return m.Storage.SaveConfig(m.config)
}


// GetAvatarState returns the file path for a specific avatar state
func (m *Manager) GetAvatarState(id string, state types.AvatarState) (string, error) {
	avatar, err := m.Storage.GetAvatar(id)
	if err != nil {
		return "", fmt.Errorf("get avatar: %w", err)
	}

	path, ok := avatar.States[state]
	if !ok {
		return "", fmt.Errorf("state not found: %s", state)
	}

	return path, nil
}

// UpdateAvatarState updates a specific state for an avatar
func (m *Manager) UpdateAvatarState(id string, state types.AvatarState, imagePath string) error {
	log.Printf("UpdateAvatarState called - ID: %s, State: %s, Path: %s", id, state, imagePath)

	// Verify image exists
	if _, err := m.Storage.GetAvatarImage(imagePath); err != nil {
		log.Printf("Image not found: %v", err)
		return fmt.Errorf("image not found: %w", err)
	}

	avatar, err := m.Storage.GetAvatar(id)
	if err != nil {
		log.Printf("Failed to get avatar: %v", err)
		return fmt.Errorf("get avatar: %w", err)
	}

	log.Printf("Found avatar: %+v", avatar)

	if avatar.States == nil {
		log.Printf("Initializing States map")
		avatar.States = make(map[types.AvatarState]string)
	}

	avatar.States[state] = imagePath
	log.Printf("Updated avatar states: %+v", avatar.States)

	return m.Storage.SaveAvatar(avatar)
}

// ListAvatars returns all available avatars
func (m *Manager) ListAvatars() ([]types.Avatar, error) {
	return m.Storage.ListAvatars()
}

// DeleteAvatar removes an avatar
func (m *Manager) DeleteAvatar(id string) error {
	// Don't allow deleting the default avatar
	avatar, err := m.Storage.GetAvatar(id)
	if err != nil {
		return fmt.Errorf("get avatar: %w", err)
	}

	if avatar.IsDefault {
		return fmt.Errorf("cannot delete default avatar")
	}

	// First delete from storage
	if err := m.Storage.DeleteAvatar(id); err != nil {
		return fmt.Errorf("delete from storage: %w", err)
	}

	// Then remove from config
	var updatedAvatars []types.Avatar
	for _, a := range m.config.Avatars {
		if a.ID != id {
			updatedAvatars = append(updatedAvatars, a)
		}
	}
	m.config.Avatars = updatedAvatars

	return m.Storage.SaveConfig(m.config)
}

// ValidateAvatarState checks if a file is valid for the given state
func (m *Manager) ValidateAvatarState(filePath string, state types.AvatarState) error {
	ext := filepath.Ext(filePath)
	switch state {
	case types.StateIdle:
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
			return fmt.Errorf("idle state must be a static image (png/jpg)")
		}
	case types.StateTalking:
		if ext != ".gif" {
			return fmt.Errorf("talking state must be an animated gif")
		}
	default:
		return fmt.Errorf("unknown state: %s", state)
	}
	return nil
}

// RegisterAvatarImage adds a new avatar image to the system
func (m *Manager) RegisterAvatarImage(path string) error {
	ext := filepath.Ext(path)
	image := types.AvatarImage{
		Path:      path,
		Type:      ext[1:], // remove dot
		CreatedAt: time.Now().Unix(),
	}
	return m.Storage.SaveAvatarImage(image)
}

// DeleteAvatarImage removes an avatar image from the system
func (m *Manager) DeleteAvatarImage(path string) error {
	// First check if image is in use by any avatar
	avatars, err := m.Storage.ListAvatars()
	if err != nil {
		return fmt.Errorf("list avatars: %w", err)
	}

	for _, avatar := range avatars {
		for _, statePath := range avatar.States {
			if statePath == path {
				return fmt.Errorf("image is in use by avatar %s", avatar.ID)
			}
		}
	}

	// Delete from storage
	if err := m.Storage.DeleteAvatarImage(path); err != nil {
		return fmt.Errorf("delete from storage: %w", err)
	}

	// Delete physical file
	filePath := filepath.Join(AssetsDir, path)
	if err := os.Remove(filePath); err != nil {
		log.Printf("Warning: Failed to delete physical file: %v", err)
	}

	return nil
} 