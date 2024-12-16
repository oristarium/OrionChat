package tts

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const (
	tikTokAPIBaseURL = "https://api16-normal-v6.tiktokv.com/media/api/text/speech/invoke/"
	tikTokUserAgent  = "com.zhiliaoapp.musically/2022600030 (Linux; U; Android 7.1.2; es_ES; SM-G988N; Build/NRD90M;tt-ok/3.12.13.1)"
)

// TikTokProvider implements the Provider interface for TikTok TTS
type TikTokProvider struct {
	client *http.Client
	sanitizer *TextSanitizer
}

// NewTikTokProvider creates a new TikTok provider instance
func NewTikTokProvider() Provider {
	log.Printf("Initializing %s provider", ProviderTikTok)
	return &TikTokProvider{
		client: &http.Client{},
		sanitizer: NewTextSanitizer(),
	}
}

// GetAudioBase64 converts text to speech using TikTok's API
func (p *TikTokProvider) GetAudioBase64(text string, voiceID string, options map[string]interface{}) (string, error) {
	log.Printf("TikTok TTS Request - Text: %q, Voice: %q", text, voiceID)

	// Basic voice ID validation
	if voiceID == "" {
		return "", fmt.Errorf("voice ID cannot be empty")
	}

	// Sanitize text
	text = p.sanitizer.Sanitize(text, ProviderTikTok)

	// Ensure text is not empty after sanitization
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("text cannot be empty")
	}

	// Construct request URL with query parameters
	reqURL := fmt.Sprintf("%s?text_speaker=%s&req_text=%s&speaker_map_type=0&aid=1233",
		tikTokAPIBaseURL,
		url.QueryEscape(voiceID),
		url.QueryEscape(strings.TrimSpace(text)),
	)

	// Create request
	req, err := http.NewRequest("POST", reqURL, nil)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("User-Agent", tikTokUserAgent)
	req.Header.Set("Cookie", fmt.Sprintf("sessionid=%s", TikTokSessionID))

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		log.Printf("Request failed: %v", err)
		return "", fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response: %v", err)
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	// Parse response
	var result struct {
		Message string `json:"message"`
		Data    struct {
			VStr     string `json:"v_str"`
			Duration string `json:"duration"`
			Speaker  string `json:"speaker"`
		} `json:"data"`
		StatusCode int `json:"status_code"`
		Extra      struct {
			LogID string `json:"log_id"`
		} `json:"extra"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("Failed to parse response: %v", err)
		log.Printf("Response body: %s", string(body))
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	// Check for error message
	if result.Message == "Couldn't load speech. Try again." {
		log.Printf("TikTok error: Session ID is invalid")
		return "", fmt.Errorf("session ID is invalid")
	}

	// Convert duration to int for logging if needed
	var duration int
	if result.Data.Duration != "" {
		if _, err := fmt.Sscanf(result.Data.Duration, "%d", &duration); err != nil {
			log.Printf("Warning: Could not parse duration '%s': %v", result.Data.Duration, err)
		}
	}

	log.Printf("TikTok response - Status: %s, Code: %d, Duration: %s, Speaker: %s, LogID: %s",
		result.Message, result.StatusCode, result.Data.Duration, result.Data.Speaker, result.Extra.LogID)

	return result.Data.VStr, nil
}

// GetVoiceIDs returns available voice IDs for TikTok
func (p *TikTokProvider) GetVoiceIDs() []string {
	// We'll implement this later with the complete list of voice IDs
	return []string{"en_us_001", "en_us_002", "en_us_006", "en_us_007", "en_us_009", "en_us_010"}
}

// ValidateVoiceID checks if a voice ID is valid for TikTok
func (p *TikTokProvider) ValidateVoiceID(voiceID string) bool {
	// For now, accept any voice ID since we'll handle validation in the frontend
	return true
} 