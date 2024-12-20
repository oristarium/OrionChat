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

// TTSQueueItem represents an item in the TTS queue
type TTSQueueItem struct {
	Data     map[string]interface{}
	AvatarID string
	BlobURL  string
	VoiceID  string
	Provider string
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type cleanupJob struct {
	path      string
	cleanupAt time.Time
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

	// Cleanup channel
	cleanupChan chan cleanupJob
}

func NewTTSMiddleware() *TTSMiddleware {
	blobDir := filepath.Join(os.TempDir(), "tts_blobs")
	os.MkdirAll(blobDir, 0755)

	tm := &TTSMiddleware{
		clients:         make(map[*websocket.Conn]string),
		avatarIds:       make(map[string]bool),
		blobDir:        blobDir,
		queue:          make([]TTSQueueItem, 0),
		isSpeaking:     false,
		maxLastUsed:    3,
		lastUsedAvatars: make([]string, 0),
		cleanupChan:    make(chan cleanupJob, 100), // Buffer for cleanup requests
	}

	// Start cleanup goroutine
	go tm.cleanupWorker()

	return tm
}

// cleanupWorker handles blob deletion in a separate goroutine
func (tm *TTSMiddleware) cleanupWorker() {
	// Use a priority queue to track delayed cleanups
	type scheduledCleanup struct {
		job      cleanupJob
		deadline time.Time
	}
	
	scheduled := make(map[string]scheduledCleanup)
	
	for {
		select {
		case job := <-tm.cleanupChan:
			if job.cleanupAt.IsZero() {
				// Immediate cleanup
				if err := os.Remove(job.path); err != nil {
					if !os.IsNotExist(err) {
						log.Printf("Queue: Error removing blob file in cleanup worker - %v", err)
					}
				} else {
					log.Printf("Queue: Cleaned up blob file in worker - %s", filepath.Base(job.path))
				}
			} else {
				// Schedule for later cleanup
				scheduled[job.path] = scheduledCleanup{
					job:      job,
					deadline: job.cleanupAt,
				}
				log.Printf("Queue: Scheduled cleanup for %s at %v", filepath.Base(job.path), job.cleanupAt)
			}
			
		case <-time.After(10 * time.Second): // Check scheduled cleanups every 10 seconds
			now := time.Now()
			for path, cleanup := range scheduled {
				if now.After(cleanup.deadline) {
					if err := os.Remove(cleanup.job.path); err != nil {
						if !os.IsNotExist(err) {
							log.Printf("Queue: Error removing scheduled blob file - %v", err)
						}
					} else {
						log.Printf("Queue: Cleaned up scheduled blob file - %s", filepath.Base(cleanup.job.path))
					}
					delete(scheduled, path)
				}
			}
		}
	}
}

// queueCleanup adds a blob path to the cleanup queue
func (tm *TTSMiddleware) queueCleanup(blobPath string, delay time.Duration) {
	job := cleanupJob{
		path: blobPath,
	}
	if delay > 0 {
		job.cleanupAt = time.Now().Add(delay)
	}
	
	select {
	case tm.cleanupChan <- job:
		// Successfully queued for cleanup
	default:
		// Channel is full, log warning and try to delete immediately
		log.Printf("Queue: Cleanup channel full, attempting immediate deletion - %s", filepath.Base(blobPath))
		if err := os.Remove(blobPath); err != nil {
			log.Printf("Queue: Error removing blob file (immediate) - %v", err)
		}
	}
}

// processNextInQueue processes the next item in the queue if available
func (tm *TTSMiddleware) processNextInQueue() {
	tm.queueMux.Lock()
	if tm.isSpeaking || len(tm.queue) == 0 {
		tm.queueMux.Unlock()
		return
	}

	// Get next item and mark as speaking
	item := tm.queue[0]
	tm.queue = tm.queue[1:]
	tm.isSpeaking = true
	queueLength := len(tm.queue)
	tm.queueMux.Unlock()

	log.Printf("Queue: Processing next item - Avatar: %s, Queue length: %d", 
		item.AvatarID, queueLength)

	// Update last used avatars list
	tm.updateLastUsedAvatars(item.AvatarID)

	// Prepare WebSocket message
	message := map[string]interface{}{
		"signal":       "avatar_speak",
		"content":      item.Data["content"],
		"avatar_audio": item.BlobURL,
	}

	// Send to matching clients
	tm.clientsMux.RLock()
	sent := false
	for client, avatarId := range tm.clients {
		if avatarId == item.AvatarID {
			if err := client.WriteJSON(message); err != nil {
				log.Printf("Queue: Error sending to client - %v", err)
				continue
			}
			sent = true
			log.Printf("Queue: Sent message to avatar %s", avatarId)
		}
	}
	tm.clientsMux.RUnlock()

	if !sent {
		log.Printf("Queue: No matching clients found for avatar %s, skipping audio", 
			item.AvatarID)
		
		// Clean up the blob since it won't be used
		filename := filepath.Base(item.BlobURL)
		blobPath := filepath.Join(tm.blobDir, filename)
		tm.queueCleanup(blobPath, 0)

		// Mark as not speaking and process next item
		tm.queueMux.Lock()
		tm.isSpeaking = false
		tm.queueMux.Unlock()

		// Try next item
		tm.processNextInQueue()
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
	tm.queueCleanup(blobFile.Name(), 5*time.Minute)

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
	// Upgrade HTTP connection to WebSocket
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Get avatarId from query parameters
	avatarId := r.URL.Query().Get("avatarId")
	if avatarId == "" {
		log.Printf("WebSocket connection attempt without avatarId")
		c.Close()
		return
	}

	// Register new client
	tm.clientsMux.Lock()
	tm.clients[c] = avatarId
	tm.avatarIds[avatarId] = true
	numClients := len(tm.clients)
	tm.clientsMux.Unlock()

	log.Printf("New WebSocket client connected - Avatar: %s, Total clients: %d", 
		avatarId, numClients)

	// Schedule queue processing after 4 seconds
	go func() {
		time.Sleep(4 * time.Second)
		tm.queueMux.Lock()
		if tm.isSpeaking {
			log.Printf("Queue: New client connected but queue is already speaking")
			tm.queueMux.Unlock()
			return
		}
		tm.queueMux.Unlock()
		
		log.Printf("Queue: Processing queue after new client connection delay")
		tm.processNextInQueue()
	}()

	// Handle incoming messages
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			tm.clientsMux.Lock()
			delete(tm.clients, c)
			
			// Check if this was the last connection for this avatar
			lastConnection := true
			for conn, id := range tm.clients {
				if id == avatarId && conn != c {
					lastConnection = false
					break
				}
			}
			
			// If this was the last connection for this avatar, remove it from active avatars
			if lastConnection {
				delete(tm.avatarIds, avatarId)
			}
			
			c.Close()
			remainingAvatars := len(tm.avatarIds)
			tm.clientsMux.Unlock()
			
			log.Printf("Queue: WebSocket client disconnected - Avatar: %s, Remaining avatars: %d", 
				avatarId, remainingAvatars)
			return
		}

		// Process the message
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Queue: Error parsing WebSocket message - %v", err)
			continue
		}

		// Handle avatar_finished signal
		if signal, ok := msg["signal"].(string); ok && signal == "avatar_finished" {
			if blobURL, ok := msg["avatar_audio"].(string); ok {
				log.Printf("Queue: Received avatar_finished signal - Avatar: %s, Audio: %s", 
					avatarId, blobURL)

				// Extract filename from URL and queue for cleanup
				filename := filepath.Base(blobURL)
				blobPath := filepath.Join(tm.blobDir, filename)
				tm.queueCleanup(blobPath, 0) // immediate cleanup

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

// GetConnectedAvatars returns a list of currently connected avatar IDs
func (tm *TTSMiddleware) GetConnectedAvatars() []string {
	tm.clientsMux.RLock()
	defer tm.clientsMux.RUnlock()

	// Use a map to deduplicate avatar IDs
	avatarMap := make(map[string]bool)
	for _, avatarId := range tm.clients {
		avatarMap[avatarId] = true
	}

	// Convert map to slice
	avatars := make([]string, 0, len(avatarMap))
	for avatarId := range avatarMap {
		avatars = append(avatars, avatarId)
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
			tm.queueCleanup(blobPath, 0)
		}
		
		// Clear the queue and reset speaking state
		tm.queue = make([]TTSQueueItem, 0)
		tm.isSpeaking = false
		tm.queueMux.Unlock()
		
		log.Printf("Queue: Cleared %d items from queue due to clear_tts signal", queueLength)
		return true // Allow the clear signal to be broadcasted
	}

	// Handle TTS updates
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
		isSpeaking := tm.isSpeaking
		tm.queueMux.Unlock()

		processingTime := time.Since(startTime)
		log.Printf("Queue: Item added - Avatar: %s, Position: %d, Processing time: %v", 
			chosenAvatarId, queueLength, processingTime)

		// Process queue if not currently speaking
		if !isSpeaking {
			go tm.processNextInQueue()
		}

		return false // Don't broadcast
	}

	return true // Continue with broadcast
}

// updateLastUsedAvatars maintains a list of recently used avatars
func (tm *TTSMiddleware) updateLastUsedAvatars(avatarId string) {
	// Add the new avatar to the front of the list
	tm.lastUsedAvatars = append([]string{avatarId}, tm.lastUsedAvatars...)

	// Trim the list if it exceeds maxLastUsed
	if len(tm.lastUsedAvatars) > tm.maxLastUsed {
		tm.lastUsedAvatars = tm.lastUsedAvatars[:tm.maxLastUsed]
	}

	log.Printf("Queue: Updated last used avatars: %s", strings.Join(tm.lastUsedAvatars, ", "))
}
 