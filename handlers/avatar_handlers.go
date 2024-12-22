package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
		defaultAvatar := types.Avatar{
			ID:          fmt.Sprintf("avatar_%d", time.Now().UnixNano()),
			Name:        "Default",
			Description: "Default avatar",
			States: map[types.AvatarState]string{
				types.StateIdle:    fmt.Sprintf("/%s/idle.png", avatar.AvatarAssetsDir),
				types.StateTalking: fmt.Sprintf("/%s/talking.gif", avatar.AvatarAssetsDir),
			},
			IsDefault: true,
			CreatedAt: time.Now().Unix(),
			SortOrder: 0,
		}
		
		response := types.AvatarList{
			Avatars: []types.Avatar{defaultAvatar},
		}
		
		json.NewEncoder(w).Encode(response)
		return
	}


	response := types.AvatarList{
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
	case "sort":
		if r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.handleSetAvatarSort(w, r, id)
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
		States      map[types.AvatarState]string `json:"states,omitempty"`
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
	
	// Update states if provided
	if request.States != nil {
		// Initialize states map if it doesn't exist
		if existingAvatar.States == nil {
			existingAvatar.States = make(map[types.AvatarState]string)
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



// HandleCreateAvatar handles POST /api/avatars/create
func (h *AvatarHandler) HandleCreateAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get all existing avatars to determine the highest sort order
	existingAvatars, err := h.avatarManager.ListAvatars()
	if err != nil {
		log.Printf("Error getting existing avatars: %v", err)
		http.Error(w, "Failed to create avatar", http.StatusInternalServerError)
		return
	}

	// Find the highest sort order
	maxSortOrder := 0
	maxID := 0
	for _, a := range existingAvatars {
		if a.SortOrder > maxSortOrder {
			maxSortOrder = a.SortOrder
		}
		// Extract ID number
		if id, err := strconv.Atoi(a.ID); err == nil && id > maxID {
			maxID = id
		}
	}

	// Generate ID sequentially
	id := fmt.Sprintf("%d", maxID+1)
	
	newAvatar := types.Avatar{
		ID:          id,  // Set the ID explicitly
		Name:        "New Avatar",
		Description: "New avatar",
		States: map[types.AvatarState]string{
			types.StateIdle:    "/avatars/idle.png",
			types.StateTalking: "/avatars/talking.gif",
		},
		IsDefault: false,
		CreatedAt: time.Now().Unix(),
		SortOrder: maxSortOrder + 1, // Set the sort order to be highest + 1
		TTSVoices: []types.TTSVoice{
			{
				VoiceID: "id_male_darma",
				Provider: "tiktok",
			},
		},
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

// HandleAvatarImageDelete handles DELETE /api/avatar-images/delete
func (h *AvatarHandler) HandleAvatarImageDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Method not allowed",
		})
		return
	}

	var request struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	if request.Path == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Path is required",
		})
		return
	}

	if err := h.avatarManager.DeleteAvatarImage(request.Path); err != nil {
		log.Printf("Error deleting avatar image: %v", err)
		
		var statusCode int
		var errorMessage string
		
		switch {
		case strings.Contains(err.Error(), "not found"):
			statusCode = http.StatusNotFound
			errorMessage = fmt.Sprintf("Image not found: %s", request.Path)
		case strings.Contains(err.Error(), "in use"):
			statusCode = http.StatusBadRequest
			avatarID := "unknown"
			if parts := strings.Split(err.Error(), "avatar "); len(parts) > 1 {
				avatarID = parts[1]
			}
			errorMessage = fmt.Sprintf("Image is in use by avatar %s", avatarID)
		default:
			statusCode = http.StatusInternalServerError
			errorMessage = fmt.Sprintf("Failed to delete avatar image: %v", err)
		}
		
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]string{
			"error": errorMessage,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Image deleted successfully",
	})
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

// handleSetAvatarSort handles PUT /api/avatars/{id}/sort
func (h *AvatarHandler) handleSetAvatarSort(w http.ResponseWriter, r *http.Request, id string) {
	var request struct {
		SortOrder int `json:"sort_order"`
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

	// Update sort order
	existingAvatar.SortOrder = request.SortOrder

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
	avatars, err := h.avatarManager.ListAvatars()
	if err != nil {
		log.Printf("Error getting active avatars for broadcast: %v", err)
		return
	}

	// Create update data
	updateData := map[string]interface{}{
		"avatars": avatars,
	}

	// Broadcast the update
	if err := h.broadcaster.Broadcast(broadcast.Update{
		Type: BroadcastTypeAvatarUpdate,
		Data: updateData,
	}); err != nil {
		log.Printf("Error broadcasting avatar update: %v", err)
	}
} 