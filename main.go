package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/oristarium/orionchat/avatar"
	"github.com/oristarium/orionchat/tts"
	"github.com/oristarium/orionchat/ui"

	"github.com/oristarium/orionchat/broadcast"
	"github.com/oristarium/orionchat/handlers"
	"github.com/oristarium/orionchat/storage"
	"github.com/oristarium/orionchat/types"
)

// Constants
const (
	ServerPort    = ":7777"
)

// Server represents the application server
type Server struct {
	config  types.Config
	fileHandler *handlers.FileHandler
	ttsHandler *handlers.TTSHandler
	avatarManager *avatar.Manager
	avatarHandler *handlers.AvatarHandler
	broadcaster *broadcast.Broadcaster
	ttsMiddleware *tts.TTSMiddleware
}

// NewServer creates and configures a new server instance
func NewServer() *Server {
	store, err := storage.NewBBoltStorage(avatar.DBPath)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize avatar manager
	avatarManager, err := avatar.NewManager(avatar.NewStorage(store.GetDB()))
	if err != nil {
		log.Fatal(err)
	}

	fileHandler := handlers.NewFileHandler(store)

	ttsService := tts.NewTTSService()
	ttsMiddleware := tts.NewTTSMiddleware()

	server := &Server{
		config: types.Config{
			Port:      ServerPort,
			AssetsDir: avatar.AssetsDir,
			Routes: map[string]string{
				"/control":  "/control.html",
				"/display": "/display.html",
				"/tutorial": "/tutorial.html",
			},
		},
		fileHandler:   fileHandler,
		ttsHandler:    handlers.NewTTSHandler(ttsService),
		avatarManager: avatarManager,
		broadcaster:   broadcast.New(),
		ttsMiddleware: ttsMiddleware,
	}

	server.avatarHandler = handlers.NewAvatarHandler(
		server.avatarManager, 
		server.fileHandler,
		server,
	)

	// Connect the TTS middleware to the broadcaster
	server.broadcaster.SetTTSMiddleware(server.ttsMiddleware)

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
	http.HandleFunc("/sse", s.broadcaster.HandleSSE)
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
	http.HandleFunc("/api/avatars/create", s.avatarHandler.HandleCreateAvatar)
	http.HandleFunc("/api/avatar-images", s.avatarHandler.HandleAvatarImages)
	http.HandleFunc("/api/avatar-images/upload", s.avatarHandler.HandleAvatarImageUpload)
	http.HandleFunc("/api/avatar-images/delete", s.avatarHandler.HandleAvatarImageDelete)
	http.HandleFunc("/api/avatar/upload", s.avatarHandler.HandleAvatarUpload)
	http.HandleFunc("/tts-service", s.ttsHandler.HandleTTS)
	http.HandleFunc("/api/kv/", s.handleKeyValue)

	// Add WebSocket endpoint for TTS
	http.HandleFunc("/ws/tts", s.ttsMiddleware.HandleWebSocket)

	// Handle TTS individual pages with ID parameter
	http.HandleFunc("/avatar/", func(w http.ResponseWriter, r *http.Request) {
		avatarId := strings.TrimPrefix(r.URL.Path, "/avatar/")
		if avatarId == "" {
			http.Redirect(w, r, "/avatar-index.html", http.StatusFound)
			return
		}
		serveAvatarPage(w, r, avatarId)
	})
	http.HandleFunc("/avatar-index.html", serveFile("assets/avatar-index.html"))

	// Handle TTS audio blobs
	http.HandleFunc("/tts-blob/", func(w http.ResponseWriter, r *http.Request) {
		blobName := filepath.Base(r.URL.Path)
		blobPath := filepath.Join(os.TempDir(), "tts_blobs", blobName)

		// Check if file exists
		if _, err := os.Stat(blobPath); os.IsNotExist(err) {
			http.Error(w, "Audio not found", http.StatusNotFound)
			return
		}

		// Open and validate the file
		file, err := os.Open(blobPath)
		if err != nil {
			http.Error(w, "Failed to read audio file", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		// Set proper headers
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		
		// Serve the file
		http.ServeFile(w, r, blobPath)
	})
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var update broadcast.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.broadcaster.Broadcast(update); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) Broadcast(update broadcast.Update) error {
	return s.broadcaster.Broadcast(update)
}

func (s *Server) handleKeyValue(w http.ResponseWriter, r *http.Request) {
	store := s.fileHandler.GetStorage()
	if store == nil {
		http.Error(w, "Storage not available", http.StatusInternalServerError)
		return
	}

	key := r.URL.Path[len("/api/kv/"):]
	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		value, err := store.Get(key, storage.GeneralBucket)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if value == "" {
			http.Error(w, "Key not found", http.StatusNotFound)
			return
		}

		// Try to parse as JSON first
		var jsonData interface{}
		if err := json.Unmarshal([]byte(value), &jsonData); err == nil {
			// If it parses as JSON, return it as JSON
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(jsonData)
		} else {
			// If it's not JSON, return as plain text
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(value))
		}

	case http.MethodPut:
		contentType := r.Header.Get("Content-Type")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// If content type is JSON, validate it
		if contentType == "application/json" {
			var jsonData interface{}
			if err := json.Unmarshal(body, &jsonData); err != nil {
				http.Error(w, "Invalid JSON format", http.StatusBadRequest)
				return
			}
		}

		// Store the raw value regardless of type
		err = store.Save(key, string(body), storage.GeneralBucket)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Helper function to serve static files
func serveFile(filepath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		content, err := os.ReadFile(filepath)
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		
		// Set content type based on file extension
		ext := path.Ext(filepath)
		switch ext {
		case ".html":
			w.Header().Set("Content-Type", "text/html")
		case ".js":
			w.Header().Set("Content-Type", "application/javascript")
		case ".css":
			w.Header().Set("Content-Type", "text/css")
		}
		
		w.Write(content)
	}
}

// Helper function to serve avatar pages with ID replacement
func serveAvatarPage(w http.ResponseWriter, r *http.Request, avatarId string) {
	// Read the template file
	filePath := "assets/avatar.html"
	content, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	// Replace placeholder with actual avatarId
	modifiedContent := strings.Replace(
		string(content),
		"__TTS_ID__",
		avatarId,
		-1,
	)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(modifiedContent))
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