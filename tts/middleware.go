package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"encoding/base64"

	"github.com/gorilla/websocket"
	"github.com/oristarium/orionchat/types"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

type TTSMiddleware struct {
	clients    map[*websocket.Conn]string // Maps connection to avatarId
	avatarIds  map[string]bool           // Tracks which avatarIds are connected
	clientsMux sync.RWMutex
	blobDir    string                    // Directory for temporary audio blobs
}

func NewTTSMiddleware() *TTSMiddleware {
	blobDir := filepath.Join(os.TempDir(), "tts_blobs")
	os.MkdirAll(blobDir, 0755)

	return &TTSMiddleware{
		clients:   make(map[*websocket.Conn]string),
		avatarIds: make(map[string]bool),
		blobDir:   blobDir,
	}
}

// getAudioBlob fetches TTS audio and stores it as a temporary blob
func (tm *TTSMiddleware) getAudioBlob(text string, voiceId string, provider string) (string, error) {
	// Prepare TTS request
	ttsReq := map[string]interface{}{
		"text":           text,
		"voice_id":       voiceId,
		"voice_provider": provider,
	}

	jsonData, err := json.Marshal(ttsReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal TTS request: %v", err)
	}

	// Make request to TTS service
	resp, err := http.Post("http://localhost:7777/tts-service", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to fetch TTS audio: %v", err)
	}
	defer resp.Body.Close()

	var ttsResp struct {
		Audio string `json:"audio"` // base64 encoded audio
	}
	if err := json.NewDecoder(resp.Body).Decode(&ttsResp); err != nil {
		return "", fmt.Errorf("failed to decode TTS response: %v", err)
	}

	// Remove any potential data URL prefix (e.g., "data:audio/mp3;base64,")
	base64Data := ttsResp.Audio
	if idx := strings.Index(base64Data, ","); idx != -1 {
		base64Data = base64Data[idx+1:]
	}

	// Decode base64 to raw audio bytes
	audioData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 audio: %v", err)
	}

	// Create temporary file
	blobFile, err := os.CreateTemp(tm.blobDir, "tts_*.mp3")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}

	// Write the raw audio data and close the file
	if _, err := blobFile.Write(audioData); err != nil {
		blobFile.Close()
		os.Remove(blobFile.Name())
		return "", fmt.Errorf("failed to write audio data: %v", err)
	}
	blobFile.Close()

	// Schedule cleanup after 5 minutes
	go func(filename string) {
		time.Sleep(5 * time.Minute)
		os.Remove(filename)
		log.Printf("Cleaned up audio blob: %s", filename)
	}(blobFile.Name())

	// Return the blob URL
	return fmt.Sprintf("/tts-blob/%s", filepath.Base(blobFile.Name())), nil
}

// getRandomAvatarVoice fetches voice details for a given avatar ID
func (tm *TTSMiddleware) getRandomAvatarVoice(avatarId string) (map[string]interface{}, error) {
	// Make internal request to get avatar details
	url := fmt.Sprintf("http://localhost:7777/api/avatars/%s/get", avatarId)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch avatar details: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var avatar types.Avatar
	if err := json.Unmarshal(body, &avatar); err != nil {
		return nil, fmt.Errorf("failed to unmarshal avatar data: %v", err)
	}

	// Check if there are any TTS voices
	if len(avatar.TTSVoices) == 0 {
		return nil, fmt.Errorf("avatar has no TTS voices")
	}

	// Pick a random TTS voice
	randomVoice := avatar.TTSVoices[rand.Intn(len(avatar.TTSVoices))]

	// Create response with avatar ID and voice details
	return map[string]interface{}{
		"avatar_id": avatarId,
		"voice_id":  randomVoice.VoiceID,
		"provider":  randomVoice.Provider,
	}, nil
}

