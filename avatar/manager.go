package avatar

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Manager handles avatar business logic
type Manager struct {
	Storage *Storage
	config  AvatarList
}

// NewManager creates a new avatar manager
func NewManager(storage *Storage) (*Manager, error) {
	m := &Manager{
		Storage: storage,
		config:  AvatarList{},
	}

	// Load or initialize config
	config, err := storage.GetConfig()
	if err != nil {
		if err.Error() != "bucket not found" && err.Error() != "config not found" {
			return nil, fmt.Errorf("get config: %w", err)
		}
		// Initialize empty config if not found
		config = AvatarList{}
		if err := storage.SaveConfig(config); err != nil {
			return nil, fmt.Errorf("save initial config: %w", err)
		}
	}
	m.config = config

	// // Initialize default avatar if needed
	// if m.config.CurrentID == "" {
	// 	log.Printf("No current avatar, initializing default")
	// 	if err := m.initializeDefaultAvatar(); err != nil {
	// 		return nil, fmt.Errorf("initialize default avatar: %w", err)
	// 	}
	// }

	return m, nil
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

	defaultAvatar := Avatar{
		ID:          fmt.Sprintf("avatar_%d", time.Now().UnixNano()),
		Name:        "Default",
		Description: "Default avatar",
		States: map[AvatarState]string{
			StateIdle:    fmt.Sprintf("/%s/idle.png", AvatarAssetsDir),
			StateTalking: fmt.Sprintf("/%s/talking.gif", AvatarAssetsDir),
		},
		IsDefault: true,
		IsActive:  true,
		CreatedAt: time.Now().Unix(),
	}

	log.Printf("Saving default avatar with ID: %s", defaultAvatar.ID)
	if err := m.Storage.SaveAvatar(defaultAvatar); err != nil {
		return fmt.Errorf("save default avatar: %w", err)
	}

	m.config.Avatars = []Avatar{defaultAvatar}
	// m.config.DefaultID = defaultAvatar.ID
	// m.config.CurrentID = defaultAvatar.ID

	log.Printf("Saving config with default avatar")
	return m.Storage.SaveConfig(m.config)
}

// CreateAvatar creates a new avatar
func (m *Manager) CreateAvatar(name, description string, states map[AvatarState]string) (*Avatar, error) {
	avatar := Avatar{
		Name:        name,
		Description: description,
		States:      states,
	}

	if err := m.Storage.SaveAvatar(avatar); err != nil {
		return nil, fmt.Errorf("save avatar: %w", err)
	}

	m.config.Avatars = append(m.config.Avatars, avatar)
	if err := m.Storage.SaveConfig(m.config); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}

	return &avatar, nil
}

// GetCurrentAvatar returns the currently active avatar
// func (m *Manager) GetCurrentAvatar() (*Avatar, error) {
// 	// // If no current ID set, try to initialize
// 	// if m.config.CurrentID == "" {
// 	// 	if err := m.initializeDefaultAvatar(); err != nil {
// 	// 		return nil, fmt.Errorf("failed to initialize default avatar: %w", err)
// 	// 	}
// 	// }

// 	avatar, err := m.Storage.GetAvatar(m.config.CurrentID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &avatar, nil
// }

// SetCurrentAvatar sets the active avatar
// func (m *Manager) SetCurrentAvatar(id string) error {
// 	// Verify avatar exists
// 	if _, err := m.Storage.GetAvatar(id); err != nil {
// 		return fmt.Errorf("avatar not found: %w", err)
// 	}

// 	// Update active states
// 	avatars, err := m.Storage.ListAvatars()
// 	if err != nil {
// 		return fmt.Errorf("list avatars: %w", err)
// 	}

// 	for _, avatar := range avatars {
// 		avatar.IsActive = avatar.ID == id
// 		if err := m.Storage.SaveAvatar(avatar); err != nil {
// 			return fmt.Errorf("save avatar: %w", err)
// 		}
// 	}

// 	m.config.CurrentID = id
// 	return m.Storage.SaveConfig(m.config)
// }

// GetAvatarState returns the file path for a specific avatar state
func (m *Manager) GetAvatarState(id string, state AvatarState) (string, error) {
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
func (m *Manager) UpdateAvatarState(id string, state AvatarState, imagePath string) error {
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
		avatar.States = make(map[AvatarState]string)
	}

	avatar.States[state] = imagePath
	log.Printf("Updated avatar states: %+v", avatar.States)

	return m.Storage.SaveAvatar(avatar)
}

// ListAvatars returns all available avatars
func (m *Manager) ListAvatars() ([]Avatar, error) {
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

	// Remove avatar from list
	for i, a := range m.config.Avatars {
		if a.ID == id {
			m.config.Avatars = append(m.config.Avatars[:i], m.config.Avatars[i+1:]...)
			break
		}
	}

	return m.Storage.SaveConfig(m.config)
}

// ValidateAvatarState checks if a file is valid for the given state
func (m *Manager) ValidateAvatarState(filePath string, state AvatarState) error {
	ext := filepath.Ext(filePath)
	switch state {
	case StateIdle:
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
			return fmt.Errorf("idle state must be a static image (png/jpg)")
		}
	case StateTalking:
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
	image := AvatarImage{
		Path:      path,
		Type:      ext[1:], // remove dot
		CreatedAt: time.Now().Unix(),
	}
	return m.Storage.SaveAvatarImage(image)
}

// GetActiveAvatars returns all active avatars
func (m *Manager) GetActiveAvatars() ([]Avatar, error) {
	avatars, err := m.Storage.ListAvatars()
	if err != nil {
		return nil, fmt.Errorf("list avatars: %w", err)
	}

	var activeAvatars []Avatar
	for _, avatar := range avatars {
		if avatar.IsActive {
			activeAvatars = append(activeAvatars, avatar)
		}
	}

	return activeAvatars, nil
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