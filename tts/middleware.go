package tts

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
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
}

func NewTTSMiddleware() *TTSMiddleware {
	return &TTSMiddleware{
		clients:   make(map[*websocket.Conn]string),
		avatarIds: make(map[string]bool),
	}
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

	// Wait for disconnect
	for {
		if _, _, err := conn.NextReader(); err != nil {
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
		
		// Forward to all WebSocket clients
		tm.clientsMux.RLock()
		for client := range tm.clients {
			if err := client.WriteJSON(data); err != nil {
				log.Printf("Error sending to WebSocket: %v", err)
				// If we can't write to the client, remove it
				avatarId := tm.clients[client]
				tm.clientsMux.RUnlock()
				tm.clientsMux.Lock()
				delete(tm.clients, client)
				
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
				}
				
				client.Close()
				tm.clientsMux.Unlock()
				tm.clientsMux.RLock()
			}
		}
		tm.clientsMux.RUnlock()
		
		return false // Don't broadcast
	}
	return true // Continue with broadcast
} 