package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/oristarium/orionchat/avatar" // Replace with your actual module name
	"github.com/oristarium/orionchat/tts"    // Replace with your actual module name
	"github.com/oristarium/orionchat/ui"     // Add this import

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
	ttsHandler *tts.TTSHandler
	avatarManager *avatar.Manager
	avatarHandler *handlers.AvatarHandler
	broadcaster *broadcast.Broadcaster
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

	server := &Server{
		config: types.Config{
			Port:      ServerPort,
			AssetsDir: avatar.AssetsDir,
			Routes: map[string]string{
				"/control":  "/control.html",
				"/display": "/display.html",
				"/tts":     "/tts.html",
				"/tutorial": "/tutorial.html",
			},
		},
		fileHandler:   fileHandler,
		ttsHandler:    tts.NewTTSHandler(ttsService),
		avatarManager: avatarManager,
		broadcaster: broadcast.New(),
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
	http.HandleFunc("/api/avatars/active", s.avatarHandler.HandleActiveAvatars)
	http.HandleFunc("/api/avatars/create", s.avatarHandler.HandleCreateAvatar)
	http.HandleFunc("/api/avatar-images", s.avatarHandler.HandleAvatarImages)
	http.HandleFunc("/api/avatar-images/upload", s.avatarHandler.HandleAvatarImageUpload)
	http.HandleFunc("/api/avatar-images/delete/", s.avatarHandler.HandleAvatarImageDelete)
	http.HandleFunc("/api/avatar/upload", s.avatarHandler.HandleAvatarUpload)
	http.HandleFunc("/tts-service", s.ttsHandler.HandleTTS)
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