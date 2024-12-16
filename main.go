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
	"sync"
	"time"

	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/oristarium/orionchat/tts" // Replace with your actual module name

	"go.etcd.io/bbolt"
)

// Constants
const (
	ServerPort    = ":7777"
	AssetsDir     = "assets"
	DBPath        = "avatars.db"
	AvatarBucket  = "avatars"
	AppName       = "OrionChat"
	AppIcon       = "assets/icon.ico"

	// Window dimensions
	WindowWidth   = 300
	WindowHeight  = 350

	// Text sizes
	DefaultTextSize = 14

	// Avatar keys
	AvatarIdleKey    = "avatar_idle"
	AvatarTalkingKey = "avatar_talking"
	
	// Default avatar paths
	DefaultIdleAvatar    = "/avatars/idle.png"
	DefaultTalkingAvatar = "/avatars/talking.gif"

	// Donation info
	DonationLabel = "Support us on"
	DonationLink = "https://trakteer.id/oristarium"
	DonationText = "Trakteer ☕"

	// Layout dimensions
	StatusBottomMargin = 16
	ButtonSpacing = 60

	// Button labels
	CopyLabel = "Copy %s URL 📋"
	CopiedLabel = "Copied! ✓"
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
}

// NewServer creates and configures a new server instance
func NewServer() *Server {
	storage, err := NewBBoltStorage(DBPath)
	if err != nil {
		log.Fatal(err)
	}

	// Set default avatars
	err = storage.Save(AvatarIdleKey, DefaultIdleAvatar, AvatarBucket)
	if err != nil {
		log.Fatal(err)
	}

	err = storage.Save(AvatarTalkingKey, DefaultTalkingAvatar, AvatarBucket)
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
				"/tutorial": "/tutorial.html",
			},
		},
		clients: make(map[SSEClient]bool),
		fileHandler: NewFileHandler(storage),
		ttsService: tts.NewTTSService(),
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
	http.HandleFunc("/upload-avatar", s.handleAvatarUpload)
	http.HandleFunc("/get-avatars", s.handleGetAvatars)
	http.HandleFunc("/list-avatars", s.handleListAvatars)
	http.HandleFunc("/set-avatar", s.handleSetAvatar)
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
				VoiceID: "en",
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
		VoiceID: "en",
	}

	if err := s.broadcastUpdate(update); err != nil {
		http.Error(w, "Failed to broadcast update", http.StatusInternalServerError)
		return
	}

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
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("TTS Request - Text: %q, VoiceID: %q", request.Text, request.VoiceID)

	// Set default voice ID if empty
	if request.VoiceID == "" {
		request.VoiceID = "en"
		log.Printf("Using default voice ID: %q", request.VoiceID)
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
		audio, err := s.ttsService.GetAudioBase64(chunk, request.VoiceID, false)
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

	// Create Fyne application in the main thread
	fyneApp := app.New()
	
	// Set application icon
	icon, err := fyne.LoadResourceFromPath(AppIcon)
	if err != nil {
		log.Printf("Failed to load icon: %v", err)
	} else {
		fyneApp.SetIcon(icon)
	}
	
	window := fyneApp.NewWindow(AppName)
	
	// Disable window maximizing
	window.SetFixedSize(true)

	// Create and configure the large icon image
	iconBig := canvas.NewImageFromFile("assets/icon-big.png")
	iconBig.SetMinSize(fyne.NewSize(100, 100))
	iconBig.Resize(fyne.NewSize(100, 100))
	iconBigContainer := container.NewCenter(iconBig)

	// Create status text with initial color
	status := canvas.NewText("Starting server...", color.Gray{Y: 200})
	status.TextStyle = fyne.TextStyle{Bold: true}
	status.TextSize = DefaultTextSize
	statusContainer := container.NewVBox(
		container.NewCenter(status),
		container.NewHBox(layout.NewSpacer()), // Add bottom margin
	)
	statusContainer.Resize(fyne.NewSize(0, StatusBottomMargin))

	// Create copy buttons with feedback
	createCopyButton := func(label, url string) *widget.Button {
		btn := widget.NewButton(fmt.Sprintf(CopyLabel, label), nil)
		btn.OnTapped = func() {
			window.Clipboard().SetContent(fmt.Sprintf("http://localhost%s%s", ServerPort, url))
			originalText := btn.Text
			btn.SetText(CopiedLabel)
			
			// Reset text after 2 seconds
			go func() {
				time.Sleep(2 * time.Second)
				btn.SetText(originalText)
			}()
		}
		return btn
	}

	copyControlBtn := createCopyButton("Control", "/control")
	copyTTSBtn := createCopyButton("TTS", "/tts")
	copyDisplayBtn := createCopyButton("Display", "/display")

	// Create buttons container
	buttonsContainer := container.NewVBox(
		copyControlBtn,
		copyTTSBtn,
		copyDisplayBtn,
	)

	// Create tutorial button
	tutorialBtn := widget.NewButton("How to use?", func() {
		url := fmt.Sprintf("http://localhost%s/tutorial", ServerPort)
		if err := fyne.CurrentApp().OpenURL(parseURL(url)); err != nil {
			log.Printf("Failed to open tutorial: %v", err)
		}
	})

	// Create quit button with red text
	quitBtn := widget.NewButton("Quit Application", func() {
		window.Close()
	})
	quitBtn.Importance = widget.DangerImportance  // This will make the button text red

	// Create footer with donation link and copyright
	footerText := canvas.NewText(DonationLabel, color.Gray{Y: 200})
	footerText.TextSize = DefaultTextSize
	footerLink := widget.NewHyperlink(DonationText, parseURL(DonationLink))
	
	// Add copyright notice with smaller text
	copyrightText := canvas.NewText("© 2024 Oristarium. All rights reserved.", color.Gray{Y: 150})
	copyrightText.TextSize = DefaultTextSize - 4  // Smaller text size
	
	footerContainer := container.NewVBox(
		container.NewHBox(
			footerText,
			container.New(&spacingLayout{}, footerLink),
		),
		container.NewCenter(copyrightText),
	)
	footer := container.NewCenter(footerContainer)

	// Layout
	content := container.NewVBox(
		iconBigContainer,  // Add the icon container first
		statusContainer,
		layout.NewSpacer(), // Add margin after status
		buttonsContainer,
		container.NewHBox(layout.NewSpacer()), // Add 60px spacer
		container.NewVBox(
			tutorialBtn,
			quitBtn,
			footer,
		),
	)

	// Set fixed height for the spacer
	content.Objects[3].Resize(fyne.NewSize(0, ButtonSpacing))  // Set height to 60px

	// Add padding around the content
	paddedContent := container.NewPadded(content)
	window.SetContent(paddedContent)

	// Handle window closing
	window.SetCloseIntercept(func() {
		status.Text = "Shutting down..."
		close(shutdown)
	})

	// Set window size
	window.Resize(fyne.NewSize(WindowWidth, WindowHeight))

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

	// Update status when server starts
	go func() {
		select {
		case <-serverStarted:
			status.Color = color.RGBA{G: 180, A: 255}  // Green color
			status.Text = fmt.Sprintf("Server running on port %s", ServerPort)
			status.Refresh()
		case <-shutdown:
			status.Color = color.RGBA{R: 180, A: 255}  // Red color
			status.Text = "Shutting down..."
			status.Refresh()
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
		fyneApp.Quit()
	}()

	// Run the GUI in the main thread
	window.ShowAndRun()
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
	Data    map[string]string      `json:"data,omitempty"`
	Message ChatMessage            `json:"message"`
	VoiceID string                `json:"voice_id"`
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

type spacingLayout struct{}

func (s *spacingLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return objects[0].MinSize()
}

func (s *spacingLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	objects[0].Resize(objects[0].MinSize())
	objects[0].Move(fyne.NewPos(0, 0)) // No left padding
}

//go:generate goversioninfo -icon=AppIcon -manifest=manifest.xml