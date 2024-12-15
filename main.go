package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

// Constants
const (
	ServerPort = ":7777"
	AssetsDir  = "assets"
)

// Config holds server configuration
type Config struct {
	Port      string
	AssetsDir string
	Routes    map[string]string
}

// Server represents the application server
type Server struct {
	config  Config
	clients map[SSEClient]bool
	mu      sync.RWMutex
}

// NewServer creates and configures a new server instance
func NewServer() *Server {
	return &Server{
		config: Config{
			Port:      ServerPort,
			AssetsDir: AssetsDir,
			Routes: map[string]string{
				"/control": "/control.html",
				"/display": "/display.html",
				"/tts":     "/tts.html",
			},
		},
		clients: make(map[SSEClient]bool),
	}
}

// Start initializes and starts the server
func (s *Server) Start() error {
	s.setupRoutes()
	addr := fmt.Sprintf("http://localhost%s", s.config.Port)
	log.Printf("Server starting on %s", addr)
	return http.ListenAndServe(s.config.Port, nil)
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Serve static files
	http.Handle("/", http.FileServer(http.Dir(s.config.AssetsDir)))
	
	// Configure SSE and update endpoints
	http.HandleFunc("/sse", s.handleSSE)
	http.HandleFunc("/update", s.handleUpdate)
	
	// Configure page routes
	for route, file := range s.config.Routes {
		filePath := s.config.AssetsDir + file
		http.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filePath)
		})
	}
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	headers := map[string]string{
		"Content-Type":                "text/event-stream",
		"Cache-Control":               "no-cache",
		"Connection":                  "keep-alive",
		"Access-Control-Allow-Origin": "*",
	}
	
	for key, value := range headers {
		w.Header().Set(key, value)
	}

	client := make(SSEClient)
	
	s.addClient(client)

	notify := w.(http.CloseNotifier).CloseNotify()
	go func() {
		<-notify
		s.removeClient(client)
	}()

	for msg := range client {
		fmt.Fprintf(w, "data: %s\n\n", msg)
		w.(http.Flusher).Flush()
	}

	s.removeClient(client)
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var update Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.broadcastUpdate(update); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) broadcastUpdate(update Update) error {
	message, err := json.Marshal(update)
	if err != nil {
		return err
	}
	
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for client := range s.clients {
		client <- string(message)
	}
	return nil
}

func (s *Server) addClient(client SSEClient) {
	s.mu.Lock()
	s.clients[client] = true
	s.mu.Unlock()
}

func (s *Server) removeClient(client SSEClient) {
	s.mu.Lock()
	if _, exists := s.clients[client]; exists {
		delete(s.clients, client)
		close(client)
	}
	s.mu.Unlock()
}

func main() {
	server := NewServer()
	log.Fatal(server.Start())
}

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