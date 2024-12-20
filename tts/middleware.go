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

// TTSQueueItem represents a queued TTS message
type TTSQueueItem struct {
	Data       map[string]interface{}
	AvatarID   string
	BlobURL    string
	VoiceID    string
	Provider   string
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type TTSMiddleware struct {
	clients     map[*websocket.Conn]string
	avatarIds   map[string]bool
	clientsMux  sync.RWMutex
	blobDir     string
	
	// Queue management
	queue       []TTSQueueItem
	queueMux    sync.Mutex
	isSpeaking  bool

	// Avatar selection tracking
	lastUsedAvatars []string
	maxLastUsed     int
}

func NewTTSMiddleware() *TTSMiddleware {
	blobDir := filepath.Join(os.TempDir(), "tts_blobs")
	os.MkdirAll(blobDir, 0755)

	return &TTSMiddleware{
		clients:         make(map[*websocket.Conn]string),
		avatarIds:       make(map[string]bool),
		blobDir:        blobDir,
		queue:          make([]TTSQueueItem, 0),
		isSpeaking:     false,
		maxLastUsed:    3, // Keep track of last 3 avatars used
		lastUsedAvatars: make([]string, 0),
	}
}

// processNextInQueue attempts to process the next item in the queue
func (tm *TTSMiddleware) processNextInQueue() {
	tm.queueMux.Lock()
	defer tm.queueMux.Unlock()

	queueLength := len(tm.queue)
	if tm.isSpeaking {
		log.Printf("Queue: Cannot process next item - Avatar currently speaking. Queue length: %d", queueLength)
		return
	}
	
	if queueLength == 0 {
		log.Printf("Queue: Empty - No items to process")
		return
	}

	// Get next item
	item := tm.queue[0]
	tm.queue = tm.queue[1:]
	tm.isSpeaking = true

	log.Printf("Queue: Processing item - Avatar: %s, Voice: %s (%s), Queue position: 1/%d", 
		item.AvatarID, item.VoiceID, item.Provider, queueLength)

	// Add audio URL and signal to the data
	item.Data["avatar_audio"] = item.BlobURL
	item.Data["signal"] = "avatar_speak"

	// Send only to clients of the chosen avatar
	tm.clientsMux.RLock()
	sentCount := 0
	for client, avatarId := range tm.clients {
		if avatarId == item.AvatarID {
			if err := client.WriteJSON(item.Data); err != nil {
				log.Printf("Queue: Error sending to client - Avatar: %s, Error: %v", avatarId, err)
			} else {
				sentCount++
				log.Printf("Queue: Sent message to client - Avatar: %s", avatarId)
			}
		}
	}
	tm.clientsMux.RUnlock()

	if sentCount == 0 {
		log.Printf("Queue: Warning - No clients found for avatar %s, message dropped", item.AvatarID)
		// Reset speaking state since no clients received the message
		tm.isSpeaking = false
		// Try next item
		tm.processNextInQueue()
	} else {
		log.Printf("Queue: Message sent to %d client(s) for avatar %s, Remaining in queue: %d", 
			sentCount, item.AvatarID, len(tm.queue))
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
		log.Printf("Queue: WebSocket connection attempt without avatarId")
		http.Error(w, "avatarId is required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Queue: WebSocket upgrade failed - %v", err)
		return
	}

	// Add new client to the pool
	tm.clientsMux.Lock()
	tm.clients[conn] = avatarId
	tm.avatarIds[avatarId] = true
	connectedAvatars := len(tm.avatarIds)
	tm.clientsMux.Unlock()

	log.Printf("Queue: New WebSocket client connected - Avatar: %s, Total avatars: %d", 
		avatarId, connectedAvatars)

	// Wait for messages or disconnect
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			tm.clientsMux.Lock()
			delete(tm.clients, conn)
			
			lastConnection := true
			for _, id := range tm.clients {
				if id == avatarId {
					lastConnection = false
					break
				}
			}
			if lastConnection {
				delete(tm.avatarIds, avatarId)
				log.Printf("Queue: Avatar %s disconnected - No more connections for this avatar", avatarId)
			}
			
			conn.Close()
			remainingAvatars := len(tm.avatarIds)
			tm.clientsMux.Unlock()
			
			log.Printf("Queue: WebSocket client disconnected - Remaining avatars: %d", remainingAvatars)
			break
		}

		// Handle incoming messages
		if messageType == websocket.TextMessage {
			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("Queue: Error unmarshaling message - %v", err)
				continue
			}

			// Handle avatar_finished signal
			if signal, ok := msg["signal"].(string); ok && signal == "avatar_finished" {
				if blobURL, ok := msg["avatar_audio"].(string); ok {
					log.Printf("Queue: Received avatar_finished signal - Avatar: %s, Audio: %s", 
						avatarId, blobURL)

					// Extract filename from URL
					filename := filepath.Base(blobURL)
					blobPath := filepath.Join(tm.blobDir, filename)
					
					// Remove the blob file
					if err := os.Remove(blobPath); err != nil {
						log.Printf("Queue: Error removing blob file - %v", err)
					} else {
						log.Printf("Queue: Cleaned up blob file - %s", filename)
					}

					// Mark as not speaking and process next item
					tm.queueMux.Lock()
					tm.isSpeaking = false
					queueLength := len(tm.queue)
					tm.queueMux.Unlock()
					
					log.Printf("Queue: Avatar finished speaking - Avatar: %s, Remaining in queue: %d", 
						avatarId, queueLength)
					
					tm.processNextInQueue()
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

// getRandomAvatarWithWeights selects an avatar with reduced probability for recently used ones
func (tm *TTSMiddleware) getRandomAvatarWithWeights() string {
	avatars := tm.GetConnectedAvatars()
	if len(avatars) == 0 {
		return ""
	}

	// If we only have one avatar, just return it
	if len(avatars) == 1 {
		return avatars[0]
	}

	// Create a map of weights for each avatar
	weights := make(map[string]int)
	for _, avatar := range avatars {
		weights[avatar] = 100 // Base weight
	}

	// Reduce weights for recently used avatars
	for i, avatar := range tm.lastUsedAvatars {
		if _, exists := weights[avatar]; exists {
			// More recent avatars get bigger weight reduction
			reduction := 30 * (len(tm.lastUsedAvatars) - i)
			weights[avatar] = max(10, weights[avatar] - reduction) // Minimum weight of 10
		}
	}

	// Calculate total weight
	totalWeight := 0
	for _, weight := range weights {
		totalWeight += weight
	}

	// Pick a random number between 0 and total weight
	r := rand.Intn(totalWeight)
	
	// Find the avatar that corresponds to this weight
	currentWeight := 0
	for avatar, weight := range weights {
		currentWeight += weight
		if r < currentWeight {
			// Update last used avatars
			tm.lastUsedAvatars = append(tm.lastUsedAvatars, avatar)
			if len(tm.lastUsedAvatars) > tm.maxLastUsed {
				tm.lastUsedAvatars = tm.lastUsedAvatars[1:]
			}
			return avatar
		}
	}

	// Fallback to first avatar (shouldn't happen)
	return avatars[0]
}

// InterceptTTS handles TTS updates and returns whether the update should be broadcasted
func (tm *TTSMiddleware) InterceptTTS(updateType string, data interface{}) bool {
	// Handle clear_tts command
	if updateType == "clear_tts" {
		tm.queueMux.Lock()
		queueLength := len(tm.queue)
		
		// Clean up blobs for all queued items
		for _, item := range tm.queue {
			filename := filepath.Base(item.BlobURL)
			blobPath := filepath.Join(tm.blobDir, filename)
			
			if err := os.Remove(blobPath); err != nil {
				log.Printf("Queue: Error removing blob file during clear - %v", err)
			} else {
				log.Printf("Queue: Cleaned up blob file during clear - %s", filename)
			}
		}
		
		// Clear the queue and reset speaking state
		tm.queue = make([]TTSQueueItem, 0)
		tm.isSpeaking = false
		tm.queueMux.Unlock()
		
		log.Printf("Queue: Cleared %d items from queue due to clear_tts signal", queueLength)
		return true // Allow the clear signal to be broadcasted
	}

	if updateType == "tts" {
		startTime := time.Now()
		log.Printf("Queue: New TTS update received at %s", startTime.Format(time.RFC3339))

		// Get list of connected avatars
		avatars := tm.GetConnectedAvatars()
		if len(avatars) == 0 {
			log.Printf("Queue: No connected avatars available for TTS")
			return false
		}

		// Pick an avatar using weighted random selection
		chosenAvatarId := tm.getRandomAvatarWithWeights()
		if chosenAvatarId == "" {
			log.Printf("Queue: Failed to select an avatar")
			return false
		}
		
		recentAvatars := strings.Join(tm.lastUsedAvatars, ", ")
		log.Printf("Queue: Selected avatar %s from %d connected avatars (Recent avatars: %s)", 
			chosenAvatarId, len(avatars), recentAvatars)
		
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
			log.Printf("Queue: Failed to get voice details for avatar %s - %v", chosenAvatarId, err)
			return false
		}
		log.Printf("Queue: Selected voice %s (%s) for avatar %s", 
			avatar["voice_id"], avatar["provider"], chosenAvatarId)

		// Extract text from the message
		var messageText string
		if content, ok := enrichedData["content"].(map[string]interface{}); ok {
			if sanitized, ok := content["sanitized"].(string); ok {
				messageText = sanitized
			}
		}

		if messageText == "" {
			log.Printf("Queue: No text content found in message")
			return false
		}
		log.Printf("Queue: Processing message text (length: %d characters)", len(messageText))

		// Get audio blob URL
		blobURL, err := tm.getAudioBlob(messageText, avatar["voice_id"].(string), avatar["provider"].(string))
		if err != nil {
			log.Printf("Queue: Failed to get audio blob - %v", err)
			return false
		}
		log.Printf("Queue: Generated audio blob: %s", blobURL)

		// Create queue item
		queueItem := TTSQueueItem{
			Data:     enrichedData,
			AvatarID: chosenAvatarId,
			BlobURL:  blobURL,
			VoiceID:  avatar["voice_id"].(string),
			Provider: avatar["provider"].(string),
		}

		// Add to queue
		tm.queueMux.Lock()
		tm.queue = append(tm.queue, queueItem)
		queueLength := len(tm.queue)
		tm.queueMux.Unlock()

		processingTime := time.Since(startTime)
		log.Printf("Queue: Item added - Avatar: %s, Position: %d, Processing time: %v", 
			chosenAvatarId, queueLength, processingTime)

		// Try to process next item
		tm.processNextInQueue()
		
		return false // Don't broadcast
	}
	return true // Continue with broadcast
} 