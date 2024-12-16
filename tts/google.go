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
	defaultHost    = "https://translate.google.com"
	defaultTimeout = 10000 // ms
)

// GoogleTranslateProvider implements the Provider interface for Google Translate
type GoogleTranslateProvider struct {
	host    string
	client  *http.Client
	timeout int
}

// NewGoogleTranslateProvider creates a new Google Translate provider instance
func NewGoogleTranslateProvider() Provider {
	log.Printf("Initializing %s provider", ProviderGoogle)
	return &GoogleTranslateProvider{
		host:    defaultHost,
		client:  &http.Client{},
		timeout: defaultTimeout,
	}
}

// GetAudioBase64 converts text to speech using Google Translate
func (p *GoogleTranslateProvider) GetAudioBase64(text string, voiceID string, options map[string]interface{}) (string, error) {
	log.Printf("Google TTS Request - Text: %q, Lang: %q", text, voiceID)
	
	slow := false
	if val, ok := options["slow"].(bool); ok {
		slow = val
	}

	if len(text) > MaxTextLength {
		log.Printf("Text too long: %d characters (max: %d)", len(text), MaxTextLength)
		return "", fmt.Errorf("text length (%d) should be less than %d characters", len(text), MaxTextLength)
	}

	innerData := []interface{}{text, voiceID, slow, "null"}
	innerJSON, err := json.Marshal(innerData)
	if err != nil {
		log.Printf("Failed to marshal inner data: %v", err)
		return "", fmt.Errorf("failed to marshal inner data: %v", err)
	}

	data := [][3]interface{}{
		{
			"jQ1olc",
			string(innerJSON),
			nil,
		},
	}

	outerData := []interface{}{data}
	outerJSON, err := json.Marshal(outerData)
	if err != nil {
		log.Printf("Failed to marshal outer data: %v", err)
		return "", fmt.Errorf("failed to marshal outer data: %v", err)
	}

	formData := url.Values{}
	formData.Set("f.req", string(outerJSON))

	log.Printf("Sending request to Google Translate TTS")
	req, err := http.NewRequest("POST", 
		p.host+"/_/TranslateWebserverUi/data/batchexecute",
		strings.NewReader(formData.Encode()))
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	p.setHeaders(req)
	
	resp, err := p.client.Do(req)
	if err != nil {
		log.Printf("Request failed: %v", err)
		return "", fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("Response status: %s", resp.Status)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response: %v", err)
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	log.Printf("Response body: %s", string(body))
	return p.parseResponse(body)
}

// GetVoiceIDs returns available voice IDs for Google Translate
func (p *GoogleTranslateProvider) GetVoiceIDs() []string {
	return []string{
		"af", "ar", "bg", "bn", "bs", "ca", "cs", "da", "de", "el", "en", "es",
		"et", "fi", "fr", "gu", "hi", "hr", "hu", "id", "is", "it", "iw", "ja",
		"jw", "km", "kn", "ko", "la", "lv", "ml", "mr", "ms", "my", "ne", "nl",
		"no", "pl", "pt", "ro", "ru", "si", "sk", "sq", "sr", "su", "sv", "sw",
		"ta", "te", "th", "tl", "tr", "uk", "ur", "vi", "zh-CN", "zh-TW",
	}
}

// ValidateVoiceID checks if a voice ID is valid for Google Translate
func (p *GoogleTranslateProvider) ValidateVoiceID(voiceID string) bool {
	for _, id := range p.GetVoiceIDs() {
		if id == voiceID {
			return true
		}
	}
	return false
}

// Private helper methods
func (p *GoogleTranslateProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Origin", "https://translate.google.com")
	req.Header.Set("Referer", "https://translate.google.com/")
	req.Header.Set("sec-ch-ua", `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("x-client-data", "CIa2yQEIpbbJAQipncoBCMKTywEIkqHLAQiFoM0BCJyrzQEI2rHNAQjcuM0BCNy4zQEI/rnNAQiWus0BCMq6zQEI1rrNAQjpu80BCIPKzQEIhcrNAQ==")
}

func (p *GoogleTranslateProvider) parseResponse(body []byte) (string, error) {
	responseStr := string(body)
	if len(responseStr) < 6 {
		log.Printf("Invalid response format (length: %d)", len(responseStr))
		return "", fmt.Errorf("invalid response format")
	}

	responseStr = responseStr[5:] // Remove ")]}'"\n prefix

	var result interface{}
	if err := json.Unmarshal([]byte(responseStr), &result); err != nil {
		log.Printf("Failed to parse response: %v", err)
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	resultArray, ok := result.([]interface{})
	if !ok || len(resultArray) == 0 {
		log.Printf("Invalid response format: not an array or empty")
		return "", fmt.Errorf("invalid response format")
	}

	firstArray, ok := resultArray[0].([]interface{})
	if !ok || len(firstArray) < 3 {
		log.Printf("Invalid response structure: first array invalid or too short")
		return "", fmt.Errorf("invalid response structure")
	}

	audioData := firstArray[2]
	if audioData == nil {
		log.Printf("No audio data found in response")
		return "", fmt.Errorf("no audio data found")
	}

	audioStr, ok := audioData.(string)
	if !ok {
		log.Printf("Invalid audio data format: not a string")
		return "", fmt.Errorf("invalid audio data format")
	}

	var finalData []interface{}
	if err := json.Unmarshal([]byte(audioStr), &finalData); err != nil {
		log.Printf("Failed to parse audio data: %v", err)
		return "", fmt.Errorf("failed to parse audio data: %v", err)
	}

	if len(finalData) == 0 {
		log.Printf("No audio data in final parse")
		return "", fmt.Errorf("no audio data in final parse")
	}

	base64Audio, ok := finalData[0].(string)
	if !ok {
		log.Printf("Invalid base64 audio format")
		return "", fmt.Errorf("invalid base64 audio format")
	}

	log.Printf("Successfully parsed audio data (base64 length: %d)", len(base64Audio))
	return base64Audio, nil
} 