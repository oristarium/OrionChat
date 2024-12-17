package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oristarium/orionchat/avatar" // Update with your actual module name
	"github.com/oristarium/orionchat/broadcast"
	"github.com/oristarium/orionchat/types"
)

// AvatarHandler handles all avatar-related HTTP requests
type AvatarHandler struct {
	avatarManager *avatar.Manager
	fileHandler   *FileHandler
	broadcaster   Broadcaster
}

// Broadcaster interface defines methods for broadcasting updates
type Broadcaster interface {
	Broadcast(update broadcast.Update) error
}

// NewAvatarHandler creates a new AvatarHandler instance
func NewAvatarHandler(avatarManager *avatar.Manager, fileHandler *FileHandler, broadcaster Broadcaster) *AvatarHandler {
	return &AvatarHandler{
		avatarManager: avatarManager,
		fileHandler:   fileHandler,
		broadcaster:   broadcaster,
	}
}

// HandleAvatars handles GET /api/avatars
func (h *AvatarHandler) HandleAvatars(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	avatars, err := h.avatarManager.ListAvatars()
	if err != nil {
		log.Printf("Error listing avatars: %v", err)
		// Return default avatar instead of error
		defaultAvatar := avatar.Avatar{
			ID:          fmt.Sprintf("avatar_%d", time.Now().UnixNano()),
			Name:        "Default",
			Description: "Default avatar",
			States: map[avatar.AvatarState]string{
				avatar.StateIdle:    fmt.Sprintf("/%s/idle.png", avatar.AvatarAssetsDir),
				avatar.StateTalking: fmt.Sprintf("/%s/talking.gif", avatar.AvatarAssetsDir),
			},
			IsDefault: true,
			CreatedAt: time.Now().Unix(),
		}
		
		response := avatar.AvatarList{
			Avatars: []avatar.Avatar{defaultAvatar},
		}
		
		json.NewEncoder(w).Encode(response)
		return
	}


	response := avatar.AvatarList{
		Avatars: avatars,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// HandleAvatarDetail handles /api/avatars/{id}/(get|set|delete|voices)
func (h *AvatarHandler) HandleAvatarDetail(w http.ResponseWriter, r *http.Request) {
	segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(segments) != 4 { // api/avatars/{id}/{action}
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	id := segments[2]
	action := segments[3]

	switch action {
	case "get":
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.handleGetAvatar(w, r, id)
	case "set":
		if r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.handleSetAvatarDetail(w, r, id)
	case "delete":
		if r.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.handleDeleteAvatar(w, r, id)
	case "voices":
		switch r.Method {
		case http.MethodGet:
			h.handleGetAvatarVoices(w, r, id)
		case http.MethodPut:
			h.handleSetAvatarVoices(w, r, id)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
	}
}

// handleGetAvatar handles GET /api/avatars/{id}/get
func (h *AvatarHandler) handleGetAvatar(w http.ResponseWriter, _ *http.Request, id string) {
	avatar, err := h.avatarManager.Storage.GetAvatar(id)
	if err != nil {
		log.Printf("Error getting avatar: %v", err)
		http.Error(w, "Avatar not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(avatar)
}

// handleSetAvatarDetail handles PUT /api/avatars/{id}/set
func (h *AvatarHandler) handleSetAvatarDetail(w http.ResponseWriter, r *http.Request, id string) {
	var request struct {
		Name        string                        `json:"name,omitempty"`
		Description string                        `json:"description,omitempty"`
		States      map[avatar.AvatarState]string `json:"states,omitempty"`
		IsActive    *bool                         `json:"is_active,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get existing avatar
	existingAvatar, err := h.avatarManager.Storage.GetAvatar(id)
	if err != nil {
		http.Error(w, "Avatar not found", http.StatusNotFound)
		return
	}

	// Update fields if provided
	if request.Name != "" {
		existingAvatar.Name = request.Name
	}
	if request.Description != "" {
		existingAvatar.Description = request.Description
	}
	if request.IsActive != nil {
		existingAvatar.IsActive = *request.IsActive
	}
	
	// Update states if provided
	if request.States != nil {
		// Initialize states map if it doesn't exist
		if existingAvatar.States == nil {
			existingAvatar.States = make(map[avatar.AvatarState]string)
		}
		// Merge new states with existing ones
		for state, path := range request.States {
			existingAvatar.States[state] = path
		}
	}
	
	// Save updated avatar
	if err := h.avatarManager.Storage.SaveAvatar(existingAvatar); err != nil {
		http.Error(w, "Failed to save avatar", http.StatusInternalServerError)
		return
	}

	// Broadcast the update
	h.broadcastAvatarUpdate()

	w.WriteHeader(http.StatusOK)
}

// HandleActiveAvatars handles GET /api/avatars/active
func (h *AvatarHandler) HandleActiveAvatars(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	avatars, err := h.avatarManager.GetActiveAvatars()
	if err != nil {
		log.Printf("Error getting active avatars: %v", err)
		http.Error(w, "Failed to get active avatars", http.StatusInternalServerError)
		return
	}

	response := avatar.AvatarList{
		Avatars: avatars,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// HandleCreateAvatar handles POST /api/avatars/create
func (h *AvatarHandler) HandleCreateAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Generate ID first
	id := fmt.Sprintf("avatar_%d", time.Now().UnixNano())
	
	newAvatar := avatar.Avatar{
		ID:          id,  // Set the ID explicitly
		Name:        "New Avatar",
		Description: "New avatar",
		States: map[avatar.AvatarState]string{
			avatar.StateIdle:    "/avatars/idle.png",
			avatar.StateTalking: "/avatars/talking.gif",
		},
		IsDefault: false,
		IsActive:  false,
		CreatedAt: time.Now().Unix(),
	}

	if err := h.avatarManager.Storage.SaveAvatar(newAvatar); err != nil {
		log.Printf("Error saving new avatar: %v", err)
		http.Error(w, "Failed to create avatar", http.StatusInternalServerError)
		return
	}

	// Update config to include new avatar
	config, err := h.avatarManager.Storage.GetConfig()
	if err != nil {
		log.Printf("Error getting config: %v", err)
		http.Error(w, "Failed to update config", http.StatusInternalServerError)
		return
	}

	config.Avatars = append(config.Avatars, newAvatar)
	if err := h.avatarManager.Storage.SaveConfig(config); err != nil {
		log.Printf("Error saving config: %v", err)
		http.Error(w, "Failed to update config", http.StatusInternalServerError)
		return
	}

	// Broadcast the update
	h.broadcastAvatarUpdate()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newAvatar)
}

// handleDeleteAvatar handles DELETE /api/avatars/{id}/delete
func (h *AvatarHandler) handleDeleteAvatar(w http.ResponseWriter, _ *http.Request, id string) {
	// First check if avatar exists
	avatar, err := h.avatarManager.Storage.GetAvatar(id)
	if err != nil {
		http.Error(w, "Avatar not found", http.StatusNotFound)
		return
	}

	// Don't allow deletion of default avatar
	if avatar.IsDefault {
		http.Error(w, "Cannot delete default avatar", http.StatusBadRequest)
		return
	}

	if err := h.avatarManager.DeleteAvatar(id); err != nil {
		log.Printf("Error deleting avatar: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast the update
	h.broadcastAvatarUpdate()

	w.WriteHeader(http.StatusOK)
}

// HandleAvatarUpload handles POST /api/avatars/upload
func (h *AvatarHandler) HandleAvatarUpload(w http.ResponseWriter, r *http.Request) {
	h.fileHandler.HandleUpload(w, r, struct {
		FileField  string
		TypeField  string
		Directory  string
		Bucket     string
		KeyPrefix  string
		OnSuccess  func(string) error
	}{
		FileField:  "avatar",
		TypeField:  "type",
		Directory:  avatar.AvatarAssetsDir,
		Bucket:     avatar.AvatarBucket,
		KeyPrefix:  "avatar",
	})
}

// HandleAvatarImages handles GET /api/avatars/images
func (h *AvatarHandler) HandleAvatarImages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	avatarDir := filepath.Join(avatar.AssetsDir, avatar.AvatarAssetsDir)
	files, err := os.ReadDir(avatarDir)
	if err != nil {
		log.Printf("Error reading avatar directory: %v", err)
		defaultImages := []string{
			fmt.Sprintf("/%s/idle.png", avatar.AvatarAssetsDir),
			fmt.Sprintf("/%s/talking.gif", avatar.AvatarAssetsDir),
		}
		json.NewEncoder(w).Encode(map[string][]string{
			"avatar-images": defaultImages,
		})
		return
	}

	var paths []string
	for _, file := range files {
		if !file.IsDir() {
			paths = append(paths, fmt.Sprintf("/%s/%s", avatar.AvatarAssetsDir, file.Name()))
		}
	}

	json.NewEncoder(w).Encode(map[string][]string{
		"avatar-images": paths,
	})
}

// HandleAvatarImageUpload handles POST /api/avatars/images/upload
func (h *AvatarHandler) HandleAvatarImageUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.fileHandler.HandleUpload(w, r, struct {
		FileField  string
		TypeField  string
		Directory  string
		Bucket     string
		KeyPrefix  string
		OnSuccess  func(string) error
	}{
		FileField:  "avatar",
		TypeField:  "type",
		Directory:  avatar.AvatarAssetsDir,
		Bucket:     avatar.AvatarBucket,
		KeyPrefix:  "avatar",
		OnSuccess: func(path string) error {
			return h.avatarManager.RegisterAvatarImage(path)
		},
	})
}

// HandleAvatarImageDelete handles DELETE /api/avatars/images/delete/{path}
func (h *AvatarHandler) HandleAvatarImageDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/avatar-images/delete/")
	if path == "" {
		http.Error(w, "Path is required", http.StatusBadRequest)
		return
	}

	if err := h.avatarManager.DeleteAvatarImage(path); err != nil {
		log.Printf("Error deleting avatar image: %v", err)
		http.Error(w, "Failed to delete avatar image", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleGetAvatarVoices handles GET /api/avatars/{id}/voices
func (h *AvatarHandler) handleGetAvatarVoices(w http.ResponseWriter, _ *http.Request, id string) {
	avatar, err := h.avatarManager.Storage.GetAvatar(id)
	if err != nil {
		http.Error(w, "Avatar not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]types.TTSVoice{
		"tts_voices": avatar.TTSVoices,
	})
}

// handleSetAvatarVoices handles PUT /api/avatars/{id}/voices
func (h *AvatarHandler) handleSetAvatarVoices(w http.ResponseWriter, r *http.Request, id string) {
	var request struct {
		Voices []types.TTSVoice `json:"voices"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get existing avatar
	existingAvatar, err := h.avatarManager.Storage.GetAvatar(id)
	if err != nil {
		http.Error(w, "Avatar not found", http.StatusNotFound)
		return
	}

	// Update voices
	existingAvatar.TTSVoices = request.Voices

	// Save updated avatar
	if err := h.avatarManager.Storage.SaveAvatar(existingAvatar); err != nil {
		http.Error(w, "Failed to save avatar", http.StatusInternalServerError)
		return
	}

	// Broadcast the update
	h.broadcastAvatarUpdate()

	w.WriteHeader(http.StatusOK)
}

// Add this constant at the top of the file
const (
	BroadcastTypeAvatarUpdate = "avatar_update"
)

// Modify the helper method to only send active avatars
func (h *AvatarHandler) broadcastAvatarUpdate() {
	// Get active avatars
	activeAvatars, err := h.avatarManager.GetActiveAvatars()
	if err != nil {
		log.Printf("Error getting active avatars for broadcast: %v", err)
		return
	}

	// Create update data
	updateData := map[string]interface{}{
		"avatars": activeAvatars,
	}

	// Broadcast the update
	if err := h.broadcaster.Broadcast(broadcast.Update{
		Type: BroadcastTypeAvatarUpdate,
		Data: updateData,
	}); err != nil {
		log.Printf("Error broadcasting avatar update: %v", err)
	}
} 