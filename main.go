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
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/oristarium/orionchat/avatar" // Replace with your actual module name
	"github.com/oristarium/orionchat/tts"    // Replace with your actual module name
	"github.com/oristarium/orionchat/ui"     // Add this import

	"github.com/oristarium/orionchat/handlers"
	"go.etcd.io/bbolt"
)

// Constants
const (
	ServerPort    = ":7777"
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
	fileHandler *handlers.FileHandler
	ttsService *tts.TTSService
	avatarManager *avatar.Manager
	avatarHandler *handlers.AvatarHandler
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

	fileHandler := handlers.NewFileHandler(storage)

	server := &Server{
		config: Config{
			Port:      ServerPort,
			AssetsDir: avatar.AssetsDir,
			Routes: map[string]string{
				"/control":  "/control.html",
				"/display": "/display.html",
				"/tts":     "/tts.html",
				"/tutorial": "/tutorial.html",
			},
		},
		clients:       make(map[SSEClient]bool),
		fileHandler:   fileHandler,
		ttsService:    tts.NewTTSService(),
		avatarManager: avatarManager,
	}

	server.avatarHandler = handlers.NewAvatarHandler(
		server.avatarManager, 
		server.fileHandler,
		server,
	)

	return server
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
	http.HandleFunc("/api/avatars", s.avatarHandler.HandleAvatars)
	http.HandleFunc("/api/avatars/", s.avatarHandler.HandleAvatarDetail)
	http.HandleFunc("/api/avatars/active", s.avatarHandler.HandleActiveAvatars)
	http.HandleFunc("/api/avatars/create", s.avatarHandler.HandleCreateAvatar)
	http.HandleFunc("/api/avatar-images", s.avatarHandler.HandleAvatarImages)
	http.HandleFunc("/api/avatar-images/upload", s.avatarHandler.HandleAvatarImageUpload)
	http.HandleFunc("/api/avatar-images/delete/", s.avatarHandler.HandleAvatarImageDelete)
	http.HandleFunc("/api/avatar/upload", s.avatarHandler.HandleAvatarUpload)
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
	Type string     `json:"type"`
	Data UpdateData `json:"data"`
}

// UpdateData represents the data payload for different update types
type UpdateData struct {
	Path          string      `json:"path,omitempty"`
	AvatarType    string      `json:"avatar_type,omitempty"`
	Message       interface{} `json:"message,omitempty"`
	VoiceID       string      `json:"voice_id,omitempty"`
	VoiceProvider string      `json:"voice_provider,omitempty"`
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

// Add this method to Server struct
func (s *Server) BroadcastUpdate(update handlers.Update) error {
	return s.broadcastUpdate(Update{
		Type: update.Type,
		Data: UpdateData{
			Path:       update.Data.Path,
			AvatarType: update.Data.AvatarType,
			Message:    update.Data.Message,
			VoiceID:    update.Data.VoiceID,
			VoiceProvider: update.Data.VoiceProvider,
		},
	})
}