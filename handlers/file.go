package handlers

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"
)

// FileStorage interface defines methods for file operations
type FileStorage interface {
	Upload(file *multipart.File, filename string, directory string) (string, error)
	Save(key string, value string, bucket string) error
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
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile(options.FileField)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	itemType := r.FormValue(options.TypeField)
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), filepath.Ext(handler.Filename))

	path, err := h.storage.Upload(&file, filename, options.Directory)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	key := fmt.Sprintf("%s_%s", options.KeyPrefix, itemType)
	if err := h.storage.Save(key, path, options.Bucket); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if options.OnSuccess != nil {
		if err := options.OnSuccess(path); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(w).Encode(map[string]string{"path": path})
} 