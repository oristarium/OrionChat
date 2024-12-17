package avatar

import (
	"fmt"
	"log"
	"path/filepath"
	"time"
)

// Manager handles avatar business logic
type Manager struct {
	storage *Storage
	config  AvatarList
}

// NewManager creates a new avatar manager
func NewManager(storage *Storage) (*Manager, error) {
	m := &Manager{
		storage: storage,
	}

	// Load or initialize config
	config, err := storage.GetConfig()
	if err != nil {
		if err != fmt.Errorf("bucket not found") {
			return nil, fmt.Errorf("get config: %w", err)
		}
		config = AvatarList{}
	}
	m.config = config

	// Initialize default avatar if needed
	if m.config.CurrentID == "" {
		log.Printf("No current avatar, initializing default")
		if err := m.initializeDefaultAvatar(); err != nil {
			return nil, fmt.Errorf("initialize default avatar: %w", err)
		}
	}

	return m, nil
}

// initializeDefaultAvatar creates the default avatar
func (m *Manager) initializeDefaultAvatar() error {
	log.Printf("Creating default avatar")
	defaultAvatar := Avatar{
		ID:          fmt.Sprintf("avatar_%d", time.Now().UnixNano()),
		Name:        "Default",
		Description: "Default avatar",
		States: map[AvatarState]string{
			StateIdle:    "/avatars/idle.png",
			StateTalking: "/avatars/talking.gif",
		},
		IsDefault: true,
	}

	log.Printf("Saving default avatar with ID: %s", defaultAvatar.ID)
	if err := m.storage.SaveAvatar(defaultAvatar); err != nil {
		return fmt.Errorf("save default avatar: %w", err)
	}

	m.config.Avatars = []Avatar{defaultAvatar}
	m.config.DefaultID = defaultAvatar.ID
	m.config.CurrentID = defaultAvatar.ID

	log.Printf("Saving config with default avatar")
	return m.storage.SaveConfig(m.config)
}

// CreateAvatar creates a new avatar
func (m *Manager) CreateAvatar(name, description string, states map[AvatarState]string) (*Avatar, error) {
	avatar := Avatar{
		Name:        name,
		Description: description,
		States:      states,
	}

	if err := m.storage.SaveAvatar(avatar); err != nil {
		return nil, fmt.Errorf("save avatar: %w", err)
	}

	m.config.Avatars = append(m.config.Avatars, avatar)
	if err := m.storage.SaveConfig(m.config); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}

	return &avatar, nil
}

// GetCurrentAvatar returns the currently active avatar
func (m *Manager) GetCurrentAvatar() (*Avatar, error) {
	// If no current ID set, try to initialize
	if m.config.CurrentID == "" {
		if err := m.initializeDefaultAvatar(); err != nil {
			return nil, fmt.Errorf("failed to initialize default avatar: %w", err)
		}
	}

	avatar, err := m.storage.GetAvatar(m.config.CurrentID)
	if err != nil {
		return nil, err
	}
	return &avatar, nil
}

// SetCurrentAvatar sets the active avatar
func (m *Manager) SetCurrentAvatar(id string) error {
	// Verify avatar exists
	if _, err := m.storage.GetAvatar(id); err != nil {
		return fmt.Errorf("avatar not found: %w", err)
	}

	m.config.CurrentID = id
	return m.storage.SaveConfig(m.config)
}

// GetAvatarState returns the file path for a specific avatar state
func (m *Manager) GetAvatarState(id string, state AvatarState) (string, error) {
	avatar, err := m.storage.GetAvatar(id)
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
func (m *Manager) UpdateAvatarState(id string, state AvatarState, filePath string) error {
	log.Printf("UpdateAvatarState called - ID: %s, State: %s, Path: %s", id, state, filePath)

	avatar, err := m.storage.GetAvatar(id)
	if err != nil {
		log.Printf("Failed to get avatar: %v", err)
		return fmt.Errorf("get avatar: %w", err)
	}

	log.Printf("Found avatar: %+v", avatar)

	if avatar.States == nil {
		log.Printf("Initializing States map")
		avatar.States = make(map[AvatarState]string)
	}

	avatar.States[state] = filePath
	log.Printf("Updated avatar states: %+v", avatar.States)

	return m.storage.SaveAvatar(avatar)
}

// ListAvatars returns all available avatars
func (m *Manager) ListAvatars() ([]Avatar, error) {
	return m.storage.ListAvatars()
}

// DeleteAvatar removes an avatar
func (m *Manager) DeleteAvatar(id string) error {
	// Don't allow deleting the default avatar
	avatar, err := m.storage.GetAvatar(id)
	if err != nil {
		return fmt.Errorf("get avatar: %w", err)
	}

	if avatar.IsDefault {
		return fmt.Errorf("cannot delete default avatar")
	}

	// If deleting current avatar, switch to default
	if id == m.config.CurrentID {
		m.config.CurrentID = m.config.DefaultID
		if err := m.storage.SaveConfig(m.config); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
	}

	// Remove avatar from list
	for i, a := range m.config.Avatars {
		if a.ID == id {
			m.config.Avatars = append(m.config.Avatars[:i], m.config.Avatars[i+1:]...)
			break
		}
	}

	return m.storage.SaveConfig(m.config)
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