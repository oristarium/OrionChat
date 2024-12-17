//go:build !js

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/oristarium/orionchat/avatar" // Replace with your actual module name
	"github.com/oristarium/orionchat/tts"    // Replace with your actual module name
	"github.com/oristarium/orionchat/ui"     // Add this import

	"go.etcd.io/bbolt"
)

// Constants
const (
	ServerPort    = ":7777"
	AppName       = "OrionChat"
	AppIcon       = "assets/icon.ico"

	// Window dimensions
	WindowWidth   = 300
	WindowHeight  = 350

	// Text sizes
	DefaultTextSize = 14

	CopyLabel = "Copy %s URL 📋"
	CopiedLabel = "Copied! ✓"

	DonationLabel = "Support us on"
	DonationLink = "https://trakteer.id/oristarium"
	DonationText = "Trakteer ☕"

	// Layout dimensions
	StatusBottomMargin = 16
	ButtonSpacing = 60

	// Button labels

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
	fileHandler *FileHandler
	ttsService *tts.TTSService
	avatarManager *avatar.Manager
}

// NewServer creates and configures a new server instance
func NewServer() *Server {
	storage, err := NewBBoltStorage(avatar.DBPath)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize avatar manager
	avatarManager, err := avatar.NewManager(avatar.NewStorage(storage.db))
	if err != nil {
		log.Fatal(err)
	}

	return &Server{
		config: Config{
			Port:      ServerPort,
				AssetsDir: avatar.AssetsDir,
				Routes: map[string]string{
					"/control": "/control.html",
					"/display": "/display.html",
					"/tts":     "/tts.html",
					"/tutorial": "/tutorial.html",
				},
			},
			clients: make(map[SSEClient]bool),
			fileHandler: NewFileHandler(storage),
			ttsService: tts.NewTTSService(),
			avatarManager: avatarManager,
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
	// Add MIME types for JavaScript modules
	mime := map[string]string{
		".js":   "application/javascript",
		".mjs":  "application/javascript",
		".css":  "text/css",
		".html": "text/html",
	}
	
	// Serve static files directly (revert the /assets/ handling)
	fs := http.FileServer(http.Dir(s.config.AssetsDir))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Set correct MIME type based on file extension
		ext := filepath.Ext(r.URL.Path)
		if mimeType, ok := mime[ext]; ok {
			w.Header().Set("Content-Type", mimeType)
		}
		fs.ServeHTTP(w, r)
	})
	
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
	http.HandleFunc("/api/avatars", s.handleAvatars)
	http.HandleFunc("/api/avatars/", s.handleAvatarDetail)
	http.HandleFunc("/api/avatars/active", s.handleActiveAvatars)
	http.HandleFunc("/api/avatars/create", s.handleCreateAvatar)
	http.HandleFunc("/api/avatar-images", s.handleAvatarImages)
	http.HandleFunc("/api/avatar-images/upload", s.handleAvatarImageUpload)
	http.HandleFunc("/api/avatar-images/delete/", s.handleAvatarImageDelete)
	http.HandleFunc("/tts-service", s.handleTTS)
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
	// First save the file
	s.fileHandler.HandleUpload(w, r, struct {
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
			// Register the image in our database
			if err := s.avatarManager.RegisterAvatarImage(path); err != nil {
				return fmt.Errorf("register avatar image: %w", err)
			}

			update := Update{
				Type: "avatar_update",
				Data: UpdateData{
					AvatarType: r.FormValue("type"),
						Path: path,
				},
			}
			return s.broadcastUpdate(update)
		},
	})
}

