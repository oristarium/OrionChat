package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

// ChatAuthor represents the author of a chat message
type ChatAuthor struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Roles       struct {
		Broadcaster bool `json:"broadcaster"`
		Moderator   bool `json:"moderator"`
		Subscriber  bool `json:"subscriber"`
		Verified    bool `json:"verified"`
	} `json:"roles"`
	Badges []struct {
		Type     string `json:"type"`
		Label    string `json:"label"`
		ImageURL string `json:"image_url"`
	} `json:"badges"`
}

// ChatContent represents the content of a chat message
type ChatContent struct {
	Raw       string `json:"raw"`
	Formatted string `json:"formatted"`
	Sanitized string `json:"sanitized"`
	RawHTML   string `json:"rawHtml"`
	Elements  []struct {
		Type     string `json:"type"`
		Value    string `json:"value"`
		Position []int  `json:"position"`
		Metadata *struct {
			URL      string `json:"url"`
			Alt      string `json:"alt"`
			IsCustom bool   `json:"is_custom"`
		} `json:"metadata,omitempty"`
	} `json:"elements"`
}

// ChatMetadata represents metadata for a chat message
type ChatMetadata struct {
	Type         string `json:"type"`
	MonetaryData *struct {
		Amount    string `json:"amount"`
		Formatted string `json:"formatted"`
		Color     string `json:"color"`
	} `json:"monetary_data,omitempty"`
	Sticker *struct {
		URL string `json:"url"`
		Alt string `json:"alt"`
	} `json:"sticker,omitempty"`
}

// ChatMessageData represents the inner data of a chat message
type ChatMessageData struct {
	Author   ChatAuthor   `json:"author"`
	Content  ChatContent  `json:"content"`
	Metadata ChatMetadata `json:"metadata"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Type      string         `json:"type"`
	Platform  string         `json:"platform"`
	Timestamp string         `json:"timestamp"`
	MessageID string         `json:"message_id"`
	RoomID    string         `json:"room_id"`
	Data      ChatMessageData `json:"data"`
}

// Update represents the message sent to display
type Update struct {
	Message ChatMessage `json:"message"`
	Lang    string     `json:"lang"`
}

type SSEClient chan string
var (
	clients    = make(map[SSEClient]bool)
	clientsMux sync.RWMutex
)

func main() {
	// Serve static files
	http.Handle("/", http.FileServer(http.Dir("static")))
	
	// SSE endpoint
	http.HandleFunc("/sse", handleSSE)
	
	// Update endpoint
	http.HandleFunc("/update", handleUpdate)

	// Redirect root to control
	http.HandleFunc("/control", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/control.html")
	})

	http.HandleFunc("/display", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/display.html")
	})

	http.HandleFunc("/tts", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/tts.html")
	})

	log.Printf("Server starting on http://localhost:7777")
	log.Fatal(http.ListenAndServe(":7777", nil))
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	client := make(SSEClient)
	
	clientsMux.Lock()
	clients[client] = true
	clientsMux.Unlock()

	defer func() {
		clientsMux.Lock()
		delete(clients, client)
		clientsMux.Unlock()
	}()

	notify := w.(http.CloseNotifier).CloseNotify()
	go func() {
		<-notify
		clientsMux.Lock()
		delete(clients, client)
		close(client)
		clientsMux.Unlock()
	}()

	for {
		msg, ok := <-client
		if !ok {
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", msg)
		w.(http.Flusher).Flush()
	}
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var update Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast to all SSE clients
	message, err := json.Marshal(update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	clientsMux.RLock()
	for client := range clients {
		client <- string(message)
	}
	clientsMux.RUnlock()

	w.WriteHeader(http.StatusOK)
} 