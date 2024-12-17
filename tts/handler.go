package tts

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// TTSHandler handles HTTP requests for TTS operations
type TTSHandler struct {
	service *TTSService
}

// NewTTSHandler creates a new TTS handler
func NewTTSHandler(service *TTSService) *TTSHandler {
	return &TTSHandler{
		service: service,
	}
}

// HandleTTS handles POST /tts-service requests
func (h *TTSHandler) HandleTTS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Text          string `json:"text"`
		VoiceID       string `json:"voice_id"`
		VoiceProvider string `json:"voice_provider"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("TTS Request - Text: %q, VoiceID: %q, Provider: %q", request.Text, request.VoiceID, request.VoiceProvider)

	// Get the requested provider
	provider, err := GetProvider(request.VoiceProvider)
	if err != nil {
		log.Printf("Provider error: %v", err)
		http.Error(w, fmt.Sprintf("TTS provider error: %v", err), http.StatusBadRequest)
		return
	}

	// Split long text into chunks if needed
	chunks, err := h.service.SplitLongText(request.Text, "")
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
		audio, err := h.service.GetAudioBase64WithProvider(chunk, request.VoiceID, provider, false)
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