// HandleWebSocket handles new WebSocket connections
func (tm *TTSMiddleware) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	avatarId := r.URL.Query().Get("avatarId")
	if avatarId == "" {
		log.Printf("WebSocket connection attempt without avatarId")
		http.Error(w, "avatarId is required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Add new client to the pool
	tm.clientsMux.Lock()
	tm.clients[conn] = avatarId
	tm.avatarIds[avatarId] = true
	connectedAvatars := len(tm.avatarIds)
	tm.clientsMux.Unlock()

	log.Printf("New WebSocket client connected for avatarId: %s. Total connected avatars: %d", avatarId, connectedAvatars)

	// Wait for messages or disconnect
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			tm.clientsMux.Lock()
			delete(tm.clients, conn)
			
			// Check if this was the last connection for this avatarId
			lastConnection := true
			for _, id := range tm.clients {
				if id == avatarId {
					lastConnection = false
					break
				}
			}
			if lastConnection {
				delete(tm.avatarIds, avatarId)
				log.Printf("Avatar %s disconnected. No more connections for this avatar.", avatarId)
			}
			
			conn.Close()
			remainingAvatars := len(tm.avatarIds)
			tm.clientsMux.Unlock()
			
			log.Printf("WebSocket client disconnected. Remaining connected avatars: %d", remainingAvatars)
			break
		}

		// Handle incoming messages
		if messageType == websocket.TextMessage {
			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				continue
			}

			// Handle avatar_finished signal
			if signal, ok := msg["signal"].(string); ok && signal == "avatar_finished" {
				if blobURL, ok := msg["avatar_audio"].(string); ok {
					// Extract filename from URL
					filename := filepath.Base(blobURL)
					blobPath := filepath.Join(tm.blobDir, filename)
					
					// Remove the blob file
					if err := os.Remove(blobPath); err != nil {
						log.Printf("Error removing blob file: %v", err)
					} else {
						log.Printf("Cleaned up blob file after playback: %s", filename)
					}
				}
			}
		}
	}
}

// GetConnectedAvatars returns a list of currently connected avatar IDs
func (tm *TTSMiddleware) GetConnectedAvatars() []string {
	tm.clientsMux.RLock()
	defer tm.clientsMux.RUnlock()
	
	avatars := make([]string, 0, len(tm.avatarIds))
	for id := range tm.avatarIds {
		avatars = append(avatars, id)
	}
	return avatars
}

// InterceptTTS handles TTS updates and returns whether the update should be broadcasted
func (tm *TTSMiddleware) InterceptTTS(updateType string, data interface{}) bool {
	if updateType == "tts" {
		log.Printf("TTS Update intercepted - Content: %v", data)

		// Get list of connected avatars and pick one randomly
		avatars := tm.GetConnectedAvatars()
		if len(avatars) == 0 {
			log.Printf("No connected avatars")
			return false
		}

		// Pick a random avatar ID
		chosenAvatarId := avatars[rand.Intn(len(avatars))]
		
		// Create enriched data with the original data
		enrichedData := make(map[string]interface{})
		if originalData, ok := data.(map[string]interface{}); ok {
			for k, v := range originalData {
				enrichedData[k] = v
			}
		}

		// Get voice details for the chosen avatar
		avatar, err := tm.getRandomAvatarVoice(chosenAvatarId)
		if err != nil {
			log.Printf("Failed to get avatar voice details: %v", err)
			return false
		}

		// Extract text from the message
		var messageText string
		if content, ok := enrichedData["content"].(map[string]interface{}); ok {
			if sanitized, ok := content["sanitized"].(string); ok {
				messageText = sanitized
			}
		}

		if messageText == "" {
			log.Printf("No text content found in message")
			return false
		}

		// Get audio blob URL
		blobURL, err := tm.getAudioBlob(messageText, avatar["voice_id"].(string), avatar["provider"].(string))
		if err != nil {
			log.Printf("Failed to get audio blob: %v", err)
			return false
		}

		// Add audio URL to enriched data
		enrichedData["avatar_audio"] = blobURL
		enrichedData["signal"] = "avatar_speak"
		
		// Forward only to clients of the chosen avatar
		tm.clientsMux.RLock()
		for client, avatarId := range tm.clients {
			if avatarId == chosenAvatarId {
				if err := client.WriteJSON(enrichedData); err != nil {
					log.Printf("Error sending to WebSocket: %v", err)
					// If we can't write to the client, remove it
					tm.clientsMux.RUnlock()
					tm.clientsMux.Lock()
					delete(tm.clients, client)
					
					// Check if this was the last connection for this avatarId
					lastConnection := true
					for _, id := range tm.clients {
						if id == chosenAvatarId {
							lastConnection = false
							break
						}
					}
					if lastConnection {
						delete(tm.avatarIds, chosenAvatarId)
						log.Printf("Avatar %s removed due to write error", chosenAvatarId)
					}
					
					client.Close()
					tm.clientsMux.Unlock()
					tm.clientsMux.RLock()
				}
			}
		}
		tm.clientsMux.RUnlock()
		
		log.Printf("Message forwarded to avatar: %s with audio URL: %s", chosenAvatarId, blobURL)
		return false // Don't broadcast
	}
	return true // Continue with broadcast
} 