func (s *Server) handleGetAvatars(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	currentAvatar, err := s.avatarManager.GetCurrentAvatar()
	if err != nil {
		// Return default paths if no avatar found
		json.NewEncoder(w).Encode(map[string]string{
			"idle":    avatar.DefaultIdleAvatar,
			"talking": avatar.DefaultTalkingAvatar,
		})
		return
	}

	response := map[string]string{
		"idle":    currentAvatar.States[avatar.StateIdle],
		"talking": currentAvatar.States[avatar.StateTalking],
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleListAvatarImages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get images from database
	images, err := s.avatarManager.Storage.ListAvatarImages()
	if err != nil {
		log.Printf("Error listing avatar images: %v", err)
		http.Error(w, "Failed to list avatar images", http.StatusInternalServerError)
		return
	}

	// Convert to paths array for backwards compatibility
	var paths []string
	for _, img := range images {
		paths = append(paths, img.Path)
	}

	json.NewEncoder(w).Encode(map[string][]string{
		"avatars": paths,
	})
}

func (s *Server) handleListAvatars(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	avatars, err := s.avatarManager.ListAvatars()
	if err != nil {
		log.Printf("Error listing avatars: %v", err)
		// Return default avatar instead of error
		defaultAvatar := avatar.Avatar{
			ID:          fmt.Sprintf("avatar_%d", time.Now().UnixNano()),
			Name:        "Default",
			Description: "Default avatar",
			States: map[avatar.AvatarState]string{
				avatar.StateIdle:    avatar.DefaultIdleAvatar,
				avatar.StateTalking: avatar.DefaultTalkingAvatar,
			},
			IsDefault: true,
			CreatedAt: time.Now().Unix(),
		}
		
		response := struct {
			Avatars   []avatar.Avatar `json:"avatars"`
			CurrentID string          `json:"current_id"`
		}{
			Avatars:   []avatar.Avatar{defaultAvatar},
			CurrentID: defaultAvatar.ID,
		}
		
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get current avatar for marking default
	currentAvatar, err := s.avatarManager.GetCurrentAvatar()
	if err != nil {
		log.Printf("Error getting current avatar: %v", err)
		// Use first avatar as current if none set
		if len(avatars) > 0 {
			currentAvatar = &avatars[0]
		}
	}

	// Convert to response format
	response := struct {
		Avatars   []avatar.Avatar `json:"avatars"`
		CurrentID string          `json:"current_id"`
	}{
		Avatars:   avatars,
		CurrentID: func() string {
			if currentAvatar != nil {
				return currentAvatar.ID
			}
			return avatars[0].ID
		}(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) handleSetAvatar(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received avatar update request - Method: %s", r.Method)

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Methods", "POST, PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight requests
	if r.Method == http.MethodOptions {
		log.Printf("Handling OPTIONS request")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		log.Printf("Invalid method: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Type string `json:"type"`
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Decoded request - Type: %s, Path: %s", request.Type, request.Path)

	// Get current avatar
	currentAvatar, err := s.avatarManager.GetCurrentAvatar()
	if err != nil {
		log.Printf("Failed to get current avatar: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update state
	state := avatar.AvatarState(request.Type)
	log.Printf("Updating avatar state - ID: %s, State: %s, Path: %s", 
		currentAvatar.ID, state, request.Path)

	if err := s.avatarManager.UpdateAvatarState(currentAvatar.ID, state, request.Path); err != nil {
		if err.Error() == "image not found" {
			log.Printf("Image not registered, attempting to register: %s", request.Path)
			// Try to register the image first
			if err := s.avatarManager.RegisterAvatarImage(request.Path); err != nil {
				log.Printf("Failed to register image: %v", err)
				http.Error(w, "Failed to register image", http.StatusInternalServerError)
				return
			}
			// Try update again
			if err := s.avatarManager.UpdateAvatarState(currentAvatar.ID, state, request.Path); err != nil {
				log.Printf("Failed to update avatar state after registration: %v", err)
				http.Error(w, "Failed to save avatar setting", http.StatusInternalServerError)
				return
			}
		} else {
			log.Printf("Failed to update avatar state: %v", err)
			http.Error(w, "Failed to save avatar setting", http.StatusInternalServerError)
			return
		}
	}

	log.Printf("Successfully updated avatar state")

	// Broadcast update
	update := Update{
		Type: "avatar_update",
		Data: UpdateData{
			AvatarType: string(state),
			Path: request.Path,
		},
	}

	if err := s.broadcastUpdate(update); err != nil {
		log.Printf("Failed to broadcast update: %v", err)
		http.Error(w, "Failed to broadcast update", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully broadcasted update")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleTTS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Text    string `json:"text"`
		VoiceID string `json:"voice_id"`
		VoiceProvider string `json:"voice_provider"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("TTS Request - Text: %q, VoiceID: %q, Provider: %q", request.Text, request.VoiceID, request.VoiceProvider)

	// Get the requested provider
	provider, err := tts.GetProvider(request.VoiceProvider)
	if err != nil {
		log.Printf("Provider error: %v", err)
		http.Error(w, fmt.Sprintf("TTS provider error: %v", err), http.StatusBadRequest)
		return
	}

	// Split long text into chunks if needed
	chunks, err := s.ttsService.SplitLongText(request.Text, "")
	if err != nil {
		log.Printf("Text splitting error: %v", err)
		http.Error(w, fmt.Sprintf("Text splitting error: %v", err), http.StatusBadRequest)
		return
	}

	// If text is short enough to not need splitting, put it in a single chunk
	if len(chunks) == 0 {
		chunks = []string{request.Text}
	}

	log.Printf("Split into %d chunks: %v", len(chunks), chunks)

	// Get audio for all chunks and combine them
	var combinedAudio string
	for _, chunk := range chunks {
		audio, err := s.ttsService.GetAudioBase64WithProvider(chunk, request.VoiceID, provider, false)
		if err != nil {
			log.Printf("TTS error for chunk %q: %v", chunk, err)
			http.Error(w, fmt.Sprintf("TTS error: %v", err), http.StatusInternalServerError)
			return
		}
		if combinedAudio == "" {
			combinedAudio = audio
		} else {
			combinedAudio += audio
		}
		log.Printf("Successfully processed chunk: %q", chunk)
	}

	log.Printf("Successfully combined %d audio chunks", len(chunks))

	response := map[string]string{
		"audio": combinedAudio,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
	log.Printf("Successfully sent response with audio length: %d", len(combinedAudio))
}

func main() {
	// Create channels to coordinate shutdown
	shutdown := make(chan bool)
	serverStarted := make(chan bool)
	var wg sync.WaitGroup

	// Create HTTP server
	server := NewServer()
	httpServer := &http.Server{
		Addr:    ServerPort,
		Handler: http.DefaultServeMux,
	}

	// Start server in a separate goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		server.setupRoutes()
		
		// Signal that we're about to start the server
		serverStarted <- true
		
		// Start the server
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
			close(shutdown)
			return
		}
	}()

	// Handle shutdown
	go func() {
		<-shutdown
		
		// Shutdown the HTTP server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
		
		wg.Wait()
	}()

	// Start the UI
	ui.RunUI(ServerPort, shutdown, serverStarted, &wg)
}

// Helper function to safely parse URLs
func parseURL(urlStr string) *url.URL {
	url, err := url.Parse(urlStr)
	if err != nil {
		log.Printf("Error parsing URL: %v", err)
		return nil
	}
	return url
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
	Data    UpdateData             `json:"data"`
}

// UpdateData represents the data payload for different update types
type UpdateData struct {
	Message  *ChatMessage         `json:"message,omitempty"`
	VoiceID  string              `json:"voice_id,omitempty"`
	VoiceProvider string         `json:"voice_provider,omitempty"`
	Path     string              `json:"path,omitempty"`
	AvatarType string            `json:"avatar_type,omitempty"`
}

type SSEClient chan string

// FileStorage interface defines methods for file operations
type FileStorage interface {
	Upload(file *multipart.File, filename string, directory string) (string, error)
	Get(key string, bucket string) (string, error)
	Save(key string, value string, bucket string) error
}

// BBoltStorage implements FileStorage using BBolt
type BBoltStorage struct {
	db *bbolt.DB
}

func NewBBoltStorage(dbPath string) (*BBoltStorage, error) {
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, err
	}

	// Initialize required buckets
	err = db.Update(func(tx *bbolt.Tx) error {
		// Create avatar bucket
		if _, err := tx.CreateBucketIfNotExists([]byte(avatar.AvatarBucket)); err != nil {
			return fmt.Errorf("create avatar bucket: %w", err)
		}
		
		// Create images bucket
		if _, err := tx.CreateBucketIfNotExists([]byte(avatar.ImagesBucket)); err != nil {
			return fmt.Errorf("create images bucket: %w", err)
		}
		
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("initialize buckets: %w", err)
	}

	return &BBoltStorage{db: db}, nil
}

func (s *BBoltStorage) Upload(file *multipart.File, filename string, directory string) (string, error) {
	filepath := filepath.Join(avatar.AssetsDir, directory, filename)
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(directory, 0755); err != nil {
		return "", err
	}

	dst, err := os.Create(filepath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, *file); err != nil {
		return "", err
	}

	return fmt.Sprintf("/%s/%s", directory, filename), nil
}

func (s *BBoltStorage) Get(key string, bucket string) (string, error) {
	var value string
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		value = string(b.Get([]byte(key)))
		return nil
	})
	return value, err
}

func (s *BBoltStorage) Save(key string, value string, bucket string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}
		return b.Put([]byte(key), []byte(value))
	})
}

// FileHandler handles file-related operations
type FileHandler struct {
	storage FileStorage
}

func NewFileHandler(storage FileStorage) *FileHandler {
	return &FileHandler{storage: storage}
}

func (h *FileHandler) HandleUpload(w http.ResponseWriter, r *http.Request, options struct {
	FileField  string
	TypeField  string
	Directory  string
	Bucket     string
	KeyPrefix  string
	OnSuccess  func(string) error
}) {
	log.Printf("Upload request received for %s", options.Directory)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		log.Printf("Form parse error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile(options.FileField)
	if err != nil {
		log.Printf("File retrieval error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	itemType := r.FormValue(options.TypeField)
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), filepath.Ext(handler.Filename))

	path, err := h.storage.Upload(&file, filename, options.Directory)
	if err != nil {
		log.Printf("File upload error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	key := fmt.Sprintf("%s_%s", options.KeyPrefix, itemType)
	if err := h.storage.Save(key, path, options.Bucket); err != nil {
		log.Printf("Storage save error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if options.OnSuccess != nil {
		if err := options.OnSuccess(path); err != nil {
			log.Printf("OnSuccess callback error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(w).Encode(map[string]string{"path": path})
}

//go:generate goversioninfo -icon=AppIcon -manifest=manifest.xml

// handleAvatars handles GET /api/avatars
func (s *Server) handleAvatars(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	avatars, err := s.avatarManager.ListAvatars()
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
		
		response := struct {
			Avatars   []avatar.Avatar `json:"avatars"`
			CurrentID string          `json:"current_id"`
		}{
			Avatars:   []avatar.Avatar{defaultAvatar},
			CurrentID: defaultAvatar.ID,
		}
		
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get current avatar for marking default
	currentAvatar, err := s.avatarManager.GetCurrentAvatar()
	if err != nil {
		log.Printf("Error getting current avatar: %v", err)
		if len(avatars) > 0 {
			currentAvatar = &avatars[0]
		}
	}

	response := struct {
		Avatars   []avatar.Avatar `json:"avatars"`
		CurrentID string          `json:"current_id"`
	}{
		Avatars:   avatars,
		CurrentID: func() string {
			if currentAvatar != nil {
				return currentAvatar.ID
			}
			return avatars[0].ID
		}(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleAvatarDetail handles /api/avatars/{id}/(get|set)
func (s *Server) handleAvatarDetail(w http.ResponseWriter, r *http.Request) {
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
		s.handleGetAvatar(w, r, id)
	case "set":
		if r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleSetAvatarDetail(w, r, id)
	case "delete":
		s.handleDeleteAvatar(w, r, id)
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
	}
}

// handleAvatarImages handles GET /api/avatar-images
func (s *Server) handleAvatarImages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	// Read directly from avatars directory
	avatarDir := filepath.Join(avatar.AssetsDir, avatar.AvatarAssetsDir)
	files, err := os.ReadDir(avatarDir)
	if err != nil {
		log.Printf("Error reading avatar directory: %v", err)
		// Return default images instead of error
		defaultImages := []string{
			fmt.Sprintf("/%s/idle.png", avatar.AvatarAssetsDir),
			fmt.Sprintf("/%s/talking.gif", avatar.AvatarAssetsDir),
		}
		json.NewEncoder(w).Encode(map[string][]string{
			"avatar-images": defaultImages,
		})
		return
	}

	// Convert to paths array for backwards compatibility
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

// handleAvatarImageUpload handles POST /api/avatar-images/upload
func (s *Server) handleAvatarImageUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Handle file upload and registration
	s.fileHandler.HandleUpload(w, r, struct {
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
			return s.avatarManager.RegisterAvatarImage(path)
		},
	})
}

// handleGetAvatar handles GET /api/avatars/{id}/get
func (s *Server) handleGetAvatar(w http.ResponseWriter, r *http.Request, id string) {
	avatar, err := s.avatarManager.Storage.GetAvatar(id)
	if err != nil {
		log.Printf("Error getting avatar: %v", err)
		http.Error(w, "Avatar not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(avatar)
}

// handleSetAvatarDetail handles PUT /api/avatars/{id}/set
func (s *Server) handleSetAvatarDetail(w http.ResponseWriter, r *http.Request, id string) {
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
	existingAvatar, err := s.avatarManager.Storage.GetAvatar(id)
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
	if err := s.avatarManager.Storage.SaveAvatar(existingAvatar); err != nil {
		http.Error(w, "Failed to save avatar", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleActiveAvatars handles GET /api/avatars/active
func (s *Server) handleActiveAvatars(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	avatars, err := s.avatarManager.GetActiveAvatars()
	if err != nil {
		log.Printf("Error getting active avatars: %v", err)
		http.Error(w, "Failed to get active avatars", http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(map[string][]avatar.Avatar{
		"avatars": avatars,
	}); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleAvatarImageDelete handles DELETE /api/avatar-images/delete/{path}
func (s *Server) handleAvatarImageDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract path from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/avatar-images/delete/")
	if path == "" {
		http.Error(w, "Path is required", http.StatusBadRequest)
		return
	}

	// Delete the image
	if err := s.avatarManager.DeleteAvatarImage(path); err != nil {
		log.Printf("Error deleting avatar image: %v", err)
		http.Error(w, "Failed to delete avatar image", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleDeleteAvatar handles DELETE /api/avatars/{id}
func (s *Server) handleDeleteAvatar(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.avatarManager.DeleteAvatar(id); err != nil {
		log.Printf("Error deleting avatar: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleCreateAvatar handles POST /api/avatars/create
func (s *Server) handleCreateAvatar(w http.ResponseWriter, r *http.Request) {
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

	if err := s.avatarManager.Storage.SaveAvatar(newAvatar); err != nil {
		log.Printf("Error saving new avatar: %v", err)
		http.Error(w, "Failed to create avatar", http.StatusInternalServerError)
		return
	}

	// Update config to include new avatar
	config, err := s.avatarManager.Storage.GetConfig()
	if err != nil {
		log.Printf("Error getting config: %v", err)
		http.Error(w, "Failed to update config", http.StatusInternalServerError)
		return
	}

	config.Avatars = append(config.Avatars, newAvatar)
	if err := s.avatarManager.Storage.SaveConfig(config); err != nil {
		log.Printf("Error saving config: %v", err)
		http.Error(w, "Failed to update config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newAvatar)
}