package main

import (
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
	fileHandler *FileHandler
}

// NewServer creates and configures a new server instance
func NewServer() *Server {
	storage, err := NewBBoltStorage(DBPath)
	if err != nil {
		log.Fatal(err)
	}

	// Set default avatars
	err = storage.Save("avatar_idle", "/avatars/idle.png", AvatarBucket)
	if err != nil {
		log.Fatal(err)
	}

	err = storage.Save("avatar_talking", "/avatars/talking.gif", AvatarBucket)
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
		fileHandler: NewFileHandler(storage),
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
	http.HandleFunc("/list-avatars", s.handleListAvatars)
	http.HandleFunc("/set-avatar", s.handleSetAvatar)
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
		Directory:  "avatars",
		Bucket:     AvatarBucket,
		KeyPrefix:  "avatar",
		OnSuccess: func(path string) error {
			update := Update{
				Type: "avatar_update",
				Data: map[string]string{
					"type": r.FormValue("type"),
					"path": path,
				},
				Message: ChatMessage{
					Type: "avatar_update",
					Data: ChatMessageData{
						Content: ChatContent{
							Raw: path,
						},
					},
				},
				Lang: "en",
			}
			return s.broadcastUpdate(update)
		},
	})
}

func (s *Server) handleGetAvatars(w http.ResponseWriter, r *http.Request) {
	avatars := make(map[string]string)
	
	idle, err := s.fileHandler.storage.Get("avatar_idle", AvatarBucket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	avatars["idle"] = idle

	talking, err := s.fileHandler.storage.Get("avatar_talking", AvatarBucket)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	avatars["talking"] = talking

	json.NewEncoder(w).Encode(avatars)
}

func (s *Server) handleListAvatars(w http.ResponseWriter, r *http.Request) {
	avatarDir := filepath.Join(AssetsDir, "avatars")
	files, err := os.ReadDir(avatarDir)
	if err != nil {
		log.Printf("Error reading avatar directory: %v", err)
		http.Error(w, "Failed to read avatars", http.StatusInternalServerError)
		return
	}

	var avatars []string
	for _, file := range files {
		if !file.IsDir() {
			avatars = append(avatars, fmt.Sprintf("/avatars/%s", file.Name()))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]string{
		"avatars": avatars,
	})
}

func (s *Server) handleSetAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Type string `json:"type"`  // idle or talking
		Path string `json:"path"`  // path to existing avatar
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.Type != "idle" && request.Type != "talking" {
		http.Error(w, "Invalid avatar type", http.StatusBadRequest)
		return
	}

	// Save to DB
	key := fmt.Sprintf("avatar_%s", request.Type)
	if err := s.fileHandler.storage.Save(key, request.Path, AvatarBucket); err != nil {
		http.Error(w, "Failed to save avatar setting", http.StatusInternalServerError)
		return
	}

	// Broadcast update
	update := Update{
		Type: "avatar_update",
		Data: map[string]string{
			"type": request.Type,
			"path": request.Path,
		},
		Message: ChatMessage{
			Type: "avatar_update",
			Data: ChatMessageData{
				Content: ChatContent{
					Raw: request.Path,
				},
			},
		},
		Lang: "en",
	}

	if err := s.broadcastUpdate(update); err != nil {
		http.Error(w, "Failed to broadcast update", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
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

	return &BBoltStorage{db: db}, nil
}

func (s *BBoltStorage) Upload(file *multipart.File, filename string, directory string) (string, error) {
	filepath := filepath.Join(AssetsDir, directory, filename)
	
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