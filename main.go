package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.etcd.io/bbolt"
)

// Constants
const (
	ServerPort = ":7777"
	AssetsDir  = "assets"
	DBPath = "avatars.db"
	AvatarBucket = "avatars"
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
	db *bbolt.DB
}

// NewServer creates and configures a new server instance
func NewServer() *Server {
	db, err := bbolt.Open(DBPath, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize default avatars in DB
	err = db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(AvatarBucket))
		if err != nil {
			return err
		}

		// Set defaults if not exists
		if b.Get([]byte("avatar_idle")) == nil {
			b.Put([]byte("avatar_idle"), []byte("/avatars/idle.png"))
		}
		if b.Get([]byte("avatar_talking")) == nil {
			b.Put([]byte("avatar_talking"), []byte("/avatars/talking.gif"))
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

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
		db: db,
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
	http.HandleFunc("/upload-avatar", s.handleAvatarUpload)
	http.HandleFunc("/get-avatars", s.handleGetAvatars)
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
	log.Println("Starting broadcast...")
	message, err := json.Marshal(update)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return err
	}
	
	log.Printf("Broadcasting message: %s", string(message))
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for client := range s.clients {
		client <- string(message)
	}
	log.Printf("Broadcast completed to %d clients", len(s.clients))
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

func (s *Server) handleAvatarUpload(w http.ResponseWriter, r *http.Request) {
	log.Printf("Avatar upload request received: %s", r.Method)
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	log.Println("Parsing multipart form...")
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		log.Printf("Form parse error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Println("Getting file from form...")
	file, handler, err := r.FormFile("avatar")
	if err != nil {
		log.Printf("File retrieval error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	avatarType := r.FormValue("type")
	log.Printf("Avatar type: %s", avatarType)
	if avatarType != "idle" && avatarType != "talking" {
		log.Printf("Invalid avatar type: %s", avatarType)
		http.Error(w, "Invalid avatar type", http.StatusBadRequest)
		return
	}

	// Generate filename with timestamp
	ext := filepath.Ext(handler.Filename)
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	filepath := filepath.Join("assets", "avatars", filename)
	log.Printf("Generated filepath: %s", filepath)

	// Create file
	log.Println("Creating destination file...")
	dst, err := os.Create(filepath)
	if err != nil {
		log.Printf("File creation error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy file contents
	log.Println("Copying file contents...")
	if _, err = io.Copy(dst, file); err != nil {
		log.Printf("File copy error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update DB
	dbKey := fmt.Sprintf("avatar_%s", avatarType)
	dbValue := fmt.Sprintf("/avatars/%s", filename)
	log.Printf("Updating DB - Key: %s, Value: %s", dbKey, dbValue)
	
	err = s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(AvatarBucket))
		return b.Put([]byte(dbKey), []byte(dbValue))
	})

	if err != nil {
		log.Printf("DB update error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast update to TTS clients
	log.Println("Preparing broadcast update...")
	update := Update{
		Type: "avatar_update",
		Data: map[string]string{
			"type": avatarType,
			"path": dbValue,
		},
		Message: ChatMessage{
			Type: "avatar_update",
			Data: ChatMessageData{
				Content: ChatContent{
					Raw: dbValue,
				},
			},
		},
		Lang: "en",
	}
	log.Printf("Broadcasting update: %+v", update)
	s.broadcastUpdate(update)

	// Return success response
	log.Println("Sending success response...")
	json.NewEncoder(w).Encode(map[string]string{
		"path": dbValue,
	})
	log.Println("Avatar upload completed successfully")
}

func (s *Server) handleGetAvatars(w http.ResponseWriter, r *http.Request) {
	avatars := make(map[string]string)
	
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(AvatarBucket))
		avatars["idle"] = string(b.Get([]byte("avatar_idle")))
		avatars["talking"] = string(b.Get([]byte("avatar_talking")))
		return nil
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(avatars)
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
	Type    string                 `json:"type"`
	Data    map[string]string      `json:"data,omitempty"`
	Message ChatMessage            `json:"message"`
	Lang    string                `json:"lang"`
}

type SSEClient chan string